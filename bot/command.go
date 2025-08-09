package bot

import (
	"context"
	"time"

	"github.com/gotd/td/tg"
)

// CommandHandler defines the interface for handling bot commands
type CommandHandler interface {
	// Handle processes a command with the given context
	Handle(ctx context.Context, cmdCtx *CommandContext) error
	// Command returns the command string this handler processes (e.g., "start", "ping")
	Command() string
}

// CommandContext provides context information for command processing
type CommandContext struct {
	// Update contains the original Telegram update
	Update *tg.UpdateNewMessage
	// UserID is the ID of the user who sent the command
	UserID int64
	// ChatID is the ID of the chat where the command was sent
	ChatID int64
	// MessageID is the ID of the message containing the command
	MessageID int
	// Username is the username of the user (may be empty)
	Username string
	// FirstName is the first name of the user
	FirstName string
	// LastName is the last name of the user (may be empty)
	LastName string
	// Command is the command string without the leading slash
	Command string
	// Args contains command arguments (text after the command)
	Args string
	// ReplyToMessageID is the ID of the message being replied to (0 if not a reply)
	ReplyToMessageID int32
	// Timestamp is when the command was received
	Timestamp time.Time
}
