// Package downloader provides a modular system for downloading and processing
// Apple Music songs with progress reporting capabilities.
//
// The package defines core interfaces and data structures for:
//   - SongDownloader: Core download functionality with progress callbacks
//   - ProgressReporter: Progress reporting interface for external systems
//   - Error handling with structured DownloadError types
//   - Data structures for Apple Music API integration
//
// This package is designed to be used with the Telegram bot system to provide
// real-time progress updates during song download and upload operations.
package downloader