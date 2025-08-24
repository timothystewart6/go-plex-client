package plex

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Test boolToOneOrZero function
func TestBoolToOneOrZero(t *testing.T) {
	tests := []struct {
		input    bool
		expected string
	}{
		{true, "1"},
		{false, "0"},
	}

	for _, test := range tests {
		result := boolToOneOrZero(test.input)
		if result != test.expected {
			t.Errorf("boolToOneOrZero(%v) = %s; want %s", test.input, result, test.expected)
		}
	}
}

// Test get function
func TestGet(t *testing.T) {
	tests := []struct {
		name         string
		headers      headers
		expectError  bool
		statusCode   int
		responseBody string
	}{
		{
			name: "successful get request",
			headers: headers{
				Accept:           "application/json",
				Platform:         "test",
				PlatformVersion:  "1.0",
				Provides:         "controller",
				ClientIdentifier: "test-client",
				Product:          "test-product",
				Version:          "1.0",
				Device:           "test-device",
				Token:            "test-token",
			},
			expectError:  false,
			statusCode:   http.StatusOK,
			responseBody: `{"result": "success"}`,
		},
		{
			name: "get request without token",
			headers: headers{
				Accept:           "application/json",
				Platform:         "test",
				ClientIdentifier: "test-client",
				Product:          "test-product",
				Version:          "1.0",
				Device:           "test-device",
			},
			expectError:  false,
			statusCode:   http.StatusOK,
			responseBody: `{"result": "success"}`,
		},
		{
			name: "get request with target client identifier - not supported in get function",
			headers: headers{
				Accept:                 "application/json",
				Platform:               "test",
				ClientIdentifier:       "test-client",
				Product:                "test-product",
				Version:                "1.0",
				Device:                 "test-device",
				Token:                  "test-token",
				TargetClientIdentifier: "target-client",
			},
			expectError:  false,
			statusCode:   http.StatusOK,
			responseBody: `{"result": "success"}`,
		},
		{
			name: "server error",
			headers: headers{
				Accept:           "application/json",
				ClientIdentifier: "test-client",
			},
			expectError:  false,
			statusCode:   http.StatusInternalServerError,
			responseBody: "Internal Server Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check method
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET method, got %s", r.Method)
				}

				// Check headers
				if r.Header.Get("Accept") != tt.headers.Accept {
					t.Errorf("Expected Accept header %s, got %s", tt.headers.Accept, r.Header.Get("Accept"))
				}
				if r.Header.Get("X-Plex-Platform") != tt.headers.Platform {
					t.Errorf("Expected Platform header %s, got %s", tt.headers.Platform, r.Header.Get("X-Plex-Platform"))
				}
				if r.Header.Get("X-Plex-Client-Identifier") != tt.headers.ClientIdentifier {
					t.Errorf("Expected ClientIdentifier header %s, got %s", tt.headers.ClientIdentifier, r.Header.Get("X-Plex-Client-Identifier"))
				}

				if tt.headers.Token != "" {
					if r.Header.Get("X-Plex-Token") != tt.headers.Token {
						t.Errorf("Expected Token header %s, got %s", tt.headers.Token, r.Header.Get("X-Plex-Token"))
					}
				}

				if tt.headers.TargetClientIdentifier != "" {
					// Note: get function doesn't support TargetClientIdentifier
					// This is expected behavior - only Plex methods support this header
				}

				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			resp, err := get(server.URL, tt.headers)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				if resp.StatusCode != tt.statusCode {
					t.Errorf("Expected status code %d, got %d", tt.statusCode, resp.StatusCode)
				}

				resp.Body.Close()
			}
		})
	}
}

// Test get function with invalid URL
func TestGet_InvalidURL(t *testing.T) {
	headers := headers{Accept: "application/json"}

	_, err := get("://invalid-url", headers)
	if err == nil {
		t.Errorf("Expected error for invalid URL but got none")
	}
}

// Test get function with timeout
func TestGet_Timeout(t *testing.T) {
	// Create a server that never responds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second) // Sleep longer than the 3 second timeout
	}))
	defer server.Close()

	headers := headers{Accept: "application/json"}

	_, err := get(server.URL, headers)
	if err == nil {
		t.Errorf("Expected timeout error but got none")
	}

	if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "deadline") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

// Test post function
func TestPost(t *testing.T) {
	tests := []struct {
		name         string
		body         []byte
		headers      headers
		expectError  bool
		statusCode   int
		responseBody string
	}{
		{
			name: "successful post request",
			body: []byte(`{"test": "data"}`),
			headers: headers{
				Accept:           "application/json",
				Platform:         "test",
				PlatformVersion:  "1.0",
				Provides:         "controller",
				ClientIdentifier: "test-client",
				Product:          "test-product",
				Version:          "1.0",
				Device:           "test-device",
				Token:            "test-token",
				ContentType:      "application/json",
			},
			expectError:  false,
			statusCode:   http.StatusOK,
			responseBody: `{"result": "success"}`,
		},
		{
			name: "post request without token",
			body: []byte(`{"test": "data"}`),
			headers: headers{
				Accept:           "application/json",
				Platform:         "test",
				ClientIdentifier: "test-client",
				Product:          "test-product",
				Version:          "1.0",
				Device:           "test-device",
				ContentType:      "application/json",
			},
			expectError:  false,
			statusCode:   http.StatusCreated,
			responseBody: `{"result": "created"}`,
		},
		{
			name: "post request with empty body",
			body: []byte{},
			headers: headers{
				Accept:           "application/json",
				ClientIdentifier: "test-client",
			},
			expectError:  false,
			statusCode:   http.StatusOK,
			responseBody: `{"result": "success"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check method
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST method, got %s", r.Method)
				}

				// Check headers
				if r.Header.Get("Accept") != tt.headers.Accept {
					t.Errorf("Expected Accept header %s, got %s", tt.headers.Accept, r.Header.Get("Accept"))
				}
				if r.Header.Get("X-Plex-Platform") != tt.headers.Platform {
					t.Errorf("Expected Platform header %s, got %s", tt.headers.Platform, r.Header.Get("X-Plex-Platform"))
				}
				if r.Header.Get("X-Plex-Client-Identifier") != tt.headers.ClientIdentifier {
					t.Errorf("Expected ClientIdentifier header %s, got %s", tt.headers.ClientIdentifier, r.Header.Get("X-Plex-Client-Identifier"))
				}

				if tt.headers.Token != "" {
					if r.Header.Get("X-Plex-Token") != tt.headers.Token {
						t.Errorf("Expected Token header %s, got %s", tt.headers.Token, r.Header.Get("X-Plex-Token"))
					}
				}

				if tt.headers.ContentType != "" {
					if r.Header.Get("Content-Type") != tt.headers.ContentType {
						t.Errorf("Expected Content-Type header %s, got %s", tt.headers.ContentType, r.Header.Get("Content-Type"))
					}
				}

				// Read and verify body
				buf := make([]byte, len(tt.body))
				r.Body.Read(buf)
				if string(buf) != string(tt.body) {
					t.Errorf("Expected body %s, got %s", string(tt.body), string(buf))
				}

				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			resp, err := post(server.URL, tt.body, tt.headers)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				if resp.StatusCode != tt.statusCode {
					t.Errorf("Expected status code %d, got %d", tt.statusCode, resp.StatusCode)
				}

				resp.Body.Close()
			}
		})
	}
}

// Test post function with invalid URL
func TestPost_InvalidURL(t *testing.T) {
	headers := headers{Accept: "application/json"}
	body := []byte(`{"test": "data"}`)

	_, err := post("://invalid-url", body, headers)
	if err == nil {
		t.Errorf("Expected error for invalid URL but got none")
	}
}

// Test post function with timeout
func TestPost_Timeout(t *testing.T) {
	// Create a server that never responds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second) // Sleep longer than the 3 second timeout
	}))
	defer server.Close()

	headers := headers{Accept: "application/json"}
	body := []byte(`{"test": "data"}`)

	_, err := post(server.URL, body, headers)
	if err == nil {
		t.Errorf("Expected timeout error but got none")
	}

	if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "deadline") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}
