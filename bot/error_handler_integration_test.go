package bot

import (
	"context"
	"errors"
	"log"
	"strings"
	"testing"
	"time"

	"go-alac-bot/config"
)

func TestErrorHandlerIntegration(t *testing.T) {
	// Create a test configuration
	cfg := &config.BotConfig{
		Token:    "test_token",
		APIID:    12345,
		APIHash:  "test_hash",
		LogLevel: "INFO",
	}
	
	// Capture log output
	var logOutput strings.Builder
	logger := log.New(&logOutput, "TEST: ", log.LstdFlags)
	
	// Create bot with error handler
	bot, err := NewTelegramBot(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create bot: %v", err)
	}
	
	// Verify error handler is initialized
	errorHandler := bot.GetErrorHandler()
	if errorHandler == nil {
		t.Fatal("Error handler should be initialized")
	}
	
	// Test command error handling
	cmdCtx := &CommandContext{
		UserID:    12345,
		ChatID:    67890,
		Command:   "test",
		Timestamp: time.Now(),
	}
	
	testError := errors.New("test command processing error")
	errorHandler.HandleCommandError(testError, cmdCtx)
	
	// Verify error was logged
	logContent := logOutput.String()
	if !strings.Contains(logContent, "COMMAND") {
		t.Error("Command error should be logged with COMMAND type")
	}
	
	if !strings.Contains(logContent, "test command processing error") {
		t.Error("Error message should be logged")
	}
	
	if !strings.Contains(logContent, "User: 12345") {
		t.Error("User ID should be logged")
	}
	
	if !strings.Contains(logContent, "Chat: 67890") {
		t.Error("Chat ID should be logged")
	}
}

func TestErrorHandlerWithRouter(t *testing.T) {
	// Create a test configuration
	cfg := &config.BotConfig{
		Token:    "test_token",
		APIID:    12345,
		APIHash:  "test_hash",
		LogLevel: "INFO",
	}
	
	// Capture log output
	var logOutput strings.Builder
	logger := log.New(&logOutput, "TEST: ", log.LstdFlags)
	
	// Create bot with error handler
	bot, err := NewTelegramBot(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create bot: %v", err)
	}
	
	// Get router and verify error handler is set
	router := bot.GetRouter()
	if router == nil {
		t.Fatal("Router should be initialized")
	}
	
	if router.errorHandler == nil {
		t.Fatal("Router should have error handler set")
	}
	
	// Create a test handler that always fails
	failingHandler := &TestFailingHandler{}
	router.RegisterHandler(failingHandler)
	
	// Create a mock update (simplified for testing)
	cmdCtx := &CommandContext{
		UserID:    12345,
		ChatID:    67890,
		Command:   "fail",
		Timestamp: time.Now(),
	}
	
	// This should not return an error because the error handler catches it
	err = failingHandler.Handle(context.Background(), cmdCtx)
	if err == nil {
		t.Error("Test handler should return an error")
	}
	
	// But when processed through router with error handler, it should be handled
	router.errorHandler.HandleCommandError(err, cmdCtx)
	
	// Verify error was logged
	logContent := logOutput.String()
	if !strings.Contains(logContent, "COMMAND") {
		t.Error("Command error should be logged")
	}
}

func TestErrorHandlerNetworkRetry(t *testing.T) {
	// Capture log output
	var logOutput strings.Builder
	logger := log.New(&logOutput, "TEST: ", log.LstdFlags)
	
	errorHandler := NewErrorHandler(logger, nil)
	
	// Test network error with retry
	networkErr := errors.New("connection refused")
	
	// This should identify it as a network error
	if !errorHandler.IsNetworkError(networkErr) {
		t.Error("Should identify connection refused as network error")
	}
	
	// Test retry logic
	callCount := 0
	operation := func() error {
		callCount++
		if callCount < 2 {
			return errors.New("connection refused")
		}
		return nil
	}
	
	err := errorHandler.retryWithBackoff(operation, 3, 10*time.Millisecond)
	if err != nil {
		t.Errorf("Retry should succeed: %v", err)
	}
	
	if callCount != 2 {
		t.Errorf("Operation should be called 2 times, was called %d times", callCount)
	}
	
	// Verify retry logs
	logContent := logOutput.String()
	if !strings.Contains(logContent, "Retrying operation") {
		t.Error("Should log retry attempts")
	}
	
	if !strings.Contains(logContent, "Operation succeeded after") {
		t.Error("Should log successful retry")
	}
}

func TestErrorHandlerPanicRecovery(t *testing.T) {
	// Capture log output
	var logOutput strings.Builder
	logger := log.New(&logOutput, "TEST: ", log.LstdFlags)
	
	errorHandler := NewErrorHandler(logger, nil)
	
	// Test panic recovery
	func() {
		defer errorHandler.RecoverFromPanic()
		panic("test panic for recovery")
	}()
	
	// Verify panic was logged as runtime error
	logContent := logOutput.String()
	if !strings.Contains(logContent, "RUNTIME") {
		t.Error("Panic should be logged as runtime error")
	}
	
	if !strings.Contains(logContent, "recovered from panic") {
		t.Error("Should log panic recovery")
	}
	
	if !strings.Contains(logContent, "test panic for recovery") {
		t.Error("Should log original panic message")
	}
}

// TestFailingHandler is a test handler that always returns an error
type TestFailingHandler struct{}

func (h *TestFailingHandler) Command() string {
	return "fail"
}

func (h *TestFailingHandler) Handle(ctx context.Context, cmdCtx *CommandContext) error {
	return errors.New("test handler failure")
}