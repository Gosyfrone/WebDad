package service

import (
	"context"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/webdad/message-service/internal/database"
	"github.com/webdad/message-service/internal/models"
	"github.com/webdad/message-service/internal/mongotest"
	"github.com/webdad/message-service/internal/repository"
)

// faultEnv expose le service ET la base, pour pouvoir CORROMPRE un document
// (date stockée en chaîne, validation contournée) : le 1er accès au dépôt réussit
// mais le décodage du document corrompu échoue ensuite — ce qui exerce les
// branches de remontée d'erreur situées APRÈS le premier appel repo (inatteignables
// autrement avec une vraie Mongo).
type faultEnv struct {
	svc  *MessageService
	repo *repository.MessageRepository
	db   *mongo.Database
	ctx  context.Context
}

func newFaultEnv(t *testing.T) *faultEnv {
	t.Helper()
	db := mongotest.DB(t)
	ctx := context.Background()
	if err := database.EnsureSchema(ctx, db); err != nil {
		t.Fatalf("EnsureSchema : %v", err)
	}
	repo := repository.NewMessageRepository(db)
	return &faultEnv{svc: NewMessageService(repo), repo: repo, db: db, ctx: ctx}
}

// corrupt écrase created_at par une chaîne (type invalide) en contournant le
// validateur → tout décodage ultérieur du document échoue.
func (e *faultEnv) corrupt(t *testing.T, coll string, id bson.ObjectID) {
	t.Helper()
	_, err := e.db.Collection(coll).UpdateOne(e.ctx,
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"created_at": "pas-une-date"}},
		options.UpdateOne().SetBypassDocumentValidation(true))
	if err != nil {
		t.Fatalf("corrupt %s : %v", coll, err)
	}
}

func (e *faultEnv) corruptMember(t *testing.T, convID, userID string) {
	t.Helper()
	_, err := e.db.Collection("members").UpdateOne(e.ctx,
		bson.M{"conversation_id": convID, "user_id": userID},
		bson.M{"$set": bson.M{"created_at": "pas-une-date"}},
		options.UpdateOne().SetBypassDocumentValidation(true))
	if err != nil {
		t.Fatalf("corrupt member : %v", err)
	}
}

// hexID convertit l'id hex d'une vue en ObjectID.
func hexID(t *testing.T, id string) bson.ObjectID {
	t.Helper()
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		t.Fatalf("ObjectIDFromHex(%q) : %v", id, err)
	}
	return oid
}

// Une conversation corrompue : requireMember (lecture des membres) réussit, mais
// getConversationByID/GetConversation échoue au décodage → couvre les branches
// d'erreur de chargement de conversation de nombreuses méthodes.
func TestSvc_Fault_CorruptConversation(t *testing.T) {
	e := newFaultEnv(t)
	g, err := e.svc.CreateGroup(e.ctx, uA, "t", "n", env(uA, uB))
	if err != nil {
		t.Fatalf("CreateGroup : %v", err)
	}
	// Insère un message valide (pour les méthodes qui le lisent avant la conv).
	msg, _, err := e.svc.SendMessage(e.ctx, g.ID, uA, "c", "n", nil)
	if err != nil {
		t.Fatalf("SendMessage : %v", err)
	}
	e.corrupt(t, "conversations", hexID(t, g.ID))

	mustErr := func(name string, err error) {
		if err == nil {
			t.Errorf("%s : erreur attendue (conversation corrompue)", name)
		}
	}

	_, err = e.svc.AddMember(e.ctx, g.ID, uA, uC, "env")
	mustErr("AddMember", err)
	_, err = e.svc.RemoveMember(e.ctx, g.ID, uA, uB)
	mustErr("RemoveMember", err)
	_, err = e.svc.DeleteGroup(e.ctx, g.ID, uA)
	mustErr("DeleteGroup", err)
	_, err = e.svc.UpdateGroup(e.ctx, g.ID, uA, "x", "y")
	mustErr("UpdateGroup", err)
	_, err = e.svc.SetMemberRole(e.ctx, g.ID, uA, uB, models.MemberTalker)
	mustErr("SetMemberRole", err)
	_, err = e.svc.PinConversation(e.ctx, g.ID, uA, true)
	mustErr("PinConversation", err)
	_, err = e.svc.MuteConversation(e.ctx, g.ID, uA, true)
	mustErr("MuteConversation", err)
	_, _, err = e.svc.DeleteMessage(e.ctx, g.ID, msg.ID.Hex(), uA)
	mustErr("DeleteMessage", err)
}

// Le document membre du DEMANDEUR est corrompu : viewFor échoue à lire son
// appartenance (GetMember non-ErrNoDocuments).
func TestSvc_Fault_CorruptRequesterMember(t *testing.T) {
	e := newFaultEnv(t)
	g, _ := e.svc.CreateGroup(e.ctx, uA, "t", "n", env(uA, uB))
	e.corruptMember(t, g.ID, uA)
	if _, err := e.svc.GetConversation(e.ctx, g.ID, uA); err == nil {
		t.Fatalf("GetConversation avec membre demandeur corrompu doit échouer")
	}
}

// Un AUTRE membre est corrompu : viewFor (GetMember du demandeur) réussit, mais
// attachReceipts (ListMembers) échoue au décodage → branche best-effort.
func TestSvc_Fault_CorruptOtherMember(t *testing.T) {
	e := newFaultEnv(t)
	dm, _ := e.svc.CreateDM(e.ctx, uA, uB, env(uA, uB))
	e.corruptMember(t, dm.ID, uB)
	// attachReceipts est best-effort : la vue est renvoyée malgré l'échec interne.
	if _, err := e.svc.GetConversation(e.ctx, dm.ID, uA); err != nil {
		t.Fatalf("GetConversation (autre membre corrompu) doit rester OK : %v", err)
	}
}

// Un message corrompu : EditMessage/DeleteMessage lisent le message (décodage
// échoue) après les contrôles d'appartenance.
func TestSvc_Fault_CorruptMessage(t *testing.T) {
	e := newFaultEnv(t)
	g, _ := e.svc.CreateGroup(e.ctx, uA, "t", "n", env(uA, uB))
	msg, _, _ := e.svc.SendMessage(e.ctx, g.ID, uA, "c", "n", nil)
	// Corrompt le message (created_at en chaîne).
	_, err := e.db.Collection("messages").UpdateOne(e.ctx,
		bson.M{"_id": msg.ID},
		bson.M{"$set": bson.M{"created_at": "pas-une-date"}},
		options.UpdateOne().SetBypassDocumentValidation(true))
	if err != nil {
		t.Fatalf("corrupt message : %v", err)
	}
	if _, _, err := e.svc.EditMessage(e.ctx, g.ID, msg.ID.Hex(), uA, "c2", "n2", nil); err == nil {
		t.Fatalf("EditMessage sur message corrompu doit échouer")
	}
	if _, _, err := e.svc.DeleteMessage(e.ctx, g.ID, msg.ID.Hex(), uA); err == nil {
		t.Fatalf("DeleteMessage sur message corrompu doit échouer")
	}
	// ModerateDeleteMessage lit aussi le message par id.
	if _, _, err := e.svc.ModerateDeleteMessage(e.ctx, msg.ID.Hex()); err == nil {
		t.Fatalf("ModerateDeleteMessage sur message corrompu doit échouer")
	}
}

// Le document membre de la CIBLE est corrompu : AddMember/RemoveMember lisent
// ce membre (GetMember) APRÈS les contrôles d'appartenance et d'autorisation →
// décodage en erreur (distinct de ErrNoDocuments).
func TestSvc_Fault_CorruptTargetMember(t *testing.T) {
	e := newFaultEnv(t)
	g, _ := e.svc.CreateGroup(e.ctx, uA, "t", "n", env(uA, uB))
	e.corruptMember(t, g.ID, uB)

	// AddMember : la cible uB existe (corrompue) → GetMember échoue au décodage.
	if _, err := e.svc.AddMember(e.ctx, g.ID, uA, uB, "env"); err == nil {
		t.Errorf("AddMember (cible corrompue) doit échouer")
	}
	// RemoveMember : exclusion de uB par l'owner → GetMember(uB) échoue au décodage.
	if _, err := e.svc.RemoveMember(e.ctx, g.ID, uA, uB); err == nil {
		t.Errorf("RemoveMember (cible corrompue) doit échouer")
	}
}

// JoinCommunity : le membre déjà présent (corrompu) fait échouer GetMember avec
// une erreur distincte de ErrNoDocuments.
func TestSvc_Fault_CorruptJoinMember(t *testing.T) {
	e := newFaultEnv(t)
	com, _ := e.svc.CreateCommunity(e.ctx, uA, "C", "ck")
	if _, _, err := e.svc.JoinCommunity(e.ctx, com.ID, uB); err != nil {
		t.Fatalf("Join : %v", err)
	}
	e.corruptMember(t, com.ID, uB)
	if _, _, err := e.svc.JoinCommunity(e.ctx, com.ID, uB); err == nil {
		t.Errorf("JoinCommunity (membre corrompu) doit échouer")
	}
}

// --- Gains nominaux résiduels ------------------------------------------------

// UpdateGroup sur un DM → ErrNotManageable (avant tout accès en écriture).
func TestSvc_UpdateGroupOnDM(t *testing.T) {
	svc, _, _, ctx := newSvc(t)
	dm, _ := svc.CreateDM(ctx, uA, uB, env(uA, uB))
	if _, err := svc.UpdateGroup(ctx, dm.ID, uA, "x", "y"); !errors.Is(err, ErrNotManageable) {
		t.Fatalf("UpdateGroup sur DM → %v, attendu ErrNotManageable", err)
	}
}

// EditMessage par un viewer (lecture seule) → ErrCannotWrite.
func TestSvc_EditByViewer(t *testing.T) {
	svc, _, _, ctx := newSvc(t)
	com, _ := svc.CreateCommunity(ctx, uA, "C", "ck")
	msg, _, _ := svc.SendMessage(ctx, com.ID, uA, "c", "n", nil)
	if _, _, err := svc.JoinCommunity(ctx, com.ID, uB); err != nil {
		t.Fatalf("Join : %v", err)
	}
	if _, _, err := svc.EditMessage(ctx, com.ID, msg.ID.Hex(), uB, "c2", "n2", nil); !errors.Is(err, ErrCannotWrite) {
		t.Fatalf("Edit par viewer → %v, attendu ErrCannotWrite", err)
	}
}
