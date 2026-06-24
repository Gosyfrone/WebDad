package client

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestEmit_HTTPBranches(t *testing.T) {
	tests := []struct {
		name string
		resp *http.Response
		err  error
	}{
		{name: "success", resp: response(http.StatusNoContent, "")},
		{name: "server error", resp: response(http.StatusInternalServerError, strings.Repeat("x", 600))},
		{name: "transport error", err: errors.New("notification indisponible")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			done := make(chan *http.Request, 1)
			c := &NotificationClient{
				url:    "http://notification.test/internal/events",
				secret: "shared-secret",
				client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					done <- r
					return tc.resp, tc.err
				})},
			}
			ev := Event{Type: TypeFollowRequest, ActorID: "actor", RecipientID: "recipient", Retract: true}
			c.Emit(ev)
			select {
			case req := <-done:
				if req.Method != http.MethodPost || req.Header.Get("Content-Type") != "application/json" || req.Header.Get("X-Internal-Secret") != "shared-secret" {
					t.Fatalf("requête inattendue: %+v", req)
				}
				body, err := io.ReadAll(req.Body)
				if err != nil {
					t.Fatal(err)
				}
				var got Event
				if err := json.Unmarshal(body, &got); err != nil || got != ev {
					t.Fatalf("event reçu = %+v, err=%v", got, err)
				}
			case <-time.After(time.Second):
				t.Fatal("Emit n'a pas exécuté la requête")
			}
		})
	}
}

func TestEmit_InvalidURLReturnsSilently(t *testing.T) {
	c := &NotificationClient{url: "http://invalid\x7f.test", client: &http.Client{}}
	c.Emit(Event{Type: TypeFollow})
	time.Sleep(10 * time.Millisecond)
}
