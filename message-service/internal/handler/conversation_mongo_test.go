package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/webdad/message-service/internal/database"
	"github.com/webdad/message-service/internal/middleware"
	"github.com/webdad/message-service/internal/models"
	"github.com/webdad/message-service/internal/mongotest"
	"github.com/webdad/message-service/internal/realtime"
	"github.com/webdad/message-service/internal/repository"
	"github.com/webdad/message-service/internal/service"
)

// liveEnv assemble un routeur réel branché sur une Mongo jetable, plus de quoi
// fabriquer des tokens. Couvre les chemins SUCCÈS des handlers.
type liveEnv struct {
	router *gin.Engine
	svc    *service.MessageService
	repo   *repository.MessageRepository
	ctx    context.Context
}

func newLiveEnv(t *testing.T) *liveEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := mongotest.DB(t)
	ctx := context.Background()
	if err := database.EnsureSchema(ctx, db); err != nil {
		t.Fatalf("EnsureSchema : %v", err)
	}
	repo := repository.NewMessageRepository(db)
	svc := service.NewMessageService(repo)

	r := gin.New()
	r.Use(gin.Recovery())
	RegisterRoutes(r, "message-test", svc, realtime.NewHub(), msgTestSecret, []string{"http://localhost:3000"})
	return &liveEnv{router: r, svc: svc, repo: repo, ctx: ctx}
}

// token forge un JWT pour un user donné (vérifié, rôle user par défaut).
func (e *liveEnv) token(t *testing.T, userID, role string) string {
	t.Helper()
	now := time.Now()
	claims := middleware.Claims{
		UserID: userID, Email: userID + "@breezy.dev", EmailVerified: true, Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(msgTestSecret))
	if err != nil {
		t.Fatalf("token : %v", err)
	}
	return tok
}

// do exécute une requête authentifiée et renvoie le recorder.
func (e *liveEnv) do(t *testing.T, method, path, tok string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var rdr *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, rdr)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return w
}

// dataID extrait data.id d'une réponse JSON.
func dataID(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var resp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal (%s) : %v", w.Body.String(), err)
	}
	return resp.Data.ID
}

const (
	hA = "11111111-1111-1111-1111-111111111111" // = msgTestUserID
	hB = "22222222-2222-2222-2222-222222222222"
	hC = "33333333-3333-3333-3333-333333333333"
)

func envM(ids ...string) map[string]string {
	m := make(map[string]string, len(ids))
	for _, id := range ids {
		m[id] = "env-" + id
	}
	return m
}

// Parcours complet : clés, DM, message, édition, lecture, membres, communauté.
func TestHandler_FullFlow(t *testing.T) {
	e := newLiveEnv(t)
	tokA := e.token(t, hA, "user")
	tokB := e.token(t, hB, "user")

	// PublishKey + GetKey.
	if w := e.do(t, http.MethodPut, "/messages/keys", tokA, models.PublishKeyRequest{PublicKey: "pkA"}); w.Code != http.StatusOK {
		t.Fatalf("PublishKey = %d (%s)", w.Code, w.Body.String())
	}
	if w := e.do(t, http.MethodGet, "/messages/keys/"+hA, tokB, nil); w.Code != http.StatusOK {
		t.Fatalf("GetKey = %d", w.Code)
	}

	// Backup : status (false) → put → get → status (true).
	if w := e.do(t, http.MethodGet, "/messages/keys/backup/status", tokA, nil); w.Code != http.StatusOK {
		t.Fatalf("BackupStatus = %d", w.Code)
	}
	backup := models.PutBackupRequest{Salt: "s", Nonce: "n", WrappedPrivateKey: "w", KDFParams: "k", PublicKey: "p"}
	if w := e.do(t, http.MethodPut, "/messages/keys/backup", tokA, backup); w.Code != http.StatusOK {
		t.Fatalf("PutBackup = %d (%s)", w.Code, w.Body.String())
	}
	if w := e.do(t, http.MethodGet, "/messages/keys/backup", tokA, nil); w.Code != http.StatusOK {
		t.Fatalf("GetBackup = %d", w.Code)
	}

	// CreateDM.
	w := e.do(t, http.MethodPost, "/messages/conversations", tokA, models.CreateConversationRequest{
		Type: models.TypeDM, PeerID: hB, Envelopes: envM(hA, hB),
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("CreateDM = %d (%s)", w.Code, w.Body.String())
	}
	dmID := dataID(t, w)

	// ListConversations + GetConversation.
	if w := e.do(t, http.MethodGet, "/messages/conversations", tokA, nil); w.Code != http.StatusOK {
		t.Fatalf("ListConversations = %d", w.Code)
	}
	if w := e.do(t, http.MethodGet, "/messages/conversations/"+dmID, tokA, nil); w.Code != http.StatusOK {
		t.Fatalf("GetConversation = %d", w.Code)
	}

	// SendMessage + ListMessages.
	w = e.do(t, http.MethodPost, "/messages/conversations/"+dmID+"/messages", tokA, models.SendMessageRequest{Ciphertext: "c", Nonce: "n"})
	if w.Code != http.StatusCreated {
		t.Fatalf("SendMessage = %d (%s)", w.Code, w.Body.String())
	}
	msgID := dataID(t, w)
	if w := e.do(t, http.MethodGet, "/messages/conversations/"+dmID+"/messages", tokA, nil); w.Code != http.StatusOK {
		t.Fatalf("ListMessages = %d", w.Code)
	}

	// EditMessage + DeleteMessage.
	if w := e.do(t, http.MethodPatch, "/messages/conversations/"+dmID+"/messages/"+msgID, tokA, models.EditMessageRequest{Ciphertext: "c2", Nonce: "n2"}); w.Code != http.StatusOK {
		t.Fatalf("EditMessage = %d (%s)", w.Code, w.Body.String())
	}
	if w := e.do(t, http.MethodDelete, "/messages/conversations/"+dmID+"/messages/"+msgID, tokA, nil); w.Code != http.StatusOK {
		t.Fatalf("DeleteMessage = %d", w.Code)
	}

	// Typing + MarkRead + UnreadCount.
	if w := e.do(t, http.MethodPost, "/messages/conversations/"+dmID+"/typing", tokA, nil); w.Code != http.StatusNoContent {
		t.Fatalf("Typing = %d", w.Code)
	}
	if w := e.do(t, http.MethodPut, "/messages/conversations/"+dmID+"/read", tokA, nil); w.Code != http.StatusNoContent {
		t.Fatalf("MarkRead = %d", w.Code)
	}
	if w := e.do(t, http.MethodGet, "/messages/unread-count", tokA, nil); w.Code != http.StatusOK {
		t.Fatalf("UnreadCount = %d", w.Code)
	}

	// Pin / unpin / mute / unmute.
	for _, tc := range []struct {
		m, p string
		code int
	}{
		{http.MethodPatch, "/pin", http.StatusOK},
		{http.MethodDelete, "/pin", http.StatusOK},
		{http.MethodPatch, "/mute", http.StatusOK},
		{http.MethodDelete, "/mute", http.StatusOK},
	} {
		if w := e.do(t, tc.m, "/messages/conversations/"+dmID+tc.p, tokA, nil); w.Code != tc.code {
			t.Fatalf("%s %s = %d, attendu %d", tc.m, tc.p, w.Code, tc.code)
		}
	}
}

func TestHandler_GroupAndCommunity(t *testing.T) {
	e := newLiveEnv(t)
	tokA := e.token(t, hA, "user")
	tokB := e.token(t, hB, "user")

	// Create group.
	w := e.do(t, http.MethodPost, "/messages/conversations", tokA, models.CreateConversationRequest{
		Type: models.TypeGroup, Title: "t", TitleNonce: "n", Envelopes: envM(hA, hB),
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("CreateGroup = %d (%s)", w.Code, w.Body.String())
	}
	gID := dataID(t, w)

	// ListMembers + AddMember + RemoveMember.
	if w := e.do(t, http.MethodGet, "/messages/conversations/"+gID+"/members", tokA, nil); w.Code != http.StatusOK {
		t.Fatalf("ListMembers = %d", w.Code)
	}
	if w := e.do(t, http.MethodPost, "/messages/conversations/"+gID+"/members", tokA, models.AddMemberRequest{UserID: hC, Envelope: "env"}); w.Code != http.StatusCreated {
		t.Fatalf("AddMember = %d (%s)", w.Code, w.Body.String())
	}
	if w := e.do(t, http.MethodDelete, "/messages/conversations/"+gID+"/members/"+hC, tokA, nil); w.Code != http.StatusNoContent {
		t.Fatalf("RemoveMember = %d", w.Code)
	}
	// "me" → quitter.
	if w := e.do(t, http.MethodDelete, "/messages/conversations/"+gID+"/members/me", tokB, nil); w.Code != http.StatusNoContent {
		t.Fatalf("RemoveMember(me) = %d", w.Code)
	}

	// UpdateConversation (rename) + DeleteConversation.
	if w := e.do(t, http.MethodPatch, "/messages/conversations/"+gID, tokA, models.UpdateGroupRequest{Title: "t2", TitleNonce: "n2"}); w.Code != http.StatusOK {
		t.Fatalf("UpdateConversation = %d (%s)", w.Code, w.Body.String())
	}
	if w := e.do(t, http.MethodDelete, "/messages/conversations/"+gID, tokA, nil); w.Code != http.StatusNoContent {
		t.Fatalf("DeleteConversation = %d", w.Code)
	}

	// Community : create → list → join → set role → clear.
	w = e.do(t, http.MethodPost, "/messages/conversations", tokA, models.CreateConversationRequest{
		Type: models.TypeCommunity, Title: "Gophers", ContentKey: "ck",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("CreateCommunity = %d (%s)", w.Code, w.Body.String())
	}
	cID := dataID(t, w)
	if w := e.do(t, http.MethodGet, "/messages/communities?q=go", tokA, nil); w.Code != http.StatusOK {
		t.Fatalf("ListCommunities = %d", w.Code)
	}
	if w := e.do(t, http.MethodPost, "/messages/conversations/"+cID+"/join", tokB, nil); w.Code != http.StatusOK {
		t.Fatalf("JoinCommunity = %d (%s)", w.Code, w.Body.String())
	}
	if w := e.do(t, http.MethodPatch, "/messages/conversations/"+cID+"/members/"+hB, tokA, models.SetMemberRoleRequest{Role: models.MemberTalker}); w.Code != http.StatusOK {
		t.Fatalf("SetMemberRole = %d (%s)", w.Code, w.Body.String())
	}
	if w := e.do(t, http.MethodDelete, "/messages/conversations/"+cID+"/me", tokB, nil); w.Code != http.StatusNoContent {
		t.Fatalf("ClearConversation = %d", w.Code)
	}
}

// Modération de plateforme : suppression d'un message par son seul id (mod/admin).
func TestHandler_ModerationAndPurge(t *testing.T) {
	e := newLiveEnv(t)
	tokA := e.token(t, hA, "user")
	tokMod := e.token(t, hB, "moderator")
	tokAdmin := e.token(t, hC, "admin")

	// Crée un DM + un message.
	w := e.do(t, http.MethodPost, "/messages/conversations", tokA, models.CreateConversationRequest{
		Type: models.TypeDM, PeerID: hB, Envelopes: envM(hA, hB),
	})
	dmID := dataID(t, w)
	w = e.do(t, http.MethodPost, "/messages/conversations/"+dmID+"/messages", tokA, models.SendMessageRequest{Ciphertext: "c", Nonce: "n"})
	msgID := dataID(t, w)

	// Modération supprime par id.
	if w := e.do(t, http.MethodDelete, "/messages/moderation/"+msgID, tokMod, nil); w.Code != http.StatusOK {
		t.Fatalf("ModerateDelete = %d (%s)", w.Code, w.Body.String())
	}
	// Un simple user n'a pas le droit (garde ModeratorOnly).
	if w := e.do(t, http.MethodDelete, "/messages/moderation/"+msgID, tokA, nil); w.Code != http.StatusForbidden {
		t.Fatalf("ModerateDelete(user) = %d, attendu 403", w.Code)
	}

	// PurgeUser (admin).
	if w := e.do(t, http.MethodDelete, "/messages/users/"+hB, tokAdmin, nil); w.Code != http.StatusNoContent {
		t.Fatalf("PurgeUser = %d", w.Code)
	}
	// Non-admin refusé.
	if w := e.do(t, http.MethodDelete, "/messages/users/"+hB, tokA, nil); w.Code != http.StatusForbidden {
		t.Fatalf("PurgeUser(user) = %d, attendu 403", w.Code)
	}
}

// Erreurs propagées par les handlers : 404 conversation, 403 non membre.
func TestHandler_Errors(t *testing.T) {
	e := newLiveEnv(t)
	tokA := e.token(t, hA, "user")

	// Conversation inexistante → 404.
	if w := e.do(t, http.MethodGet, "/messages/conversations/deadbeefdeadbeefdeadbeef", tokA, nil); w.Code != http.StatusNotFound {
		t.Fatalf("GetConversation inexistante = %d, attendu 404", w.Code)
	}
	// Clé inexistante → 404.
	if w := e.do(t, http.MethodGet, "/messages/keys/unknown-user", tokA, nil); w.Code != http.StatusNotFound {
		t.Fatalf("GetKey inconnue = %d, attendu 404", w.Code)
	}
	// Backup inexistant → 404.
	if w := e.do(t, http.MethodGet, "/messages/keys/backup", tokA, nil); w.Code != http.StatusNotFound {
		t.Fatalf("GetBackup absent = %d, attendu 404", w.Code)
	}
	// Sans token → 401.
	req := httptest.NewRequest(http.MethodGet, "/messages/conversations", nil)
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("sans token = %d, attendu 401", rec.Code)
	}
}
