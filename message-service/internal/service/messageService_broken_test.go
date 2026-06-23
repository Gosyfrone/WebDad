package service

import (
	"context"
	"testing"
	"time"

	"github.com/webdad/message-service/internal/models"
	"github.com/webdad/message-service/internal/mongotest"
	"github.com/webdad/message-service/internal/repository"
)

// newBrokenSvc monte un service dont le client Mongo est DÉCONNECTÉ : chaque
// appel au dépôt échoue, ce qui couvre les branches de remontée d'erreur du
// service (les `if err != nil { return ..., err }` après un appel repo).
func newBrokenSvc(t *testing.T) (*MessageService, context.Context) {
	t.Helper()
	db := mongotest.DisposableDB(t)
	if err := db.Client().Disconnect(context.Background()); err != nil {
		t.Fatalf("Disconnect : %v", err)
	}
	return NewMessageService(repository.NewMessageRepository(db)), context.Background()
}

func TestSvc_BrokenDB_ErrorPaths(t *testing.T) {
	s, ctx := newBrokenSvc(t)
	cid := "507f1f77bcf86cd799439011" // ObjectID valide → on passe parseID
	mid := "507f1f77bcf86cd799439012"

	mustErr := func(name string, err error) {
		if err == nil {
			t.Errorf("%s : erreur attendue (DB en panne)", name)
		}
	}

	mustErr("PublishKey", s.PublishKey(ctx, "u", "k"))
	mustErr("PurgeUser", s.PurgeUser(ctx, "u"))
	_, err := s.GetKey(ctx, "u")
	mustErr("GetKey", err)
	mustErr("PutBackup", s.PutBackup(ctx, "u", models.PutBackupRequest{}))
	_, err = s.GetBackup(ctx, "u")
	mustErr("GetBackup", err)
	_, err = s.BackupStatus(ctx, "u")
	mustErr("BackupStatus", err)

	_, err = s.CreateDM(ctx, "a", "b", map[string]string{"a": "e", "b": "e"})
	mustErr("CreateDM", err)
	_, err = s.CreateGroup(ctx, "a", "t", "n", map[string]string{"a": "e", "b": "e"})
	mustErr("CreateGroup", err)
	_, err = s.CreateCommunity(ctx, "a", "t", "ck")
	mustErr("CreateCommunity", err)
	_, _, err = s.JoinCommunity(ctx, cid, "u")
	mustErr("JoinCommunity", err)
	_, err = s.SetMemberRole(ctx, cid, "a", "b", models.MemberTalker)
	mustErr("SetMemberRole", err)
	_, err = s.ListCommunities(ctx, "u", 10, 0, "")
	mustErr("ListCommunities", err)
	_, err = s.AddMember(ctx, cid, "a", "b", "e")
	mustErr("AddMember", err)
	_, err = s.RemoveMember(ctx, cid, "a", "b")
	mustErr("RemoveMember", err)
	_, err = s.DeleteGroup(ctx, cid, "a")
	mustErr("DeleteGroup", err)
	_, err = s.UpdateGroup(ctx, cid, "a", "t", "n")
	mustErr("UpdateGroup", err)
	_, err = s.ListMembers(ctx, cid, "a")
	mustErr("ListMembers", err)
	_, err = s.ListConversations(ctx, "u")
	mustErr("ListConversations", err)
	_, err = s.PinConversation(ctx, cid, "u", true)
	mustErr("PinConversation", err)
	mustErr("ClearConversation", s.ClearConversation(ctx, cid, "u"))
	_, err = s.MuteConversation(ctx, cid, "u", true)
	mustErr("MuteConversation", err)
	_, _, err = s.MarkRead(ctx, cid, "u")
	mustErr("MarkRead", err)
	_, _, err = s.TouchDelivered(ctx, cid, "u")
	mustErr("TouchDelivered", err)
	_, err = s.TypingTargets(ctx, cid, "u")
	mustErr("TypingTargets", err)
	_, err = s.UnreadCount(ctx, "u")
	mustErr("UnreadCount", err)
	_, err = s.GetConversation(ctx, cid, "u")
	mustErr("GetConversation", err)
	_, err = s.ListMessages(ctx, cid, "u", 10, "")
	mustErr("ListMessages", err)
	_, _, err = s.SendMessage(ctx, cid, "u", "c", "n", nil)
	mustErr("SendMessage", err)
	_, _, err = s.EditMessage(ctx, cid, mid, "u", "c", "n", nil)
	mustErr("EditMessage", err)
	_, _, err = s.DeleteMessage(ctx, cid, mid, "u")
	mustErr("DeleteMessage", err)
	_, _, err = s.ModerateDeleteMessage(ctx, mid)
	mustErr("ModerateDeleteMessage", err)
	_, err = s.IsMember(ctx, cid, "u")
	mustErr("IsMember", err)

	// MarkDelivered est best-effort (pas de retour) : ne doit pas paniquer.
	s.MarkDelivered(ctx, cid, []string{"u"}, time.Now())
}
