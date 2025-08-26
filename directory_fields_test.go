package plex

import (
	"encoding/json"
	"testing"
)

func TestDirectory_CountAndScannedFields(t *testing.T) {
	// Test JSON that includes the count and scanned fields that music libraries should return
	jsonData := `{
		"MediaContainer": {
			"Directory": [
				{
					"key": "1",
					"title": "Movies",
					"type": "movie",
					"agent": "com.plexapp.agents.imdb",
					"scanner": "Plex Movie Scanner",
					"count": 150,
					"scanned": true
				},
				{
					"key": "2", 
					"title": "Music",
					"type": "artist",
					"agent": "com.plexapp.agents.lastfm",
					"scanner": "Plex Music Scanner",
					"count": 0,
					"scanned": false
				},
				{
					"key": "3",
					"title": "TV Shows", 
					"type": "show",
					"agent": "com.plexapp.agents.thetvdb",
					"scanner": "Plex Series Scanner",
					"count": 75,
					"scanned": true
				}
			]
		}
	}`

	var libraries LibrarySections
	err := json.Unmarshal([]byte(jsonData), &libraries)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if len(libraries.MediaContainer.Directory) != 3 {
		t.Errorf("Expected 3 libraries, got %d", len(libraries.MediaContainer.Directory))
	}

	// Test Movies library
	movies := libraries.MediaContainer.Directory[0]
	if movies.Title != "Movies" {
		t.Errorf("Expected Movies library title, got %s", movies.Title)
	}
	if movies.Count != 150 {
		t.Errorf("Expected Movies count 150, got %d", movies.Count)
	}
	if !movies.Scanned {
		t.Errorf("Expected Movies to be scanned")
	}

	// Test Music library (the problematic one)
	music := libraries.MediaContainer.Directory[1]
	if music.Title != "Music" {
		t.Errorf("Expected Music library title, got %s", music.Title)
	}
	if music.Count != 0 {
		t.Errorf("Expected Music count 0 (the issue we're fixing), got %d", music.Count)
	}
	if music.Scanned {
		t.Errorf("Expected Music to not be scanned")
	}
	if music.Type != "artist" {
		t.Errorf("Expected Music type to be artist, got %s", music.Type)
	}

	// Test TV Shows library
	tvShows := libraries.MediaContainer.Directory[2]
	if tvShows.Title != "TV Shows" {
		t.Errorf("Expected TV Shows library title, got %s", tvShows.Title)
	}
	if tvShows.Count != 75 {
		t.Errorf("Expected TV Shows count 75, got %d", tvShows.Count)
	}
	if !tvShows.Scanned {
		t.Errorf("Expected TV Shows to be scanned")
	}
}
