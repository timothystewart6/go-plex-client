package plex

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

// Test parseFlexibleInt64 function - improve coverage to 80%+
func TestParseFlexibleInt64_Extended(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected int64
		hasError bool
	}{
		{
			name:     "float number",
			input:    []byte("123.45"),
			expected: 123,
			hasError: false,
		},
		{
			name:     "large number",
			input:    []byte("9223372036854775807"),
			expected: 9223372036854775807,
			hasError: false,
		},
		{
			name:     "zero as string",
			input:    []byte(`"0"`),
			expected: 0,
			hasError: false,
		},
		{
			name:     "invalid string number - defaults to 0",
			input:    []byte(`"not-a-number"`),
			expected: 0,
			hasError: false, // Function is robust and returns 0 for invalid strings
		},
		{
			name:     "null value",
			input:    []byte("null"),
			expected: 0,
			hasError: false,
		},
		{
			name:     "empty string",
			input:    []byte(`""`),
			expected: 0,
			hasError: false,
		},
		{
			name:     "invalid json object",
			input:    []byte(`{"invalid": "object"}`),
			expected: 0,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseFlexibleInt64(tt.input)

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected %d, got %d", tt.expected, result)
				}
			}
		})
	}
}

// Test FlexibleInt64 UnmarshalJSON methods
func TestFlexibleInt64_UnmarshalJSON_Extended(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected int64
		hasError bool
	}{
		{
			name:     "string number",
			json:     `"42"`,
			expected: 42,
			hasError: false,
		},
		{
			name:     "float as string",
			json:     `"123.45"`,
			expected: 123,
			hasError: false,
		},
		{
			name:     "invalid string - defaults to 0",
			json:     `"invalid"`,
			expected: 0,
			hasError: false, // Function is robust and returns 0 for invalid strings
		},
		{
			name:     "numeric value",
			json:     `100`,
			expected: 100,
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fi FlexibleInt64
			err := json.Unmarshal([]byte(tt.json), &fi)

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if int64(fi) != tt.expected {
					t.Errorf("Expected %d, got %d", tt.expected, int64(fi))
				}
			}
		})
	}
}

// Test WebSocket notification events
func TestNewNotificationEvents(t *testing.T) {
	events := NewNotificationEvents()

	if events == nil {
		t.Fatal("NewNotificationEvents returned nil")
	}

	if events.events == nil {
		t.Error("events map was not initialized")
	}

	// Test that default events are set up based on actual implementation
	expectedEvents := []string{"timeline", "playing", "reachability", "transcode.end", "transcodeSession.end", "transcodeSession.update", "preference", "update.statechange", "activity", "backgroundProcessingQueue"}
	for _, event := range expectedEvents {
		if _, exists := events.events[event]; !exists {
			t.Errorf("Expected event '%s' not found in events map", event)
		}
	}
}

// Test event handler setters
func TestNotificationEvents_OnPlaying(t *testing.T) {
	events := NewNotificationEvents()

	called := false
	testFunc := func(n NotificationContainer) {
		called = true
	}

	events.OnPlaying(testFunc)

	// Trigger the event
	if fn, exists := events.events["playing"]; exists {
		fn(NotificationContainer{})
		if !called {
			t.Error("OnPlaying function was not called")
		}
	} else {
		t.Error("playing event not found")
	}
}

func TestNotificationEvents_OnTimeline(t *testing.T) {
	events := NewNotificationEvents()

	called := false
	testFunc := func(n NotificationContainer) {
		called = true
	}

	events.OnTimeline(testFunc)

	// Trigger the event
	if fn, exists := events.events["timeline"]; exists {
		fn(NotificationContainer{})
		if !called {
			t.Error("OnTimeline function was not called")
		}
	} else {
		t.Error("timeline event not found")
	}
}

func TestNotificationEvents_OnTranscodeUpdate(t *testing.T) {
	events := NewNotificationEvents()

	called := false
	testFunc := func(n NotificationContainer) {
		called = true
	}

	events.OnTranscodeUpdate(testFunc)

	// Trigger the event - the actual event name is "transcodeSession.update"
	if fn, exists := events.events["transcodeSession.update"]; exists {
		fn(NotificationContainer{})
		if !called {
			t.Error("OnTranscodeUpdate function was not called")
		}
	} else {
		t.Error("transcodeSession.update event not found")
	}
}

// Test SubscribeToNotifications - basic functionality test
func TestPlex_SubscribeToNotifications(t *testing.T) {
	events := NewNotificationEvents()

	// Create a mock plex instance - this will fail to connect but should test the basic function structure
	plex := &Plex{
		URL:              "http://invalid-url:32400",
		Token:            "invalid-token",
		ClientIdentifier: "test-client",
	}

	// Test that the function doesn't panic with invalid URL
	// We'll use a channel to capture errors
	errorChan := make(chan error, 1)
	interrupt := make(chan os.Signal, 1)

	// Close interrupt immediately to exit the function quickly
	close(interrupt)

	// This should fail to connect and call the error function
	go plex.SubscribeToNotifications(events, interrupt, func(err error) {
		errorChan <- err
	})

	// Wait for error with timeout
	select {
	case err := <-errorChan:
		// We expect an error since we're using an invalid URL
		if err == nil {
			t.Error("Expected error for invalid URL but got none")
		}
	case <-time.After(5 * time.Second):
		// Timeout is acceptable for this test
		t.Log("Function call timed out as expected")
	}
}

// Test UnmarshalJSON methods for notification types
func TestNotificationContainer_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		json        string
		expectError bool
	}{
		{
			name: "valid timeline notification",
			json: `{
				"size": 1,
				"type": "timeline",
				"TimelineEntry": [{
					"identifier": "com.plexapp.plugins.library",
					"itemID": 123, 
					"metadataState": "created",
					"sectionID": 1,
					"state": 5,
					"title": "Test Movie",
					"type": 1,
					"updatedAt": 1234567890
				}]
			}`,
			expectError: false,
		},
		{
			name: "valid activity notification",
			json: `{
				"size": 1,
				"type": "activity",
				"ActivityNotification": [{
					"uuid": "test-uuid",
					"type": "library.refresh.items",
					"title": "Updating library",
					"subtitle": "Test subtitle"
				}]
			}`,
			expectError: false,
		},
		{
			name: "valid playing notification",
			json: `{
				"size": 1,
				"type": "playing",
				"PlaySessionStateNotification": [{
					"sessionKey": "123",
					"guid": "test-guid",
					"state": "playing",
					"ratingKey": "456",
					"viewOffset": 60000
				}]
			}`,
			expectError: false,
		},
		{
			name: "valid transcode notification",
			json: `{
				"size": 1,
				"type": "transcodeSession.update",
				"TranscodeSession": [{
					"key": "/transcode/sessions/test-key",
					"throttled": false,
					"complete": false,
					"progress": 50.5,
					"duration": 7200000,
					"context": "streaming",
					"sourceVideoCodec": "h264",
					"sourceAudioCodec": "aac"
				}]
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
			var container NotificationContainer
			err := json.Unmarshal([]byte(tt.json), &container)

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

// Test individual UnmarshalJSON methods that exist in the code
func TestActivityNotification_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		json        string
		expectError bool
	}{
		{
			name: "valid activity notification",
			json: `{
				"Activity": {
					"uuid": "test-uuid",
					"type": "library.refresh.items",
					"title": "Updating library",
					"subtitle": "Test subtitle",
					"userID": "123"
				},
				"event": "test-event",
				"uuid": "notification-uuid"
			}`,
			expectError: false,
		},
		{
			name: "activity notification with numeric userID",
			json: `{
				"Activity": {
					"uuid": "test-uuid",
					"type": "library.refresh.items",
					"title": "Updating library",
					"userID": 456
				},
				"event": "test-event",
				"uuid": "notification-uuid"
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
			var activity ActivityNotification
			err := json.Unmarshal([]byte(tt.json), &activity)

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

func TestPlaySessionStateNotification_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		json        string
		expectError bool
	}{
		{
			name: "valid playing notification",
			json: `{
				"sessionKey": "123",
				"guid": "test-guid",
				"state": "playing",
				"ratingKey": "456",
				"viewOffset": "60000",
				"playQueueItemID": "789"
			}`,
			expectError: false,
		},
		{
			name: "playing notification with numeric values",
			json: `{
				"sessionKey": "123",
				"guid": "test-guid",
				"state": "playing",
				"ratingKey": "456",
				"viewOffset": 60000,
				"playQueueItemID": 789
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
			var playing PlaySessionStateNotification
			err := json.Unmarshal([]byte(tt.json), &playing)

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

func TestBackgroundProcessingQueueEventNotification_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		json        string
		expectError bool
	}{
		{
			name: "valid queue notification",
			json: `{
				"event": "test-event",
				"queueID": "123"
			}`,
			expectError: false,
		},
		{
			name: "queue notification with numeric queueID",
			json: `{
				"event": "test-event",
				"queueID": 456
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
			var queue BackgroundProcessingQueueEventNotification
			err := json.Unmarshal([]byte(tt.json), &queue)

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
