package downloader

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gotd/td/tg"
)

// MockTelegramAPI is a mock implementation of TelegramAPI for testing
type MockTelegramAPI struct {
	mu                    sync.RWMutex
	sendMessageCalls      []SendMessageCall
	editMessageCalls      []EditMessageCall
	shouldFailSend        bool
	shouldFailEdit        bool
	nextMessageID         int
	sendMessageError      error
	editMessageError      error
}

type SendMessageCall struct {
	Request *tg.MessagesSendMessageRequest
}

type EditMessageCall struct {
	Request *tg.MessagesEditMessageRequest
}

func NewMockTelegramAPI() *MockTelegramAPI {
	return &MockTelegramAPI{
		nextMessageID: 1,
	}
}

func (m *MockTelegramAPI) MessagesSendMessage(ctx context.Context, request *tg.MessagesSendMessageRequest) (tg.UpdatesClass, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.sendMessageCalls = append(m.sendMessageCalls, SendMessageCall{Request: request})
	
	if m.shouldFailSend {
		if m.sendMessageError != nil {
			return nil, m.sendMessageError
		}
		return nil, fmt.Errorf("mock send message error")
	}
	
	// Return a mock UpdateShortSentMessage with the next message ID
	messageID := m.nextMessageID
	m.nextMessageID++
	
	return &tg.UpdateShortSentMessage{
		ID:   messageID,
		Pts:  1,
		Date: int(time.Now().Unix()),
	}, nil
}

func (m *MockTelegramAPI) MessagesEditMessage(ctx context.Context, request *tg.MessagesEditMessageRequest) (tg.UpdatesClass, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.editMessageCalls = append(m.editMessageCalls, EditMessageCall{Request: request})
	
	if m.shouldFailEdit {
		if m.editMessageError != nil {
			return nil, m.editMessageError
		}
		return nil, fmt.Errorf("mock edit message error")
	}
	
	// Return a mock Updates response
	return &tg.Updates{
		Updates: []tg.UpdateClass{},
		Users:   []tg.UserClass{},
		Chats:   []tg.ChatClass{},
		Date:    int(time.Now().Unix()),
		Seq:     1,
	}, nil
}

func (m *MockTelegramAPI) GetSendMessageCalls() []SendMessageCall {
	m.mu.RLock()
	defer m.mu.RUnlock()
	calls := make([]SendMessageCall, len(m.sendMessageCalls))
	copy(calls, m.sendMessageCalls)
	return calls
}

func (m *MockTelegramAPI) GetEditMessageCalls() []EditMessageCall {
	m.mu.RLock()
	defer m.mu.RUnlock()
	calls := make([]EditMessageCall, len(m.editMessageCalls))
	copy(calls, m.editMessageCalls)
	return calls
}

func (m *MockTelegramAPI) SetShouldFailSend(fail bool, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFailSend = fail
	m.sendMessageError = err
}

func (m *MockTelegramAPI) SetShouldFailEdit(fail bool, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFailEdit = fail
	m.editMessageError = err
}

func (m *MockTelegramAPI) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sendMessageCalls = nil
	m.editMessageCalls = nil
	m.shouldFailSend = false
	m.shouldFailEdit = false
	m.nextMessageID = 1
	m.sendMessageError = nil
	m.editMessageError = nil
}

func TestTelegramProgressReporter_NewTelegramProgressReporter(t *testing.T) {
	api := NewMockTelegramAPI()
	reporter := NewTelegramProgressReporter(api)
	
	if reporter == nil {
		t.Fatal("NewTelegramProgressReporter returned nil")
	}
	
	if reporter.api != api {
		t.Error("API not set correctly")
	}
	
	if reporter.IsActive() {
		t.Error("Reporter should not be active initially")
	}
}

func TestTelegramProgressReporter_StartTracking(t *testing.T) {
	api := NewMockTelegramAPI()
	reporter := NewTelegramProgressReporter(api)
	ctx := context.Background()
	
	chatID := int64(12345)
	songName := "Test Song - Artist"
	
	// Test successful start
	err := reporter.StartTracking(ctx, chatID, songName)
	if err != nil {
		t.Fatalf("Failed to start tracking: %v", err)
	}
	
	if !reporter.IsActive() {
		t.Error("Reporter should be active after StartTracking")
	}
	
	if reporter.GetCurrentSong() != songName {
		t.Errorf("Expected song name %s, got %s", songName, reporter.GetCurrentSong())
	}
	
	// Verify initial message was sent
	sendCalls := api.GetSendMessageCalls()
	if len(sendCalls) != 1 {
		t.Errorf("Expected 1 send message call, got %d", len(sendCalls))
	}
	
	if sendCalls[0].Request.Message == "" {
		t.Error("Initial message should not be empty")
	}
	
	if !strings.Contains(sendCalls[0].Request.Message, songName) {
		t.Error("Initial message should contain song name")
	}
	
	// Test double start should fail
	err = reporter.StartTracking(ctx, chatID, songName)
	if err == nil {
		t.Error("Expected error when starting already active reporter")
	}
}

func TestTelegramProgressReporter_StartTrackingWithSendError(t *testing.T) {
	api := NewMockTelegramAPI()
	api.SetShouldFailSend(true, nil)
	reporter := NewTelegramProgressReporter(api)
	ctx := context.Background()
	
	err := reporter.StartTracking(ctx, 12345, "Test Song")
	if err == nil {
		t.Error("Expected error when send message fails")
	}
	
	if reporter.IsActive() {
		t.Error("Reporter should not be active when start tracking fails")
	}
}

func TestTelegramProgressReporter_UpdateProgress(t *testing.T) {
	api := NewMockTelegramAPI()
	reporter := NewTelegramProgressReporter(api)
	ctx := context.Background()
	
	// Start tracking first
	err := reporter.StartTracking(ctx, 12345, "Test Song")
	if err != nil {
		t.Fatalf("Failed to start tracking: %v", err)
	}
	
	// Update progress
	progress := Progress{
		BytesProcessed: 1024,
		TotalBytes:     2048,
		Speed:          512,
		ETA:            30 * time.Second,
		Percentage:     50.0,
	}
	
	err = reporter.UpdateProgress(PhaseDownloading, progress)
	if err != nil {
		t.Fatalf("Failed to update progress: %v", err)
	}
	
	// Verify edit message was called
	editCalls := api.GetEditMessageCalls()
	if len(editCalls) != 1 {
		t.Errorf("Expected 1 edit message call, got %d", len(editCalls))
	}
	
	message := editCalls[0].Request.Message
	if !strings.Contains(message, "50.0%") {
		t.Error("Progress message should contain percentage")
	}
	
	if !strings.Contains(message, "⬇️") {
		t.Error("Progress message should contain downloading emoji")
	}
}

func TestTelegramProgressReporter_UpdateProgressWhenNotActive(t *testing.T) {
	api := NewMockTelegramAPI()
	reporter := NewTelegramProgressReporter(api)
	
	// Try to update progress when not active
	err := reporter.UpdateProgress(PhaseDownloading, Progress{})
	if err != nil {
		t.Errorf("UpdateProgress should not return error when not active, got: %v", err)
	}
	
	// Should not have made any API calls
	editCalls := api.GetEditMessageCalls()
	if len(editCalls) != 0 {
		t.Errorf("Expected 0 edit message calls when not active, got %d", len(editCalls))
	}
}

func TestTelegramProgressReporter_ReportPhaseChange(t *testing.T) {
	api := NewMockTelegramAPI()
	reporter := NewTelegramProgressReporter(api)
	ctx := context.Background()
	
	// Start tracking first
	err := reporter.StartTracking(ctx, 12345, "Test Song")
	if err != nil {
		t.Fatalf("Failed to start tracking: %v", err)
	}
	
	// Report phase change
	err = reporter.ReportPhaseChange(PhaseValidating, PhaseDownloading)
	if err != nil {
		t.Fatalf("Failed to report phase change: %v", err)
	}
	
	// Verify edit message was called
	editCalls := api.GetEditMessageCalls()
	if len(editCalls) != 1 {
		t.Errorf("Expected 1 edit message call, got %d", len(editCalls))
	}
	
	message := editCalls[0].Request.Message
	if !strings.Contains(message, "⬇️") {
		t.Error("Phase change message should contain downloading emoji")
	}
	
	if !strings.Contains(message, "Downloading song") {
		t.Error("Phase change message should contain phase description")
	}
}

func TestTelegramProgressReporter_ReportError(t *testing.T) {
	api := NewMockTelegramAPI()
	reporter := NewTelegramProgressReporter(api)
	ctx := context.Background()
	
	// Start tracking first
	err := reporter.StartTracking(ctx, 12345, "Test Song")
	if err != nil {
		t.Fatalf("Failed to start tracking: %v", err)
	}
	
	// Report error
	testError := NewDownloadError(ErrorNetworkFailure, "Connection failed")
	err = reporter.ReportError(testError)
	if err != nil {
		t.Fatalf("Failed to report error: %v", err)
	}
	
	// Verify edit message was called
	editCalls := api.GetEditMessageCalls()
	if len(editCalls) != 1 {
		t.Errorf("Expected 1 edit message call, got %d", len(editCalls))
	}
	
	message := editCalls[0].Request.Message
	if !strings.Contains(message, "❌") {
		t.Error("Error message should contain error emoji")
	}
	
	if !strings.Contains(message, "Connection failed") {
		t.Error("Error message should contain error description")
	}
}

func TestTelegramProgressReporter_ReportComplete(t *testing.T) {
	api := NewMockTelegramAPI()
	reporter := NewTelegramProgressReporter(api)
	ctx := context.Background()
	
	// Start tracking first
	err := reporter.StartTracking(ctx, 12345, "Test Song")
	if err != nil {
		t.Fatalf("Failed to start tracking: %v", err)
	}
	
	// Report completion
	duration := 45 * time.Second
	filePath := "/path/to/song.m4a"
	
	err = reporter.ReportComplete(duration, filePath)
	if err != nil {
		t.Fatalf("Failed to report completion: %v", err)
	}
	
	// Verify edit message was called
	editCalls := api.GetEditMessageCalls()
	if len(editCalls) != 1 {
		t.Errorf("Expected 1 edit message call, got %d", len(editCalls))
	}
	
	message := editCalls[0].Request.Message
	if !strings.Contains(message, "✅") {
		t.Error("Completion message should contain success emoji")
	}
	
	if !strings.Contains(message, "45s") {
		t.Error("Completion message should contain duration")
	}
	
	if !strings.Contains(message, "Complete") {
		t.Error("Completion message should contain completion text")
	}
}

func TestTelegramProgressReporter_Stop(t *testing.T) {
	api := NewMockTelegramAPI()
	reporter := NewTelegramProgressReporter(api)
	ctx := context.Background()
	
	// Start tracking first
	err := reporter.StartTracking(ctx, 12345, "Test Song")
	if err != nil {
		t.Fatalf("Failed to start tracking: %v", err)
	}
	
	if !reporter.IsActive() {
		t.Error("Reporter should be active before Stop")
	}
	
	// Stop tracking
	reporter.Stop()
	
	if reporter.IsActive() {
		t.Error("Reporter should not be active after Stop")
	}
	
	if reporter.GetCurrentSong() != "" {
		t.Error("Current song should be empty after Stop")
	}
}

func TestTelegramProgressReporter_FormatBytes(t *testing.T) {
	api := NewMockTelegramAPI()
	reporter := NewTelegramProgressReporter(api)
	
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{1099511627776, "1.0 TB"},
	}
	
	for _, test := range tests {
		result := reporter.formatBytes(test.bytes)
		if result != test.expected {
			t.Errorf("formatBytes(%d) = %s, expected %s", test.bytes, result, test.expected)
		}
	}
}

func TestTelegramProgressReporter_CreateProgressBar(t *testing.T) {
	api := NewMockTelegramAPI()
	reporter := NewTelegramProgressReporter(api)
	
	tests := []struct {
		percentage float64
		length     int
		expected   string
	}{
		{0, 10, "░░░░░░░░░░"},
		{50, 10, "█████░░░░░"},
		{100, 10, "██████████"},
		{25, 4, "█░░░"},
		{75, 4, "███░"},
	}
	
	for _, test := range tests {
		result := reporter.createProgressBar(test.percentage, test.length)
		if result != test.expected {
			t.Errorf("createProgressBar(%.1f, %d) = %s, expected %s", 
				test.percentage, test.length, result, test.expected)
		}
	}
}

func TestTelegramProgressReporter_GetPhaseEmoji(t *testing.T) {
	api := NewMockTelegramAPI()
	reporter := NewTelegramProgressReporter(api)
	
	tests := []struct {
		phase    Phase
		expected string
	}{
		{PhaseValidating, "🔍"},
		{PhaseDownloading, "⬇️"},
		{PhaseDecrypting, "🔓"},
		{PhaseWriting, "💾"},
		{PhaseComplete, "✅"},
		{PhaseError, "❌"},
		{Phase(-1), "⏳"}, // Unknown phase
	}
	
	for _, test := range tests {
		result := reporter.getPhaseEmoji(test.phase)
		if result != test.expected {
			t.Errorf("getPhaseEmoji(%v) = %s, expected %s", test.phase, result, test.expected)
		}
	}
}

func TestTelegramProgressReporter_GetPhaseDescription(t *testing.T) {
	api := NewMockTelegramAPI()
	reporter := NewTelegramProgressReporter(api)
	
	tests := []struct {
		phase    Phase
		expected string
	}{
		{PhaseValidating, "Validating song URL..."},
		{PhaseDownloading, "Downloading song..."},
		{PhaseDecrypting, "Decrypting audio..."},
		{PhaseWriting, "Writing file..."},
		{PhaseComplete, "Download complete!"},
		{PhaseError, "Error occurred"},
		{Phase(-1), "Processing..."}, // Unknown phase
	}
	
	for _, test := range tests {
		result := reporter.getPhaseDescription(test.phase)
		if result != test.expected {
			t.Errorf("getPhaseDescription(%v) = %s, expected %s", test.phase, result, test.expected)
		}
	}
}

func TestTelegramProgressReporter_ExtractMessageID(t *testing.T) {
	api := NewMockTelegramAPI()
	reporter := NewTelegramProgressReporter(api)
	
	// Test UpdateShortSentMessage
	shortSent := &tg.UpdateShortSentMessage{ID: 123}
	messageID := reporter.extractMessageID(shortSent)
	if messageID != 123 {
		t.Errorf("Expected message ID 123, got %d", messageID)
	}
	
	// Test Updates with UpdateNewMessage
	updates := &tg.Updates{
		Updates: []tg.UpdateClass{
			&tg.UpdateNewMessage{
				Message: &tg.Message{ID: 456},
			},
		},
	}
	messageID = reporter.extractMessageID(updates)
	if messageID != 456 {
		t.Errorf("Expected message ID 456, got %d", messageID)
	}
	
	// Test empty updates
	emptyUpdates := &tg.Updates{Updates: []tg.UpdateClass{}}
	messageID = reporter.extractMessageID(emptyUpdates)
	if messageID != 0 {
		t.Errorf("Expected message ID 0 for empty updates, got %d", messageID)
	}
}

func TestTelegramProgressReporter_ConcurrentOperations(t *testing.T) {
	api := NewMockTelegramAPI()
	reporter := NewTelegramProgressReporter(api)
	ctx := context.Background()
	
	// Start tracking
	err := reporter.StartTracking(ctx, 12345, "Test Song")
	if err != nil {
		t.Fatalf("Failed to start tracking: %v", err)
	}
	
	// Run concurrent operations
	var wg sync.WaitGroup
	numGoroutines := 10
	
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			
			// Update progress
			progress := Progress{
				BytesProcessed: int64(id * 100),
				TotalBytes:     1000,
				Percentage:     float64(id * 10),
			}
			reporter.UpdateProgress(PhaseDownloading, progress)
			
			// Report phase change
			reporter.ReportPhaseChange(PhaseDownloading, PhaseDecrypting)
			
			// Check if active (should not panic)
			reporter.IsActive()
			reporter.GetCurrentSong()
			reporter.GetElapsedTime()
		}(i)
	}
	
	wg.Wait()
	
	// Should still be active and not panic
	if !reporter.IsActive() {
		t.Error("Reporter should still be active after concurrent operations")
	}
	
	reporter.Stop()
}