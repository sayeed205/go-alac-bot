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
	handlers     map[string]CommandHandler
	logger       *log.Logger
	errorHandler *ErrorHandler
}

// NewCommandRouter creates a new command router instance
func NewCommandRouter(logger *log.Logger) *CommandRouter {
	return &CommandRouter{
		handlers: make(map[string]CommandHandler),
		logger:   logger,
	}
}

// SetErrorHandler sets the error handler for the router
func (r *CommandRouter) SetErrorHandler(errorHandler *ErrorHandler) {
	r.errorHandler = errorHandler
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

	// Execute the handler with panic recovery
	r.logger.Printf("Routing command /%s to handler (user: %d, chat: %d)",
		cmdCtx.Command, cmdCtx.UserID, cmdCtx.ChatID)

	// Add panic recovery for command handlers
	defer func() {
		if r.errorHandler != nil {
			r.errorHandler.RecoverFromPanic()
		}
	}()

	if err := handler.Handle(ctx, cmdCtx); err != nil {
		// Use error handler if available, otherwise return the error
		if r.errorHandler != nil {
			r.errorHandler.HandleCommandError(err, cmdCtx)
			return nil // Error has been handled, don't propagate
		}
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

	// Extract reply message ID if this is a reply
	var replyToMessageID int32
	if message.ReplyTo != nil {
		if replyHeader, ok := message.ReplyTo.(*tg.MessageReplyHeader); ok {
			replyToMessageID = int32(replyHeader.ReplyToMsgID)
		}
	}

	return &CommandContext{
		Update:           update,
		UserID:           userID,
		ChatID:           chatID,
		MessageID:        message.ID,
		Username:         username,
		FirstName:        firstName,
		LastName:         lastName,
		Command:          command,
		Args:             args,
		ReplyToMessageID: replyToMessageID,
		Timestamp:        time.Now(),
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
