package gen

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lugondev/go-carbon/internal/codegen"
)

// GenerateAll generates all files from an IDL.
func GenerateAll(idl *codegen.IDL, packageName, outputDir string) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Validate IDL
	if err := validateIDL(idl); err != nil {
		return fmt.Errorf("invalid IDL: %w", err)
	}

	// Generate program.go (ProgramID and metadata)
	if err := GenerateProgramFile(idl, packageName, outputDir); err != nil {
		return fmt.Errorf("failed to generate program.go: %w", err)
	}

	// Generate types.go (custom types: structs and enums)
	if len(idl.Types) > 0 {
		if err := GenerateTypesFile(idl, packageName, outputDir); err != nil {
			return fmt.Errorf("failed to generate types.go: %w", err)
		}
	}

	// Generate accounts.go (account types with discriminators)
	if len(idl.Accounts) > 0 {
		if err := GenerateAccountsFile(idl, packageName, outputDir); err != nil {
			return fmt.Errorf("failed to generate accounts.go: %w", err)
		}
	}

	// Generate instructions.go (instruction builders and parser)
	if len(idl.Instructions) > 0 {
		if err := GenerateInstructionsFile(idl, packageName, outputDir); err != nil {
			return fmt.Errorf("failed to generate instructions.go: %w", err)
		}
	}

	// Generate events.go (event types and parser)
	if len(idl.Events) > 0 {
		if err := GenerateEventsFile(idl, packageName, outputDir); err != nil {
			return fmt.Errorf("failed to generate events.go: %w", err)
		}
	}

	return nil
}

// validateIDL performs basic validation on the IDL.
func validateIDL(idl *codegen.IDL) error {
	if idl == nil {
		return fmt.Errorf("IDL is nil")
	}

	if idl.Address == "" {
		return fmt.Errorf("program address is required")
	}

	if idl.Metadata.Name == "" {
		return fmt.Errorf("program name is required")
	}

	return nil
}

// GetGeneratedFiles returns the list of files that will be generated.
func GetGeneratedFiles(idl *codegen.IDL) []string {
	files := []string{"program.go"}

	if len(idl.Types) > 0 {
		files = append(files, "types.go")
	}

	if len(idl.Accounts) > 0 {
		files = append(files, "accounts.go")
	}

	if len(idl.Instructions) > 0 {
		files = append(files, "instructions.go")
	}

	if len(idl.Events) > 0 {
		files = append(files, "events.go")
	}

	return files
}

// CleanOutputDir removes all generated files from the output directory.
func CleanOutputDir(outputDir string) error {
	generatedFiles := []string{
		"program.go",
		"types.go",
		"accounts.go",
		"instructions.go",
		"events.go",
	}

	for _, file := range generatedFiles {
		filePath := filepath.Join(outputDir, file)
		if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove %s: %w", file, err)
		}
	}

	return nil
}
