package bot

import (
	"testing"
)

func TestExtractURLMeta(t *testing.T) {
	testCases := []struct {
		name     string
		inputURL string
		expected *URLMeta
	}{
		{
			name:     "Song URL",
			inputURL: "https://music.apple.com/in/song/never-gonna-give-you-up/1559523359",
			expected: &URLMeta{
				Storefront: "in",
				URLType:    "songs",
				ID:         "1559523359",
			},
		},
		{
			name:     "Album URL",
			inputURL: "https://music.apple.com/us/album/3-originals/1559523357",
			expected: &URLMeta{
				Storefront: "us",
				URLType:    "albums",
				ID:         "1559523357",
			},
		},
		{
			name:     "Album URL with song parameter (i)",
			inputURL: "https://music.apple.com/in/album/never-gonna-give-you-up/1559523357?i=1559523359",
			expected: &URLMeta{
				Storefront: "in",
				URLType:    "songs", // Should be converted to song
				ID:         "1559523359", // Should use the 'i' parameter
			},
		},
		{
			name:     "Playlist URL",
			inputURL: "https://music.apple.com/library/playlist/p.vMO5kRQiX1xGMr",
			expected: &URLMeta{
				Storefront: "library", // Note: this might not be a valid storefront but matches the regex
				URLType:    "playlists",
				ID:         "p.vMO5kRQiX1xGMr",
			},
		},
		{
			name:     "Invalid URL",
			inputURL: "https://spotify.com/track/123",
			expected: nil,
		},
		{
			name:     "Empty URL",
			inputURL: "",
			expected: nil,
		},
		{
			name:     "Malformed Apple Music URL",
			inputURL: "https://music.apple.com/invalid",
			expected: nil,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ExtractURLMeta(tc.inputURL)
			
			if tc.expected == nil {
				if result != nil {
					t.Errorf("Expected nil, got %+v", result)
				}
				return
			}
			
			if result == nil {
				t.Errorf("Expected %+v, got nil", tc.expected)
				return
			}
			
			if result.Storefront != tc.expected.Storefront {
				t.Errorf("Storefront: expected %q, got %q", tc.expected.Storefront, result.Storefront)
			}
			
			if result.URLType != tc.expected.URLType {
				t.Errorf("URLType: expected %q, got %q", tc.expected.URLType, result.URLType)
			}
			
			if result.ID != tc.expected.ID {
				t.Errorf("ID: expected %q, got %q", tc.expected.ID, result.ID)
			}
		})
	}
}