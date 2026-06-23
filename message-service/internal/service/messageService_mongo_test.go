package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/webdad/message-service/internal/database"
	"github.com/webdad/message-service/internal/models"
	"github.com/webdad/message-service/internal/mongotest"
	"github.com/webdad/message-service/internal/notifier"
	"github.com/webdad/message-service/internal/repository"
)

// spyNotifier collecte les événements émis (au lieu d'appeler le réseau).
type spyNotifier struct{ events []notifier.Event }

func (s *spyNotifier) Emit(ev notifier.Event) { s.events = append(s.events, ev) }

func (s *spyNotifier) countType(typ string) int {
	n := 0
	for _, e := range s.events {
		if e.Type == typ {
			n++
		}
	}
	return n
}

// newSvc monte service + dépôt sur une base jetable, avec notifier espion.
func newSvc(t *testing.T) (*MessageService, *repository.MessageRepository, *spyNotifier, context.Context) {
	t.Helper()
	db := mongotest.DB(t)
	ctx := context.Background()
	if err := database.EnsureSchema(ctx, db); err != nil {
		t.Fatalf("EnsureSchema : %v", err)
	}
	repo := repository.NewMessageRepository(db)
	spy := &spyNotifier{}
	svc := NewMessageService(repo)
	svc.SetNotifier(spy)
	svc.SetNotifier(nil) // no-op : ne doit pas écraser le notifier existant
	return svc, repo, spy, ctx
}

const (
	uA = "user-aaaa"
	uB = "user-bbbb"
	uC = "user-cccc"
)

func env(ids ...string) map[string]string {
	m := make(map[string]string, len(ids))
	for _, id := range ids {
		m[id] = "env-" + id
	}
	return m
}

// --- Clés / sauvegardes ------------------------------------------------------

func TestSvc_KeysAndBackups(t *testing.T) {
	svc, _, _, ctx := newSvc(t)

	if _, err := svc.GetKey(ctx, uA); !errors.Is(err, ErrKeyNotFound) {
		t.Fatalf("GetKey absente → %v, attendu ErrKeyNotFound", err)
	}
	if err := svc.PublishKey(ctx, uA, "pub"); err != nil {
		t.Fatalf("PublishKey : %v", err)
	}
	if k, err := svc.GetKey(ctx, uA); err != nil || k.PublicKey != "pub" {
		t.Fatalf("GetKey = %+v, %v", k, err)
	}

	if _, err := svc.GetBackup(ctx, uA); !errors.Is(err, ErrBackupNotFound) {
		t.Fatalf("GetBackup absente → %v", err)
	}
	if ok, _ := svc.BackupStatus(ctx, uA); ok {
		t.Fatalf("BackupStatus doit être false")
	}
	if err := svc.PutBackup(ctx, uA, models.PutBackupRequest{Salt: "s", Nonce: "n", WrappedPrivateKey: "w", KDFParams: "k", PublicKey: "p"}); err != nil {
		t.Fatalf("PutBackup : %v", err)
	}
	if ok, _ := svc.BackupStatus(ctx, uA); !ok {
		t.Fatalf("BackupStatus doit être true")
	}
	if b, err := svc.GetBackup(ctx, uA); err != nil || b.Salt != "s" {
		t.Fatalf("GetBackup = %+v, %v", b, err)
	}

	// PurgeUser idempotent.
	if err := svc.PurgeUser(ctx, uA); err != nil {
		t.Fatalf("PurgeUser : %v", err)
	}
}

// --- DM ----------------------------------------------------------------------

func TestSvc_CreateDM(t *testing.T) {
	svc, _, _, ctx := newSvc(t)

	if _, err := svc.CreateDM(ctx, uA, uA, nil); !errors.Is(err, ErrSelfConversation) {
		t.Fatalf("DM avec soi-même → %v", err)
	}
	if _, err := svc.CreateDM(ctx, uA, uB, env(uA)); !errors.Is(err, ErrMissingEnvelope) {
		t.Fatalf("enveloppe manquante → %v", err)
	}

	view, err := svc.CreateDM(ctx, uA, uB, env(uA, uB))
	if err != nil || view.Type != models.TypeDM || view.MyRole != models.MemberTalker {
		t.Fatalf("CreateDM = %+v, %v", view, err)
	}
	// Idempotent : même DM renvoyé.
	again, err := svc.CreateDM(ctx, uB, uA, env(uA, uB))
	if err != nil || again.ID != view.ID {
		t.Fatalf("CreateDM idempotent = %+v, %v", again, err)
	}
}

// --- Groupe ------------------------------------------------------------------

func TestSvc_CreateGroupValidation(t *testing.T) {
	svc, _, _, ctx := newSvc(t)

	if _, err := svc.CreateGroup(ctx, uA, "t", "n", nil); !errors.Is(err, ErrInvalidGroup) {
		t.Fatalf("groupe sans enveloppes → %v", err)
	}
	if _, err := svc.CreateGroup(ctx, uA, "t", "n", env(uB)); !errors.Is(err, ErrInvalidGroup) {
		t.Fatalf("créateur absent → %v", err)
	}
	// Cap dépassé.
	big := map[string]string{uA: "env"}
	for i := 0; i < MaxTalkers; i++ {
		big[string(rune('a'+i))+"-x"] = "env"
	}
	if _, err := svc.CreateGroup(ctx, uA, "t", "n", big); !errors.Is(err, ErrTalkersFull) {
		t.Fatalf("cap dépassé → %v", err)
	}
	// Enveloppe vide.
	if _, err := svc.CreateGroup(ctx, uA, "t", "n", map[string]string{uA: "e", uB: "  "}); !errors.Is(err, ErrMissingEnvelope) {
		t.Fatalf("enveloppe vide → %v", err)
	}
}

func TestSvc_CreateGroupSuccess(t *testing.T) {
	svc, _, _, ctx := newSvc(t)
	view, err := svc.CreateGroup(ctx, uA, "ct", "nz", env(uA, uB, uC))
	if err != nil || view.Type != models.TypeGroup || view.MyRole != models.MemberOwner {
		t.Fatalf("CreateGroup = %+v, %v", view, err)
	}
	if len(view.MemberIDs) != 3 {
		t.Fatalf("3 membres attendus, %d", len(view.MemberIDs))
	}
}

// --- Communauté --------------------------------------------------------------

func TestSvc_Community(t *testing.T) {
	svc, _, _, ctx := newSvc(t)

	if _, err := svc.CreateCommunity(ctx, uA, "", "ck"); !errors.Is(err, ErrInvalidCommunity) {
		t.Fatalf("communauté sans titre → %v", err)
	}
	view, err := svc.CreateCommunity(ctx, uA, "Gophers", "content-key")
	if err != nil || view.ContentKey != "content-key" {
		t.Fatalf("CreateCommunity = %+v, %v", view, err)
	}

	// Join : not found / not community / success / idempotent.
	if _, _, err := svc.JoinCommunity(ctx, "deadbeefdeadbeefdeadbeef", uB); !errors.Is(err, ErrConversationNotFound) {
		t.Fatalf("Join introuvable → %v", err)
	}
	v2, notify, err := svc.JoinCommunity(ctx, view.ID, uB)
	if err != nil || v2.MyRole != models.MemberViewer || len(notify) == 0 {
		t.Fatalf("Join = %+v, notify=%v, %v", v2, notify, err)
	}
	if v3, _, err := svc.JoinCommunity(ctx, view.ID, uB); err != nil || v3.MyRole != models.MemberViewer {
		t.Fatalf("Join idempotent = %+v, %v", v3, err)
	}

	// SetMemberRole.
	if _, err := svc.SetMemberRole(ctx, view.ID, uA, uB, "bad"); !errors.Is(err, ErrInvalidRole) {
		t.Fatalf("rôle invalide → %v", err)
	}
	if _, err := svc.SetMemberRole(ctx, view.ID, uB, uA, models.MemberTalker); !errors.Is(err, ErrOwnerOnly) {
		t.Fatalf("non-owner → %v", err)
	}
	if _, err := svc.SetMemberRole(ctx, view.ID, uA, "ghost", models.MemberTalker); !errors.Is(err, ErrTargetNotMember) {
		t.Fatalf("cible non membre → %v", err)
	}
	notify, err = svc.SetMemberRole(ctx, view.ID, uA, uB, models.MemberTalker)
	if err != nil || len(notify) == 0 {
		t.Fatalf("promotion = %v, %v", notify, err)
	}
	// Owner non modifiable.
	if _, err := svc.SetMemberRole(ctx, view.ID, uA, uA, models.MemberViewer); !errors.Is(err, ErrOwnerOnly) {
		t.Fatalf("owner non modifiable → %v", err)
	}

	// ListCommunities.
	items, err := svc.ListCommunities(ctx, uB, 10, 0, "")
	if err != nil || len(items) != 1 || !items[0].IsMember {
		t.Fatalf("ListCommunities = %+v, %v", items, err)
	}

	// SetMemberRole sur un non-community → ErrNotCommunity.
	g, _ := svc.CreateGroup(ctx, uA, "t", "n", env(uA, uB))
	if _, err := svc.SetMemberRole(ctx, g.ID, uA, uB, models.MemberTalker); !errors.Is(err, ErrNotCommunity) {
		t.Fatalf("SetMemberRole sur groupe → %v", err)
	}
	// JoinCommunity sur un groupe → ErrNotCommunity.
	if _, _, err := svc.JoinCommunity(ctx, g.ID, uC); !errors.Is(err, ErrNotCommunity) {
		t.Fatalf("Join sur groupe → %v", err)
	}
}

// --- Membres (groupe) --------------------------------------------------------

func TestSvc_Members(t *testing.T) {
	svc, _, _, ctx := newSvc(t)
	g, _ := svc.CreateGroup(ctx, uA, "t", "n", env(uA, uB))

	// AddMember : non membre.
	if _, err := svc.AddMember(ctx, g.ID, "stranger", uC, "env"); !errors.Is(err, ErrNotMember) {
		t.Fatalf("AddMember non membre → %v", err)
	}
	// Déjà membre.
	if _, err := svc.AddMember(ctx, g.ID, uA, uB, "env"); !errors.Is(err, ErrAlreadyMember) {
		t.Fatalf("AddMember déjà membre → %v", err)
	}
	// Succès.
	if notify, err := svc.AddMember(ctx, g.ID, uA, uC, "env"); err != nil || len(notify) != 3 {
		t.Fatalf("AddMember = %v, %v", notify, err)
	}
	// AddMember sur DM → ErrNotGroup.
	dm, _ := svc.CreateDM(ctx, uA, uB, env(uA, uB))
	if _, err := svc.AddMember(ctx, dm.ID, uA, uC, "env"); !errors.Is(err, ErrNotGroup) {
		t.Fatalf("AddMember DM → %v", err)
	}

	// ListMembers (membre requis).
	if _, err := svc.ListMembers(ctx, g.ID, "stranger"); !errors.Is(err, ErrNotMember) {
		t.Fatalf("ListMembers non membre → %v", err)
	}
	if ms, err := svc.ListMembers(ctx, g.ID, uA); err != nil || len(ms) != 3 {
		t.Fatalf("ListMembers = %v, %v", ms, err)
	}

	// RemoveMember : owner ne peut pas quitter.
	if _, err := svc.RemoveMember(ctx, g.ID, uA, uA); !errors.Is(err, ErrOwnerCannotLeave) {
		t.Fatalf("owner quitte → %v", err)
	}
	// talker exclut autrui → ErrOwnerOnly.
	if _, err := svc.RemoveMember(ctx, g.ID, uB, uC); !errors.Is(err, ErrOwnerOnly) {
		t.Fatalf("non-owner exclut → %v", err)
	}
	// owner exclut un membre.
	if notify, err := svc.RemoveMember(ctx, g.ID, uA, uC); err != nil || len(notify) != 3 {
		t.Fatalf("owner exclut = %v, %v", notify, err)
	}
	// owner exclut un fantôme → ErrTargetNotMember.
	if _, err := svc.RemoveMember(ctx, g.ID, uA, "ghost"); !errors.Is(err, ErrTargetNotMember) {
		t.Fatalf("exclut fantôme → %v", err)
	}
	// talker quitte (self) → OK.
	if _, err := svc.RemoveMember(ctx, g.ID, uB, uB); err != nil {
		t.Fatalf("talker quitte : %v", err)
	}
	// RemoveMember sur DM → ErrNotManageable.
	if _, err := svc.RemoveMember(ctx, dm.ID, uA, uB); !errors.Is(err, ErrNotManageable) {
		t.Fatalf("RemoveMember DM → %v", err)
	}
}

// --- Update / Delete groupe --------------------------------------------------

func TestSvc_UpdateDeleteGroup(t *testing.T) {
	svc, _, _, ctx := newSvc(t)
	g, _ := svc.CreateGroup(ctx, uA, "t", "n", env(uA, uB))

	// UpdateGroup non-owner.
	if _, err := svc.UpdateGroup(ctx, g.ID, uB, "new", "nz"); !errors.Is(err, ErrOwnerOnly) {
		t.Fatalf("UpdateGroup non-owner → %v", err)
	}
	if v, err := svc.UpdateGroup(ctx, g.ID, uA, "new", "nz"); err != nil || v.Title != "new" {
		t.Fatalf("UpdateGroup = %+v, %v", v, err)
	}

	// DeleteGroup non-owner.
	if _, err := svc.DeleteGroup(ctx, g.ID, uB); !errors.Is(err, ErrOwnerOnly) {
		t.Fatalf("DeleteGroup non-owner → %v", err)
	}
	if notify, err := svc.DeleteGroup(ctx, g.ID, uA); err != nil || len(notify) != 2 {
		t.Fatalf("DeleteGroup = %v, %v", notify, err)
	}
	// Conversation supprimée → GetConversation 404.
	if _, err := svc.GetConversation(ctx, g.ID, uA); !errors.Is(err, ErrConversationNotFound) {
		t.Fatalf("GetConversation après delete → %v", err)
	}

	// DeleteGroup sur DM → ErrNotManageable.
	dm, _ := svc.CreateDM(ctx, uA, uB, env(uA, uB))
	if _, err := svc.DeleteGroup(ctx, dm.ID, uA); !errors.Is(err, ErrNotManageable) {
		t.Fatalf("DeleteGroup DM → %v", err)
	}
}

// --- Conversations : list / pin / mute / clear / read ------------------------

func TestSvc_ConversationState(t *testing.T) {
	svc, _, _, ctx := newSvc(t)
	dm, _ := svc.CreateDM(ctx, uA, uB, env(uA, uB))
	g, _ := svc.CreateGroup(ctx, uA, "t", "n", env(uA, uC))

	// Liste : 2 conversations pour uA.
	views, err := svc.ListConversations(ctx, uA)
	if err != nil || len(views) != 2 {
		t.Fatalf("ListConversations = %d, %v", len(views), err)
	}

	// Pin g → remonte en tête.
	if _, err := svc.PinConversation(ctx, g.ID, uA, true); err != nil {
		t.Fatalf("Pin : %v", err)
	}
	views, _ = svc.ListConversations(ctx, uA)
	if views[0].ID != g.ID {
		t.Fatalf("conversation épinglée doit être en tête")
	}
	if _, err := svc.PinConversation(ctx, g.ID, uA, false); err != nil {
		t.Fatalf("Unpin : %v", err)
	}

	// Mute / unmute.
	if v, err := svc.MuteConversation(ctx, dm.ID, uA, true); err != nil || !v.Muted {
		t.Fatalf("Mute = %+v, %v", v, err)
	}
	if v, err := svc.MuteConversation(ctx, dm.ID, uA, false); err != nil || v.Muted {
		t.Fatalf("Unmute = %+v, %v", v, err)
	}

	// MarkRead + receipts diffusés (les autres membres).
	at, others, err := svc.MarkRead(ctx, dm.ID, uA)
	if err != nil || at.IsZero() || len(others) != 1 {
		t.Fatalf("MarkRead = %v, %v, %v", at, others, err)
	}

	// MarkDelivered + TouchDelivered.
	svc.MarkDelivered(ctx, dm.ID, []string{uB}, time.Now())
	if _, others, err := svc.TouchDelivered(ctx, dm.ID, uA); err != nil || len(others) != 1 {
		t.Fatalf("TouchDelivered = %v, %v", others, err)
	}

	// TypingTargets.
	if targets, err := svc.TypingTargets(ctx, dm.ID, uA); err != nil || len(targets) != 1 {
		t.Fatalf("TypingTargets = %v, %v", targets, err)
	}
	if _, err := svc.TypingTargets(ctx, dm.ID, "stranger"); !errors.Is(err, ErrNotMember) {
		t.Fatalf("TypingTargets non membre → %v", err)
	}

	// UnreadCount.
	if _, err := svc.UnreadCount(ctx, uA); err != nil {
		t.Fatalf("UnreadCount : %v", err)
	}

	// IsMember.
	if ok, _ := svc.IsMember(ctx, dm.ID, uA); !ok {
		t.Fatalf("IsMember uA attendu true")
	}
	if ok, _ := svc.IsMember(ctx, dm.ID, "stranger"); ok {
		t.Fatalf("IsMember stranger attendu false")
	}

	// Clear : masque la conversation (sans message plus récent).
	if err := svc.ClearConversation(ctx, dm.ID, uA); err != nil {
		t.Fatalf("Clear : %v", err)
	}
	views, _ = svc.ListConversations(ctx, uA)
	for _, v := range views {
		if v.ID == dm.ID {
			t.Fatalf("conversation effacée côté user doit être masquée")
		}
	}

	// GetConversation : id invalide + non membre + succès.
	if _, err := svc.GetConversation(ctx, "not-hex", uA); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("GetConversation id invalide → %v", err)
	}
	if _, err := svc.GetConversation(ctx, g.ID, "stranger"); !errors.Is(err, ErrNotMember) {
		t.Fatalf("GetConversation non membre → %v", err)
	}
	if v, err := svc.GetConversation(ctx, g.ID, uA); err != nil || v.ID != g.ID {
		t.Fatalf("GetConversation = %+v, %v", v, err)
	}
}

// ListConversations doit RÉVÉLER une conversation effacée si un message plus
// récent arrive (couvre la branche hasNewer).
func TestSvc_ClearedReappears(t *testing.T) {
	svc, _, _, ctx := newSvc(t)
	dm, _ := svc.CreateDM(ctx, uA, uB, env(uA, uB))
	if err := svc.ClearConversation(ctx, dm.ID, uA); err != nil {
		t.Fatalf("Clear : %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	// uB écrit après le clear.
	if _, _, err := svc.SendMessage(ctx, dm.ID, uB, "c", "n", nil); err != nil {
		t.Fatalf("SendMessage : %v", err)
	}
	views, _ := svc.ListConversations(ctx, uA)
	found := false
	for _, v := range views {
		if v.ID == dm.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("conversation effacée doit réapparaître après un nouveau message")
	}
}

// --- Messages ----------------------------------------------------------------

func TestSvc_Messages(t *testing.T) {
	svc, _, spy, ctx := newSvc(t)
	g, _ := svc.CreateGroup(ctx, uA, "t", "n", env(uA, uB))

	// Send : non membre.
	if _, _, err := svc.SendMessage(ctx, g.ID, "stranger", "c", "n", nil); !errors.Is(err, ErrNotMember) {
		t.Fatalf("Send non membre → %v", err)
	}

	// Send succès + notif `message` + mention de uB.
	msg, members, err := svc.SendMessage(ctx, g.ID, uA, "ciph", "nonce", []string{uB, "ghost"})
	if err != nil || msg.Ciphertext != "ciph" || len(members) != 2 {
		t.Fatalf("Send = %+v, %v", msg, err)
	}
	if spy.countType(notifier.TypeMessage) != 1 {
		t.Fatalf("1 notif message attendue, %d", spy.countType(notifier.TypeMessage))
	}
	if spy.countType(notifier.TypeMessageMention) != 1 {
		t.Fatalf("1 mention attendue (ghost ignoré), %d", spy.countType(notifier.TypeMessageMention))
	}

	// ListMessages (membre requis + succès).
	if _, err := svc.ListMessages(ctx, g.ID, "stranger", 0, ""); !errors.Is(err, ErrNotMember) {
		t.Fatalf("ListMessages non membre → %v", err)
	}
	if _, err := svc.ListMessages(ctx, g.ID, uA, 0, "not-hex"); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("ListMessages before invalide → %v", err)
	}
	msgs, err := svc.ListMessages(ctx, g.ID, uA, 10, "")
	if err != nil || len(msgs) != 1 {
		t.Fatalf("ListMessages = %d, %v", len(msgs), err)
	}

	// Edit : pas l'auteur.
	if _, _, err := svc.EditMessage(ctx, g.ID, msg.ID.Hex(), uB, "c2", "n2", nil); !errors.Is(err, ErrNotMessageOwner) {
		t.Fatalf("Edit pas auteur → %v", err)
	}
	// Edit : message introuvable.
	if _, _, err := svc.EditMessage(ctx, g.ID, "deadbeefdeadbeefdeadbeef", uA, "c2", "n2", nil); !errors.Is(err, ErrMessageNotFound) {
		t.Fatalf("Edit introuvable → %v", err)
	}
	// Edit : id invalide.
	if _, _, err := svc.EditMessage(ctx, g.ID, "not-hex", uA, "c2", "n2", nil); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("Edit id invalide → %v", err)
	}
	// Edit succès + mention.
	upd, _, err := svc.EditMessage(ctx, g.ID, msg.ID.Hex(), uA, "c2", "n2", []string{uB})
	if err != nil || upd.Ciphertext != "c2" {
		t.Fatalf("Edit = %+v, %v", upd, err)
	}

	// Delete : id invalide / introuvable / succès.
	if _, _, err := svc.DeleteMessage(ctx, g.ID, "not-hex", uA); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("Delete id invalide → %v", err)
	}
	if _, _, err := svc.DeleteMessage(ctx, g.ID, "deadbeefdeadbeefdeadbeef", uA); !errors.Is(err, ErrMessageNotFound) {
		t.Fatalf("Delete introuvable → %v", err)
	}
	del, _, err := svc.DeleteMessage(ctx, g.ID, msg.ID.Hex(), uA)
	if err != nil || del.DeletedAt == nil {
		t.Fatalf("Delete = %+v, %v", del, err)
	}
}

// SendMessage par un viewer (lecture seule) → ErrCannotWrite.
func TestSvc_SendViewerForbidden(t *testing.T) {
	svc, _, _, ctx := newSvc(t)
	com, _ := svc.CreateCommunity(ctx, uA, "C", "ck")
	if _, _, err := svc.JoinCommunity(ctx, com.ID, uB); err != nil {
		t.Fatalf("Join : %v", err)
	}
	if _, _, err := svc.SendMessage(ctx, com.ID, uB, "c", "n", nil); !errors.Is(err, ErrCannotWrite) {
		t.Fatalf("viewer écrit → %v", err)
	}
	// Communauté : pas de notif `message`.
	if _, _, err := svc.SendMessage(ctx, com.ID, uA, "c", "n", nil); err != nil {
		t.Fatalf("owner écrit : %v", err)
	}
}

// DeleteMessage : un tiers sans droit → ErrCannotDelete ; owner de groupe → OK.
func TestSvc_DeletePermissions(t *testing.T) {
	svc, _, _, ctx := newSvc(t)
	g, _ := svc.CreateGroup(ctx, uA, "t", "n", env(uA, uB))
	// uB poste.
	msg, _, _ := svc.SendMessage(ctx, g.ID, uB, "c", "n", nil)
	// uC n'est pas membre → ErrNotMember (avant la règle de suppression).
	if _, _, err := svc.DeleteMessage(ctx, g.ID, msg.ID.Hex(), uC); !errors.Is(err, ErrNotMember) {
		t.Fatalf("Delete non membre → %v", err)
	}
	// owner uA supprime le message de uB (modération de groupe).
	if _, _, err := svc.DeleteMessage(ctx, g.ID, msg.ID.Hex(), uA); err != nil {
		t.Fatalf("owner supprime : %v", err)
	}
}

// ModerateDeleteMessage : id invalide / introuvable / succès.
func TestSvc_ModerateDelete(t *testing.T) {
	svc, _, _, ctx := newSvc(t)
	g, _ := svc.CreateGroup(ctx, uA, "t", "n", env(uA, uB))
	msg, _, _ := svc.SendMessage(ctx, g.ID, uA, "c", "n", nil)

	if _, _, err := svc.ModerateDeleteMessage(ctx, "not-hex"); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("Moderate id invalide → %v", err)
	}
	if _, _, err := svc.ModerateDeleteMessage(ctx, "deadbeefdeadbeefdeadbeef"); !errors.Is(err, ErrMessageNotFound) {
		t.Fatalf("Moderate introuvable → %v", err)
	}
	del, members, err := svc.ModerateDeleteMessage(ctx, msg.ID.Hex())
	if err != nil || !del.DeletedByModeration || len(members) != 2 {
		t.Fatalf("Moderate = %+v, %v, %v", del, members, err)
	}
}
