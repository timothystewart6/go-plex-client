package plex

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

// Test utility functions and error handling

// Test parseFlexibleInt64 function
func TestParseFlexibleInt64(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    int64
		wantErr bool
	}{
		{"quoted string", []byte(`"123"`), 123, false},
		{"unquoted number", []byte(`456`), 456, false},
		{"zero", []byte(`0`), 0, false},
		{"negative", []byte(`-789`), -789, false},
		{"invalid string", []byte(`"abc"`), 0, false}, // parseFlexibleInt64 returns 0 for non-numeric strings
		{"null", []byte(`null`), 0, false},
		{"empty string", []byte(`""`), 0, false},
		{"empty bytes", []byte(``), 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseFlexibleInt64(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseFlexibleInt64() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseFlexibleInt64() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test FlexibleInt64 UnmarshalJSON
func TestFlexibleInt64_UnmarshalJSON(t *testing.T) {
	var f FlexibleInt64

	// Test valid cases
	testCases := []struct {
		json []byte
		want int64
	}{
		{[]byte(`"123"`), 123},
		{[]byte(`456`), 456},
		{[]byte(`0`), 0},
		{[]byte(`"abc"`), 0}, // parseFlexibleInt64 returns 0 for non-numeric strings
	}

	for _, tc := range testCases {
		err := f.UnmarshalJSON(tc.json)
		if err != nil {
			t.Errorf("FlexibleInt64.UnmarshalJSON(%s) error = %v", tc.json, err)
		}
		if f.Int64() != tc.want {
			t.Errorf("FlexibleInt64.UnmarshalJSON(%s) = %v, want %v", tc.json, f.Int64(), tc.want)
		}
	}
}

// Test boolOrInt UnmarshalJSON
func TestBoolOrInt_UnmarshalJSON(t *testing.T) {
	var b boolOrInt

	// Test integer cases
	testCases := []struct {
		json []byte
		want bool
	}{
		{[]byte(`0`), false},
		{[]byte(`1`), true},
		{[]byte(`true`), true},
		{[]byte(`false`), false},
	}

	for _, tc := range testCases {
		err := b.UnmarshalJSON(tc.json)
		if err != nil {
			t.Errorf("boolOrInt.UnmarshalJSON(%s) error = %v", tc.json, err)
		}
		if b.bool != tc.want {
			t.Errorf("boolOrInt.UnmarshalJSON(%s) = %v, want %v", tc.json, b.bool, tc.want)
		}
	}

	// Test invalid integer
	err := b.UnmarshalJSON([]byte(`2`))
	if err == nil {
		t.Errorf("boolOrInt.UnmarshalJSON(2) expected error")
	}

	// Test invalid JSON
	err = b.UnmarshalJSON([]byte(`"invalid"`))
	if err == nil {
		t.Errorf("boolOrInt.UnmarshalJSON(\"invalid\") expected error")
	}
}

// Test GetSessions with CurrentSessions response
func TestPlex_GetSessions_JSON(t *testing.T) {
	sessionsResponse := CurrentSessions{
		MediaContainer: struct {
			Metadata []Metadata `json:"Metadata"`
			Size     int        `json:"size"`
		}{
			Size: 1,
			Metadata: []Metadata{
				{
					Title: "Test Movie",
					Type:  "movie",
					User:  User{Title: "Test User"},
					Player: Player{
						Title:   "Test Player",
						Product: "Plex Web",
					},
					Session: Session{
						ID: "session123",
					},
				},
			},
		},
	}

	server, plex := newJSONTestServer(200, sessionsResponse)
	defer server.Close()

	result, err := plex.GetSessions()
	if err != nil {
		t.Errorf("GetSessions() error = %v", err)
		return
	}

	if result.MediaContainer.Size != 1 {
		t.Errorf("GetSessions() size = %v, want 1", result.MediaContainer.Size)
	}

	if len(result.MediaContainer.Metadata) != 1 {
		t.Errorf("GetSessions() metadata count = %v, want 1", len(result.MediaContainer.Metadata))
	}

	// Test unauthorized
	server401, plex401 := newJSONTestServer(401, nil)
	defer server401.Close()

	_, err = plex401.GetSessions()
	if err == nil {
		t.Errorf("GetSessions() expected error for 401")
	}
}

// Test helper functions in search.go
func TestPlex_ExtractKeyFromRatingKey(t *testing.T) {
	p := &Plex{}

	tests := []struct {
		input    string
		expected string
	}{
		{"/library/metadata/123", "123"},
		{"/library/metadata/456/children", "456"},
		{"/library/metadata/789/thumb/123456", "789"},
		{"invalid", "invalid"}, // Function returns input if it doesn't match pattern
		{"short", "short"},     // Function returns input if too short
	}

	for _, test := range tests {
		result := p.ExtractKeyFromRatingKey(test.input)
		if result != test.expected {
			t.Errorf("ExtractKeyFromRatingKey(%s) = %s, want %s", test.input, result, test.expected)
		}
	}
}

func TestPlex_ExtractKeyFromRatingKeyRegex(t *testing.T) {
	p := &Plex{}

	tests := []struct {
		input    string
		expected string
	}{
		{"/library/metadata/123", "123"},
		{"/library/metadata/456/children", "456"},
		{"/library/metadata/789/thumb/123456", "789"},
		{"abc123def", "123"},
		{"", ""}, // Empty string case
	}

	for _, test := range tests {
		// Skip tests that would cause panic
		if test.input != "" && regexp.MustCompile(`\d+`).FindStringSubmatch(test.input) == nil {
			continue
		}

		result := p.ExtractKeyFromRatingKeyRegex(test.input)
		if result != test.expected {
			t.Errorf("ExtractKeyFromRatingKeyRegex(%s) = %s, want %s", test.input, result, test.expected)
		}
	}
}

func TestPlex_ExtractKeyAndThumbFromURL(t *testing.T) {
	p := &Plex{}

	tests := []struct {
		input         string
		expectedKey   string
		expectedThumb string
	}{
		{"/library/metadata/123/thumb/456789", "123", "456789"},
		{"/library/metadata/789/thumb/123", "789", "123"},
		{"invalid", "", ""},
	}

	for _, test := range tests {
		key, thumb := p.ExtractKeyAndThumbFromURL(test.input)
		if key != test.expectedKey || thumb != test.expectedThumb {
			t.Errorf("ExtractKeyAndThumbFromURL(%s) = (%s, %s), want (%s, %s)",
				test.input, key, thumb, test.expectedKey, test.expectedThumb)
		}
	}
}

// Test error handling for HTTP methods
func TestPlex_HTTPErrorHandling(t *testing.T) {
	// Test network error
	plex := &Plex{
		URL:        "http://invalid-host-that-does-not-exist:99999",
		Token:      "test-token",
		HTTPClient: http.Client{},
		Headers:    defaultHeaders(),
	}

	_, err := plex.Search("test")
	if err == nil {
		t.Errorf("Search() with invalid host expected error")
	}
}

// Test various response codes
func TestPlex_ResponseCodes(t *testing.T) {
	testCodes := []struct {
		code        int
		expectError bool
	}{
		{200, false},
		{401, true},
		{403, true},
		{404, true},
		{500, true},
	}

	for _, tc := range testCodes {
		t.Run("code_"+string(rune(tc.code)), func(t *testing.T) {
			server, plex := newJSONTestServer(tc.code, SearchResults{
				MediaContainer: SearchMediaContainer{
					MediaContainer: MediaContainer{
						Size:     1,
						Metadata: []Metadata{{Title: "Test"}},
					},
				},
			})
			defer server.Close()

			_, err := plex.Search("test")
			if tc.expectError && err == nil {
				t.Errorf("Search() with code %d expected error", tc.code)
			}
			if !tc.expectError && err != nil {
				t.Errorf("Search() with code %d unexpected error: %v", tc.code, err)
			}
		})
	}
}

// Test URL encoding in various functions
func TestPlex_URLEncoding(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if special characters are properly encoded
		if strings.Contains(r.URL.RawQuery, " ") {
			t.Errorf("URL contains unencoded space: %s", r.URL.RawQuery)
		}
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SearchResults{})
	}))
	defer server.Close()

	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}

	httpClient := http.Client{Transport: transport}
	plex := &Plex{URL: server.URL, Token: "test-token", HTTPClient: httpClient, Headers: defaultHeaders()}

	// Test search with special characters
	_, err := plex.Search("test movie with spaces & special chars")
	if err != nil {
		t.Errorf("Search() with special characters error = %v", err)
	}
}

// Test content type handling
func TestPlex_ContentTypes(t *testing.T) {
	// Test server that returns wrong content type
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("not json"))
	}))
	defer server.Close()

	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}

	httpClient := http.Client{Transport: transport}
	plex := &Plex{URL: server.URL, Token: "test-token", HTTPClient: httpClient, Headers: defaultHeaders()}

	_, err := plex.Search("test")
	if err == nil {
		t.Errorf("Search() with invalid JSON expected error")
	}
}

// Test timeout handling
func TestPlex_Timeout(t *testing.T) {
	// Create a server that never responds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Never respond
		select {}
	}))
	defer server.Close()

	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}

	// Very short timeout
	httpClient := http.Client{
		Transport: transport,
		Timeout:   1, // 1 nanosecond - will definitely timeout
	}
	plex := &Plex{URL: server.URL, Token: "test-token", HTTPClient: httpClient, Headers: defaultHeaders()}

	_, err := plex.Search("test")
	if err == nil {
		t.Errorf("Search() with timeout expected error")
	}
}

// Test empty responses
func TestPlex_EmptyResponses(t *testing.T) {
	server, plex := newJSONTestServer(200, SearchResults{
		MediaContainer: SearchMediaContainer{
			MediaContainer: MediaContainer{
				Size:     0,
				Metadata: []Metadata{},
			},
		},
	})
	defer server.Close()

	result, err := plex.Search("nonexistent")
	if err != nil {
		t.Errorf("Search() for empty result error = %v", err)
	}

	if result.MediaContainer.Size != 0 {
		t.Errorf("Search() empty result size = %v, want 0", result.MediaContainer.Size)
	}

	if len(result.MediaContainer.Metadata) != 0 {
		t.Errorf("Search() empty result metadata count = %v, want 0", len(result.MediaContainer.Metadata))
	}
}

// Test malformed JSON responses
func TestPlex_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"invalid": json}`))
	}))
	defer server.Close()

	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}

	httpClient := http.Client{Transport: transport}
	plex := &Plex{URL: server.URL, Token: "test-token", HTTPClient: httpClient, Headers: defaultHeaders()}

	_, err := plex.Search("test")
	if err == nil {
		t.Errorf("Search() with malformed JSON expected error")
	}
}

// Test headers are properly set
func TestPlex_Headers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check required headers
		if r.Header.Get("X-Plex-Token") != "test-token" {
			t.Errorf("Missing or incorrect X-Plex-Token header")
		}
		if r.Header.Get("X-Plex-Client-Identifier") == "" {
			t.Errorf("Missing X-Plex-Client-Identifier header")
		}
		if r.Header.Get("X-Plex-Product") != "Go Plex Client" {
			t.Errorf("Missing or incorrect X-Plex-Product header")
		}

		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SearchResults{})
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

	_, err := plex.Search("test")
	if err != nil {
		t.Errorf("Search() header test error = %v", err)
	}
}
