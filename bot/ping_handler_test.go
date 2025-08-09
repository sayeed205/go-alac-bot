package bot

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/gotd/td/tg"
	"go-alac-bot/config"
)

func TestPingHandler_Command(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	cfg := &config.BotConfig{
		Token:   "test_token",
		APIID:   12345,
		APIHash: "test_hash",
	}
	
	bot, err := NewTelegramBot(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create bot: %v", err)
	}
	
	handler := NewPingHandler(bot, logger)
	
	expected := "ping"
	if got := handler.Command(); got != expected {
		t.Errorf("PingHandler.Command() = %v, want %v", got, expected)
	}
}

func TestPingHandler_Handle_WithoutClient(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	cfg := &config.BotConfig{
		Token:   "test_token",
		APIID:   12345,
		APIHash: "test_hash",
	}
	
	bot, err := NewTelegramBot(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create bot: %v", err)
	}
	
	handler := NewPingHandler(bot, logger)
	
	// Create test command context
	cmdCtx := &CommandContext{
		Update: &tg.UpdateNewMessage{
			Message: &tg.Message{
				ID:      1,
				Message: "/ping",
			},
		},
		UserID:    12345,
		ChatID:    67890,
		Username:  "testuser",
		FirstName: "Test",
		LastName:  "User",
		Command:   "ping",
		Args:      "",
		Timestamp: time.Now(),
	}
	
	ctx := context.Background()
	
	// This should fail because the client is not initialized (no actual Telegram connection)
	err = handler.Handle(ctx, cmdCtx)
	if err == nil {
		t.Error("Expected error when client is not initialized, but got nil")
	}
	
	// Check that error message indicates client initialization issue
	expectedErrorSubstring := "bot client is not initialized"
	if err != nil && !containsString(err.Error(), expectedErrorSubstring) {
		t.Errorf("Expected error to contain '%s', but got: %v", expectedErrorSubstring, err)
	}
}

func TestPingHandler_CreatePongMessage(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	cfg := &config.BotConfig{
		Token:   "test_token",
		APIID:   12345,
		APIHash: "test_hash",
	}
	
	bot, err := NewTelegramBot(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create bot: %v", err)
	}
	
	handler := NewPingHandler(bot, logger)
	
	// Test createPongMessage
	testTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	testLatency := 50 * time.Millisecond
	
	message := handler.createPongMessage(testTime, testLatency)
	
	// Check that message contains expected elements
	expectedSubstrings := []string{
		"🏓 **Pong!**",
		"📅 **Timestamp:**",
		"⚡ **Command Latency:**",
		"✅ **Status:** Bot is responsive and operational",
		"2024-01-01 12:00:00",
		"50ms",
	}
	
	for _, expected := range expectedSubstrings {
		if !containsString(message, expected) {
			t.Errorf("Expected message to contain '%s', but message was: %s", expected, message)
		}
	}
}

func TestPingHandler_TimingRequirements(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	cfg := &config.BotConfig{
		Token:   "test_token",
		APIID:   12345,
		APIHash: "test_hash",
	}
	
	bot, err := NewTelegramBot(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create bot: %v", err)
	}
	
	handler := NewPingHandler(bot, logger)
	
	// Test that handler processes quickly (even without sending message)
	startTime := time.Now()
	
	cmdCtx := &CommandContext{
		Update: &tg.UpdateNewMessage{
			Message: &tg.Message{
				ID:      1,
				Message: "/ping",
			},
		},
		UserID:    12345,
		ChatID:    67890,
		Username:  "testuser",
		FirstName: "Test",
		LastName:  "User",
		Command:   "ping",
		Args:      "",
		Timestamp: startTime.Add(-10 * time.Millisecond), // Simulate 10ms command latency
	}
	
	ctx := context.Background()
	
	// Even though this will fail due to no client, we can measure processing time
	_ = handler.Handle(ctx, cmdCtx)
	
	processingTime := time.Since(startTime)
	
	// Processing should be very fast (under 100ms even with timeout)
	maxExpectedTime := 100 * time.Millisecond
	if processingTime > maxExpectedTime {
		t.Errorf("Handler took too long to process: %v (expected under %v)", processingTime, maxExpectedTime)
	}
}

func TestNewPingHandler(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	cfg := &config.BotConfig{
		Token:   "test_token",
		APIID:   12345,
		APIHash: "test_hash",
	}
	
	bot, err := NewTelegramBot(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create bot: %v", err)
	}
	
	handler := NewPingHandler(bot, logger)
	
	if handler == nil {
		t.Error("NewPingHandler returned nil")
	}
	
	if handler.client != bot {
		t.Error("PingHandler client not set correctly")
	}
	
	if handler.logger != logger {
		t.Error("PingHandler logger not set correctly")
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > len(substr) && (s[:len(substr)] == substr || 
		s[len(s)-len(substr):] == substr || 
		containsStringHelper(s, substr))))
}

func containsStringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}