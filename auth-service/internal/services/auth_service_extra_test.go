// Tests supplémentaires couvrant les chemins anti-énumération et les gardes
// précoces de l'AuthService via le fake SQL driver (aucune vraie DB).
package services

import (
	"database/sql/driver"
	"errors"
	"testing"
	"time"
)

// userRow construit une ligne credentials complète pour le fake driver.
// cols: id, email, role, is_active, email_verified, created_at
func userRow(id, email, role string, active, verified bool) *fakeQ {
	return &fakeQ{
		cols: []string{"id", "email", "role", "is_active", "email_verified", "created_at"},
		rows: [][]driver.Value{
			{id, email, role, active, verified, time.Now()},
		},
	}
}

// ─── Login / LoginByUserID — anti-énumération (compte absent) ─────────────────

func TestLogin_ComptAbsent_ErrInvalidCredentials(t *testing.T) {
	db := openFakeDB(t) // fake vide → 0 lignes
	svc := newSvc(t, db)
	_, err := svc.Login("nobody@x.dev", "secret")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login(absent) → %v, attendu ErrInvalidCredentials", err)
	}
}

func TestLoginByUserID_ComptAbsent_ErrInvalidCredentials(t *testing.T) {
	db := openFakeDB(t)
	svc := newSvc(t, db)
	_, err := svc.LoginByUserID("non-existent-uid", "secret")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("LoginByUserID(absent) → %v, attendu ErrInvalidCredentials", err)
	}
}

// ─── Refresh — token inconnu ──────────────────────────────────────────────────

func TestRefresh_TokenAbsent_ErrInvalidRefreshToken(t *testing.T) {
	db := openFakeDB(t) // fake vide → 0 lignes
	svc := newSvc(t, db)
	_, _, _, err := svc.Refresh("raw-token-inexistant")
	if !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("Refresh(absent) → %v, attendu ErrInvalidRefreshToken", err)
	}
}

// ─── VerifyEmail / ResetPassword — token inconnu ─────────────────────────────

func TestVerifyEmail_TokenAbsent_ErrInvalidToken(t *testing.T) {
	db := openFakeDB(t) // consumeAccountToken → 0 lignes → ErrInvalidToken
	svc := newSvc(t, db)
	_, _, _, err := svc.VerifyEmail("unknown-token")
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("VerifyEmail(absent) → %v, attendu ErrInvalidToken", err)
	}
}

func TestResetPassword_TokenAbsent_ErrInvalidToken(t *testing.T) {
	db := openFakeDB(t) // consumeAccountToken → 0 lignes → ErrInvalidToken
	svc := newSvc(t, db)
	err := svc.ResetPassword("unknown-token", "NewP@ss1!")
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("ResetPassword(absent) → %v, attendu ErrInvalidToken", err)
	}
}

// ─── RequestEmailChange — utilisateur absent ──────────────────────────────────

func TestRequestEmailChange_UserAbsent_ErrUserNotFound(t *testing.T) {
	db := openFakeDB(t) // SELECT → 0 lignes → ErrUserNotFound
	svc := newSvc(t, db)
	err := svc.RequestEmailChange("non-existent-uid", "new@x.dev")
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("RequestEmailChange(absent) → %v, attendu ErrUserNotFound", err)
	}
}

// ─── sendResetMail — mailer nil (via ForgotPassword avec un vrai utilisateur) ─

// ForgotPassword → SELECT retourne un utilisateur actif → sendResetMail → mailer
// nil → early return (pas de panique, pas d'erreur propagée).
func TestForgotPassword_UserActif_SendResetMailNoOp(t *testing.T) {
	db := openFakeDB(t)
	gFakeDrv.queries = []*fakeQ{
		userRow("uid1", "alice@x.dev", "user", true, true),
	}
	svc := newSvc(t, db) // mailer nil
	err := svc.ForgotPassword("alice@x.dev")
	if err != nil {
		t.Fatalf("ForgotPassword (actif, mailer nil) → %v, attendu nil", err)
	}
}

// Compte désactivé → ForgotPassword retourne nil sans appeler sendResetMail.
func TestForgotPassword_UserInactif_NoOp(t *testing.T) {
	db := openFakeDB(t)
	gFakeDrv.queries = []*fakeQ{
		userRow("uid2", "banned@x.dev", "user", false, true),
	}
	svc := newSvc(t, db)
	if err := svc.ForgotPassword("banned@x.dev"); err != nil {
		t.Fatalf("ForgotPassword (inactif) → %v, attendu nil", err)
	}
}

// ─── sendVerificationMail — mailer nil (via ResendVerification) ───────────────

// ResendVerification → SELECT retourne un utilisateur non vérifié →
// sendVerificationMail → mailer nil → early return (pas de panique).
func TestResendVerification_NonVerifié_SendVerifMailNoOp(t *testing.T) {
	db := openFakeDB(t)
	gFakeDrv.queries = []*fakeQ{
		userRow("uid3", "unverified@x.dev", "user", true, false),
	}
	svc := newSvc(t, db) // mailer nil
	err := svc.ResendVerification("unverified@x.dev")
	if err != nil {
		t.Fatalf("ResendVerification (non vérifié, mailer nil) → %v, attendu nil", err)
	}
}

// Compte déjà vérifié → ResendVerification retourne nil sans envoyer de mail.
func TestResendVerification_DéjàVerifié_NoOp(t *testing.T) {
	db := openFakeDB(t)
	gFakeDrv.queries = []*fakeQ{
		userRow("uid4", "verified@x.dev", "user", true, true),
	}
	svc := newSvc(t, db)
	if err := svc.ResendVerification("verified@x.dev"); err != nil {
		t.Fatalf("ResendVerification (déjà vérifié) → %v, attendu nil", err)
	}
}
