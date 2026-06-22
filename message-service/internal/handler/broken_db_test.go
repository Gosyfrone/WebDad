package handler

import (
	"context"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/webdad/message-service/internal/database"
	"github.com/webdad/message-service/internal/models"
	"github.com/webdad/message-service/internal/mongotest"
	"github.com/webdad/message-service/internal/realtime"
	"github.com/webdad/message-service/internal/repository"
	"github.com/webdad/message-service/internal/service"
)

// newBrokenEnv monte un routeur réel dont le client Mongo est DÉCONNECTÉ : tout
// appel au dépôt échoue. Les requêtes (authentifiées, corps valides) traversent
// donc les handlers jusqu'à `respondError` — ce qui couvre les branches d'erreur
// que le chemin nominal n'emprunte jamais.
func newBrokenEnv(t *testing.T) *liveEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := mongotest.DisposableDB(t)
	ctx := context.Background()
	_ = database.EnsureSchema(ctx, db)
	repo := repository.NewMessageRepository(db)
	svc := service.NewMessageService(repo)
	r := gin.New()
	r.Use(gin.Recovery())
	RegisterRoutes(r, "message-test", svc, realtime.NewHub(), msgTestSecret, []string{"http://localhost:3000"})

	if err := db.Client().Disconnect(ctx); err != nil {
		t.Fatalf("Disconnect : %v", err)
	}
	return &liveEnv{router: r, svc: svc, repo: repo, ctx: ctx}
}

// Toutes les routes authentifiées doivent répondre par une erreur (≥400) quand
// le dépôt est en panne, sans jamais paniquer : on traverse `respondError`.
func TestHandlers_BrokenDB_ErrorPaths(t *testing.T) {
	e := newBrokenEnv(t)
	tokA := e.token(t, hA, "user")
	tokAdmin := e.token(t, hA, "admin")
	tokMod := e.token(t, hA, "moderator")
	cid := "507f1f77bcf86cd799439011"
	mid := "507f1f77bcf86cd799439012"

	cases := []struct {
		name, method, path, tok string
		body                    any
	}{
		{"PublishKey", http.MethodPut, "/messages/keys", tokA, models.PublishKeyRequest{PublicKey: "k"}},
		{"GetKey", http.MethodGet, "/messages/keys/" + hB, tokA, nil},
		{"PutBackup", http.MethodPut, "/messages/keys/backup", tokA, models.PutBackupRequest{Salt: "s", Nonce: "n", WrappedPrivateKey: "w", KDFParams: "k", PublicKey: "p"}},
		{"GetBackup", http.MethodGet, "/messages/keys/backup", tokA, nil},
		{"BackupStatus", http.MethodGet, "/messages/keys/backup/status", tokA, nil},
		{"CreateDM", http.MethodPost, "/messages/conversations", tokA, models.CreateConversationRequest{Type: models.TypeDM, PeerID: hB, Envelopes: envM(hA, hB)}},
		{"CreateGroup", http.MethodPost, "/messages/conversations", tokA, models.CreateConversationRequest{Type: models.TypeGroup, Title: "t", Envelopes: envM(hA, hB)}},
		{"CreateCommunity", http.MethodPost, "/messages/conversations", tokA, models.CreateConversationRequest{Type: models.TypeCommunity, Title: "c", ContentKey: "ck"}},
		{"ListConversations", http.MethodGet, "/messages/conversations", tokA, nil},
		{"GetConversation", http.MethodGet, "/messages/conversations/" + cid, tokA, nil},
		{"UpdateConversation", http.MethodPatch, "/messages/conversations/" + cid, tokA, models.UpdateGroupRequest{Title: "t"}},
		{"DeleteConversation", http.MethodDelete, "/messages/conversations/" + cid, tokA, nil},
		{"JoinCommunity", http.MethodPost, "/messages/conversations/" + cid + "/join", tokA, nil},
		{"ListCommunities", http.MethodGet, "/messages/communities", tokA, nil},
		{"SetMemberRole", http.MethodPatch, "/messages/conversations/" + cid + "/members/" + hB, tokA, models.SetMemberRoleRequest{Role: "talker"}},
		{"PinConversation", http.MethodPatch, "/messages/conversations/" + cid + "/pin", tokA, nil},
		{"UnpinConversation", http.MethodDelete, "/messages/conversations/" + cid + "/pin", tokA, nil},
		{"MuteConversation", http.MethodPatch, "/messages/conversations/" + cid + "/mute", tokA, nil},
		{"UnmuteConversation", http.MethodDelete, "/messages/conversations/" + cid + "/mute", tokA, nil},
		{"ClearConversation", http.MethodDelete, "/messages/conversations/" + cid + "/me", tokA, nil},
		{"MarkRead", http.MethodPut, "/messages/conversations/" + cid + "/read", tokA, nil},
		{"UnreadCount", http.MethodGet, "/messages/unread-count", tokA, nil},
		{"ListMembers", http.MethodGet, "/messages/conversations/" + cid + "/members", tokA, nil},
		{"AddMember", http.MethodPost, "/messages/conversations/" + cid + "/members", tokA, models.AddMemberRequest{UserID: hB, Envelope: "e"}},
		{"RemoveMember", http.MethodDelete, "/messages/conversations/" + cid + "/members/" + hB, tokA, nil},
		{"ListMessages", http.MethodGet, "/messages/conversations/" + cid + "/messages", tokA, nil},
		{"SendMessage", http.MethodPost, "/messages/conversations/" + cid + "/messages", tokA, models.SendMessageRequest{Ciphertext: "c", Nonce: "n"}},
		{"EditMessage", http.MethodPatch, "/messages/conversations/" + cid + "/messages/" + mid, tokA, models.EditMessageRequest{Ciphertext: "c", Nonce: "n"}},
		{"DeleteMessage", http.MethodDelete, "/messages/conversations/" + cid + "/messages/" + mid, tokA, nil},
		{"Typing", http.MethodPost, "/messages/conversations/" + cid + "/typing", tokA, nil},
		{"ModerateDeleteMessage", http.MethodDelete, "/messages/moderation/" + mid, tokMod, nil},
		{"PurgeUser", http.MethodDelete, "/messages/users/" + hB, tokAdmin, nil},
	}

	for _, tc := range cases {
		w := e.do(t, tc.method, tc.path, tc.tok, tc.body)
		if w.Code < 400 {
			t.Errorf("%s (DB en panne) = %d, attendu une erreur ≥400", tc.name, w.Code)
		}
	}
}
