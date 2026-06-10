package monitoring

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestProbeOne(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "ok", "service": "post-service", "uptime_seconds": 42,
		})
	}))
	defer up.Close()

	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer down.Close()

	// Serveur fermé d'avance → injoignable.
	closed := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	unreachable := closed.URL
	closed.Close()

	client := &http.Client{Timeout: time.Second}

	if h := probeOne(context.Background(), client, "/posts", up.URL); h.Status != "up" ||
		h.UptimeSeconds != 42 || h.Name != "post-service" {
		t.Fatalf("service up : %+v", h)
	}
	if h := probeOne(context.Background(), client, "/users", down.URL); h.Status != "down" {
		t.Fatalf("service 500 doit être down : %+v", h)
	}
	if h := probeOne(context.Background(), client, "/x", unreachable); h.Status != "down" || h.Error == "" {
		t.Fatalf("service injoignable doit être down + error : %+v", h)
	}
}

// TestProbeAllIncludesGateway : la liste commence toujours par le gateway, « up ».
func TestProbeAllIncludesGateway(t *testing.T) {
	res := probeAll(context.Background(), &http.Client{Timeout: time.Second}, map[string]string{})
	if len(res) != 1 || res[0].Name != "api-gateway" || res[0].Status != "up" {
		t.Fatalf("le gateway doit être en tête et up : %+v", res)
	}
}

// TestProbeAllFanOut : un service down et un service up sont tous deux présents.
func TestProbeAllFanOut(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok", "service": "auth", "uptime_seconds": 1})
	}))
	defer up.Close()
	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer down.Close()

	res := probeAll(context.Background(), &http.Client{Timeout: time.Second}, map[string]string{
		"/auth": up.URL, "/users": down.URL,
	})
	if len(res) != 3 { // gateway + 2
		t.Fatalf("attendu 3 entrées, eu %d : %+v", len(res), res)
	}
	statuses := map[string]string{}
	for _, h := range res {
		statuses[h.Prefix] = h.Status
	}
	if statuses["/auth"] != "up" || statuses["/users"] != "down" {
		t.Fatalf("statuts inattendus : %+v", statuses)
	}
}
