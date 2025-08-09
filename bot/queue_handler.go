package bot

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gotd/td/tg"
)

// QueueHandler implements CommandHandler for the /queue command
type QueueHandler struct {
	client       *TelegramBot
	logger       *log.Logger
	errorHandler *ErrorHandler
	songHandler  *SongHandler
}

// NewQueueHandler creates a new QueueHandler instance
func NewQueueHandler(client *TelegramBot, logger *log.Logger, songHandler *SongHandler) *QueueHandler {
	handler := &QueueHandler{
		client:      client,
		logger:      logger,
		songHandler: songHandler,
	}

	// Set error handler if client is available
	if client != nil {
		handler.errorHandler = client.GetErrorHandler()
	}

	return handler
}

// Command returns the command string this handler processes
func (h *QueueHandler) Command() string {
	return "queue"
}

// Handle processes the /queue command and shows current queue status
func (h *QueueHandler) Handle(ctx context.Context, cmdCtx *CommandContext) error {
	startTime := time.Now()

	h.logger.Printf("Processing /queue command for user %d in chat %d", cmdCtx.UserID, cmdCtx.ChatID)

	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Get queue from song handler
	queue := h.songHandler.GetQueue()
	if queue == nil {
		return h.sendErrorMessage(timeoutCtx, cmdCtx.ChatID, "Queue system is not available.")
	}

	// Generate queue status message
	message := h.createQueueStatusMessage(queue)

	// Send the queue status message
	if err := h.sendMessage(timeoutCtx, cmdCtx.ChatID, message); err != nil {
		h.logger.Printf("Failed to send queue status message: %v", err)

		if h.errorHandler != nil && h.errorHandler.IsNetworkError(err) {
			return h.errorHandler.HandleNetworkError(err, false)
		}

		return fmt.Errorf("failed to send queue status: %w", err)
	}

	// Log successful processing with timing
	processingTime := time.Since(startTime)
	h.logger.Printf("Successfully processed /queue command for user %d (took %v)",
		cmdCtx.UserID, processingTime)

	return nil
}

// createQueueStatusMessage creates a formatted queue status message
func (h *QueueHandler) createQueueStatusMessage(queue *SongQueue) string {
	queueSize := queue.GetQueueSize()
	isProcessing := queue.IsProcessing()
	currentlyProcessing := queue.GetCurrentlyProcessing()

	message := "📊 **Song Queue Status**\n\n"

	// Queue capacity
	message += fmt.Sprintf("**Capacity:** %d/%d requests\n\n", queueSize, MaxQueueSize)

	// Current processing status
	if isProcessing && currentlyProcessing != nil {
		message += fmt.Sprintf("🎵 **Currently Processing:**\n")
		message += fmt.Sprintf("• Request ID: `%s`\n", currentlyProcessing.UniqueID)
		message += fmt.Sprintf("• From user: %d\n", currentlyProcessing.SenderID)
		elapsed := time.Since(currentlyProcessing.RequestTime)
		message += fmt.Sprintf("• Processing time: %s\n\n", elapsed.Round(time.Second))
	} else {
		message += "🎵 **Currently Processing:** None\n\n"
	}

	// Queued requests
	if queueSize > 0 {
		message += fmt.Sprintf("📋 **Queued Requests (%d):**\n", queueSize)

		// Get queue details (we need to add this method to SongQueue)
		queueInfo := queue.GetQueueInfo()
		for i, request := range queueInfo {
			message += fmt.Sprintf("%d. User %d (requested %s ago)\n",
				i+1,
				request.SenderID,
				time.Since(request.RequestTime).Round(time.Second))
		}
	} else {
		message += "📋 **Queue:** Empty\n"
	}

	message += "\n💡 Use `/song <url>` to add a new song to the queue"

	return message
}

// sendErrorMessage sends an error message to the user
func (h *QueueHandler) sendErrorMessage(ctx context.Context, chatID int64, errorMsg string) error {
	return h.sendMessage(ctx, chatID, "❌ "+errorMsg)
}

// sendMessage sends a text message to the specified chat
func (h *QueueHandler) sendMessage(ctx context.Context, chatID int64, message string) error {
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
