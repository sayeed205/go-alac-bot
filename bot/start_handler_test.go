package bot

import (
	"context"
	"log"
	"os"
	"testing"
	"time"
)

func TestStartHandler_Command(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	handler := NewStartHandler(nil, logger)
	
	expected := "start"
	if got := handler.Command(); got != expected {
		t.Errorf("StartHandler.Command() = %v, want %v", got, expected)
	}
}

func TestStartHandler_ExtractUserName(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	handler := NewStartHandler(nil, logger)
	
	tests := []struct {
		name     string
		cmdCtx   *CommandContext
		expected string
	}{
		{
			name: "First and last name available",
			cmdCtx: &CommandContext{
				FirstName: "John",
				LastName:  "Doe",
				Username:  "johndoe",
			},
			expected: "John Doe",
		},
		{
			name: "Only first name available",
			cmdCtx: &CommandContext{
				FirstName: "John",
				Username:  "johndoe",
			},
			expected: "John",
		},
		{
			name: "Only username available",
			cmdCtx: &CommandContext{
				Username: "johndoe",
			},
			expected: "@johndoe",
		},
		{
			name: "No name information available",
			cmdCtx: &CommandContext{
				UserID: 12345,
			},
			expected: "there",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.extractUserName(tt.cmdCtx)
			if result != tt.expected {
				t.Errorf("extractUserName() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestStartHandler_CreateWelcomeMessage(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	handler := NewStartHandler(nil, logger)
	
	tests := []struct {
		name     string
		userName string
	}{
		{
			name:     "With full name",
			userName: "John Doe",
		},
		{
			name:     "With username",
			userName: "@johndoe",
		},
		{
			name:     "With fallback",
			userName: "there",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := handler.createWelcomeMessage(tt.userName)
			
			// Check that message contains the user name
			if len(message) == 0 {
				t.Error("createWelcomeMessage() returned empty message")
			}
			
			// Check that message contains expected elements
			expectedElements := []string{
				"Hello " + tt.userName,
				"Welcome to the bot",
				"/start",
				"/ping",
			}
			
			for _, element := range expectedElements {
				if !contains(message, element) {
					t.Errorf("createWelcomeMessage() missing expected element: %s", element)
				}
			}
		})
	}
}

func TestStartHandler_Handle_WithoutClient(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	handler := NewStartHandler(nil, logger)
	
	ctx := context.Background()
	cmdCtx := &CommandContext{
		UserID:    12345,
		ChatID:    12345,
		FirstName: "John",
		Command:   "start",
		Timestamp: time.Now(),
	}
	
	// This should fail because no client is provided
	err := handler.Handle(ctx, cmdCtx)
	if err == nil {
		t.Error("Handle() should return error when client is nil")
	}
}

func TestStartHandler_Handle_Timing(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	
	// Create a mock bot that will simulate a slow response
	mockBot := &TelegramBot{}
	handler := NewStartHandler(mockBot, logger)
	
	ctx := context.Background()
	cmdCtx := &CommandContext{
		UserID:    12345,
		ChatID:    12345,
		FirstName: "John",
		Command:   "start",
		Timestamp: time.Now(),
	}
	
	start := time.Now()
	
	// This will fail due to no real client, but we can test timing
	_ = handler.Handle(ctx, cmdCtx)
	
	elapsed := time.Since(start)
	
	// Should complete quickly (within reasonable time for processing)
	if elapsed > 1*time.Second {
		t.Errorf("Handle() took too long: %v, should be much faster for processing", elapsed)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > len(substr) && (s[:len(substr)] == substr || 
		s[len(s)-len(substr):] == substr || 
		containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}