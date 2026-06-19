package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/webdad/message-service/internal/middleware"
	"github.com/webdad/message-service/internal/realtime"
	"github.com/webdad/message-service/internal/service"
)

const (
	msgTestSecret = "msg-test-secret"
	// UserID injecté dans tous les tokens de test.
	msgTestUserID = "11111111-1111-1111-1111-111111111111"
)

// newNilSvcRouter monte les routes avec un MessageService à repo nil.
// gin.Recovery() convertit les panics (nil repo) en 500.
func newNilSvcRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	svc := service.NewMessageService(nil)
	RegisterRoutes(r, "message-test", svc, realtime.NewHub(), msgTestSecret, nil)
	return r
}

func makeMsgToken(t *testing.T, role string) string {
	t.Helper()
	now := time.Now()
	claims := middleware.Claims{
		UserID: msgTestUserID,
		Email:  "user@breezy.dev",
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(msgTestSecret))
	if err != nil {
		t.Fatalf("makeMsgToken: %v", err)
	}
	return tok
}

// ─── CreateConversation — gardes avant repo ────────────────────────────────────

func TestCreateConversation_JSONInvalide_400(t *testing.T) {
	r := newNilSvcRouter(t)
	tok := makeMsgToken(t, "user")
	req := httptest.NewRequest(http.MethodPost, "/messages/conversations", strings.NewReader(`{bad}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("CreateConversation(JSON invalide) = %d, attendu 400", w.Code)
	}
}

// DM avec soi-même → ErrSelfConversation → respondError → 400.
// Le claims.UserID = msgTestUserID ; peer_id = même id.
func TestCreateConversation_DM_Self_400(t *testing.T) {
	r := newNilSvcRouter(t)
	tok := makeMsgToken(t, "user")
	body := `{"type":"dm","peer_id":"` + msgTestUserID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/messages/conversations", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("CreateConversation(DM self) = %d, attendu 400", w.Code)
	}
}

// ─── ListConversations — nil repo → 500 ──────────────────────────────────────

func TestListConversations_NilRepo_500(t *testing.T) {
	r := newNilSvcRouter(t)
	tok := makeMsgToken(t, "user")
	req := httptest.NewRequest(http.MethodGet, "/messages/conversations", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ListConversations(nil repo) = %d, attendu 500", w.Code)
	}
}

// ─── GetConversation — nil repo → 500 ────────────────────────────────────────

func TestGetConversation_NilRepo_500(t *testing.T) {
	r := newNilSvcRouter(t)
	tok := makeMsgToken(t, "user")
	req := httptest.NewRequest(http.MethodGet, "/messages/conversations/some-conv-id", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("GetConversation(nil repo) = %d, attendu 500", w.Code)
	}
}

// ─── ListMessages — nil repo → 500 ───────────────────────────────────────────

func TestListMessages_NilRepo_500(t *testing.T) {
	r := newNilSvcRouter(t)
	tok := makeMsgToken(t, "user")
	req := httptest.NewRequest(http.MethodGet, "/messages/conversations/some-conv-id/messages", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ListMessages(nil repo) = %d, attendu 500", w.Code)
	}
}

// ─── SendMessage — JSON invalide → 400 ───────────────────────────────────────

func TestSendMessage_JSONInvalide_400(t *testing.T) {
	r := newNilSvcRouter(t)
	tok := makeMsgToken(t, "user")
	req := httptest.NewRequest(http.MethodPost, "/messages/conversations/conv-id/messages", strings.NewReader(`{bad}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("SendMessage(JSON invalide) = %d, attendu 400", w.Code)
	}
}

// ─── EditMessage — JSON invalide → 400 ───────────────────────────────────────

func TestEditMessage_JSONInvalide_400(t *testing.T) {
	r := newNilSvcRouter(t)
	tok := makeMsgToken(t, "user")
	req := httptest.NewRequest(http.MethodPatch, "/messages/conversations/conv-id/messages/msg-id", strings.NewReader(`{bad}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("EditMessage(JSON invalide) = %d, attendu 400", w.Code)
	}
}

// ─── DeleteMessage — nil repo → 500 ──────────────────────────────────────────

func TestDeleteMessage_NilRepo_500(t *testing.T) {
	r := newNilSvcRouter(t)
	tok := makeMsgToken(t, "user")
	req := httptest.NewRequest(http.MethodDelete, "/messages/conversations/conv-id/messages/msg-id", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("DeleteMessage(nil repo) = %d, attendu 500", w.Code)
	}
}

// ─── MarkRead — nil repo → 500 ───────────────────────────────────────────────

func TestMarkRead_NilRepo_500(t *testing.T) {
	r := newNilSvcRouter(t)
	tok := makeMsgToken(t, "user")
	req := httptest.NewRequest(http.MethodPut, "/messages/conversations/conv-id/read", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("MarkRead(nil repo) = %d, attendu 500", w.Code)
	}
}

// ─── UnreadCount — nil repo → 500 ────────────────────────────────────────────

func TestUnreadCount_NilRepo_500(t *testing.T) {
	r := newNilSvcRouter(t)
	tok := makeMsgToken(t, "user")
	req := httptest.NewRequest(http.MethodGet, "/messages/unread-count", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("UnreadCount(nil repo) = %d, attendu 500", w.Code)
	}
}

// ─── ListCommunities — nil repo → 500 ────────────────────────────────────────

func TestListCommunities_NilRepo_500(t *testing.T) {
	r := newNilSvcRouter(t)
	tok := makeMsgToken(t, "user")
	req := httptest.NewRequest(http.MethodGet, "/messages/communities", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ListCommunities(nil repo) = %d, attendu 500", w.Code)
	}
}

// ─── ListMembers — nil repo → 500 ────────────────────────────────────────────

func TestListMembers_NilRepo_500(t *testing.T) {
	r := newNilSvcRouter(t)
	tok := makeMsgToken(t, "user")
	req := httptest.NewRequest(http.MethodGet, "/messages/conversations/conv-id/members", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ListMembers(nil repo) = %d, attendu 500", w.Code)
	}
}

// ─── AddMember — JSON invalide → 400 ─────────────────────────────────────────

func TestAddMember_JSONInvalide_400(t *testing.T) {
	r := newNilSvcRouter(t)
	tok := makeMsgToken(t, "user")
	req := httptest.NewRequest(http.MethodPost, "/messages/conversations/conv-id/members", strings.NewReader(`{bad}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("AddMember(JSON invalide) = %d, attendu 400", w.Code)
	}
}

// ─── SetMemberRole — JSON invalide → 400 ─────────────────────────────────────

func TestSetMemberRole_JSONInvalide_400(t *testing.T) {
	r := newNilSvcRouter(t)
	tok := makeMsgToken(t, "user")
	req := httptest.NewRequest(http.MethodPatch, "/messages/conversations/conv-id/members/u2", strings.NewReader(`{bad}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("SetMemberRole(JSON invalide) = %d, attendu 400", w.Code)
	}
}

// ─── PublishKey — JSON invalide → 400 ────────────────────────────────────────

func TestPublishKey_JSONInvalide_400(t *testing.T) {
	r := newNilSvcRouter(t)
	tok := makeMsgToken(t, "user")
	req := httptest.NewRequest(http.MethodPut, "/messages/keys", strings.NewReader(`{bad}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("PublishKey(JSON invalide) = %d, attendu 400", w.Code)
	}
}

// ─── GetKey — nil repo → 500 ─────────────────────────────────────────────────

func TestGetKey_NilRepo_500(t *testing.T) {
	r := newNilSvcRouter(t)
	tok := makeMsgToken(t, "user")
	req := httptest.NewRequest(http.MethodGet, "/messages/keys/some-user-id", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("GetKey(nil repo) = %d, attendu 500", w.Code)
	}
}

// ─── PutBackup — JSON invalide → 400 ─────────────────────────────────────────

func TestPutBackup_JSONInvalide_400(t *testing.T) {
	r := newNilSvcRouter(t)
	tok := makeMsgToken(t, "user")
	req := httptest.NewRequest(http.MethodPut, "/messages/keys/backup", strings.NewReader(`{bad}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("PutBackup(JSON invalide) = %d, attendu 400", w.Code)
	}
}

// ─── GetBackup — nil repo → 500 ──────────────────────────────────────────────

func TestGetBackup_NilRepo_500(t *testing.T) {
	r := newNilSvcRouter(t)
	tok := makeMsgToken(t, "user")
	req := httptest.NewRequest(http.MethodGet, "/messages/keys/backup", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("GetBackup(nil repo) = %d, attendu 500", w.Code)
	}
}

// ─── BackupStatus — nil repo → 500 ───────────────────────────────────────────

func TestBackupStatus_NilRepo_500(t *testing.T) {
	r := newNilSvcRouter(t)
	tok := makeMsgToken(t, "user")
	req := httptest.NewRequest(http.MethodGet, "/messages/keys/backup/status", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("BackupStatus(nil repo) = %d, attendu 500", w.Code)
	}
}

// ─── PinConversation / UnpinConversation / MuteConversation — nil repo → 500 ─

func TestPinConversation_NilRepo_500(t *testing.T) {
	r := newNilSvcRouter(t)
	tok := makeMsgToken(t, "user")
	req := httptest.NewRequest(http.MethodPatch, "/messages/conversations/conv-id/pin", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("PinConversation(nil repo) = %d, attendu 500", w.Code)
	}
}

func TestMuteConversation_NilRepo_500(t *testing.T) {
	r := newNilSvcRouter(t)
	tok := makeMsgToken(t, "user")
	req := httptest.NewRequest(http.MethodPatch, "/messages/conversations/conv-id/mute", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("MuteConversation(nil repo) = %d, attendu 500", w.Code)
	}
}

// ─── PurgeUser — admin route, nil repo → 500 ─────────────────────────────────

func TestPurgeUser_NilRepo_500(t *testing.T) {
	r := newNilSvcRouter(t)
	tok := makeMsgToken(t, "admin")
	req := httptest.NewRequest(http.MethodDelete, "/messages/users/some-user", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("PurgeUser(nil repo) = %d, attendu 500", w.Code)
	}
}
