package downloader

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Sorrow446/go-mp4tag"
	"github.com/abema/go-mp4"
	"github.com/grafov/m3u8"
	"github.com/schollz/progressbar/v3"
)

const (
	defaultId   = "0"
	prefetchKey = "skd://itunes.apple.com/P000000000/s1/e1"
)

func init() {
	mp4.AddBoxDef((*Alac)(nil))
}

// SongDownloaderImpl implements the SongDownloader interface
type SongDownloaderImpl struct {
	deviceUrl      string
	decryptionUrl  string
	forbiddenNames *regexp.Regexp

	// State management
	mu         sync.RWMutex
	status     DownloadStatus
	cancelFunc context.CancelFunc
	isActive   bool
}

// NewSongDownloaderImpl creates a new instance of SongDownloaderImpl
func NewSongDownloaderImpl() SongDownloader {
	return &SongDownloaderImpl{
		deviceUrl:      getEnv("M3U8_URL", "127.0.0.1:20020"),
		decryptionUrl:  getEnv("DEC_URL", "127.0.0.1:10020"),
		forbiddenNames: regexp.MustCompile(`[\\/<>:"|?*]`),
		status: DownloadStatus{
			Phase:    PhaseValidating,
			IsActive: false,
		},
	}
}

// Helper function to get environment variables with fallback
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// Download implements the SongDownloader interface
func (sd *SongDownloaderImpl) Download(ctx context.Context, url string, callbacks ProgressCallbacks) (*DownloadResult, error) {
	sd.mu.Lock()
	if sd.isActive {
		sd.mu.Unlock()
		return nil, NewDownloadError(ErrorUnknown, "download already in progress")
	}

	// Create cancellable context
	downloadCtx, cancel := context.WithCancel(ctx)
	sd.cancelFunc = cancel
	sd.isActive = true
	sd.status.StartTime = time.Now()
	sd.status.IsActive = true
	sd.status.Phase = PhaseValidating
	sd.mu.Unlock()

	defer func() {
		sd.mu.Lock()
		sd.isActive = false
		sd.status.IsActive = false
		sd.cancelFunc = nil
		sd.mu.Unlock()
	}()

	// Clean URL input
	url = strings.TrimFunc(url, func(r rune) bool {
		return r < 32 || r == 127
	})
	if idx := strings.IndexByte(url, 0); idx != -1 {
		url = url[:idx]
	}

	// Phase 1: Validate URL and extract metadata
	sd.updatePhase(PhaseValidating, callbacks)

	urlMeta, err := sd.ExtractUrlMeta(url)
	if err != nil {
		return nil, sd.handleError(ErrorInvalidURL, "failed to extract URL metadata", err, callbacks)
	}

	// Check for cancellation
	if err := downloadCtx.Err(); err != nil {
		return nil, sd.handleError(ErrorCancelled, "download cancelled", err, callbacks)
	}

	// Get authentication token
	token, err := sd.GetToken()
	if err != nil {
		return nil, sd.handleError(ErrorNetworkFailure, "failed to get authentication token", err, callbacks)
	}

	// Get song metadata
	meta, err := sd.GetSongMeta(urlMeta, token)
	if err != nil {
		return nil, sd.handleError(ErrorNetworkFailure, "failed to get song metadata", err, callbacks)
	}

	if meta.Attributes.ExtendedAssetUrls["enhancedHls"] == "" {
		return nil, sd.handleError(ErrorALACNotAvailable, "ALAC format not available for this song", nil, callbacks)
	}

	// Get enhanced HLS URL
	enhancedHls, err := sd.GetEnhanceHls(meta.ID)
	if err != nil {
		return nil, sd.handleError(ErrorNetworkFailure, "failed to get enhanced HLS URL", err, callbacks)
	}

	if strings.HasSuffix(enhancedHls, "m3u8") {
		meta.Attributes.ExtendedAssetUrls["enhancedHls"] = enhancedHls
	}

	// Generate song filename
	songName := fmt.Sprintf("%s - %s", meta.Attributes.Name, meta.Attributes.ArtistName)
	songName = fmt.Sprintf("%s.m4a", sd.forbiddenNames.ReplaceAllString(songName, "_"))

	// Update status with song name
	sd.mu.Lock()
	sd.status.SongName = songName
	sd.mu.Unlock()

	// Check if file already exists
	filePath := filepath.Join("downloads", songName)
	if _, err := os.Stat(filePath); err == nil {
		// File exists, create result and return
		fileInfo, _ := os.Stat(filePath)
		result := &DownloadResult{
			FilePath: filePath,
			SongMeta: &SongMetadata{
				Title:          meta.Attributes.Name,
				Artist:         meta.Attributes.ArtistName,
				Album:          meta.Attributes.AlbumName,
				AppleMusicID:   meta.ID,
				ArtworkURL:     meta.Attributes.Artwork.URL,
				Duration:       time.Duration(meta.Attributes.DurationInMillis) * time.Millisecond,
				DurationMillis: meta.Attributes.DurationInMillis,
			},
			FileSize: fileInfo.Size(),
			Format:   "m4a",
			Duration: time.Since(sd.status.StartTime),
		}

		sd.updatePhase(PhaseComplete, callbacks)
		if callbacks.OnComplete != nil {
			callbacks.OnComplete(result)
		}

		return result, nil
	}

	// Extract media information
	trackUrl, keys, err := sd.ExtractMedia(meta.Attributes.ExtendedAssetUrls["enhancedHls"])
	if err != nil {
		return nil, sd.handleError(ErrorNetworkFailure, "failed to extract media information", err, callbacks)
	}

	// Check for cancellation before starting download
	if err := downloadCtx.Err(); err != nil {
		return nil, sd.handleError(ErrorCancelled, "download cancelled", err, callbacks)
	}

	// Phase 2: Download song data
	sd.updatePhase(PhaseDownloading, callbacks)

	info, err := sd.extractSong(downloadCtx, trackUrl, callbacks)
	if err != nil {
		return nil, sd.handleError(ErrorNetworkFailure, "failed to download song data", err, callbacks)
	}

	// Validate samples and keys
	samplesOk := true
	for _, sample := range info.samples {
		if int(sample.descIndex) >= len(keys) {
			samplesOk = false
			break
		}
	}

	if !samplesOk {
		return nil, sd.handleError(ErrorDecryptionFailure, "decryption size mismatch", nil, callbacks)
	}

	// Check for cancellation before decryption
	if err := downloadCtx.Err(); err != nil {
		return nil, sd.handleError(ErrorCancelled, "download cancelled", err, callbacks)
	}

	// Phase 3: Decrypt song
	sd.updatePhase(PhaseDecrypting, callbacks)

	decrypted, err := sd.decryptSong(downloadCtx, info, keys, meta, callbacks)
	if err != nil {
		return nil, sd.handleError(ErrorDecryptionFailure, "failed to decrypt song", err, callbacks)
	}

	// Check for cancellation before writing
	if err := downloadCtx.Err(); err != nil {
		return nil, sd.handleError(ErrorCancelled, "download cancelled", err, callbacks)
	}

	// Phase 4: Write file
	sd.updatePhase(PhaseWriting, callbacks)

	// Create downloads directory
	err = os.MkdirAll("downloads", os.ModePerm)
	if err != nil {
		return nil, sd.handleError(ErrorFileSystemError, "failed to create downloads directory", err, callbacks)
	}

	// Create and write the file
	file, err := os.Create(filePath)
	if err != nil {
		return nil, sd.handleError(ErrorFileSystemError, "failed to create output file", err, callbacks)
	}
	defer file.Close()

	err = sd.WriteM4a(mp4.NewWriter(file), info, meta, decrypted)
	if err != nil {
		return nil, sd.handleError(ErrorFileSystemError, "failed to write M4A file", err, callbacks)
	}

	// Add artwork
	err = sd.addArtwork(filePath, meta)
	if err != nil {
		// Don't fail the entire download for artwork issues, just log
		fmt.Printf("Warning: failed to add artwork: %v\n", err)
	}

	// Get final file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, sd.handleError(ErrorFileSystemError, "failed to get file info", err, callbacks)
	}

	// Create result
	result := &DownloadResult{
		FilePath: filePath,
		SongMeta: &SongMetadata{
			Title:          meta.Attributes.Name,
			Artist:         meta.Attributes.ArtistName,
			Album:          meta.Attributes.AlbumName,
			AppleMusicID:   meta.ID,
			ArtworkURL:     meta.Attributes.Artwork.URL,
			Duration:       time.Duration(meta.Attributes.DurationInMillis) * time.Millisecond,
			DurationMillis: meta.Attributes.DurationInMillis,
		},
		FileSize: fileInfo.Size(),
		Format:   "m4a",
		Duration: time.Since(sd.status.StartTime),
	}

	// Phase 5: Complete
	sd.updatePhase(PhaseComplete, callbacks)
	if callbacks.OnComplete != nil {
		callbacks.OnComplete(result)
	}

	return result, nil
}

// Cancel implements the SongDownloader interface
func (sd *SongDownloaderImpl) Cancel(ctx context.Context) error {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	if !sd.isActive {
		return NewDownloadError(ErrorUnknown, "no active download to cancel")
	}

	if sd.cancelFunc != nil {
		sd.cancelFunc()
	}

	return nil
}

// GetStatus implements the SongDownloader interface
func (sd *SongDownloaderImpl) GetStatus() DownloadStatus {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	return sd.status
}

// updatePhase updates the current phase and notifies callbacks
func (sd *SongDownloaderImpl) updatePhase(newPhase Phase, callbacks ProgressCallbacks) {
	sd.mu.Lock()
	oldPhase := sd.status.Phase
	sd.status.Phase = newPhase
	sd.mu.Unlock()

	if callbacks.OnPhaseChange != nil && oldPhase != newPhase {
		callbacks.OnPhaseChange(oldPhase, newPhase)
	}
}

// handleError creates a DownloadError and notifies callbacks
func (sd *SongDownloaderImpl) handleError(errorType ErrorType, message string, cause error, callbacks ProgressCallbacks) error {
	sd.mu.Lock()
	sd.status.Phase = PhaseError
	sd.status.Error = cause
	sd.mu.Unlock()

	err := NewDownloadErrorWithCause(errorType, message, cause)

	if callbacks.OnError != nil {
		callbacks.OnError(err)
	}

	return err
} // ExtractUrlMeta extracts metadata from Apple Music URLs
func (sd *SongDownloaderImpl) ExtractUrlMeta(inputURL string) (*URLMeta, error) {
	// Define a regex pattern to match album, song, and playlist URLs, including full playlist ID with hyphens
	reAlbumOrSongOrPlaylist := regexp.MustCompile(`https://music\.apple\.com/(?P<storefront>[a-z]{2})/(?P<type>album|song|playlist)/.*/(?P<id>[0-9a-zA-Z\-.]+)`)

	// Try matching the input URL
	if matches := reAlbumOrSongOrPlaylist.FindStringSubmatch(inputURL); matches != nil {
		// Extract the storefront, type (album, song, or playlist), and ID
		storefront := matches[1]
		urlType := matches[2]
		id := matches[3]

		// Check if it's an album URL with the "i" query parameter (song within album)
		if urlType == "album" {
			u, err := url.Parse(inputURL)
			if err != nil {
				return nil, fmt.Errorf("invalid URL: %v", err)
			}

			// If the query contains "i", use the "i" value as the song ID
			if songID := u.Query().Get("i"); songID != "" {
				id = songID
				urlType = "song" // Treat it as a song since "i" parameter is found
			}
		}

		// Return the parsed metadata
		return &URLMeta{
			Storefront: storefront,
			URLType:    urlType + "s", // Pluralize to "albums", "songs", or "playlists"
			ID:         id,
		}, nil
	}

	return nil, fmt.Errorf("invalid Apple Music URL format")
}

// GetToken retrieves authentication token from Apple Music
func (sd *SongDownloaderImpl) GetToken() (string, error) {
	client := &http.Client{}

	// Step 1: Fetch the main page to find the JS file
	mainPageURL := "https://beta.music.apple.com"
	req, err := http.NewRequest("GET", mainPageURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Find the index-legacy JS URI using regex
	regex := regexp.MustCompile(`/assets/index-legacy-[^/]+\.js`)
	indexJsUri := regex.FindString(string(body))
	if indexJsUri == "" {
		return "", errors.New("index JS file not found")
	}

	// Step 2: Fetch the JS file to extract the token
	jsFileURL := mainPageURL + indexJsUri
	req, err = http.NewRequest("GET", jsFileURL, nil)
	if err != nil {
		return "", err
	}

	resp, err = client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read the JS file content
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Extract the token using regex
	regex = regexp.MustCompile(`eyJh[^"]+`)
	token := regex.FindString(string(body))
	if token == "" {
		return "", errors.New("token not found in JS file")
	}

	return token, nil
}

// GetSongMeta retrieves song metadata from Apple Music API
func (sd *SongDownloaderImpl) GetSongMeta(urlMeta *URLMeta, token string) (*AutoSong, error) {
	URL := fmt.Sprintf("https://amp-api.music.apple.com/v1/catalog/%s/%s/%s", urlMeta.Storefront, urlMeta.URLType, urlMeta.ID)

	req, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		return nil, err
	}

	// Set headers for the request
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Origin", "https://music.apple.com")

	// Prepare query parameters
	query := url.Values{}
	query.Set("include", "albums,explicit")
	query.Set("extend", "extendedAssetUrls")
	query.Set("l", "")
	req.URL.RawQuery = query.Encode()

	// Make the HTTP request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check for a successful response
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(resp.Status)
	}

	// Decode the response body into the SongResponse struct
	var songResponse SongResponse
	if err := json.NewDecoder(resp.Body).Decode(&songResponse); err != nil {
		return nil, err
	}

	for _, d := range songResponse.Data {
		if d.ID == urlMeta.ID {
			return &d, nil
		}
	}

	return nil, errors.New("song not found in response")
} // GetEnhanceHls retrieves enhanced HLS URL from device service
func (sd *SongDownloaderImpl) GetEnhanceHls(songId string) (string, error) {
	conn, err := net.Dial("tcp", sd.deviceUrl)
	if err != nil {
		return "", fmt.Errorf("error connecting to device: %w", err)
	}
	defer conn.Close()

	adamIDBuffer := []byte(songId)
	lengthBuffer := []byte{byte(len(adamIDBuffer))}

	// Write length and adamID to the connection
	_, err = conn.Write(lengthBuffer)
	if err != nil {
		return "", fmt.Errorf("error writing length to device: %w", err)
	}

	_, err = conn.Write(adamIDBuffer)
	if err != nil {
		return "", fmt.Errorf("error writing adamID to device: %w", err)
	}

	// Read the response (URL) from the device
	response, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		return "", fmt.Errorf("error reading response from device: %w", err)
	}

	// Trim any newline characters from the response
	response = bytes.TrimSpace(response)
	if len(response) == 0 {
		return "", errors.New("received empty response from device")
	}

	return string(response), nil
}

// ExtractMedia extracts media URL and keys from HLS manifest
func (sd *SongDownloaderImpl) ExtractMedia(urlStr string) (string, []string, error) {
	masterUrl, err := url.Parse(urlStr)
	if err != nil {
		return "", nil, err
	}

	resp, err := http.Get(urlStr)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", nil, errors.New(resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, err
	}
	masterString := string(body)
	from, listType, err := m3u8.DecodeFrom(strings.NewReader(masterString), true)
	if err != nil || listType != m3u8.MASTER {
		return "", nil, errors.New("m3u8 not of master type")
	}
	master := from.(*m3u8.MasterPlaylist)
	var streamUrl *url.URL
	sort.Slice(master.Variants, func(i, j int) bool {
		return master.Variants[i].AverageBandwidth > master.Variants[j].AverageBandwidth
	})

	for _, variant := range master.Variants {
		if variant.Codecs == "alac" {
			split := strings.Split(variant.Audio, "-")
			length := len(split)
			lengthInt, err := strconv.Atoi(split[length-2])
			if err != nil {
				return "", nil, err
			}
			if lengthInt <= 192000 {
				fmt.Printf("%s-bit / %s Hz\n", split[length-1], split[length-2])
				streamUrlTemp, err := masterUrl.Parse(variant.URI)
				if err != nil {
					return "", nil, err
				}
				streamUrl = streamUrlTemp
				break
			}
		}
	}

	if streamUrl == nil {
		return "", nil, errors.New("no codec found")
	}
	var keys []string
	keys = append(keys, prefetchKey)
	streamUrl.Path = strings.TrimSuffix(streamUrl.Path, ".m3u8") + "_m.mp4"
	regex := regexp.MustCompile(`"(skd?://[^"]*)"`)
	matches := regex.FindAllStringSubmatch(masterString, -1)
	for _, match := range matches {
		if strings.HasSuffix(match[1], "c23") || strings.HasSuffix(match[1], "c6") {
			keys = append(keys, match[1])
		}
	}
	return streamUrl.String(), keys, nil
} // ProgressReader wraps an io.Reader to provide progress callbacks
type ProgressReader struct {
	reader     io.Reader
	total      int64
	read       int64
	onProgress func(read, total int64)
}

func (pr *ProgressReader) Read(p []byte) (n int, err error) {
	n, err = pr.reader.Read(p)
	pr.read += int64(n)
	if pr.onProgress != nil {
		pr.onProgress(pr.read, pr.total)
	}
	return
}

// extractSong downloads and extracts song data with progress reporting
func (sd *SongDownloaderImpl) extractSong(ctx context.Context, url string, callbacks ProgressCallbacks) (*SongInfo, error) {
	track, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer track.Body.Close()
	if track.StatusCode != http.StatusOK {
		return nil, errors.New(track.Status)
	}

	contentLength := track.ContentLength

	// Create a progress reader to track download progress
	progressReader := &ProgressReader{
		reader: track.Body,
		total:  contentLength,
		onProgress: func(read, total int64) {
			if callbacks.OnProgress != nil {
				progress := Progress{
					BytesProcessed: read,
					TotalBytes:     total,
					Percentage:     float64(read) / float64(total) * 100,
				}
				callbacks.OnProgress(PhaseDownloading, progress)
			}
		},
	}

	rawSong, err := io.ReadAll(progressReader)
	if err != nil {
		return nil, err
	}

	// Check for cancellation
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	f := bytes.NewReader(rawSong)

	trex, err := mp4.ExtractBoxWithPayload(f, nil, []mp4.BoxType{
		mp4.BoxTypeMoov(),
		mp4.BoxTypeMvex(),
		mp4.BoxTypeTrex(),
	})
	if err != nil || len(trex) != 1 {
		return nil, err
	}
	trexPay := trex[0].Payload.(*mp4.Trex)

	stbl, err := mp4.ExtractBox(f, nil, []mp4.BoxType{
		mp4.BoxTypeMoov(),
		mp4.BoxTypeTrak(),
		mp4.BoxTypeMdia(),
		mp4.BoxTypeMinf(),
		mp4.BoxTypeStbl(),
	})
	if err != nil || len(stbl) != 1 {
		return nil, err
	}

	var extracted *SongInfo
	enca, err := mp4.ExtractBoxWithPayload(f, stbl[0], []mp4.BoxType{
		mp4.BoxTypeStsd(),
		mp4.BoxTypeEnca(),
	})
	if err != nil {
		return nil, err
	}

	aalac, err := mp4.ExtractBoxWithPayload(f, &enca[0].Info,
		[]mp4.BoxType{BoxTypeAlac()})
	if err != nil || len(aalac) != 1 {
		return nil, err
	}
	extracted = &SongInfo{
		r:         f,
		alacParam: aalac[0].Payload.(*Alac),
	}

	moofs, err := mp4.ExtractBox(f, nil, []mp4.BoxType{
		mp4.BoxTypeMoof(),
	})
	if err != nil || len(moofs) <= 0 {
		return nil, err
	}

	mdats, err := mp4.ExtractBoxWithPayload(f, nil, []mp4.BoxType{
		mp4.BoxTypeMdat(),
	})
	if err != nil || len(mdats) != len(moofs) {
		return nil, err
	}

	for i, moof := range moofs {
		tfhd, err := mp4.ExtractBoxWithPayload(f, moof, []mp4.BoxType{
			mp4.BoxTypeTraf(),
			mp4.BoxTypeTfhd(),
		})
		if err != nil || len(tfhd) != 1 {
			return nil, err
		}
		tfhdPay := tfhd[0].Payload.(*mp4.Tfhd)
		index := tfhdPay.SampleDescriptionIndex
		if index != 0 {
			index--
		}

		truns, err := mp4.ExtractBoxWithPayload(f, moof, []mp4.BoxType{
			mp4.BoxTypeTraf(),
			mp4.BoxTypeTrun(),
		})
		if err != nil || len(truns) <= 0 {
			return nil, err
		}

		mdat := mdats[i].Payload.(*mp4.Mdat).Data
		for _, t := range truns {
			for _, en := range t.Payload.(*mp4.Trun).Entries {
				info := SampleInfo{descIndex: index}

				switch {
				case t.Payload.CheckFlag(0x200):
					info.data = mdat[:en.SampleSize]
					mdat = mdat[en.SampleSize:]
				case tfhdPay.CheckFlag(0x10):
					info.data = mdat[:tfhdPay.DefaultSampleSize]
					mdat = mdat[tfhdPay.DefaultSampleSize:]
				default:
					info.data = mdat[:trexPay.DefaultSampleSize]
					mdat = mdat[trexPay.DefaultSampleSize:]
				}

				switch {
				case t.Payload.CheckFlag(0x100):
					info.duration = en.SampleDuration
				case tfhdPay.CheckFlag(0x8):
					info.duration = tfhdPay.DefaultSampleDuration
				default:
					info.duration = trexPay.DefaultSampleDuration
				}

				extracted.samples = append(extracted.samples, info)
			}
		}
		if len(mdat) != 0 {
			return nil, errors.New("offset mismatch")
		}
	}

	// Calculate total data size
	var totalSize int64 = 0
	for _, sample := range extracted.samples {
		totalSize += int64(len(sample.data))
	}
	extracted.totalDataSize = totalSize

	return extracted, nil
}

// decryptSong decrypts the song data with progress reporting
func (sd *SongDownloaderImpl) decryptSong(ctx context.Context, info *SongInfo, keys []string, manifest *AutoSong, callbacks ProgressCallbacks) ([]byte, error) {
	conn, err := net.Dial("tcp", sd.decryptionUrl)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	var decrypted []byte
	var lastIndex uint32 = math.MaxUint8
	var totalProcessed int64 = 0

	bar := progressbar.NewOptions64(info.totalDataSize,
		progressbar.OptionClearOnFinish(),
		progressbar.OptionSetElapsedTime(false),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionShowElapsedTimeOnFinish(),
		progressbar.OptionShowCount(),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowBytes(true),
		//progressbar.OptionSetDescription("Decrypting..."),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "",
			SaucerHead:    "",
			SaucerPadding: "",
			BarStart:      "",
			BarEnd:        "",
		}),
	)

	for _, sp := range info.samples {
		// Check for cancellation
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		if lastIndex != sp.descIndex {
			if len(decrypted) != 0 {
				_, err := conn.Write([]byte{0, 0, 0, 0})
				if err != nil {
					return nil, err
				}
			}
			keyUri := keys[sp.descIndex]
			id := manifest.ID
			if keyUri == prefetchKey {
				id = defaultId
			}

			_, err := conn.Write([]byte{byte(len(id))})
			if err != nil {
				return nil, err
			}
			_, err = io.WriteString(conn, id)
			if err != nil {
				return nil, err
			}

			_, err = conn.Write([]byte{byte(len(keyUri))})
			if err != nil {
				return nil, err
			}
			_, err = io.WriteString(conn, keyUri)
			if err != nil {
				return nil, err
			}
		}
		lastIndex = sp.descIndex

		err := binary.Write(conn, binary.LittleEndian, uint32(len(sp.data)))
		if err != nil {
			return nil, err
		}

		_, err = conn.Write(sp.data)
		if err != nil {
			return nil, err
		}

		de := make([]byte, len(sp.data))
		_, err = io.ReadFull(conn, de)
		if err != nil {
			return nil, err
		}

		decrypted = append(decrypted, de...)
		bar.Add(len(sp.data))
		totalProcessed += int64(len(sp.data))

		// Report progress
		if callbacks.OnProgress != nil {
			progress := Progress{
				BytesProcessed: totalProcessed,
				TotalBytes:     info.totalDataSize,
				Percentage:     float64(totalProcessed) / float64(info.totalDataSize) * 100,
			}
			callbacks.OnProgress(PhaseDecrypting, progress)
		}
	}

	_, _ = conn.Write([]byte{0, 0, 0, 0, 0})

	return decrypted, nil
}

// addArtwork adds artwork to the M4A file
func (sd *SongDownloaderImpl) addArtwork(filePath string, meta *AutoSong) error {
	artwork := meta.Attributes.Artwork
	coverUrl := strings.Replace(artwork.URL, "{w}x{h}", fmt.Sprintf("%dx%d", artwork.Width, artwork.Height), -1)
	resp, err := http.Get(coverUrl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	cover, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	mp4t, err := mp4tag.Open(filePath)
	if err != nil {
		return err
	}
	defer mp4t.Close()

	tags := &mp4tag.MP4Tags{
		Pictures: []*mp4tag.MP4Picture{{Data: cover}},
	}

	err = mp4t.Write(tags, []string{})
	if err != nil {
		return err
	}

	return nil
}
