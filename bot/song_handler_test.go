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

func TestSongHandler_CreateSongResponse(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	handler := NewSongHandler(nil, logger)
	
	url := "https://music.apple.com/in/song/test/123"
	urlMeta := &URLMeta{
		Storefront: "in",
		URLType:    "songs",
		ID:         "123",
	}
	
	response := handler.createSongResponse(url, urlMeta)
	
	// Check that response contains the URL
	if !contains(response, url) {
		t.Errorf("createSongResponse() should contain URL %q, got %q", url, response)
	}
	
	// Check that response contains metadata
	if !contains(response, "123") {
		t.Errorf("createSongResponse() should contain ID %q, got %q", "123", response)
	}
	
	// Check that response indicates development status
	if !contains(response, "under development") {
		t.Errorf("createSongResponse() should indicate development status, got %q", response)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > len(substr) && (s[:len(substr)] == substr || 
		s[len(s)-len(substr):] == substr || 
		containsAt(s, substr))))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}