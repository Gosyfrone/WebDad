package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/webdad/report-service/internal/models"
)

// canceledCtx renvoie un contexte DÉJÀ annulé : toute opération Mongo lancée avec
// échoue immédiatement (context.Canceled, jamais mongo.ErrNoDocuments). C'est le
// levier qui exerce les branches `if err != nil { return …, err }` des accès
// données — sinon jamais atteintes avec une base saine.
func canceledCtx() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

// TestRepo_ErrorPaths_CtxAnnule vérifie que chaque méthode du dépôt remonte
// l'erreur driver (et non un succès silencieux) quand l'opération Mongo échoue.
// Le dépôt est adossé à un Mongo réel (collections valides) ; seul le contexte
// est annulé, ce qui isole la branche d'erreur de chaque méthode.
func TestRepo_ErrorPaths_CtxAnnule(t *testing.T) {
	r, _ := repoForTest(t) // skip si MONGO_TEST_URI absent
	cctx := canceledCtx()
	rep := report("u1", models.ReasonSpam)
	id := bson.NewObjectID()

	if _, err := r.UpsertModerationReport(cctx, models.EntityPost, "p1", "o1", rep); err == nil || errors.Is(err, ErrAlreadyReported) {
		t.Errorf("UpsertModerationReport(ctx annulé) → %v, attendu erreur driver", err)
	}
	if _, err := r.InsertBugTicket(cctx, models.EntityApp, "", "", rep); err == nil {
		t.Error("InsertBugTicket(ctx annulé) → nil, attendu erreur")
	}
	if _, err := r.ListTickets(cctx, TicketFilter{Category: models.CategoryModeration}, 10); err == nil {
		t.Error("ListTickets(ctx annulé) → nil, attendu erreur")
	}
	if _, err := r.GetTicket(cctx, id); err == nil || errors.Is(err, mongo.ErrNoDocuments) {
		t.Errorf("GetTicket(ctx annulé) → %v, attendu erreur driver", err)
	}
	if _, err := r.GetModerationByEntity(cctx, models.EntityPost, "p1"); err == nil || errors.Is(err, mongo.ErrNoDocuments) {
		t.Errorf("GetModerationByEntity(ctx annulé) → %v, attendu erreur driver", err)
	}
	if _, err := r.AddAction(cctx, id, models.Action{Type: models.ActionReply, CreatedAt: time.Now()}, nil); err == nil || errors.Is(err, mongo.ErrNoDocuments) {
		t.Errorf("AddAction(ctx annulé) → %v, attendu erreur driver", err)
	}
	if _, err := r.GetSettings(cctx); err == nil {
		t.Error("GetSettings(ctx annulé) → nil, attendu erreur driver")
	}
	if _, err := r.UpdateThreshold(cctx, 3); err == nil {
		t.Error("UpdateThreshold(ctx annulé) → nil, attendu erreur")
	}
	if _, err := r.CreateWarning(cctx, &models.Warning{TargetUserID: "v", Message: "m", IssuedBy: "mod", CreatedAt: time.Now()}); err == nil {
		t.Error("CreateWarning(ctx annulé) → nil, attendu erreur")
	}
	if _, err := r.CountWarnings(cctx, "v"); err == nil {
		t.Error("CountWarnings(ctx annulé) → nil, attendu erreur")
	}
	if _, err := r.PendingWarnings(cctx, "v"); err == nil {
		t.Error("PendingWarnings(ctx annulé) → nil, attendu erreur")
	}
	if err := r.AckWarning(cctx, id, "v"); err == nil || errors.Is(err, mongo.ErrNoDocuments) {
		t.Errorf("AckWarning(ctx annulé) → %v, attendu erreur driver", err)
	}
}
