package bot

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gotd/td/tg"
)

// PingHandler implements CommandHandler for the /ping command
type PingHandler struct {
	client *TelegramBot
	logger *log.Logger
}

// NewPingHandler creates a new PingHandler instance
func NewPingHandler(client *TelegramBot, logger *log.Logger) *PingHandler {
	return &PingHandler{
		client: client,
		logger: logger,
	}
}

// Command returns the command string this handler processes
func (h *PingHandler) Command() string {
	return "ping"
}

// Handle processes the /ping command and sends a pong response with timestamp and latency
func (h *PingHandler) Handle(ctx context.Context, cmdCtx *CommandContext) error {
	startTime := time.Now()
	
	h.logger.Printf("Processing /ping command for user %d in chat %d", cmdCtx.UserID, cmdCtx.ChatID)
	
	// Create context with short timeout for immediate response
	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	
	// Calculate latency from command timestamp to processing start
	commandLatency := startTime.Sub(cmdCtx.Timestamp)
	
	// Create pong message with timestamp and latency information
	pongMessage := h.createPongMessage(startTime, commandLatency)
	
	// Send the pong response immediately
	if err := h.sendMessage(timeoutCtx, cmdCtx.ChatID, pongMessage); err != nil {
		h.logger.Printf("Failed to send pong message to chat %d: %v", cmdCtx.ChatID, err)
		return fmt.Errorf("failed to send pong message: %w", err)
	}
	
	// Log successful processing with total response time
	totalResponseTime := time.Since(startTime)
	h.logger.Printf("Successfully processed /ping command for user %d (response time: %v, command latency: %v)", 
		cmdCtx.UserID, totalResponseTime, commandLatency)
	
	return nil
}

// createPongMessage creates a pong response with timestamp and latency information
func (h *PingHandler) createPongMessage(responseTime time.Time, commandLatency time.Duration) string {
	return fmt.Sprintf("🏓 **Pong!**\n\n"+
		"📅 **Timestamp:** %s\n"+
		"⚡ **Command Latency:** %v\n"+
		"✅ **Status:** Bot is responsive and operational",
		responseTime.Format("2006-01-02 15:04:05 MST"),
		commandLatency.Round(time.Millisecond))
}

// sendMessage sends a text message to the specified chat
func (h *PingHandler) sendMessage(ctx context.Context, chatID int64, message string) error {
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