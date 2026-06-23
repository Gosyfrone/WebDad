package handlers

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"

	"github.com/webdad/user-service/internal/middleware"
	"github.com/webdad/user-service/internal/repository"
	"github.com/webdad/user-service/internal/service"
	"github.com/webdad/user-service/internal/testutil"
)

// idTarget : cible distincte du claims.UserID (évite les courts-circuits
// self-follow/self-block et atteint donc la couche repo).
const idTarget = "99999999-0000-0000-0000-000000000099"

// authClaims : claims d'un utilisateur authentifié pour les appels directs.
var authClaims = middleware.Claims{
	UserID: "11111111-1111-1111-1111-111111111111",
	Email:  "u@breezy.dev",
	Role:   "admin",
}

// newCtx fabrique un contexte Gin isolé pour appeler un handler directement.
func newCtx(method, target, body string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, target, strings.NewReader(body))
	if body != "" {
		c.Request.Header.Set("Content-Type", "application/json")
	}
	return c, w
}

// closedHandler renvoie un handler dont le service est adossé à un pool FERMÉ :
// toute opération DB échoue → branche d'erreur 500 (respondUserError default).
func closedHandler(t *testing.T) *Handler {
	t.Helper()
	conn, err := sql.Open("postgres", "host=127.0.0.1 port=1 dbname=x sslmode=disable")
	if err != nil {
		t.Fatalf("sql.Open : %v", err)
	}
	_ = conn.Close()
	return New(service.New(repository.New(conn), 0))
}

// ─── Branche « claims absents » (inatteignable via le middleware JWT, mais
// défensive) : on appelle le handler sans poser de claims dans le contexte. ──

func TestHandlers_ClaimsAbsents_401(t *testing.T) {
	h := New(service.New(nil, 0))

	type call struct {
		name string
		fn   func(*gin.Context)
	}
	calls := []call{
		{"Create", h.Create},
		{"GetMe", h.GetMe},
		{"UpdateMe", h.UpdateMe},
		{"Follow", h.Follow},
		{"Unfollow", h.Unfollow},
		{"RemoveFollower", h.RemoveFollower},
		{"Block", h.Block},
		{"Unblock", h.Unblock},
		{"BlockedUsers", h.BlockedUsers},
		{"AcceptFollowRequest", h.AcceptFollowRequest},
		{"RejectFollowRequest", h.RejectFollowRequest},
		{"PendingFollowRequests", h.PendingFollowRequests},
	}
	for _, cc := range calls {
		c, w := newCtx(http.MethodGet, "/", "")
		cc.fn(c)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s sans claims = %d, attendu 401", cc.name, w.Code)
		}
	}
}

// ─── Branches d'erreur 500 (échec DB) sur les handlers authentifiés ──────────

func TestHandlers_DBError_500(t *testing.T) {
	h := closedHandler(t)
	claims := &authClaims

	type tc struct {
		name   string
		method string
		body   string
		params gin.Params
		fn     func(*gin.Context)
	}
	cases := []tc{
		{"GetMe", http.MethodGet, "", nil, h.GetMe},
		{"Unfollow", http.MethodDelete, "", gin.Params{{Key: "id", Value: idTarget}}, h.Unfollow},
		{"RemoveFollower", http.MethodDelete, "", gin.Params{{Key: "id", Value: idTarget}}, h.RemoveFollower},
		{"Block", http.MethodPost, "", gin.Params{{Key: "id", Value: idTarget}}, h.Block},
		{"Unblock", http.MethodDelete, "", gin.Params{{Key: "id", Value: idTarget}}, h.Unblock},
		{"BlockedUsers", http.MethodGet, "", nil, h.BlockedUsers},
		{"AcceptFollowRequest", http.MethodPost, "", gin.Params{{Key: "followerId", Value: idTarget}}, h.AcceptFollowRequest},
		{"RejectFollowRequest", http.MethodPost, "", gin.Params{{Key: "followerId", Value: idTarget}}, h.RejectFollowRequest},
		{"PendingFollowRequests", http.MethodGet, "", nil, h.PendingFollowRequests},
		{"Following", http.MethodGet, "", gin.Params{{Key: "id", Value: idTarget}}, h.Following},
		{"Followers", http.MethodGet, "", gin.Params{{Key: "id", Value: idTarget}}, h.Followers},
		{"Delete", http.MethodDelete, "", gin.Params{{Key: "id", Value: idTarget}}, h.Delete},
		{"PurgeUser", http.MethodDelete, "", gin.Params{{Key: "id", Value: idTarget}}, h.PurgeUser},
		{"SetStatus", http.MethodPatch, `{"is_active":true}`, gin.Params{{Key: "id", Value: idTarget}}, h.SetStatus},
		{"AcceptAllFollowRequests", http.MethodPost, "", gin.Params{{Key: "ownerId", Value: idTarget}}, h.AcceptAllFollowRequests},
	}
	for _, cse := range cases {
		c, w := newCtx(cse.method, "/", cse.body)
		c.Set("claims", claims)
		c.Params = cse.params
		cse.fn(c)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("%s (DB en erreur) = %d, attendu 500 — %s", cse.name, w.Code, w.Body.String())
		}
	}
}

// Handlers publics paginés + AdminCreate : branche d'erreur 500 (échec DB),
// non atteignable avec un repo nil (panic) — ici l'erreur est propre.
func TestHandlers_PublicAndAdmin_DBError_500(t *testing.T) {
	h := closedHandler(t)

	// Search avec un terme non vide (sinon court-circuit à 200).
	c, w := newCtx(http.MethodGet, "/?q=term", "")
	h.Search(c)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Search (DB erreur) = %d, attendu 500", w.Code)
	}

	c, w = newCtx(http.MethodGet, "/", "")
	h.List(c)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("List (DB erreur) = %d, attendu 500", w.Code)
	}

	c, w = newCtx(http.MethodGet, "/", "")
	h.Suggestions(c)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Suggestions (DB erreur) = %d, attendu 500", w.Code)
	}

	// AdminCreate : body valide, mais l'insertion échoue → 500.
	c, w = newCtx(http.MethodPost, "/", `{"id":"`+idTarget+`","username":"validname"}`)
	h.AdminCreate(c)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("AdminCreate (DB erreur) = %d, attendu 500 — %s", w.Code, w.Body.String())
	}
}

// paginate : limit hors borne (> maxLimit) → repli sur le défaut (branche
// dédiée). Search avec q vide court-circuite à 200 sans toucher le repo.
func TestHandlers_Paginate_LimitTropGrand(t *testing.T) {
	h := New(service.New(nil, 0))
	c, w := newCtx(http.MethodGet, "/?q=&limit=9999", "")
	h.Search(c)
	if w.Code != http.StatusOK {
		t.Fatalf("Search(limit=9999) = %d, attendu 200", w.Code)
	}
}

// respondUserError : branche 429 (cooldown) via UpdateMe sur une vraie base.
func TestHandlers_UpdateMe_Cooldown_429(t *testing.T) {
	db := testutil.DB(t)
	repo := repository.New(db)
	if _, err := repo.Create(authClaims.UserID, "alice"); err != nil {
		t.Fatalf("seed : %v", err)
	}
	svc := service.New(repo, time.Hour) // cooldown actif
	h := New(svc)

	// 1er changement : pose la baseline (autorisé).
	c, w := newCtx(http.MethodPatch, "/", `{"username":"alice1"}`)
	c.Set("claims", &authClaims)
	h.UpdateMe(c)
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateMe#1 = %d %s", w.Code, w.Body.String())
	}
	// 2e changement immédiat : cooldown → 429.
	c, w = newCtx(http.MethodPatch, "/", `{"username":"alice2"}`)
	c.Set("claims", &authClaims)
	h.UpdateMe(c)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("UpdateMe#2 = %d, attendu 429 — %s", w.Code, w.Body.String())
	}
}

// IsFollowing / HasBlocked : handlers internes sans enveloppe d'erreur
// (renvoient false sur échec DB) → 200 avec le booléen à false.
func TestHandlers_Internal_BoolOnDBError(t *testing.T) {
	h := closedHandler(t)

	c, w := newCtx(http.MethodGet, "/", "")
	c.Params = gin.Params{{Key: "userId", Value: idTarget}, {Key: "followingId", Value: authClaims.UserID}}
	h.IsFollowing(c)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "false") {
		t.Fatalf("IsFollowing = %d %s, attendu 200/false", w.Code, w.Body.String())
	}

	c, w = newCtx(http.MethodGet, "/", "")
	c.Params = gin.Params{{Key: "blockerId", Value: idTarget}, {Key: "blockedId", Value: authClaims.UserID}}
	h.HasBlocked(c)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "false") {
		t.Fatalf("HasBlocked = %d %s, attendu 200/false", w.Code, w.Body.String())
	}
}
