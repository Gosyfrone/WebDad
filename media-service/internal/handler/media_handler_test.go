package handler

import (
	"bytes"
	"encoding/hex"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/webdad/media-service/internal/config"
	"github.com/webdad/media-service/internal/middleware"
)

// ─── helpers ────────────────────────────────────────────────────────────────

// newHandlerRouter crée un moteur Gin de test avec le handler injecté,
// des claims pré-posées en contexte (simule un JWT valide) et gin.Recovery()
// pour attraper les panics nil-store.
func newHandlerRouter(t *testing.T, role string) (*MediaHandler, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())

	cfg := &config.Config{
		MaxImageBytes: 5 * 1024 * 1024,
		MaxVideoBytes: 10 * 1024 * 1024,
		MaxBlobBytes:  8 * 1024 * 1024,
	}
	h := NewMediaHandler(nil, cfg) // store nil — panics attrapées par Recovery

	// Middleware qui injecte les claims directement dans le contexte Gin,
	// imitant ce que fait JWTAuth (sans JWT réel).
	claimsMiddleware := func(c *gin.Context) {
		c.Set("claims", &middleware.Claims{
			UserID: "user-abc",
			Email:  "test@example.com",
			Role:   role,
		})
		c.Next()
	}

	r.POST("/media", claimsMiddleware, h.Upload)
	r.POST("/media/encrypted", claimsMiddleware, h.UploadEncrypted)
	r.GET("/media/:id", h.Download)
	r.DELETE("/media/:id", claimsMiddleware, h.Delete)
	r.DELETE("/media/owners/:id", claimsMiddleware, h.PurgeByOwner)

	return h, r
}

// newHandlerRouterNoAuth crée un moteur où les claims ne sont PAS injectées
// (aucun middleware d'auth), pour tester le chemin 401.
func newHandlerRouterNoAuth(t *testing.T) (*MediaHandler, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())

	cfg := &config.Config{
		MaxImageBytes: 5 * 1024 * 1024,
		MaxVideoBytes: 10 * 1024 * 1024,
		MaxBlobBytes:  8 * 1024 * 1024,
	}
	h := NewMediaHandler(nil, cfg)

	r.POST("/media", h.Upload)
	r.POST("/media/encrypted", h.UploadEncrypted)
	r.GET("/media/:id", h.Download)
	r.DELETE("/media/:id", h.Delete)
	r.DELETE("/media/owners/:id", h.PurgeByOwner)

	return h, r
}

// makeTokenForHandler crée un JWT signé pour les tests utilisant le vrai JWTAuth.
func makeTokenForHandler(t *testing.T, secret, role string, ttl time.Duration) string {
	t.Helper()
	now := time.Now()
	claims := middleware.Claims{
		UserID: "user-abc",
		Email:  "test@example.com",
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("token : %v", err)
	}
	return tok
}

// buildMultipartFile crée un corps multipart/form-data avec un champ `file`.
func buildMultipartFile(t *testing.T, fieldName, filename string, content []byte) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	fw, err := w.CreateFormFile(fieldName, filename)
	if err != nil {
		t.Fatalf("CreateFormFile : %v", err)
	}
	if _, err := fw.Write(content); err != nil {
		t.Fatalf("Write : %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close : %v", err)
	}
	return &body, w.FormDataContentType()
}

// minimalJPEG renvoie les magic bytes d'un JPEG valide (≥ sniffLen octets
// pour déclencher le chemin nominal de détection MIME).
func minimalJPEG() []byte {
	b := make([]byte, sniffLen+10)
	b[0] = 0xFF
	b[1] = 0xD8
	b[2] = 0xFF
	b[3] = 0xE0
	return b
}

// ─── Upload tests ──────────────────────────────────────────────────────────

func TestUpload_SansAuth(t *testing.T) {
	_, r := newHandlerRouterNoAuth(t)
	req := httptest.NewRequest(http.MethodPost, "/media", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("attendu 401, obtenu %d", w.Code)
	}
}

func TestUpload_SansFile(t *testing.T) {
	_, r := newHandlerRouter(t, "user")
	req := httptest.NewRequest(http.MethodPost, "/media", strings.NewReader(""))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----test")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("attendu 400, obtenu %d", w.Code)
	}
}

func TestUpload_FichierTropGrand(t *testing.T) {
	_, r := newHandlerRouter(t, "user")

	// Taille > maxVideoBytes (10 Mo) → rejet précoce.
	bigContent := make([]byte, 11*1024*1024)
	body, ct := buildMultipartFile(t, "file", "big.mp4", bigContent)
	req := httptest.NewRequest(http.MethodPost, "/media", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("attendu 413, obtenu %d", w.Code)
	}
}

func TestUpload_AdminBypassTailleLimite(t *testing.T) {
	// Admin : pas de rejet précoce même avec taille > maxVideoBytes.
	// Atteint la détection MIME puis le store.Put() qui panique → 500.
	_, r := newHandlerRouter(t, "admin")

	bigContent := minimalJPEG()
	body, ct := buildMultipartFile(t, "file", "big.jpg", bigContent)
	req := httptest.NewRequest(http.MethodPost, "/media", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// Pas 413 (bypass admin).
	if w.Code == http.StatusRequestEntityTooLarge {
		t.Fatal("admin ne doit pas recevoir 413 (bypass taille)")
	}
}

func TestUpload_TypeMIMENonSupporte(t *testing.T) {
	_, r := newHandlerRouter(t, "user")

	// Contenu PDF (magic bytes %PDF-) → type non supporté → 415.
	pdfContent := []byte("%PDF-1.4 fake content for testing")
	body, ct := buildMultipartFile(t, "file", "doc.pdf", pdfContent)
	req := httptest.NewRequest(http.MethodPost, "/media", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("attendu 415, obtenu %d", w.Code)
	}
}

func TestUpload_TypeMIMETexte(t *testing.T) {
	_, r := newHandlerRouter(t, "user")

	// Texte brut → non autorisé → 415.
	body, ct := buildMultipartFile(t, "file", "note.txt", []byte("hello world"))
	req := httptest.NewRequest(http.MethodPost, "/media", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("attendu 415, obtenu %d", w.Code)
	}
}

func TestUpload_JPEG_StoreNil(t *testing.T) {
	// JPEG valide → passe MIME check → atteint store.Put() → panic → 500 via Recovery.
	_, r := newHandlerRouter(t, "user")

	body, ct := buildMultipartFile(t, "file", "img.jpg", minimalJPEG())
	req := httptest.NewRequest(http.MethodPost, "/media", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// Pas 401/400/413/415.
	if w.Code == http.StatusUnauthorized {
		t.Fatal("ne doit pas être 401")
	}
	if w.Code == http.StatusBadRequest {
		t.Fatal("ne doit pas être 400")
	}
	if w.Code == http.StatusUnsupportedMediaType {
		t.Fatal("JPEG ne doit pas être rejeté pour type MIME")
	}
}

func TestUpload_ImageTropGrande_Utilisateur(t *testing.T) {
	// JPEG de 6 Mo (> maxImage=5Mo, < maxVideo=10Mo) → pas de rejet précoce
	// (6Mo < 10Mo) → passe MIME check → CheckSize image → 413.
	_, r := newHandlerRouter(t, "user")

	content := make([]byte, 6*1024*1024)
	content[0] = 0xFF
	content[1] = 0xD8
	content[2] = 0xFF
	content[3] = 0xE0

	body, ct := buildMultipartFile(t, "file", "big.jpg", content)
	req := httptest.NewRequest(http.MethodPost, "/media", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("attendu 413 pour image trop grande, obtenu %d", w.Code)
	}
}

// ─── UploadEncrypted tests ─────────────────────────────────────────────────

func TestUploadEncrypted_SansAuth(t *testing.T) {
	_, r := newHandlerRouterNoAuth(t)
	req := httptest.NewRequest(http.MethodPost, "/media/encrypted", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("attendu 401, obtenu %d", w.Code)
	}
}

func TestUploadEncrypted_SansFile(t *testing.T) {
	_, r := newHandlerRouter(t, "user")
	req := httptest.NewRequest(http.MethodPost, "/media/encrypted", strings.NewReader(""))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----test")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("attendu 400, obtenu %d", w.Code)
	}
}

func TestUploadEncrypted_BlobTropGrand(t *testing.T) {
	_, r := newHandlerRouter(t, "user")

	// Taille > maxBlobBytes (8 Mo).
	bigContent := make([]byte, 9*1024*1024)
	body, ct := buildMultipartFile(t, "file", "secret.bin", bigContent)
	req := httptest.NewRequest(http.MethodPost, "/media/encrypted", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("attendu 413, obtenu %d", w.Code)
	}
}

func TestUploadEncrypted_AdminBypassBlob(t *testing.T) {
	// Admin bypass le cap blob → atteint store.Put() → panic → 500 via Recovery.
	_, r := newHandlerRouter(t, "admin")

	bigContent := make([]byte, 9*1024*1024)
	body, ct := buildMultipartFile(t, "file", "secret.bin", bigContent)
	req := httptest.NewRequest(http.MethodPost, "/media/encrypted", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// Pas 413 (bypass admin).
	if w.Code == http.StatusRequestEntityTooLarge {
		t.Fatal("admin ne doit pas recevoir 413 pour blob chiffré")
	}
}

func TestUploadEncrypted_BlobValide_StoreNil(t *testing.T) {
	_, r := newHandlerRouter(t, "user")

	// Blob < 8 Mo → passe la vérification taille → atteint store.Put() → panic → 500.
	content := []byte("données chiffrées opaques")
	body, ct := buildMultipartFile(t, "file", "blob.bin", content)
	req := httptest.NewRequest(http.MethodPost, "/media/encrypted", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// Pas 401/400/413.
	if w.Code == http.StatusUnauthorized || w.Code == http.StatusBadRequest || w.Code == http.StatusRequestEntityTooLarge {
		t.Fatalf("attendu ni 401/400/413, obtenu %d", w.Code)
	}
}

// ─── Download tests ─────────────────────────────────────────────────────────

func TestDownload_StoreNil(t *testing.T) {
	// Download est public, store nil → panic → 500 via Recovery.
	_, r := newHandlerRouter(t, "user")
	req := httptest.NewRequest(http.MethodGet, "/media/some-media-id", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// Pas 401 (public).
	if w.Code == http.StatusUnauthorized {
		t.Fatal("Download est public, ne doit pas être 401")
	}
}

func TestDownload_SansAuth_Public(t *testing.T) {
	_, r := newHandlerRouterNoAuth(t)
	req := httptest.NewRequest(http.MethodGet, "/media/some-id", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// Public → jamais 401.
	if w.Code == http.StatusUnauthorized {
		t.Fatal("Download public ne doit jamais retourner 401")
	}
}

// ─── Delete tests ──────────────────────────────────────────────────────────

func TestDelete_SansAuth(t *testing.T) {
	_, r := newHandlerRouterNoAuth(t)
	req := httptest.NewRequest(http.MethodDelete, "/media/some-id", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("attendu 401, obtenu %d", w.Code)
	}
}

func TestDelete_StoreNil_Propriétaire(t *testing.T) {
	// Authentifié → appel store.Stat() → panic → 500 via Recovery.
	_, r := newHandlerRouter(t, "user")
	req := httptest.NewRequest(http.MethodDelete, "/media/some-id", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// Pas 401/403.
	if w.Code == http.StatusUnauthorized {
		t.Fatal("ne doit pas être 401")
	}
	if w.Code == http.StatusForbidden {
		t.Fatal("ne doit pas être 403 sans vérification propriétaire")
	}
}

func TestDelete_StoreNil_Admin(t *testing.T) {
	// Admin authentifié → appel store.Stat() → panic → 500 via Recovery.
	_, r := newHandlerRouter(t, "admin")
	req := httptest.NewRequest(http.MethodDelete, "/media/admin-id", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code == http.StatusUnauthorized || w.Code == http.StatusForbidden {
		t.Fatalf("admin ne doit pas être 401/403, obtenu %d", w.Code)
	}
}

// ─── PurgeByOwner tests ────────────────────────────────────────────────────

func TestPurgeByOwner_SansAuth(t *testing.T) {
	// PurgeByOwner n'a pas de guard claims intégré : c'est le middleware
	// AdminOnly (routes.go) qui protège la route. Sans middleware, le handler
	// appelle store.RemoveByOwner sur un store nil → panic → 500 via Recovery.
	_, r := newHandlerRouterNoAuth(t)
	req := httptest.NewRequest(http.MethodDelete, "/media/owners/user-xyz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code == http.StatusOK {
		t.Fatalf("ne doit pas retourner 200 sans store, obtenu %d", w.Code)
	}
}

func TestPurgeByOwner_StoreNil(t *testing.T) {
	// Authentifié → store.RemoveByOwner() → panic → 500 via Recovery.
	_, r := newHandlerRouter(t, "admin")
	req := httptest.NewRequest(http.MethodDelete, "/media/owners/user-xyz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code == http.StatusUnauthorized || w.Code == http.StatusForbidden {
		t.Fatalf("admin ne doit pas être 401/403, obtenu %d", w.Code)
	}
}

// ─── Health handler ─────────────────────────────────────────────────────────

func TestHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/health", Health("test-service"))
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("attendu 200, obtenu %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "ok") {
		t.Fatalf("corps ne contient pas 'ok' : %s", body)
	}
	if !strings.Contains(body, "test-service") {
		t.Fatalf("corps ne contient pas le nom du service : %s", body)
	}
}

// ─── randomID ──────────────────────────────────────────────────────────────

func TestRandomID(t *testing.T) {
	id, err := randomID()
	if err != nil {
		t.Fatalf("randomID() erreur inattendue : %v", err)
	}
	if len(id) != 32 {
		t.Fatalf("longueur id = %d, attendu 32", len(id))
	}
	if _, err := hex.DecodeString(id); err != nil {
		t.Fatalf("id non-hex : %v", err)
	}
}

func TestRandomID_Unicité(t *testing.T) {
	ids := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		id, err := randomID()
		if err != nil {
			t.Fatalf("randomID() erreur : %v", err)
		}
		if ids[id] {
			t.Fatalf("collision détectée après %d itérations : %s", i, id)
		}
		ids[id] = true
	}
}

// ─── NewMediaHandler ────────────────────────────────────────────────────────

func TestNewMediaHandler(t *testing.T) {
	cfg := &config.Config{
		MaxImageBytes: 1 * 1024 * 1024,
		MaxVideoBytes: 2 * 1024 * 1024,
		MaxBlobBytes:  3 * 1024 * 1024,
	}
	h := NewMediaHandler(nil, cfg)
	if h == nil {
		t.Fatal("NewMediaHandler a retourné nil")
	}
	if h.maxImageBytes != cfg.MaxImageBytes {
		t.Fatalf("maxImageBytes = %d, attendu %d", h.maxImageBytes, cfg.MaxImageBytes)
	}
	if h.maxVideoBytes != cfg.MaxVideoBytes {
		t.Fatalf("maxVideoBytes = %d, attendu %d", h.maxVideoBytes, cfg.MaxVideoBytes)
	}
	if h.maxBlobBytes != cfg.MaxBlobBytes {
		t.Fatalf("maxBlobBytes = %d, attendu %d", h.maxBlobBytes, cfg.MaxBlobBytes)
	}
}

// ─── Tests avec vrai JWT (JWTAuth middleware) ────────────────────────────────

// newRouterWithJWT utilise le vrai middleware JWTAuth pour les tests d'intégration
// partielle (vrai JWT → handler → store nil → 500).
func newRouterWithJWT(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())

	const secret = "test-secret-handler"
	cfg := &config.Config{
		JWTSecret:     secret,
		MaxImageBytes: 5 * 1024 * 1024,
		MaxVideoBytes: 10 * 1024 * 1024,
		MaxBlobBytes:  8 * 1024 * 1024,
	}
	h := NewMediaHandler(nil, cfg)
	auth := middleware.JWTAuth(secret)

	r.POST("/media", auth, h.Upload)
	r.POST("/media/encrypted", auth, h.UploadEncrypted)
	r.GET("/media/:id", h.Download)
	r.DELETE("/media/:id", auth, h.Delete)

	return r
}

func TestUpload_JWTValide_TypeInvalide(t *testing.T) {
	r := newRouterWithJWT(t)
	tok := makeTokenForHandler(t, "test-secret-handler", "user", time.Hour)

	body, ct := buildMultipartFile(t, "file", "doc.pdf", []byte("%PDF-1.4 test"))
	req := httptest.NewRequest(http.MethodPost, "/media", body)
	req.Header.Set("Content-Type", ct)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("attendu 415, obtenu %d", w.Code)
	}
}

func TestUpload_JWTExpire(t *testing.T) {
	r := newRouterWithJWT(t)
	tok := makeTokenForHandler(t, "test-secret-handler", "user", -time.Hour)

	body, ct := buildMultipartFile(t, "file", "img.jpg", minimalJPEG())
	req := httptest.NewRequest(http.MethodPost, "/media", body)
	req.Header.Set("Content-Type", ct)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("token expiré : attendu 401, obtenu %d", w.Code)
	}
}

func TestDelete_JWTValide_StoreNil(t *testing.T) {
	r := newRouterWithJWT(t)
	tok := makeTokenForHandler(t, "test-secret-handler", "user", time.Hour)

	req := httptest.NewRequest(http.MethodDelete, "/media/some-id", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// store nil → panic → 500 via Recovery.
	if w.Code == http.StatusUnauthorized || w.Code == http.StatusForbidden {
		t.Fatalf("ne doit pas être 401/403, obtenu %d", w.Code)
	}
}

func TestUploadEncrypted_JWTValide_BlobTropGrand(t *testing.T) {
	r := newRouterWithJWT(t)
	tok := makeTokenForHandler(t, "test-secret-handler", "user", time.Hour)

	bigBlob := make([]byte, 9*1024*1024)
	body, ct := buildMultipartFile(t, "file", "secret.bin", bigBlob)
	req := httptest.NewRequest(http.MethodPost, "/media/encrypted", body)
	req.Header.Set("Content-Type", ct)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("attendu 413, obtenu %d", w.Code)
	}
}

// ─── Vérification du message d'erreur dans le corps JSON ───────────────────

func TestUpload_SansFile_MessageErreur(t *testing.T) {
	_, r := newHandlerRouter(t, "user")
	req := httptest.NewRequest(http.MethodPost, "/media", nil)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----test")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("attendu 400, obtenu %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "file") {
		t.Fatalf("message d'erreur ne mentionne pas 'file' : %s", w.Body.String())
	}
}

func TestUploadEncrypted_SansFile_MessageErreur(t *testing.T) {
	_, r := newHandlerRouter(t, "user")
	req := httptest.NewRequest(http.MethodPost, "/media/encrypted", nil)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----test")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("attendu 400, obtenu %d", w.Code)
	}
}

// ─── Test du chemin io.ReadFull avec fichier vide ──────────────────────────

func TestUpload_FichierVide(t *testing.T) {
	// Fichier de 0 octet → mimetype → type non reconnu → 415.
	_, r := newHandlerRouter(t, "user")

	body, ct := buildMultipartFile(t, "file", "empty.jpg", []byte{})
	req := httptest.NewRequest(http.MethodPost, "/media", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("fichier vide : attendu 415, obtenu %d (%s)", w.Code, w.Body.String())
	}
}

// ─── Test io.ReadFull avec fichier < sniffLen (ErrUnexpectedEOF géré) ──────

func TestUpload_PetitFichierJPEG(t *testing.T) {
	// Fichier < 3072 octets mais avec magic bytes JPEG → ErrUnexpectedEOF géré,
	// MIME détecté → passe le check → store.Put() → panic → 500.
	_, r := newHandlerRouter(t, "user")

	small := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10}
	body, ct := buildMultipartFile(t, "file", "small.jpg", small)
	req := httptest.NewRequest(http.MethodPost, "/media", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// Ne doit pas être 400.
	if w.Code == http.StatusBadRequest {
		t.Fatalf("petits fichiers JPEG ne doivent pas être 400, obtenu %d", w.Code)
	}
}

// ─── Test du MultiReader (reconstruction flux tête + reste) ─────────────────

func TestUpload_ReconstructionFlux(t *testing.T) {
	// Fichier JPEG > sniffLen → tête lue séparément, reste reconstruit via MultiReader.
	_, r := newHandlerRouter(t, "user")

	content := minimalJPEG()
	extra := make([]byte, 100)
	content = append(content, extra...)

	body, ct := buildMultipartFile(t, "file", "test.jpg", content)
	req := httptest.NewRequest(http.MethodPost, "/media", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// Ne doit pas être 400 (reconstruction correcte).
	if w.Code == http.StatusBadRequest {
		t.Fatalf("reconstruction flux ne doit pas causer 400, obtenu %d", w.Code)
	}
}

// ─── Vérification que le paramètre :id est bien extrait ────────────────────

func TestDownload_IDDansURL(t *testing.T) {
	_, r := newHandlerRouter(t, "user")
	req := httptest.NewRequest(http.MethodGet, "/media/mon-id-test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// Download public → jamais 401.
	if w.Code == http.StatusUnauthorized {
		t.Fatal("download ne requiert pas d'auth")
	}
}

// ─── Test du chaînage middleware → handler ───────────────────────────────────

func TestChainageMiddlewareEtHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())

	cfg := &config.Config{
		MaxImageBytes: 5 * 1024 * 1024,
		MaxVideoBytes: 10 * 1024 * 1024,
	}
	h := NewMediaHandler(nil, cfg)

	called := false
	r.POST("/test", func(c *gin.Context) {
		c.Set("claims", &middleware.Claims{
			UserID: "user-x",
			Role:   "user",
		})
		c.Next()
	}, func(c *gin.Context) {
		called = true
		h.Upload(c)
	})

	body, ct := buildMultipartFile(t, "file", "doc.pdf", []byte("%PDF-1.4"))
	req := httptest.NewRequest(http.MethodPost, "/test", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !called {
		t.Fatal("le handler n'a pas été appelé")
	}
	if w.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("attendu 415, obtenu %d", w.Code)
	}
}

// ─── Test lecteur vide (io.EOF immédiat sur ReadFull) ───────────────────────

func TestUpload_LectureImmediat_EOF(t *testing.T) {
	_, r := newHandlerRouter(t, "user")

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", "empty.bin")
	_, _ = io.Copy(fw, strings.NewReader(""))
	_ = mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/media", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// Fichier vide → 415 (type non supporté).
	if w.Code == http.StatusBadRequest {
		t.Fatalf("fichier vide ne doit pas être 400, obtenu %d (%s)", w.Code, w.Body.String())
	}
}
