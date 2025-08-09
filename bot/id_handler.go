package bot

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gotd/td/tg"
)

// IDHandler implements CommandHandler for the /id command
type IDHandler struct {
	client       *TelegramBot
	logger       *log.Logger
	errorHandler *ErrorHandler
}

// NewIDHandler creates a new IDHandler instance
func NewIDHandler(client *TelegramBot, logger *log.Logger) *IDHandler {
	handler := &IDHandler{
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
func (h *IDHandler) Command() string {
	return "id"
}

// Handle processes the /id command and returns chat or user ID
func (h *IDHandler) Handle(ctx context.Context, cmdCtx *CommandContext) error {
	startTime := time.Now()
	
	h.logger.Printf("Processing /id command for user %d in chat %d", cmdCtx.UserID, cmdCtx.ChatID)
	
	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	var message string
	
	// Check if this is a reply to another message
	if cmdCtx.ReplyToMessageID != 0 {
		// Get the replied message to extract user ID
		repliedMessage, err := h.getRepliedMessage(timeoutCtx, cmdCtx)
		if err != nil {
			h.logger.Printf("Failed to get replied message: %v", err)
			// Fallback to showing chat ID
			message = h.createChatIDMessage(cmdCtx.ChatID)
		} else {
			message = h.createUserIDMessage(repliedMessage.FromID)
		}
	} else {
		// No reply, show chat ID
		message = h.createChatIDMessage(cmdCtx.ChatID)
	}
	
	// Send the ID message with markdown formatting
	if err := h.sendMarkdownMessage(timeoutCtx, cmdCtx.ChatID, message); err != nil {
		h.logger.Printf("Failed to send ID message to chat %d: %v", cmdCtx.ChatID, err)
		
		// Use error handler if available for network errors
		if h.errorHandler != nil && h.errorHandler.IsNetworkError(err) {
			return h.errorHandler.HandleNetworkError(err, true)
		}
		
		return fmt.Errorf("failed to send ID message: %w", err)
	}
	
	// Log successful processing with timing
	processingTime := time.Since(startTime)
	h.logger.Printf("Successfully processed /id command for user %d (took %v)", 
		cmdCtx.UserID, processingTime)
	
	return nil
}

// getRepliedMessage retrieves the message that was replied to
func (h *IDHandler) getRepliedMessage(ctx context.Context, cmdCtx *CommandContext) (*tg.Message, error) {
	if h.client == nil || h.client.GetClient() == nil {
		return nil, fmt.Errorf("bot client is not initialized")
	}
	
	// Get the message by ID - use the correct API method
	messageIDs := []tg.InputMessageClass{
		&tg.InputMessageID{ID: int(cmdCtx.ReplyToMessageID)},
	}
	
	response, err := h.client.GetClient().API().MessagesGetMessages(ctx, messageIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get replied message: %w", err)
	}
	
	// Extract messages from response
	switch msgs := response.(type) {
	case *tg.MessagesMessages:
		if len(msgs.Messages) > 0 {
			if msg, ok := msgs.Messages[0].(*tg.Message); ok {
				return msg, nil
			}
		}
	case *tg.MessagesMessagesSlice:
		if len(msgs.Messages) > 0 {
			if msg, ok := msgs.Messages[0].(*tg.Message); ok {
				return msg, nil
			}
		}
	}
	
	return nil, fmt.Errorf("replied message not found")
}

// createUserIDMessage creates a message showing the user ID
func (h *IDHandler) createUserIDMessage(fromID tg.PeerClass) string {
	var userID int64
	
	switch peer := fromID.(type) {
	case *tg.PeerUser:
		userID = peer.UserID
	case *tg.PeerChat:
		userID = peer.ChatID
	case *tg.PeerChannel:
		userID = peer.ChannelID
	default:
		return "Unable to determine user ID"
	}
	
	return fmt.Sprintf("User id: `%d`\n(Click/Tap to copy)", userID)
}

// createChatIDMessage creates a message showing the chat ID
func (h *IDHandler) createChatIDMessage(chatID int64) string {
	return fmt.Sprintf("Chat id: `%d`\n(Click/Tap to copy)", chatID)
}

// sendMarkdownMessage sends a text message with markdown formatting to the specified chat
func (h *IDHandler) sendMarkdownMessage(ctx context.Context, chatID int64, message string) error {
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
	
	// Create message entities for markdown formatting
	entities := h.parseMarkdownEntities(message)
	
	// Create the message request with markdown entities
	request := &tg.MessagesSendMessageRequest{
		Peer:     peer,
		Message:  h.stripMarkdownSyntax(message),
		RandomID: time.Now().UnixNano(),
		Entities: entities,
	}
	
	// Send the message using gotgproto client
	_, err := h.client.GetClient().API().MessagesSendMessage(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to send markdown message via Telegram API: %w", err)
	}
	
	return nil
}

// parseMarkdownEntities parses markdown syntax and creates message entities
func (h *IDHandler) parseMarkdownEntities(text string) []tg.MessageEntityClass {
	var entities []tg.MessageEntityClass
	offset := 0
	
	i := 0
	for i < len(text) {
		char := text[i]
		
		if char == '`' {
			// Find closing backtick for code
			if end := h.findClosing(text, i+1, '`'); end != -1 {
				codeText := text[i+1 : end]
				entities = append(entities, &tg.MessageEntityCode{
					Offset: offset,
					Length: len(codeText),
				})
				offset += len(codeText)
				i = end + 1
				continue
			}
		}
		
		// Add regular character
		offset++
		i++
	}
	
	return entities
}

// findClosing finds the closing character for markdown syntax
func (h *IDHandler) findClosing(text string, start int, char byte) int {
	for i := start; i < len(text); i++ {
		if text[i] == char {
			return i
		}
	}
	return -1
}

// stripMarkdownSyntax removes markdown syntax characters from text
func (h *IDHandler) stripMarkdownSyntax(text string) string {
	result := ""
	i := 0
	
	for i < len(text) {
		char := text[i]
		
		if char == '`' {
			// Skip code markers
			if end := h.findClosing(text, i+1, '`'); end != -1 {
				result += text[i+1 : end]
				i = end + 1
				continue
			}
		}
		
		// Add regular character
		result += string(char)
		i++
	}
	
	return result
}