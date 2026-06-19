package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/webdad/api-gateway/internal/config"
)

const testSecret = "test-secret"

// testClaims reflète les champs utilisés par le middleware AdminJWT.
type testClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func makeAdminToken(t *testing.T) string {
	t.Helper()
	now := time.Now()
	cl := &testClaims{
		UserID: "11111111-1111-1111-1111-111111111111",
		Email:  "admin@breezy.dev",
		Role:   "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, cl).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("makeAdminToken: %v", err)
	}
	return tok
}

func newTestConfig(backendURL string) *config.Config {
	return &config.Config{
		AllowedOrigins: []string{"http://localhost:3000"},
		JWTSecret:      testSecret,
		Services: map[string]string{
			"/auth":  backendURL,
			"/posts": backendURL,
		},
	}
}

func newTestRouter(t *testing.T, cfg *config.Config) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r, err := New(cfg)
	if err != nil {
		t.Fatalf("router.New: %v", err)
	}
	return r
}

func TestHealth(t *testing.T) {
	r := newTestRouter(t, newTestConfig("http://localhost:9999"))
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /health = %d, attendu 200", w.Code)
	}
}

func TestAdminMonitoring_SansToken(t *testing.T) {
	r := newTestRouter(t, newTestConfig("http://localhost:9999"))
	req := httptest.NewRequest(http.MethodGet, "/admin/monitoring", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("GET /admin/monitoring sans token = %d, attendu 401", w.Code)
	}
}

func TestAdminMonitoring_TokenAdmin(t *testing.T) {
	r := newTestRouter(t, newTestConfig("http://localhost:9999"))
	req := httptest.NewRequest(http.MethodGet, "/admin/monitoring", nil)
	req.Header.Set("Authorization", "Bearer "+makeAdminToken(t))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// 200 ou 207 (multi-service monitoring), pas 401/403
	if w.Code == http.StatusUnauthorized || w.Code == http.StatusForbidden {
		t.Fatalf("GET /admin/monitoring token admin = %d, attendu pas 401/403", w.Code)
	}
}

func TestProxyRoute_ForwardsRequête(t *testing.T) {
	received := make(chan string, 1)
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	r := newTestRouter(t, newTestConfig(backend.URL))

	// httputil.ReverseProxy panique avec httptest.ResponseRecorder (pas de
	// CloseNotifier) → on passe par un vrai serveur HTTP pour éviter ce piège.
	gateway := httptest.NewServer(r)
	defer gateway.Close()

	resp, err := http.Get(gateway.URL + "/auth/login")
	if err != nil {
		t.Fatalf("requête gateway: %v", err)
	}
	_ = resp.Body.Close()

	select {
	case path := <-received:
		if path != "/auth/login" {
			t.Fatalf("backend a reçu path=%q, attendu /auth/login", path)
		}
	default:
		t.Fatal("backend n'a pas reçu la requête")
	}
}

func TestNew_ServiceURLInvalide(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		AllowedOrigins: []string{"http://localhost:3000"},
		JWTSecret:      testSecret,
		Services:       map[string]string{"/bad": "://invalid-url"},
	}
	_, err := New(cfg)
	if err == nil {
		t.Fatal("New avec URL invalide doit retourner une erreur")
	}
}
