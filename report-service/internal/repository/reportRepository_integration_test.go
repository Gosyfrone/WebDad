package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/webdad/report-service/internal/models"
	"github.com/webdad/report-service/internal/testutil"
)

// repoForTest construit un dépôt sur une base Mongo isolée (ou skip si aucune).
func repoForTest(t *testing.T) (*ReportRepository, context.Context) {
	t.Helper()
	db := testutil.MongoDB(t)
	return NewReportRepository(db), context.Background()
}

func report(reporter, reason string) models.Report {
	return models.Report{ReporterID: reporter, Reason: reason, Text: "txt", CreatedAt: time.Now()}
}

// ─── UpsertModerationReport ─────────────────────────────────────────────────

func TestUpsert_CreeEtEmpile(t *testing.T) {
	r, ctx := repoForTest(t)

	// 1er signalement → crée le ticket parent.
	tk, err := r.UpsertModerationReport(ctx, models.EntityPost, "post1", "owner1", report("u1", models.ReasonSpam))
	if err != nil {
		t.Fatalf("1er upsert : %v", err)
	}
	if tk.ReportCount != 1 || tk.Status != models.StatusOpen || tk.Category != models.CategoryModeration {
		t.Fatalf("ticket initial inattendu : %+v", tk)
	}
	if tk.ReasonTags[models.ReasonSpam] != 1 {
		t.Errorf("reason_tags spam = %d, attendu 1", tk.ReasonTags[models.ReasonSpam])
	}

	// 2e rapporteur, même entité → empile dans le même ticket.
	tk2, err := r.UpsertModerationReport(ctx, models.EntityPost, "post1", "owner1", report("u2", models.ReasonOffensive))
	if err != nil {
		t.Fatalf("2e upsert : %v", err)
	}
	if tk2.ID != tk.ID {
		t.Errorf("le 2e signalement doit cibler le même ticket (%s != %s)", tk2.ID.Hex(), tk.ID.Hex())
	}
	if tk2.ReportCount != 2 {
		t.Errorf("report_count = %d, attendu 2", tk2.ReportCount)
	}
}

func TestUpsert_MemeRapporteur_DejaSignale(t *testing.T) {
	r, ctx := repoForTest(t)
	if _, err := r.UpsertModerationReport(ctx, models.EntityPost, "post1", "o", report("u1", models.ReasonSpam)); err != nil {
		t.Fatalf("1er upsert : %v", err)
	}
	_, err := r.UpsertModerationReport(ctx, models.EntityPost, "post1", "o", report("u1", models.ReasonSpam))
	if !errors.Is(err, ErrAlreadyReported) {
		t.Fatalf("re-signalement par le même → %v, attendu ErrAlreadyReported", err)
	}
}

// ─── InsertBugTicket ─────────────────────────────────────────────────────────

func TestInsertBug_AvecEntite(t *testing.T) {
	r, ctx := repoForTest(t)
	tk, err := r.InsertBugTicket(ctx, models.EntityPost, "post9", "owner9", report("u1", models.ReasonBug))
	if err != nil {
		t.Fatalf("insert bug : %v", err)
	}
	if tk.ID.IsZero() || tk.Category != models.CategoryBug || tk.EntityType != models.EntityPost {
		t.Fatalf("ticket bug inattendu : %+v", tk)
	}
}

func TestInsertBug_SansEntite_App(t *testing.T) {
	r, ctx := repoForTest(t)
	tk, err := r.InsertBugTicket(ctx, "", "", "", report("u1", models.ReasonBug))
	if err != nil {
		t.Fatalf("insert bug app : %v", err)
	}
	if tk.EntityType != models.EntityApp {
		t.Errorf("entity_type = %q, attendu %q", tk.EntityType, models.EntityApp)
	}
}

// ─── ListTickets ─────────────────────────────────────────────────────────────

func TestListTickets_FiltresEtTri(t *testing.T) {
	r, ctx := repoForTest(t)
	// Deux tickets de modération de volumes différents + un clôturé.
	a, _ := r.UpsertModerationReport(ctx, models.EntityPost, "pa", "o", report("u1", models.ReasonSpam))
	_, _ = r.UpsertModerationReport(ctx, models.EntityPost, "pa", "o", report("u2", models.ReasonSpam)) // pa = 2 signalements
	_, _ = r.UpsertModerationReport(ctx, models.EntityPost, "pb", "o", report("u1", models.ReasonSpam)) // pb = 1
	closed, _ := r.UpsertModerationReport(ctx, models.EntityPost, "pc", "o", report("u1", models.ReasonSpam))
	if _, err := r.AddAction(ctx, closed.ID, models.Action{Type: models.ActionStatusChange, CreatedAt: time.Now()}, bson.M{"status": models.StatusClosed}); err != nil {
		t.Fatalf("clôture : %v", err)
	}
	_ = a

	out, err := r.ListTickets(ctx, TicketFilter{Category: models.CategoryModeration}, 50)
	if err != nil {
		t.Fatalf("list : %v", err)
	}
	if len(out) != 3 {
		t.Fatalf("len = %d, attendu 3", len(out))
	}
	// Actifs avant clôturés ; à volume égal, le plus signalé d'abord.
	if out[0].EntityID != "pa" {
		t.Errorf("1er ticket = %q, attendu 'pa' (actif, 2 signalements)", out[0].EntityID)
	}
	if out[len(out)-1].Status != models.StatusClosed {
		t.Errorf("dernier ticket statut = %q, attendu 'closed'", out[len(out)-1].Status)
	}

	// Filtre min_reports = 2 → seul 'pa'.
	out2, err := r.ListTickets(ctx, TicketFilter{Category: models.CategoryModeration, MinReports: 2}, 50)
	if err != nil {
		t.Fatalf("list min : %v", err)
	}
	if len(out2) != 1 || out2[0].EntityID != "pa" {
		t.Errorf("min_reports=2 → %d tickets, attendu 1 (pa)", len(out2))
	}

	// Filtre statut + plage temporelle.
	since := time.Now().Add(-time.Hour)
	until := time.Now().Add(time.Hour)
	out3, err := r.ListTickets(ctx, TicketFilter{Category: models.CategoryModeration, Status: models.StatusClosed, Since: &since, Until: &until}, 50)
	if err != nil {
		t.Fatalf("list statut+plage : %v", err)
	}
	if len(out3) != 1 || out3[0].EntityID != "pc" {
		t.Errorf("statut closed → %d tickets, attendu 1 (pc)", len(out3))
	}
}

// ─── GetTicket / GetModerationByEntity ──────────────────────────────────────

func TestGetTicket_TrouveEtAbsent(t *testing.T) {
	r, ctx := repoForTest(t)
	tk, _ := r.InsertBugTicket(ctx, "", "", "", report("u1", models.ReasonBug))

	got, err := r.GetTicket(ctx, tk.ID)
	if err != nil || got.ID != tk.ID {
		t.Fatalf("GetTicket : got=%v err=%v", got, err)
	}

	_, err = r.GetTicket(ctx, bson.NewObjectID())
	if !errors.Is(err, mongo.ErrNoDocuments) {
		t.Errorf("ticket absent → %v, attendu ErrNoDocuments", err)
	}
}

func TestGetModerationByEntity(t *testing.T) {
	r, ctx := repoForTest(t)
	if _, err := r.UpsertModerationReport(ctx, models.EntityProfile, "prof1", "o", report("u1", models.ReasonOther)); err != nil {
		t.Fatalf("upsert : %v", err)
	}
	got, err := r.GetModerationByEntity(ctx, models.EntityProfile, "prof1")
	if err != nil || got.EntityID != "prof1" {
		t.Fatalf("GetModerationByEntity : got=%v err=%v", got, err)
	}
	if _, err := r.GetModerationByEntity(ctx, models.EntityProfile, "inconnu"); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Errorf("entité absente → %v, attendu ErrNoDocuments", err)
	}
}

// ─── AddAction / AutoReopen ──────────────────────────────────────────────────

func TestAddAction_EtAbsent(t *testing.T) {
	r, ctx := repoForTest(t)
	tk, _ := r.InsertBugTicket(ctx, "", "", "", report("u1", models.ReasonBug))

	updated, err := r.AddAction(ctx, tk.ID, models.Action{ModeratorID: "m1", Type: models.ActionReply, Text: "ok", CreatedAt: time.Now()}, bson.M{"status": models.StatusClosed})
	if err != nil {
		t.Fatalf("AddAction : %v", err)
	}
	if len(updated.Actions) != 1 || updated.Status != models.StatusClosed {
		t.Fatalf("action non appliquée : %+v", updated)
	}

	if _, err := r.AddAction(ctx, bson.NewObjectID(), models.Action{Type: models.ActionReply, CreatedAt: time.Now()}, nil); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Errorf("AddAction(absent) → %v, attendu ErrNoDocuments", err)
	}
}

func TestAutoReopen(t *testing.T) {
	r, ctx := repoForTest(t)
	tk, _ := r.UpsertModerationReport(ctx, models.EntityPost, "px", "o", report("u1", models.ReasonSpam))
	reop, err := r.AutoReopen(ctx, tk.ID, time.Now())
	if err != nil {
		t.Fatalf("AutoReopen : %v", err)
	}
	if reop.Status != models.StatusReopened || reop.ReportsSinceClosed != 0 {
		t.Fatalf("réouverture inattendue : %+v", reop)
	}
}

// ─── Settings ────────────────────────────────────────────────────────────────

func TestSettings_SeedEtUpdate(t *testing.T) {
	r, ctx := repoForTest(t)
	// EnsureSchema a seedé le singleton avec le défaut.
	s, err := r.GetSettings(ctx)
	if err != nil || s.AutoHideThreshold != models.DefaultAutoHideThreshold {
		t.Fatalf("settings seed : s=%v err=%v", s, err)
	}

	upd, err := r.UpdateThreshold(ctx, 12)
	if err != nil || upd.AutoHideThreshold != 12 {
		t.Fatalf("UpdateThreshold : upd=%v err=%v", upd, err)
	}
	again, _ := r.GetSettings(ctx)
	if again.AutoHideThreshold != 12 {
		t.Errorf("seuil persistant = %d, attendu 12", again.AutoHideThreshold)
	}
}

func TestSettings_FallbackDefautSiAbsent(t *testing.T) {
	db := testutil.MongoDB(t)
	r := NewReportRepository(db)
	ctx := context.Background()
	// Supprime le singleton seedé → GetSettings doit retomber sur le défaut.
	if _, err := db.Collection("settings").DeleteMany(ctx, bson.M{}); err != nil {
		t.Fatalf("delete settings : %v", err)
	}
	s, err := r.GetSettings(ctx)
	if err != nil {
		t.Fatalf("GetSettings : %v", err)
	}
	if s.AutoHideThreshold != models.DefaultAutoHideThreshold {
		t.Errorf("fallback = %d, attendu %d", s.AutoHideThreshold, models.DefaultAutoHideThreshold)
	}
}

// ─── Warnings ────────────────────────────────────────────────────────────────

func TestWarnings_CycleComplet(t *testing.T) {
	r, ctx := repoForTest(t)
	w1, err := r.CreateWarning(ctx, &models.Warning{TargetUserID: "victim", Message: "stop", IssuedBy: "mod", CreatedAt: time.Now()})
	if err != nil || w1.ID.IsZero() {
		t.Fatalf("CreateWarning : w=%v err=%v", w1, err)
	}
	_, _ = r.CreateWarning(ctx, &models.Warning{TargetUserID: "victim", Message: "second", IssuedBy: "mod", CreatedAt: time.Now()})

	n, err := r.CountWarnings(ctx, "victim")
	if err != nil || n != 2 {
		t.Fatalf("CountWarnings = %d (err %v), attendu 2", n, err)
	}

	pending, err := r.PendingWarnings(ctx, "victim")
	if err != nil || len(pending) != 2 {
		t.Fatalf("PendingWarnings = %d (err %v), attendu 2", len(pending), err)
	}

	// Acquittement par la cible.
	if err := r.AckWarning(ctx, w1.ID, "victim"); err != nil {
		t.Fatalf("AckWarning : %v", err)
	}
	pending2, _ := r.PendingWarnings(ctx, "victim")
	if len(pending2) != 1 {
		t.Errorf("après ack : %d en attente, attendu 1", len(pending2))
	}

	// Acquittement par un autre utilisateur → pas de match.
	if err := r.AckWarning(ctx, w1.ID, "intrus"); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Errorf("ack par un tiers → %v, attendu ErrNoDocuments", err)
	}
}
