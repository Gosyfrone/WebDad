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

// newTestRouter construit un routeur câblé sur un client Mongo NON connecté.
// On ne teste ici que l'enregistrement des routes et le middleware JWT : les
// routes protégées sont rejetées (401) AVANT d'atteindre la collection, donc
// aucune connexion réelle n'est nécessaire. (La logique métier est couverte
// par les tests unitaires purs du package service.)
func newTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	client, err := mongo.Connect(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Fatalf("client mongo de test : %v", err)
	}
	db := client.Database("webdad_profil_test")
	r := gin.New()
	svc := service.New(repository.NewProfilRepository(db), 0)
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

// TestProtectedRequiresToken : les routes personnelles/mutables exigent un JWT
// (le middleware abort en 401 sans toucher la base).
func TestProtectedRequiresToken(t *testing.T) {
	cases := []struct {
		method, path string
	}{
		{http.MethodGet, "/profils/me"},
		{http.MethodPatch, "/profils/me"},
		{http.MethodPost, "/profils"},
		{http.MethodDelete, "/profils/abc"},
	}
	r := newTestRouter(t)
	for _, tc := range cases {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.path, nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s sans token = %d, attendu 401", tc.method, tc.path, w.Code)
		}
	}
}
