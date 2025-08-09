package downloader

import (
	"fmt"
)

// ErrorType represents different categories of download errors
type ErrorType int

const (
	ErrorInvalidURL ErrorType = iota
	ErrorNetworkFailure
	ErrorDecryptionFailure
	ErrorFileSystemError
	ErrorALACNotAvailable
	ErrorTimeout
	ErrorCancelled
	ErrorUnknown
)

// String returns the string representation of the error type
func (et ErrorType) String() string {
	switch et {
	case ErrorInvalidURL:
		return "invalid_url"
	case ErrorNetworkFailure:
		return "network_failure"
	case ErrorDecryptionFailure:
		return "decryption_failure"
	case ErrorFileSystemError:
		return "filesystem_error"
	case ErrorALACNotAvailable:
		return "alac_not_available"
	case ErrorTimeout:
		return "timeout"
	case ErrorCancelled:
		return "cancelled"
	case ErrorUnknown:
		return "unknown"
	default:
		return "unknown"
	}
}

// DownloadError represents a structured error that occurred during download
type DownloadError struct {
	Type    ErrorType              `json:"type"`
	Message string                 `json:"message"`
	Cause   error                  `json:"cause,omitempty"`
	Context map[string]interface{} `json:"context,omitempty"`
}

// Error implements the error interface
func (de *DownloadError) Error() string {
	if de.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", de.Type.String(), de.Message, de.Cause)
	}
	return fmt.Sprintf("%s: %s", de.Type.String(), de.Message)
}

// Unwrap returns the underlying cause error
func (de *DownloadError) Unwrap() error {
	return de.Cause
}

// NewDownloadError creates a new DownloadError with the specified type and message
func NewDownloadError(errorType ErrorType, message string) *DownloadError {
	return &DownloadError{
		Type:    errorType,
		Message: message,
		Context: make(map[string]interface{}),
	}
}

// NewDownloadErrorWithCause creates a new DownloadError with a cause
func NewDownloadErrorWithCause(errorType ErrorType, message string, cause error) *DownloadError {
	return &DownloadError{
		Type:    errorType,
		Message: message,
		Cause:   cause,
		Context: make(map[string]interface{}),
	}
}

// WithContext adds context information to the error
func (de *DownloadError) WithContext(key string, value interface{}) *DownloadError {
	if de.Context == nil {
		de.Context = make(map[string]interface{})
	}
	de.Context[key] = value
	return de
}

// IsType checks if the error is of a specific type
func (de *DownloadError) IsType(errorType ErrorType) bool {
	return de.Type == errorType
}

// IsDownloadError checks if an error is a DownloadError and optionally of a specific type
func IsDownloadError(err error, errorType ...ErrorType) bool {
	if de, ok := err.(*DownloadError); ok {
		if len(errorType) == 0 {
			return true
		}
		for _, et := range errorType {
			if de.Type == et {
				return true
			}
		}
	}
	return false
}