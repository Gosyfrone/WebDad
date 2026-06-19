package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/webdad/profil-service/internal/models"
)

// ─── normalizeActivity ───────────────────────────────────────────────────────

func TestNormalizeActivity_NilProfil(t *testing.T) {
	// ne doit pas paniquer
	normalizeActivity(nil, time.Now())
}

func TestNormalizeActivity_OfflineInchangé(t *testing.T) {
	p := &models.Profil{IsOnline: false}
	normalizeActivity(p, time.Now())
	if p.IsOnline {
		t.Fatal("un profil déjà hors-ligne ne doit pas être modifié")
	}
}

func TestNormalizeActivity_OnlineRécent(t *testing.T) {
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	loginAt := now.Add(-10 * time.Second) // dans la grâce (< 45s)
	p := &models.Profil{IsOnline: true, LastLoginAt: &loginAt}
	normalizeActivity(p, now)
	if !p.IsOnline {
		t.Fatal("un profil connecté récemment doit rester en ligne")
	}
}

func TestNormalizeActivity_OnlineExpiré(t *testing.T) {
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	loginAt := now.Add(-2 * time.Minute) // > 45s
	p := &models.Profil{IsOnline: true, LastLoginAt: &loginAt}
	normalizeActivity(p, now)
	if p.IsOnline {
		t.Fatal("un profil expiré doit passer hors-ligne")
	}
}

func TestNormalizeActivity_OnlineSansLoginAt(t *testing.T) {
	p := &models.Profil{IsOnline: true, LastLoginAt: nil}
	normalizeActivity(p, time.Now())
	// sans LastLoginAt, IsOnline ne doit pas être forcé vrai (pas de crash)
}

// ─── mapGet ──────────────────────────────────────────────────────────────────

func TestMapGet_ErrNoDocuments(t *testing.T) {
	_, err := mapGet(nil, mongo.ErrNoDocuments)
	if !errors.Is(err, ErrProfilNotFound) {
		t.Fatalf("mapGet(ErrNoDocuments) = %v, attendu ErrProfilNotFound", err)
	}
}

func TestMapGet_AutreErreur(t *testing.T) {
	sentinel := errors.New("db error")
	_, err := mapGet(nil, sentinel)
	if !errors.Is(err, sentinel) {
		t.Fatalf("mapGet(sentinel) = %v, attendu %v", err, sentinel)
	}
}

func TestMapGet_Succès(t *testing.T) {
	p := &models.Profil{UserID: "u1"}
	got, err := mapGet(p, nil)
	if err != nil {
		t.Fatalf("mapGet(nil err) = %v, attendu nil", err)
	}
	if got.UserID != "u1" {
		t.Fatalf("got.UserID = %q, attendu u1", got.UserID)
	}
}

// ─── Search (court-circuit terme vide) ───────────────────────────────────────

func TestSearch_TermeVide(t *testing.T) {
	svc := New(nil, 0)
	results, err := svc.Search(context.Background(), "", 10)
	if err != nil {
		t.Fatalf("Search('') erreur inattendue : %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("Search('') = %d résultats, attendu 0", len(results))
	}
}

func TestSearch_TermeEspaces(t *testing.T) {
	svc := New(nil, 0)
	results, err := svc.Search(context.Background(), "   ", 10)
	if err != nil {
		t.Fatalf("Search('   ') erreur inattendue : %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("Search('   ') = %d résultats, attendu 0", len(results))
	}
}

// ─── CanViewActivity ─────────────────────────────────────────────────────────

type stubFollowChecker struct {
	isFollowing bool
	err         error
}

func (s *stubFollowChecker) IsFollowing(_ context.Context, _, _ string) (bool, error) {
	return s.isFollowing, s.err
}
func (s *stubFollowChecker) AcceptAllFollowRequests(_ context.Context, _ string) error {
	return nil
}

func TestCanViewActivity(t *testing.T) {
	now := time.Now()
	p := &models.Profil{
		UserID:             "u1",
		ActivityVisibility: models.VisibilityPublic,
		Visibility:         models.VisibilityPublic,
		LastLoginAt:        &now,
	}

	svc := New(nil, 0)

	// Profil nil → false
	if svc.CanViewActivity(context.Background(), "viewer", nil) {
		t.Fatal("profil nil doit retourner false")
	}

	// ActivityVisibility privée → false
	p.ActivityVisibility = models.VisibilityPrivate
	if svc.CanViewActivity(context.Background(), "viewer", p) {
		t.Fatal("ActivityVisibility=private doit retourner false")
	}
	p.ActivityVisibility = models.VisibilityPublic

	// Propriétaire → toujours true
	if !svc.CanViewActivity(context.Background(), "u1", p) {
		t.Fatal("le propriétaire doit toujours voir son activité")
	}

	// Profil public → true pour tout viewer
	if !svc.CanViewActivity(context.Background(), "other", p) {
		t.Fatal("profil public : tout viewer doit voir l'activité")
	}

	// Profil privé, viewer vide → false
	p.Visibility = models.VisibilityPrivate
	if svc.CanViewActivity(context.Background(), "", p) {
		t.Fatal("profil privé sans viewer → false")
	}

	// Profil privé, viewer abonné → true
	svcWithFollow := New(nil, 0, WithFollowChecker(&stubFollowChecker{isFollowing: true}))
	if !svcWithFollow.CanViewActivity(context.Background(), "follower", p) {
		t.Fatal("profil privé + abonné → true")
	}

	// Profil privé, viewer non abonné → false
	svcNoFollow := New(nil, 0, WithFollowChecker(&stubFollowChecker{isFollowing: false}))
	if svcNoFollow.CanViewActivity(context.Background(), "stranger", p) {
		t.Fatal("profil privé + non-abonné → false")
	}
}
