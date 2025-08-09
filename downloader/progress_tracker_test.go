package downloader

import (
	"context"
	"sync"
	"testing"
	"time"
)

// MockProgressReporter is a mock implementation of ProgressReporter for testing
type MockProgressReporter struct {
	mu                    sync.RWMutex
	startTrackingCalls    []StartTrackingCall
	updateProgressCalls   []UpdateProgressCall
	phaseChangeCalls      []PhaseChangeCall
	errorCalls            []ErrorCall
	completeCalls         []CompleteCall
	stopCalls             int
	shouldFailUpdate      bool
	shouldFailPhaseChange bool
}

type StartTrackingCall struct {
	ChatID   int64
	SongName string
}

type UpdateProgressCall struct {
	Phase    Phase
	Progress Progress
}

type PhaseChangeCall struct {
	OldPhase Phase
	NewPhase Phase
}

type ErrorCall struct {
	Error error
}

type CompleteCall struct {
	Duration time.Duration
	FilePath string
}

func NewMockProgressReporter() *MockProgressReporter {
	return &MockProgressReporter{}
}

func (m *MockProgressReporter) StartTracking(ctx context.Context, chatID int64, songName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.startTrackingCalls = append(m.startTrackingCalls, StartTrackingCall{
		ChatID:   chatID,
		SongName: songName,
	})
	return nil
}

func (m *MockProgressReporter) UpdateProgress(phase Phase, progress Progress) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updateProgressCalls = append(m.updateProgressCalls, UpdateProgressCall{
		Phase:    phase,
		Progress: progress,
	})
	if m.shouldFailUpdate {
		return NewDownloadError(ErrorUnknown, "mock update error")
	}
	return nil
}

func (m *MockProgressReporter) ReportPhaseChange(oldPhase, newPhase Phase) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.phaseChangeCalls = append(m.phaseChangeCalls, PhaseChangeCall{
		OldPhase: oldPhase,
		NewPhase: newPhase,
	})
	if m.shouldFailPhaseChange {
		return NewDownloadError(ErrorUnknown, "mock phase change error")
	}
	return nil
}

func (m *MockProgressReporter) ReportError(err error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorCalls = append(m.errorCalls, ErrorCall{Error: err})
	return nil
}

func (m *MockProgressReporter) ReportComplete(duration time.Duration, filePath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.completeCalls = append(m.completeCalls, CompleteCall{
		Duration: duration,
		FilePath: filePath,
	})
	return nil
}

func (m *MockProgressReporter) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopCalls++
}

func (m *MockProgressReporter) GetUpdateProgressCalls() []UpdateProgressCall {
	m.mu.RLock()
	defer m.mu.RUnlock()
	calls := make([]UpdateProgressCall, len(m.updateProgressCalls))
	copy(calls, m.updateProgressCalls)
	return calls
}

func (m *MockProgressReporter) GetPhaseChangeCalls() []PhaseChangeCall {
	m.mu.RLock()
	defer m.mu.RUnlock()
	calls := make([]PhaseChangeCall, len(m.phaseChangeCalls))
	copy(calls, m.phaseChangeCalls)
	return calls
}

func (m *MockProgressReporter) GetStopCalls() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stopCalls
}

func TestProgressTracker_NewProgressTracker(t *testing.T) {
	reporter := NewMockProgressReporter()
	tracker := NewProgressTracker(reporter)
	
	if tracker == nil {
		t.Fatal("NewProgressTracker returned nil")
	}
	
	if tracker.updateInterval != 2*time.Second {
		t.Errorf("Expected update interval to be 2s, got %v", tracker.updateInterval)
	}
	
	if tracker.reporter != reporter {
		t.Error("Reporter not set correctly")
	}
	
	if tracker.isRunning {
		t.Error("Tracker should not be running initially")
	}
}

func TestProgressTracker_NewProgressTrackerWithInterval(t *testing.T) {
	reporter := NewMockProgressReporter()
	customInterval := 500 * time.Millisecond
	tracker := NewProgressTrackerWithInterval(reporter, customInterval)
	
	if tracker.updateInterval != customInterval {
		t.Errorf("Expected update interval to be %v, got %v", customInterval, tracker.updateInterval)
	}
}

func TestProgressTracker_StartAndStop(t *testing.T) {
	reporter := NewMockProgressReporter()
	tracker := NewProgressTracker(reporter)
	ctx := context.Background()
	
	// Test start
	err := tracker.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start tracker: %v", err)
	}
	
	if !tracker.IsRunning() {
		t.Error("Tracker should be running after Start()")
	}
	
	// Test double start should fail
	err = tracker.Start(ctx)
	if err == nil {
		t.Error("Expected error when starting already running tracker")
	}
	
	// Test stop
	tracker.Stop()
	
	if tracker.IsRunning() {
		t.Error("Tracker should not be running after Stop()")
	}
	
	// Verify reporter.Stop() was called
	if reporter.GetStopCalls() != 1 {
		t.Errorf("Expected 1 Stop() call on reporter, got %d", reporter.GetStopCalls())
	}
}

func TestProgressTracker_UpdateProgress(t *testing.T) {
	reporter := NewMockProgressReporter()
	tracker := NewProgressTrackerWithInterval(reporter, 100*time.Millisecond) // Faster for testing
	ctx := context.Background()
	
	err := tracker.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start tracker: %v", err)
	}
	defer tracker.Stop()
	
	// Send progress update
	progress := Progress{
		BytesProcessed: 1024,
		TotalBytes:     2048,
		Speed:          512,
		ETA:            2 * time.Second,
		Percentage:     50.0,
	}
	
	tracker.UpdateProgress(PhaseDownloading, progress)
	
	// Wait for update to be processed
	time.Sleep(150 * time.Millisecond)
	
	// Check current progress
	phase, currentProgress := tracker.GetCurrentProgress()
	if phase != PhaseDownloading {
		t.Errorf("Expected phase %v, got %v", PhaseDownloading, phase)
	}
	
	if currentProgress.BytesProcessed != progress.BytesProcessed {
		t.Errorf("Expected bytes processed %d, got %d", progress.BytesProcessed, currentProgress.BytesProcessed)
	}
}

func TestProgressTracker_PhaseChangeReporting(t *testing.T) {
	reporter := NewMockProgressReporter()
	tracker := NewProgressTrackerWithInterval(reporter, 50*time.Millisecond)
	ctx := context.Background()
	
	err := tracker.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start tracker: %v", err)
	}
	defer tracker.Stop()
	
	// Send progress updates with different phases
	tracker.UpdateProgress(PhaseValidating, Progress{})
	time.Sleep(60 * time.Millisecond)
	
	tracker.UpdateProgress(PhaseDownloading, Progress{BytesProcessed: 100, TotalBytes: 1000})
	time.Sleep(60 * time.Millisecond)
	
	tracker.UpdateProgress(PhaseDecrypting, Progress{BytesProcessed: 500, TotalBytes: 1000})
	time.Sleep(60 * time.Millisecond)
	
	// Check phase change calls
	phaseChanges := reporter.GetPhaseChangeCalls()
	if len(phaseChanges) < 2 {
		t.Errorf("Expected at least 2 phase changes, got %d", len(phaseChanges))
	}
	
	// Verify phase transitions
	expectedTransitions := []struct {
		from Phase
		to   Phase
	}{
		{-1, PhaseValidating},           // Initial transition (from -1 to validating)
		{PhaseValidating, PhaseDownloading},
		{PhaseDownloading, PhaseDecrypting},
	}
	
	for i, expected := range expectedTransitions {
		if i < len(phaseChanges) {
			if phaseChanges[i].OldPhase != expected.from || phaseChanges[i].NewPhase != expected.to {
				t.Errorf("Phase change %d: expected %v->%v, got %v->%v", 
					i, expected.from, expected.to, phaseChanges[i].OldPhase, phaseChanges[i].NewPhase)
			}
		}
	}
}

func TestProgressTracker_PeriodicUpdates(t *testing.T) {
	reporter := NewMockProgressReporter()
	tracker := NewProgressTrackerWithInterval(reporter, 100*time.Millisecond)
	ctx := context.Background()
	
	err := tracker.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start tracker: %v", err)
	}
	defer tracker.Stop()
	
	// Set initial progress
	progress := Progress{
		BytesProcessed: 1024,
		TotalBytes:     2048,
		Speed:          512,
		ETA:            2 * time.Second,
		Percentage:     50.0,
	}
	tracker.UpdateProgress(PhaseDownloading, progress)
	
	// Wait for multiple update intervals
	time.Sleep(350 * time.Millisecond)
	
	// Check that periodic updates occurred
	updateCalls := reporter.GetUpdateProgressCalls()
	if len(updateCalls) < 2 {
		t.Errorf("Expected at least 2 periodic updates, got %d", len(updateCalls))
	}
	
	// Verify the updates contain correct data
	for _, call := range updateCalls {
		if call.Phase != PhaseDownloading {
			t.Errorf("Expected phase %v in update, got %v", PhaseDownloading, call.Phase)
		}
	}
}

func TestProgressTracker_UpdateProgressWhenNotRunning(t *testing.T) {
	reporter := NewMockProgressReporter()
	tracker := NewProgressTracker(reporter)
	
	// Try to update progress when not running
	tracker.UpdateProgress(PhaseDownloading, Progress{BytesProcessed: 100})
	
	// Should not panic and should not affect state
	phase, progress := tracker.GetCurrentProgress()
	if phase != -1 || progress.BytesProcessed != 0 {
		t.Error("Progress should not be updated when tracker is not running")
	}
}

func TestProgressTracker_ContextCancellation(t *testing.T) {
	reporter := NewMockProgressReporter()
	tracker := NewProgressTrackerWithInterval(reporter, 50*time.Millisecond)
	
	ctx, cancel := context.WithCancel(context.Background())
	
	err := tracker.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start tracker: %v", err)
	}
	
	// Cancel context
	cancel()
	
	// Wait a bit for the goroutine to stop
	time.Sleep(100 * time.Millisecond)
	
	// Tracker should still report as running until Stop() is called
	if !tracker.IsRunning() {
		t.Error("Tracker should still report as running until Stop() is called")
	}
	
	tracker.Stop()
	
	if tracker.IsRunning() {
		t.Error("Tracker should not be running after Stop()")
	}
}

func TestProgressTracker_ResourceCleanup(t *testing.T) {
	reporter := NewMockProgressReporter()
	tracker := NewProgressTracker(reporter)
	ctx := context.Background()
	
	// Start and stop multiple times to test resource cleanup
	for i := 0; i < 3; i++ {
		err := tracker.Start(ctx)
		if err != nil {
			t.Fatalf("Failed to start tracker on iteration %d: %v", i, err)
		}
		
		// Send some updates
		tracker.UpdateProgress(PhaseDownloading, Progress{BytesProcessed: int64(i * 100)})
		time.Sleep(10 * time.Millisecond)
		
		tracker.Stop()
		
		if tracker.IsRunning() {
			t.Errorf("Tracker should not be running after Stop() on iteration %d", i)
		}
	}
	
	// Verify reporter.Stop() was called for each cycle
	expectedStopCalls := 3
	if reporter.GetStopCalls() != expectedStopCalls {
		t.Errorf("Expected %d Stop() calls on reporter, got %d", expectedStopCalls, reporter.GetStopCalls())
	}
}

func TestProgressTracker_ConcurrentUpdates(t *testing.T) {
	reporter := NewMockProgressReporter()
	tracker := NewProgressTrackerWithInterval(reporter, 50*time.Millisecond)
	ctx := context.Background()
	
	err := tracker.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start tracker: %v", err)
	}
	defer tracker.Stop()
	
	// Send concurrent updates
	var wg sync.WaitGroup
	numGoroutines := 10
	updatesPerGoroutine := 5
	
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < updatesPerGoroutine; j++ {
				progress := Progress{
					BytesProcessed: int64(goroutineID*100 + j),
					TotalBytes:     1000,
					Percentage:     float64(goroutineID*10 + j),
				}
				tracker.UpdateProgress(PhaseDownloading, progress)
				time.Sleep(5 * time.Millisecond)
			}
		}(i)
	}
	
	wg.Wait()
	time.Sleep(100 * time.Millisecond) // Allow time for final updates
	
	// Should not panic and should have processed some updates
	phase, progress := tracker.GetCurrentProgress()
	if phase != PhaseDownloading {
		t.Errorf("Expected final phase to be %v, got %v", PhaseDownloading, phase)
	}
	
	if progress.TotalBytes != 1000 {
		t.Errorf("Expected total bytes to be 1000, got %d", progress.TotalBytes)
	}
}