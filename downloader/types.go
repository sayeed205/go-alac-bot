package downloader

import (
	"io"

	"github.com/abema/go-mp4"
)

// URLMeta contains metadata extracted from Apple Music URLs
type URLMeta struct {
	Storefront string `json:"storefront"`
	URLType    string `json:"url_type"`
	ID         string `json:"id"`
}

// AutoSong represents a song from Apple Music API
type AutoSong struct {
	ID            string         `json:"id"`
	Type          string         `json:"type"`
	Attributes    SongAttributes `json:"attributes"`
	Relationships Relationships  `json:"relationships"`
}

// SongAttributes contains detailed song information
type SongAttributes struct {
	AlbumName                 string            `json:"albumName"`
	HasTimeSyncedLyrics       bool              `json:"hasTimeSyncedLyrics"`
	GenreNames                []string          `json:"genreNames"`
	TrackNumber               int               `json:"trackNumber"`
	DurationInMillis          int               `json:"durationInMillis"`
	ReleaseDate               string            `json:"releaseDate"`
	Name                      string            `json:"name"`
	ISRC                      string            `json:"isrc"`
	ArtistName                string            `json:"artistName"`
	DiscNumber                int               `json:"discNumber"`
	HasLyrics                 bool              `json:"hasLyrics"`
	Artwork                   Artwork           `json:"artwork"`
	ComposerName              string            `json:"composerName"`
	PlayParams                PlayParams        `json:"playParams"`
	URL                       string            `json:"url"`
	AudioTraits               []string          `json:"audioTraits"`
	ExtendedAssetUrls         map[string]string `json:"extendedAssetUrls"`
	Previews                  []Preview         `json:"previews"`
}

// Artwork contains artwork information
type Artwork struct {
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	URL        string `json:"url"`
	BgColor    string `json:"bgColor"`
	TextColor1 string `json:"textColor1"`
	TextColor2 string `json:"textColor2"`
	TextColor3 string `json:"textColor3"`
	TextColor4 string `json:"textColor4"`
}

// PlayParams contains playback parameters
type PlayParams struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
}

// Preview contains preview information
type Preview struct {
	URL string `json:"url"`
}

// Relationships contains related data
type Relationships struct {
	Albums  Relationship `json:"albums"`
	Artists Relationship `json:"artists"`
}

// Relationship represents a relationship to other entities
type Relationship struct {
	Href string             `json:"href"`
	Data []RelationshipData `json:"data"`
}

// RelationshipData contains data about related entities
type RelationshipData struct {
	ID         string           `json:"id"`
	Type       string           `json:"type"`
	Attributes *AlbumAttributes `json:"attributes,omitempty"`
}

// AlbumAttributes contains album-specific attributes
type AlbumAttributes struct {
	Copyright           string     `json:"copyright"`
	GenreNames          []string   `json:"genreNames"`
	ReleaseDate         string     `json:"releaseDate"`
	UPC                 string     `json:"upc"`
	IsMasteredForItunes bool       `json:"isMasteredForItunes"`
	Artwork             Artwork    `json:"artwork"`
	PlayParams          PlayParams `json:"playParams"`
	URL                 string     `json:"url"`
	RecordLabel         string     `json:"recordLabel"`
	TrackCount          int        `json:"trackCount"`
	IsCompilation       bool       `json:"isCompilation"`
	IsPrerelease        bool       `json:"isPrerelease"`
	AudioTraits         []string   `json:"audioTraits"`
	IsSingle            bool       `json:"isSingle"`
	Name                string     `json:"name"`
	ArtistName          string     `json:"artistName"`
	IsComplete          bool       `json:"isComplete"`
}

// SongResponse represents the API response containing songs
type SongResponse struct {
	Data []AutoSong `json:"data"`
}

// SongInfo contains internal song processing information
type SongInfo struct {
	r             io.ReadSeeker
	alacParam     *Alac
	samples       []SampleInfo
	totalDataSize int64
}

// Duration calculates the total duration of the song
func (s *SongInfo) Duration() (ret uint64) {
	for i := range s.samples {
		ret += uint64(s.samples[i].duration)
	}
	return
}

// Alac represents ALAC codec parameters
type Alac struct {
	mp4.FullBox `mp4:"extend"`

	FrameLength       uint32 `mp4:"size=32"`
	CompatibleVersion uint8  `mp4:"size=8"`
	BitDepth          uint8  `mp4:"size=8"`
	Pb                uint8  `mp4:"size=8"`
	Mb                uint8  `mp4:"size=8"`
	Kb                uint8  `mp4:"size=8"`
	NumChannels       uint8  `mp4:"size=8"`
	MaxRun            uint16 `mp4:"size=16"`
	MaxFrameBytes     uint32 `mp4:"size=32"`
	AvgBitRate        uint32 `mp4:"size=32"`
	SampleRate        uint32 `mp4:"size=32"`
}

// GetType returns the box type for ALAC
func (*Alac) GetType() mp4.BoxType {
	return BoxTypeAlac()
}

// BoxTypeAlac returns the ALAC box type
func BoxTypeAlac() mp4.BoxType { 
	return mp4.StrToBoxType("alac") 
}

// SampleInfo contains information about individual samples
type SampleInfo struct {
	data      []byte
	duration  uint32
	descIndex uint32
}