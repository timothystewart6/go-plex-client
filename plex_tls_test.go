package plex

import (
	"net/http"
	"os"
	"testing"
)

// Test that WithInsecureSkipVerify applies InsecureSkipVerify to both HTTP clients
func TestWithInsecureSkipVerifyOption(t *testing.T) {
	p, err := New("https://example.local", "token", WithInsecureSkipVerify())
	if err != nil {
		t.Fatalf("unexpected error from New: %v", err)
	}

	// HTTPClient transport
	ht, ok := p.HTTPClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected HTTPClient.Transport to be *http.Transport, got %T", p.HTTPClient.Transport)
	}

	if ht.TLSClientConfig == nil || !ht.TLSClientConfig.InsecureSkipVerify {
		t.Fatalf("expected HTTPClient.Transport TLSClientConfig.InsecureSkipVerify to be true")
	}

	// DownloadClient transport
	dt, ok := p.DownloadClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected DownloadClient.Transport to be *http.Transport, got %T", p.DownloadClient.Transport)
	}

	if dt.TLSClientConfig == nil || !dt.TLSClientConfig.InsecureSkipVerify {
		t.Fatalf("expected DownloadClient.Transport TLSClientConfig.InsecureSkipVerify to be true")
	}
}

// Test that the SKIP_TLS_VERIFICATION env var enables the insecure option
func TestEnvVarEnablesSkipTLSVerification(t *testing.T) {
	// Set env var and ensure cleanup
	if err := os.Setenv("SKIP_TLS_VERIFICATION", "1"); err != nil {
		t.Fatalf("failed to set env: %v", err)
	}
	defer os.Unsetenv("SKIP_TLS_VERIFICATION")

	p, err := New("https://example.local", "token")
	if err != nil {
		t.Fatalf("unexpected error from New: %v", err)
	}

	ht, ok := p.HTTPClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected HTTPClient.Transport to be *http.Transport, got %T", p.HTTPClient.Transport)
	}

	if ht.TLSClientConfig == nil || !ht.TLSClientConfig.InsecureSkipVerify {
		t.Fatalf("expected HTTPClient.Transport TLSClientConfig.InsecureSkipVerify to be true due to env var")
	}
}
