package plex

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

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
	result, err = plex.UpdateFriendAccess("123", defaultParams)
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
