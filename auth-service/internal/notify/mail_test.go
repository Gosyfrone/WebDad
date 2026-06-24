package notify

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func mailResponse(status int) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}
}

func TestNewMailClient(t *testing.T) {
	c := NewMailClient("http://mail:8089", "secret-test")
	if c.baseURL != "http://mail:8089" || c.secret != "secret-test" || c.http == nil {
		t.Fatalf("client mal construit: %+v", c)
	}
}

func TestSend(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		doErr   error
		wantErr bool
	}{
		{name: "success", status: http.StatusOK},
		{name: "server error", status: http.StatusInternalServerError, wantErr: true},
		{name: "transport error", doErr: errors.New("mail indisponible"), wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := NewMailClient("http://mail.test", "my-secret")
			var req *http.Request
			c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				req = r
				if tc.doErr != nil {
					return nil, tc.doErr
				}
				return mailResponse(tc.status), nil
			})
			err := c.Send("alice@breezy.dev", "Bienvenue", "<b>Hi</b>", "Hi")
			if (err != nil) != tc.wantErr {
				t.Fatalf("error=%v wantErr=%v", err, tc.wantErr)
			}
			if req == nil || req.Method != http.MethodPost || req.Header.Get("X-Internal-Secret") != "my-secret" {
				t.Fatalf("requête inattendue: %v", req)
			}
			var payload sendPayload
			if err := json.NewDecoder(req.Body).Decode(&payload); err != nil || payload.To != "alice@breezy.dev" || payload.Subject != "Bienvenue" {
				t.Fatalf("payload=%+v err=%v", payload, err)
			}
		})
	}
}

func TestSendInvalidURL(t *testing.T) {
	c := NewMailClient("://bad-url", "s")
	if err := c.Send("a@b.com", "sub", "", "text"); err == nil {
		t.Fatal("URL invalide acceptée")
	}
}
