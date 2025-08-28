package plex

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// Test that SubscribeToNotificationsWithContext cancels and the server sees the close
func TestSubscribeToNotificationsWithContext_Cancels(t *testing.T) {
	upgrader := websocket.Upgrader{}

	connected := make(chan struct{})
	closed := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("upgrade error: %v", err)
			return
		}

		// signal connected
		close(connected)

		// read until connection is closed
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				// signal closed
				close(closed)
				return
			}
		}
	}))
	defer srv.Close()

	p := &Plex{URL: srv.URL, Token: "", ClientIdentifier: "test-client"}
	events := NewNotificationEvents()

	ctx, cancel := context.WithCancel(context.Background())

	// start subscription
	p.SubscribeToNotificationsWithContext(ctx, events, func(err error) {
		if err != nil {
			// log but don't fail the test here
			t.Logf("subscribe error: %v", err)
		}
	})

	// wait for server to accept connection
	select {
	case <-connected:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for websocket connection")
	}

	// cancel the context and ensure server sees close
	cancel()

	select {
	case <-closed:
		// success
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for server to observe connection close after cancel")
	}
}

// Test that GetFriends can consume a large XML response and return the expected count
func TestGetFriends_LargePayload(t *testing.T) {
	const want = 2000

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = fmt.Fprintf(w, "<MediaContainer size=\"%d\">", want)

		for i := 0; i < want; i++ {
			_, _ = fmt.Fprintf(w, `<User id="%d" title="u%d"><Server id="%d" serverId="s%d" machineIdentifier="m%d" name="n%d" lastSeenAt="now" numLibraries="1" allLibraries="1" owned="1" pending="0"/></User>`, i, i, i, i, i, i)
		}

		_, _ = fmt.Fprint(w, "</MediaContainer>")
	}))
	defer srv.Close()

	p := &Plex{URL: srv.URL, Token: "", ClientIdentifier: "test-client"}

	got, err := p.GetFriends()
	if err != nil {
		t.Fatalf("GetFriends error: %v", err)
	}

	if len(got) != want {
		t.Fatalf("expected %d friends, got %d", want, len(got))
	}
}
