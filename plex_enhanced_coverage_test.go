package plex

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

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
