package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/webdad/profil-service/internal/middleware"
	"github.com/webdad/profil-service/internal/models"
	"github.com/webdad/profil-service/internal/service"
)

type memoryProfilRepo struct {
	profils map[string]*models.Profil
	search  []models.Profil
	err     error
}

func newMemoryProfilRepo(profils ...*models.Profil) *memoryProfilRepo {
	repo := &memoryProfilRepo{profils: map[string]*models.Profil{}}
	for _, p := range profils {
		repo.profils[p.UserID] = cloneHandlerProfil(p)
	}
	return repo
}

func (m *memoryProfilRepo) GetByUserID(_ context.Context, userID string) (*models.Profil, error) {
	if m.err != nil {
		return nil, m.err
	}
	p, ok := m.profils[userID]
	if !ok {
		return nil, service.ErrProfilNotFound
	}
	return cloneHandlerProfil(p), nil
}

func (m *memoryProfilRepo) Insert(_ context.Context, p *models.Profil) error {
	m.profils[p.UserID] = cloneHandlerProfil(p)
	return m.err
}

func (m *memoryProfilRepo) SearchByDisplayName(_ context.Context, _ string, _ int64) ([]models.Profil, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.search, nil
}

func (m *memoryProfilRepo) Update(_ context.Context, userID string, set bson.M) (*models.Profil, error) {
	if m.err != nil {
		return nil, m.err
	}
	p, ok := m.profils[userID]
	if !ok {
		return nil, service.ErrProfilNotFound
	}
	applyHandlerSet(p, set)
	return cloneHandlerProfil(p), nil
}

func (m *memoryProfilRepo) Delete(_ context.Context, userID string) error {
	if m.err != nil {
		return m.err
	}
	if _, ok := m.profils[userID]; !ok {
		return service.ErrProfilNotFound
	}
	delete(m.profils, userID)
	return nil
}

func cloneHandlerProfil(p *models.Profil) *models.Profil {
	if p == nil {
		return nil
	}
	cp := *p
	return &cp
}

func applyHandlerSet(p *models.Profil, set bson.M) {
	if v, ok := set["display_name"].(string); ok {
		p.DisplayName = v
	}
	if v, ok := set["bio"].(string); ok {
		p.Bio = v
	}
	if v, ok := set["visibility"].(string); ok {
		p.Visibility = v
	}
	if v, ok := set["likes_visibility"].(string); ok {
		p.LikesVisibility = v
	}
	if v, ok := set["activity_visibility"].(string); ok {
		p.ActivityVisibility = v
	}
	if v, ok := set["certification"].(string); ok {
		p.Certification = v
	}
	if v, ok := set["is_online"].(bool); ok {
		p.IsOnline = v
	}
	if v, ok := set["last_login_at"].(time.Time); ok {
		p.LastLoginAt = &v
	}
	if v, ok := set["nsfw_enabled"].(bool); ok {
		p.NsfwEnabled = &v
	}
	if v, ok := set["updated_at"].(time.Time); ok {
		p.UpdatedAt = v
	}
}

func newSuccessRouter(t *testing.T, repo *memoryProfilRepo) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	RegisterRoutes(r, "profil-service", service.New(repo, 0), "test-secret")
	return r
}

func decodeBody(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json invalide: %v body=%s", err, w.Body.String())
	}
	return body
}

func TestHandlerSuccess_PublicReads(t *testing.T) {
	now := time.Now().UTC()
	nsfw := true
	repo := newMemoryProfilRepo(&models.Profil{
		UserID:             "u1",
		DisplayName:        "Alice",
		Visibility:         models.VisibilityPublic,
		LikesVisibility:    "",
		ActivityVisibility: models.VisibilityPublic,
		LastLoginAt:        &now,
		IsOnline:           true,
		NsfwEnabled:        &nsfw,
	})
	repo.search = []models.Profil{*repo.profils["u1"]}
	r := newSuccessRouter(t, repo)

	cases := []struct {
		method string
		path   string
		want   int
	}{
		{http.MethodGet, "/profils/search?q=Ali&limit=999", http.StatusOK},
		{http.MethodGet, "/profils/u1", http.StatusOK},
		{http.MethodGet, "/profils/u1/activity", http.StatusOK},
		{http.MethodGet, "/profils/u1/visibility", http.StatusOK},
		{http.MethodGet, "/profils/u1/likes-visibility", http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, nil)
			r.ServeHTTP(w, req)
			if w.Code != tc.want {
				t.Fatalf("%s %s = %d body=%s", tc.method, tc.path, w.Code, w.Body.String())
			}
		})
	}
}

func TestSearch_RepositoryError(t *testing.T) {
	repo := newMemoryProfilRepo()
	repo.err = errors.New("search down")
	r := newSuccessRouter(t, repo)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/profils/search?q=Alice&limit=10", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("Search erreur repo = %d body=%s", w.Code, w.Body.String())
	}
}

func TestHandlerSuccess_PrivateOwnerRoutes(t *testing.T) {
	birthDate := time.Now().UTC().AddDate(-20, 0, 0)
	repo := newMemoryProfilRepo(&models.Profil{
		UserID:             "11111111-1111-1111-1111-111111111111",
		DisplayName:        "Alice",
		BirthDate:          &birthDate,
		Visibility:         models.VisibilityPublic,
		LikesVisibility:    models.VisibilityPublic,
		ActivityVisibility: models.VisibilityPublic,
	})
	r := newSuccessRouter(t, repo)
	token := makeProfilToken(t, models.RoleUser)

	cases := []struct {
		method string
		path   string
		body   string
		want   int
	}{
		{http.MethodGet, "/profils/me", "", http.StatusOK},
		{http.MethodPatch, "/profils/me", `{"bio":"Nouvelle bio"}`, http.StatusOK},
		{http.MethodPatch, "/profils/me/activity", "", http.StatusOK},
		{http.MethodPatch, "/profils/me/activity/offline", "", http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Authorization", "Bearer "+token)
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			r.ServeHTTP(w, req)
			if w.Code != tc.want {
				t.Fatalf("%s %s = %d body=%s", tc.method, tc.path, w.Code, w.Body.String())
			}
		})
	}
}

func TestHandlerProtectedMethodsWithoutClaims(t *testing.T) {
	repo := newMemoryProfilRepo()
	h := NewProfilHandler(service.New(repo, 0))

	cases := []struct {
		name string
		call func(*gin.Context)
		body string
	}{
		{name: "GetMe", call: h.GetMe},
		{name: "UpdateMe", call: h.UpdateMe, body: `{}`},
		{name: "TouchActivity", call: h.TouchActivity},
		{name: "TouchActivityOffline", call: h.TouchActivityOffline},
		{name: "Create", call: h.Create, body: `{"display_name":"Alice"}`},
		{name: "AdminCreate", call: h.AdminCreate, body: `{"id":"u2","display_name":"Bob"}`},
		{name: "Delete", call: h.Delete},
		{name: "UpdateCertification", call: h.UpdateCertification, body: `{"certification":"political"}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			body := strings.NewReader(tc.body)
			c.Request = httptest.NewRequest(http.MethodPost, "/", body)
			if tc.body != "" {
				c.Request.Header.Set("Content-Type", "application/json")
			}

			tc.call(c)
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestHandlerSuccess_CreateAdminCreateAndDelete(t *testing.T) {
	repo := newMemoryProfilRepo()
	r := newSuccessRouter(t, repo)
	userToken := makeProfilToken(t, models.RoleUser)
	adminToken := makeProfilToken(t, models.RoleAdmin)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/profils", strings.NewReader(`{"display_name":"Alice"}`))
	req.Header.Set("Authorization", "Bearer "+userToken)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("Create = %d body=%s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/profils/admin", strings.NewReader(`{"id":"admin-created","display_name":"Bob"}`))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("AdminCreate = %d body=%s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/profils/admin-created", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("Delete = %d body=%s", w.Code, w.Body.String())
	}
}

func TestHandlerSuccess_UpdateCertification(t *testing.T) {
	repo := newMemoryProfilRepo(&models.Profil{
		UserID:        "target",
		DisplayName:   "Alice",
		Certification: models.CertificationNone,
	})
	r := newSuccessRouter(t, repo)
	modToken := makeProfilToken(t, models.RoleModerator)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/profils/target/certification", strings.NewReader(`{"certification":"political"}`))
	req.Header.Set("Authorization", "Bearer "+modToken)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateCertification = %d body=%s", w.Code, w.Body.String())
	}
	if repo.profils["target"].Certification != models.CertificationPolitical {
		t.Fatalf("certification non persistee: %#v", repo.profils["target"])
	}
	body := decodeBody(t, w)
	data := body["data"].(map[string]any)
	if data["certification"] != models.CertificationPolitical {
		t.Fatalf("certification reponse = %#v", data["certification"])
	}
}

func TestUpdateCertification_InvalidJSON_400(t *testing.T) {
	repo := newMemoryProfilRepo(&models.Profil{UserID: "target"})
	r := newSuccessRouter(t, repo)
	token := makeProfilToken(t, models.RoleModerator)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/profils/target/certification", strings.NewReader(`{"certification":"gold"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("UpdateCertification payload invalide = %d body=%s", w.Code, w.Body.String())
	}
}

func TestCreate_JSONInvalide_400(t *testing.T) {
	repo := newMemoryProfilRepo()
	r := newSuccessRouter(t, repo)
	token := makeProfilToken(t, models.RoleUser)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/profils", strings.NewReader(`{invalid}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Create JSON invalide = %d body=%s", w.Code, w.Body.String())
	}
}

func TestDelete_NotFound_404(t *testing.T) {
	repo := newMemoryProfilRepo()
	r := newSuccessRouter(t, repo)
	token := makeProfilToken(t, models.RoleAdmin)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/profils/missing", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("Delete absent = %d body=%s", w.Code, w.Body.String())
	}
}

func TestHandlerErrors(t *testing.T) {
	repo := newMemoryProfilRepo(&models.Profil{
		UserID:      "11111111-1111-1111-1111-111111111111",
		DisplayName: "Alice",
	})
	repo.err = errors.New("db down")
	r := newSuccessRouter(t, repo)
	token := makeProfilToken(t, models.RoleAdmin)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/profils/u1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("Delete erreur interne = %d body=%s", w.Code, w.Body.String())
	}

	body := decodeBody(t, w)
	if body["error"] != "erreur interne" {
		t.Fatalf("body erreur inattendu: %#v", body)
	}
}

func TestHandlerNotFoundRoutes(t *testing.T) {
	repo := newMemoryProfilRepo()
	r := newSuccessRouter(t, repo)
	token := makeProfilToken(t, models.RoleUser)

	cases := []struct {
		method string
		path   string
		token  bool
		body   string
	}{
		{http.MethodGet, "/profils/missing", false, ""},
		{http.MethodGet, "/profils/missing/activity", false, ""},
		{http.MethodGet, "/profils/missing/visibility", false, ""},
		{http.MethodGet, "/profils/missing/likes-visibility", false, ""},
		{http.MethodGet, "/profils/me", true, ""},
		{http.MethodPatch, "/profils/me", true, `{"bio":"x"}`},
		{http.MethodPatch, "/profils/me/activity", true, ""},
		{http.MethodPatch, "/profils/me/activity/offline", true, ""},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			if tc.token {
				req.Header.Set("Authorization", "Bearer "+token)
			}
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			r.ServeHTTP(w, req)
			if w.Code != http.StatusNotFound {
				t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestHandlerForbiddenAdminRoutes(t *testing.T) {
	repo := newMemoryProfilRepo()
	r := newSuccessRouter(t, repo)
	userToken := makeProfilToken(t, models.RoleUser)

	cases := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPost, "/profils/admin", `{"id":"u2","display_name":"Bob"}`},
		{http.MethodDelete, "/profils/u2", ""},
		{http.MethodPatch, "/profils/u2/certification", `{"certification":"political"}`},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Authorization", "Bearer "+userToken)
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			r.ServeHTTP(w, req)
			if w.Code != http.StatusForbidden {
				t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestRespondProfilErrorBranches(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{name: "not found", err: service.ErrProfilNotFound, want: http.StatusNotFound},
		{name: "exists", err: service.ErrProfilExists, want: http.StatusConflict},
		{name: "birth date", err: service.ErrBirthDateLocked, want: http.StatusConflict},
		{name: "cooldown", err: service.ErrDisplayNameCooldown, want: http.StatusTooManyRequests},
		{name: "invalid display", err: service.ErrInvalidDisplayName, want: http.StatusBadRequest},
		{name: "invalid certification", err: service.ErrInvalidCertification, want: http.StatusBadRequest},
		{name: "internal", err: errors.New("boom"), want: http.StatusInternalServerError},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			respondProfilError(c, tc.err)
			if w.Code != tc.want {
				t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestSanitizePublicProfil(t *testing.T) {
	repo := newMemoryProfilRepo()
	h := NewProfilHandler(service.New(repo, 0))
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	h.sanitizePublicProfil(c, nil)

	now := time.Now().UTC()
	nsfw := true
	p := &models.Profil{
		UserID:             "u1",
		Visibility:         models.VisibilityPrivate,
		ActivityVisibility: models.VisibilityPublic,
		LastLoginAt:        &now,
		IsOnline:           true,
		NsfwEnabled:        &nsfw,
	}
	h.sanitizePublicProfil(c, p)
	if p.NsfwEnabled != nil {
		t.Fatal("nsfw_enabled doit etre masque")
	}
	if p.LastLoginAt != nil || p.IsOnline {
		t.Fatalf("activite privee non masquee: %#v", p)
	}
}

func TestSanitizePublicProfil_OwnerCanSeePrivateActivity(t *testing.T) {
	repo := newMemoryProfilRepo()
	h := NewProfilHandler(service.New(repo, 0))
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set("claims", &middleware.Claims{UserID: "u1", Role: models.RoleUser})

	now := time.Now().UTC()
	nsfw := true
	p := &models.Profil{
		UserID:             "u1",
		Visibility:         models.VisibilityPrivate,
		ActivityVisibility: models.VisibilityPublic,
		LastLoginAt:        &now,
		IsOnline:           true,
		NsfwEnabled:        &nsfw,
	}

	h.sanitizePublicProfil(c, p)
	if p.NsfwEnabled != nil {
		t.Fatal("nsfw_enabled doit etre masque")
	}
	if p.LastLoginAt == nil || !p.IsOnline {
		t.Fatalf("le proprietaire doit voir son activite: %#v", p)
	}
}
