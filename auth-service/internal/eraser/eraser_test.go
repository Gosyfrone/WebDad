package eraser

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

func eraseResponse(status int) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}
}

func TestEraseTolerates404AndReportsFailures(t *testing.T) {
	e := New(Targets{User: "http://services.test", Profil: "http://services.test", Post: "http://services.test", Message: "http://services.test", Media: "http://services.test"})
	var auth string
	e.client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		auth = r.Header.Get("Authorization")
		switch {
		case strings.HasSuffix(r.URL.Path, "/hard"):
			return eraseResponse(http.StatusInternalServerError), nil
		case strings.Contains(r.URL.Path, "/profils/"):
			return eraseResponse(http.StatusNotFound), nil
		default:
			return eraseResponse(http.StatusNoContent), nil
		}
	})
	failed := e.Erase(context.Background(), "tok123", "u1")
	if len(failed) != 1 || failed[0] != "user" || auth != "Bearer tok123" {
		t.Fatalf("failed=%v auth=%q", failed, auth)
	}
}

func TestEraseSkipsUnconfigured(t *testing.T) {
	e := New(Targets{})
	if failed := e.Erase(context.Background(), "tok", "u1"); len(failed) != 0 {
		t.Fatalf("échecs inattendus: %v", failed)
	}
}

func TestDeleteOKFailures(t *testing.T) {
	e := New(Targets{})
	if e.deleteOK(context.Background(), "token", "://bad-url") {
		t.Fatal("URL mal formée acceptée")
	}
	e.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("service indisponible")
	})
	if e.deleteOK(context.Background(), "token", "http://service.test/delete") {
		t.Fatal("erreur transport considérée comme un succès")
	}
}
