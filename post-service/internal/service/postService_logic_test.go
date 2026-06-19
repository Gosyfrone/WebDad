package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/webdad/post-service/internal/models"
	"github.com/webdad/post-service/internal/repository"
)

func newPostSvc() *PostService { return NewPostService(nil) }

// ─── CreatePost — replyAudience invalide (avant tout appel repo) ─────────────

func TestCreatePost_InvalidReplyAudience(t *testing.T) {
	svc := newPostSvc()
	_, err := svc.CreatePost(context.TODO(), "u1", "contenu", "", nil, nil, "invalid")
	if !errors.Is(err, ErrInvalidReplyAudience) {
		t.Errorf("replyAudience invalide → %v, attendu ErrInvalidReplyAudience", err)
	}
}

func TestCreatePost_InvalidReplyAudience_Random(t *testing.T) {
	svc := newPostSvc()
	_, err := svc.CreatePost(context.TODO(), "u1", "contenu", "", nil, nil, "moderators_only")
	if !errors.Is(err, ErrInvalidReplyAudience) {
		t.Errorf("replyAudience 'moderators_only' → %v, attendu ErrInvalidReplyAudience", err)
	}
}

// ─── GetPost — id invalide ───────────────────────────────────────────────────

func TestGetPost_InvalidID(t *testing.T) {
	svc := newPostSvc()
	_, err := svc.GetPost(context.TODO(), "not-a-hex", "u1", "")
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("GetPost(id invalide) → %v, attendu ErrInvalidID", err)
	}
}

func TestGetPost_EmptyID(t *testing.T) {
	svc := newPostSvc()
	_, err := svc.GetPost(context.TODO(), "", "u1", "")
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("GetPost(id vide) → %v, attendu ErrInvalidID", err)
	}
}

// ─── UpdatePost — id invalide ────────────────────────────────────────────────

func TestUpdatePost_InvalidID(t *testing.T) {
	svc := newPostSvc()
	_, err := svc.UpdatePost(context.TODO(), "not-valid", "contenu", "u1", "")
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("UpdatePost(id invalide) → %v, attendu ErrInvalidID", err)
	}
}

// ─── PinPost / UnpinPost — id invalide ──────────────────────────────────────

func TestPinPost_InvalidID(t *testing.T) {
	svc := newPostSvc()
	_, err := svc.PinPost(context.TODO(), "bad", "u1")
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("PinPost(id invalide) → %v, attendu ErrInvalidID", err)
	}
}

func TestUnpinPost_InvalidID(t *testing.T) {
	svc := newPostSvc()
	_, err := svc.UnpinPost(context.TODO(), "bad", "u1")
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("UnpinPost(id invalide) → %v, attendu ErrInvalidID", err)
	}
}

// ─── DeletePost — id invalide ────────────────────────────────────────────────

func TestDeletePost_InvalidID(t *testing.T) {
	svc := newPostSvc()
	err := svc.DeletePost(context.TODO(), "bad-id", "u1", "")
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("DeletePost(id invalide) → %v, attendu ErrInvalidID", err)
	}
}

// ─── ListHiddenPosts — rôle insuffisant ─────────────────────────────────────

func TestListHiddenPosts_NotModerator(t *testing.T) {
	svc := newPostSvc()
	_, err := svc.ListHiddenPosts(context.TODO(), models.RoleUser, repository.HiddenFilter{}, 20, 0)
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("ListHiddenPosts(user) → %v, attendu ErrForbidden", err)
	}
}

// ─── RestorePost — id invalide ───────────────────────────────────────────────

func TestRestorePost_InvalidID(t *testing.T) {
	svc := newPostSvc()
	_, err := svc.RestorePost(context.TODO(), "not-valid", models.RoleModerator)
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("RestorePost(id invalide) → %v, attendu ErrInvalidID", err)
	}
}

func TestRestorePost_NotModerator(t *testing.T) {
	svc := newPostSvc()
	_, err := svc.RestorePost(context.TODO(), "507f1f77bcf86cd799439011", models.RoleUser)
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("RestorePost(user) → %v, attendu ErrForbidden", err)
	}
}

// ─── AutoHide / AutoUnhide — id invalide ────────────────────────────────────

func TestAutoHide_InvalidID(t *testing.T) {
	svc := newPostSvc()
	_, err := svc.AutoHide(context.TODO(), "bad")
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("AutoHide(id invalide) → %v, attendu ErrInvalidID", err)
	}
}

func TestAutoUnhide_InvalidID(t *testing.T) {
	svc := newPostSvc()
	_, err := svc.AutoUnhide(context.TODO(), "bad")
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("AutoUnhide(id invalide) → %v, attendu ErrInvalidID", err)
	}
}

// ─── PurgePost — rôle insuffisant puis id invalide ──────────────────────────

func TestPurgePost_NotModerator(t *testing.T) {
	svc := newPostSvc()
	err := svc.PurgePost(context.TODO(), "507f1f77bcf86cd799439011", "u1", models.RoleUser)
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("PurgePost(user) → %v, attendu ErrForbidden", err)
	}
}

func TestPurgePost_InvalidID(t *testing.T) {
	svc := newPostSvc()
	err := svc.PurgePost(context.TODO(), "bad-id", "u1", models.RoleModerator)
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("PurgePost(mod, id invalide) → %v, attendu ErrInvalidID", err)
	}
}

// ─── PurgeUserData — rôle insuffisant ───────────────────────────────────────

func TestPurgeUserData_NotAdmin(t *testing.T) {
	svc := newPostSvc()
	_, err := svc.PurgeUserData(context.TODO(), "u1", models.RoleModerator)
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("PurgeUserData(mod) → %v, attendu ErrForbidden", err)
	}
}

func TestPurgeUserData_User(t *testing.T) {
	svc := newPostSvc()
	_, err := svc.PurgeUserData(context.TODO(), "u1", models.RoleUser)
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("PurgeUserData(user) → %v, attendu ErrForbidden", err)
	}
}

// ─── VotePoll — id invalide ──────────────────────────────────────────────────

func TestVotePoll_InvalidID(t *testing.T) {
	svc := newPostSvc()
	_, err := svc.VotePoll(context.TODO(), "bad-id", "u1", "choice1")
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("VotePoll(id invalide) → %v, attendu ErrInvalidID", err)
	}
}

// ─── ClosePoll — id invalide ─────────────────────────────────────────────────

func TestClosePoll_InvalidID(t *testing.T) {
	svc := newPostSvc()
	_, err := svc.ClosePoll(context.TODO(), "bad-id", "u1")
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("ClosePoll(id invalide) → %v, attendu ErrInvalidID", err)
	}
}

// ─── SweepPurge — sans rétention configurée ─────────────────────────────────

func TestSweepPurge_NoPurgeAfter(t *testing.T) {
	svc := newPostSvc() // purgeAfter == 0
	warned, purged, err := svc.SweepPurge(context.TODO())
	if err != nil || warned != 0 || purged != 0 {
		t.Errorf("SweepPurge sans rétention → (%d, %d, %v), attendu (0, 0, nil)", warned, purged, err)
	}
}

// ─── buildPoll — validation ──────────────────────────────────────────────────

func TestBuildPoll_Nil(t *testing.T) {
	p, err := buildPoll(nil, time.Now())
	if err != nil || p != nil {
		t.Errorf("buildPoll(nil) → (%v, %v), attendu (nil, nil)", p, err)
	}
}

func TestBuildPoll_DurationTooShort(t *testing.T) {
	req := &models.CreatePollRequest{
		DurationMinutes: 0,
		Choices: []models.CreatePollChoiceRequest{
			{Label: "A"}, {Label: "B"},
		},
	}
	_, err := buildPoll(req, time.Now())
	if !errors.Is(err, ErrInvalidPoll) {
		t.Errorf("durée 0 → %v, attendu ErrInvalidPoll", err)
	}
}

func TestBuildPoll_DurationTooLong(t *testing.T) {
	req := &models.CreatePollRequest{
		DurationMinutes: 7*24*60 + 1,
		Choices: []models.CreatePollChoiceRequest{
			{Label: "A"}, {Label: "B"},
		},
	}
	_, err := buildPoll(req, time.Now())
	if !errors.Is(err, ErrInvalidPoll) {
		t.Errorf("durée trop longue → %v, attendu ErrInvalidPoll", err)
	}
}

func TestBuildPoll_InvalidAudience(t *testing.T) {
	req := &models.CreatePollRequest{
		DurationMinutes: 60,
		Audience:        "moderators",
		Choices: []models.CreatePollChoiceRequest{
			{Label: "A"}, {Label: "B"},
		},
	}
	_, err := buildPoll(req, time.Now())
	if !errors.Is(err, ErrInvalidPoll) {
		t.Errorf("audience invalide → %v, attendu ErrInvalidPoll", err)
	}
}

func TestBuildPoll_TooFewChoices(t *testing.T) {
	req := &models.CreatePollRequest{
		DurationMinutes: 60,
		Choices:         []models.CreatePollChoiceRequest{{Label: "Seul choix"}},
	}
	_, err := buildPoll(req, time.Now())
	if !errors.Is(err, ErrInvalidPoll) {
		t.Errorf("1 choix → %v, attendu ErrInvalidPoll", err)
	}
}

func TestBuildPoll_TooManyChoices(t *testing.T) {
	req := &models.CreatePollRequest{
		DurationMinutes: 60,
		Choices: []models.CreatePollChoiceRequest{
			{Label: "A"}, {Label: "B"}, {Label: "C"}, {Label: "D"}, {Label: "E"},
		},
	}
	_, err := buildPoll(req, time.Now())
	if !errors.Is(err, ErrInvalidPoll) {
		t.Errorf("5 choix → %v, attendu ErrInvalidPoll", err)
	}
}

func TestBuildPoll_EmptyChoiceLabel(t *testing.T) {
	req := &models.CreatePollRequest{
		DurationMinutes: 60,
		Choices:         []models.CreatePollChoiceRequest{{Label: "  "}, {Label: "B"}},
	}
	_, err := buildPoll(req, time.Now())
	if !errors.Is(err, ErrInvalidPoll) {
		t.Errorf("label vide → %v, attendu ErrInvalidPoll", err)
	}
}

func TestBuildPoll_LabelTooLong(t *testing.T) {
	req := &models.CreatePollRequest{
		DurationMinutes: 60,
		Choices: []models.CreatePollChoiceRequest{
			{Label: strings.Repeat("x", 81)}, {Label: "B"},
		},
	}
	_, err := buildPoll(req, time.Now())
	if !errors.Is(err, ErrInvalidPoll) {
		t.Errorf("label 81 chars → %v, attendu ErrInvalidPoll", err)
	}
}

func TestBuildPoll_DuplicateLabels(t *testing.T) {
	req := &models.CreatePollRequest{
		DurationMinutes: 60,
		Choices: []models.CreatePollChoiceRequest{
			{Label: "Oui"}, {Label: "oui"},
		},
	}
	_, err := buildPoll(req, time.Now())
	if !errors.Is(err, ErrInvalidPoll) {
		t.Errorf("labels en double → %v, attendu ErrInvalidPoll", err)
	}
}

func TestBuildPoll_ImageURLTooLong(t *testing.T) {
	req := &models.CreatePollRequest{
		DurationMinutes: 60,
		Choices: []models.CreatePollChoiceRequest{
			{Label: "A", ImageURL: strings.Repeat("x", 513)}, {Label: "B"},
		},
	}
	_, err := buildPoll(req, time.Now())
	if !errors.Is(err, ErrInvalidPoll) {
		t.Errorf("imageURL > 512 → %v, attendu ErrInvalidPoll", err)
	}
}

func TestBuildPoll_Valid(t *testing.T) {
	req := &models.CreatePollRequest{
		DurationMinutes: 60,
		Choices: []models.CreatePollChoiceRequest{
			{Label: "Oui"}, {Label: "Non"},
		},
	}
	p, err := buildPoll(req, time.Now())
	if err != nil {
		t.Fatalf("buildPoll valide → erreur inattendue : %v", err)
	}
	if p == nil || len(p.Choices) != 2 {
		t.Fatalf("buildPoll valide → poll = %v, attendu 2 choix", p)
	}
}
