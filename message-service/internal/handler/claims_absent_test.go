package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/webdad/message-service/internal/realtime"
	"github.com/webdad/message-service/internal/service"
)

// Chaque handler authentifié commence par `ClaimsFrom(c)` : sans claims dans le
// contexte (cas défensif, normalement garanti par JWTAuth), il doit répondre 401
// AVANT de toucher au service. On appelle donc les handlers directement sur un
// contexte vierge — hermétique, sans Mongo (service à repo nil non sollicité).
func TestHandlers_ClaimsAbsent_401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := service.NewMessageService(nil)
	conv := NewConversationHandler(svc, realtime.NewHub())
	keys := NewKeyHandler(svc)

	handlers := map[string]gin.HandlerFunc{
		"CreateConversation": conv.CreateConversation,
		"ListConversations":  conv.ListConversations,
		"GetConversation":    conv.GetConversation,
		"UpdateConversation": conv.UpdateConversation,
		"DeleteConversation": conv.DeleteConversation,
		"JoinCommunity":      conv.JoinCommunity,
		"ListCommunities":    conv.ListCommunities,
		"SetMemberRole":      conv.SetMemberRole,
		"PinConversation":    conv.PinConversation,
		"UnpinConversation":  conv.UnpinConversation,
		"MuteConversation":   conv.MuteConversation,
		"UnmuteConversation": conv.UnmuteConversation,
		"ClearConversation":  conv.ClearConversation,
		"MarkRead":           conv.MarkRead,
		"UnreadCount":        conv.UnreadCount,
		"ListMembers":        conv.ListMembers,
		"AddMember":          conv.AddMember,
		"RemoveMember":       conv.RemoveMember,
		"ListMessages":       conv.ListMessages,
		"SendMessage":        conv.SendMessage,
		"EditMessage":        conv.EditMessage,
		"DeleteMessage":      conv.DeleteMessage,
		"Typing":             conv.Typing,
		"PublishKey":         keys.PublishKey,
		"PutBackup":          keys.PutBackup,
		"GetBackup":          keys.GetBackup,
		"BackupStatus":       keys.BackupStatus,
	}

	for name, h := range handlers {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
		h(c)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s sans claims = %d, attendu 401", name, w.Code)
		}
	}
}
