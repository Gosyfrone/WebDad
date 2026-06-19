// Tests de l'AuthService — logique pure et chemins couverts sans appels SQL réels.
// Stratégie :
//   - fonctions pures (GenerateToken, ParseToken, errIfNoRows…) testées directement ;
//   - gardes "MFA non configurée" (erreur précoce avant toute requête DB) ;
//   - Logout(token vide) et chemins anti-énumération qui font une requête mais
//     ne paniquent pas avec le fake driver ;
//   - panics DB nil couvertes pour Register/Login pour valider que le garde
//     de la nil-DB est bien déclenché par le runtime.
package services

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/webdad/auth-service/internal/models"
)

// ─── fake SQL driver ─────────────────────────────────────────────────────────

const testFakeDriverName = "fakeauth_svc"

var gFakeDrv *fakeSvcDrv

func init() {
	gFakeDrv = &fakeSvcDrv{}
	sql.Register(testFakeDriverName, gFakeDrv)
}

// fakeSvcDrv : driver pilotable par test. Les tests remplissent gFakeDrv.queries
// avant chaque appel ; la file est consommée dans l'ordre (FIFO).
type fakeSvcDrv struct {
	queries []*fakeQ
}

type fakeQ struct {
	cols []string
	rows [][]driver.Value
	err  error
}

func (d *fakeSvcDrv) reset() { d.queries = nil }

// driver.Driver
func (d *fakeSvcDrv) Open(_ string) (driver.Conn, error) {
	return &fakeSvcConn{drv: d}, nil
}

type fakeSvcConn struct{ drv *fakeSvcDrv }

func (c *fakeSvcConn) Prepare(_ string) (driver.Stmt, error) {
	return &fakeSvcStmt{drv: c.drv}, nil
}
func (c *fakeSvcConn) Close() error              { return nil }
func (c *fakeSvcConn) Begin() (driver.Tx, error) { return &fakeSvcTx{}, nil }

type fakeSvcTx struct{}

func (t *fakeSvcTx) Commit() error   { return nil }
func (t *fakeSvcTx) Rollback() error { return nil }

type fakeSvcStmt struct{ drv *fakeSvcDrv }

func (s *fakeSvcStmt) Close() error  { return nil }
func (s *fakeSvcStmt) NumInput() int { return -1 }

func (s *fakeSvcStmt) Exec(_ []driver.Value) (driver.Result, error) {
	if len(s.drv.queries) > 0 {
		q := s.drv.queries[0]
		s.drv.queries = s.drv.queries[1:]
		if q.err != nil {
			return nil, q.err
		}
	}
	return &fakeSvcResult{n: 1}, nil
}

func (s *fakeSvcStmt) Query(_ []driver.Value) (driver.Rows, error) {
	if len(s.drv.queries) == 0 {
		return &fakeSvcRows{}, nil
	}
	q := s.drv.queries[0]
	s.drv.queries = s.drv.queries[1:]
	if q.err != nil {
		return nil, q.err
	}
	return &fakeSvcRows{cols: q.cols, rows: q.rows}, nil
}

type fakeSvcRows struct {
	cols []string
	rows [][]driver.Value
	pos  int
}

func (r *fakeSvcRows) Columns() []string { return r.cols }
func (r *fakeSvcRows) Close() error      { return nil }
func (r *fakeSvcRows) Next(dest []driver.Value) error {
	if r.pos >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.pos])
	r.pos++
	return nil
}

type fakeSvcResult struct{ n int64 }

func (r *fakeSvcResult) LastInsertId() (int64, error) { return 0, nil }
func (r *fakeSvcResult) RowsAffected() (int64, error) { return r.n, nil }

// ─── helpers ─────────────────────────────────────────────────────────────────

func openFakeDB(t *testing.T) *sql.DB {
	t.Helper()
	gFakeDrv.reset()
	db, err := sql.Open(testFakeDriverName, "")
	if err != nil {
		t.Fatalf("sql.Open fake: %v", err)
	}
	db.SetMaxOpenConns(1)
	return db
}

func newSvc(t *testing.T, db *sql.DB) *AuthService {
	t.Helper()
	svc, err := New(db, "test-secret-jwt-key", 15*time.Minute, 24*time.Hour, nil,
		"http://localhost:3000", "", false, "")
	if err != nil {
		t.Fatalf("services.New: %v", err)
	}
	return svc
}

// ─── GenerateToken / ParseToken ───────────────────────────────────────────────

func TestGenerateAndParseToken(t *testing.T) {
	svc := newSvc(t, openFakeDB(t))
	u := &models.User{ID: "uid-1", Email: "user@test.dev", Role: models.RoleUser}
	tok, err := svc.GenerateToken(u)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	if tok == "" {
		t.Fatal("token vide")
	}
	claims, err := svc.ParseToken(tok)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}
	if claims.UserID != u.ID || claims.Email != u.Email || claims.Role != u.Role {
		t.Fatalf("claims incorrects: %+v", claims)
	}
}

func TestGenerateToken_MustChangePassword(t *testing.T) {
	svc := newSvc(t, openFakeDB(t))
	u := &models.User{ID: "uid-2", Email: "a@x.dev", Role: models.RoleAdmin, MustChangePassword: true}
	tok, _ := svc.GenerateToken(u)
	claims, err := svc.ParseToken(tok)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}
	if !claims.MustChangePassword {
		t.Fatal("MustChangePassword absent du token")
	}
}

func TestParseToken_MauvaisSecret(t *testing.T) {
	db := openFakeDB(t)
	svc := newSvc(t, db)
	tok, _ := svc.GenerateToken(&models.User{ID: "u", Role: models.RoleUser})
	svc2, _ := New(db, "autre-secret", 15*time.Minute, 24*time.Hour, nil, "", "", false, "")
	if _, err := svc2.ParseToken(tok); err == nil {
		t.Fatal("mauvais secret devrait échouer")
	}
}

func TestParseToken_TokenCorrompu(t *testing.T) {
	svc := newSvc(t, openFakeDB(t))
	if _, err := svc.ParseToken("not.a.valid.jwt"); err == nil {
		t.Fatal("token corrompu devrait échouer")
	}
}

func TestParseToken_TokenVide(t *testing.T) {
	svc := newSvc(t, openFakeDB(t))
	if _, err := svc.ParseToken(""); err == nil {
		t.Fatal("token vide devrait échouer")
	}
}

func TestParseToken_TokenExpiré(t *testing.T) {
	db := openFakeDB(t)
	svc, _ := New(db, "secret", -time.Second, 24*time.Hour, nil, "", "", false, "")
	tok, err := svc.GenerateToken(&models.User{ID: "u", Email: "u@x.dev", Role: models.RoleUser})
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	if _, err := svc.ParseToken(tok); err == nil {
		t.Fatal("token expiré devrait être refusé")
	}
}

func TestGenerateToken_TousLesRôles(t *testing.T) {
	svc := newSvc(t, openFakeDB(t))
	for _, role := range []string{models.RoleUser, models.RoleModerator, models.RoleAdmin} {
		tok, err := svc.GenerateToken(&models.User{ID: "r", Email: "r@x.dev", Role: role})
		if err != nil {
			t.Errorf("GenerateToken(%q): %v", role, err)
			continue
		}
		claims, err := svc.ParseToken(tok)
		if err != nil {
			t.Errorf("ParseToken(%q): %v", role, err)
			continue
		}
		if claims.Role != role {
			t.Errorf("Role = %q, attendu %q", claims.Role, role)
		}
	}
}

// ─── New — construction ───────────────────────────────────────────────────────

func TestNew_CléMFAInvalide(t *testing.T) {
	if _, err := New(nil, "s", time.Minute, time.Hour, nil, "", "", false, "!!!not-base64!!!"); err == nil {
		t.Fatal("clé MFA invalide devrait échouer")
	}
}

func TestNew_CléMFATropCourte(t *testing.T) {
	// "short" en base64 = 5 octets → doit être refusé (32 attendus).
	short := "c2hvcnQ="
	if _, err := New(nil, "s", time.Minute, time.Hour, nil, "", "", false, short); err == nil {
		t.Fatal("clé MFA trop courte devrait échouer")
	}
}

func TestNew_CléMFAVide_OK(t *testing.T) {
	svc, err := New(nil, "s", time.Minute, time.Hour, nil, "", "", false, "")
	if err != nil {
		t.Fatalf("clé vide ne doit pas échouer : %v", err)
	}
	if svc.MFAConfigured() {
		t.Fatal("MFA doit être désactivée")
	}
}

func TestNew_AvecMailer(t *testing.T) {
	mailer := &mockMailer{}
	svc, err := New(nil, "secret", time.Minute, time.Hour, mailer, "http://localhost:3000", "", false, "")
	if err != nil {
		t.Fatalf("New avec mailer: %v", err)
	}
	if svc.mailer == nil {
		t.Fatal("mailer doit être non-nil")
	}
}

// ─── MFAConfigured ────────────────────────────────────────────────────────────

func TestMFAConfigured_False(t *testing.T) {
	svc := newSvc(t, openFakeDB(t))
	if svc.MFAConfigured() {
		t.Fatal("MFAConfigured devrait être false")
	}
}

func TestMFAConfigured_True(t *testing.T) {
	svc, err := New(nil, "secret", time.Minute, time.Hour, nil, "", "", false, validKey(t))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if !svc.MFAConfigured() {
		t.Fatal("MFAConfigured devrait être true")
	}
}

// ─── Gardes « MFA non configurée » ───────────────────────────────────────────

func TestSetupMFA_NonConfiguré(t *testing.T) {
	svc := newSvc(t, openFakeDB(t))
	if _, err := svc.SetupMFA("uid"); !errors.Is(err, ErrMFANotConfigured) {
		t.Fatalf("attendu ErrMFANotConfigured, obtenu %v", err)
	}
}

func TestEnableMFA_NonConfiguré(t *testing.T) {
	svc := newSvc(t, openFakeDB(t))
	if err := svc.EnableMFA("uid", "123456"); !errors.Is(err, ErrMFANotConfigured) {
		t.Fatalf("attendu ErrMFANotConfigured, obtenu %v", err)
	}
}

func TestDisableMFA_NonConfiguré(t *testing.T) {
	svc := newSvc(t, openFakeDB(t))
	if err := svc.DisableMFA("uid", "", "pwd"); !errors.Is(err, ErrMFANotConfigured) {
		t.Fatalf("attendu ErrMFANotConfigured, obtenu %v", err)
	}
}

func TestVerifyMFA_NonConfiguré(t *testing.T) {
	svc := newSvc(t, openFakeDB(t))
	if _, _, _, err := svc.VerifyMFA("ch", "code"); !errors.Is(err, ErrMFANotConfigured) {
		t.Fatalf("attendu ErrMFANotConfigured, obtenu %v", err)
	}
}

// ─── Logout ───────────────────────────────────────────────────────────────────

func TestLogout_TokenVide(t *testing.T) {
	svc := newSvc(t, openFakeDB(t))
	// Token vide → idempotent, aucune requête DB.
	if err := svc.Logout(""); err != nil {
		t.Fatalf("Logout('') = %v, attendu nil", err)
	}
}

func TestLogout_TokenPrésent(t *testing.T) {
	svc := newSvc(t, openFakeDB(t))
	if err := svc.Logout("some-raw-token-value"); err != nil {
		t.Fatalf("Logout = %v, attendu nil", err)
	}
}

// ─── EnsureDefaultAdmin ───────────────────────────────────────────────────────

func TestEnsureDefaultAdmin_OK(t *testing.T) {
	svc := newSvc(t, openFakeDB(t))
	if err := svc.EnsureDefaultAdmin("admin@breezy.dev", "Admin1234!"); err != nil {
		t.Fatalf("EnsureDefaultAdmin = %v", err)
	}
}

// ─── SetRole ──────────────────────────────────────────────────────────────────

func TestSetRole_RôleInvalide(t *testing.T) {
	svc := newSvc(t, openFakeDB(t))
	if err := svc.SetRole("uid", "superadmin"); !errors.Is(err, ErrInvalidRole) {
		t.Fatalf("attendu ErrInvalidRole, obtenu %v", err)
	}
}

func TestSetRole_RôleVide(t *testing.T) {
	svc := newSvc(t, openFakeDB(t))
	if err := svc.SetRole("uid", ""); !errors.Is(err, ErrInvalidRole) {
		t.Fatalf("attendu ErrInvalidRole pour rôle vide, obtenu %v", err)
	}
}

func TestSetRole_RôleValide_PasErrInvalidRole(t *testing.T) {
	svc := newSvc(t, openFakeDB(t))
	err := svc.SetRole("some-uid", models.RoleModerator)
	if errors.Is(err, ErrInvalidRole) {
		t.Fatal("rôle valide ne doit pas retourner ErrInvalidRole")
	}
}

// ─── validRole — couverture complète ─────────────────────────────────────────

func TestValidRole_Complet(t *testing.T) {
	for _, r := range []string{models.RoleUser, models.RoleModerator, models.RoleAdmin} {
		if !validRole(r) {
			t.Errorf("validRole(%q) = false, attendu true", r)
		}
	}
	for _, r := range []string{"", "root", "User", "Admin", "MODERATOR", "superadmin"} {
		if validRole(r) {
			t.Errorf("validRole(%q) = true, attendu false", r)
		}
	}
}

// ─── mintSystemAdminToken ─────────────────────────────────────────────────────

func TestMintSystemAdminToken(t *testing.T) {
	svc := newSvc(t, openFakeDB(t))
	tok, err := svc.mintSystemAdminToken()
	if err != nil {
		t.Fatalf("mintSystemAdminToken: %v", err)
	}
	claims, err := svc.ParseToken(tok)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}
	if claims.Role != models.RoleAdmin {
		t.Fatalf("rôle = %q, attendu admin", claims.Role)
	}
	if claims.UserID != systemActorID {
		t.Fatalf("UserID = %q, attendu %q", claims.UserID, systemActorID)
	}
}

// ─── errIfNoRows ──────────────────────────────────────────────────────────────

func TestErrIfNoRows_ZeroLigne(t *testing.T) {
	if err := errIfNoRows(&fakeSvcResult{n: 0}); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("errIfNoRows(0) = %v, attendu ErrUserNotFound", err)
	}
}

func TestErrIfNoRows_UneLigne(t *testing.T) {
	if err := errIfNoRows(&fakeSvcResult{n: 1}); err != nil {
		t.Fatalf("errIfNoRows(1) = %v, attendu nil", err)
	}
}

func TestErrIfNoRows_PlusieursLignes(t *testing.T) {
	if err := errIfNoRows(&fakeSvcResult{n: 5}); err != nil {
		t.Fatalf("errIfNoRows(5) = %v, attendu nil", err)
	}
}

// ─── SweepBannedAccounts — after ≤ 0 ────────────────────────────────────────

func TestSweepBannedAccounts_AfterNul(t *testing.T) {
	svc := newSvc(t, openFakeDB(t))
	n, err := svc.SweepBannedAccounts(context.TODO(), nil, 0)
	if err != nil || n != 0 {
		t.Fatalf("SweepBannedAccounts(0) = (%d, %v), attendu (0, nil)", n, err)
	}
}

func TestSweepBannedAccounts_AfterNégatif(t *testing.T) {
	svc := newSvc(t, openFakeDB(t))
	n, err := svc.SweepBannedAccounts(context.TODO(), nil, -time.Hour)
	if err != nil || n != 0 {
		t.Fatalf("SweepBannedAccounts(-1h) = (%d, %v), attendu (0, nil)", n, err)
	}
}

// ─── RunAccountPurgeSweeper — no-op si after=0 ───────────────────────────────

func TestRunAccountPurgeSweeper_NoOp(t *testing.T) {
	svc := newSvc(t, openFakeDB(t))
	done := make(chan struct{})
	go func() {
		svc.RunAccountPurgeSweeper(context.TODO(), nil, 0, time.Hour)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("RunAccountPurgeSweeper(after=0) ne s'est pas terminé")
	}
}

// ─── hashToken — cas extrêmes ─────────────────────────────────────────────────

func TestHashToken_CasExtrêmes(t *testing.T) {
	long := string(make([]byte, 10000))
	if h := hashToken(long); len(h) != 64 {
		t.Fatalf("hashToken(long): len = %d, attendu 64", len(h))
	}
	if h := hashToken("🔒sécurité"); len(h) != 64 {
		t.Fatalf("hashToken unicode: len = %d", len(h))
	}
	h1, h2 := hashToken("abc"), hashToken("abc")
	if h1 != h2 {
		t.Fatal("hashToken doit être déterministe")
	}
}

// ─── ForgotPassword / ResendVerification — anti-énumération (base vide) ──────

func TestForgotPassword_ComptAbsent_NoOp(t *testing.T) {
	svc := newSvc(t, openFakeDB(t))
	// Base vide → 0 lignes → no-op, nil attendu.
	if err := svc.ForgotPassword("inconnu@x.dev"); err != nil {
		t.Fatalf("ForgotPassword (absent) = %v, attendu nil", err)
	}
}

func TestResendVerification_ComptAbsent_NoOp(t *testing.T) {
	svc := newSvc(t, openFakeDB(t))
	if err := svc.ResendVerification("ghost@x.dev"); err != nil {
		t.Fatalf("ResendVerification (absent) = %v, attendu nil", err)
	}
}

// ─── mockMailer ───────────────────────────────────────────────────────────────

type mockMailer struct{ sent []string }

func (m *mockMailer) Send(to, _, _, _ string) error {
	m.sent = append(m.sent, to)
	return nil
}

// Évite l'avertissement "unused import" sur fmt.
var _ = fmt.Sprintf
