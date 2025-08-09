package bot

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gotd/td/tg"
)

// HelpHandler implements CommandHandler for the /help command
type HelpHandler struct {
	client       *TelegramBot
	logger       *log.Logger
	errorHandler *ErrorHandler
}

// NewHelpHandler creates a new HelpHandler instance
func NewHelpHandler(client *TelegramBot, logger *log.Logger) *HelpHandler {
	handler := &HelpHandler{
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
func (h *HelpHandler) Command() string {
	return "help"
}

// Handle processes the /help command and sends a help message with markdown formatting
func (h *HelpHandler) Handle(ctx context.Context, cmdCtx *CommandContext) error {
	startTime := time.Now()
	
	h.logger.Printf("Processing /help command for user %d in chat %d", cmdCtx.UserID, cmdCtx.ChatID)
	
	// Create context with timeout to meet 5-second requirement
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	// Create help message with markdown formatting
	helpMessage := h.createHelpMessage()
	
	// Send the help message with markdown formatting
	if err := h.sendMarkdownMessage(timeoutCtx, cmdCtx.ChatID, helpMessage); err != nil {
		h.logger.Printf("Failed to send help message to chat %d: %v", cmdCtx.ChatID, err)
		
		// Use error handler if available for network errors
		if h.errorHandler != nil && h.errorHandler.IsNetworkError(err) {
			return h.errorHandler.HandleNetworkError(err, true)
		}
		
		return fmt.Errorf("failed to send help message: %w", err)
	}
	
	// Log successful processing with timing
	processingTime := time.Since(startTime)
	h.logger.Printf("Successfully processed /help command for user %d (took %v)", 
		cmdCtx.UserID, processingTime)
	
	return nil
}

// createHelpMessage creates the help message with markdown formatting
func (h *HelpHandler) createHelpMessage() string {
	return `*Available Commands*

/help - Show this help message
/id - Get chat or user ID (reply to message for user ID)
/song - Download a single song
/album - Get album URLs (WIP)
/playlist - Get playlist URLs (WIP)

*Examples*

*Song examples:*
` + "`/song https://music.apple.com/in/song/never-gonna-give-you-up/1559523359`" + `

` + "`/song https://music.apple.com/in/album/never-gonna-give-you-up/1559523357?i=1559523359`" + `

*Album example:*
` + "`/album https://music.apple.com/in/album/3-originals/1559523357`" + `

*Playlist example:*
` + "`/playlist https://music.apple.com/library/playlist/p.vMO5kRQiX1xGMr`" + `

*Tip:* Tap on any URL above to copy it!`
}

// sendMarkdownMessage sends a text message with markdown formatting to the specified chat
func (h *HelpHandler) sendMarkdownMessage(ctx context.Context, chatID int64, message string) error {
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
func (h *HelpHandler) parseMarkdownEntities(text string) []tg.MessageEntityClass {
	var entities []tg.MessageEntityClass
	offset := 0
	cleanText := ""
	
	i := 0
	for i < len(text) {
		char := text[i]
		
		switch char {
		case '*':
			// Find closing asterisk for bold
			if end := h.findClosing(text, i+1, '*'); end != -1 {
				boldText := text[i+1 : end]
				entities = append(entities, &tg.MessageEntityBold{
					Offset: offset,
					Length: len(boldText),
				})
				cleanText += boldText
				offset += len(boldText)
				i = end + 1
				continue
			}
		case '`':
			// Find closing backtick for code
			if end := h.findClosing(text, i+1, '`'); end != -1 {
				codeText := text[i+1 : end]
				entities = append(entities, &tg.MessageEntityCode{
					Offset: offset,
					Length: len(codeText),
				})
				cleanText += codeText
				offset += len(codeText)
				i = end + 1
				continue
			}
		}
		
		// Add regular character
		cleanText += string(char)
		offset++
		i++
	}
	
	return entities
}

// findClosing finds the closing character for markdown syntax
func (h *HelpHandler) findClosing(text string, start int, char byte) int {
	for i := start; i < len(text); i++ {
		if text[i] == char {
			return i
		}
	}
	return -1
}

// stripMarkdownSyntax removes markdown syntax characters from text
func (h *HelpHandler) stripMarkdownSyntax(text string) string {
	result := ""
	i := 0
	
	for i < len(text) {
		char := text[i]
		
		switch char {
		case '*':
			// Skip bold markers
			if end := h.findClosing(text, i+1, '*'); end != -1 {
				result += text[i+1 : end]
				i = end + 1
				continue
			}
		case '`':
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