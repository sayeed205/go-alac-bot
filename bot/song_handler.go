package bot

import (
	"context"
	"fmt"
	"log"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go-alac-bot/downloader"

	"github.com/gotd/td/telegram/uploader"
	"github.com/gotd/td/tg"
)

// SongHandler implements CommandHandler for the /song command
type SongHandler struct {
	client       *TelegramBot
	logger       *log.Logger
	errorHandler *ErrorHandler
	downloader   downloader.SongDownloader
}

// NewSongHandler creates a new SongHandler instance
func NewSongHandler(client *TelegramBot, logger *log.Logger) *SongHandler {
	handler := &SongHandler{
		client:     client,
		logger:     logger,
		downloader: downloader.NewSongDownloaderImpl(),
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

// Handle processes the /song command and downloads a single song with progress tracking
func (h *SongHandler) Handle(ctx context.Context, cmdCtx *CommandContext) error {
	startTime := time.Now()

	h.logger.Printf("Processing /song command for user %d in chat %d", cmdCtx.UserID, cmdCtx.ChatID)

	// Check if URL is provided
	if strings.TrimSpace(cmdCtx.Args) == "" {
		return h.sendErrorMessage(ctx, cmdCtx.ChatID, "Please provide a song URL.")
	}

	// Parse and validate the URL
	songURL := strings.TrimSpace(cmdCtx.Args)
	urlMeta := ExtractURLMeta(songURL)
	if urlMeta == nil {
		return h.sendErrorMessage(ctx, cmdCtx.ChatID, "Please provide a valid Apple Music URL.")
	}

	// Create Telegram progress reporter
	if h.client == nil || h.client.GetClient() == nil {
		return h.sendErrorMessage(ctx, cmdCtx.ChatID, "Bot client is not initialized.")
	}

	reporter := downloader.NewTelegramProgressReporter(h.client.GetClient().API())

	// Start progress tracking
	if err := reporter.StartTracking(ctx, cmdCtx.ChatID, "Unknown Song"); err != nil {
		h.logger.Printf("Failed to start progress tracking: %v", err)
		return h.sendErrorMessage(ctx, cmdCtx.ChatID, "Failed to initialize progress tracking.")
	}
	defer reporter.Stop()

	// Create progress tracker with 2-second update interval
	tracker := downloader.NewProgressTracker(reporter)
	if err := tracker.Start(ctx); err != nil {
		h.logger.Printf("Failed to start progress tracker: %v", err)
		reporter.ReportError(fmt.Errorf("failed to start progress tracker: %w", err))
		return nil
	}
	defer tracker.Stop()

	// Set up progress callbacks
	callbacks := downloader.ProgressCallbacks{
		OnProgress: func(phase downloader.Phase, progress downloader.Progress) {
			tracker.UpdateProgress(phase, progress)
		},
		OnPhaseChange: func(oldPhase, newPhase downloader.Phase) {
			tracker.UpdateProgress(newPhase, downloader.Progress{})
		},
		OnError: func(err error) {
			h.logger.Printf("Download error: %v", err)
			reporter.ReportError(err)
		},
		OnComplete: func(result *downloader.DownloadResult) {
			h.logger.Printf("Download completed: %s", result.FilePath)
			reporter.ReportComplete(time.Since(startTime), result.FilePath)
		},
	}

	// Download the song with progress tracking
	result, err := h.downloader.Download(ctx, songURL, callbacks)
	if err != nil {
		h.logger.Printf("Failed to download song: %v", err)

		// Use error handler if available for network errors
		if h.errorHandler != nil && h.errorHandler.IsNetworkError(err) {
			return h.errorHandler.HandleNetworkError(err, true)
		}

		// Error is already reported through callbacks, so we just return
		return nil
	}

	// Upload the downloaded file to Telegram
	if err := h.uploadFile(ctx, cmdCtx.ChatID, result); err != nil {
		h.logger.Printf("Failed to upload file: %v", err)
		reporter.ReportError(fmt.Errorf("failed to upload file: %w", err))
		return nil
	}

	// Log successful processing with timing
	processingTime := time.Since(startTime)
	h.logger.Printf("Successfully processed /song command for user %d (took %v)",
		cmdCtx.UserID, processingTime)

	return nil
}

// uploadFile uploads the downloaded file to Telegram as an audio file
func (h *SongHandler) uploadFile(ctx context.Context, chatID int64, result *downloader.DownloadResult) error {
	// Get file information
	fileInfo, err := os.Stat(result.FilePath)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// Get file properties
	fileSize := fileInfo.Size()
	fileName := filepath.Base(result.FilePath)
	fileMime := mime.TypeByExtension(filepath.Ext(result.FilePath))
	if fileMime == "" {
		fileMime = "audio/mp4" // Default for M4A files
	}

	// Create caption with song ID from Apple Music
	songID := "unknown"
	if result.SongMeta != nil && result.SongMeta.AppleMusicID != "" {
		songID = result.SongMeta.AppleMusicID
	}
	caption := fmt.Sprintf("song `%s`", songID)

	// Check if SongMeta is nil
	if result.SongMeta == nil {
		return fmt.Errorf("song metadata is missing")
	}

	// Convert duration to seconds from song metadata
	durationSeconds := int(result.SongMeta.Duration.Seconds())

	// Create upload progress reporter
	uploadReporter := downloader.NewTelegramProgressReporter(h.client.GetClient().API())
	uploadDisplayName := fmt.Sprintf("📤 %s", fileName)
	if err := uploadReporter.StartTracking(ctx, chatID, uploadDisplayName); err != nil {
		h.logger.Printf("Failed to start upload progress tracking: %v", err)
		return fmt.Errorf("failed to start upload progress tracking: %w", err)
	}
	defer uploadReporter.Stop()

	// Report upload phase start
	uploadReporter.ReportPhaseChange(downloader.PhaseComplete, downloader.PhaseUploading)

	// Show initial upload progress
	uploadReporter.UpdateProgress(downloader.PhaseUploading, downloader.Progress{
		BytesProcessed: 0,
		TotalBytes:     fileSize,
		Percentage:     0,
	})

	// Upload file with real progress tracking using gotd/td uploader
	uploadStartTime := time.Now()
	uploadedFile, err := h.uploadFileWithRealProgress(ctx, result.FilePath, fileSize, uploadReporter)
	if err != nil {
		uploadReporter.ReportError(fmt.Errorf("upload failed: %w", err))
		return fmt.Errorf("failed to upload file: %w", err)
	}

	// Report upload completion
	uploadDuration := time.Since(uploadStartTime)
	uploadReporter.UpdateProgress(downloader.PhaseComplete, downloader.Progress{
		BytesProcessed: fileSize,
		TotalBytes:     fileSize,
		Speed:          int64(float64(fileSize) / uploadDuration.Seconds()),
		Percentage:     100,
	})
	uploadReporter.ReportComplete(uploadDuration, fileName)

	// Determine peer type for chat
	var peer tg.InputPeerClass
	if chatID > 0 {
		peer = &tg.InputPeerUser{UserID: chatID}
	} else {
		peer = &tg.InputPeerChat{ChatID: -chatID}
	}

	// Create audio media with proper attributes
	media := &tg.InputMediaUploadedDocument{
		File:     uploadedFile,
		MimeType: fileMime,
		Attributes: []tg.DocumentAttributeClass{
			&tg.DocumentAttributeAudio{
				Duration:  durationSeconds,
				Title:     getStringOrDefault(result.SongMeta.Title, "Unknown Title"),
				Performer: getStringOrDefault(result.SongMeta.Artist, "Unknown Artist"),
			},
			&tg.DocumentAttributeFilename{
				FileName: fileName,
			},
		},
	}

	// Send the audio using direct API call
	_, err = h.client.GetClient().API().MessagesSendMedia(ctx, &tg.MessagesSendMediaRequest{
		Peer:     peer,
		Media:    media,
		Message:  caption,
		RandomID: time.Now().UnixNano(),
	})

	if err != nil {
		return fmt.Errorf("failed to send audio: %w", err)
	}

	// Delete the file after successful upload
	if err := os.Remove(result.FilePath); err != nil {
		h.logger.Printf("Warning: Failed to delete file after upload: %v", err)
	} else {
		h.logger.Printf("File deleted after successful upload: %s", result.FilePath)
	}

	h.logger.Printf("Successfully uploaded audio file: %s - %s (%.2f seconds, %s)",
		result.SongMeta.Artist, result.SongMeta.Title, result.SongMeta.Duration.Seconds(), h.formatBytes(fileSize))

	return nil
}

// uploadFileWithRealProgress uploads a file with actual progress tracking using gotd/td
func (h *SongHandler) uploadFileWithRealProgress(ctx context.Context, filePath string, fileSize int64, reporter downloader.ProgressReporter) (tg.InputFileClass, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Create a progress reader that tracks actual bytes read
	progressReader := &UploadProgressReader{
		reader:     file,
		totalSize:  fileSize,
		reporter:   reporter,
		startTime:  time.Now(),
		lastUpdate: time.Now(),
	}

	// Use gotd/td uploader with our progress reader
	u := uploader.NewUploader(h.client.GetClient().API())
	fileName := filepath.Base(filePath)

	// Upload using FromReader which will call our Read method
	return u.FromReader(ctx, fileName, progressReader)
}

// UploadProgressReader wraps a file reader to track actual upload progress
type UploadProgressReader struct {
	reader     *os.File
	totalSize  int64
	bytesRead  int64
	reporter   downloader.ProgressReporter
	startTime  time.Time
	lastUpdate time.Time
	mu         sync.Mutex
}

// Read implements io.Reader and tracks real progress as data is read for upload
func (upr *UploadProgressReader) Read(p []byte) (n int, err error) {
	n, err = upr.reader.Read(p)
	if n > 0 {
		upr.mu.Lock()
		upr.bytesRead += int64(n)
		currentBytes := upr.bytesRead
		upr.mu.Unlock()

		// Update progress every 1 second or on completion
		now := time.Now()
		if now.Sub(upr.lastUpdate) >= time.Second || err != nil || currentBytes >= upr.totalSize {
			upr.updateProgress(currentBytes, now)
		}
	}
	return n, err
}

// updateProgress reports the current upload progress
func (upr *UploadProgressReader) updateProgress(bytesRead int64, now time.Time) {
	upr.mu.Lock()
	upr.lastUpdate = now
	upr.mu.Unlock()

	percentage := float64(bytesRead) / float64(upr.totalSize) * 100
	if percentage > 100 {
		percentage = 100
	}

	elapsed := now.Sub(upr.startTime)
	var speed int64
	if elapsed.Seconds() > 0 {
		speed = int64(float64(bytesRead) / elapsed.Seconds())
	}

	var eta time.Duration
	if speed > 0 && bytesRead < upr.totalSize {
		remaining := upr.totalSize - bytesRead
		eta = time.Duration(float64(remaining)/float64(speed)) * time.Second
	}

	// Report progress to Telegram
	if upr.reporter != nil {
		upr.reporter.UpdateProgress(downloader.PhaseUploading, downloader.Progress{
			BytesProcessed: bytesRead,
			TotalBytes:     upr.totalSize,
			Speed:          speed,
			ETA:            eta,
			Percentage:     percentage,
		})
	}
}

// getStringOrDefault returns the value if not empty, otherwise returns the default
func getStringOrDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

// formatBytes formats byte count into human-readable format
func (h *SongHandler) formatBytes(bytes int64) string {
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
