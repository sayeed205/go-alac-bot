package bot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gotd/td/tg"
)

// CommandRouter handles routing of commands to their respective handlers
type CommandRouter struct {
	handlers map[string]CommandHandler
	logger   *log.Logger
}

// NewCommandRouter creates a new command router instance
func NewCommandRouter(logger *log.Logger) *CommandRouter {
	return &CommandRouter{
		handlers: make(map[string]CommandHandler),
		logger:   logger,
	}
}

// RegisterHandler registers a command handler for a specific command
func (r *CommandRouter) RegisterHandler(handler CommandHandler) {
	command := handler.Command()
	r.handlers[command] = handler
	r.logger.Printf("Registered handler for command: /%s", command)
}

// RouteCommand processes an incoming message and routes it to the appropriate handler
func (r *CommandRouter) RouteCommand(ctx context.Context, update *tg.UpdateNewMessage) error {
	// Extract command context from the update
	cmdCtx, err := r.extractCommandContext(update)
	if err != nil {
		return fmt.Errorf("failed to extract command context: %w", err)
	}

	// Skip if not a command
	if cmdCtx.Command == "" {
		return nil
	}

	// Find the appropriate handler
	handler, exists := r.handlers[cmdCtx.Command]
	if !exists {
		r.logger.Printf("No handler found for command: /%s", cmdCtx.Command)
		return nil // Not an error, just no handler available
	}

	// Execute the handler
	r.logger.Printf("Routing command /%s to handler (user: %d, chat: %d)", 
		cmdCtx.Command, cmdCtx.UserID, cmdCtx.ChatID)
	
	if err := handler.Handle(ctx, cmdCtx); err != nil {
		return fmt.Errorf("handler failed for command /%s: %w", cmdCtx.Command, err)
	}

	return nil
}

// extractCommandContext extracts command context information from a Telegram update
func (r *CommandRouter) extractCommandContext(update *tg.UpdateNewMessage) (*CommandContext, error) {
	message, ok := update.Message.(*tg.Message)
	if !ok {
		return nil, fmt.Errorf("update does not contain a message")
	}

	// Extract message text
	messageText := ""
	if message.Message != "" {
		messageText = message.Message
	}

	// Check if this is a command (starts with /)
	if !strings.HasPrefix(messageText, "/") {
		return &CommandContext{
			Update:    update,
			Timestamp: time.Now(),
		}, nil
	}

	// Parse command and arguments
	parts := strings.SplitN(messageText[1:], " ", 2) // Remove leading slash
	command := parts[0]
	args := ""
	if len(parts) > 1 {
		args = parts[1]
	}

	// Extract user information
	var userID int64
	var username, firstName, lastName string

	if fromUser, ok := message.FromID.(*tg.PeerUser); ok {
		userID = fromUser.UserID
	}

	// Extract chat ID
	var chatID int64
	switch peer := message.PeerID.(type) {
	case *tg.PeerUser:
		chatID = peer.UserID
	case *tg.PeerChat:
		chatID = peer.ChatID
	case *tg.PeerChannel:
		chatID = peer.ChannelID
	}

	return &CommandContext{
		Update:    update,
		UserID:    userID,
		ChatID:    chatID,
		Username:  username,
		FirstName: firstName,
		LastName:  lastName,
		Command:   command,
		Args:      args,
		Timestamp: time.Now(),
	}, nil
}

// GetRegisteredCommands returns a list of all registered commands
func (r *CommandRouter) GetRegisteredCommands() []string {
	commands := make([]string, 0, len(r.handlers))
	for command := range r.handlers {
		commands = append(commands, command)
	}
	return commands
}

// HasHandler returns true if a handler is registered for the given command
func (r *CommandRouter) HasHandler(command string) bool {
	_, exists := r.handlers[command]
	return exists
}