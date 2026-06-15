package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/webdad/post-service/internal/realtime"
	"github.com/webdad/post-service/internal/repository"
	"github.com/webdad/post-service/internal/service"
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
	db := client.Database("webdad_post_test")
	r := gin.New()
	svc := service.NewPostService(
		repository.NewPostRepository(db),
		service.WithBookmarkWindow(5*time.Minute),
	)
	RegisterRoutes(r, "post-service", svc, "test-secret", realtime.NewHub(), nil)
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

// TestProtectedRequiresToken : les routes mutables exigent un JWT (le
// middleware abort en 401 sans toucher la base).
func TestProtectedRequiresToken(t *testing.T) {
	cases := []struct {
		method, path string
	}{
		{http.MethodPost, "/posts"},
		{http.MethodPatch, "/posts/507f1f77bcf86cd799439011"},
		{http.MethodDelete, "/posts/507f1f77bcf86cd799439011"},
		{http.MethodPatch, "/posts/507f1f77bcf86cd799439011/pin"},
		{http.MethodDelete, "/posts/507f1f77bcf86cd799439011/pin"},
		{http.MethodPost, "/posts/507f1f77bcf86cd799439011/like"},
		{http.MethodDelete, "/posts/507f1f77bcf86cd799439011/like"},
		{http.MethodPost, "/posts/507f1f77bcf86cd799439011/comments"},
		{http.MethodDelete, "/posts/507f1f77bcf86cd799439011/comments/507f1f77bcf86cd799439012"},
		{http.MethodGet, "/posts/me/liked-ids"},
		{http.MethodGet, "/posts/me/bookmarked-ids"},
		{http.MethodGet, "/posts/bookmarks"},
		{http.MethodGet, "/posts/bookmarks/collections"},
		{http.MethodPost, "/posts/bookmarks/collections"},
		{http.MethodPatch, "/posts/bookmarks/collections/507f1f77bcf86cd799439011"},
		{http.MethodDelete, "/posts/bookmarks/collections/507f1f77bcf86cd799439011"},
		{http.MethodGet, "/posts/bookmarks/collections/507f1f77bcf86cd799439011/posts"},
		{http.MethodPost, "/posts/507f1f77bcf86cd799439011/bookmark"},
		{http.MethodDelete, "/posts/507f1f77bcf86cd799439011/bookmark"},
		{http.MethodGet, "/posts/507f1f77bcf86cd799439011/bookmark/collections"},
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

// TestSplitIDs : découpe une liste d'ids, ignore les segments vides/espaces.
func TestSplitIDs(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"a", 1},
		{"a,b,c", 3},
		{"a,,b, ,c", 3},
		{" a , b ", 2},
	}
	for _, tc := range cases {
		if got := len(splitIDs(tc.in)); got != tc.want {
			t.Fatalf("splitIDs(%q) = %d ids, attendu %d", tc.in, got, tc.want)
		}
	}
}
