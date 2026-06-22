package service

import (
	"errors"
	"testing"
	"time"

	"github.com/webdad/message-service/internal/models"
)

// ListMessages avec un curseur `before` valide (chemin de pagination arrière).
func TestSvc_ListMessagesBeforeCursor(t *testing.T) {
	svc, _, _, ctx := newSvc(t)
	g, _ := svc.CreateGroup(ctx, uA, "t", "n", env(uA))

	var lastID string
	for i := 0; i < 3; i++ {
		m, _, err := svc.SendMessage(ctx, g.ID, uA, "c", "n", nil)
		if err != nil {
			t.Fatalf("Send : %v", err)
		}
		lastID = m.ID.Hex()
		time.Sleep(time.Millisecond)
	}
	// before = dernier message → renvoie les 2 antérieurs.
	msgs, err := svc.ListMessages(ctx, g.ID, uA, 10, lastID)
	if err != nil || len(msgs) != 2 {
		t.Fatalf("ListMessages(before) = %d, %v ; attendu 2", len(msgs), err)
	}
}

// DeleteMessage : un membre sans droit (DM, message d'autrui) → ErrCannotDelete.
func TestSvc_DeleteCannotDeleteDM(t *testing.T) {
	svc, _, _, ctx := newSvc(t)
	dm, _ := svc.CreateDM(ctx, uA, uB, env(uA, uB))
	msg, _, _ := svc.SendMessage(ctx, dm.ID, uA, "c", "n", nil)
	// uB (talker, pas owner/admin, pas l'auteur) ne peut pas supprimer en DM.
	if _, _, err := svc.DeleteMessage(ctx, dm.ID, msg.ID.Hex(), uB); !errors.Is(err, ErrCannotDelete) {
		t.Fatalf("Delete par non-auteur en DM → %v, attendu ErrCannotDelete", err)
	}
}

// AddMember sur un groupe au maximum de talkers → ErrTalkersFull.
func TestSvc_AddMemberCapFull(t *testing.T) {
	svc, _, _, ctx := newSvc(t)
	// Groupe rempli au cap (MaxTalkers membres, créateur inclus).
	envs := map[string]string{uA: "env-" + uA}
	for i := 0; i < MaxTalkers-1; i++ {
		envs["member-"+string(rune('A'+i))] = "env"
	}
	g, err := svc.CreateGroup(ctx, uA, "t", "n", envs)
	if err != nil {
		t.Fatalf("CreateGroup : %v", err)
	}
	if _, err := svc.AddMember(ctx, g.ID, uA, "one-too-many", "env"); !errors.Is(err, ErrTalkersFull) {
		t.Fatalf("AddMember au-delà du cap → %v, attendu ErrTalkersFull", err)
	}
}

// ListConversations ignore les appartenances orphelines : conversation_id non
// parsable (continue) ou conversation supprimée (continue).
func TestSvc_ListConversationsOrphans(t *testing.T) {
	svc, repo, _, ctx := newSvc(t)

	// Appartenance avec un conversation_id non-ObjectID → ignorée (parseID).
	_ = repo.AddMember(ctx, &models.Member{
		ConversationID: "not-an-objectid", UserID: uA, Role: models.MemberTalker, CreatedAt: time.Now(),
	})
	// Appartenance vers une conversation inexistante → ignorée (GetConversation).
	_ = repo.AddMember(ctx, &models.Member{
		ConversationID: "507f1f77bcf86cd799439011", UserID: uA, Role: models.MemberTalker, CreatedAt: time.Now(),
	})

	views, err := svc.ListConversations(ctx, uA)
	if err != nil {
		t.Fatalf("ListConversations : %v", err)
	}
	if len(views) != 0 {
		t.Fatalf("appartenances orphelines doivent être ignorées, obtenu %d", len(views))
	}
}

// SetMemberRole : promotion viewer→talker dans une communauté saturée → ErrTalkersFull.
func TestSvc_SetMemberRoleCapFull(t *testing.T) {
	svc, repo, _, ctx := newSvc(t)
	com, err := svc.CreateCommunity(ctx, uA, "C", "ck")
	if err != nil {
		t.Fatalf("CreateCommunity : %v", err)
	}
	now := time.Now()
	// Sature les talkers : owner (1) + (MaxTalkers-1) talkers = MaxTalkers.
	for i := 0; i < MaxTalkers-1; i++ {
		_ = repo.AddMember(ctx, &models.Member{
			ConversationID: com.ID, UserID: "talker-" + string(rune('A'+i)), Role: models.MemberTalker, CreatedAt: now,
		})
	}
	// Un viewer supplémentaire à promouvoir.
	_ = repo.AddMember(ctx, &models.Member{
		ConversationID: com.ID, UserID: uB, Role: models.MemberViewer, CreatedAt: now,
	})

	if _, err := svc.SetMemberRole(ctx, com.ID, uA, uB, models.MemberTalker); !errors.Is(err, ErrTalkersFull) {
		t.Fatalf("promotion au-delà du cap → %v, attendu ErrTalkersFull", err)
	}
}
