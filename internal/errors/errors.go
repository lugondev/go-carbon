// Package errors defines error types used throughout the carbon-core framework.
//
// The Error type captures various error cases that can occur within the framework,
// providing detailed error messages and support for custom error handling.
package errors

import (
	"errors"
	"fmt"
)

// Error codes for the carbon framework.
const (
	ErrCodeMissingUpdateType         = "MISSING_UPDATE_TYPE"
	ErrCodeFailedToReceiveUpdates    = "FAILED_TO_RECEIVE_UPDATES"
	ErrCodeMissingFeePayer           = "MISSING_FEE_PAYER"
	ErrCodeMissingInnerInstructions  = "MISSING_INNER_INSTRUCTIONS"
	ErrCodeMissingAccount            = "MISSING_ACCOUNT"
	ErrCodeMissingInstructionData    = "MISSING_INSTRUCTION_DATA"
	ErrCodeFailedToConsumeDatasource = "FAILED_TO_CONSUME_DATASOURCE"
	ErrCodeCustom                    = "CUSTOM"
	ErrCodeContextCanceled           = "CONTEXT_CANCELED"
	ErrCodeChannelClosed             = "CHANNEL_CLOSED"
	ErrCodeDecodeFailed              = "DECODE_FAILED"
	ErrCodeProcessFailed             = "PROCESS_FAILED"
)

// CarbonError represents an error in the carbon framework.
type CarbonError struct {
	// Code is a unique error code for this error type.
	Code string

	// Message is a human-readable error message.
	Message string

	// Cause is the underlying error, if any.
	Cause error

	// Details contains additional error context.
	Details map[string]any
}

// Error implements the error interface.
func (e *CarbonError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error.
func (e *CarbonError) Unwrap() error {
	return e.Cause
}

// Is reports whether the error matches the target.
func (e *CarbonError) Is(target error) bool {
	t, ok := target.(*CarbonError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// WithCause adds a cause to the error.
func (e *CarbonError) WithCause(cause error) *CarbonError {
	e.Cause = cause
	return e
}

// WithDetails adds details to the error.
func (e *CarbonError) WithDetails(details map[string]any) *CarbonError {
	e.Details = details
	return e
}

// NewError creates a new CarbonError.
func NewError(code, message string) *CarbonError {
	return &CarbonError{
		Code:    code,
		Message: message,
	}
}

// Pre-defined errors for common error cases.
var (
	// ErrMissingUpdateType is returned when a required update type is missing from a datasource.
	ErrMissingUpdateType = NewError(ErrCodeMissingUpdateType, "missing update type in datasource")

	// ErrMissingFeePayer is returned when a transaction is missing its fee payer.
	ErrMissingFeePayer = NewError(ErrCodeMissingFeePayer, "transaction missing fee payer")

	// ErrMissingInnerInstructions is returned when inner instructions are missing.
	ErrMissingInnerInstructions = NewError(ErrCodeMissingInnerInstructions, "missing inner instructions")

	// ErrMissingAccount is returned when an account is missing from a transaction.
	ErrMissingAccount = NewError(ErrCodeMissingAccount, "missing account in transaction")

	// ErrMissingInstructionData is returned when instruction data is missing.
	ErrMissingInstructionData = NewError(ErrCodeMissingInstructionData, "missing instruction data")

	// ErrContextCanceled is returned when the context is canceled.
	ErrContextCanceled = NewError(ErrCodeContextCanceled, "context canceled")

	// ErrChannelClosed is returned when a channel is closed unexpectedly.
	ErrChannelClosed = NewError(ErrCodeChannelClosed, "channel closed")
)

// FailedToReceiveUpdates creates an error for failed update reception.
func FailedToReceiveUpdates(reason string) *CarbonError {
	return NewError(ErrCodeFailedToReceiveUpdates, fmt.Sprintf("failed to receive updates: %s", reason))
}

// FailedToConsumeDatasource creates an error for datasource consumption failure.
func FailedToConsumeDatasource(reason string) *CarbonError {
	return NewError(ErrCodeFailedToConsumeDatasource, fmt.Sprintf("failed to consume datasource: %s", reason))
}

// Custom creates a custom error with the given message.
func Custom(message string) *CarbonError {
	return NewError(ErrCodeCustom, message)
}

// DecodeFailed creates an error for decoding failures.
func DecodeFailed(what string, cause error) *CarbonError {
	return NewError(ErrCodeDecodeFailed, fmt.Sprintf("failed to decode %s", what)).WithCause(cause)
}

// ProcessFailed creates an error for processing failures.
func ProcessFailed(what string, cause error) *CarbonError {
	return NewError(ErrCodeProcessFailed, fmt.Sprintf("failed to process %s", what)).WithCause(cause)
}

// Wrap wraps an error with additional context.
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// Is reports whether any error in err's chain matches target.
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As finds the first error in err's chain that matches target.
func As(err error, target any) bool {
	return errors.As(err, target)
}

// Join returns an error that wraps the given errors.
func Join(errs ...error) error {
	return errors.Join(errs...)
}
