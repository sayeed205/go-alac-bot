package bot

import (
	"context"
	"fmt"
	"log"
	"math"
	"net"
	"strings"
	"syscall"
	"time"

	"github.com/gotd/td/tg"
)

// ErrorType represents different categories of errors
type ErrorType int

const (
	ErrorTypeConfiguration ErrorType = iota
	ErrorTypeNetwork
	ErrorTypeCommand
	ErrorTypeRuntime
)

// String returns the string representation of ErrorType
func (et ErrorType) String() string {
	switch et {
	case ErrorTypeConfiguration:
		return "CONFIGURATION"
	case ErrorTypeNetwork:
		return "NETWORK"
	case ErrorTypeCommand:
		return "COMMAND"
	case ErrorTypeRuntime:
		return "RUNTIME"
	default:
		return "UNKNOWN"
	}
}

// ErrorContext provides context information for error handling
type ErrorContext struct {
	UserID        int64
	ChatID        int64
	Command       string
	CorrelationID string
	Timestamp     time.Time
}

// ErrorHandler provides centralized error management for the bot
type ErrorHandler struct {
	logger *log.Logger
	client *TelegramBot
}

// NewErrorHandler creates a new ErrorHandler instance
func NewErrorHandler(logger *log.Logger, client *TelegramBot) *ErrorHandler {
	return &ErrorHandler{
		logger: logger,
		client: client,
	}
}

// HandleConfigError handles configuration-related errors
// These are critical errors that should cause the application to exit
func (e *ErrorHandler) HandleConfigError(err error) {
	e.logStructuredError(ErrorTypeConfiguration, err, nil, "Configuration error occurred")
	// Configuration errors are fatal - log and exit
	e.logger.Fatalf("FATAL: Configuration error: %v", err)
}

// HandleNetworkError handles network-related errors with retry logic
func (e *ErrorHandler) HandleNetworkError(err error, retryable bool) error {
	errorCtx := &ErrorContext{
		CorrelationID: e.generateCorrelationID(),
		Timestamp:     time.Now(),
	}
	
	e.logStructuredError(ErrorTypeNetwork, err, errorCtx, "Network error occurred")
	
	if !retryable {
		return err
	}
	
	// Implement exponential backoff retry logic
	return e.retryWithBackoff(func() error {
		// The actual retry logic would be implemented by the caller
		// This method provides the framework for retry handling
		return err
	}, 3, time.Second)
}

// HandleCommandError handles command processing errors
func (e *ErrorHandler) HandleCommandError(err error, cmdCtx *CommandContext) {
	errorCtx := &ErrorContext{
		UserID:        cmdCtx.UserID,
		ChatID:        cmdCtx.ChatID,
		Command:       cmdCtx.Command,
		CorrelationID: e.generateCorrelationID(),
		Timestamp:     time.Now(),
	}
	
	e.logStructuredError(ErrorTypeCommand, err, errorCtx, "Command processing error occurred")
	
	// Send user-friendly error message
	if err := e.sendUserErrorMessage(cmdCtx.ChatID, err, errorCtx.CorrelationID); err != nil {
		e.logger.Printf("ERROR: Failed to send error message to user (chat: %d, correlation: %s): %v", 
			cmdCtx.ChatID, errorCtx.CorrelationID, err)
	}
}

// HandleRuntimeError handles unexpected runtime errors
func (e *ErrorHandler) HandleRuntimeError(err error) {
	errorCtx := &ErrorContext{
		CorrelationID: e.generateCorrelationID(),
		Timestamp:     time.Now(),
	}
	
	e.logStructuredError(ErrorTypeRuntime, err, errorCtx, "Runtime error occurred")
	
	// Runtime errors are logged but don't stop the application
	// The application should continue running for other users
}

// logStructuredError logs errors with structured information
func (e *ErrorHandler) logStructuredError(errorType ErrorType, err error, ctx *ErrorContext, message string) {
	logEntry := fmt.Sprintf("[%s] %s: %v", errorType.String(), message, err)
	
	if ctx != nil {
		logEntry += fmt.Sprintf(" | Correlation: %s | Timestamp: %s", 
			ctx.CorrelationID, ctx.Timestamp.Format(time.RFC3339))
		
		if ctx.UserID != 0 {
			logEntry += fmt.Sprintf(" | User: %d", ctx.UserID)
		}
		
		if ctx.ChatID != 0 {
			logEntry += fmt.Sprintf(" | Chat: %d", ctx.ChatID)
		}
		
		if ctx.Command != "" {
			logEntry += fmt.Sprintf(" | Command: /%s", ctx.Command)
		}
	}
	
	// Log based on error severity
	switch errorType {
	case ErrorTypeConfiguration:
		e.logger.Printf("FATAL: %s", logEntry)
	case ErrorTypeNetwork, ErrorTypeRuntime:
		e.logger.Printf("ERROR: %s", logEntry)
	case ErrorTypeCommand:
		e.logger.Printf("WARN: %s", logEntry)
	default:
		e.logger.Printf("ERROR: %s", logEntry)
	}
}

// sendUserErrorMessage sends a user-friendly error message to the chat
func (e *ErrorHandler) sendUserErrorMessage(chatID int64, err error, correlationID string) error {
	if e.client == nil || e.client.GetClient() == nil {
		return fmt.Errorf("bot client is not available")
	}
	
	// Create user-friendly error message based on error type
	userMessage := e.createUserFriendlyMessage(err, correlationID)
	
	// Create context with timeout for error message sending
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Determine peer type for the chat
	var peer tg.InputPeerClass
	if chatID > 0 {
		peer = &tg.InputPeerUser{UserID: chatID}
	} else {
		peer = &tg.InputPeerChat{ChatID: -chatID}
	}
	
	// Create and send the error message
	request := &tg.MessagesSendMessageRequest{
		Peer:     peer,
		Message:  userMessage,
		RandomID: time.Now().UnixNano(),
	}
	
	_, err = e.client.GetClient().API().MessagesSendMessage(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to send error message via Telegram API: %w", err)
	}
	
	return nil
}

// createUserFriendlyMessage creates a user-friendly error message
func (e *ErrorHandler) createUserFriendlyMessage(err error, correlationID string) string {
	// Analyze error type and create appropriate user message
	errorMsg := strings.ToLower(err.Error())
	
	var userMessage string
	
	switch {
	case strings.Contains(errorMsg, "network") || strings.Contains(errorMsg, "connection"):
		userMessage = "🌐 I'm having trouble connecting to Telegram's servers. Please try again in a moment."
	case strings.Contains(errorMsg, "timeout"):
		userMessage = "⏱️ The request took too long to process. Please try again."
	case strings.Contains(errorMsg, "rate limit") || strings.Contains(errorMsg, "too many"):
		userMessage = "🚦 I'm receiving too many requests right now. Please wait a moment and try again."
	case strings.Contains(errorMsg, "permission") || strings.Contains(errorMsg, "forbidden"):
		userMessage = "🔒 I don't have permission to perform this action. Please check my permissions."
	case strings.Contains(errorMsg, "not found"):
		userMessage = "🔍 The requested resource was not found. Please check your command and try again."
	default:
		userMessage = "❌ Something went wrong while processing your request. Please try again."
	}
	
	// Add correlation ID for debugging (only show first 8 characters)
	if len(correlationID) >= 8 {
		userMessage += fmt.Sprintf("\n\n🔧 Error ID: %s", correlationID[:8])
	}
	
	return userMessage
}

// retryWithBackoff implements exponential backoff retry logic
func (e *ErrorHandler) retryWithBackoff(operation func() error, maxRetries int, baseDelay time.Duration) error {
	var lastErr error
	
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Calculate exponential backoff delay
			delay := time.Duration(float64(baseDelay) * math.Pow(2, float64(attempt-1)))
			
			// Cap the maximum delay at 30 seconds
			if delay > 30*time.Second {
				delay = 30 * time.Second
			}
			
			e.logger.Printf("INFO: Retrying operation in %v (attempt %d/%d)", delay, attempt+1, maxRetries+1)
			time.Sleep(delay)
		}
		
		err := operation()
		if err == nil {
			if attempt > 0 {
				e.logger.Printf("INFO: Operation succeeded after %d retries", attempt)
			}
			return nil
		}
		
		lastErr = err
		
		// Check if error is retryable
		if !e.isRetryableError(err) {
			e.logger.Printf("WARN: Non-retryable error encountered: %v", err)
			break
		}
		
		e.logger.Printf("WARN: Operation failed (attempt %d/%d): %v", attempt+1, maxRetries+1, err)
	}
	
	return fmt.Errorf("operation failed after %d retries: %w", maxRetries+1, lastErr)
}

// isRetryableError determines if an error is retryable
func (e *ErrorHandler) isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	
	errorMsg := strings.ToLower(err.Error())
	
	// Network errors that are typically retryable
	retryableErrors := []string{
		"connection refused",
		"connection reset",
		"timeout",
		"temporary failure",
		"network is unreachable",
		"no route to host",
		"connection timed out",
	}
	
	for _, retryableError := range retryableErrors {
		if strings.Contains(errorMsg, retryableError) {
			return true
		}
	}
	
	// Check for specific network error types
	if netErr, ok := err.(net.Error); ok {
		return netErr.Temporary() || netErr.Timeout()
	}
	
	// Check for syscall errors
	if opErr, ok := err.(*net.OpError); ok {
		if syscallErr, ok := opErr.Err.(*syscall.Errno); ok {
			switch *syscallErr {
			case syscall.ECONNREFUSED, syscall.ECONNRESET, syscall.ETIMEDOUT:
				return true
			}
		}
	}
	
	return false
}

// generateCorrelationID generates a unique correlation ID for error tracking
func (e *ErrorHandler) generateCorrelationID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Nanosecond()%1000000)
}

// IsNetworkError checks if an error is network-related
func (e *ErrorHandler) IsNetworkError(err error) bool {
	if err == nil {
		return false
	}
	
	errorMsg := strings.ToLower(err.Error())
	networkKeywords := []string{
		"network", "connection", "timeout", "dns", "tcp", "udp", "http", "tls", "ssl",
	}
	
	for _, keyword := range networkKeywords {
		if strings.Contains(errorMsg, keyword) {
			return true
		}
	}
	
	// Check for network error types
	_, isNetError := err.(net.Error)
	_, isOpError := err.(*net.OpError)
	
	return isNetError || isOpError
}

// RecoverFromPanic recovers from panics and logs them as runtime errors
func (e *ErrorHandler) RecoverFromPanic() {
	if r := recover(); r != nil {
		var err error
		if e, ok := r.(error); ok {
			err = e
		} else {
			err = fmt.Errorf("panic: %v", r)
		}
		
		e.HandleRuntimeError(fmt.Errorf("recovered from panic: %w", err))
	}
}