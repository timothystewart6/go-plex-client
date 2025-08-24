package plex

import (
	"testing"
)

// Test helper functions in helpers.go

// Test GetMediaTypeID function
func TestGetMediaTypeID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"movie", "1"},
		{"show", "2"},
		{"season", "3"},
		{"episode", "4"},
		{"trailer", "5"},
		{"comic", "6"},
		{"person", "7"},
		{"artist", "8"},
		{"album", "9"},
		{"track", "10"},
		{"photoAlbum", "11"},
		{"picture", "12"},
		{"photo", "13"},
		{"clip", "14"},
		{"playlistItem", "15"},
		{"invalid", "invalid"},
		{"", ""},
	}

	for _, test := range tests {
		result := GetMediaTypeID(test.input)
		if result != test.expected {
			t.Errorf("GetMediaTypeID(%s) = %s, want %s", test.input, result, test.expected)
		}
	}
}

// Test GetMediaType function
func TestGetMediaType(t *testing.T) {
	// Test with metadata containing type
	metadata := MediaMetadata{
		MediaContainer: MediaContainer{
			Metadata: []Metadata{
				{Type: "movie"},
			},
		},
	}

	result := GetMediaType(metadata)
	if result != "movie" {
		t.Errorf("GetMediaType() = %s, want movie", result)
	}

	// Test with empty metadata
	emptyMetadata := MediaMetadata{
		MediaContainer: MediaContainer{
			Metadata: []Metadata{
				{Type: ""},
			},
		},
	}

	result = GetMediaType(emptyMetadata)
	if result != "" {
		t.Errorf("GetMediaType() = %s, want empty string", result)
	}
}

// Test LibraryParamsFromMediaType function
func TestLibraryParamsFromMediaType(t *testing.T) {
	tests := []struct {
		input           string
		expectedType    string
		expectedAgent   string
		expectedScanner string
		shouldError     bool
	}{
		{"movie", "movie", "com.plexapp.agents.imdb", "Plex Movie Scanner", false},
		{"show", "show", "com.plexapp.agents.thetvdb", "Plex Series Scanner", false},
		{"music", "music", "com.plexapp.agents.lastfm", "Plex Music Scanner", false},
		{"photo", "photo", "com.plexapp.agents.none", "Plex Photo Scanner", false},
		{"homevideo", "homevideo", "com.plexapp.agents.none", "Plex Video Files Scanner", false},
		{"invalid", "invalid", "", "", true},
		{"", "", "", "", true},
	}

	for _, test := range tests {
		params, err := LibraryParamsFromMediaType(test.input)

		if test.shouldError {
			if err == nil {
				t.Errorf("LibraryParamsFromMediaType(%s) expected error", test.input)
			}
			continue
		}

		if err != nil {
			t.Errorf("LibraryParamsFromMediaType(%s) unexpected error: %v", test.input, err)
			continue
		}

		if params.LibraryType != test.expectedType {
			t.Errorf("LibraryParamsFromMediaType(%s) type = %s, want %s", test.input, params.LibraryType, test.expectedType)
		}
		if params.Agent != test.expectedAgent {
			t.Errorf("LibraryParamsFromMediaType(%s) agent = %s, want %s", test.input, params.Agent, test.expectedAgent)
		}
		if params.Scanner != test.expectedScanner {
			t.Errorf("LibraryParamsFromMediaType(%s) scanner = %s, want %s", test.input, params.Scanner, test.expectedScanner)
		}
	}
}
