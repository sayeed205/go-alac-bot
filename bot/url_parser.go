package bot

import (
	"net/url"
	"regexp"
)

// URLMeta represents the parsed metadata from an Apple Music URL
type URLMeta struct {
	Storefront string `json:"storefront"`
	URLType    string `json:"urlType"`
	ID         string `json:"id"`
}

// ExtractURLMeta extracts metadata from Apple Music URLs
func ExtractURLMeta(inputURL string) *URLMeta {
	// Regex to match album, song, and playlist URLs, including full playlist IDs with hyphens
	reAlbumOrSongOrPlaylist := regexp.MustCompile(`https://music\.apple\.com/(?P<storefront>[a-z]{2})/(?P<type>album|song|playlist)/.*/(?P<id>[0-9a-zA-Z\-.]+)`)
	
	matches := reAlbumOrSongOrPlaylist.FindStringSubmatch(inputURL)
	if len(matches) == 0 {
		return nil
	}
	
	// Extract named groups
	names := reAlbumOrSongOrPlaylist.SubexpNames()
	result := make(map[string]string)
	for i, match := range matches {
		if i > 0 && names[i] != "" {
			result[names[i]] = match
		}
	}
	
	storefront := result["storefront"]
	urlType := result["type"]
	id := result["id"]
	
	// Handle album URLs with the "i" query parameter (song within album)
	if urlType == "album" {
		parsedURL, err := url.Parse(inputURL)
		if err == nil {
			// If the query contains "i", use the "i" value as the song ID
			songID := parsedURL.Query().Get("i")
			if songID != "" {
				id = songID
				urlType = "song" // Treat as "song" since "i" parameter is found
			}
		}
	}
	
	// Return the parsed metadata, pluralizing the type
	return &URLMeta{
		Storefront: storefront,
		URLType:    urlType + "s", // Pluralize to 'albums', 'songs', or 'playlists'
		ID:         id,
	}
}