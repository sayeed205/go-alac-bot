package bot

import (
	"log"
	"os"
	"testing"

	"go-alac-bot/config"
)

func TestNewTelegramBot(t *testing.T) {
	// Create test config
	cfg := &config.BotConfig{
		Token:    "test_token",
		APIID:    12345,
		APIHash:  "test_hash",
		LogLevel: "INFO",
	}
	
	// Create test logger
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	
	// Test successful creation
	bot, err := NewTelegramBot(cfg, logger)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	
	if bot == nil {
		t.Fatal("Expected bot to be created, got nil")
	}
	
	if bot.config != cfg {
		t.Error("Expected config to be set correctly")
	}
	
	if bot.logger != logger {
		t.Error("Expected logger to be set correctly")
	}
	
	if bot.ctx == nil {
		t.Error("Expected context to be initialized")
	}
	
	if bot.cancel == nil {
		t.Error("Expected cancel function to be initialized")
	}
}

func TestNewTelegramBot_NilConfig(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	
	bot, err := NewTelegramBot(nil, logger)
	if err == nil {
		t.Fatal("Expected error for nil config, got nil")
	}
	
	if bot != nil {
		t.Error("Expected bot to be nil when config is nil")
	}
	
	expectedError := "config cannot be nil"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestNewTelegramBot_NilLogger(t *testing.T) {
	cfg := &config.BotConfig{
		Token:    "test_token",
		APIID:    12345,
		APIHash:  "test_hash",
		LogLevel: "INFO",
	}
	
	bot, err := NewTelegramBot(cfg, nil)
	if err == nil {
		t.Fatal("Expected error for nil logger, got nil")
	}
	
	if bot != nil {
		t.Error("Expected bot to be nil when logger is nil")
	}
	
	expectedError := "logger cannot be nil"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestTelegramBot_IsRunning(t *testing.T) {
	cfg := &config.BotConfig{
		Token:    "test_token",
		APIID:    12345,
		APIHash:  "test_hash",
		LogLevel: "INFO",
	}
	
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	
	bot, err := NewTelegramBot(cfg, logger)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	
	// Initially should not be running (client not started)
	if bot.IsRunning() {
		t.Error("Expected bot to not be running initially")
	}
	
	// After stopping context, should not be running
	bot.Stop()
	if bot.IsRunning() {
		t.Error("Expected bot to not be running after stop")
	}
}

func TestTelegramBot_GetClient(t *testing.T) {
	cfg := &config.BotConfig{
		Token:    "test_token",
		APIID:    12345,
		APIHash:  "test_hash",
		LogLevel: "INFO",
	}
	
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	
	bot, err := NewTelegramBot(cfg, logger)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	
	// Initially client should be nil
	client := bot.GetClient()
	if client != nil {
		t.Error("Expected client to be nil before Start() is called")
	}
}