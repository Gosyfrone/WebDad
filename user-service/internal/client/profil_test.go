package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func response(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestNewProfilClient(t *testing.T) {
	c := NewProfilClient("http://profil-service:8083")
	if c.baseURL != "http://profil-service:8083" || c.httpClient == nil {
		t.Fatalf("client mal construit: %+v", c)
	}
}

func TestVisibility(t *testing.T) {
	tests := []struct {
		name         string
		status       int
		body         string
		want         string
		wantErr      bool
		transportErr error
	}{
		{name: "public", status: http.StatusOK, body: `{"visibility":"public"}`, want: "public"},
		{name: "private", status: http.StatusOK, body: `{"visibility":"private"}`, want: VisibilityPrivate},
		{name: "not found", status: http.StatusNotFound, want: ""},
		{name: "bad request", status: http.StatusBadRequest, wantErr: true},
		{name: "server error", status: http.StatusInternalServerError, wantErr: true},
		{name: "invalid json", status: http.StatusOK, body: `{`, wantErr: true},
		{name: "transport error", wantErr: true, transportErr: errors.New("profil indisponible")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotRequest *http.Request
			c := NewProfilClient("http://profil.test")
			c.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				gotRequest = r
				if tc.transportErr != nil {
					return nil, tc.transportErr
				}
				return response(tc.status, tc.body), nil
			})
			got, err := c.Visibility(context.Background(), "user/with space")
			if (err != nil) != tc.wantErr {
				t.Fatalf("Visibility error = %v, wantErr %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Fatalf("Visibility = %q, attendu %q", got, tc.want)
			}
			if gotRequest == nil || gotRequest.Method != http.MethodGet || gotRequest.URL.EscapedPath() != "/profils/user%2Fwith%20space/visibility" {
				t.Fatalf("requête inattendue: %v", gotRequest)
			}
		})
	}
}

func TestVisibility_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := NewProfilClient("http://profil.test")
	c.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return nil, r.Context().Err()
	})
	if _, err := c.Visibility(ctx, "user"); err == nil {
		t.Fatal("un contexte annulé doit retourner une erreur")
	}
}

func TestVisibility_InvalidURL(t *testing.T) {
	c := NewProfilClient("http://exemple\x7f.invalide")
	if _, err := c.Visibility(context.Background(), "x"); err == nil {
		t.Fatal("une URL invalide doit être refusée")
	}
}
