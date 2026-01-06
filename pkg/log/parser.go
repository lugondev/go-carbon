// Package log provides utilities for parsing and extracting data from Solana transaction logs.
//
// The log parser is designed to be modular and extensible, allowing developers to:
//   - Extract "Program data:" messages from transaction logs
//   - Filter logs by instruction path (nested instruction support)
//   - Parse structured events from log messages
//   - Register custom log processors
//
// Example usage:
//
//	parser := log.NewParser()
//	events := parser.ExtractProgramData(logMessages)
//	for _, event := range events {
//	    // Process event data
//	}
package log

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
)

// LogType represents the type of a log message.
type LogType int

const (
	// LogTypeUnknown represents an unrecognized log message.
	LogTypeUnknown LogType = iota
	// LogTypeInvoke represents a "Program X invoke [N]" message.
	LogTypeInvoke
	// LogTypeSuccess represents a "Program X success" message.
	LogTypeSuccess
	// LogTypeFailed represents a "Program X failed" message.
	LogTypeFailed
	// LogTypeData represents a "Program data: BASE64" message.
	LogTypeData
	// LogTypeLog represents a "Program log: MESSAGE" message.
	LogTypeLog
	// LogTypeComputeUnits represents a compute units consumed message.
	LogTypeComputeUnits
)

// String returns the string representation of LogType.
func (lt LogType) String() string {
	switch lt {
	case LogTypeInvoke:
		return "Invoke"
	case LogTypeSuccess:
		return "Success"
	case LogTypeFailed:
		return "Failed"
	case LogTypeData:
		return "Data"
	case LogTypeLog:
		return "Log"
	case LogTypeComputeUnits:
		return "ComputeUnits"
	default:
		return "Unknown"
	}
}

// ParsedLog represents a parsed log message with its type and extracted data.
type ParsedLog struct {
	// Type is the type of the log message.
	Type LogType

	// StackHeight is the call stack depth (1-indexed).
	// Only relevant for Invoke logs.
	StackHeight int

	// ProgramID is the program that produced this log.
	// Extracted from "Program X ..." messages.
	ProgramID string

	// Data is the decoded data from "Program data:" messages.
	Data []byte

	// Message is the text from "Program log:" messages.
	Message string

	// ComputeUnits is the number of compute units consumed.
	// Only relevant for ComputeUnits logs.
	ComputeUnits *uint64

	// RawLog is the original log message.
	RawLog string
}

// LogParser parses Solana transaction logs.
type LogParser struct {
	// patterns are compiled regex patterns for log parsing.
	patterns *logPatterns
}

// logPatterns contains compiled regex patterns for various log types.
type logPatterns struct {
	invoke       *regexp.Regexp
	success      *regexp.Regexp
	failed       *regexp.Regexp
	data         *regexp.Regexp
	log          *regexp.Regexp
	computeUnits *regexp.Regexp
}

// NewParser creates a new LogParser.
func NewParser() *LogParser {
	return &LogParser{
		patterns: &logPatterns{
			invoke:       regexp.MustCompile(`^Program (\S+) invoke \[(\d+)\]`),
			success:      regexp.MustCompile(`^Program (\S+) success`),
			failed:       regexp.MustCompile(`^Program (\S+) failed`),
			data:         regexp.MustCompile(`^Program data: (.+)$`),
			log:          regexp.MustCompile(`^Program log: (.+)$`),
			computeUnits: regexp.MustCompile(`consumed (\d+) of \d+ compute units`),
		},
	}
}

// Parse parses a single log message and returns a ParsedLog.
func (p *LogParser) Parse(logMessage string) *ParsedLog {
	result := &ParsedLog{
		Type:   LogTypeUnknown,
		RawLog: logMessage,
	}

	// Try to match invoke pattern
	if matches := p.patterns.invoke.FindStringSubmatch(logMessage); matches != nil {
		result.Type = LogTypeInvoke
		result.ProgramID = matches[1]
		fmt.Sscanf(matches[2], "%d", &result.StackHeight)
		return result
	}

	// Try to match success pattern
	if matches := p.patterns.success.FindStringSubmatch(logMessage); matches != nil {
		result.Type = LogTypeSuccess
		result.ProgramID = matches[1]
		return result
	}

	// Try to match failed pattern
	if matches := p.patterns.failed.FindStringSubmatch(logMessage); matches != nil {
		result.Type = LogTypeFailed
		result.ProgramID = matches[1]
		return result
	}

	// Try to match data pattern
	if matches := p.patterns.data.FindStringSubmatch(logMessage); matches != nil {
		result.Type = LogTypeData
		if decoded, err := base64.StdEncoding.DecodeString(matches[1]); err == nil {
			result.Data = decoded
		}
		return result
	}

	// Try to match log pattern
	if matches := p.patterns.log.FindStringSubmatch(logMessage); matches != nil {
		result.Type = LogTypeLog
		result.Message = matches[1]
		return result
	}

	// Try to match compute units pattern
	if matches := p.patterns.computeUnits.FindStringSubmatch(logMessage); matches != nil {
		result.Type = LogTypeComputeUnits
		var cu uint64
		if _, err := fmt.Sscanf(matches[1], "%d", &cu); err == nil {
			result.ComputeUnits = &cu
		}
		return result
	}

	return result
}

// ParseAll parses all log messages and returns a slice of ParsedLog.
func (p *LogParser) ParseAll(logMessages []string) []*ParsedLog {
	results := make([]*ParsedLog, 0, len(logMessages))
	for _, log := range logMessages {
		results = append(results, p.Parse(log))
	}
	return results
}

// ExtractProgramData extracts all "Program data:" messages and returns decoded data.
func (p *LogParser) ExtractProgramData(logMessages []string) [][]byte {
	var data [][]byte
	for _, log := range logMessages {
		if parsed := p.Parse(log); parsed.Type == LogTypeData && len(parsed.Data) > 0 {
			data = append(data, parsed.Data)
		}
	}
	return data
}

// ExtractProgramLogs extracts all "Program log:" messages.
func (p *LogParser) ExtractProgramLogs(logMessages []string) []string {
	var logs []string
	for _, log := range logMessages {
		if parsed := p.Parse(log); parsed.Type == LogTypeLog {
			logs = append(logs, parsed.Message)
		}
	}
	return logs
}

// InstructionPath represents the path to a specific instruction in the transaction.
// The path is a sequence of indexes representing nested instruction calls.
// For example: [0] = first top-level instruction, [0, 1] = second inner instruction of first top-level.
type InstructionPath []uint8

// String returns a string representation of the path.
func (path InstructionPath) String() string {
	if len(path) == 0 {
		return "[]"
	}
	parts := make([]string, len(path))
	for i, idx := range path {
		parts[i] = fmt.Sprintf("%d", idx)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// Equals checks if two paths are equal.
func (path InstructionPath) Equals(other InstructionPath) bool {
	if len(path) != len(other) {
		return false
	}
	for i := range path {
		if path[i] != other[i] {
			return false
		}
	}
	return true
}

// IsParentOf checks if this path is a parent of the other path.
func (path InstructionPath) IsParentOf(other InstructionPath) bool {
	if len(path) >= len(other) {
		return false
	}
	for i := range path {
		if path[i] != other[i] {
			return false
		}
	}
	return true
}

// FilterByInstructionPath filters logs to only include those from a specific instruction path.
// This is useful for extracting logs from a specific nested instruction.
func (p *LogParser) FilterByInstructionPath(logMessages []string, targetPath InstructionPath) []string {
	var filtered []string
	var currentPath InstructionPath
	var lastStackHeight int

	stackPositions := make(map[int]uint8)

	for _, log := range logMessages {
		parsed := p.Parse(log)

		switch parsed.Type {
		case LogTypeInvoke:
			currentStackHeight := parsed.StackHeight

			// Update position at this stack level
			if currentStackHeight > lastStackHeight {
				// Going deeper - new position starts at 0
				stackPositions[currentStackHeight] = 0
			} else if pos, exists := stackPositions[currentStackHeight]; exists {
				// Same level or coming back - increment position
				stackPositions[currentStackHeight] = pos + 1
			} else {
				stackPositions[currentStackHeight] = 0
			}

			// Build current path
			currentPath = make(InstructionPath, 0, currentStackHeight)
			for level := 1; level <= currentStackHeight; level++ {
				if pos, exists := stackPositions[level]; exists {
					currentPath = append(currentPath, pos)
				} else {
					currentPath = append(currentPath, 0)
				}
			}

			lastStackHeight = currentStackHeight

		case LogTypeSuccess, LogTypeFailed:
			// Instruction finished - pop from stack
			if len(currentPath) > 0 {
				currentPath = currentPath[:len(currentPath)-1]
			}
		}

		// Include log if it matches the target path
		if currentPath.Equals(targetPath) && (parsed.Type == LogTypeData || parsed.Type == LogTypeLog) {
			filtered = append(filtered, log)
		}
	}

	return filtered
}

// LogProcessor is an interface for custom log processing.
type LogProcessor interface {
	// ProcessLog processes a parsed log and returns true if it was handled.
	ProcessLog(log *ParsedLog) bool

	// GetName returns the name of this processor.
	GetName() string
}

// ProcessorFunc is a function type that implements LogProcessor.
type ProcessorFunc func(log *ParsedLog) bool

// ProcessLog implements LogProcessor interface.
func (f ProcessorFunc) ProcessLog(log *ParsedLog) bool {
	return f(log)
}

// GetName returns a default name.
func (f ProcessorFunc) GetName() string {
	return "ProcessorFunc"
}

// ParserWithProcessors extends LogParser with custom processors.
type ParserWithProcessors struct {
	*LogParser
	processors []LogProcessor
}

// NewParserWithProcessors creates a new parser with custom processors.
func NewParserWithProcessors(processors ...LogProcessor) *ParserWithProcessors {
	return &ParserWithProcessors{
		LogParser:  NewParser(),
		processors: processors,
	}
}

// AddProcessor adds a custom log processor.
func (p *ParserWithProcessors) AddProcessor(processor LogProcessor) {
	p.processors = append(p.processors, processor)
}

// ParseWithProcessors parses logs and runs all processors.
func (p *ParserWithProcessors) ParseWithProcessors(logMessages []string) []*ParsedLog {
	results := p.ParseAll(logMessages)

	// Run processors on each log
	for _, result := range results {
		for _, processor := range p.processors {
			if processor.ProcessLog(result) {
				// Processor handled this log
				break
			}
		}
	}

	return results
}
