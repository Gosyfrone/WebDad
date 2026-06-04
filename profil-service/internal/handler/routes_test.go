package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/webdad/profil-service/internal/repository"
	"github.com/webdad/profil-service/internal/service"
)

// newTestRouter construit un routeur câblé sur un client Mongo NON connecté
// (mongo.Connect ne dialogue pas avec le serveur tant qu'aucune requête n'est
// émise). Les handlers du squelette renvoient 501 sans toucher la collection,
// donc aucune connexion réelle n'est nécessaire.
func newTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	client, err := mongo.Connect(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Fatalf("client mongo de test : %v", err)
	}
	db := client.Database("webdad_profil_test")
	r := gin.New()
	svc := service.New(repository.NewProfilRepository(db))
	RegisterRoutes(r, "profil-service", svc, "test-secret")
	return r
}

// TestRoutesRegister vérifie que toutes les routes s'enregistrent sans panic.
func TestRoutesRegister(t *testing.T) {
	r := newTestRouter(t)
	if len(r.Routes()) == 0 {
		t.Fatal("aucune route enregistrée")
	}
}

// TestHealthOK : /health répond 200 (route fonctionnelle, pas un stub).
func TestHealthOK(t *testing.T) {
	r := newTestRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /health = %d, attendu 200", w.Code)
	}
}

// TestPublicProfilStub : GET /profils/:userId (public) renvoie 501 au stade
// squelette (chaîne handler→service→repository câblée, corps non implémenté).
func TestPublicProfilStub(t *testing.T) {
	r := newTestRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/profils/abc", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotImplemented {
		t.Fatalf("GET /profils/:userId = %d, attendu 501", w.Code)
	}
}

// TestProtectedRequiresToken : /profils/me sans token = 401 (middleware JWT).
func TestProtectedRequiresToken(t *testing.T) {
	r := newTestRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/profils/me", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("GET /profils/me sans token = %d, attendu 401", w.Code)
	}
}
