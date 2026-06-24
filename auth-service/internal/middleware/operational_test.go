package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestOperationalMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("request id supplied and generated", func(t *testing.T) {
		for _, supplied := range []string{"known-id", ""} {
			r := gin.New()
			r.Use(RequestID())
			r.GET("/", func(c *gin.Context) { c.String(200, c.GetString("request_id")) })
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("X-Request-Id", supplied)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != 200 || w.Header().Get("X-Request-Id") == "" {
				t.Fatalf("response: %d %#v", w.Code, w.Header())
			}
			if supplied != "" && w.Body.String() != supplied {
				t.Fatalf("request id = %q", w.Body.String())
			}
		}
	})

	t.Run("logger statuses", func(t *testing.T) {
		for _, status := range []int{200, 400, 500} {
			r := gin.New()
			r.Use(RequestLogger())
			r.GET("/", func(c *gin.Context) { c.Set("user_id", "u"); c.Status(status) })
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/?secret=x", nil))
			if w.Code != status {
				t.Fatalf("status = %d", w.Code)
			}
		}
	})

	t.Run("recovery", func(t *testing.T) {
		r := gin.New()
		r.Use(Recovery())
		r.GET("/", func(*gin.Context) { panic("boom") })
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d", w.Code)
		}
	})
}

func TestRateLimiter(t *testing.T) {
	rl := NewRateLimiter(2, 10*time.Millisecond)
	if ok, _ := rl.allow("ip"); !ok {
		t.Fatal("first request denied")
	}
	if ok, _ := rl.allow("ip"); !ok {
		t.Fatal("second request denied")
	}
	if ok, retry := rl.allow("ip"); ok || retry <= 0 {
		t.Fatalf("third request = %v, %s", ok, retry)
	}
	time.Sleep(12 * time.Millisecond)
	if ok, _ := rl.allow("ip"); !ok {
		t.Fatal("request after reset denied")
	}

	r := gin.New()
	limited := NewRateLimiter(1, time.Hour)
	r.Use(limited.Middleware())
	r.GET("/", func(c *gin.Context) { c.Status(204) })
	for i, want := range []int{204, 429} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
		if w.Code != want {
			t.Fatalf("request %d = %d, want %d", i, w.Code, want)
		}
		if want == 429 && w.Header().Get("Retry-After") == "" {
			t.Fatal("missing Retry-After")
		}
	}

	cleanup := NewRateLimiter(1, time.Millisecond)
	cleanup.mu.Lock()
	cleanup.visitors["old"] = &window{count: 1, resetAt: time.Now().Add(-time.Second)}
	cleanup.mu.Unlock()
	time.Sleep(4 * time.Millisecond)
	cleanup.mu.Lock()
	_, exists := cleanup.visitors["old"]
	cleanup.mu.Unlock()
	if exists {
		t.Fatal("cleanup did not remove expired visitor")
	}
}
