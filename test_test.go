package plex

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test helper functions
func newJSONTestServer(code int, body interface{}) (*httptest.Server, *Plex) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(code)
		w.Header().Set("Content-Type", applicationJson)
		if body != nil {
			json.NewEncoder(w).Encode(body)
		}
	}))

	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}

	httpClient := http.Client{Transport: transport}
	plex := &Plex{URL: server.URL, Token: "test-token", HTTPClient: httpClient, Headers: defaultHeaders()}

	return server, plex
}

func newXMLTestServer(code int, body string) (*httptest.Server, *Plex) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(code)
		w.Header().Set("Content-Type", applicationXml)
		fmt.Fprintln(w, body)
	}))

	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}

	httpClient := http.Client{Transport: transport}
	plex := &Plex{URL: server.URL, Token: "test-token", HTTPClient: httpClient, Headers: defaultHeaders()}

	return server, plex
}

// Test New function with various scenarios
func TestNew_AllCombinations(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		token   string
		wantErr bool
		errMsg  string
	}{
		{"Both empty", "", "", true, ErrorUrlTokenRequired},
		{"Valid both", "http://localhost:32400", "token123", false, ""},
		{"Only token", "", "token123", false, ""},
		{"Only URL", "http://localhost:32400", "", false, ""},
		{"Invalid URL", "not-a-url", "token123", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plex, err := New(tt.baseURL, tt.token)

			if tt.wantErr {
				if err == nil {
					t.Errorf("New() expected error but got none")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("New() error = %v, expected to contain %v", err, tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("New() unexpected error = %v", err)
				return
			}

			if plex == nil {
				t.Errorf("New() returned nil plex instance")
				return
			}

			// Verify fields are set correctly
			if tt.baseURL != "" && plex.URL != tt.baseURL {
				t.Errorf("New() URL = %v, want %v", plex.URL, tt.baseURL)
			}
			if tt.token != "" && plex.Token != tt.token {
				t.Errorf("New() Token = %v, want %v", plex.Token, tt.token)
			}
		})
	}
}

// Test defaultHeaders function
func TestDefaultHeaders(t *testing.T) {
	headers := defaultHeaders()

	if headers.Product != "Go Plex Client" {
		t.Errorf("defaultHeaders() Product = %v, want 'Go Plex Client'", headers.Product)
	}
	if headers.Accept != applicationJson {
		t.Errorf("defaultHeaders() Accept = %v, want %v", headers.Accept, applicationJson)
	}
	if headers.ContentType != applicationJson {
		t.Errorf("defaultHeaders() ContentType = %v, want %v", headers.ContentType, applicationJson)
	}
}

// Test Search function
func TestPlex_Search(t *testing.T) {
	searchResponse := SearchResults{
		MediaContainer: SearchMediaContainer{
			MediaContainer: MediaContainer{
				Size: 1,
				Metadata: []Metadata{
					{Title: "Test Movie", Type: "movie", Year: 2023},
				},
			},
		},
	}

	server, plex := newJSONTestServer(200, searchResponse)
	defer server.Close()

	// Test successful search
	results, err := plex.Search("test movie")
	if err != nil {
		t.Errorf("Search() error = %v", err)
		return
	}

	if results.MediaContainer.Size != 1 {
		t.Errorf("Search() size = %v, want 1", results.MediaContainer.Size)
	}

	// Test empty title
	_, err = plex.Search("")
	if err == nil {
		t.Errorf("Search() expected error for empty title")
	}

	// Test unauthorized
	server401, plex401 := newJSONTestServer(401, nil)
	defer server401.Close()

	_, err = plex401.Search("test")
	if err == nil {
		t.Errorf("Search() expected error for 401")
	}
}

// Test GetMetadata function
func TestPlex_GetMetadata(t *testing.T) {
	metadataResponse := MediaMetadata{
		MediaContainer: MediaContainer{
			Size: 1,
			Metadata: []Metadata{
				{Title: "Test Episode", Type: "episode", Year: 2023},
			},
		},
	}

	server, plex := newJSONTestServer(200, metadataResponse)
	defer server.Close()

	// Test successful metadata retrieval
	result, err := plex.GetMetadata("12345")
	if err != nil {
		t.Errorf("GetMetadata() error = %v", err)
		return
	}

	if len(result.MediaContainer.Metadata) != 1 {
		t.Errorf("GetMetadata() metadata count = %v, want 1", len(result.MediaContainer.Metadata))
	}

	// Test empty key
	_, err = plex.GetMetadata("")
	if err == nil {
		t.Errorf("GetMetadata() expected error for empty key")
	}

	// Test server error
	server500, plex500 := newJSONTestServer(500, nil)
	defer server500.Close()

	_, err = plex500.GetMetadata("12345")
	if err == nil {
		t.Errorf("GetMetadata() expected error for 500")
	}
}

// Test GetMetadataChildren function
func TestPlex_GetMetadataChildren(t *testing.T) {
	childrenResponse := MetadataChildren{
		MediaContainer: MediaContainer{
			Size: 2,
			Metadata: []Metadata{
				{Title: "Season 1", Type: "season"},
				{Title: "Season 2", Type: "season"},
			},
		},
	}

	server, plex := newJSONTestServer(200, childrenResponse)
	defer server.Close()

	result, err := plex.GetMetadataChildren("12345")
	if err != nil {
		t.Errorf("GetMetadataChildren() error = %v", err)
		return
	}

	if len(result.MediaContainer.Metadata) != 2 {
		t.Errorf("GetMetadataChildren() metadata count = %v, want 2", len(result.MediaContainer.Metadata))
	}

	// Test empty key
	_, err = plex.GetMetadataChildren("")
	if err == nil {
		t.Errorf("GetMetadataChildren() expected error for empty key")
	}
}

// Test GetEpisodes function
func TestPlex_GetEpisodes(t *testing.T) {
	episodesResponse := SearchResultsEpisode{
		MediaContainer: MediaContainer{
			Size: 3,
			Metadata: []Metadata{
				{Title: "Episode 1", Type: "episode"},
				{Title: "Episode 2", Type: "episode"},
				{Title: "Episode 3", Type: "episode"},
			},
		},
	}

	server, plex := newJSONTestServer(200, episodesResponse)
	defer server.Close()

	result, err := plex.GetEpisodes("season123")
	if err != nil {
		t.Errorf("GetEpisodes() error = %v", err)
		return
	}

	if len(result.MediaContainer.Metadata) != 3 {
		t.Errorf("GetEpisodes() episode count = %v, want 3", len(result.MediaContainer.Metadata))
	}

	// Test empty key
	_, err = plex.GetEpisodes("")
	if err == nil {
		t.Errorf("GetEpisodes() expected error for empty key")
	}
}

// Test GetEpisode function
func TestPlex_GetEpisode(t *testing.T) {
	episodeResponse := SearchResultsEpisode{
		MediaContainer: MediaContainer{
			Size: 1,
			Metadata: []Metadata{
				{Title: "Pilot", Type: "episode", Year: 2023},
			},
		},
	}

	server, plex := newJSONTestServer(200, episodeResponse)
	defer server.Close()

	result, err := plex.GetEpisode("episode123")
	if err != nil {
		t.Errorf("GetEpisode() error = %v", err)
		return
	}

	if len(result.MediaContainer.Metadata) != 1 {
		t.Errorf("GetEpisode() metadata count = %v, want 1", len(result.MediaContainer.Metadata))
	}

	// Test empty key
	_, err = plex.GetEpisode("")
	if err == nil {
		t.Errorf("GetEpisode() expected error for empty key")
	}
}

// Test GetOnDeck function
func TestPlex_GetOnDeck(t *testing.T) {
	onDeckResponse := SearchResultsEpisode{
		MediaContainer: MediaContainer{
			Size: 2,
			Metadata: []Metadata{
				{Title: "Continue Watching 1", Type: "episode"},
				{Title: "Continue Watching 2", Type: "movie"},
			},
		},
	}

	server, plex := newJSONTestServer(200, onDeckResponse)
	defer server.Close()

	result, err := plex.GetOnDeck()
	if err != nil {
		t.Errorf("GetOnDeck() error = %v", err)
		return
	}

	if len(result.MediaContainer.Metadata) != 2 {
		t.Errorf("GetOnDeck() metadata count = %v, want 2", len(result.MediaContainer.Metadata))
	}
}

// Test GetPlaylist function
func TestPlex_GetPlaylist(t *testing.T) {
	playlistResponse := SearchResultsEpisode{
		MediaContainer: MediaContainer{
			Size: 3,
			Metadata: []Metadata{
				{Title: "Song 1", Type: "track"},
				{Title: "Song 2", Type: "track"},
				{Title: "Song 3", Type: "track"},
			},
		},
	}

	server, plex := newJSONTestServer(200, playlistResponse)
	defer server.Close()

	result, err := plex.GetPlaylist(123)
	if err != nil {
		t.Errorf("GetPlaylist() error = %v", err)
		return
	}

	if len(result.MediaContainer.Metadata) != 3 {
		t.Errorf("GetPlaylist() metadata count = %v, want 3", len(result.MediaContainer.Metadata))
	}

	// Test server error
	server500, plex500 := newJSONTestServer(500, nil)
	defer server500.Close()

	_, err = plex500.GetPlaylist(123)
	if err == nil {
		t.Errorf("GetPlaylist() expected error for 500")
	}
}

// Test GetThumbnail function
func TestPlex_GetThumbnail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/library/metadata/123/thumb/456") {
			t.Errorf("GetThumbnail() wrong path = %v", r.URL.Path)
		}
		w.WriteHeader(200)
		w.Write([]byte("fake image data"))
	}))
	defer server.Close()

	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}

	httpClient := http.Client{Transport: transport}
	plex := &Plex{URL: server.URL, Token: "test-token", HTTPClient: httpClient, Headers: defaultHeaders()}

	resp, err := plex.GetThumbnail("123", "456")
	if err != nil {
		t.Errorf("GetThumbnail() error = %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("GetThumbnail() status = %v, want 200", resp.StatusCode)
	}
}

// Test KillTranscodeSession function
func TestPlex_KillTranscodeSession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/video/:/transcode/universal/stop") {
			t.Errorf("KillTranscodeSession() wrong path = %v", r.URL.Path)
		}
		if !strings.Contains(r.URL.RawQuery, "session=abc123") {
			t.Errorf("KillTranscodeSession() missing session param in query = %v", r.URL.RawQuery)
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}

	httpClient := http.Client{Transport: transport}
	plex := &Plex{URL: server.URL, Token: "test-token", HTTPClient: httpClient, Headers: defaultHeaders()}

	result, err := plex.KillTranscodeSession("abc123")
	if err != nil {
		t.Errorf("KillTranscodeSession() error = %v", err)
		return
	}

	if !result {
		t.Errorf("KillTranscodeSession() result = %v, want true", result)
	}

	// Test empty session key
	_, err = plex.KillTranscodeSession("")
	if err == nil {
		t.Errorf("KillTranscodeSession() expected error for empty session key")
	}

	// Test unauthorized
	server401 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
	}))
	defer server401.Close()

	transport401 := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server401.URL)
		},
	}

	httpClient401 := http.Client{Transport: transport401}
	plex401 := &Plex{URL: server401.URL, Token: "test-token", HTTPClient: httpClient401, Headers: defaultHeaders()}

	_, err = plex401.KillTranscodeSession("abc123")
	if err == nil {
		t.Errorf("KillTranscodeSession() expected error for 401")
	}
}

// Test GetTranscodeSessions function
func TestPlex_GetTranscodeSessions(t *testing.T) {
	transcodeResponse := TranscodeSessionsResponse{
		Children: []struct {
			ElementType      string  `json:"_elementType"`
			AudioChannels    int     `json:"audioChannels"`
			AudioCodec       string  `json:"audioCodec"`
			AudioDecision    string  `json:"audioDecision"`
			SubtitleDecision string  `json:"subtitleDecision"`
			Container        string  `json:"container"`
			Context          string  `json:"context"`
			Duration         int     `json:"duration"`
			Height           int     `json:"height"`
			Key              string  `json:"key"`
			Progress         float64 `json:"progress"`
			Protocol         string  `json:"protocol"`
			Remaining        int     `json:"remaining"`
			Speed            float64 `json:"speed"`
			Throttled        bool    `json:"throttled"`
			VideoCodec       string  `json:"videoCodec"`
			VideoDecision    string  `json:"videoDecision"`
			Width            int     `json:"width"`
		}{
			{Key: "session1", Progress: 50.0, VideoCodec: "h264"},
		},
	}

	server, plex := newJSONTestServer(200, transcodeResponse)
	defer server.Close()

	result, err := plex.GetTranscodeSessions()
	if err != nil {
		t.Errorf("GetTranscodeSessions() error = %v", err)
		return
	}

	if len(result.Children) != 1 {
		t.Errorf("GetTranscodeSessions() children count = %v, want 1", len(result.Children))
	}
}

// Test GetLibraries function
func TestPlex_GetLibraries(t *testing.T) {
	librariesResponse := LibrarySections{
		MediaContainer: struct {
			Directory []Directory `json:"Directory"`
		}{
			Directory: []Directory{
				{Key: "1", Title: "Movies", Type: "movie"},
				{Key: "2", Title: "TV Shows", Type: "show"},
			},
		},
	}

	server, plex := newJSONTestServer(200, librariesResponse)
	defer server.Close()

	result, err := plex.GetLibraries()
	if err != nil {
		t.Errorf("GetLibraries() error = %v", err)
		return
	}

	if len(result.MediaContainer.Directory) != 2 {
		t.Errorf("GetLibraries() directory count = %v, want 2", len(result.MediaContainer.Directory))
	}

	// Test server error
	server500, plex500 := newJSONTestServer(500, nil)
	defer server500.Close()

	_, err = plex500.GetLibraries()
	if err == nil {
		t.Errorf("GetLibraries() expected error for 500")
	}
}

// Test GetLibrariesWithCounts function
func TestPlex_GetLibrariesWithCounts(t *testing.T) {
	// Mock the /library/sections response
	sectionsResponse := LibrarySections{
		MediaContainer: struct {
			Directory []Directory `json:"Directory"`
		}{
			Directory: []Directory{
				{Key: "1", Title: "Movies", Type: "movie"},
				{Key: "2", Title: "Music", Type: "artist"},
				{Key: "3", Title: "TV Shows", Type: "show"},
			},
		},
	}

	// Mock responses for individual library content
	movieContent := SearchResults{
		MediaContainer: SearchMediaContainer{
			MediaContainer: MediaContainer{
				Size: 150, // Movies count
			},
		},
	}

	musicContent := SearchResults{
		MediaContainer: SearchMediaContainer{
			MediaContainer: MediaContainer{
				Size: 1250, // Music tracks count
			},
		},
	}

	tvContent := SearchResults{
		MediaContainer: SearchMediaContainer{
			MediaContainer: MediaContainer{
				Size: 75, // TV episodes count
			},
		},
	}

	// Create test server that handles multiple endpoints
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		switch r.URL.Path {
		case "/library/sections":
			json.NewEncoder(w).Encode(sectionsResponse)
		case "/library/sections/1/all":
			json.NewEncoder(w).Encode(movieContent)
		case "/library/sections/2/all":
			json.NewEncoder(w).Encode(musicContent)
		case "/library/sections/3/all":
			json.NewEncoder(w).Encode(tvContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create Plex client with test server
	plex := &Plex{
		URL:     server.URL,
		Token:   "test-token",
		Headers: defaultHeaders(),
	}

	// Test the function
	result, err := plex.GetLibrariesWithCounts()
	if err != nil {
		t.Errorf("GetLibrariesWithCounts() error = %v", err)
		return
	}

	if len(result.MediaContainer.Directory) != 3 {
		t.Errorf("GetLibrariesWithCounts() directory count = %v, want 3", len(result.MediaContainer.Directory))
		return
	}

	// Check Movies library
	movies := result.MediaContainer.Directory[0]
	if movies.Title != "Movies" {
		t.Errorf("Movies library title = %v, want Movies", movies.Title)
	}
	if movies.Count != 150 {
		t.Errorf("Movies library count = %v, want 150", movies.Count)
	}

	// Check Music library (this was the problematic one)
	music := result.MediaContainer.Directory[1]
	if music.Title != "Music" {
		t.Errorf("Music library title = %v, want Music", music.Title)
	}
	if music.Count != 1250 {
		t.Errorf("Music library count = %v, want 1250", music.Count)
	}

	// Check TV Shows library
	tv := result.MediaContainer.Directory[2]
	if tv.Title != "TV Shows" {
		t.Errorf("TV Shows library title = %v, want TV Shows", tv.Title)
	}
	if tv.Count != 75 {
		t.Errorf("TV Shows library count = %v, want 75", tv.Count)
	}
}

// Test GetLibrariesWithCounts error handling
// Duplicate TestPlex_GetLibrariesWithCounts_ErrorHandling removed to fix redeclaration error.

// Test Directory CountAndScanned Fields
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

// Test GetLibraryContent function
func TestPlex_GetLibraryContent(t *testing.T) {
	contentResponse := SearchResults{
		MediaContainer: SearchMediaContainer{
			MediaContainer: MediaContainer{
				Size: 2,
				Metadata: []Metadata{
					{Title: "Movie 1", Type: "movie"},
					{Title: "Movie 2", Type: "movie"},
				},
			},
		},
	}

	server, plex := newJSONTestServer(200, contentResponse)
	defer server.Close()

	result, err := plex.GetLibraryContent("1", "")
	if err != nil {
		t.Errorf("GetLibraryContent() error = %v", err)
		return
	}

	if len(result.MediaContainer.Metadata) != 2 {
		t.Errorf("GetLibraryContent() metadata count = %v, want 2", len(result.MediaContainer.Metadata))
	}

	// Test with filter
	result, err = plex.GetLibraryContent("1", "?type=1")
	if err != nil {
		t.Errorf("GetLibraryContent() with filter error = %v", err)
	}
}

// Test CreateLibrary function
func TestPlex_CreateLibrary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("CreateLibrary() method = %v, want POST", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/library/sections") {
			t.Errorf("CreateLibrary() path = %v", r.URL.Path)
		}
		w.WriteHeader(201)
	}))
	defer server.Close()

	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}

	httpClient := http.Client{Transport: transport}
	plex := &Plex{URL: server.URL, Token: "test-token", HTTPClient: httpClient, Headers: defaultHeaders()}

	params := CreateLibraryParams{
		Name:        "Test Library",
		Location:    "/path/to/media",
		LibraryType: "movie",
		Agent:       "com.plexapp.agents.imdb",
		Scanner:     "Plex Movie Scanner",
		Language:    "en",
	}

	err := plex.CreateLibrary(params)
	if err != nil {
		t.Errorf("CreateLibrary() error = %v", err)
	}

	// Test missing required fields
	testCases := []struct {
		name   string
		params CreateLibraryParams
	}{
		{"missing name", CreateLibraryParams{Location: "/path", LibraryType: "movie", Agent: "agent", Scanner: "scanner"}},
		{"missing location", CreateLibraryParams{Name: "test", LibraryType: "movie", Agent: "agent", Scanner: "scanner"}},
		{"missing type", CreateLibraryParams{Name: "test", Location: "/path", Agent: "agent", Scanner: "scanner"}},
		{"missing agent", CreateLibraryParams{Name: "test", Location: "/path", LibraryType: "movie", Scanner: "scanner"}},
		{"missing scanner", CreateLibraryParams{Name: "test", Location: "/path", LibraryType: "movie", Agent: "agent"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := plex.CreateLibrary(tc.params)
			if err == nil {
				t.Errorf("CreateLibrary() expected error for %s", tc.name)
			}
		})
	}
}

// Test DeleteLibrary function
func TestPlex_DeleteLibrary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("DeleteLibrary() method = %v, want DELETE", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/library/sections/123") {
			t.Errorf("DeleteLibrary() path = %v", r.URL.Path)
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}

	httpClient := http.Client{Transport: transport}
	plex := &Plex{URL: server.URL, Token: "test-token", HTTPClient: httpClient, Headers: defaultHeaders()}

	err := plex.DeleteLibrary("123")
	if err != nil {
		t.Errorf("DeleteLibrary() error = %v", err)
	}

	// Test server error
	server500 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer server500.Close()

	transport500 := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server500.URL)
		},
	}

	httpClient500 := http.Client{Transport: transport500}
	plex500 := &Plex{URL: server500.URL, Token: "test-token", HTTPClient: httpClient500, Headers: defaultHeaders()}

	err = plex500.DeleteLibrary("123")
	if err == nil {
		t.Errorf("DeleteLibrary() expected error for 500")
	}
}

// Test DeleteMediaByID function
func TestPlex_DeleteMediaByID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("DeleteMediaByID() method = %v, want DELETE", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/library/metadata/123") {
			t.Errorf("DeleteMediaByID() path = %v", r.URL.Path)
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}

	httpClient := http.Client{Transport: transport}
	plex := &Plex{URL: server.URL, Token: "test-token", HTTPClient: httpClient, Headers: defaultHeaders()}

	err := plex.DeleteMediaByID("123")
	if err != nil {
		t.Errorf("DeleteMediaByID() error = %v", err)
	}
}

// Test GetLibraryLabels function
func TestPlex_GetLibraryLabels(t *testing.T) {
	labelsResponse := LibraryLabels{
		ElementType: "Directory",
		Title1:      "Labels",
		Children: []struct {
			ElementType string `json:"_elementType"`
			FastKey     string `json:"fastKey"`
			Key         string `json:"key"`
			Title       string `json:"title"`
		}{
			{Title: "Action", Key: "action"},
			{Title: "Comedy", Key: "comedy"},
		},
	}

	server, plex := newJSONTestServer(200, labelsResponse)
	defer server.Close()

	result, err := plex.GetLibraryLabels("1", "")
	if err != nil {
		t.Errorf("GetLibraryLabels() error = %v", err)
		return
	}

	if len(result.Children) != 2 {
		t.Errorf("GetLibraryLabels() children count = %v, want 2", len(result.Children))
	}

	// Test with section index
	result, err = plex.GetLibraryLabels("1", "2")
	if err != nil {
		t.Errorf("GetLibraryLabels() with section index error = %v", err)
	}
}

// Test AddLabelToMedia function
func TestPlex_AddLabelToMedia(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("AddLabelToMedia() method = %v, want PUT", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/library/sections/1/all") {
			t.Errorf("AddLabelToMedia() path = %v", r.URL.Path)
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}

	httpClient := http.Client{Transport: transport}
	plex := &Plex{URL: server.URL, Token: "test-token", HTTPClient: httpClient, Headers: defaultHeaders()}

	result, err := plex.AddLabelToMedia("1", "1", "123", "Action", "0")
	if err != nil {
		t.Errorf("AddLabelToMedia() error = %v", err)
	}

	if !result {
		t.Errorf("AddLabelToMedia() result = %v, want true", result)
	}
}

// Test RemoveLabelFromMedia function
func TestPlex_RemoveLabelFromMedia(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("RemoveLabelFromMedia() method = %v, want PUT", r.Method)
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}

	httpClient := http.Client{Transport: transport}
	plex := &Plex{URL: server.URL, Token: "test-token", HTTPClient: httpClient, Headers: defaultHeaders()}

	result, err := plex.RemoveLabelFromMedia("1", "1", "123", "Action", "0")
	if err != nil {
		t.Errorf("RemoveLabelFromMedia() error = %v", err)
	}

	if !result {
		t.Errorf("RemoveLabelFromMedia() result = %v, want true", result)
	}
}

// Test TerminateSession function
func TestPlex_TerminateSession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/status/sessions/terminate") {
			t.Errorf("TerminateSession() path = %v", r.URL.Path)
		}
		if !strings.Contains(r.URL.RawQuery, "sessionId=abc123") {
			t.Errorf("TerminateSession() missing sessionId in query = %v", r.URL.RawQuery)
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}

	httpClient := http.Client{Transport: transport}
	plex := &Plex{URL: server.URL, Token: "test-token", HTTPClient: httpClient, Headers: defaultHeaders()}

	err := plex.TerminateSession("abc123", "Test termination")
	if err != nil {
		t.Errorf("TerminateSession() error = %v", err)
	}

	// Test with default reason
	err = plex.TerminateSession("abc123", "")
	if err != nil {
		t.Errorf("TerminateSession() with default reason error = %v", err)
	}
}

// Test Download function
func TestPlex_Download(t *testing.T) {
	// Create temporary directory for test
	tmpDir := filepath.Join(os.TempDir(), "plex_test_download")
	err := os.MkdirAll(tmpDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/library/parts/") && strings.Contains(r.URL.RawQuery, "download=1") {
			w.WriteHeader(200)
			w.Write([]byte("fake media content"))
		} else {
			w.WriteHeader(404)
		}
	}))
	defer server.Close()

	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}

	downloadClient := http.Client{Transport: transport}
	plex := &Plex{URL: server.URL, Token: "test-token", DownloadClient: downloadClient, Headers: defaultHeaders()}

	// Test metadata with no media
	emptyMetadata := Metadata{Title: "Test"}
	err = plex.Download(emptyMetadata, tmpDir, false, false)
	if err == nil {
		t.Errorf("Download() expected error for metadata with no media")
	}

	// Test metadata with media
	metadata := Metadata{
		Title: "Test Movie",
		Media: []Media{
			{
				Part: []Part{
					{Key: "/library/parts/123/file.mp4", File: "/path/to/file.mp4"},
				},
			},
		},
	}

	err = plex.Download(metadata, tmpDir, false, false)
	if err != nil {
		t.Errorf("Download() error = %v", err)
	}

	// Test with folder creation for TV show
	tvMetadata := Metadata{
		Title:            "Episode 1",
		ParentTitle:      "Season 1",
		GrandparentTitle: "Test Show",
		Media: []Media{
			{
				Part: []Part{
					{Key: "/library/parts/456/episode.mkv", File: "/path/to/episode.mkv"},
				},
			},
		},
	}

	err = plex.Download(tvMetadata, tmpDir, true, false)
	if err != nil {
		t.Errorf("Download() with folders error = %v", err)
	}
}

// Test SignIn function
func TestPlex_SignIn(t *testing.T) {
	// Test successful sign in
	successResponse := UserPlexTV{
		AuthToken: "test-auth-token-123",
		UUID:      "user-123",
		Email:     "test@example.com",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("SignIn() method = %v, want POST", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/api/v2/users/signin") {
			t.Errorf("SignIn() path = %v", r.URL.Path)
		}

		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", applicationJson)
		json.NewEncoder(w).Encode(successResponse)
	}))
	defer server.Close()

	// Override global plexURL for this test
	originalPlexURL := plexURL
	plexURL = server.URL
	defer func() { plexURL = originalPlexURL }()

	plex, err := SignIn("testuser", "testpass")
	if err != nil {
		t.Errorf("SignIn() error = %v", err)
	}

	if plex.Token != "test-auth-token-123" {
		t.Errorf("SignIn() token = %v, want test-auth-token-123", plex.Token)
	}

	// Test sign in error response
	serverError := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized"))
	}))
	defer serverError.Close()

	plexURL = serverError.URL
	_, err = SignIn("baduser", "badpass")
	if err == nil {
		t.Errorf("SignIn() expected error for unauthorized")
	}

	// Test invalid JSON response
	serverBadJSON := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("invalid json"))
	}))
	defer serverBadJSON.Close()

	plexURL = serverBadJSON.URL
	_, err = SignIn("user", "pass")
	if err == nil {
		t.Errorf("SignIn() expected error for invalid JSON")
	}
}

// Test Test function
func TestPlex_Test(t *testing.T) {
	// Test successful connection
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/api/servers") {
			t.Errorf("Test() path = %v", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	originalPlexURL := plexURL
	plexURL = server.URL
	defer func() { plexURL = originalPlexURL }()

	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}

	httpClient := http.Client{Transport: transport}
	headers := defaultHeaders()
	plex := &Plex{URL: server.URL, Token: "test-token", ClientIdentifier: headers.ClientIdentifier, HTTPClient: httpClient, Headers: headers}

	result, err := plex.Test()
	if err != nil {
		t.Errorf("Test() error = %v", err)
	}
	if !result {
		t.Errorf("Test() result = %v, want true", result)
	}

	// Test unauthorized response
	serverUnauth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized"))
	}))
	defer serverUnauth.Close()

	plexURL = serverUnauth.URL
	transportUnauth := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(serverUnauth.URL)
		},
	}

	plexUnauth := &Plex{URL: serverUnauth.URL, Token: "invalid-token", ClientIdentifier: headers.ClientIdentifier, HTTPClient: http.Client{Transport: transportUnauth}, Headers: headers}

	_, err = plexUnauth.Test()
	if err == nil {
		t.Errorf("Test() expected error for unauthorized")
	}

	// Test non-OK status
	serverError := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Server Error"))
	}))
	defer serverError.Close()

	plexURL = serverError.URL
	transportError := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(serverError.URL)
		},
	}

	plexError := &Plex{URL: serverError.URL, Token: "test-token", ClientIdentifier: headers.ClientIdentifier, HTTPClient: http.Client{Transport: transportError}, Headers: headers}

	_, err = plexError.Test()
	if err == nil {
		t.Errorf("Test() expected error for server error")
	}
}

// Test GetPlexTokens function
func TestPlex_GetPlexTokens(t *testing.T) {
	devicesResponse := DevicesResponse{
		ID:         1,
		LastSeenAt: "2023-01-02T00:00:00Z",
		Name:       "Test Device",
		Product:    "Plex Media Server",
		Version:    "1.0.0",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/devices.json") {
			t.Errorf("GetPlexTokens() path = %v", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", applicationJson)
		json.NewEncoder(w).Encode(devicesResponse)
	}))
	defer server.Close()

	originalPlexURL := plexURL
	plexURL = server.URL
	defer func() { plexURL = originalPlexURL }()

	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}

	httpClient := http.Client{Transport: transport}
	headers := defaultHeaders()
	plex := &Plex{URL: server.URL, Token: "test-token", ClientIdentifier: headers.ClientIdentifier, HTTPClient: httpClient, Headers: headers}

	result, err := plex.GetPlexTokens("test-token")
	if err != nil {
		t.Errorf("GetPlexTokens() error = %v", err)
	}

	if result.Name != "Test Device" {
		t.Errorf("GetPlexTokens() device name = %v, want Test Device", result.Name)
	}

	// Test unauthorized response
	serverUnauth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized"))
	}))
	defer serverUnauth.Close()

	plexURL = serverUnauth.URL
	transportUnauth := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(serverUnauth.URL)
		},
	}

	plexUnauth := &Plex{URL: serverUnauth.URL, Token: "invalid-token", ClientIdentifier: headers.ClientIdentifier, HTTPClient: http.Client{Transport: transportUnauth}, Headers: headers}

	_, err = plexUnauth.GetPlexTokens("invalid-token")
	if err == nil {
		t.Errorf("GetPlexTokens() expected error for unauthorized")
	}
}

// Test DeletePlexToken function
func TestPlex_DeletePlexToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/devices/test-token.json") {
			t.Errorf("DeletePlexToken() path = %v", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", applicationJson)
		json.NewEncoder(w).Encode(true)
	}))
	defer server.Close()

	originalPlexURL := plexURL
	plexURL = server.URL
	defer func() { plexURL = originalPlexURL }()

	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}

	httpClient := http.Client{Transport: transport}
	headers := defaultHeaders()
	plex := &Plex{URL: server.URL, Token: "test-token", ClientIdentifier: headers.ClientIdentifier, HTTPClient: httpClient, Headers: headers}

	result, err := plex.DeletePlexToken("test-token")
	if err != nil {
		t.Errorf("DeletePlexToken() error = %v", err)
	}

	if !result {
		t.Errorf("DeletePlexToken() result = %v, want true", result)
	}

	// Test unauthorized response
	serverUnauth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized"))
	}))
	defer serverUnauth.Close()

	plexURL = serverUnauth.URL
	transportUnauth := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(serverUnauth.URL)
		},
	}

	plexUnauth := &Plex{URL: serverUnauth.URL, Token: "invalid-token", ClientIdentifier: headers.ClientIdentifier, HTTPClient: http.Client{Transport: transportUnauth}, Headers: headers}

	_, err = plexUnauth.DeletePlexToken("test-token")
	if err == nil {
		t.Errorf("DeletePlexToken() expected error for unauthorized")
	}
}

// Test SearchPlex function from search.go
func TestPlex_SearchPlex(t *testing.T) {
	searchResponse := SearchResults{
		MediaContainer: SearchMediaContainer{
			MediaContainer: MediaContainer{
				Size: 5,
				Metadata: []Metadata{
					{Title: "Test Movie 1", Year: 2023, Type: "movie"},
					{Title: "Test Movie 2", Year: 2022, Type: "movie"},
					{Title: "Test Movie 3", Year: 2021, Type: "movie"},
					{Title: "Test Movie 4", Year: 2020, Type: "movie"},
					{Title: "Test Movie 5", Year: 2019, Type: "movie"},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/search") {
			t.Errorf("SearchPlex() path = %v", r.URL.Path)
		}
		if !strings.Contains(r.URL.RawQuery, "query=test") {
			t.Errorf("SearchPlex() query = %v", r.URL.RawQuery)
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", applicationJson)
		json.NewEncoder(w).Encode(searchResponse)
	}))
	defer server.Close()

	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}

	httpClient := http.Client{Transport: transport}
	headers := defaultHeaders()
	plex := &Plex{URL: server.URL, Token: "test-token", ClientIdentifier: headers.ClientIdentifier, HTTPClient: httpClient, Headers: headers}

	result, err := plex.SearchPlex("test")
	if err != nil {
		t.Errorf("SearchPlex() error = %v", err)
	}

	// SearchPlex should return only the first 4 results
	if len(result.MediaContainer.Metadata) != 4 {
		t.Errorf("SearchPlex() metadata count = %v, want 4", len(result.MediaContainer.Metadata))
	}

	if result.MediaContainer.Metadata[0].Title != "Test Movie 1" {
		t.Errorf("SearchPlex() title = %v, want Test Movie 1", result.MediaContainer.Metadata[0].Title)
	}

	// Test empty title
	_, err = plex.SearchPlex("")
	if err == nil {
		t.Errorf("SearchPlex() expected error for empty title")
	}

	// Test with less than 4 results
	smallResponse := SearchResults{
		MediaContainer: SearchMediaContainer{
			MediaContainer: MediaContainer{
				Size: 2,
				Metadata: []Metadata{
					{Title: "Test Movie 1", Year: 2023, Type: "movie"},
					{Title: "Test Movie 2", Year: 2022, Type: "movie"},
				},
			},
		},
	}

	serverSmall := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", applicationJson)
		json.NewEncoder(w).Encode(smallResponse)
	}))
	defer serverSmall.Close()

	transportSmall := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(serverSmall.URL)
		},
	}

	plexSmall := &Plex{URL: serverSmall.URL, Token: "test-token", ClientIdentifier: headers.ClientIdentifier, HTTPClient: http.Client{Transport: transportSmall}, Headers: headers}

	resultSmall, err := plexSmall.SearchPlex("test")
	if err != nil {
		t.Errorf("SearchPlex() error = %v", err)
	}

	// With 2 results, we should get all 2 (not limited to 4)
	if len(resultSmall.MediaContainer.Metadata) != 2 {
		t.Errorf("SearchPlex() small metadata count = %v, want 2", len(resultSmall.MediaContainer.Metadata))
	}
}

// Test RemoveInvitedFriend function - a comprehensive test for the 0% coverage function
func TestPlex_RemoveInvitedFriend_Coverage(t *testing.T) {
	tests := []struct {
		name         string
		inviteID     string
		isFriend     bool
		isServer     bool
		isHome       bool
		statusCode   int
		responseXML  string
		expectError  bool
		expectResult bool
	}{
		{
			name:         "successful removal - friend invite",
			inviteID:     "test@example.com",
			isFriend:     true,
			isServer:     false,
			isHome:       false,
			statusCode:   http.StatusOK,
			responseXML:  `<?xml version="1.0" encoding="UTF-8"?><Response><Response code="0" status="success"></Response></Response>`,
			expectError:  false,
			expectResult: true,
		},
		{
			name:         "successful removal - server invite",
			inviteID:     "server-invite-id",
			isFriend:     false,
			isServer:     true,
			isHome:       false,
			statusCode:   http.StatusOK,
			responseXML:  `<?xml version="1.0" encoding="UTF-8"?><Response><Response code="0" status="success"></Response></Response>`,
			expectError:  false,
			expectResult: true,
		},
		{
			name:         "successful removal - home invite",
			inviteID:     "home-invite-id",
			isFriend:     false,
			isServer:     false,
			isHome:       true,
			statusCode:   http.StatusOK,
			responseXML:  `<?xml version="1.0" encoding="UTF-8"?><Response><Response code="0" status="success"></Response></Response>`,
			expectError:  false,
			expectResult: true,
		},
		{
			name:         "failed removal - non-zero code",
			inviteID:     "invalid-id",
			isFriend:     true,
			isServer:     false,
			isHome:       false,
			statusCode:   http.StatusOK,
			responseXML:  `<?xml version="1.0" encoding="UTF-8"?><Response><Response code="1" status="error"></Response></Response>`,
			expectError:  false,
			expectResult: false,
		},
		{
			name:         "bad request response",
			inviteID:     "bad-request-id",
			isFriend:     true,
			isServer:     false,
			isHome:       false,
			statusCode:   http.StatusBadRequest,
			responseXML:  `<?xml version="1.0" encoding="UTF-8"?><Response><Response code="1" status="error"></Response></Response>`,
			expectError:  false,
			expectResult: false,
		},
		{
			name:        "unauthorized response",
			inviteID:    "unauthorized-id",
			isFriend:    true,
			isServer:    false,
			isHome:      false,
			statusCode:  http.StatusUnauthorized,
			expectError: true,
		},
		{
			name:        "malformed XML response",
			inviteID:    "malformed-xml-id",
			isFriend:    true,
			isServer:    false,
			isHome:      false,
			statusCode:  http.StatusOK,
			responseXML: `<?xml version="1.0" encoding="UTF-8"?><Invalid>`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodDelete {
					t.Errorf("Expected DELETE method, got %s", r.Method)
				}

				// Check URL path contains the invite ID
				if !strings.Contains(r.URL.Path, tt.inviteID) {
					t.Errorf("Expected path to contain %s, got %s", tt.inviteID, r.URL.Path)
				}

				// Check query parameters
				query := r.URL.Query()
				expectedFriend := boolToOneOrZero(tt.isFriend)
				expectedServer := boolToOneOrZero(tt.isServer)
				expectedHome := boolToOneOrZero(tt.isHome)

				if query.Get("friend") != expectedFriend {
					t.Errorf("Expected friend=%s, got %s", expectedFriend, query.Get("friend"))
				}
				if query.Get("server") != expectedServer {
					t.Errorf("Expected server=%s, got %s", expectedServer, query.Get("server"))
				}
				if query.Get("home") != expectedHome {
					t.Errorf("Expected home=%s, got %s", expectedHome, query.Get("home"))
				}

				w.WriteHeader(tt.statusCode)
				if tt.responseXML != "" {
					w.Header().Set("Content-Type", "application/xml")
					w.Write([]byte(tt.responseXML))
				}
			}))
			defer server.Close()

			// Override plexURL for testing
			originalURL := plexURL
			plexURL = server.URL
			defer func() { plexURL = originalURL }()

			plex := &Plex{Headers: defaultHeaders()}
			result, err := plex.RemoveInvitedFriend(tt.inviteID, tt.isFriend, tt.isServer, tt.isHome)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expectResult {
					t.Errorf("Expected result %v, got %v", tt.expectResult, result)
				}
			}
		})
	}
}

// Test InviteFriend function - currently at 0% coverage
func TestPlex_InviteFriend(t *testing.T) {
	tests := []struct {
		name         string
		params       InviteFriendParams
		statusCode   int
		response     interface{}
		expectError  bool
		errorMessage string
	}{
		{
			name: "successful friend invite",
			params: InviteFriendParams{
				UsernameOrEmail: "friend@example.com",
				MachineID:       "test-machine-123",
				LibraryIDs:      []int{1, 2},
				Label:           "Movies",
			},
			statusCode:  http.StatusCreated,
			response:    inviteFriendResponse{ID: 123, OwnerID: 456},
			expectError: false,
		},
		{
			name: "friend invite with empty label",
			params: InviteFriendParams{
				UsernameOrEmail: "friend@example.com",
				MachineID:       "test-machine-123",
				LibraryIDs:      []int{1},
				Label:           "",
			},
			statusCode:  http.StatusCreated,
			response:    inviteFriendResponse{ID: 789, OwnerID: 456},
			expectError: false,
		},
		{
			name: "server error",
			params: InviteFriendParams{
				UsernameOrEmail: "friend@example.com",
				MachineID:       "test-machine-123",
				LibraryIDs:      []int{1},
			},
			statusCode:   http.StatusInternalServerError,
			response:     "",
			expectError:  true,
			errorMessage: "500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST method, got %s", r.Method)
				}
				if !strings.Contains(r.URL.Path, "/api/v2/shared_servers") {
					t.Errorf("Expected /api/v2/shared_servers path, got %s", r.URL.Path)
				}

				w.WriteHeader(tt.statusCode)
				if tt.response != nil && tt.response != "" {
					json.NewEncoder(w).Encode(tt.response)
				}
			}))
			defer server.Close()

			// Override plexURL for testing
			originalURL := plexURL
			plexURL = server.URL
			defer func() { plexURL = originalURL }()

			plex := &Plex{Headers: defaultHeaders()}
			err := plex.InviteFriend(tt.params)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				if tt.errorMessage != "" && !strings.Contains(err.Error(), tt.errorMessage) {
					t.Errorf("Expected error message to contain '%s', got '%s'", tt.errorMessage, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// Test inviteFriendResponse UnmarshalJSON - currently at 0% coverage
func TestInviteFriendResponse_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		json        string
		expectError bool
	}{
		{
			name: "valid response with string numbers",
			json: `{
                "id": "123",
                "ownerId": "456",
                "invitedId": "789",
                "serverId": "321",
                "numLibraries": "2",
                "invited": {"id": "789"},
                "sharingSettings": {"allowTuners": "1"},
                "libraries": [{"id": "1", "key": "lib1"}]
            }`,
			expectError: false,
		},
		{
			name: "valid response with numeric values",
			json: `{
                "id": 123,
                "ownerId": 456,
                "invitedId": 789,
                "serverId": 321,
                "numLibraries": 2,
                "invited": {"id": 789},
                "sharingSettings": {"allowTuners": 1},
                "libraries": [{"id": 1, "key": "lib1"}]
            }`,
			expectError: false,
		},
		{
			name:        "invalid json",
			json:        `{invalid}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp inviteFriendResponse
			err := json.Unmarshal([]byte(tt.json), &resp)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// Test XML response functions that use XML instead of JSON

// Test GetFriends function
func TestPlex_GetFriends(t *testing.T) {
	xmlResponse := `<?xml version="1.0" encoding="UTF-8"?>
    <MediaContainer friendlyName="myPlex" identifier="com.plexapp.plugins.myplex" machineIdentifier="abc123" size="2">
        <User id="1" title="Friend1" email="friend1@example.com" thumb="avatar1"/>
        <User id="2" title="Friend2" email="friend2@example.com" thumb="avatar2"/>
    </MediaContainer>`

	server, plex := newXMLTestServer(200, xmlResponse)
	defer server.Close()

	// Override plexURL for testing
	originalPlexURL := plexURL
	plexURL = server.URL
	defer func() { plexURL = originalPlexURL }()

	friends, err := plex.GetFriends()
	if err != nil {
		t.Errorf("GetFriends() error = %v", err)
		return
	}

	if len(friends) != 2 {
		t.Errorf("GetFriends() friends count = %v, want 2", len(friends))
	}

	// Test unauthorized
	server401, plex401 := newXMLTestServer(401, "")
	defer server401.Close()

	plexURL = server401.URL
	_, err = plex401.GetFriends()
	if err == nil {
		t.Errorf("GetFriends() expected error for 401")
	}
}

// Test RemoveFriend function
func TestPlex_RemoveFriend(t *testing.T) {
	xmlResponse := `<?xml version="1.0" encoding="UTF-8"?>
    <Response code="0" status="Success"/>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("RemoveFriend() method = %v, want DELETE", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/api/friends/123") {
			t.Errorf("RemoveFriend() path = %v", r.URL.Path)
		}
		w.WriteHeader(200)
		w.Header().Set("Content-Type", applicationXml)
		fmt.Fprintln(w, xmlResponse)
	}))
	defer server.Close()

	// Override plexURL for testing
	originalPlexURL := plexURL
	plexURL = server.URL
	defer func() { plexURL = originalPlexURL }()

	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}

	httpClient := http.Client{Transport: transport}
	plex := &Plex{URL: server.URL, Token: "test-token", HTTPClient: httpClient, Headers: defaultHeaders()}

	result, err := plex.RemoveFriend("123")
	if err != nil {
		t.Errorf("RemoveFriend() error = %v", err)
	}

	if !result {
		t.Errorf("RemoveFriend() result = %v, want true", result)
	}

	// Test error response
	xmlErrorResponse := `<?xml version="1.0" encoding="UTF-8"?>
    <Response><Response code="1" status="Error"/></Response>`

	serverError := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", applicationXml)
		fmt.Fprintln(w, xmlErrorResponse)
	}))
	defer serverError.Close()

	transportError := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(serverError.URL)
		},
	}

	httpClientError := http.Client{Transport: transportError}
	plexError := &Plex{URL: serverError.URL, Token: "test-token", HTTPClient: httpClientError, Headers: defaultHeaders()}

	plexURL = serverError.URL
	result, err = plexError.RemoveFriend("123")
	if err != nil {
		t.Errorf("RemoveFriend() error = %v", err)
	}

	if result {
		t.Errorf("RemoveFriend() result = %v, want false for error response", result)
	}
}

// Test UpdateFriendAccess function
func TestPlex_UpdateFriendAccess(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("UpdateFriendAccess() method = %v, want PUT", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/api/friends/123") {
			t.Errorf("UpdateFriendAccess() path = %v", r.URL.Path)
		}

		// Check query parameters based on which call this is
		query := r.URL.Query()
		callCount++
		if callCount == 1 {
			// First call with explicit values
			if query.Get("allowSync") != "1" {
				t.Errorf("UpdateFriendAccess() allowSync = %v, want 1", query.Get("allowSync"))
			}
		} else if callCount == 2 {
			// Second call with defaults
			if query.Get("allowSync") != "0" {
				t.Errorf("UpdateFriendAccess() with defaults allowSync = %v, want 0", query.Get("allowSync"))
			}
		}

		w.WriteHeader(200)
	}))
	defer server.Close()

	// Override plexURL for testing
	originalPlexURL := plexURL
	plexURL = server.URL
	defer func() { plexURL = originalPlexURL }()

	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}

	httpClient := http.Client{Transport: transport}
	plex := &Plex{URL: server.URL, Token: "test-token", HTTPClient: httpClient, Headers: defaultHeaders()}

	params := UpdateFriendParams{
		AllowSync:         "1",
		AllowCameraUpload: "1",
		AllowChannels:     "1",
		FilterMovies:      "label=action",
		FilterMusic:       "",
		FilterTelevision:  "",
		FilterPhotos:      "",
	}

	result, err := plex.UpdateFriendAccess("123", params)
	if err != nil {
		t.Errorf("UpdateFriendAccess() error = %v", err)
	}

	if !result {
		t.Errorf("UpdateFriendAccess() result = %v, want true", result)
	}

	// Test with default values
	defaultParams := UpdateFriendParams{}
	_, err = plex.UpdateFriendAccess("123", defaultParams)
	if err != nil {
		t.Errorf("UpdateFriendAccess() with defaults error = %v", err)
	}

	// Test server error
	server500 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer server500.Close()

	transport500 := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server500.URL)
		},
	}

	httpClient500 := http.Client{Transport: transport500}
	plex500 := &Plex{URL: server500.URL, Token: "test-token", HTTPClient: httpClient500, Headers: defaultHeaders()}

	plexURL = server500.URL
	_, err = plex500.UpdateFriendAccess("123", params)
	if err == nil {
		t.Errorf("UpdateFriendAccess() expected error for 500")
	}
}

// Test RemoveFriendAccessToLibrary function
func TestPlex_RemoveFriendAccessToLibrary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("RemoveFriendAccessToLibrary() method = %v, want DELETE", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/api/servers/machine123/shared_servers/server456") {
			t.Errorf("RemoveFriendAccessToLibrary() path = %v", r.URL.Path)
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	// Override plexURL for testing
	originalPlexURL := plexURL
	plexURL = server.URL
	defer func() { plexURL = originalPlexURL }()

	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}

	httpClient := http.Client{Transport: transport}
	plex := &Plex{URL: server.URL, Token: "test-token", HTTPClient: httpClient, Headers: defaultHeaders()}

	result, err := plex.RemoveFriendAccessToLibrary("user123", "machine123", "server456")
	if err != nil {
		t.Errorf("RemoveFriendAccessToLibrary() error = %v", err)
	}

	if !result {
		t.Errorf("RemoveFriendAccessToLibrary() result = %v, want true", result)
	}
}

// Test GetInvitedFriends function
func TestPlex_GetInvitedFriends(t *testing.T) {
	xmlResponse := `<?xml version="1.0" encoding="UTF-8"?>
    <MediaContainer friendlyName="myPlex" identifier="com.plexapp.plugins.myplex" machineIdentifier="abc123" size="2">
        <Invite id="email1@gmail.com" createdAt="1639964970" friend="0" home="0" server="1" username="" email="email1@gmail.com" thumb="" friendlyName="email1@gmail.com">
            <Server name="Server123" numLibraries="3"/>
        </Invite>
        <Invite id="19661994" createdAt="1643379560" friend="0" home="1" server="0" username="home-user" email="home-user@gmail.com" thumb="https://plex.tv/users/abc/avatar?c=123" friendlyName="home-user"/>
    </MediaContainer>`

	server, plex := newXMLTestServer(200, xmlResponse)
	defer server.Close()

	// Override plexURL for testing
	originalPlexURL := plexURL
	plexURL = server.URL
	defer func() { plexURL = originalPlexURL }()

	invites, err := plex.GetInvitedFriends()
	if err != nil {
		t.Errorf("GetInvitedFriends() error = %v", err)
		return
	}

	if len(invites) != 2 {
		t.Errorf("GetInvitedFriends() invites count = %v, want 2", len(invites))
	}

	// Test unauthorized
	server401, plex401 := newXMLTestServer(401, "")
	defer server401.Close()

	plexURL = server401.URL
	_, err = plex401.GetInvitedFriends()
	if err == nil {
		t.Errorf("GetInvitedFriends() expected error for 401")
	}
}

// Test CheckUsernameOrEmail function
func TestPlex_CheckUsernameOrEmail(t *testing.T) {
	xmlResponse := `<?xml version="1.0" encoding="UTF-8"?>
    <Response code="0" status="Valid user"/>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("CheckUsernameOrEmail() method = %v, want POST", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/api/users/validate") {
			t.Errorf("CheckUsernameOrEmail() path = %v", r.URL.Path)
		}
		if !strings.Contains(r.URL.RawQuery, "invited_email=test%40example.com") {
			t.Errorf("CheckUsernameOrEmail() query = %v", r.URL.RawQuery)
		}
		w.WriteHeader(200)
		w.Header().Set("Content-Type", applicationXml)
		fmt.Fprintln(w, xmlResponse)
	}))
	defer server.Close()

	// Override plexURL for testing
	originalPlexURL := plexURL
	plexURL = server.URL
	defer func() { plexURL = originalPlexURL }()

	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}

	httpClient := http.Client{Transport: transport}
	plex := &Plex{URL: server.URL, Token: "test-token", HTTPClient: httpClient, Headers: defaultHeaders()}

	result, err := plex.CheckUsernameOrEmail("test@example.com")
	if err != nil {
		t.Errorf("CheckUsernameOrEmail() error = %v", err)
	}

	if !result {
		t.Errorf("CheckUsernameOrEmail() result = %v, want true", result)
	}

	// Test invalid user
	xmlInvalidResponse := `<?xml version="1.0" encoding="UTF-8"?>
    <Response><Response code="1" status="Invalid user"/></Response>`

	serverInvalid := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", applicationXml)
		fmt.Fprintln(w, xmlInvalidResponse)
	}))
	defer serverInvalid.Close()

	transportInvalid := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(serverInvalid.URL)
		},
	}

	httpClientInvalid := http.Client{Transport: transportInvalid}
	plexInvalid := &Plex{URL: serverInvalid.URL, Token: "test-token", HTTPClient: httpClientInvalid, Headers: defaultHeaders()}

	plexURL = serverInvalid.URL
	result, err = plexInvalid.CheckUsernameOrEmail("invalid@example.com")
	if err != nil {
		t.Errorf("CheckUsernameOrEmail() error = %v", err)
	}

	if result {
		t.Errorf("CheckUsernameOrEmail() result = %v, want false for invalid user", result)
	}
}

// Test StopPlayback function
func TestPlex_StopPlayback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("StopPlayback() method = %v, want GET", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/player/playback/stop") {
			t.Errorf("StopPlayback() path = %v", r.URL.Path)
		}

		// Check headers for target client identifier
		if r.Header.Get("X-Plex-Target-Identifier") != "machine123" {
			t.Errorf("StopPlayback() missing target client identifier header")
		}

		w.WriteHeader(200)
	}))
	defer server.Close()

	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}

	httpClient := http.Client{Transport: transport}
	plex := &Plex{URL: server.URL, Token: "test-token", HTTPClient: httpClient, Headers: defaultHeaders()}

	err := plex.StopPlayback("machine123")
	if err != nil {
		t.Errorf("StopPlayback() error = %v", err)
	}

	// Test server error
	server500 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer server500.Close()

	transport500 := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server500.URL)
		},
	}

	httpClient500 := http.Client{Transport: transport500}
	plex500 := &Plex{URL: server500.URL, Token: "test-token", HTTPClient: httpClient500, Headers: defaultHeaders()}

	err = plex500.StopPlayback("machine123")
	if err == nil {
		t.Errorf("StopPlayback() expected error for 500")
	}
}

// Test GetDevices function
func TestPlex_GetDevices(t *testing.T) {
	xmlResponse := `<?xml version="1.0" encoding="UTF-8"?>
    <MediaContainer size="2">
        <Device name="My Server" product="Plex Media Server" provides="server" clientIdentifier="abc123" />
        <Device name="My Player" product="Plex for Android" provides="player" clientIdentifier="def456" />
    </MediaContainer>`

	server, plex := newXMLTestServer(200, xmlResponse)
	defer server.Close()

	// Override plexURL for testing
	originalPlexURL := plexURL
	plexURL = server.URL
	defer func() { plexURL = originalPlexURL }()

	devices, err := plex.GetDevices()
	if err != nil {
		t.Errorf("GetDevices() error = %v", err)
		return
	}

	if len(devices) != 2 {
		t.Errorf("GetDevices() devices count = %v, want 2", len(devices))
	}

	// Test server error
	server500, plex500 := newXMLTestServer(500, "")
	defer server500.Close()

	plexURL = server500.URL
	_, err = plex500.GetDevices()
	if err == nil {
		t.Errorf("GetDevices() expected error for 500")
	}
}

// Test GetServers function
func TestPlex_GetServers(t *testing.T) {
	xmlResponse := `<?xml version="1.0" encoding="UTF-8"?>
    <MediaContainer size="3">
        <Device name="My Server" product="Plex Media Server" provides="server" clientIdentifier="abc123" />
        <Device name="My Player" product="Plex for Android" provides="player" clientIdentifier="def456" />
        <Device name="Another Server" product="Plex Media Server" provides="server" clientIdentifier="ghi789" />
    </MediaContainer>`

	server, plex := newXMLTestServer(200, xmlResponse)
	defer server.Close()

	// Override plexURL for testing
	originalPlexURL := plexURL
	plexURL = server.URL
	defer func() { plexURL = originalPlexURL }()

	servers, err := plex.GetServers()
	if err != nil {
		t.Errorf("GetServers() error = %v", err)
		return
	}

	// Should only return servers, not players
	if len(servers) != 2 {
		t.Errorf("GetServers() servers count = %v, want 2", len(servers))
	}
}

// Test GetSections function
func TestPlex_GetSections(t *testing.T) {
	xmlResponse := `<?xml version="1.0" encoding="UTF-8"?>
    <MediaContainer friendlyName="myPlex" identifier="com.plexapp.plugins.myplex" machineIdentifier="abc123" size="1">
        <Server name="My Server" machineIdentifier="target123">
            <Section id="1" key="1" type="movie" title="Movies"/>
            <Section id="2" key="2" type="show" title="TV Shows"/>
        </Server>
        <Server name="Other Server" machineIdentifier="other456">
            <Section id="3" key="3" type="artist" title="Music"/>
        </Server>
    </MediaContainer>`

	server, plex := newXMLTestServer(200, xmlResponse)
	defer server.Close()

	// Override plexURL for testing
	originalPlexURL := plexURL
	plexURL = server.URL
	defer func() { plexURL = originalPlexURL }()

	sections, err := plex.GetSections("target123")
	if err != nil {
		t.Errorf("GetSections() error = %v", err)
		return
	}

	if len(sections) != 2 {
		t.Errorf("GetSections() sections count = %v, want 2", len(sections))
	}

	// Test machine ID not found
	sections, err = plex.GetSections("notfound")
	if err != nil {
		t.Errorf("GetSections() error = %v", err)
	}

	if len(sections) != 0 {
		t.Errorf("GetSections() sections count = %v, want 0 for not found", len(sections))
	}
}

// Test GetServersInfo function with XML server
func TestPlex_GetServersInfo_XML(t *testing.T) {
	xmlResponse := `<?xml version="1.0" encoding="UTF-8"?>
    <MediaContainer friendlyName="myPlex" machineIdentifier="main123" size="2">
        <Server name="Server1" host="192.168.1.100" machineIdentifier="server1" accessToken="token1" owned="1"/>
        <Server name="Server2" host="192.168.1.101" machineIdentifier="server2" accessToken="token2" owned="0"/>
    </MediaContainer>`

	server, plex := newXMLTestServer(200, xmlResponse)
	defer server.Close()

	// Override plexURL for testing
	originalPlexURL := plexURL
	plexURL = server.URL
	defer func() { plexURL = originalPlexURL }()

	info, err := plex.GetServersInfo()
	if err != nil {
		t.Errorf("GetServersInfo() error = %v", err)
		return
	}

	if len(info.Server) != 2 {
		t.Errorf("GetServersInfo() servers count = %v, want 2", len(info.Server))
	}

	if info.FriendlyName != "myPlex" {
		t.Errorf("GetServersInfo() friendly name = %v, want myPlex", info.FriendlyName)
	}
}

// Test GetMachineID function
func TestPlex_GetMachineID(t *testing.T) {
	xmlResponse := `<?xml version="1.0" encoding="UTF-8"?>
    <MediaContainer friendlyName="myPlex" machineIdentifier="main123" size="2">
        <Server name="Server1" host="192.168.1.100" machineIdentifier="server1" accessToken="wrong-token" owned="1"/>
        <Server name="Server2" host="192.168.1.101" machineIdentifier="server2" accessToken="test-token" owned="1"/>
    </MediaContainer>`

	server, plex := newXMLTestServer(200, xmlResponse)
	defer server.Close()

	// Override plexURL for testing
	originalPlexURL := plexURL
	plexURL = server.URL
	defer func() { plexURL = originalPlexURL }()

	plex.Token = "test-token"

	machineID, err := plex.GetMachineID()
	if err != nil {
		t.Errorf("GetMachineID() error = %v", err)
		return
	}

	if machineID != "server2" {
		t.Errorf("GetMachineID() machine ID = %v, want server2", machineID)
	}

	// Test no token
	plex.Token = ""
	_, err = plex.GetMachineID()
	if err == nil {
		t.Errorf("GetMachineID() expected error for no token")
	}

	// Test token not found
	plex.Token = "not-found-token"
	_, err = plex.GetMachineID()
	if err == nil {
		t.Errorf("GetMachineID() expected error for token not found")
	}
}

func TestFlexibleIntUnmarshaling(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		expected int64
	}{
		{
			name:     "String number",
			jsonData: `{"librarySectionID": "123"}`,
			expected: 123,
		},
		{
			name:     "Integer number",
			jsonData: `{"librarySectionID": 456}`,
			expected: 456,
		},
		{
			name:     "Empty string",
			jsonData: `{"librarySectionID": ""}`,
			expected: 0,
		},
		{
			name:     "Invalid string",
			jsonData: `{"librarySectionID": "abc"}`,
			expected: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var metadata Metadata
			if err := json.Unmarshal([]byte(test.jsonData), &metadata); err != nil {
				t.Errorf("Failed to unmarshal JSON: %v", err)
				return
			}

			actual := metadata.LibrarySectionID.Int64()

			if actual != test.expected {
				t.Errorf("Expected %d, got %d", test.expected, actual)
			}
		})
	}
}

func TestSubtitleDecisionInTranscodeSession(t *testing.T) {
	jsonData := `{
		"audioChannels": 2,
		"audioCodec": "aac",
		"audioDecision": "transcode",
		"complete": false,
		"container": "mkv",
		"context": "streaming",
		"duration": 7200000,
		"key": "transcode/session/abc123",
		"progress": 25.5,
		"protocol": "http",
		"remaining": 5400000,
		"sourceAudioCodec": "ac3",
		"sourceVideoCodec": "h264",
		"speed": 1.0,
		"subtitleDecision": "burn",
		"throttled": false,
		"transcodeHwRequested": true,
		"videoCodec": "h264",
		"videoDecision": "copy"
	}`

	var session TranscodeSession
	if err := json.Unmarshal([]byte(jsonData), &session); err != nil {
		t.Errorf("Failed to unmarshal TranscodeSession: %v", err)
		return
	}

	if session.SubtitleDecision != "burn" {
		t.Errorf("Expected SubtitleDecision to be 'burn', got '%s'", session.SubtitleDecision)
	}

	if session.AudioDecision != "transcode" {
		t.Errorf("Expected AudioDecision to be 'transcode', got '%s'", session.AudioDecision)
	}

	if session.VideoDecision != "copy" {
		t.Errorf("Expected VideoDecision to be 'copy', got '%s'", session.VideoDecision)
	}
}

func TestTimelineEventHandler(t *testing.T) {
	events := NewNotificationEvents()

	// Test that we can set a timeline handler
	events.OnTimeline(func(n NotificationContainer) {
		// Handler logic would go here
	})

	// Verify the handler was set by checking if the events map contains it
	if events.events["timeline"] == nil {
		t.Error("Timeline event handler was not set")
	}
}
