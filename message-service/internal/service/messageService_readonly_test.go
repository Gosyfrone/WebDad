package service

import (
	"context"
	"testing"

	"github.com/webdad/message-service/internal/database"
	"github.com/webdad/message-service/internal/models"
	"github.com/webdad/message-service/internal/mongotest"
	"github.com/webdad/message-service/internal/repository"
)

// Les écritures du service via un client Mongo EN LECTURE SEULE échouent APRÈS
// que les lectures (contrôles d'appartenance/rôle) ont réussi → on traverse les
// branches `if err != nil { return ..., err }` posées sur l'appel repo en
// écriture, qui suivent un ou plusieurs appels en lecture réussis.
func TestSvc_ReadOnly_WriteErrorPaths(t *testing.T) {
	rootDB, roDB := mongotest.ReadOnlyDB(t)
	ctx := context.Background()
	if err := database.EnsureSchema(ctx, rootDB); err != nil {
		t.Fatalf("EnsureSchema : %v", err)
	}

	// Peuplement via le client root (lecture/écriture).
	root := NewMessageService(repository.NewMessageRepository(rootDB))
	grp, err := root.CreateGroup(ctx, uA, "t", "n", env(uA, uB))
	if err != nil {
		t.Fatalf("seed group : %v", err)
	}
	com, err := root.CreateCommunity(ctx, uA, "C", "ck")
	if err != nil {
		t.Fatalf("seed community : %v", err)
	}
	if _, _, err := root.JoinCommunity(ctx, com.ID, uB); err != nil {
		t.Fatalf("seed join : %v", err)
	}
	msg, _, err := root.SendMessage(ctx, grp.ID, uA, "c", "n", nil)
	if err != nil {
		t.Fatalf("seed message : %v", err)
	}

	// Service branché sur le client LECTURE SEULE.
	ro := NewMessageService(repository.NewMessageRepository(roDB))

	mustErr := func(name string, err error) {
		if err == nil {
			t.Errorf("%s : erreur d'écriture attendue (client lecture seule)", name)
		}
	}

	// Écritures sans contrôle de lecture préalable.
	mustErr("PublishKey", ro.PublishKey(ctx, uA, "k"))
	mustErr("PurgeUser", ro.PurgeUser(ctx, uA))
	mustErr("PutBackup", ro.PutBackup(ctx, uA, models.PutBackupRequest{}))
	_, err = ro.CreateCommunity(ctx, uA, "x", "ck")
	mustErr("CreateCommunity", err)
	_, err = ro.CreateGroup(ctx, uA, "t", "n", env(uA, uB))
	mustErr("CreateGroup", err)
	_, err = ro.CreateDM(ctx, "p1", "p2", map[string]string{"p1": "e", "p2": "e"})
	mustErr("CreateDM", err)

	// Écritures précédées de lectures réussies (appartenance/rôle OK).
	_, _, err = ro.JoinCommunity(ctx, com.ID, "newcomer")
	mustErr("JoinCommunity.AddMember", err)
	_, err = ro.AddMember(ctx, grp.ID, uA, "newcomer", "env")
	mustErr("AddMember.Insert", err)
	_, err = ro.RemoveMember(ctx, grp.ID, uA, uB)
	mustErr("RemoveMember", err)
	_, err = ro.SetMemberRole(ctx, com.ID, uA, uB, models.MemberTalker)
	mustErr("SetMemberRole", err)
	_, err = ro.UpdateGroup(ctx, grp.ID, uA, "x", "y")
	mustErr("UpdateGroup", err)
	_, err = ro.DeleteGroup(ctx, grp.ID, uA)
	mustErr("DeleteGroup", err)
	_, err = ro.PinConversation(ctx, grp.ID, uA, true)
	mustErr("PinConversation", err)
	mustErr("ClearConversation", ro.ClearConversation(ctx, grp.ID, uA))
	_, err = ro.MuteConversation(ctx, grp.ID, uA, true)
	mustErr("MuteConversation", err)
	_, _, err = ro.MarkRead(ctx, grp.ID, uA)
	mustErr("MarkRead", err)
	_, _, err = ro.TouchDelivered(ctx, grp.ID, uA)
	mustErr("TouchDelivered", err)
	_, _, err = ro.SendMessage(ctx, grp.ID, uA, "c", "n", nil)
	mustErr("SendMessage.Insert", err)
	_, _, err = ro.EditMessage(ctx, grp.ID, msg.ID.Hex(), uA, "c2", "n2", nil)
	mustErr("EditMessage.Update", err)
	_, _, err = ro.DeleteMessage(ctx, grp.ID, msg.ID.Hex(), uA)
	mustErr("DeleteMessage.SoftDelete", err)
	_, _, err = ro.ModerateDeleteMessage(ctx, msg.ID.Hex())
	mustErr("ModerateDeleteMessage", err)

	// MarkDelivered est best-effort (pas de retour) : ne doit pas paniquer.
	ro.MarkDelivered(ctx, grp.ID, []string{uA}, msg.CreatedAt)
}
