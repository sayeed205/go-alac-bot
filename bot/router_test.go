package bot

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/gotd/td/tg"
)

// MockCommandHandler is a test implementation of CommandHandler
type MockCommandHandler struct {
	command     string
	handleCalls int
	lastContext *CommandContext
}

func (m *MockCommandHandler) Handle(ctx context.Context, cmdCtx *CommandContext) error {
	m.handleCalls++
	m.lastContext = cmdCtx
	return nil
}

func (m *MockCommandHandler) Command() string {
	return m.command
}

func TestCommandRouter_RegisterHandler(t *testing.T) {
	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	router := NewCommandRouter(logger)

	handler := &MockCommandHandler{command: "test"}
	router.RegisterHandler(handler)

	if !router.HasHandler("test") {
		t.Error("Expected handler to be registered for 'test' command")
	}

	commands := router.GetRegisteredCommands()
	if len(commands) != 1 || commands[0] != "test" {
		t.Errorf("Expected registered commands to contain 'test', got: %v", commands)
	}
}

func TestCommandRouter_ExtractCommandContext(t *testing.T) {
	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	router := NewCommandRouter(logger)

	// Create a mock update with a command message
	message := &tg.Message{
		Message: "/start hello world",
		PeerID:  &tg.PeerUser{UserID: 12345},
		FromID:  &tg.PeerUser{UserID: 12345},
	}

	update := &tg.UpdateNewMessage{
		Message: message,
	}

	cmdCtx, err := router.extractCommandContext(update)
	if err != nil {
		t.Fatalf("Failed to extract command context: %v", err)
	}

	if cmdCtx.Command != "start" {
		t.Errorf("Expected command 'start', got: %s", cmdCtx.Command)
	}

	if cmdCtx.Args != "hello world" {
		t.Errorf("Expected args 'hello world', got: %s", cmdCtx.Args)
	}

	if cmdCtx.UserID != 12345 {
		t.Errorf("Expected UserID 12345, got: %d", cmdCtx.UserID)
	}

	if cmdCtx.ChatID != 12345 {
		t.Errorf("Expected ChatID 12345, got: %d", cmdCtx.ChatID)
	}
}

func TestCommandRouter_RouteCommand(t *testing.T) {
	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	router := NewCommandRouter(logger)

	handler := &MockCommandHandler{command: "ping"}
	router.RegisterHandler(handler)

	// Create a mock update with a ping command
	message := &tg.Message{
		Message: "/ping",
		PeerID:  &tg.PeerUser{UserID: 12345},
		FromID:  &tg.PeerUser{UserID: 12345},
	}

	update := &tg.UpdateNewMessage{
		Message: message,
	}

	ctx := context.Background()
	err := router.RouteCommand(ctx, update)
	if err != nil {
		t.Fatalf("Failed to route command: %v", err)
	}

	if handler.handleCalls != 1 {
		t.Errorf("Expected handler to be called once, got: %d", handler.handleCalls)
	}

	if handler.lastContext.Command != "ping" {
		t.Errorf("Expected command 'ping', got: %s", handler.lastContext.Command)
	}
}

func TestCommandRouter_NonCommand(t *testing.T) {
	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	router := NewCommandRouter(logger)

	handler := &MockCommandHandler{command: "test"}
	router.RegisterHandler(handler)

	// Create a mock update with a regular message (not a command)
	message := &tg.Message{
		Message: "hello world",
		PeerID:  &tg.PeerUser{UserID: 12345},
		FromID:  &tg.PeerUser{UserID: 12345},
	}

	update := &tg.UpdateNewMessage{
		Message: message,
	}

	ctx := context.Background()
	err := router.RouteCommand(ctx, update)
	if err != nil {
		t.Fatalf("Failed to route non-command: %v", err)
	}

	// Handler should not be called for non-commands
	if handler.handleCalls != 0 {
		t.Errorf("Expected handler not to be called for non-command, got: %d calls", handler.handleCalls)
	}
}