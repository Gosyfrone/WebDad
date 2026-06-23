package service

import (
	"context"
	"errors"
	"testing"

	"github.com/webdad/report-service/internal/models"
	"github.com/webdad/report-service/internal/repository"
	"github.com/webdad/report-service/internal/testutil"
)

// spyPosts enregistre les appels d'auto-modération pour les assertions.
type spyPosts struct {
	hidden   []string
	unhidden []string
}

func (s *spyPosts) AutoHide(id string)   { s.hidden = append(s.hidden, id) }
func (s *spyPosts) AutoUnhide(id string) { s.unhidden = append(s.unhidden, id) }

// svcForTest construit un service sur un dépôt Mongo réel + un espion de posts.
func svcForTest(t *testing.T) (*ReportService, *spyPosts, context.Context) {
	t.Helper()
	db := testutil.MongoDB(t)
	spy := &spyPosts{}
	svc := NewReportService(repository.NewReportRepository(db), spy)
	return svc, spy, context.Background()
}

func modInput(entityID string) CreateReportInput {
	return CreateReportInput{
		Category: models.CategoryModeration, Reason: models.ReasonSpam,
		EntityType: models.EntityPost, EntityID: entityID, EntityOwnerID: "owner",
	}
}

// ─── CreateReport ────────────────────────────────────────────────────────────

func TestSvc_CreateReport_Bug(t *testing.T) {
	svc, _, ctx := svcForTest(t)
	tk, err := svc.CreateReport(ctx, "u1", CreateReportInput{
		Category: models.CategoryBug, Reason: models.ReasonBug, Text: "ça plante",
		EntityType: models.EntityPost, EntityID: "p1",
	})
	if err != nil {
		t.Fatalf("CreateReport bug : %v", err)
	}
	if tk.Category != models.CategoryBug {
		t.Errorf("catégorie = %q, attendu bug", tk.Category)
	}
}

func TestSvc_CreateReport_BugSansEntite(t *testing.T) {
	svc, _, ctx := svcForTest(t)
	tk, err := svc.CreateReport(ctx, "u1", CreateReportInput{
		Category: models.CategoryBug, Reason: models.ReasonBug, Text: "bug global",
		EntityType: "bad_type", // ignoré → app
	})
	if err != nil {
		t.Fatalf("CreateReport bug : %v", err)
	}
	if tk.EntityType != models.EntityApp {
		t.Errorf("entity_type = %q, attendu app", tk.EntityType)
	}
}

func TestSvc_CreateReport_Moderation(t *testing.T) {
	svc, _, ctx := svcForTest(t)
	tk, err := svc.CreateReport(ctx, "u1", modInput("post1"))
	if err != nil {
		t.Fatalf("CreateReport moderation : %v", err)
	}
	if tk.ReportCount != 1 {
		t.Errorf("report_count = %d, attendu 1", tk.ReportCount)
	}
}

func TestSvc_CreateReport_Duplicate(t *testing.T) {
	svc, _, ctx := svcForTest(t)
	if _, err := svc.CreateReport(ctx, "u1", modInput("post1")); err != nil {
		t.Fatalf("1er : %v", err)
	}
	_, err := svc.CreateReport(ctx, "u1", modInput("post1"))
	if !errors.Is(err, ErrAlreadyReported) {
		t.Errorf("doublon → %v, attendu ErrAlreadyReported", err)
	}
}

func TestSvc_CreateReport_Locked(t *testing.T) {
	svc, _, ctx := svcForTest(t)
	tk, _ := svc.CreateReport(ctx, "u1", modInput("post1"))
	// Approbation → verrou.
	if _, err := svc.Approve(ctx, tk.ID.Hex(), "mod"); err != nil {
		t.Fatalf("Approve : %v", err)
	}
	_, err := svc.CreateReport(ctx, "u2", modInput("post1"))
	if !errors.Is(err, ErrReportingLocked) {
		t.Errorf("entité approuvée → %v, attendu ErrReportingLocked", err)
	}
}

func TestSvc_CreateReport_AutoReopen(t *testing.T) {
	svc, _, ctx := svcForTest(t)
	tk, _ := svc.CreateReport(ctx, "u1", modInput("post1"))
	// Clôture manuelle (remet reports_since_closed à 0).
	if _, err := svc.ChangeStatus(ctx, tk.ID.Hex(), "mod", models.StatusClosed); err != nil {
		t.Fatalf("ChangeStatus : %v", err)
	}
	// Deux nouveaux signalements depuis la clôture → atteint ReopenThreshold (2).
	if _, err := svc.CreateReport(ctx, "u2", modInput("post1")); err != nil {
		t.Fatalf("re-signalement u2 : %v", err)
	}
	tk3, err := svc.CreateReport(ctx, "u3", modInput("post1"))
	if err != nil {
		t.Fatalf("re-signalement u3 : %v", err)
	}
	if tk3.Status != models.StatusReopened {
		t.Errorf("statut = %q, attendu reopened (seuil de réouverture atteint)", tk3.Status)
	}
}

func TestSvc_CreateReport_AutoHide(t *testing.T) {
	svc, spy, ctx := svcForTest(t)
	// Seuil bas pour déclencher l'auto-masquage au 1er signalement.
	if _, err := svc.UpdateThreshold(ctx, 1); err != nil {
		t.Fatalf("UpdateThreshold : %v", err)
	}
	tk, err := svc.CreateReport(ctx, "u1", modInput("post1"))
	if err != nil {
		t.Fatalf("CreateReport : %v", err)
	}
	if len(spy.hidden) != 1 || spy.hidden[0] != "post1" {
		t.Fatalf("AutoHide non appelé : %v", spy.hidden)
	}
	// Action auto_hidden journalisée (idempotence).
	var hidden bool
	for _, a := range tk.Actions {
		if a.Type == models.ActionAutoHidden {
			hidden = true
		}
	}
	if !hidden {
		t.Error("action auto_hidden non journalisée sur le ticket")
	}
	// 2e signalement → idempotent : pas de second AutoHide.
	if _, err := svc.CreateReport(ctx, "u2", modInput("post1")); err != nil {
		t.Fatalf("2e signalement : %v", err)
	}
	if len(spy.hidden) != 1 {
		t.Errorf("AutoHide appelé %d fois, attendu 1 (idempotent)", len(spy.hidden))
	}
}

// ─── Approve ─────────────────────────────────────────────────────────────────

func TestSvc_Approve_Demasque(t *testing.T) {
	svc, spy, ctx := svcForTest(t)
	tk, _ := svc.CreateReport(ctx, "u1", modInput("post1"))
	got, err := svc.Approve(ctx, tk.ID.Hex(), "mod")
	if err != nil {
		t.Fatalf("Approve : %v", err)
	}
	if got.Status != models.StatusApproved {
		t.Errorf("statut = %q, attendu approved", got.Status)
	}
	if len(spy.unhidden) != 1 || spy.unhidden[0] != "post1" {
		t.Errorf("AutoUnhide non appelé : %v", spy.unhidden)
	}
}

func TestSvc_Approve_NotFound(t *testing.T) {
	svc, _, ctx := svcForTest(t)
	_, err := svc.Approve(ctx, "507f1f77bcf86cd799439011", "mod")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Approve(absent) → %v, attendu ErrNotFound", err)
	}
}

func TestSvc_Approve_BlockingSanction(t *testing.T) {
	svc, _, ctx := svcForTest(t)
	tk, _ := svc.CreateReport(ctx, "u1", modInput("post1"))
	// Retrait du contenu → sanction terminale, valider devient impossible.
	if _, err := svc.LogRemoval(ctx, tk.ID.Hex(), "mod"); err != nil {
		t.Fatalf("LogRemoval : %v", err)
	}
	_, err := svc.Approve(ctx, tk.ID.Hex(), "mod")
	if !errors.Is(err, ErrAlreadyActioned) {
		t.Errorf("Approve après sanction → %v, attendu ErrAlreadyActioned", err)
	}
}

// ─── Lecture / mutations simples ─────────────────────────────────────────────

func TestSvc_ListGetSettings(t *testing.T) {
	svc, _, ctx := svcForTest(t)
	if _, err := svc.CreateReport(ctx, "u1", modInput("post1")); err != nil {
		t.Fatalf("seed : %v", err)
	}
	list, err := svc.ListTickets(ctx, repository.TicketFilter{Category: models.CategoryModeration}, 10)
	if err != nil || len(list) != 1 {
		t.Fatalf("ListTickets = %d (err %v), attendu 1", len(list), err)
	}
	got, err := svc.GetTicket(ctx, list[0].ID.Hex())
	if err != nil || got.ID != list[0].ID {
		t.Fatalf("GetTicket : got=%v err=%v", got, err)
	}
	s, err := svc.Settings(ctx)
	if err != nil || s.AutoHideThreshold != models.DefaultAutoHideThreshold {
		t.Fatalf("Settings : s=%v err=%v", s, err)
	}
}

// TestSvc_ListTickets_LimiteParDefaut couvre le bornage de la limite (≤0 et
// > MaxLimit retombent sur DefaultLimit).
func TestSvc_ListTickets_LimiteParDefaut(t *testing.T) {
	svc, _, ctx := svcForTest(t)
	if _, err := svc.CreateReport(ctx, "u1", modInput("post1")); err != nil {
		t.Fatalf("seed : %v", err)
	}
	for _, lim := range []int64{0, -5, MaxLimit + 1} {
		if _, err := svc.ListTickets(ctx, repository.TicketFilter{Category: models.CategoryModeration}, lim); err != nil {
			t.Errorf("ListTickets(limit=%d) : %v", lim, err)
		}
	}
}

func TestSvc_GetTicket_NotFound(t *testing.T) {
	svc, _, ctx := svcForTest(t)
	if _, err := svc.GetTicket(ctx, "507f1f77bcf86cd799439011"); !errors.Is(err, ErrNotFound) {
		t.Errorf("GetTicket(absent) → %v, attendu ErrNotFound", err)
	}
}

func TestSvc_ReplyChangeStatusRemoval(t *testing.T) {
	svc, _, ctx := svcForTest(t)
	tk, _ := svc.CreateReport(ctx, "u1", modInput("post1"))
	id := tk.ID.Hex()

	if _, err := svc.Reply(ctx, id, "mod", "réponse interne"); err != nil {
		t.Fatalf("Reply : %v", err)
	}
	if _, err := svc.ChangeStatus(ctx, id, "mod", models.StatusClosed); err != nil {
		t.Fatalf("ChangeStatus : %v", err)
	}
	if _, err := svc.LogRemoval(ctx, id, "mod"); err != nil {
		t.Fatalf("LogRemoval : %v", err)
	}

	// Branches NotFound.
	absent := "507f1f77bcf86cd799439011"
	if _, err := svc.Reply(ctx, absent, "mod", "x"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Reply(absent) → %v, attendu ErrNotFound", err)
	}
	if _, err := svc.ChangeStatus(ctx, absent, "mod", models.StatusOpen); !errors.Is(err, ErrNotFound) {
		t.Errorf("ChangeStatus(absent) → %v, attendu ErrNotFound", err)
	}
	if _, err := svc.LogRemoval(ctx, absent, "mod"); !errors.Is(err, ErrNotFound) {
		t.Errorf("LogRemoval(absent) → %v, attendu ErrNotFound", err)
	}
}

// ─── Transfer ────────────────────────────────────────────────────────────────

func TestSvc_Transfer(t *testing.T) {
	svc, _, ctx := svcForTest(t)
	bug, _ := svc.CreateReport(ctx, "u1", CreateReportInput{
		Category: models.CategoryBug, Reason: models.ReasonBug, Text: "bug",
	})
	got, err := svc.Transfer(ctx, bug.ID.Hex(), "admin")
	if err != nil {
		t.Fatalf("Transfer : %v", err)
	}
	if got.Category != models.CategoryModeration {
		t.Errorf("catégorie après transfert = %q, attendu moderation", got.Category)
	}

	// Un ticket de modération ne peut pas être transféré.
	mod, _ := svc.CreateReport(ctx, "u2", modInput("post1"))
	if _, err := svc.Transfer(ctx, mod.ID.Hex(), "admin"); !errors.Is(err, ErrNotBug) {
		t.Errorf("Transfer(modération) → %v, attendu ErrNotBug", err)
	}

	if _, err := svc.Transfer(ctx, "507f1f77bcf86cd799439011", "admin"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Transfer(absent) → %v, attendu ErrNotFound", err)
	}
}

// ─── Warnings ────────────────────────────────────────────────────────────────

func TestSvc_Warnings(t *testing.T) {
	svc, _, ctx := svcForTest(t)
	tk, _ := svc.CreateReport(ctx, "u1", modInput("post1"))

	// Émission depuis un ticket → journalise l'action warned.
	w, err := svc.IssueWarning(ctx, "mod", "victim", tk.ID.Hex(), "comportement inapproprié")
	if err != nil || w.ID.IsZero() {
		t.Fatalf("IssueWarning : w=%v err=%v", w, err)
	}
	got, _ := svc.GetTicket(ctx, tk.ID.Hex())
	var warned bool
	for _, a := range got.Actions {
		if a.Type == models.ActionWarned {
			warned = true
		}
	}
	if !warned {
		t.Error("action warned non journalisée sur le ticket")
	}

	// Émission sans ticket.
	if _, err := svc.IssueWarning(ctx, "mod", "victim", "", "second avertissement"); err != nil {
		t.Fatalf("IssueWarning sans ticket : %v", err)
	}

	n, err := svc.WarningCount(ctx, "victim")
	if err != nil || n != 2 {
		t.Fatalf("WarningCount = %d (err %v), attendu 2", n, err)
	}

	pending, err := svc.PendingWarnings(ctx, "victim")
	if err != nil || len(pending) != 2 {
		t.Fatalf("PendingWarnings = %d (err %v), attendu 2", len(pending), err)
	}

	if err := svc.AckWarning(ctx, pending[0].ID.Hex(), "victim"); err != nil {
		t.Fatalf("AckWarning : %v", err)
	}
	if err := svc.AckWarning(ctx, "507f1f77bcf86cd799439011", "victim"); !errors.Is(err, ErrNotFound) {
		t.Errorf("AckWarning(absent) → %v, attendu ErrNotFound", err)
	}
}
