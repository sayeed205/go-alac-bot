package downloader

import (
	"context"
	"time"
)

// Phase represents the current phase of the download process
type Phase int

const (
	PhaseValidating Phase = iota
	PhaseDownloading
	PhaseDecrypting
	PhaseWriting
	PhaseUploading
	PhaseComplete
	PhaseError
)

// String returns the string representation of the phase
func (p Phase) String() string {
	switch p {
	case PhaseValidating:
		return "validating"
	case PhaseDownloading:
		return "downloading"
	case PhaseDecrypting:
		return "decrypting"
	case PhaseWriting:
		return "writing"
	case PhaseUploading:
		return "uploading"
	case PhaseComplete:
		return "complete"
	case PhaseError:
		return "error"
	default:
		return "unknown"
	}
}

// Progress represents the current progress of an operation
type Progress struct {
	BytesProcessed int64         `json:"bytes_processed"`
	TotalBytes     int64         `json:"total_bytes"`
	Speed          int64         `json:"speed"` // bytes per second
	ETA            time.Duration `json:"eta"`
	Percentage     float64       `json:"percentage"`
}

// ProgressCallbacks defines callback functions for progress reporting
type ProgressCallbacks struct {
	OnProgress    func(phase Phase, progress Progress)
	OnPhaseChange func(oldPhase, newPhase Phase)
	OnError       func(err error)
	OnComplete    func(result *DownloadResult)
}

// DownloadResult contains the result of a successful download
type DownloadResult struct {
	FilePath string        `json:"file_path"`
	SongMeta *SongMetadata `json:"song_meta"`
	Duration time.Duration `json:"duration"`
	FileSize int64         `json:"file_size"`
	Format   string        `json:"format"`
}

// SongMetadata contains metadata about the downloaded song
type SongMetadata struct {
	Title          string        `json:"title"`
	Artist         string        `json:"artist"`
	Album          string        `json:"album"`
	Duration       time.Duration `json:"duration"`
	DurationMillis int           `json:"duration_millis"`
	ArtworkURL     string        `json:"artwork_url"`
	AppleMusicID   string        `json:"apple_music_id"`
}

// SongDownloader interface defines the contract for downloading songs
type SongDownloader interface {
	// Download starts downloading a song from the given URL with progress callbacks
	Download(ctx context.Context, url string, callbacks ProgressCallbacks) (*DownloadResult, error)

	// Cancel cancels any ongoing download operation
	Cancel(ctx context.Context) error

	// GetStatus returns the current download status
	GetStatus() DownloadStatus
}

// DownloadStatus represents the current status of a download
type DownloadStatus struct {
	Phase     Phase     `json:"phase"`
	Progress  Progress  `json:"progress"`
	StartTime time.Time `json:"start_time"`
	SongName  string    `json:"song_name"`
	IsActive  bool      `json:"is_active"`
	Error     error     `json:"error,omitempty"`
}

// ProgressReporter interface defines the contract for reporting progress
type ProgressReporter interface {
	// StartTracking begins progress tracking for a specific chat and song
	StartTracking(ctx context.Context, chatID int64, songName string) error

	// UpdateProgress reports progress for the current phase
	UpdateProgress(phase Phase, progress Progress) error

	// ReportPhaseChange reports a transition between phases
	ReportPhaseChange(oldPhase, newPhase Phase) error

	// ReportError reports an error that occurred during processing
	ReportError(err error) error

	// ReportComplete reports successful completion with summary information
	ReportComplete(duration time.Duration, filePath string) error

	// Stop stops progress tracking and cleans up resources
	Stop()
}
