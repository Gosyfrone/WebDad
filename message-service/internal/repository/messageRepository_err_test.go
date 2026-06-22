package repository

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/webdad/message-service/internal/models"
	"github.com/webdad/message-service/internal/mongotest"
)

// brokenRepo renvoie un dépôt dont le client a été DÉCONNECTÉ : chaque appel au
// driver échoue, ce qui couvre les branches de remontée d'erreur (`if err != nil`)
// que le chemin nominal ne touche jamais.
func brokenRepo(t *testing.T) (*MessageRepository, context.Context) {
	t.Helper()
	db := mongotest.DisposableDB(t)
	if err := db.Client().Disconnect(context.Background()); err != nil {
		t.Fatalf("Disconnect : %v", err)
	}
	return NewMessageRepository(db), context.Background()
}

func TestRepo_ErrorBranches(t *testing.T) {
	r, ctx := brokenRepo(t)
	oid := bson.NewObjectID()
	now := time.Now()

	// On appelle chaque méthode : l'erreur driver doit être remontée (non nil).
	mustErr := func(name string, err error) {
		if err == nil {
			t.Errorf("%s : erreur attendue sur client déconnecté", name)
		}
	}

	mustErr("PurgeUser", r.PurgeUser(ctx, "u"))
	mustErr("UpsertKey", r.UpsertKey(ctx, "u", "k"))
	_, err := r.GetKey(ctx, "u")
	mustErr("GetKey", err)
	mustErr("UpsertBackup", r.UpsertBackup(ctx, "u", models.PutBackupRequest{}))
	_, err = r.GetBackup(ctx, "u")
	mustErr("GetBackup", err)
	_, err = r.BackupExists(ctx, "u")
	mustErr("BackupExists", err)
	_, err = r.FindDMByKey(ctx, "a:b")
	mustErr("FindDMByKey", err)
	mustErr("CreateConversation", r.CreateConversation(ctx, &models.Conversation{}))
	_, err = r.GetConversation(ctx, oid)
	mustErr("GetConversation", err)
	mustErr("TouchConversation", r.TouchConversation(ctx, oid, now))
	_, err = r.UpdateTitle(ctx, oid, "t", "n")
	mustErr("UpdateTitle", err)
	mustErr("PushMemberID", r.PushMemberID(ctx, oid, "u"))
	mustErr("PullMemberID", r.PullMemberID(ctx, oid, "u"))
	mustErr("DeleteConversation", r.DeleteConversation(ctx, oid))
	mustErr("AddMember", r.AddMember(ctx, &models.Member{}))
	_, err = r.GetMember(ctx, "c", "u")
	mustErr("GetMember", err)
	_, err = r.ListMembersByUser(ctx, "u")
	mustErr("ListMembersByUser", err)
	_, err = r.ListMembers(ctx, "c")
	mustErr("ListMembers", err)
	_, err = r.CountMembers(ctx, "c")
	mustErr("CountMembers", err)
	_, err = r.CountWritableMembers(ctx, "c")
	mustErr("CountWritableMembers", err)
	mustErr("SetMemberPinned", r.SetMemberPinned(ctx, "c", "u", &now))
	mustErr("SetMemberMuted", r.SetMemberMuted(ctx, "c", "u", &now))
	mustErr("SetMemberCleared", r.SetMemberCleared(ctx, "c", "u", now))
	_, err = r.HasMessagesAfter(ctx, "c", now)
	mustErr("HasMessagesAfter", err)
	mustErr("SetMemberRead", r.SetMemberRead(ctx, "c", "u", now))
	mustErr("SetMemberDelivered", r.SetMemberDelivered(ctx, "c", "u", now))
	_, err = r.CountUnreadConversations(ctx, "u")
	mustErr("CountUnreadConversations", err)
	mustErr("SetMemberRole", r.SetMemberRole(ctx, "c", "u", "talker"))
	_, err = r.ListCommunities(ctx, 10, 0, "q")
	mustErr("ListCommunities", err)
	_, err = r.RemoveMember(ctx, "c", "u")
	mustErr("RemoveMember", err)
	mustErr("DeleteMembersByConversation", r.DeleteMembersByConversation(ctx, "c"))
	mustErr("DeleteMessagesByConversation", r.DeleteMessagesByConversation(ctx, "c"))
	_, err = r.MemberIDs(ctx, "c")
	mustErr("MemberIDs", err)
	mustErr("InsertMessage", r.InsertMessage(ctx, &models.Message{}))
	_, err = r.GetMessage(ctx, "c", oid)
	mustErr("GetMessage", err)
	_, err = r.GetMessageByID(ctx, oid)
	mustErr("GetMessageByID", err)
	_, err = r.ModerateSoftDelete(ctx, oid, now)
	mustErr("ModerateSoftDelete", err)
	_, err = r.UpdateMessageCiphertext(ctx, &models.Message{ID: oid}, "c", "n", now)
	mustErr("UpdateMessageCiphertext", err)
	_, err = r.SoftDeleteMessage(ctx, "c", oid, now)
	mustErr("SoftDeleteMessage", err)
	_, err = r.ListMessages(ctx, "c", 10, nil, nil)
	mustErr("ListMessages", err)
}
