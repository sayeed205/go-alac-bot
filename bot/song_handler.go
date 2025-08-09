package bot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gotd/td/tg"
)

// SongHandler implements CommandHandler for the /song command
type SongHandler struct {
	client       *TelegramBot
	logger       *log.Logger
	errorHandler *ErrorHandler
}

// NewSongHandler creates a new SongHandler instance
func NewSongHandler(client *TelegramBot, logger *log.Logger) *SongHandler {
	handler := &SongHandler{
		client: client,
		logger: logger,
	}
	
	// Set error handler if client is available
	if client != nil {
		handler.errorHandler = client.GetErrorHandler()
	}
	
	return handler
}

// Command returns the command string this handler processes
func (h *SongHandler) Command() string {
	return "song"
}

// Handle processes the /song command and downloads a single song
func (h *SongHandler) Handle(ctx context.Context, cmdCtx *CommandContext) error {
	startTime := time.Now()
	
	h.logger.Printf("Processing /song command for user %d in chat %d", cmdCtx.UserID, cmdCtx.ChatID)
	
	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	// Check if URL is provided
	if strings.TrimSpace(cmdCtx.Args) == "" {
		return h.sendErrorMessage(timeoutCtx, cmdCtx.ChatID, "Please provide a song URL.")
	}
	
	// Parse and validate the URL
	songURL := strings.TrimSpace(cmdCtx.Args)
	urlMeta := ExtractURLMeta(songURL)
	if urlMeta == nil {
		return h.sendErrorMessage(timeoutCtx, cmdCtx.ChatID, "Please provide a valid Apple Music URL.")
	}
	
	// Log the song request with parsed metadata
	h.logger.Printf("Received song request: %s (Type: %s, ID: %s, Storefront: %s)", 
		songURL, urlMeta.URLType, urlMeta.ID, urlMeta.Storefront)
	
	// Send processing message
	if err := h.sendMessage(timeoutCtx, cmdCtx.ChatID, "🎵 Processing your request..."); err != nil {
		h.logger.Printf("Failed to send processing message: %v", err)
	}
	
	// TODO: Implement actual song download logic here
	// For now, just send a placeholder response with parsed metadata
	responseMessage := h.createSongResponse(songURL, urlMeta)
	
	// Send the response
	if err := h.sendMessage(timeoutCtx, cmdCtx.ChatID, responseMessage); err != nil {
		h.logger.Printf("Failed to send song response to chat %d: %v", cmdCtx.ChatID, err)
		
		// Use error handler if available for network errors
		if h.errorHandler != nil && h.errorHandler.IsNetworkError(err) {
			return h.errorHandler.HandleNetworkError(err, true)
		}
		
		return fmt.Errorf("failed to send song response: %w", err)
	}
	
	// Log successful processing with timing
	processingTime := time.Since(startTime)
	h.logger.Printf("Successfully processed /song command for user %d (took %v)", 
		cmdCtx.UserID, processingTime)
	
	return nil
}

// createSongResponse creates a response message for the song request
func (h *SongHandler) createSongResponse(url string, urlMeta *URLMeta) string {
	// Create response based on the URL type
	var typeEmoji string
	var typeName string
	
	switch urlMeta.URLType {
	case "songs":
		typeEmoji = "🎵"
		typeName = "Song"
	case "albums":
		typeEmoji = "💿"
		typeName = "Album"
	case "playlists":
		typeEmoji = "📋"
		typeName = "Playlist"
	default:
		typeEmoji = "🎵"
		typeName = "Content"
	}
	
	return fmt.Sprintf("%s %s detected!\n\n"+
		"📍 Storefront: %s\n"+
		"🆔 ID: %s\n"+
		"🔗 URL: %s\n\n"+
		"⚠️ Download functionality is currently under development.",
		typeEmoji, typeName, urlMeta.Storefront, urlMeta.ID, url)
}

// sendErrorMessage sends an error message to the user
func (h *SongHandler) sendErrorMessage(ctx context.Context, chatID int64, errorMsg string) error {
	return h.sendMessage(ctx, chatID, "❌ "+errorMsg)
}

// sendMessage sends a text message to the specified chat
func (h *SongHandler) sendMessage(ctx context.Context, chatID int64, message string) error {
	if h.client == nil || h.client.GetClient() == nil {
		return fmt.Errorf("bot client is not initialized")
	}
	
	// For bot API, we need to determine the correct peer type
	var peer tg.InputPeerClass
	
	// If chatID is positive, it's likely a user chat
	if chatID > 0 {
		peer = &tg.InputPeerUser{UserID: chatID}
	} else {
		// For negative chat IDs, it could be a group or channel
		peer = &tg.InputPeerChat{ChatID: -chatID}
	}
	
	// Create the message request
	request := &tg.MessagesSendMessageRequest{
		Peer:     peer,
		Message:  message,
		RandomID: time.Now().UnixNano(),
	}
	
	// Send the message using gotgproto client
	_, err := h.client.GetClient().API().MessagesSendMessage(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to send message via Telegram API: %w", err)
	}
	
	return nil
}