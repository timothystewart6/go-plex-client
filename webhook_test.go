package plex

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Test Handler function
func TestWebhookEvents_Handler(t *testing.T) {
	webhook := Webhook{
		Event: "media.play",
		User:  true,
		Owner: false,
		Account: struct {
			ID    int    `json:"id"`
			Thumb string `json:"thumb"`
			Title string `json:"title"`
		}{
			ID:    123,
			Thumb: "thumb.jpg",
			Title: "Test User",
		},
		Server: struct {
			Title string `json:"title"`
			UUID  string `json:"uuid"`
		}{
			Title: "Test Server",
			UUID:  "server-uuid",
		},
		Player: struct {
			Local         bool   `json:"local"`
			PublicAddress string `json:"PublicAddress"`
			Title         string `json:"title"`
			UUID          string `json:"uuid"`
		}{
			Local:         true,
			PublicAddress: "192.168.1.100",
			Title:         "Test Player",
			UUID:          "player-uuid",
		},
		Metadata: struct {
			LibrarySectionType   string `json:"librarySectionType"`
			RatingKey            string `json:"ratingKey"`
			Key                  string `json:"key"`
			ParentRatingKey      string `json:"parentRatingKey"`
			GrandparentRatingKey string `json:"grandparentRatingKey"`
			GUID                 string `json:"guid"`
			LibrarySectionID     int    `json:"librarySectionID"`
			MediaType            string `json:"type"`
			Title                string `json:"title"`
			GrandparentKey       string `json:"grandparentKey"`
			ParentKey            string `json:"parentKey"`
			GrandparentTitle     string `json:"grandparentTitle"`
			ParentTitle          string `json:"parentTitle"`
			Summary              string `json:"summary"`
			Index                int    `json:"index"`
			ParentIndex          int    `json:"parentIndex"`
			RatingCount          int    `json:"ratingCount"`
			Thumb                string `json:"thumb"`
			Art                  string `json:"art"`
			ParentThumb          string `json:"parentThumb"`
			GrandparentThumb     string `json:"grandparentThumb"`
			GrandparentArt       string `json:"grandparentArt"`
			AddedAt              int    `json:"addedAt"`
			UpdatedAt            int    `json:"updatedAt"`
		}{
			LibrarySectionType: "movie",
			RatingKey:          "123",
			Key:                "/library/metadata/123",
			MediaType:          "movie",
			Title:              "Test Movie",
			Summary:            "A test movie",
		},
	}

	tests := []struct {
		name        string
		setupFunc   func(wh *WebhookEvents) bool // returns true if function was called
		payload     interface{}
		eventType   string
		expectCall  bool
		expectError bool
	}{
		{
			name: "successful play event",
			setupFunc: func(wh *WebhookEvents) bool {
				called := false
				wh.OnPlay(func(w Webhook) {
					called = true
				})
				return called
			},
			payload:    webhook,
			eventType:  "media.play",
			expectCall: true,
		},
		{
			name: "successful pause event",
			setupFunc: func(wh *WebhookEvents) bool {
				called := false
				wh.OnPause(func(w Webhook) {
					called = true
				})
				return called
			},
			payload:    Webhook{Event: "media.pause", User: true},
			eventType:  "media.pause",
			expectCall: true,
		},
		{
			name: "unknown event",
			setupFunc: func(wh *WebhookEvents) bool {
				return false
			},
			payload:   Webhook{Event: "unknown.event", User: true},
			eventType: "", // No event type so no handler is set up
		},
		{
			name: "invalid JSON payload",
			setupFunc: func(wh *WebhookEvents) bool {
				return false
			},
			payload: `{"invalid": json}`,
		},
		{
			name: "no payload",
			setupFunc: func(wh *WebhookEvents) bool {
				return false
			},
			payload: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wh := NewWebhook()
			var functionCalled bool

			if tt.setupFunc != nil {
				tt.setupFunc(wh)

				// After setup, override the handler to capture if it was called
				if tt.eventType != "" {
					wh.events[tt.eventType] = func(w Webhook) {
						functionCalled = true
					}
				}
			}

			// Create multipart form data
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)

			if tt.payload != nil {
				var payloadBytes []byte
				var err error

				if webhookData, ok := tt.payload.(Webhook); ok {
					payloadBytes, err = json.Marshal(webhookData)
					if err != nil {
						t.Fatalf("Failed to marshal webhook: %v", err)
					}
				} else if str, ok := tt.payload.(string); ok {
					payloadBytes = []byte(str)
				}

				if len(payloadBytes) > 0 {
					writer.WriteField("payload", string(payloadBytes))
				}
			}

			writer.Close()

			req := httptest.NewRequest(http.MethodPost, "/webhook", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			w := httptest.NewRecorder()

			wh.Handler(w, req)

			if tt.expectCall && !functionCalled {
				t.Errorf("Expected webhook function to be called but it wasn't")
			} else if !tt.expectCall && functionCalled {
				t.Errorf("Expected webhook function not to be called but it was")
			}
		})
	}
}

// Test form parsing error
func TestWebhookEvents_Handler_FormParseError(t *testing.T) {
	wh := NewWebhook()

	// Create a request with wrong content type (should be multipart/form-data)
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader("invalid form data"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()

	// Capture the error output (this will print an error message)
	wh.Handler(w, req)

	// The handler should handle the error gracefully and not panic
}

// Test newWebhookEvent function
func TestWebhookEvents_newWebhookEvent(t *testing.T) {
	wh := NewWebhook()

	tests := []struct {
		name        string
		eventName   string
		expectError bool
	}{
		{
			name:        "valid play event",
			eventName:   "media.play",
			expectError: false,
		},
		{
			name:        "valid pause event",
			eventName:   "media.pause",
			expectError: false,
		},
		{
			name:        "valid resume event",
			eventName:   "media.resume",
			expectError: false,
		},
		{
			name:        "valid stop event",
			eventName:   "media.stop",
			expectError: false,
		},
		{
			name:        "valid scrobble event",
			eventName:   "media.scrobble",
			expectError: false,
		},
		{
			name:        "valid rate event",
			eventName:   "media.rate",
			expectError: false,
		},
		{
			name:        "invalid event",
			eventName:   "invalid.event",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			testFunc := func(w Webhook) {
				called = true
			}

			err := wh.newWebhookEvent(tt.eventName, testFunc)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				if err.Error() != "invalid event name" {
					t.Errorf("Expected 'invalid event name' error, got '%s'", err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				// Test that the function was properly set
				if fn, exists := wh.events[tt.eventName]; exists {
					fn(Webhook{})
					if !called {
						t.Errorf("Function was not called when event was triggered")
					}
				} else {
					t.Errorf("Event function was not set in events map")
				}
			}
		})
	}
}

// Test NewWebhook function
func TestNewWebhook(t *testing.T) {
	wh := NewWebhook()

	if wh == nil {
		t.Fatal("NewWebhook returned nil")
	}

	if wh.events == nil {
		t.Fatal("events map was not initialized")
	}

	expectedEvents := []string{
		"media.play",
		"media.pause",
		"media.resume",
		"media.stop",
		"media.scrobble",
		"media.rate",
	}

	for _, event := range expectedEvents {
		if _, exists := wh.events[event]; !exists {
			t.Errorf("Expected event '%s' not found in events map", event)
		}
	}

	// Test that all default functions are no-ops
	for event, fn := range wh.events {
		// Should not panic when called
		fn(Webhook{})
		t.Logf("Default function for %s executed without panic", event)
	}
}

// Test OnPlay function
func TestWebhookEvents_OnPlay(t *testing.T) {
	wh := NewWebhook()

	called := false
	testFunc := func(w Webhook) {
		called = true
	}

	err := wh.OnPlay(testFunc)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Trigger the event
	if fn, exists := wh.events["media.play"]; exists {
		fn(Webhook{})
		if !called {
			t.Errorf("OnPlay function was not called")
		}
	} else {
		t.Errorf("media.play event not found")
	}
}

// Test OnPause function
func TestWebhookEvents_OnPause(t *testing.T) {
	wh := NewWebhook()

	called := false
	testFunc := func(w Webhook) {
		called = true
	}

	err := wh.OnPause(testFunc)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Trigger the event
	if fn, exists := wh.events["media.pause"]; exists {
		fn(Webhook{})
		if !called {
			t.Errorf("OnPause function was not called")
		}
	} else {
		t.Errorf("media.pause event not found")
	}
}

// Test OnResume function
func TestWebhookEvents_OnResume(t *testing.T) {
	wh := NewWebhook()

	called := false
	testFunc := func(w Webhook) {
		called = true
	}

	err := wh.OnResume(testFunc)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Trigger the event
	if fn, exists := wh.events["media.resume"]; exists {
		fn(Webhook{})
		if !called {
			t.Errorf("OnResume function was not called")
		}
	} else {
		t.Errorf("media.resume event not found")
	}
}

// Test OnStop function
func TestWebhookEvents_OnStop(t *testing.T) {
	wh := NewWebhook()

	called := false
	testFunc := func(w Webhook) {
		called = true
	}

	err := wh.OnStop(testFunc)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Trigger the event
	if fn, exists := wh.events["media.stop"]; exists {
		fn(Webhook{})
		if !called {
			t.Errorf("OnStop function was not called")
		}
	} else {
		t.Errorf("media.stop event not found")
	}
}

// Test OnScrobble function
func TestWebhookEvents_OnScrobble(t *testing.T) {
	wh := NewWebhook()

	called := false
	testFunc := func(w Webhook) {
		called = true
	}

	err := wh.OnScrobble(testFunc)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Trigger the event
	if fn, exists := wh.events["media.scrobble"]; exists {
		fn(Webhook{})
		if !called {
			t.Errorf("OnScrobble function was not called")
		}
	} else {
		t.Errorf("media.scrobble event not found")
	}
}

// Test OnRate function
func TestWebhookEvents_OnRate(t *testing.T) {
	wh := NewWebhook()

	called := false
	testFunc := func(w Webhook) {
		called = true
	}

	err := wh.OnRate(testFunc)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Trigger the event
	if fn, exists := wh.events["media.rate"]; exists {
		fn(Webhook{})
		if !called {
			t.Errorf("OnRate function was not called")
		}
	} else {
		t.Errorf("media.rate event not found")
	}
}

// Test complete webhook flow
func TestWebhookEvents_CompleteFlow(t *testing.T) {
	wh := NewWebhook()

	// Set up event handlers
	var playEventReceived Webhook
	var pauseEventReceived Webhook

	wh.OnPlay(func(w Webhook) {
		playEventReceived = w
	})

	wh.OnPause(func(w Webhook) {
		pauseEventReceived = w
	})

	// Create test webhooks
	playWebhook := Webhook{
		Event: "media.play",
		User:  true,
		Account: struct {
			ID    int    `json:"id"`
			Thumb string `json:"thumb"`
			Title string `json:"title"`
		}{
			ID:    123,
			Title: "Test User",
		},
		Metadata: struct {
			LibrarySectionType   string `json:"librarySectionType"`
			RatingKey            string `json:"ratingKey"`
			Key                  string `json:"key"`
			ParentRatingKey      string `json:"parentRatingKey"`
			GrandparentRatingKey string `json:"grandparentRatingKey"`
			GUID                 string `json:"guid"`
			LibrarySectionID     int    `json:"librarySectionID"`
			MediaType            string `json:"type"`
			Title                string `json:"title"`
			GrandparentKey       string `json:"grandparentKey"`
			ParentKey            string `json:"parentKey"`
			GrandparentTitle     string `json:"grandparentTitle"`
			ParentTitle          string `json:"parentTitle"`
			Summary              string `json:"summary"`
			Index                int    `json:"index"`
			ParentIndex          int    `json:"parentIndex"`
			RatingCount          int    `json:"ratingCount"`
			Thumb                string `json:"thumb"`
			Art                  string `json:"art"`
			ParentThumb          string `json:"parentThumb"`
			GrandparentThumb     string `json:"grandparentThumb"`
			GrandparentArt       string `json:"grandparentArt"`
			AddedAt              int    `json:"addedAt"`
			UpdatedAt            int    `json:"updatedAt"`
		}{
			Title:     "Test Movie",
			MediaType: "movie",
		},
	}

	pauseWebhook := Webhook{
		Event: "media.pause",
		User:  true,
		Account: struct {
			ID    int    `json:"id"`
			Thumb string `json:"thumb"`
			Title string `json:"title"`
		}{
			ID:    456,
			Title: "Another User",
		},
	}

	// Test play event
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	playloadBytes, _ := json.Marshal(playWebhook)
	writer.WriteField("payload", string(playloadBytes))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/webhook", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	wh.Handler(w, req)

	if playEventReceived.Event != "media.play" {
		t.Errorf("Expected play event, got %s", playEventReceived.Event)
	}
	if playEventReceived.Account.ID != 123 {
		t.Errorf("Expected account ID 123, got %d", playEventReceived.Account.ID)
	}

	// Test pause event
	body = &bytes.Buffer{}
	writer = multipart.NewWriter(body)
	pausePayloadBytes, _ := json.Marshal(pauseWebhook)
	writer.WriteField("payload", string(pausePayloadBytes))
	writer.Close()

	req = httptest.NewRequest(http.MethodPost, "/webhook", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w = httptest.NewRecorder()

	wh.Handler(w, req)

	if pauseEventReceived.Event != "media.pause" {
		t.Errorf("Expected pause event, got %s", pauseEventReceived.Event)
	}
	if pauseEventReceived.Account.ID != 456 {
		t.Errorf("Expected account ID 456, got %d", pauseEventReceived.Account.ID)
	}
}
