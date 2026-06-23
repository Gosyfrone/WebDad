package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/webdad/user-service/internal/client"
	"github.com/webdad/user-service/internal/middleware"
	"github.com/webdad/user-service/internal/repository"
	"github.com/webdad/user-service/internal/router"
	"github.com/webdad/user-service/internal/service"
	"github.com/webdad/user-service/internal/testutil"
)

const itSecret = "integration-test-secret"

const (
	idAlice = "aaaaaaaa-2222-0000-0000-000000000001"
	idBob   = "bbbbbbbb-2222-0000-0000-000000000002"
	idCarol = "cccccccc-2222-0000-0000-000000000003"
)

// fakeProfil simule profil-service (visibilité fixe).
type fakeProfil struct{ vis string }

func (f fakeProfil) Visibility(_ context.Context, _ string) (string, error) { return f.vis, nil }

// fakeNotifier ignore les événements (best-effort).
type fakeNotifier struct{}

func (fakeNotifier) Emit(_ client.Event) {}

// setup construit un routeur complet adossé à une base fraîche + fakes.
// vis pilote la visibilité renvoyée par le faux profil-service.
func setup(t *testing.T, vis string) (*gin.Engine, *repository.UserRepository) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testutil.DB(t)
	repo := repository.New(db)
	svc := service.New(repo, 0,
		service.WithProfilClient(fakeProfil{vis: vis}),
		service.WithNotificationClient(fakeNotifier{}),
	)
	return router.New(svc, itSecret), repo
}

func token(t *testing.T, userID, role string) string {
	t.Helper()
	now := time.Now()
	claims := middleware.Claims{
		UserID: userID,
		Email:  "u@breezy.dev",
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(itSecret))
	if err != nil {
		t.Fatalf("token : %v", err)
	}
	return tok
}

// do exécute une requête HTTP et renvoie le recorder.
func do(t *testing.T, r *gin.Engine, method, path, body, tok string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func seed(t *testing.T, repo *repository.UserRepository, id, username string) {
	t.Helper()
	if _, err := repo.Create(id, username); err != nil {
		t.Fatalf("seed %s : %v", username, err)
	}
}

func TestHTTP_Health(t *testing.T) {
	r, _ := setup(t, "public")
	w := do(t, r, http.MethodGet, "/health", "", "")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"ok"`) {
		t.Fatalf("GET /health = %d %s", w.Code, w.Body.String())
	}
}

func TestHTTP_Create_GetMe_UpdateMe(t *testing.T) {
	r, _ := setup(t, "public")
	tok := token(t, idAlice, "user")

	// Create.
	if w := do(t, r, http.MethodPost, "/users", `{"username":"alice"}`, tok); w.Code != http.StatusCreated {
		t.Fatalf("POST /users = %d %s", w.Code, w.Body.String())
	}
	// Doublon → 409.
	tokBob := token(t, idBob, "user")
	if w := do(t, r, http.MethodPost, "/users", `{"username":"alice"}`, tokBob); w.Code != http.StatusConflict {
		t.Fatalf("POST /users doublon = %d, attendu 409", w.Code)
	}
	// GetMe (alice existe déjà).
	if w := do(t, r, http.MethodGet, "/users/me", "", tok); w.Code != http.StatusOK {
		t.Fatalf("GET /users/me = %d %s", w.Code, w.Body.String())
	}
	// GetMe provisioning paresseux pour un nouvel utilisateur.
	tokCarol := token(t, idCarol, "user")
	if w := do(t, r, http.MethodGet, "/users/me", "", tokCarol); w.Code != http.StatusOK {
		t.Fatalf("GET /users/me (provision) = %d %s", w.Code, w.Body.String())
	}
	// UpdateMe : change username + locale.
	if w := do(t, r, http.MethodPatch, "/users/me", `{"username":"alice2","preferred_locale":"fr"}`, tok); w.Code != http.StatusOK {
		t.Fatalf("PATCH /users/me = %d %s", w.Code, w.Body.String())
	}
	// UpdateMe : locale invalide → 400.
	if w := do(t, r, http.MethodPatch, "/users/me", `{"preferred_locale":"xx"}`, tok); w.Code != http.StatusBadRequest {
		t.Fatalf("PATCH /users/me locale invalide = %d, attendu 400", w.Code)
	}
}

func TestHTTP_AdminCreate(t *testing.T) {
	r, _ := setup(t, "public")
	admin := token(t, idAlice, "admin")
	body := `{"id":"` + idBob + `","username":"bob"}`
	if w := do(t, r, http.MethodPost, "/users/admin", body, admin); w.Code != http.StatusCreated {
		t.Fatalf("POST /users/admin = %d %s", w.Code, w.Body.String())
	}
	// Rôle user → 403.
	user := token(t, idCarol, "user")
	if w := do(t, r, http.MethodPost, "/users/admin", body, user); w.Code != http.StatusForbidden {
		t.Fatalf("POST /users/admin (user) = %d, attendu 403", w.Code)
	}
}

func TestHTTP_PublicReads(t *testing.T) {
	r, repo := setup(t, "public")
	seed(t, repo, idAlice, "alice")
	seed(t, repo, idBob, "bob")

	cases := []struct {
		path string
		code int
	}{
		{"/users/search?q=ali", http.StatusOK},
		{"/users/suggestions", http.StatusOK},
		{"/users/by-username/alice", http.StatusOK},
		{"/users/by-username/inconnu", http.StatusNotFound},
		{"/users/" + idAlice, http.StatusOK},
		{"/users/not-a-uuid", http.StatusNotFound},
		{"/users/" + idCarol, http.StatusNotFound},
		{"/users/" + idAlice + "/followers", http.StatusOK},
		{"/users/" + idAlice + "/following", http.StatusOK},
		{"/users/" + idCarol + "/followers", http.StatusNotFound},
	}
	for _, c := range cases {
		if w := do(t, r, http.MethodGet, c.path, "", ""); w.Code != c.code {
			t.Errorf("GET %s = %d, attendu %d (%s)", c.path, w.Code, c.code, w.Body.String())
		}
	}

	// GET /users (liste) exige une session.
	if w := do(t, r, http.MethodGet, "/users", "", ""); w.Code != http.StatusUnauthorized {
		t.Errorf("GET /users sans token = %d, attendu 401", w.Code)
	}
	if w := do(t, r, http.MethodGet, "/users", "", token(t, idAlice, "user")); w.Code != http.StatusOK {
		t.Errorf("GET /users avec token = %d, attendu 200", w.Code)
	}
}

func TestHTTP_FollowPublic_UnfollowAndLists(t *testing.T) {
	r, repo := setup(t, "public")
	seed(t, repo, idAlice, "alice")
	seed(t, repo, idBob, "bob")
	tok := token(t, idAlice, "user")

	// alice suit bob (profil public → following direct).
	w := do(t, r, http.MethodPost, "/users/"+idBob+"/follow", "", tok)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "following") {
		t.Fatalf("POST follow = %d %s", w.Code, w.Body.String())
	}

	// is-following interne.
	w = do(t, r, http.MethodGet, "/internal/"+idAlice+"/is-following/"+idBob, "", "")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "true") {
		t.Fatalf("is-following = %d %s", w.Code, w.Body.String())
	}

	// bob retire alice de ses abonnés.
	tokBob := token(t, idBob, "user")
	if w := do(t, r, http.MethodDelete, "/users/me/followers/"+idAlice, "", tokBob); w.Code != http.StatusNoContent {
		t.Fatalf("DELETE followers = %d %s", w.Code, w.Body.String())
	}

	// re-follow puis unfollow.
	do(t, r, http.MethodPost, "/users/"+idBob+"/follow", "", tok)
	if w := do(t, r, http.MethodDelete, "/users/"+idBob+"/follow", "", tok); w.Code != http.StatusNoContent {
		t.Fatalf("DELETE follow = %d %s", w.Code, w.Body.String())
	}

	// self-follow → 400.
	if w := do(t, r, http.MethodPost, "/users/"+idAlice+"/follow", "", tok); w.Code != http.StatusBadRequest {
		t.Fatalf("self-follow = %d, attendu 400", w.Code)
	}
}

func TestHTTP_FollowPrivate_AcceptReject_Pending(t *testing.T) {
	r, repo := setup(t, client.VisibilityPrivate)
	seed(t, repo, idAlice, "alice")
	seed(t, repo, idBob, "bob") // bob privé
	tokAlice := token(t, idAlice, "user")
	tokBob := token(t, idBob, "user")

	// alice demande à suivre bob (privé) → pending.
	w := do(t, r, http.MethodPost, "/users/"+idBob+"/follow", "", tokAlice)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "pending") {
		t.Fatalf("follow privé = %d %s", w.Code, w.Body.String())
	}

	// demandes sortantes d'alice.
	w = do(t, r, http.MethodGet, "/users/me/follow-requests/outgoing", "", tokAlice)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), idBob) {
		t.Fatalf("outgoing = %d %s", w.Code, w.Body.String())
	}

	// bob accepte.
	if w := do(t, r, http.MethodPost, "/users/follow-requests/"+idAlice+"/accept", "", tokBob); w.Code != http.StatusOK {
		t.Fatalf("accept = %d %s", w.Code, w.Body.String())
	}
	// Ré-accepter une demande inexistante → 404.
	if w := do(t, r, http.MethodPost, "/users/follow-requests/"+idAlice+"/accept", "", tokBob); w.Code != http.StatusNotFound {
		t.Fatalf("accept absent = %d, attendu 404", w.Code)
	}

	// Nouvelle demande puis rejet.
	do(t, r, http.MethodDelete, "/users/"+idBob+"/follow", "", tokAlice)
	do(t, r, http.MethodPost, "/users/"+idBob+"/follow", "", tokAlice)
	if w := do(t, r, http.MethodPost, "/users/follow-requests/"+idAlice+"/reject", "", tokBob); w.Code != http.StatusOK {
		t.Fatalf("reject = %d %s", w.Code, w.Body.String())
	}
}

func TestHTTP_Blocks(t *testing.T) {
	r, repo := setup(t, "public")
	seed(t, repo, idAlice, "alice")
	seed(t, repo, idBob, "bob")
	tok := token(t, idAlice, "user")

	if w := do(t, r, http.MethodPost, "/users/"+idBob+"/block", "", tok); w.Code != http.StatusOK {
		t.Fatalf("block = %d %s", w.Code, w.Body.String())
	}
	// mes blocages.
	w := do(t, r, http.MethodGet, "/users/me/blocks", "", tok)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), idBob) {
		t.Fatalf("GET blocks = %d %s", w.Code, w.Body.String())
	}
	// has-blocked interne.
	w = do(t, r, http.MethodGet, "/internal/users/"+idAlice+"/has-blocked/"+idBob, "", "")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "true") {
		t.Fatalf("has-blocked = %d %s", w.Code, w.Body.String())
	}
	// unblock.
	if w := do(t, r, http.MethodDelete, "/users/"+idBob+"/block", "", tok); w.Code != http.StatusNoContent {
		t.Fatalf("unblock = %d %s", w.Code, w.Body.String())
	}
	// self-block → 400.
	if w := do(t, r, http.MethodPost, "/users/"+idAlice+"/block", "", tok); w.Code != http.StatusBadRequest {
		t.Fatalf("self-block = %d, attendu 400", w.Code)
	}
}

func TestHTTP_Admin_SetStatus_Delete_Purge(t *testing.T) {
	r, repo := setup(t, "public")
	seed(t, repo, idAlice, "admin_acc")
	seed(t, repo, idBob, "target")
	admin := token(t, idAlice, "admin")
	mod := token(t, idAlice, "moderator")
	user := token(t, idBob, "user")

	// SetStatus (modération) : désactive.
	if w := do(t, r, http.MethodPatch, "/users/"+idBob+"/status", `{"is_active":false}`, mod); w.Code != http.StatusOK {
		t.Fatalf("PATCH status (mod) = %d %s", w.Code, w.Body.String())
	}
	// user → 403.
	if w := do(t, r, http.MethodPatch, "/users/"+idBob+"/status", `{"is_active":true}`, user); w.Code != http.StatusForbidden {
		t.Fatalf("PATCH status (user) = %d, attendu 403", w.Code)
	}
	// SetStatus sur un compte inexistant → 404.
	if w := do(t, r, http.MethodPatch, "/users/"+idCarol+"/status", `{"is_active":true}`, admin); w.Code != http.StatusNotFound {
		t.Fatalf("PATCH status (absent) = %d, attendu 404", w.Code)
	}

	// Delete (soft) admin.
	if w := do(t, r, http.MethodDelete, "/users/"+idBob, "", admin); w.Code != http.StatusNoContent {
		t.Fatalf("DELETE user = %d %s", w.Code, w.Body.String())
	}
	// Delete par un user → 403.
	if w := do(t, r, http.MethodDelete, "/users/"+idBob, "", user); w.Code != http.StatusForbidden {
		t.Fatalf("DELETE user (user) = %d, attendu 403", w.Code)
	}
	// Purge RGPD admin.
	if w := do(t, r, http.MethodDelete, "/users/"+idBob+"/hard", "", admin); w.Code != http.StatusNoContent {
		t.Fatalf("DELETE hard = %d %s", w.Code, w.Body.String())
	}
}

func TestHTTP_AcceptAllFollowRequests_Internal(t *testing.T) {
	r, repo := setup(t, client.VisibilityPrivate)
	seed(t, repo, idAlice, "owner")
	seed(t, repo, idBob, "fan1")
	seed(t, repo, idCarol, "fan2")

	// fan1 et fan2 demandent à suivre owner (privé).
	do(t, r, http.MethodPost, "/users/"+idAlice+"/follow", "", token(t, idBob, "user"))
	do(t, r, http.MethodPost, "/users/"+idAlice+"/follow", "", token(t, idCarol, "user"))

	// Acceptation en masse (appel interne profil→user).
	w := do(t, r, http.MethodPost, "/internal/users/"+idAlice+"/accept-all-follow-requests", "", "")
	if w.Code != http.StatusOK {
		t.Fatalf("accept-all = %d %s", w.Code, w.Body.String())
	}

	// Les deux suivent désormais owner.
	w = do(t, r, http.MethodGet, "/internal/"+idBob+"/is-following/"+idAlice, "", "")
	if !strings.Contains(w.Body.String(), "true") {
		t.Fatalf("fan1 devrait suivre owner : %s", w.Body.String())
	}
}

func TestHTTP_AuthRequired(t *testing.T) {
	r, _ := setup(t, "public")
	protected := []struct {
		method, path string
	}{
		{http.MethodPost, "/users"},
		{http.MethodGet, "/users/me"},
		{http.MethodPatch, "/users/me"},
		{http.MethodGet, "/users/me/blocks"},
		{http.MethodGet, "/users/me/follow-requests/outgoing"},
		{http.MethodPost, "/users/" + idBob + "/follow"},
		{http.MethodPost, "/users/" + idBob + "/block"},
	}
	for _, p := range protected {
		if w := do(t, r, p.method, p.path, "", ""); w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s sans token = %d, attendu 401", p.method, p.path, w.Code)
		}
	}
}

// Vérifie la forme {"data": ...} d'une réponse de détail.
func TestHTTP_GetByID_Envelope(t *testing.T) {
	r, repo := setup(t, "public")
	seed(t, repo, idAlice, "alice")
	w := do(t, r, http.MethodGet, "/users/"+idAlice, "", "")
	var body struct {
		Data struct {
			ID       string `json:"id"`
			Username string `json:"username"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("réponse non JSON : %v", err)
	}
	if body.Data.ID != idAlice || body.Data.Username != "alice" {
		t.Fatalf("enveloppe data inattendue : %+v", body)
	}
}
