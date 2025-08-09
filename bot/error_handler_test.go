package bot

import (
	"errors"
	"log"
	"net"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestNewErrorHandler(t *testing.T) {
	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	client := &TelegramBot{}
	
	handler := NewErrorHandler(logger, client)
	
	if handler == nil {
		t.Fatal("NewErrorHandler returned nil")
	}
	
	if handler.logger != logger {
		t.Error("Logger not set correctly")
	}
	
	if handler.client != client {
		t.Error("Client not set correctly")
	}
}

func TestErrorType_String(t *testing.T) {
	tests := []struct {
		errorType ErrorType
		expected  string
	}{
		{ErrorTypeConfiguration, "CONFIGURATION"},
		{ErrorTypeNetwork, "NETWORK"},
		{ErrorTypeCommand, "COMMAND"},
		{ErrorTypeRuntime, "RUNTIME"},
		{ErrorType(999), "UNKNOWN"},
	}
	
	for _, test := range tests {
		result := test.errorType.String()
		if result != test.expected {
			t.Errorf("ErrorType.String() = %s, expected %s", result, test.expected)
		}
	}
}

func TestHandleRuntimeError(t *testing.T) {
	// Capture log output
	var logOutput strings.Builder
	logger := log.New(&logOutput, "", 0)
	
	handler := NewErrorHandler(logger, nil)
	
	testError := errors.New("test runtime error")
	handler.HandleRuntimeError(testError)
	
	logContent := logOutput.String()
	if !strings.Contains(logContent, "RUNTIME") {
		t.Error("Log should contain RUNTIME error type")
	}
	
	if !strings.Contains(logContent, "test runtime error") {
		t.Error("Log should contain the error message")
	}
	
	if !strings.Contains(logContent, "Correlation:") {
		t.Error("Log should contain correlation ID")
	}
}

func TestHandleCommandError(t *testing.T) {
	// Capture log output
	var logOutput strings.Builder
	logger := log.New(&logOutput, "", 0)
	
	handler := NewErrorHandler(logger, nil)
	
	cmdCtx := &CommandContext{
		UserID:  12345,
		ChatID:  67890,
		Command: "test",
	}
	
	testError := errors.New("test command error")
	handler.HandleCommandError(testError, cmdCtx)
	
	logContent := logOutput.String()
	if !strings.Contains(logContent, "COMMAND") {
		t.Error("Log should contain COMMAND error type")
	}
	
	if !strings.Contains(logContent, "test command error") {
		t.Error("Log should contain the error message")
	}
	
	if !strings.Contains(logContent, "User: 12345") {
		t.Error("Log should contain user ID")
	}
	
	if !strings.Contains(logContent, "Chat: 67890") {
		t.Error("Log should contain chat ID")
	}
	
	if !strings.Contains(logContent, "Command: /test") {
		t.Error("Log should contain command")
	}
}

func TestIsRetryableError(t *testing.T) {
	handler := NewErrorHandler(log.New(os.Stdout, "", 0), nil)
	
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "connection refused",
			err:      errors.New("connection refused"),
			expected: true,
		},
		{
			name:     "connection reset",
			err:      errors.New("connection reset by peer"),
			expected: true,
		},
		{
			name:     "timeout error",
			err:      errors.New("request timeout"),
			expected: true,
		},
		{
			name:     "temporary failure",
			err:      errors.New("temporary failure in name resolution"),
			expected: true,
		},
		{
			name:     "network unreachable",
			err:      errors.New("network is unreachable"),
			expected: true,
		},
		{
			name:     "non-retryable error",
			err:      errors.New("invalid token"),
			expected: false,
		},
		{
			name:     "permission error",
			err:      errors.New("permission denied"),
			expected: false,
		},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := handler.isRetryableError(test.err)
			if result != test.expected {
				t.Errorf("isRetryableError(%v) = %v, expected %v", test.err, result, test.expected)
			}
		})
	}
}

func TestIsNetworkError(t *testing.T) {
	handler := NewErrorHandler(log.New(os.Stdout, "", 0), nil)
	
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "network error",
			err:      errors.New("network connection failed"),
			expected: true,
		},
		{
			name:     "connection error",
			err:      errors.New("connection timeout"),
			expected: true,
		},
		{
			name:     "dns error",
			err:      errors.New("dns resolution failed"),
			expected: true,
		},
		{
			name:     "http error",
			err:      errors.New("http request failed"),
			expected: true,
		},
		{
			name:     "non-network error",
			err:      errors.New("invalid input"),
			expected: false,
		},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := handler.IsNetworkError(test.err)
			if result != test.expected {
				t.Errorf("IsNetworkError(%v) = %v, expected %v", test.err, result, test.expected)
			}
		})
	}
}

func TestCreateUserFriendlyMessage(t *testing.T) {
	handler := NewErrorHandler(log.New(os.Stdout, "", 0), nil)
	
	tests := []struct {
		name           string
		err            error
		correlationID  string
		expectedSubstr string
	}{
		{
			name:           "network error",
			err:            errors.New("network connection failed"),
			correlationID:  "12345678",
			expectedSubstr: "🌐 I'm having trouble connecting",
		},
		{
			name:           "timeout error",
			err:            errors.New("request timeout"),
			correlationID:  "12345678",
			expectedSubstr: "⏱️ The request took too long",
		},
		{
			name:           "rate limit error",
			err:            errors.New("rate limit exceeded"),
			correlationID:  "12345678",
			expectedSubstr: "🚦 I'm receiving too many requests",
		},
		{
			name:           "permission error",
			err:            errors.New("permission denied"),
			correlationID:  "12345678",
			expectedSubstr: "🔒 I don't have permission",
		},
		{
			name:           "not found error",
			err:            errors.New("resource not found"),
			correlationID:  "12345678",
			expectedSubstr: "🔍 The requested resource was not found",
		},
		{
			name:           "generic error",
			err:            errors.New("something went wrong"),
			correlationID:  "12345678",
			expectedSubstr: "❌ Something went wrong",
		},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := handler.createUserFriendlyMessage(test.err, test.correlationID)
			
			if !strings.Contains(result, test.expectedSubstr) {
				t.Errorf("createUserFriendlyMessage() should contain %q, got %q", test.expectedSubstr, result)
			}
			
			// Check that correlation ID is included (first 8 characters)
			if len(test.correlationID) >= 8 {
				expectedID := test.correlationID[:8]
				if !strings.Contains(result, expectedID) {
					t.Errorf("createUserFriendlyMessage() should contain correlation ID %q, got %q", expectedID, result)
				}
			}
		})
	}
}

func TestGenerateCorrelationID(t *testing.T) {
	handler := NewErrorHandler(log.New(os.Stdout, "", 0), nil)
	
	// Generate multiple correlation IDs
	id1 := handler.generateCorrelationID()
	time.Sleep(1 * time.Millisecond) // Ensure different timestamps
	id2 := handler.generateCorrelationID()
	
	// Check that IDs are not empty
	if id1 == "" {
		t.Error("generateCorrelationID() returned empty string")
	}
	
	if id2 == "" {
		t.Error("generateCorrelationID() returned empty string")
	}
	
	// Check that IDs are different
	if id1 == id2 {
		t.Error("generateCorrelationID() returned duplicate IDs")
	}
	
	// Check format (should contain a dash)
	if !strings.Contains(id1, "-") {
		t.Error("generateCorrelationID() should contain a dash separator")
	}
}

func TestRetryWithBackoff(t *testing.T) {
	handler := NewErrorHandler(log.New(os.Stdout, "", 0), nil)
	
	t.Run("successful operation", func(t *testing.T) {
		callCount := 0
		operation := func() error {
			callCount++
			return nil
		}
		
		err := handler.retryWithBackoff(operation, 3, 10*time.Millisecond)
		
		if err != nil {
			t.Errorf("retryWithBackoff() should succeed, got error: %v", err)
		}
		
		if callCount != 1 {
			t.Errorf("operation should be called once, was called %d times", callCount)
		}
	})
	
	t.Run("operation succeeds after retries", func(t *testing.T) {
		callCount := 0
		operation := func() error {
			callCount++
			if callCount < 3 {
				return errors.New("connection refused") // retryable error
			}
			return nil
		}
		
		err := handler.retryWithBackoff(operation, 3, 10*time.Millisecond)
		
		if err != nil {
			t.Errorf("retryWithBackoff() should succeed after retries, got error: %v", err)
		}
		
		if callCount != 3 {
			t.Errorf("operation should be called 3 times, was called %d times", callCount)
		}
	})
	
	t.Run("operation fails with non-retryable error", func(t *testing.T) {
		callCount := 0
		operation := func() error {
			callCount++
			return errors.New("invalid token") // non-retryable error
		}
		
		err := handler.retryWithBackoff(operation, 3, 10*time.Millisecond)
		
		if err == nil {
			t.Error("retryWithBackoff() should fail with non-retryable error")
		}
		
		if callCount != 1 {
			t.Errorf("operation should be called once for non-retryable error, was called %d times", callCount)
		}
	})
	
	t.Run("operation fails after max retries", func(t *testing.T) {
		callCount := 0
		operation := func() error {
			callCount++
			return errors.New("connection refused") // retryable error
		}
		
		err := handler.retryWithBackoff(operation, 2, 10*time.Millisecond)
		
		if err == nil {
			t.Error("retryWithBackoff() should fail after max retries")
		}
		
		if callCount != 3 { // initial attempt + 2 retries
			t.Errorf("operation should be called 3 times (initial + 2 retries), was called %d times", callCount)
		}
		
		if !strings.Contains(err.Error(), "operation failed after 3 retries") {
			t.Errorf("error should mention retry count, got: %v", err)
		}
	})
}

func TestRecoverFromPanic(t *testing.T) {
	// Capture log output
	var logOutput strings.Builder
	logger := log.New(&logOutput, "", 0)
	
	handler := NewErrorHandler(logger, nil)
	
	// Test panic recovery
	func() {
		defer handler.RecoverFromPanic()
		panic("test panic")
	}()
	
	logContent := logOutput.String()
	if !strings.Contains(logContent, "RUNTIME") {
		t.Error("Log should contain RUNTIME error type after panic recovery")
	}
	
	if !strings.Contains(logContent, "recovered from panic") {
		t.Error("Log should contain panic recovery message")
	}
	
	if !strings.Contains(logContent, "test panic") {
		t.Error("Log should contain the panic message")
	}
}

// Mock network error for testing
type mockNetError struct {
	msg       string
	timeout   bool
	temporary bool
}

func (e *mockNetError) Error() string   { return e.msg }
func (e *mockNetError) Timeout() bool   { return e.timeout }
func (e *mockNetError) Temporary() bool { return e.temporary }

func TestIsRetryableError_NetworkErrors(t *testing.T) {
	handler := NewErrorHandler(log.New(os.Stdout, "", 0), nil)
	
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "temporary network error",
			err:      &mockNetError{msg: "temporary failure", temporary: true, timeout: false},
			expected: true,
		},
		{
			name:     "timeout network error",
			err:      &mockNetError{msg: "timeout", temporary: false, timeout: true},
			expected: true,
		},
		{
			name:     "permanent network error",
			err:      &mockNetError{msg: "permanent failure", temporary: false, timeout: false},
			expected: false,
		},
		{
			name:     "syscall ECONNREFUSED",
			err:      &net.OpError{Err: syscall.ECONNREFUSED},
			expected: true,
		},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := handler.isRetryableError(test.err)
			if result != test.expected {
				t.Errorf("isRetryableError(%v) = %v, expected %v", test.err, result, test.expected)
			}
		})
	}
}