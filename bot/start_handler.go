package bot

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gotd/td/tg"
)

// StartHandler implements CommandHandler for the /start command
type StartHandler struct {
	client *TelegramBot
	logger *log.Logger
}

// NewStartHandler creates a new StartHandler instance
func NewStartHandler(client *TelegramBot, logger *log.Logger) *StartHandler {
	return &StartHandler{
		client: client,
		logger: logger,
	}
}

// Command returns the command string this handler processes
func (h *StartHandler) Command() string {
	return "start"
}

// Handle processes the /start command and sends a welcome message
func (h *StartHandler) Handle(ctx context.Context, cmdCtx *CommandContext) error {
	startTime := time.Now()
	
	h.logger.Printf("Processing /start command for user %d in chat %d", cmdCtx.UserID, cmdCtx.ChatID)
	
	// Create context with timeout to meet 5-second requirement
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	// Extract user information for personalized welcome message
	userName := h.extractUserName(cmdCtx)
	
	// Create welcome message
	welcomeMessage := h.createWelcomeMessage(userName)
	
	// Send the welcome message
	if err := h.sendMessage(timeoutCtx, cmdCtx.ChatID, welcomeMessage); err != nil {
		h.logger.Printf("Failed to send welcome message to chat %d: %v", cmdCtx.ChatID, err)
		return fmt.Errorf("failed to send welcome message: %w", err)
	}
	
	// Log successful processing with timing
	processingTime := time.Since(startTime)
	h.logger.Printf("Successfully processed /start command for user %d (took %v)", 
		cmdCtx.UserID, processingTime)
	
	return nil
}

// extractUserName extracts the best available name for the user
func (h *StartHandler) extractUserName(cmdCtx *CommandContext) string {
	// Priority: FirstName > Username > "there" (fallback)
	if cmdCtx.FirstName != "" {
		if cmdCtx.LastName != "" {
			return cmdCtx.FirstName + " " + cmdCtx.LastName
		}
		return cmdCtx.FirstName
	}
	
	if cmdCtx.Username != "" {
		return "@" + cmdCtx.Username
	}
	
	return "there"
}

// createWelcomeMessage creates a personalized welcome message
func (h *StartHandler) createWelcomeMessage(userName string) string {
	return fmt.Sprintf("👋 Hello %s!\n\n"+
		"Welcome to the bot! I'm here to help you.\n\n"+
		"Available commands:\n"+
		"• /start - Show this welcome message\n"+
		"• /ping - Check if the bot is responsive\n\n"+
		"Feel free to try any of these commands!", userName)
}

// sendMessage sends a text message to the specified chat
func (h *StartHandler) sendMessage(ctx context.Context, chatID int64, message string) error {
	if h.client == nil || h.client.GetClient() == nil {
		return fmt.Errorf("bot client is not initialized")
	}
	
	// For bot API, we need to determine the correct peer type
	// In most cases for bots, we're dealing with private chats (users)
	var peer tg.InputPeerClass
	
	// If chatID is positive, it's likely a user chat
	if chatID > 0 {
		peer = &tg.InputPeerUser{UserID: chatID}
	} else {
		// For negative chat IDs, it could be a group or channel
		// For now, we'll treat it as a chat (group)
		peer = &tg.InputPeerChat{ChatID: -chatID}
	}
	
	// Create the message request
	request := &tg.MessagesSendMessageRequest{
		Peer:    peer,
		Message: message,
		RandomID: time.Now().UnixNano(), // Add random ID to prevent duplicate messages
	}
	
	// Send the message using gotgproto client
	_, err := h.client.GetClient().API().MessagesSendMessage(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to send message via Telegram API: %w", err)
	}
	
	return nil
}