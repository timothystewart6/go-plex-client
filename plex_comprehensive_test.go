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
