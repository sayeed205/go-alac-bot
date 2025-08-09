package bot

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

const (
	MaxQueueSize = 7
)

// QueueRequest represents a single song download request in the queue
type QueueRequest struct {
	UniqueID    string
	SenderID    int64
	ChatID      int64
	MessageID   int
	URL         string
	RequestTime time.Time
	Status      QueueStatus
}

// QueueStatus represents the current status of a queue request
type QueueStatus int

const (
	StatusQueued QueueStatus = iota
	StatusProcessing
	StatusCompleted
	StatusFailed
)

// String returns string representation of queue status
func (s QueueStatus) String() string {
	switch s {
	case StatusQueued:
		return "queued"
	case StatusProcessing:
		return "processing"
	case StatusCompleted:
		return "completed"
	case StatusFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// SongQueue manages the queue of song download requests
type SongQueue struct {
	queue           []*QueueRequest
	processing      *QueueRequest
	mu              sync.RWMutex
	logger          *log.Logger
	songHandler     *SongHandler
	processingMutex sync.Mutex
	isProcessing    bool
}

// NewSongQueue creates a new song queue manager
func NewSongQueue(logger *log.Logger, songHandler *SongHandler) *SongQueue {
	return &SongQueue{
		queue:       make([]*QueueRequest, 0),
		logger:      logger,
		songHandler: songHandler,
	}
}

// GenerateUniqueID creates a unique ID for a request
func GenerateUniqueID(senderID, chatID int64, messageID int) string {
	return fmt.Sprintf("%d:%d:%d", senderID, chatID, messageID)
}

// AddRequest adds a new request to the queue
func (sq *SongQueue) AddRequest(senderID, chatID int64, messageID int, url string) (*QueueRequest, error) {
	sq.mu.Lock()
	defer sq.mu.Unlock()

	// Check if queue is full
	if len(sq.queue) >= MaxQueueSize {
		return nil, fmt.Errorf("queue is full (max %d requests)", MaxQueueSize)
	}

	// Generate unique ID
	uniqueID := GenerateUniqueID(senderID, chatID, messageID)

	// Check if request already exists
	if sq.findRequestByID(uniqueID) != nil {
		return nil, fmt.Errorf("request with ID %s already exists", uniqueID)
	}

	// Create new request
	request := &QueueRequest{
		UniqueID:    uniqueID,
		SenderID:    senderID,
		ChatID:      chatID,
		MessageID:   messageID,
		URL:         url,
		RequestTime: time.Now(),
		Status:      StatusQueued,
	}

	// Add to queue
	sq.queue = append(sq.queue, request)
	sq.logger.Printf("Added request %s to queue (position: %d)", uniqueID, len(sq.queue))

	// Start processing if not already processing
	go sq.processQueue()

	return request, nil
}

// GetQueuePosition returns the position of a request in the queue (1-based)
func (sq *SongQueue) GetQueuePosition(uniqueID string) int {
	sq.mu.RLock()
	defer sq.mu.RUnlock()

	for i, request := range sq.queue {
		if request.UniqueID == uniqueID {
			return i + 1 // 1-based position
		}
	}
	return -1 // Not found
}

// GetQueueSize returns the current size of the queue
func (sq *SongQueue) GetQueueSize() int {
	sq.mu.RLock()
	defer sq.mu.RUnlock()
	return len(sq.queue)
}

// IsProcessing returns whether a request is currently being processed
func (sq *SongQueue) IsProcessing() bool {
	sq.mu.RLock()
	defer sq.mu.RUnlock()
	return sq.processing != nil
}

// GetCurrentlyProcessing returns the currently processing request
func (sq *SongQueue) GetCurrentlyProcessing() *QueueRequest {
	sq.mu.RLock()
	defer sq.mu.RUnlock()
	return sq.processing
}

// findRequestByID finds a request by its unique ID (must be called with lock held)
func (sq *SongQueue) findRequestByID(uniqueID string) *QueueRequest {
	for _, request := range sq.queue {
		if request.UniqueID == uniqueID {
			return request
		}
	}
	return nil
}

// removeRequest removes a request from the queue (must be called with lock held)
func (sq *SongQueue) removeRequest(uniqueID string) bool {
	for i, request := range sq.queue {
		if request.UniqueID == uniqueID {
			// Remove from slice
			sq.queue = append(sq.queue[:i], sq.queue[i+1:]...)
			sq.logger.Printf("Removed request %s from queue", uniqueID)
			return true
		}
	}
	return false
}

// processQueue processes requests from the queue one at a time
func (sq *SongQueue) processQueue() {
	// Prevent multiple processing goroutines
	sq.processingMutex.Lock()
	if sq.isProcessing {
		sq.processingMutex.Unlock()
		return
	}
	sq.isProcessing = true
	sq.processingMutex.Unlock()

	defer func() {
		sq.processingMutex.Lock()
		sq.isProcessing = false
		sq.processingMutex.Unlock()
	}()

	for {
		// Get next request from queue
		sq.mu.Lock()
		if len(sq.queue) == 0 {
			sq.mu.Unlock()
			break
		}

		// Take the first request
		request := sq.queue[0]
		sq.queue = sq.queue[1:]
		sq.processing = request
		sq.mu.Unlock()

		// Update request status
		request.Status = StatusProcessing
		sq.logger.Printf("Processing request %s: %s", request.UniqueID, request.URL)

		// Create command context for the request
		cmdCtx := &CommandContext{
			Command:   "song",
			Args:      request.URL,
			UserID:    request.SenderID,
			ChatID:    request.ChatID,
			MessageID: request.MessageID,
			Timestamp: request.RequestTime,
		}

		// Process the request
		ctx := context.Background()
		err := sq.songHandler.ProcessDownload(ctx, cmdCtx)

		// Update request status based on result
		sq.mu.Lock()
		if err != nil {
			request.Status = StatusFailed
			sq.logger.Printf("Request %s failed: %v", request.UniqueID, err)
		} else {
			request.Status = StatusCompleted
			sq.logger.Printf("Request %s completed successfully", request.UniqueID)
		}
		sq.processing = nil
		sq.mu.Unlock()

		// Small delay between requests to avoid overwhelming
		time.Sleep(1 * time.Second)
	}

	sq.logger.Printf("Queue processing completed")
}

// GetQueueStatus returns the current queue status for display
func (sq *SongQueue) GetQueueStatus() string {
	sq.mu.RLock()
	defer sq.mu.RUnlock()

	status := fmt.Sprintf("📊 Queue Status:\n")
	status += fmt.Sprintf("• Queue size: %d/%d\n", len(sq.queue), MaxQueueSize)

	if sq.processing != nil {
		status += fmt.Sprintf("• Currently processing: %s\n", sq.processing.UniqueID)
	} else {
		status += "• Currently processing: None\n"
	}

	if len(sq.queue) > 0 {
		status += "\n📋 Queued requests:\n"
		for i, request := range sq.queue {
			status += fmt.Sprintf("%d. %s (from user %d)\n", i+1, request.UniqueID, request.SenderID)
		}
	}

	return status
}

// GetQueueInfo returns detailed information about all queued requests
func (sq *SongQueue) GetQueueInfo() []*QueueRequest {
	sq.mu.RLock()
	defer sq.mu.RUnlock()

	// Return a copy of the queue slice to avoid race conditions
	queueCopy := make([]*QueueRequest, len(sq.queue))
	copy(queueCopy, sq.queue)
	return queueCopy
}

// ClearQueue clears all requests from the queue (for admin use)
func (sq *SongQueue) ClearQueue() int {
	sq.mu.Lock()
	defer sq.mu.Unlock()

	cleared := len(sq.queue)
	sq.queue = make([]*QueueRequest, 0)
	sq.logger.Printf("Cleared %d requests from queue", cleared)
	return cleared
}
