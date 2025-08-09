package downloader

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gotd/td/tg"
)

// TelegramAPI defines the interface for Telegram API operations needed by the progress reporter
type TelegramAPI interface {
	MessagesSendMessage(ctx context.Context, request *tg.MessagesSendMessageRequest) (tg.UpdatesClass, error)
	MessagesEditMessage(ctx context.Context, request *tg.MessagesEditMessageRequest) (tg.UpdatesClass, error)
}

// TelegramProgressReporter implements ProgressReporter for Telegram message updates
type TelegramProgressReporter struct {
	api       TelegramAPI
	mu        sync.RWMutex
	chatID    int64
	messageID int
	songName  string
	isActive  bool
	startTime time.Time
}

// NewTelegramProgressReporter creates a new TelegramProgressReporter
func NewTelegramProgressReporter(api TelegramAPI) *TelegramProgressReporter {
	return &TelegramProgressReporter{
		api: api,
	}
}

// StartTracking begins progress tracking for a specific chat and song
func (tpr *TelegramProgressReporter) StartTracking(ctx context.Context, chatID int64, songName string) error {
	tpr.mu.Lock()
	defer tpr.mu.Unlock()

	if tpr.isActive {
		return NewDownloadError(ErrorUnknown, "progress tracking is already active")
	}

	tpr.chatID = chatID
	tpr.songName = songName
	tpr.isActive = true
	tpr.startTime = time.Now()
	tpr.messageID = 0 // Will be set when first message is sent

	// Send initial message - check if it's an upload (has 📤 emoji)
	var initialMessage string
	if strings.Contains(songName, "📤") {
		initialMessage = fmt.Sprintf("🎵 **%s**\n\n⏳ Initializing upload...", songName)
	} else {
		initialMessage = fmt.Sprintf("🎵 **%s**\n\n⏳ Initializing download...", songName)
	}
	messageID, err := tpr.sendMessage(ctx, initialMessage)
	if err != nil {
		tpr.isActive = false
		return NewDownloadErrorWithCause(ErrorNetworkFailure, "failed to send initial progress message", err)
	}

	tpr.messageID = messageID
	return nil
}

// UpdateProgress reports progress for the current phase
func (tpr *TelegramProgressReporter) UpdateProgress(phase Phase, progress Progress) error {
	tpr.mu.RLock()
	if !tpr.isActive || tpr.messageID == 0 {
		tpr.mu.RUnlock()
		return nil // Not active or no message to update
	}

	chatID := tpr.chatID
	messageID := tpr.messageID
	songName := tpr.songName
	startTime := tpr.startTime
	tpr.mu.RUnlock()

	// Format progress message
	message := tpr.formatProgressMessage(songName, phase, progress, startTime)

	// Update the message
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return tpr.editMessage(ctx, chatID, messageID, message)
}

// ReportPhaseChange reports a transition between phases
func (tpr *TelegramProgressReporter) ReportPhaseChange(oldPhase, newPhase Phase) error {
	tpr.mu.RLock()
	if !tpr.isActive || tpr.messageID == 0 {
		tpr.mu.RUnlock()
		return nil
	}

	chatID := tpr.chatID
	messageID := tpr.messageID
	songName := tpr.songName
	startTime := tpr.startTime
	tpr.mu.RUnlock()

	// Create phase transition message
	message := fmt.Sprintf("🎵 **%s**\n\n%s %s\n\n⏱️ Elapsed: %s",
		songName,
		tpr.getPhaseEmoji(newPhase),
		tpr.getPhaseDescription(newPhase),
		time.Since(startTime).Round(time.Second))

	// Update the message
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return tpr.editMessage(ctx, chatID, messageID, message)
}

// ReportError reports an error that occurred during processing
func (tpr *TelegramProgressReporter) ReportError(err error) error {
	tpr.mu.RLock()
	if !tpr.isActive {
		tpr.mu.RUnlock()
		return nil
	}

	chatID := tpr.chatID
	messageID := tpr.messageID
	songName := tpr.songName
	startTime := tpr.startTime
	tpr.mu.RUnlock()

	// Format error message
	errorMsg := "An error occurred"
	if downloadErr, ok := err.(*DownloadError); ok {
		errorMsg = downloadErr.Message
	} else if err != nil {
		errorMsg = err.Error()
	}

	message := fmt.Sprintf("🎵 **%s**\n\n❌ **Error**: %s\n\n⏱️ Elapsed: %s",
		songName,
		errorMsg,
		time.Since(startTime).Round(time.Second))

	// Update the message
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return tpr.editMessage(ctx, chatID, messageID, message)
}

// ReportComplete reports successful completion with summary information
func (tpr *TelegramProgressReporter) ReportComplete(duration time.Duration, filePath string) error {
	tpr.mu.RLock()
	if !tpr.isActive {
		tpr.mu.RUnlock()
		return nil
	}

	chatID := tpr.chatID
	messageID := tpr.messageID
	songName := tpr.songName
	tpr.mu.RUnlock()

	// Format completion message - check if it's an upload
	var message string
	if strings.Contains(songName, "📤") {
		message = fmt.Sprintf("🎵 **%s**\n\n✅ **Upload Complete!**\n\n⏱️ Total time: %s",
			songName,
			duration.Round(time.Second))
	} else {
		message = fmt.Sprintf("🎵 **%s**\n\n✅ **Download Complete!**\n\n⏱️ Total time: %s\n📁 Ready for upload...",
			songName,
			duration.Round(time.Second))
	}

	// Update the message
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return tpr.editMessage(ctx, chatID, messageID, message)
}

// Stop stops progress tracking and cleans up resources
func (tpr *TelegramProgressReporter) Stop() {
	tpr.mu.Lock()
	defer tpr.mu.Unlock()

	tpr.isActive = false
	tpr.messageID = 0
	tpr.chatID = 0
	tpr.songName = ""
}

// sendMessage sends a new message and returns the message ID
func (tpr *TelegramProgressReporter) sendMessage(ctx context.Context, message string) (int, error) {
	if tpr.api == nil {
		return 0, NewDownloadError(ErrorUnknown, "telegram API is not initialized")
	}

	// Determine peer type based on chat ID
	var peer tg.InputPeerClass
	if tpr.chatID > 0 {
		peer = &tg.InputPeerUser{UserID: tpr.chatID}
	} else {
		peer = &tg.InputPeerChat{ChatID: -tpr.chatID}
	}

	request := &tg.MessagesSendMessageRequest{
		Peer:     peer,
		Message:  message,
		RandomID: time.Now().UnixNano(),
	}

	updates, err := tpr.api.MessagesSendMessage(ctx, request)
	if err != nil {
		return 0, err
	}

	// Extract message ID from updates
	messageID := tpr.extractMessageID(updates)
	return messageID, nil
}

// editMessage edits an existing message
func (tpr *TelegramProgressReporter) editMessage(ctx context.Context, chatID int64, messageID int, message string) error {
	if tpr.api == nil {
		return NewDownloadError(ErrorUnknown, "telegram API is not initialized")
	}

	// Determine peer type based on chat ID
	var peer tg.InputPeerClass
	if chatID > 0 {
		peer = &tg.InputPeerUser{UserID: chatID}
	} else {
		peer = &tg.InputPeerChat{ChatID: -chatID}
	}

	request := &tg.MessagesEditMessageRequest{
		Peer:    peer,
		ID:      messageID,
		Message: message,
	}

	_, err := tpr.api.MessagesEditMessage(ctx, request)
	return err
}

// extractMessageID extracts the message ID from Telegram API updates
func (tpr *TelegramProgressReporter) extractMessageID(updates tg.UpdatesClass) int {
	switch u := updates.(type) {
	case *tg.Updates:
		for _, update := range u.Updates {
			if msgUpdate, ok := update.(*tg.UpdateNewMessage); ok {
				if msg, ok := msgUpdate.Message.(*tg.Message); ok {
					return msg.ID
				}
			}
		}
	case *tg.UpdateShortSentMessage:
		return u.ID
	}
	return 0
}

// formatProgressMessage formats a progress update message
func (tpr *TelegramProgressReporter) formatProgressMessage(songName string, phase Phase, progress Progress, startTime time.Time) string {
	var builder strings.Builder

	// Song title
	builder.WriteString(fmt.Sprintf("🎵 **%s**\n\n", songName))

	// Phase indicator
	builder.WriteString(fmt.Sprintf("%s %s\n\n", tpr.getPhaseEmoji(phase), tpr.getPhaseDescription(phase)))

	// Progress bar and percentage
	if progress.TotalBytes > 0 {
		progressBar := tpr.createProgressBar(progress.Percentage, 20)
		builder.WriteString(fmt.Sprintf("📊 %s %.1f%%\n", progressBar, progress.Percentage))

		// File size info
		builder.WriteString(fmt.Sprintf("📦 %s / %s\n",
			tpr.formatBytes(progress.BytesProcessed),
			tpr.formatBytes(progress.TotalBytes)))

		// Speed and ETA
		if progress.Speed > 0 {
			builder.WriteString(fmt.Sprintf("⚡ %s/s", tpr.formatBytes(progress.Speed)))
			if progress.ETA > 0 {
				builder.WriteString(fmt.Sprintf(" • ETA: %s", progress.ETA.Round(time.Second)))
			}
			builder.WriteString("\n")
		}
	}

	// Elapsed time
	builder.WriteString(fmt.Sprintf("\n⏱️ Elapsed: %s", time.Since(startTime).Round(time.Second)))

	return builder.String()
}

// createProgressBar creates a visual progress bar
func (tpr *TelegramProgressReporter) createProgressBar(percentage float64, length int) string {
	if percentage < 0 {
		percentage = 0
	}
	if percentage > 100 {
		percentage = 100
	}

	filled := int((percentage / 100.0) * float64(length))
	empty := length - filled

	return strings.Repeat("█", filled) + strings.Repeat("░", empty)
}

// formatBytes formats byte count into human-readable format
func (tpr *TelegramProgressReporter) formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"KB", "MB", "GB", "TB"}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}

// getPhaseEmoji returns an emoji for the given phase
func (tpr *TelegramProgressReporter) getPhaseEmoji(phase Phase) string {
	switch phase {
	case PhaseValidating:
		return "🔍"
	case PhaseDownloading:
		return "⬇️"
	case PhaseDecrypting:
		return "🔓"
	case PhaseWriting:
		return "💾"
	case PhaseUploading:
		return "📤"
	case PhaseComplete:
		return "✅"
	case PhaseError:
		return "❌"
	default:
		return "⏳"
	}
}

// getPhaseDescription returns a description for the given phase
func (tpr *TelegramProgressReporter) getPhaseDescription(phase Phase) string {
	switch phase {
	case PhaseValidating:
		return "Validating song URL..."
	case PhaseDownloading:
		return "Downloading song..."
	case PhaseDecrypting:
		return "Decrypting audio..."
	case PhaseWriting:
		return "Writing file..."
	case PhaseUploading:
		return "Uploading to Telegram..."
	case PhaseComplete:
		return "Upload complete!"
	case PhaseError:
		return "Error occurred"
	default:
		return "Processing..."
	}
}

// IsActive returns whether the reporter is currently tracking progress
func (tpr *TelegramProgressReporter) IsActive() bool {
	tpr.mu.RLock()
	defer tpr.mu.RUnlock()
	return tpr.isActive
}

// GetCurrentSong returns the name of the currently tracked song
func (tpr *TelegramProgressReporter) GetCurrentSong() string {
	tpr.mu.RLock()
	defer tpr.mu.RUnlock()
	return tpr.songName
}

// GetElapsedTime returns the elapsed time since tracking started
func (tpr *TelegramProgressReporter) GetElapsedTime() time.Duration {
	tpr.mu.RLock()
	defer tpr.mu.RUnlock()
	if !tpr.isActive {
		return 0
	}
	return time.Since(tpr.startTime)
}
