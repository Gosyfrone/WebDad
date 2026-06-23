package service

import (
	"context"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/webdad/report-service/internal/models"
	"github.com/webdad/report-service/internal/repository"
)

// canceledCtx renvoie un contexte DÉJÀ annulé : les accès Mongo sous-jacents
// échouent (context.Canceled, ≠ mongo.ErrNoDocuments), ce qui exerce les
// branches `return …, err` génériques du service — non atteintes en marche
// nominale ni sur les cas NotFound déjà testés.
func canceledCtx() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

// hexID : un ObjectID valide (format correct) pour franchir le parse et atteindre
// l'accès dépôt, qui échouera sur le contexte annulé.
const hexID = "507f1f77bcf86cd799439011"

// TestSvc_ErrorPaths_CtxAnnule couvre les remontées d'erreur dépôt (≠ NotFound)
// de chaque méthode du service. Service adossé à un Mongo réel ; seul le contexte
// est annulé.
func TestSvc_ErrorPaths_CtxAnnule(t *testing.T) {
	svc, _, _ := svcForTest(t) // skip si MONGO_TEST_URI absent
	cctx := canceledCtx()

	// CreateReport (modération) : UpsertModerationReport échoue → branche `err`.
	if _, err := svc.CreateReport(cctx, "u1", modInput("post1")); err == nil ||
		errors.Is(err, ErrAlreadyReported) || errors.Is(err, ErrReportingLocked) {
		t.Errorf("CreateReport(ctx annulé) → %v, attendu erreur dépôt", err)
	}
	// CreateReport (bug) : InsertBugTicket échoue.
	if _, err := svc.CreateReport(cctx, "u1", CreateReportInput{
		Category: models.CategoryBug, Reason: models.ReasonBug, Text: "x",
	}); err == nil {
		t.Error("CreateReport(bug, ctx annulé) → nil, attendu erreur dépôt")
	}

	// Pour ces méthodes, le format est (ticket, err) ; on n'attend ni succès, ni
	// NotFound (id valide mais accès dépôt en échec), donc une erreur dépôt brute.
	check := func(name string, err error) {
		if err == nil || errors.Is(err, ErrNotFound) || errors.Is(err, ErrInvalidID) {
			t.Errorf("%s(ctx annulé) → %v, attendu erreur dépôt", name, err)
		}
	}
	_, err := svc.Approve(cctx, hexID, "mod")
	check("Approve", err)
	_, err = svc.GetTicket(cctx, hexID)
	check("GetTicket", err)
	_, err = svc.Reply(cctx, hexID, "mod", "texte")
	check("Reply", err)
	_, err = svc.ChangeStatus(cctx, hexID, "mod", models.StatusClosed)
	check("ChangeStatus", err)
	_, err = svc.LogRemoval(cctx, hexID, "mod")
	check("LogRemoval", err)
	_, err = svc.Transfer(cctx, hexID, "admin")
	check("Transfer", err)

	if _, err := svc.ListTickets(cctx, repository.TicketFilter{Category: models.CategoryModeration}, 10); err == nil {
		t.Error("ListTickets(ctx annulé) → nil, attendu erreur dépôt")
	}
	if _, err := svc.UpdateThreshold(cctx, 3); err == nil {
		t.Error("UpdateThreshold(ctx annulé) → nil, attendu erreur dépôt")
	}
	if _, err := svc.Settings(cctx); err == nil {
		t.Error("Settings(ctx annulé) → nil, attendu erreur dépôt")
	}
	if _, err := svc.IssueWarning(cctx, "mod", "victim", "", "message"); err == nil {
		t.Error("IssueWarning(ctx annulé) → nil, attendu erreur dépôt")
	}
	if _, err := svc.WarningCount(cctx, "victim"); err == nil {
		t.Error("WarningCount(ctx annulé) → nil, attendu erreur dépôt")
	}
	if _, err := svc.PendingWarnings(cctx, "victim"); err == nil {
		t.Error("PendingWarnings(ctx annulé) → nil, attendu erreur dépôt")
	}
	if err := svc.AckWarning(cctx, hexID, "victim"); err == nil || errors.Is(err, ErrNotFound) {
		t.Errorf("AckWarning(ctx annulé) → %v, attendu erreur dépôt", err)
	}
}

// TestSvc_MaybeAutoHide_SettingsErreur couvre la branche best-effort de
// maybeAutoHide où GetSettings échoue (ctx annulé) : la fonction s'abstient
// sans paniquer. Le ticket est un POST de modération non terminal pour passer
// les gardes initiales et atteindre l'appel GetSettings.
func TestSvc_MaybeAutoHide_SettingsErreur(t *testing.T) {
	svc, spy, _ := svcForTest(t)
	tk := &models.Ticket{
		ID:         bson.NewObjectID(),
		EntityType: models.EntityPost,
		EntityID:   "post1",
		Status:     models.StatusOpen,
	}
	// GetSettings échoue (ctx annulé) → return best-effort, aucun masquage.
	svc.maybeAutoHide(canceledCtx(), tk)
	if len(spy.hidden) != 0 {
		t.Errorf("maybeAutoHide ne doit pas masquer quand Settings échoue : %v", spy.hidden)
	}
}
