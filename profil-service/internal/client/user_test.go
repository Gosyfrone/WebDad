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

func clientResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
}

func TestNewUserClient(t *testing.T) {
	c := NewUserClient("http://user-service:8082///")
	if c.baseURL != "http://user-service:8082" || c.http == nil {
		t.Fatalf("client mal construit: %+v", c)
	}
}

func TestAcceptAllFollowRequests(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		doErr   error
		wantErr bool
	}{
		{name: "success", status: http.StatusNoContent},
		{name: "server error", status: http.StatusInternalServerError, wantErr: true},
		{name: "transport error", doErr: errors.New("user-service indisponible"), wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var request *http.Request
			c := NewUserClient("http://user.test")
			c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				request = r
				if tc.doErr != nil {
					return nil, tc.doErr
				}
				return clientResponse(tc.status, ""), nil
			})
			err := c.AcceptAllFollowRequests(context.Background(), "owner-123")
			if (err != nil) != tc.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tc.wantErr)
			}
			if request == nil || request.Method != http.MethodPost || request.URL.Path != "/internal/users/owner-123/accept-all-follow-requests" {
				t.Fatalf("requête inattendue: %v", request)
			}
		})
	}
}

func TestAcceptAllFollowRequests_InvalidURL(t *testing.T) {
	c := NewUserClient("%")
	if err := c.AcceptAllFollowRequests(context.Background(), "owner"); err == nil {
		t.Fatal("URL invalide acceptée")
	}
}

func TestIsFollowing(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		doErr   error
		want    bool
		wantErr bool
	}{
		{name: "true", status: http.StatusOK, body: `{"isFollowing":true}`, want: true},
		{name: "false", status: http.StatusOK, body: `{"isFollowing":false}`},
		{name: "bad status", status: http.StatusBadGateway, wantErr: true},
		{name: "invalid json", status: http.StatusOK, body: `{`, wantErr: true},
		{name: "transport error", doErr: errors.New("user-service indisponible"), wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var request *http.Request
			c := NewUserClient("http://user.test")
			c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				request = r
				if tc.doErr != nil {
					return nil, tc.doErr
				}
				return clientResponse(tc.status, tc.body), nil
			})
			got, err := c.IsFollowing(context.Background(), "follower", "following")
			if (err != nil) != tc.wantErr || got != tc.want {
				t.Fatalf("IsFollowing = %v, %v; attendu %v, wantErr=%v", got, err, tc.want, tc.wantErr)
			}
			if request == nil || request.Method != http.MethodGet || request.URL.Path != "/internal/follows/follower/is-following/following" {
				t.Fatalf("requête inattendue: %v", request)
			}
		})
	}
}

func TestIsFollowing_InvalidURL(t *testing.T) {
	c := NewUserClient("%")
	if _, err := c.IsFollowing(context.Background(), "a", "b"); err == nil {
		t.Fatal("URL invalide acceptée")
	}
}
