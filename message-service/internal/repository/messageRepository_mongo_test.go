package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/webdad/message-service/internal/database"
	"github.com/webdad/message-service/internal/models"
	"github.com/webdad/message-service/internal/mongotest"
)

// newRepo monte un dépôt sur une base jetable, schéma initialisé (couvre aussi
// database.EnsureSchema). Renvoie le dépôt + un contexte de test.
func newRepo(t *testing.T) (*MessageRepository, context.Context) {
	t.Helper()
	db := mongotest.DB(t)
	ctx := context.Background()
	if err := database.EnsureSchema(ctx, db); err != nil {
		t.Fatalf("EnsureSchema : %v", err)
	}
	return NewMessageRepository(db), ctx
}

// --- Clés publiques + sauvegardes -------------------------------------------

func TestRepo_Keys(t *testing.T) {
	r, ctx := newRepo(t)

	// Absente → ErrNoDocuments.
	if _, err := r.GetKey(ctx, "u1"); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("GetKey absente → %v, attendu ErrNoDocuments", err)
	}

	if err := r.UpsertKey(ctx, "u1", "pub-1"); err != nil {
		t.Fatalf("UpsertKey : %v", err)
	}
	// Upsert idempotent (met à jour).
	if err := r.UpsertKey(ctx, "u1", "pub-2"); err != nil {
		t.Fatalf("UpsertKey maj : %v", err)
	}
	k, err := r.GetKey(ctx, "u1")
	if err != nil || k.PublicKey != "pub-2" {
		t.Fatalf("GetKey = %+v, %v ; attendu pub-2", k, err)
	}
}

func TestRepo_Backups(t *testing.T) {
	r, ctx := newRepo(t)

	if ok, err := r.BackupExists(ctx, "u1"); err != nil || ok {
		t.Fatalf("BackupExists vide = %v, %v ; attendu false", ok, err)
	}
	if _, err := r.GetBackup(ctx, "u1"); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("GetBackup absente → %v", err)
	}

	b := models.PutBackupRequest{
		Salt: "s", Nonce: "n", WrappedPrivateKey: "w", KDFParams: "k", PublicKey: "p",
	}
	if err := r.UpsertBackup(ctx, "u1", b); err != nil {
		t.Fatalf("UpsertBackup : %v", err)
	}
	if ok, err := r.BackupExists(ctx, "u1"); err != nil || !ok {
		t.Fatalf("BackupExists après upsert = %v, %v ; attendu true", ok, err)
	}
	got, err := r.GetBackup(ctx, "u1")
	if err != nil || got.WrappedPrivateKey != "w" {
		t.Fatalf("GetBackup = %+v, %v", got, err)
	}
}

// --- Conversations + membres -------------------------------------------------

// seedConv insère une conversation de type `typ` avec les membres donnés (rôle
// indexé par user). Renvoie l'id hex.
func seedConv(t *testing.T, r *MessageRepository, ctx context.Context, typ string, roles map[string]string) string {
	t.Helper()
	now := time.Now()
	ids := make([]string, 0, len(roles))
	for uid := range roles {
		ids = append(ids, uid)
	}
	conv := &models.Conversation{
		Type:      typ,
		MemberIDs: ids,
		CreatedBy: ids[0],
		CreatedAt: now,
		UpdatedAt: now,
	}
	if typ == models.TypeDM {
		conv.DMKey = ""
	}
	if err := r.CreateConversation(ctx, conv); err != nil {
		t.Fatalf("CreateConversation : %v", err)
	}
	cid := conv.ID.Hex()
	for uid, role := range roles {
		if err := r.AddMember(ctx, &models.Member{
			ConversationID: cid, UserID: uid, Role: role, CreatedAt: now,
		}); err != nil {
			t.Fatalf("AddMember : %v", err)
		}
	}
	return cid
}

func TestRepo_ConversationsAndMembers(t *testing.T) {
	r, ctx := newRepo(t)

	cid := seedConv(t, r, ctx, models.TypeGroup, map[string]string{
		"owner": models.MemberOwner, "talk": models.MemberTalker,
	})
	oid, _ := bson.ObjectIDFromHex(cid)

	// GetConversation.
	conv, err := r.GetConversation(ctx, oid)
	if err != nil || conv.Type != models.TypeGroup {
		t.Fatalf("GetConversation = %+v, %v", conv, err)
	}

	// GetMember présent / absent.
	if _, err := r.GetMember(ctx, cid, "owner"); err != nil {
		t.Fatalf("GetMember owner : %v", err)
	}
	if _, err := r.GetMember(ctx, cid, "ghost"); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("GetMember ghost → %v", err)
	}

	// Counts.
	if n, _ := r.CountMembers(ctx, cid); n != 2 {
		t.Fatalf("CountMembers = %d, attendu 2", n)
	}
	if n, _ := r.CountWritableMembers(ctx, cid); n != 2 {
		t.Fatalf("CountWritableMembers = %d, attendu 2", n)
	}

	// Listes.
	if ms, _ := r.ListMembers(ctx, cid); len(ms) != 2 {
		t.Fatalf("ListMembers = %d", len(ms))
	}
	if ms, _ := r.ListMembersByUser(ctx, "owner"); len(ms) != 1 {
		t.Fatalf("ListMembersByUser = %d", len(ms))
	}
	if ids, _ := r.MemberIDs(ctx, cid); len(ids) != 2 {
		t.Fatalf("MemberIDs = %d", len(ids))
	}

	// Title + touch + member_ids push/pull.
	if _, err := r.UpdateTitle(ctx, oid, "ct", "nz"); err != nil {
		t.Fatalf("UpdateTitle : %v", err)
	}
	if err := r.TouchConversation(ctx, oid, time.Now()); err != nil {
		t.Fatalf("TouchConversation : %v", err)
	}
	if err := r.PushMemberID(ctx, oid, "newbie"); err != nil {
		t.Fatalf("PushMemberID : %v", err)
	}
	if err := r.PullMemberID(ctx, oid, "newbie"); err != nil {
		t.Fatalf("PullMemberID : %v", err)
	}

	// Role.
	if err := r.SetMemberRole(ctx, cid, "talk", models.MemberViewer); err != nil {
		t.Fatalf("SetMemberRole : %v", err)
	}
	if err := r.SetMemberRole(ctx, cid, "ghost", models.MemberViewer); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("SetMemberRole ghost → %v", err)
	}

	// RemoveMember.
	if ok, err := r.RemoveMember(ctx, cid, "talk"); err != nil || !ok {
		t.Fatalf("RemoveMember = %v, %v", ok, err)
	}
	if ok, _ := r.RemoveMember(ctx, cid, "talk"); ok {
		t.Fatalf("RemoveMember 2e fois doit être false")
	}

	// DeleteConversation présent / absent.
	if err := r.DeleteConversation(ctx, oid); err != nil {
		t.Fatalf("DeleteConversation : %v", err)
	}
	if err := r.DeleteConversation(ctx, oid); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("DeleteConversation absente → %v", err)
	}
}

func TestRepo_DMKeyAndCommunities(t *testing.T) {
	r, ctx := newRepo(t)

	now := time.Now()
	conv := &models.Conversation{
		Type: models.TypeDM, MemberIDs: []string{"a", "b"}, DMKey: "a:b",
		CreatedBy: "a", CreatedAt: now, UpdatedAt: now,
	}
	if err := r.CreateConversation(ctx, conv); err != nil {
		t.Fatalf("CreateConversation dm : %v", err)
	}
	if _, err := r.FindDMByKey(ctx, "a:b"); err != nil {
		t.Fatalf("FindDMByKey : %v", err)
	}
	if _, err := r.FindDMByKey(ctx, "x:y"); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("FindDMByKey absent → %v", err)
	}

	// Communautés : annuaire + recherche.
	c1 := &models.Conversation{Type: models.TypeCommunity, MemberIDs: []string{"a"}, Title: "Golang", CreatedBy: "a", CreatedAt: now, UpdatedAt: now}
	c2 := &models.Conversation{Type: models.TypeCommunity, MemberIDs: []string{"a"}, Title: "Rustaceans", CreatedBy: "a", CreatedAt: now, UpdatedAt: now}
	_ = r.CreateConversation(ctx, c1)
	_ = r.CreateConversation(ctx, c2)
	if list, _ := r.ListCommunities(ctx, 10, 0, ""); len(list) != 2 {
		t.Fatalf("ListCommunities = %d, attendu 2", len(list))
	}
	if list, _ := r.ListCommunities(ctx, 10, 0, "go"); len(list) != 1 {
		t.Fatalf("ListCommunities(q=go) = %d, attendu 1", len(list))
	}
}

// --- État par-utilisateur (pin/mute/clear/read/delivered) --------------------

func TestRepo_MemberState(t *testing.T) {
	r, ctx := newRepo(t)
	cid := seedConv(t, r, ctx, models.TypeGroup, map[string]string{"u1": models.MemberOwner})
	now := time.Now()

	// Pin / unpin.
	if err := r.SetMemberPinned(ctx, cid, "u1", &now); err != nil {
		t.Fatalf("SetMemberPinned : %v", err)
	}
	if err := r.SetMemberPinned(ctx, cid, "u1", nil); err != nil {
		t.Fatalf("SetMemberPinned unset : %v", err)
	}
	if err := r.SetMemberPinned(ctx, cid, "ghost", &now); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("SetMemberPinned ghost → %v", err)
	}

	// Mute / unmute.
	if err := r.SetMemberMuted(ctx, cid, "u1", &now); err != nil {
		t.Fatalf("SetMemberMuted : %v", err)
	}
	if err := r.SetMemberMuted(ctx, cid, "u1", nil); err != nil {
		t.Fatalf("SetMemberMuted unset : %v", err)
	}
	if err := r.SetMemberMuted(ctx, cid, "ghost", nil); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("SetMemberMuted ghost → %v", err)
	}

	// Cleared.
	if err := r.SetMemberCleared(ctx, cid, "u1", now); err != nil {
		t.Fatalf("SetMemberCleared : %v", err)
	}
	if err := r.SetMemberCleared(ctx, cid, "ghost", now); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("SetMemberCleared ghost → %v", err)
	}

	// Read / delivered.
	if err := r.SetMemberRead(ctx, cid, "u1", now); err != nil {
		t.Fatalf("SetMemberRead : %v", err)
	}
	if err := r.SetMemberRead(ctx, cid, "ghost", now); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("SetMemberRead ghost → %v", err)
	}
	if err := r.SetMemberDelivered(ctx, cid, "u1", now); err != nil {
		t.Fatalf("SetMemberDelivered : %v", err)
	}
	if err := r.SetMemberDelivered(ctx, cid, "ghost", now); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("SetMemberDelivered ghost → %v", err)
	}
}

// --- Messages ----------------------------------------------------------------

func TestRepo_Messages(t *testing.T) {
	r, ctx := newRepo(t)
	cid := seedConv(t, r, ctx, models.TypeGroup, map[string]string{"u1": models.MemberOwner})

	// Insert + list.
	var first bson.ObjectID
	for i := 0; i < 3; i++ {
		m := &models.Message{
			ConversationID: cid, SenderID: "u1", Ciphertext: "c", Nonce: "n",
			CreatedAt: time.Now().Add(time.Duration(i) * time.Millisecond),
		}
		if err := r.InsertMessage(ctx, m); err != nil {
			t.Fatalf("InsertMessage : %v", err)
		}
		if i == 0 {
			first = m.ID
		}
	}

	msgs, err := r.ListMessages(ctx, cid, 10, nil, nil)
	if err != nil || len(msgs) != 3 {
		t.Fatalf("ListMessages = %d, %v ; attendu 3", len(msgs), err)
	}
	// Tri ascendant.
	if !msgs[0].CreatedAt.Before(msgs[2].CreatedAt) {
		t.Fatalf("ListMessages doit être trié ascendant")
	}

	// Pagination `before` : aucun message avant le 1er.
	if got, _ := r.ListMessages(ctx, cid, 10, &first, nil); len(got) != 0 {
		t.Fatalf("ListMessages(before=first) = %d, attendu 0", len(got))
	}
	// `after` : tout après le futur = 0.
	future := time.Now().Add(time.Hour)
	if got, _ := r.ListMessages(ctx, cid, 10, nil, &future); len(got) != 0 {
		t.Fatalf("ListMessages(after=future) = %d, attendu 0", len(got))
	}

	// HasMessagesAfter.
	past := time.Now().Add(-time.Hour)
	if ok, _ := r.HasMessagesAfter(ctx, cid, past); !ok {
		t.Fatalf("HasMessagesAfter(past) attendu true")
	}
	if ok, _ := r.HasMessagesAfter(ctx, cid, future); ok {
		t.Fatalf("HasMessagesAfter(future) attendu false")
	}

	// GetMessage / GetMessageByID.
	if _, err := r.GetMessage(ctx, cid, first); err != nil {
		t.Fatalf("GetMessage : %v", err)
	}
	if _, err := r.GetMessageByID(ctx, first); err != nil {
		t.Fatalf("GetMessageByID : %v", err)
	}

	// Edit (conserve l'original).
	upd, err := r.UpdateMessageCiphertext(ctx, &models.Message{ID: first, ConversationID: cid, Ciphertext: "c", Nonce: "n"}, "c2", "n2", time.Now())
	if err != nil || upd.Ciphertext != "c2" || upd.OriginalCiphertext != "c" {
		t.Fatalf("UpdateMessageCiphertext = %+v, %v", upd, err)
	}

	// Soft delete (tombstone).
	del, err := r.SoftDeleteMessage(ctx, cid, first, time.Now())
	if err != nil || del.Ciphertext != "" || del.DeletedAt == nil {
		t.Fatalf("SoftDeleteMessage = %+v, %v", del, err)
	}

	// Moderate soft delete.
	mod, err := r.ModerateSoftDelete(ctx, first, time.Now())
	if err != nil || !mod.DeletedByModeration {
		t.Fatalf("ModerateSoftDelete = %+v, %v", mod, err)
	}

	// Cascades.
	if err := r.DeleteMessagesByConversation(ctx, cid); err != nil {
		t.Fatalf("DeleteMessagesByConversation : %v", err)
	}
	if err := r.DeleteMembersByConversation(ctx, cid); err != nil {
		t.Fatalf("DeleteMembersByConversation : %v", err)
	}
}

func TestRepo_CountUnreadAndPurge(t *testing.T) {
	r, ctx := newRepo(t)
	cid := seedConv(t, r, ctx, models.TypeDM, map[string]string{"u1": models.MemberTalker, "u2": models.MemberTalker})

	// u2 envoie → u1 a un non-lu.
	_ = r.InsertMessage(ctx, &models.Message{ConversationID: cid, SenderID: "u2", Ciphertext: "c", Nonce: "n", CreatedAt: time.Now()})

	if n, err := r.CountUnreadConversations(ctx, "u1"); err != nil || n != 1 {
		t.Fatalf("CountUnread u1 = %d, %v ; attendu 1", n, err)
	}
	// u2 ne compte pas ses propres messages.
	if n, _ := r.CountUnreadConversations(ctx, "u2"); n != 0 {
		t.Fatalf("CountUnread u2 = %d, attendu 0", n)
	}

	// Après lecture, plus de non-lu.
	_ = r.SetMemberRead(ctx, cid, "u1", time.Now())
	if n, _ := r.CountUnreadConversations(ctx, "u1"); n != 0 {
		t.Fatalf("CountUnread u1 après lecture = %d, attendu 0", n)
	}

	// PurgeUser : efface clés + appartenances + messages de u2.
	_ = r.UpsertKey(ctx, "u2", "pub")
	if err := r.PurgeUser(ctx, "u2"); err != nil {
		t.Fatalf("PurgeUser : %v", err)
	}
	if _, err := r.GetKey(ctx, "u2"); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("clé u2 doit être purgée")
	}
	if _, err := r.GetMember(ctx, cid, "u2"); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("appartenance u2 doit être purgée")
	}
}
