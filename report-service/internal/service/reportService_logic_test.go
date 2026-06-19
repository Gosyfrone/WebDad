package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/webdad/report-service/internal/models"
	"github.com/webdad/report-service/internal/repository"
)

func newReportSvc() *ReportService { return NewReportService(nil, nil) }

// ─── CreateReport — validation pure (avant tout appel repo) ─────────────────

func TestCreateReport_InvalidReason(t *testing.T) {
	svc := newReportSvc()
	_, err := svc.CreateReport(context.TODO(), "u1", CreateReportInput{
		Category: models.CategoryModeration, Reason: "__invalid__",
		EntityType: models.EntityPost, EntityID: "abc123",
	})
	if !errors.Is(err, ErrValidation) {
		t.Errorf("motif invalide → %v, attendu ErrValidation", err)
	}
}

func TestCreateReport_EmptyReason(t *testing.T) {
	svc := newReportSvc()
	_, err := svc.CreateReport(context.TODO(), "u1", CreateReportInput{
		Category: models.CategoryModeration, Reason: "   ",
		EntityType: models.EntityPost, EntityID: "abc123",
	})
	if !errors.Is(err, ErrValidation) {
		t.Errorf("motif vide (espaces) → %v, attendu ErrValidation", err)
	}
}

func TestCreateReport_BugTextTooLong(t *testing.T) {
	svc := newReportSvc()
	_, err := svc.CreateReport(context.TODO(), "u1", CreateReportInput{
		Category: models.CategoryBug,
		Reason:   models.ReasonBug,
		Text:     strings.Repeat("a", models.MaxBugText+1),
	})
	if !errors.Is(err, ErrValidation) {
		t.Errorf("texte bug trop long → %v, attendu ErrValidation", err)
	}
}

func TestCreateReport_ModerationTextTooLong(t *testing.T) {
	svc := newReportSvc()
	_, err := svc.CreateReport(context.TODO(), "u1", CreateReportInput{
		Category: models.CategoryModeration, Reason: models.ReasonSpam,
		EntityType: models.EntityPost, EntityID: "abc123",
		Text: strings.Repeat("a", models.MaxModerationText+1),
	})
	if !errors.Is(err, ErrValidation) {
		t.Errorf("texte modération trop long → %v, attendu ErrValidation", err)
	}
}

func TestCreateReport_ModerationInvalidEntityType(t *testing.T) {
	svc := newReportSvc()
	_, err := svc.CreateReport(context.TODO(), "u1", CreateReportInput{
		Category: models.CategoryModeration, Reason: models.ReasonSpam,
		EntityType: "bad_type", EntityID: "abc123",
	})
	if !errors.Is(err, ErrValidation) {
		t.Errorf("type entité invalide → %v, attendu ErrValidation", err)
	}
}

func TestCreateReport_ModerationEmptyEntityID(t *testing.T) {
	svc := newReportSvc()
	_, err := svc.CreateReport(context.TODO(), "u1", CreateReportInput{
		Category: models.CategoryModeration, Reason: models.ReasonSpam,
		EntityType: models.EntityPost, EntityID: "  ",
	})
	if !errors.Is(err, ErrValidation) {
		t.Errorf("entity_id vide → %v, attendu ErrValidation", err)
	}
}

func TestCreateReport_UnknownCategory(t *testing.T) {
	svc := newReportSvc()
	_, err := svc.CreateReport(context.TODO(), "u1", CreateReportInput{
		Category: "unknown", Reason: models.ReasonSpam,
		EntityType: models.EntityPost, EntityID: "abc123",
	})
	if !errors.Is(err, ErrValidation) {
		t.Errorf("catégorie inconnue → %v, attendu ErrValidation", err)
	}
}

// ─── maybeAutoHide — gardes d'early-return ──────────────────────────────────

func TestMaybeAutoHide_NilTicket(t *testing.T) {
	newReportSvc().maybeAutoHide(context.TODO(), nil)
}

func TestMaybeAutoHide_NonPostEntities(t *testing.T) {
	svc := newReportSvc()
	for _, et := range []string{models.EntityProfile, models.EntityMessage, models.EntityGroupMessage} {
		svc.maybeAutoHide(context.TODO(), &models.Ticket{EntityType: et, EntityID: "x", Status: models.StatusOpen})
	}
}

func TestMaybeAutoHide_EmptyEntityID(t *testing.T) {
	newReportSvc().maybeAutoHide(context.TODO(), &models.Ticket{EntityType: models.EntityPost, EntityID: ""})
}

func TestMaybeAutoHide_ApprovedTicket(t *testing.T) {
	newReportSvc().maybeAutoHide(context.TODO(), &models.Ticket{
		EntityType: models.EntityPost, EntityID: "x", Status: models.StatusApproved,
	})
}

func TestMaybeAutoHide_ClosedTicket(t *testing.T) {
	newReportSvc().maybeAutoHide(context.TODO(), &models.Ticket{
		EntityType: models.EntityPost, EntityID: "x", Status: models.StatusClosed,
	})
}

// ─── hasBlockingSanction ─────────────────────────────────────────────────────

func TestHasBlockingSanction_Empty(t *testing.T) {
	if hasBlockingSanction(nil) {
		t.Error("liste vide → attendu false")
	}
}

func TestHasBlockingSanction_ContentRemoved(t *testing.T) {
	if !hasBlockingSanction([]models.Action{{Type: models.ActionContentRemoved}}) {
		t.Error("content_removed → attendu true")
	}
}

func TestHasBlockingSanction_Warned(t *testing.T) {
	if !hasBlockingSanction([]models.Action{{Type: models.ActionWarned}}) {
		t.Error("warned → attendu true")
	}
}

func TestHasBlockingSanction_OtherActions(t *testing.T) {
	if hasBlockingSanction([]models.Action{{Type: models.ActionReply}, {Type: models.ActionApproved}}) {
		t.Error("reply+approved → attendu false")
	}
}

// ─── Approve ─────────────────────────────────────────────────────────────────

func TestApprove_InvalidID(t *testing.T) {
	_, err := newReportSvc().Approve(context.TODO(), "not-valid-hex", "mod1")
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("Approve(id invalide) → %v, attendu ErrInvalidID", err)
	}
}

func TestApprove_EmptyID(t *testing.T) {
	_, err := newReportSvc().Approve(context.TODO(), "", "mod1")
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("Approve(id vide) → %v, attendu ErrInvalidID", err)
	}
}

// ─── UpdateThreshold ─────────────────────────────────────────────────────────

func TestUpdateThreshold_Negative(t *testing.T) {
	_, err := newReportSvc().UpdateThreshold(context.TODO(), -1)
	if !errors.Is(err, ErrValidation) {
		t.Errorf("seuil négatif → %v, attendu ErrValidation", err)
	}
}

func TestUpdateThreshold_LargeNegative(t *testing.T) {
	_, err := newReportSvc().UpdateThreshold(context.TODO(), -9999)
	if !errors.Is(err, ErrValidation) {
		t.Errorf("seuil -9999 → %v, attendu ErrValidation", err)
	}
}

// ─── ListTickets ─────────────────────────────────────────────────────────────

func TestListTickets_InvalidStatus(t *testing.T) {
	_, err := newReportSvc().ListTickets(context.TODO(), repository.TicketFilter{Status: "unknown_status"}, 10)
	if !errors.Is(err, ErrValidation) {
		t.Errorf("statut inconnu → %v, attendu ErrValidation", err)
	}
}

func TestListTickets_ApprovedStatusInvalid(t *testing.T) {
	_, err := newReportSvc().ListTickets(context.TODO(), repository.TicketFilter{Status: models.StatusApproved}, 10)
	if !errors.Is(err, ErrValidation) {
		t.Errorf("statut 'approved' non filtrable → %v, attendu ErrValidation", err)
	}
}

// ─── GetTicket ───────────────────────────────────────────────────────────────

func TestGetTicket_InvalidID(t *testing.T) {
	_, err := newReportSvc().GetTicket(context.TODO(), "not-valid")
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("GetTicket(id invalide) → %v, attendu ErrInvalidID", err)
	}
}

func TestGetTicket_EmptyID(t *testing.T) {
	_, err := newReportSvc().GetTicket(context.TODO(), "")
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("GetTicket(id vide) → %v, attendu ErrInvalidID", err)
	}
}

// ─── Reply ───────────────────────────────────────────────────────────────────

func TestReply_InvalidID(t *testing.T) {
	_, err := newReportSvc().Reply(context.TODO(), "bad-id", "mod1", "texte valide")
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("Reply(id invalide) → %v, attendu ErrInvalidID", err)
	}
}

func TestReply_EmptyText(t *testing.T) {
	_, err := newReportSvc().Reply(context.TODO(), "507f1f77bcf86cd799439011", "mod1", "  ")
	if !errors.Is(err, ErrValidation) {
		t.Errorf("Reply(texte vide) → %v, attendu ErrValidation", err)
	}
}

func TestReply_TextTooLong(t *testing.T) {
	_, err := newReportSvc().Reply(context.TODO(), "507f1f77bcf86cd799439011", "mod1",
		strings.Repeat("x", models.MaxBugText+1))
	if !errors.Is(err, ErrValidation) {
		t.Errorf("Reply(texte trop long) → %v, attendu ErrValidation", err)
	}
}

// ─── ChangeStatus ────────────────────────────────────────────────────────────

func TestChangeStatus_InvalidID(t *testing.T) {
	_, err := newReportSvc().ChangeStatus(context.TODO(), "bad-id", "mod1", models.StatusClosed)
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("ChangeStatus(id invalide) → %v, attendu ErrInvalidID", err)
	}
}

func TestChangeStatus_InvalidStatus(t *testing.T) {
	_, err := newReportSvc().ChangeStatus(context.TODO(), "507f1f77bcf86cd799439011", "mod1", "unknown")
	if !errors.Is(err, ErrValidation) {
		t.Errorf("ChangeStatus(statut invalide) → %v, attendu ErrValidation", err)
	}
}

// ─── LogRemoval ──────────────────────────────────────────────────────────────

func TestLogRemoval_InvalidID(t *testing.T) {
	_, err := newReportSvc().LogRemoval(context.TODO(), "bad-id", "mod1")
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("LogRemoval(id invalide) → %v, attendu ErrInvalidID", err)
	}
}

// ─── Transfer ────────────────────────────────────────────────────────────────

func TestTransfer_InvalidID(t *testing.T) {
	_, err := newReportSvc().Transfer(context.TODO(), "bad-id", "admin1")
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("Transfer(id invalide) → %v, attendu ErrInvalidID", err)
	}
}

// ─── IssueWarning ────────────────────────────────────────────────────────────

func TestIssueWarning_EmptyTargetUserID(t *testing.T) {
	_, err := newReportSvc().IssueWarning(context.TODO(), "mod1", "  ", "", "message valide")
	if !errors.Is(err, ErrValidation) {
		t.Errorf("IssueWarning(target vide) → %v, attendu ErrValidation", err)
	}
}

func TestIssueWarning_EmptyMessage(t *testing.T) {
	_, err := newReportSvc().IssueWarning(context.TODO(), "mod1", "user1", "", "  ")
	if !errors.Is(err, ErrValidation) {
		t.Errorf("IssueWarning(message vide) → %v, attendu ErrValidation", err)
	}
}

func TestIssueWarning_MessageTooLong(t *testing.T) {
	_, err := newReportSvc().IssueWarning(context.TODO(), "mod1", "user1", "",
		strings.Repeat("x", models.MaxWarningMessage+1))
	if !errors.Is(err, ErrValidation) {
		t.Errorf("IssueWarning(message trop long) → %v, attendu ErrValidation", err)
	}
}

// ─── WarningCount ────────────────────────────────────────────────────────────

func TestWarningCount_EmptyUserID(t *testing.T) {
	_, err := newReportSvc().WarningCount(context.TODO(), "  ")
	if !errors.Is(err, ErrValidation) {
		t.Errorf("WarningCount(user vide) → %v, attendu ErrValidation", err)
	}
}

// ─── AckWarning ──────────────────────────────────────────────────────────────

func TestAckWarning_InvalidID(t *testing.T) {
	err := newReportSvc().AckWarning(context.TODO(), "bad-id", "user1")
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("AckWarning(id invalide) → %v, attendu ErrInvalidID", err)
	}
}

func TestAckWarning_EmptyID(t *testing.T) {
	err := newReportSvc().AckWarning(context.TODO(), "", "user1")
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("AckWarning(id vide) → %v, attendu ErrInvalidID", err)
	}
}
