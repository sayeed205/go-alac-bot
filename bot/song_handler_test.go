package bot

import (
	"context"
	"log"
	"os"
	"testing"
	"time"
)

func TestSongHandler_Command(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	handler := NewSongHandler(nil, logger)
	
	expected := "song"
	if got := handler.Command(); got != expected {
		t.Errorf("Command() = %v, want %v", got, expected)
	}
}

func TestSongHandler_URLParsing(t *testing.T) {
	testCases := []struct {
		url      string
		expected bool
	}{
		{"https://music.apple.com/in/song/never-gonna-give-you-up/1559523359", true},
		{"https://music.apple.com/in/album/never-gonna-give-you-up/1559523357?i=1559523359", true},
		{"https://music.apple.com/us/album/test/123", true},
		{"https://spotify.com/track/123", false},
		{"https://youtube.com/watch?v=123", false},
		{"not a url", false},
		{"", false},
	}
	
	for _, tc := range testCases {
		result := ExtractURLMeta(tc.url)
		isValid := result != nil
		if isValid != tc.expected {
			t.Errorf("ExtractURLMeta(%q) validity = %v, want %v", tc.url, isValid, tc.expected)
		}
	}
}

func TestSongHandler_Handle_NoURL(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	handler := NewSongHandler(nil, logger)
	
	// Create test command context without URL
	cmdCtx := &CommandContext{
		UserID:    12345,
		ChatID:    67890,
		Command:   "song",
		Args:      "", // No URL provided
		Timestamp: time.Now(),
	}
	
	// Test that handler returns error with nil client (expected behavior)
	err := handler.Handle(context.Background(), cmdCtx)
	
	// With nil client, we expect an error
	if err == nil {
		t.Error("Expected error with nil client, got nil")
	}
}

func TestSongHandler_Handle_WithURL(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	handler := NewSongHandler(nil, logger)
	
	// Create test command context with valid URL
	cmdCtx := &CommandContext{
		UserID:    12345,
		ChatID:    67890,
		Command:   "song",
		Args:      "https://music.apple.com/in/song/test/123",
		Timestamp: time.Now(),
	}
	
	// Test that handler returns error with nil client (expected behavior)
	err := handler.Handle(context.Background(), cmdCtx)
	
	// With nil client, we expect an error
	if err == nil {
		t.Error("Expected error with nil client, got nil")
	}
}

