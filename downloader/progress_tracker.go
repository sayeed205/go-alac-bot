package downloader

import (
	"context"
	"sync"
	"time"
)

// ProgressTracker manages periodic progress updates with a 2-second interval
type ProgressTracker struct {
	// Configuration
	updateInterval time.Duration
	reporter       ProgressReporter
	
	// State management
	mu           sync.RWMutex
	isRunning    bool
	currentPhase Phase
	currentProgress Progress
	
	// Goroutine management
	ctx        context.Context
	cancel     context.CancelFunc
	ticker     *time.Ticker
	updateChan chan progressUpdate
	stopChan   chan struct{}
	doneChan   chan struct{}
}

// progressUpdate represents an internal progress update
type progressUpdate struct {
	phase    Phase
	progress Progress
}

// NewProgressTracker creates a new ProgressTracker with the specified reporter
func NewProgressTracker(reporter ProgressReporter) *ProgressTracker {
	return &ProgressTracker{
		updateInterval: 2 * time.Second,
		reporter:       reporter,
		currentPhase:   -1, // Initialize to invalid phase to detect first phase change
	}
}

// NewProgressTrackerWithInterval creates a ProgressTracker with a custom update interval
func NewProgressTrackerWithInterval(reporter ProgressReporter, interval time.Duration) *ProgressTracker {
	pt := NewProgressTracker(reporter)
	pt.updateInterval = interval
	return pt
}

// Start begins the progress tracking with periodic updates
func (pt *ProgressTracker) Start(ctx context.Context) error {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	
	if pt.isRunning {
		return NewDownloadError(ErrorUnknown, "progress tracker is already running")
	}
	
	// Create new channels for this session
	pt.updateChan = make(chan progressUpdate, 10)
	pt.stopChan = make(chan struct{})
	pt.doneChan = make(chan struct{})
	
	// Create cancellable context
	pt.ctx, pt.cancel = context.WithCancel(ctx)
	pt.ticker = time.NewTicker(pt.updateInterval)
	pt.isRunning = true
	
	// Start the update loop in a separate goroutine
	go pt.updateLoop()
	
	return nil
}

// Stop stops the progress tracking and cleans up resources
func (pt *ProgressTracker) Stop() {
	pt.mu.Lock()
	if !pt.isRunning {
		pt.mu.Unlock()
		return
	}
	
	// Signal stop and cancel context
	select {
	case <-pt.stopChan:
		// Already closed
	default:
		close(pt.stopChan)
	}
	
	if pt.cancel != nil {
		pt.cancel()
	}
	pt.isRunning = false
	pt.mu.Unlock()
	
	// Wait for the update loop to finish
	<-pt.doneChan
	
	// Clean up resources
	if pt.ticker != nil {
		pt.ticker.Stop()
		pt.ticker = nil
	}
	
	// Stop the reporter
	if pt.reporter != nil {
		pt.reporter.Stop()
	}
}

// UpdateProgress updates the current progress information
func (pt *ProgressTracker) UpdateProgress(phase Phase, progress Progress) {
	pt.mu.RLock()
	if !pt.isRunning {
		pt.mu.RUnlock()
		return
	}
	pt.mu.RUnlock()
	
	// Send update through channel (non-blocking)
	select {
	case pt.updateChan <- progressUpdate{phase: phase, progress: progress}:
	default:
		// Channel is full, skip this update to prevent blocking
	}
}

// GetCurrentProgress returns the current progress state (thread-safe)
func (pt *ProgressTracker) GetCurrentProgress() (Phase, Progress) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	return pt.currentPhase, pt.currentProgress
}

// IsRunning returns whether the tracker is currently running
func (pt *ProgressTracker) IsRunning() bool {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	return pt.isRunning
}

// updateLoop runs the main update loop in a separate goroutine
func (pt *ProgressTracker) updateLoop() {
	defer close(pt.doneChan)
	
	var lastReportedPhase Phase = -1 // Initialize to invalid phase
	
	for {
		select {
		case <-pt.ctx.Done():
			return
			
		case <-pt.stopChan:
			return
			
		case update := <-pt.updateChan:
			// Update current state
			pt.mu.Lock()
			oldPhase := pt.currentPhase
			pt.currentPhase = update.phase
			pt.currentProgress = update.progress
			pt.mu.Unlock()
			
			// Report phase change if needed (including initial phase set)
			if oldPhase != update.phase && pt.reporter != nil {
				if err := pt.reporter.ReportPhaseChange(oldPhase, update.phase); err != nil {
					// Log error but continue (could add logging here)
				}
			}
			
		case <-pt.ticker.C:
			// Periodic update every 2 seconds
			pt.mu.RLock()
			currentPhase := pt.currentPhase
			currentProgress := pt.currentProgress
			pt.mu.RUnlock()
			
			// Only report if we have valid progress and phase has been set (not -1)
			if pt.reporter != nil && currentPhase >= 0 && 
			   (currentPhase != lastReportedPhase || 
			    (currentProgress.TotalBytes > 0 && currentProgress.BytesProcessed >= 0)) {
				if err := pt.reporter.UpdateProgress(currentPhase, currentProgress); err != nil {
					// Log error but continue (could add logging here)
				}
				lastReportedPhase = currentPhase
			}
		}
	}
}