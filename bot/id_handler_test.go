package bot

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/gotd/td/tg"
)

func TestIDHandler_Command(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	handler := NewIDHandler(nil, logger)
	
	expected := "id"
	if got := handler.Command(); got != expected {
		t.Errorf("Command() = %v, want %v", got, expected)
	}
}

func TestIDHandler_Handle_ChatID(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	
	// Create a mock bot client (nil for testing)
	handler := NewIDHandler(nil, logger)
	
	// Create test command context without reply
	cmdCtx := &CommandContext{
		UserID:           12345,
		ChatID:           67890,
		Command:          "id",
		Args:             "",
		ReplyToMessageID: 0, // No reply
		Timestamp:        time.Now(),
	}
	
	// Test that handler doesn't panic with nil client
	// In real implementation, this would send a message
	err := handler.Handle(context.Background(), cmdCtx)
	
	// With nil client, we expect an error
	if err == nil {
		t.Error("Expected error with nil client, got nil")
	}
}

func TestIDHandler_CreateChatIDMessage(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	handler := NewIDHandler(nil, logger)
	
	chatID := int64(12345)
	expected := "Chat id: `12345`\n(Click/Tap to copy)"
	
	result := handler.createChatIDMessage(chatID)
	if result != expected {
		t.Errorf("createChatIDMessage() = %v, want %v", result, expected)
	}
}

func TestIDHandler_CreateUserIDMessage(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	handler := NewIDHandler(nil, logger)
	
	// Test with PeerUser
	userPeer := &tg.PeerUser{UserID: 54321}
	expected := "User id: `54321`\n(Click/Tap to copy)"
	
	result := handler.createUserIDMessage(userPeer)
	if result != expected {
		t.Errorf("createUserIDMessage() = %v, want %v", result, expected)
	}
	
	// Test with PeerChat
	chatPeer := &tg.PeerChat{ChatID: 98765}
	expected = "User id: `98765`\n(Click/Tap to copy)"
	
	result = handler.createUserIDMessage(chatPeer)
	if result != expected {
		t.Errorf("createUserIDMessage() = %v, want %v", result, expected)
	}
}

func TestIDHandler_ParseMarkdownEntities(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	handler := NewIDHandler(nil, logger)
	
	text := "Chat id: `12345`\n(Click/Tap to copy)"
	entities := handler.parseMarkdownEntities(text)
	
	// Should have one code entity
	if len(entities) != 1 {
		t.Errorf("Expected 1 entity, got %d", len(entities))
		return
	}
	
	// Check if it's a code entity
	if codeEntity, ok := entities[0].(*tg.MessageEntityCode); ok {
		if codeEntity.Offset != 9 { // "Chat id: " is 9 characters
			t.Errorf("Expected offset 9, got %d", codeEntity.Offset)
		}
		if codeEntity.Length != 5 { // "12345" is 5 characters
			t.Errorf("Expected length 5, got %d", codeEntity.Length)
		}
	} else {
		t.Error("Expected MessageEntityCode, got different type")
	}
}

func TestIDHandler_StripMarkdownSyntax(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	handler := NewIDHandler(nil, logger)
	
	text := "Chat id: `12345`\n(Click/Tap to copy)"
	expected := "Chat id: 12345\n(Click/Tap to copy)"
	
	result := handler.stripMarkdownSyntax(text)
	if result != expected {
		t.Errorf("stripMarkdownSyntax() = %v, want %v", result, expected)
	}
}