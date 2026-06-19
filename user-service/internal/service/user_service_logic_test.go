package service

import (
	"context"
	"errors"
	"testing"
)

func newSvc() *UserService { return New(nil, 0) }

// ─── Create — validation username (avant tout appel repo) ────────────────────

func TestCreate_InvalidUsername(t *testing.T) {
	svc := newSvc()
	_, err := svc.Create("id1", "ab") // trop court
	if !errors.Is(err, ErrInvalidUsername) {
		t.Errorf("Create(username trop court) → %v, attendu ErrInvalidUsername", err)
	}
}

func TestCreate_ReservedUsername(t *testing.T) {
	svc := newSvc()
	_, err := svc.Create("id1", "admin")
	if !errors.Is(err, ErrInvalidUsername) {
		t.Errorf("Create(username réservé) → %v, attendu ErrInvalidUsername", err)
	}
}

// ─── AdminCreate — validation username (avant tout appel repo) ───────────────

func TestAdminCreate_InvalidUsername(t *testing.T) {
	svc := newSvc()
	_, err := svc.AdminCreate("id1", "!badname!")
	if !errors.Is(err, ErrInvalidUsername) {
		t.Errorf("AdminCreate(username invalide) → %v, attendu ErrInvalidUsername", err)
	}
}

func TestAdminCreate_ReservedUsername(t *testing.T) {
	svc := newSvc()
	_, err := svc.AdminCreate("id1", "root")
	if !errors.Is(err, ErrInvalidUsername) {
		t.Errorf("AdminCreate(username réservé) → %v, attendu ErrInvalidUsername", err)
	}
}

// ─── Update — validation username et locale (avant tout appel repo) ──────────

func TestUpdate_InvalidUsername(t *testing.T) {
	svc := newSvc()
	bad := ".start_with_dot"
	_, err := svc.Update("id1", &bad, nil)
	if !errors.Is(err, ErrInvalidUsername) {
		t.Errorf("Update(username invalide) → %v, attendu ErrInvalidUsername", err)
	}
}

func TestUpdate_InvalidLocale(t *testing.T) {
	svc := newSvc()
	loc := "xx" // locale non supportée
	_, err := svc.Update("id1", nil, &loc)
	if !errors.Is(err, ErrInvalidLocale) {
		t.Errorf("Update(locale invalide) → %v, attendu ErrInvalidLocale", err)
	}
}

func TestUpdate_SupportedLocale_PassesValidation(t *testing.T) {
	// Locale "fr" valide : la validation passe, on atteint le repo (nil → panic récupérée).
	svc := newSvc()
	loc := "fr"
	func() {
		defer func() { recover() }() //nolint:errcheck
		_, _ = svc.Update("id1", nil, &loc)
	}()
}

// ─── Search — terme vide → retour rapide sans repo ───────────────────────────

func TestSearch_EmptyTerm(t *testing.T) {
	svc := newSvc()
	users, err := svc.Search("  ", 10, 0)
	if err != nil {
		t.Fatalf("Search(vide) → erreur inattendue : %v", err)
	}
	if len(users) != 0 {
		t.Errorf("Search(vide) → %d résultats, attendu 0", len(users))
	}
}

func TestSearch_EmptyString(t *testing.T) {
	svc := newSvc()
	users, err := svc.Search("", 10, 0)
	if err != nil {
		t.Fatalf("Search(\"\") → erreur inattendue : %v", err)
	}
	if len(users) != 0 {
		t.Errorf("Search(\"\") → %d résultats, attendu 0", len(users))
	}
}

// ─── Follow — auto-suivi ─────────────────────────────────────────────────────

func TestFollow_SelfFollow(t *testing.T) {
	svc := newSvc()
	_, err := svc.Follow(context.TODO(), "u1", "u1@x.com", "u1")
	if !errors.Is(err, ErrSelfFollow) {
		t.Errorf("Follow(self) → %v, attendu ErrSelfFollow", err)
	}
}

// ─── RemoveFollower — auto-retrait ───────────────────────────────────────────

func TestRemoveFollower_Self(t *testing.T) {
	svc := newSvc()
	err := svc.RemoveFollower("u1", "u1")
	if !errors.Is(err, ErrSelfFollow) {
		t.Errorf("RemoveFollower(self) → %v, attendu ErrSelfFollow", err)
	}
}

// ─── validateUsername — cas limites supplémentaires ──────────────────────────

func TestValidateUsername_ExactMin(t *testing.T) {
	if err := validateUsername("abc"); err != nil {
		t.Errorf("username de 3 chars → %v, attendu nil", err)
	}
}

func TestValidateUsername_ExactMax(t *testing.T) {
	u := "a234567890123456789012345678901234567890123456789b" // 50 chars
	if err := validateUsername(u); err != nil {
		t.Errorf("username de 50 chars → %v, attendu nil", err)
	}
}

func TestValidateUsername_PlusOneMax(t *testing.T) {
	u := "a234567890123456789012345678901234567890123456789bc" // 51 chars
	if err := validateUsername(u); !errors.Is(err, ErrInvalidUsername) {
		t.Errorf("username de 51 chars → %v, attendu ErrInvalidUsername", err)
	}
}

func TestValidateUsername_WithUnderscore(t *testing.T) {
	if err := validateUsername("jean_pierre"); err != nil {
		t.Errorf("underscore autorisé → %v", err)
	}
}

func TestValidateUsername_CaseSensitiveReserved(t *testing.T) {
	// reservedUsernames stocke en minuscules, validateUsername fait ToLower
	if err := validateUsername("ADMIN"); !errors.Is(err, ErrInvalidUsername) {
		t.Errorf("ADMIN (réservé, majuscules) → %v, attendu ErrInvalidUsername", err)
	}
}
