package services

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"

	"github.com/webdad/auth-service/internal/eraser"
	"github.com/webdad/auth-service/internal/models"
)

type flowMailer struct {
	to  string
	err error
}

func (m *flowMailer) Send(to, _, _, _ string) error { m.to = to; return m.err }

func qrow(cols []string, values ...driver.Value) *fakeQ {
	return &fakeQ{cols: cols, rows: [][]driver.Value{values}}
}

func qerr(err error) *fakeQ { return &fakeQ{err: err} }

func accountRow(id, email, password string, active, verified, mustChange, mfa bool) *fakeQ {
	return qrow(
		[]string{"id", "email", "password", "role", "is_active", "email_verified", "must_change_password", "mfa_enabled", "created_at"},
		id, email, password, models.RoleUser, active, verified, mustChange, mfa, time.Now(),
	)
}

func issuedTokenQueries() []*fakeQ {
	return []*fakeQ{
		qrow([]string{"terms_accepted_version"}, int64(models.CurrentTermsVersion)),
		{},
	}
}

func appendIssued(qs ...*fakeQ) []*fakeQ { return append(qs, issuedTokenQueries()...) }

func TestRegisterAndAdminCreateUserFlows(t *testing.T) {
	t.Run("register success", func(t *testing.T) {
		db := openFakeDB(t)
		gFakeDrv.queries = appendIssued(qrow(
			[]string{"id", "email", "role", "is_active", "created_at"},
			"u1", "a@example.test", models.RoleUser, true, time.Now(),
		))
		s := newSvc(t, db)
		access, refresh, user, err := s.Register("a@example.test", "correct horse battery staple")
		if err != nil || access == "" || refresh == "" || user.ID != "u1" {
			t.Fatalf("Register() = %q, %q, %#v, %v", access, refresh, user, err)
		}
	})

	for name, dbErr := range map[string]struct {
		err  error
		want error
	}{
		"duplicate": {&fakePGErr{code: "23505"}, ErrEmailTaken},
		"database":  {errors.New("insert failed"), nil},
	} {
		t.Run("register "+name, func(t *testing.T) {
			db := openFakeDB(t)
			gFakeDrv.queries = []*fakeQ{qerr(dbErr.err)}
			_, _, _, err := newSvc(t, db).Register("a@example.test", "password")
			if dbErr.want != nil && !errors.Is(err, dbErr.want) {
				t.Fatalf("got %v, want %v", err, dbErr.want)
			}
			if dbErr.want == nil && err == nil {
				t.Fatal("expected wrapped database error")
			}
		})
	}

	t.Run("admin create success", func(t *testing.T) {
		db := openFakeDB(t)
		gFakeDrv.queries = []*fakeQ{qrow(
			[]string{"id", "email", "role", "is_active", "email_verified", "must_change_password", "created_at"},
			"u2", "new@example.test", models.RoleUser, true, true, true, time.Now(),
		)}
		s, _ := New(db, "secret", time.Minute, time.Hour, nil, "", "", true, "")
		u, err := s.AdminCreateUser("new@example.test", "temporary-password", "New User")
		if err != nil || !u.EmailVerified || !u.MustChangePassword {
			t.Fatalf("AdminCreateUser() = %#v, %v", u, err)
		}
	})
}

func TestPasswordTermsAndLoginFlows(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("old-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("change password success", func(t *testing.T) {
		db := openFakeDB(t)
		gFakeDrv.queries = appendIssued(
			qrow([]string{"id", "email", "password", "role", "is_active", "email_verified", "must_change_password", "created_at"},
				"u1", "a@example.test", string(hash), models.RoleUser, true, true, true, time.Now()),
			&fakeQ{}, &fakeQ{},
		)
		access, refresh, u, err := newSvc(t, db).ChangePassword("u1", "old-password", "new-password")
		if err != nil || access == "" || refresh == "" || u.MustChangePassword {
			t.Fatalf("ChangePassword() = %q, %q, %#v, %v", access, refresh, u, err)
		}
	})

	for name, row := range map[string]struct {
		row  *fakeQ
		want error
	}{
		"missing":        {&fakeQ{}, ErrUserNotFound},
		"oauth account":  {qrow([]string{"id", "email", "password", "role", "is_active", "email_verified", "must_change_password", "created_at"}, "u", "a@b.c", nil, models.RoleUser, true, true, false, time.Now()), ErrInvalidCurrentPassword},
		"wrong password": {qrow([]string{"id", "email", "password", "role", "is_active", "email_verified", "must_change_password", "created_at"}, "u", "a@b.c", string(hash), models.RoleUser, true, true, false, time.Now()), ErrInvalidCurrentPassword},
	} {
		t.Run("change password "+name, func(t *testing.T) {
			db := openFakeDB(t)
			gFakeDrv.queries = []*fakeQ{row.row}
			_, _, _, err := newSvc(t, db).ChangePassword("u", "bad", "new")
			if !errors.Is(err, row.want) {
				t.Fatalf("got %v, want %v", err, row.want)
			}
		})
	}

	t.Run("accept terms success", func(t *testing.T) {
		db := openFakeDB(t)
		gFakeDrv.queries = appendIssued(qrow(
			[]string{"id", "email", "role", "is_active", "email_verified", "must_change_password", "created_at"},
			"u1", "a@example.test", models.RoleUser, true, true, false, time.Now(),
		))
		access, refresh, u, err := newSvc(t, db).AcceptTerms("u1")
		if err != nil || access == "" || refresh == "" || u.ID != "u1" {
			t.Fatalf("AcceptTerms: %#v %v", u, err)
		}
	})

	t.Run("login variants", func(t *testing.T) {
		cases := []struct {
			name             string
			active, verified bool
			password         driver.Value
			supplied         string
			want             error
		}{
			{"inactive", false, true, string(hash), "old-password", ErrUserInactive},
			{"oauth", true, true, nil, "old-password", ErrNoLocalPassword},
			{"wrong password", true, true, string(hash), "wrong", ErrInvalidCredentials},
			{"unverified", true, false, string(hash), "old-password", ErrEmailNotVerified},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				db := openFakeDB(t)
				gFakeDrv.queries = []*fakeQ{accountRow("u", "a@example.test", "", tc.active, tc.verified, false, false)}
				gFakeDrv.queries[0].rows[0][2] = tc.password
				_, err := newSvc(t, db).Login("a@example.test", tc.supplied)
				if !errors.Is(err, tc.want) {
					t.Fatalf("got %v, want %v", err, tc.want)
				}
			})
		}
	})

	t.Run("login success", func(t *testing.T) {
		db := openFakeDB(t)
		gFakeDrv.queries = appendIssued(accountRow("u", "a@example.test", string(hash), true, true, false, false))
		out, err := newSvc(t, db).LoginByUserID("u", "old-password")
		if err != nil || out.Token == "" || out.Refresh == "" {
			t.Fatalf("LoginByUserID: %#v %v", out, err)
		}
	})
}

func TestTokenRefreshVerificationAndAdministrationFlows(t *testing.T) {
	t.Run("refresh success expired inactive", func(t *testing.T) {
		row := func(active bool, expires time.Time) *fakeQ {
			return qrow([]string{"id", "email", "role", "is_active", "must_change_password", "email_verified", "created_at", "expires_at"},
				"u", "a@example.test", models.RoleUser, active, false, true, time.Now(), expires)
		}
		db := openFakeDB(t)
		gFakeDrv.queries = appendIssued(row(true, time.Now().Add(time.Hour)), &fakeQ{})
		a, r, _, err := newSvc(t, db).Refresh("refresh")
		if err != nil || a == "" || r == "" {
			t.Fatalf("Refresh success: %v", err)
		}

		db = openFakeDB(t)
		gFakeDrv.queries = []*fakeQ{row(true, time.Now().Add(-time.Hour)), {}}
		_, _, _, err = newSvc(t, db).Refresh("expired")
		if !errors.Is(err, ErrInvalidRefreshToken) {
			t.Fatalf("expired: %v", err)
		}

		db = openFakeDB(t)
		gFakeDrv.queries = []*fakeQ{row(false, time.Now().Add(time.Hour))}
		_, _, _, err = newSvc(t, db).Refresh("inactive")
		if !errors.Is(err, ErrUserInactive) {
			t.Fatalf("inactive: %v", err)
		}
	})

	t.Run("account token lifecycle and verify", func(t *testing.T) {
		db := openFakeDB(t)
		s := newSvc(t, db)
		gFakeDrv.queries = []*fakeQ{{}, {}}
		raw, err := s.createAccountToken("u", purposeVerify, time.Hour)
		if err != nil || raw == "" {
			t.Fatalf("createAccountToken: %v", err)
		}

		gFakeDrv.queries = []*fakeQ{qrow([]string{"id", "user_id", "expires_at", "used_at"}, "t", "u", time.Now().Add(time.Hour), nil), {}}
		uid, err := s.consumeAccountToken(raw, purposeVerify)
		if err != nil || uid != "u" {
			t.Fatalf("consumeAccountToken: %q %v", uid, err)
		}

		gFakeDrv.queries = appendIssued(
			qrow([]string{"id", "user_id", "expires_at", "used_at"}, "t", "u", time.Now().Add(time.Hour), nil), &fakeQ{},
			qrow([]string{"id", "email", "role", "is_active", "email_verified", "must_change_password", "created_at"}, "u", "a@example.test", models.RoleUser, true, true, false, time.Now()),
		)
		a, r, _, err := s.VerifyEmail(raw)
		if err != nil || a == "" || r == "" {
			t.Fatalf("VerifyEmail: %v", err)
		}
	})

	t.Run("admin repository operations", func(t *testing.T) {
		db := openFakeDB(t)
		s := newSvc(t, db)
		gFakeDrv.queries = []*fakeQ{qrow([]string{"id", "email", "role", "is_active", "deactivated_at", "created_at"}, "u1", "a@example.test", models.RoleUser, true, nil, time.Now())}
		users, err := s.ListUsers(10, 0, "a")
		if err != nil || len(users) != 1 {
			t.Fatalf("ListUsers: %#v %v", users, err)
		}

		gFakeDrv.queries = []*fakeQ{{}}
		if err := s.SetRole("u1", models.RoleModerator); err != nil {
			t.Fatal(err)
		}
		if err := s.SetRole("u1", "root"); !errors.Is(err, ErrInvalidRole) {
			t.Fatalf("SetRole invalid: %v", err)
		}

		gFakeDrv.queries = []*fakeQ{qrow([]string{"role"}, models.RoleModerator)}
		if role, err := s.RoleOf("u1"); err != nil || role != models.RoleModerator {
			t.Fatalf("RoleOf: %q %v", role, err)
		}

		gFakeDrv.queries = []*fakeQ{{}, {}}
		if err := s.SetActive("u1", false); err != nil {
			t.Fatal(err)
		}
		gFakeDrv.queries = []*fakeQ{{}, {}}
		if err := s.DeleteAccount("u1"); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("sweeper guards", func(t *testing.T) {
		db := openFakeDB(t)
		s := newSvc(t, db)
		if n, err := s.SweepBannedAccounts(context.Background(), nil, 0); err != nil || n != 0 {
			t.Fatalf("Sweep guard: %d %v", n, err)
		}
		s.RunAccountPurgeSweeper(context.Background(), nil, 0, time.Second)
	})
}

func TestMFAHappyPathsAndStatus(t *testing.T) {
	db := openFakeDB(t)
	s, err := New(db, "secret", time.Minute, time.Hour, nil, "", "", false, validKey(t))
	if err != nil {
		t.Fatal(err)
	}

	gFakeDrv.queries = []*fakeQ{qrow([]string{"email", "mfa_enabled"}, "a@example.test", false), {}}
	setup, err := s.SetupMFA("u")
	if err != nil || setup.Secret == "" || setup.QRDataURI == "" {
		t.Fatalf("SetupMFA: %#v %v", setup, err)
	}

	enc, err := s.mfaCipher.encrypt(setup.Secret)
	if err != nil {
		t.Fatal(err)
	}
	code, err := totp.GenerateCode(setup.Secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	gFakeDrv.queries = []*fakeQ{qrow([]string{"mfa_secret", "mfa_enabled"}, enc, false), {}}
	if err := s.EnableMFA("u", code); err != nil {
		t.Fatalf("EnableMFA: %v", err)
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
	gFakeDrv.queries = []*fakeQ{qrow([]string{"mfa_secret", "mfa_enabled", "password"}, enc, true, string(hash)), {}}
	if err := s.DisableMFA("u", "", "password"); err != nil {
		t.Fatalf("DisableMFA: %v", err)
	}

	gFakeDrv.queries = []*fakeQ{qrow([]string{"mfa_enabled"}, true)}
	if enabled, err := s.MFAStatus("u"); err != nil || !enabled {
		t.Fatalf("MFAStatus: %v %v", enabled, err)
	}

	tracker := newMFAAttemptTracker()
	if tracker.incr("x") != 1 || tracker.incr("x") != 2 {
		t.Fatal("attempt counter")
	}
	tracker.clear("x")
	if tracker.incr("x") != 1 {
		t.Fatal("attempt counter was not cleared")
	}
}

func TestEmailChangeAndMailFlows(t *testing.T) {
	t.Run("request success", func(t *testing.T) {
		db := openFakeDB(t)
		mailer := &flowMailer{}
		s, _ := New(db, "secret", time.Minute, time.Hour, mailer, "https://app.test/", "", false, "")
		gFakeDrv.queries = []*fakeQ{
			qrow([]string{"email", "is_active"}, "old@example.test", true),
			qrow([]string{"exists"}, false),
			&fakeQ{}, &fakeQ{}, &fakeQ{},
		}
		if err := s.RequestEmailChange("u", " NEW@Example.Test "); err != nil {
			t.Fatal(err)
		}
		if mailer.to != "new@example.test" {
			t.Fatalf("mail sent to %q", mailer.to)
		}
	})

	t.Run("request guards", func(t *testing.T) {
		cases := []struct {
			name    string
			queries []*fakeQ
			want    error
		}{
			{"inactive", []*fakeQ{qrow([]string{"email", "is_active"}, "old@example.test", false)}, ErrUserInactive},
			{"same", []*fakeQ{qrow([]string{"email", "is_active"}, "old@example.test", true)}, ErrSameEmail},
			{"taken", []*fakeQ{qrow([]string{"email", "is_active"}, "old@example.test", true), qrow([]string{"exists"}, true)}, ErrEmailTaken},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				db := openFakeDB(t)
				gFakeDrv.queries = tc.queries
				newEmail := "new@example.test"
				if tc.name == "same" {
					newEmail = "OLD@example.test"
				}
				if err := newSvc(t, db).RequestEmailChange("u", newEmail); !errors.Is(err, tc.want) {
					t.Fatalf("got %v want %v", err, tc.want)
				}
			})
		}
	})

	t.Run("mail failure is compensated", func(t *testing.T) {
		db := openFakeDB(t)
		s, _ := New(db, "secret", time.Minute, time.Hour, &flowMailer{err: errors.New("smtp down")}, "", "", false, "")
		gFakeDrv.queries = []*fakeQ{
			qrow([]string{"email", "is_active"}, "old@example.test", true), qrow([]string{"exists"}, false),
			{}, {}, {}, {}, {},
		}
		if err := s.RequestEmailChange("u", "new@example.test"); !errors.Is(err, ErrEmailDelivery) {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("confirm success and invalid", func(t *testing.T) {
		db := openFakeDB(t)
		s := newSvc(t, db)
		gFakeDrv.queries = appendIssued(
			qrow([]string{"id", "user_id", "expires_at", "used_at"}, "tok", "u", time.Now().Add(time.Hour), nil),
			qrow([]string{"id", "email", "role", "is_active", "email_verified", "must_change_password", "created_at"}, "u", "new@example.test", models.RoleUser, true, true, false, time.Now()),
			&fakeQ{}, &fakeQ{}, &fakeQ{},
		)
		a, r, u, err := s.ConfirmEmailChange("raw")
		if err != nil || a == "" || r == "" || u.Email != "new@example.test" {
			t.Fatalf("ConfirmEmailChange: %#v %v", u, err)
		}

		db = openFakeDB(t)
		gFakeDrv.queries = []*fakeQ{{}}
		_, _, _, err = newSvc(t, db).ConfirmEmailChange("bad")
		if !errors.Is(err, ErrInvalidToken) {
			t.Fatalf("invalid token: %v", err)
		}
	})

	t.Run("transaction helpers and mail variants", func(t *testing.T) {
		db := openFakeDB(t)
		s := newSvc(t, db)
		gFakeDrv.queries = []*fakeQ{{}}
		if err := s.clearPendingEmail("u", "a@b.c"); err != nil {
			t.Fatal(err)
		}
		gFakeDrv.queries = []*fakeQ{{}, {}}
		if err := s.cancelEmailChangeRequest("u", "a@b.c", "raw"); err != nil {
			t.Fatal(err)
		}

		mailer := &flowMailer{}
		s, _ = New(db, "secret", time.Minute, time.Hour, mailer, "https://app.test", "", false, "")
		s.sendAdminWelcomeMail(&models.User{ID: "u", Email: "a@b.c", EmailVerified: true}, "", "temporary")
		if mailer.to != "a@b.c" {
			t.Fatalf("welcome mail to %q", mailer.to)
		}
		gFakeDrv.queries = []*fakeQ{{}, {}}
		s.sendAdminWelcomeMail(&models.User{ID: "u", Email: "a@b.c"}, "Alice", "temporary")
		s.sendVerificationMail(&models.User{ID: "u", Email: "a@b.c"})
		gFakeDrv.queries = []*fakeQ{{}, {}}
		s.sendResetMail(&models.User{ID: "u", Email: "a@b.c"})
	})
}

func TestOAuthAndResetFlows(t *testing.T) {
	t.Run("oauth existing", func(t *testing.T) {
		db := openFakeDB(t)
		s := newSvc(t, db)
		gFakeDrv.queries = appendIssued(
			qrow([]string{"id", "email", "role", "is_active", "must_change_password", "created_at", "provider"}, "u", "a@example.test", models.RoleUser, true, false, time.Now(), "google"),
			&fakeQ{},
		)
		out, err := s.LoginWithOAuth("google", "subject", "a@example.test")
		if err != nil || out.Token == "" || out.RefreshToken == "" {
			t.Fatalf("LoginWithOAuth: %#v %v", out, err)
		}
	})

	t.Run("oauth onboarding", func(t *testing.T) {
		db := openFakeDB(t)
		s := newSvc(t, db)
		gFakeDrv.queries = []*fakeQ{{}, {}, {}}
		out, err := s.LoginWithOAuth("google", "subject", "new@example.test")
		if err != nil || !out.OnboardingRequired || out.PendingToken == "" {
			t.Fatalf("onboarding: %#v %v", out, err)
		}
	})

	t.Run("complete oauth signup", func(t *testing.T) {
		db := openFakeDB(t)
		s := newSvc(t, db)
		gFakeDrv.queries = appendIssued(
			qrow([]string{"id", "provider", "provider_subject", "email", "expires_at", "used_at"}, "tok", "google", "sub", "new@example.test", time.Now().Add(time.Hour), nil),
			&fakeQ{},
			qrow([]string{"id", "email", "role", "is_active", "email_verified", "must_change_password", "created_at", "provider"}, "u", "new@example.test", models.RoleUser, true, true, false, time.Now(), "google"),
			&fakeQ{},
		)
		a, r, u, err := s.CompleteOAuthSignup("google", "pending")
		if err != nil || a == "" || r == "" || u.Provider != "google" {
			t.Fatalf("CompleteOAuthSignup: %#v %v", u, err)
		}
	})

	t.Run("reset password success", func(t *testing.T) {
		db := openFakeDB(t)
		s := newSvc(t, db)
		gFakeDrv.queries = []*fakeQ{
			qrow([]string{"id", "user_id", "expires_at", "used_at"}, "tok", "u", time.Now().Add(time.Hour), nil),
			{}, {}, {},
		}
		if err := s.ResetPassword("raw", "new-password"); err != nil {
			t.Fatal(err)
		}
	})
}

func TestMFAChallengeAndBannedListing(t *testing.T) {
	db := openFakeDB(t)
	s, err := New(db, "secret", time.Minute, time.Hour, nil, "", "", false, validKey(t))
	if err != nil {
		t.Fatal(err)
	}
	secret := "JBSWY3DPEHPK3PXP"
	enc, _ := s.mfaCipher.encrypt(secret)
	code, _ := totp.GenerateCode(secret, time.Now())
	gFakeDrv.queries = appendIssued(
		qrow([]string{"id", "user_id", "expires_at", "used_at"}, "tok", "u", time.Now().Add(time.Hour), nil),
		qrow([]string{"id", "email", "role", "is_active", "email_verified", "must_change_password", "mfa_enabled", "created_at", "mfa_secret"}, "u", "a@example.test", models.RoleUser, true, true, false, true, time.Now(), enc),
		&fakeQ{},
	)
	a, r, _, err := s.VerifyMFA("challenge", code)
	if err != nil || a == "" || r == "" {
		t.Fatalf("VerifyMFA: %v", err)
	}

	gFakeDrv.queries = []*fakeQ{qrow([]string{"id", "email", "role", "is_active", "deactivated_at", "created_at"}, "u", "a@example.test", models.RoleUser, false, time.Now().Add(-time.Hour), time.Now())}
	users, err := s.ListBannedBefore(time.Now(), 100)
	if err != nil || len(users) != 1 {
		t.Fatalf("ListBannedBefore: %#v %v", users, err)
	}
}

func TestServiceDatabaseFailurePaths(t *testing.T) {
	dbFailure := errors.New("database unavailable")

	t.Run("simple operations", func(t *testing.T) {
		tests := []struct {
			name string
			call func(*AuthService) error
		}{
			{"admin create", func(s *AuthService) error { _, err := s.AdminCreateUser("a@b.c", "password", "a"); return err }},
			{"accept terms", func(s *AuthService) error { _, _, _, err := s.AcceptTerms("u"); return err }},
			{"logout", func(s *AuthService) error { return s.Logout("token") }},
			{"seed admin", func(s *AuthService) error { return s.EnsureDefaultAdmin("a@b.c", "password") }},
			{"list users", func(s *AuthService) error { _, err := s.ListUsers(1, 0, ""); return err }},
			{"role", func(s *AuthService) error { _, err := s.RoleOf("u"); return err }},
			{"active", func(s *AuthService) error { return s.SetActive("u", true) }},
			{"list banned", func(s *AuthService) error { _, err := s.ListBannedBefore(time.Now(), 1); return err }},
			{"mfa status", func(s *AuthService) error { _, err := s.MFAStatus("u"); return err }},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				db := openFakeDB(t)
				gFakeDrv.queries = []*fakeQ{qerr(dbFailure)}
				if err := tc.call(newSvc(t, db)); err == nil {
					t.Fatal("expected error")
				}
			})
		}
	})

	t.Run("token helper failures", func(t *testing.T) {
		db := openFakeDB(t)
		s := newSvc(t, db)
		gFakeDrv.queries = []*fakeQ{qerr(dbFailure)}
		if _, err := s.createAccountToken("u", purposeReset, time.Hour); err == nil {
			t.Fatal("createAccountToken delete")
		}
		gFakeDrv.queries = []*fakeQ{{}, qerr(dbFailure)}
		if _, err := s.createAccountToken("u", purposeReset, time.Hour); err == nil {
			t.Fatal("createAccountToken insert")
		}
		gFakeDrv.queries = []*fakeQ{qerr(dbFailure)}
		if _, err := s.consumeAccountToken("raw", purposeReset); err == nil {
			t.Fatal("consume query")
		}
		gFakeDrv.queries = []*fakeQ{qrow([]string{"id", "user_id", "expires_at", "used_at"}, "t", "u", time.Now().Add(-time.Hour), nil)}
		if _, err := s.consumeAccountToken("raw", purposeReset); !errors.Is(err, ErrInvalidToken) {
			t.Fatalf("expired: %v", err)
		}
		gFakeDrv.queries = []*fakeQ{qrow([]string{"id", "user_id", "expires_at", "used_at"}, "t", "u", time.Now().Add(time.Hour), time.Now())}
		if _, err := s.consumeAccountToken("raw", purposeReset); !errors.Is(err, ErrInvalidToken) {
			t.Fatalf("used: %v", err)
		}
		gFakeDrv.queries = []*fakeQ{qrow([]string{"id", "user_id", "expires_at", "used_at"}, "t", "u", time.Now().Add(time.Hour), nil), qerr(dbFailure)}
		if _, err := s.consumeAccountToken("raw", purposeReset); err == nil {
			t.Fatal("consume update")
		}
		gFakeDrv.queries = []*fakeQ{qerr(dbFailure)}
		if _, err := s.createOAuthSignupToken("google", "sub", "a@b.c"); err == nil {
			t.Fatal("oauth delete")
		}
		gFakeDrv.queries = []*fakeQ{{}, qerr(dbFailure)}
		if _, err := s.createOAuthSignupToken("google", "sub", "a@b.c"); err == nil {
			t.Fatal("oauth insert")
		}
		gFakeDrv.queries = []*fakeQ{qerr(dbFailure)}
		if _, err := s.createRefreshToken("u"); err == nil {
			t.Fatal("refresh insert")
		}
	})

	t.Run("login and refresh database failures", func(t *testing.T) {
		db := openFakeDB(t)
		gFakeDrv.queries = []*fakeQ{qerr(dbFailure)}
		if _, err := newSvc(t, db).Login("a@b.c", "x"); err == nil {
			t.Fatal("login query")
		}
		db = openFakeDB(t)
		gFakeDrv.queries = []*fakeQ{qerr(dbFailure)}
		if _, _, _, err := newSvc(t, db).Refresh("raw"); err == nil {
			t.Fatal("refresh query")
		}
		db = openFakeDB(t)
		gFakeDrv.queries = []*fakeQ{qrow([]string{"id", "email", "role", "is_active", "must_change_password", "email_verified", "created_at", "expires_at"}, "u", "a@b.c", models.RoleUser, true, false, true, time.Now(), time.Now().Add(time.Hour)), qerr(dbFailure)}
		if _, _, _, err := newSvc(t, db).Refresh("raw"); err == nil {
			t.Fatal("refresh revoke")
		}
	})

	t.Run("verify failures", func(t *testing.T) {
		db := openFakeDB(t)
		s := newSvc(t, db)
		gFakeDrv.queries = []*fakeQ{qrow([]string{"id", "user_id", "expires_at", "used_at"}, "t", "u", time.Now().Add(time.Hour), nil), {}, qerr(dbFailure)}
		if _, _, _, err := s.VerifyEmail("raw"); err == nil {
			t.Fatal("verify update")
		}
		gFakeDrv.queries = []*fakeQ{qrow([]string{"id", "user_id", "expires_at", "used_at"}, "t", "u", time.Now().Add(time.Hour), nil), {}, qrow([]string{"id", "email", "role", "is_active", "email_verified", "must_change_password", "created_at"}, "u", "a@b.c", models.RoleUser, false, true, false, time.Now())}
		if _, _, _, err := s.VerifyEmail("raw"); !errors.Is(err, ErrUserInactive) {
			t.Fatalf("inactive: %v", err)
		}
	})
}

func TestMFAGuardsAndFailures(t *testing.T) {
	dbFailure := errors.New("db failure")
	newMFA := func(t *testing.T) *AuthService {
		t.Helper()
		db := openFakeDB(t)
		s, err := New(db, "secret", time.Minute, time.Hour, nil, "", "", false, validKey(t))
		if err != nil {
			t.Fatal(err)
		}
		return s
	}

	t.Run("setup guards", func(t *testing.T) {
		s := newMFA(t)
		for _, tc := range []struct {
			q    *fakeQ
			want error
		}{
			{&fakeQ{}, ErrUserNotFound}, {qerr(dbFailure), dbFailure},
			{qrow([]string{"email", "mfa_enabled"}, "a@b.c", true), ErrMFAAlreadyEnabled},
		} {
			gFakeDrv.queries = []*fakeQ{tc.q}
			_, err := s.SetupMFA("u")
			if tc.want == dbFailure {
				if err == nil {
					t.Fatal("expected db error")
				}
			} else if !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
		}
		gFakeDrv.queries = []*fakeQ{qrow([]string{"email", "mfa_enabled"}, "a@b.c", false), qerr(dbFailure)}
		if _, err := s.SetupMFA("u"); err == nil {
			t.Fatal("expected update error")
		}
	})

	t.Run("enable guards", func(t *testing.T) {
		s := newMFA(t)
		for _, tc := range []struct {
			q    *fakeQ
			code string
			want error
		}{
			{&fakeQ{}, "", ErrUserNotFound},
			{qrow([]string{"mfa_secret", "mfa_enabled"}, nil, true), "", ErrMFAAlreadyEnabled},
			{qrow([]string{"mfa_secret", "mfa_enabled"}, nil, false), "", ErrMFANotPending},
			{qrow([]string{"mfa_secret", "mfa_enabled"}, "not-base64", false), "", nil},
		} {
			gFakeDrv.queries = []*fakeQ{tc.q}
			err := s.EnableMFA("u", tc.code)
			if tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
			if tc.want == nil && err == nil {
				t.Fatal("expected decrypt error")
			}
		}
		enc, _ := s.mfaCipher.encrypt("JBSWY3DPEHPK3PXP")
		gFakeDrv.queries = []*fakeQ{qrow([]string{"mfa_secret", "mfa_enabled"}, enc, false)}
		if err := s.EnableMFA("u", "000000"); !errors.Is(err, ErrInvalidMFACode) {
			t.Fatalf("invalid code: %v", err)
		}
	})

	t.Run("disable guards", func(t *testing.T) {
		s := newMFA(t)
		gFakeDrv.queries = []*fakeQ{{}}
		if err := s.DisableMFA("u", "", ""); !errors.Is(err, ErrUserNotFound) {
			t.Fatalf("missing: %v", err)
		}
		gFakeDrv.queries = []*fakeQ{qrow([]string{"mfa_secret", "mfa_enabled", "password"}, nil, false, nil)}
		if err := s.DisableMFA("u", "", ""); !errors.Is(err, ErrMFANotEnabled) {
			t.Fatalf("disabled: %v", err)
		}
		gFakeDrv.queries = []*fakeQ{qrow([]string{"mfa_secret", "mfa_enabled", "password"}, nil, true, nil)}
		if err := s.DisableMFA("u", "", ""); !errors.Is(err, ErrInvalidMFACode) {
			t.Fatalf("invalid: %v", err)
		}
	})

	t.Run("challenge guards and attempts", func(t *testing.T) {
		s := newMFA(t)
		gFakeDrv.queries = []*fakeQ{{}}
		if _, _, _, err := s.VerifyMFA("bad", "0"); !errors.Is(err, ErrInvalidToken) {
			t.Fatalf("missing: %v", err)
		}
		enc, _ := s.mfaCipher.encrypt("JBSWY3DPEHPK3PXP")
		for i := 0; i < maxMFAVerifyAttempts; i++ {
			gFakeDrv.queries = []*fakeQ{
				qrow([]string{"id", "user_id", "expires_at", "used_at"}, "t", "u", time.Now().Add(time.Hour), nil),
				qrow([]string{"id", "email", "role", "is_active", "email_verified", "must_change_password", "mfa_enabled", "created_at", "mfa_secret"}, "u", "a@b.c", models.RoleUser, true, true, false, true, time.Now(), enc),
			}
			if i == maxMFAVerifyAttempts-1 {
				gFakeDrv.queries = append(gFakeDrv.queries, &fakeQ{})
			}
			if _, _, _, err := s.VerifyMFA("challenge", "000000"); !errors.Is(err, ErrInvalidMFACode) {
				t.Fatalf("attempt %d: %v", i, err)
			}
		}
	})
}

func TestSweeperEmptyAndCancelled(t *testing.T) {
	db := openFakeDB(t)
	s := newSvc(t, db)
	gFakeDrv.queries = []*fakeQ{{}}
	if n, err := s.SweepBannedAccounts(context.Background(), nil, time.Hour); err != nil || n != 0 {
		t.Fatalf("Sweep empty: %d %v", n, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	gFakeDrv.queries = []*fakeQ{{}}
	s.RunAccountPurgeSweeper(ctx, nil, time.Hour, time.Millisecond)
}

func TestSweeperPurgesAndContinuesAfterDeleteFailure(t *testing.T) {
	db := openFakeDB(t)
	s := newSvc(t, db)
	rows := &fakeQ{cols: []string{"id", "email", "role", "is_active", "deactivated_at", "created_at"}, rows: [][]driver.Value{
		{"u1", "one@example.test", models.RoleUser, false, time.Now().Add(-2 * time.Hour), time.Now()},
		{"u2", "two@example.test", models.RoleUser, false, time.Now().Add(-2 * time.Hour), time.Now()},
	}}
	gFakeDrv.queries = []*fakeQ{rows, {}, {}, {}, qerr(errors.New("delete failed"))}
	n, err := s.SweepBannedAccounts(context.Background(), eraser.New(eraser.Targets{}), time.Hour)
	if err != nil || n != 1 {
		t.Fatalf("SweepBannedAccounts = %d, %v", n, err)
	}
}

func TestAdditionalBranchCoverage(t *testing.T) {
	dbFailure := errors.New("db failure")

	t.Run("login creates MFA challenge", func(t *testing.T) {
		hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
		db := openFakeDB(t)
		gFakeDrv.queries = []*fakeQ{accountRow("u", "a@b.c", string(hash), true, true, false, true), {}, {}}
		out, err := newSvc(t, db).Login("a@b.c", "password")
		if err != nil || !out.MFARequired || out.Challenge == "" {
			t.Fatalf("MFA login: %#v %v", out, err)
		}
	})

	t.Run("oauth guards", func(t *testing.T) {
		cases := []struct {
			name  string
			query *fakeQ
			extra []*fakeQ
			want  error
		}{
			{"query", qerr(dbFailure), nil, nil},
			{"inactive", qrow([]string{"id", "email", "role", "is_active", "must_change_password", "created_at", "provider"}, "u", "a@b.c", models.RoleUser, false, false, time.Now(), "google"), nil, ErrUserInactive},
			{"link", qrow([]string{"id", "email", "role", "is_active", "must_change_password", "created_at", "provider"}, "u", "a@b.c", models.RoleUser, true, false, time.Now(), "google"), []*fakeQ{qerr(dbFailure)}, nil},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				db := openFakeDB(t)
				gFakeDrv.queries = append([]*fakeQ{tc.query}, tc.extra...)
				_, err := newSvc(t, db).LoginWithOAuth("google", "sub", "a@b.c")
				if tc.want != nil && !errors.Is(err, tc.want) {
					t.Fatalf("got %v want %v", err, tc.want)
				}
				if tc.want == nil && err == nil {
					t.Fatal("expected error")
				}
			})
		}
	})

	t.Run("complete oauth invalid states", func(t *testing.T) {
		cases := []struct {
			name    string
			queries []*fakeQ
			want    error
		}{
			{"missing", []*fakeQ{{}}, ErrInvalidToken},
			{"provider mismatch", []*fakeQ{qrow([]string{"id", "provider", "provider_subject", "email", "expires_at", "used_at"}, "t", "github", "sub", "a@b.c", time.Now().Add(time.Hour), nil)}, ErrInvalidToken},
			{"expired", []*fakeQ{qrow([]string{"id", "provider", "provider_subject", "email", "expires_at", "used_at"}, "t", "google", "sub", "a@b.c", time.Now().Add(-time.Hour), nil)}, ErrInvalidToken},
			{"exists", []*fakeQ{qrow([]string{"id", "provider", "provider_subject", "email", "expires_at", "used_at"}, "t", "google", "sub", "a@b.c", time.Now().Add(time.Hour), nil), qrow([]string{"one"}, 1)}, ErrOAuthAccountExists},
			{"exists query", []*fakeQ{qrow([]string{"id", "provider", "provider_subject", "email", "expires_at", "used_at"}, "t", "google", "sub", "a@b.c", time.Now().Add(time.Hour), nil), qerr(dbFailure)}, nil},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				db := openFakeDB(t)
				gFakeDrv.queries = tc.queries
				_, _, _, err := newSvc(t, db).CompleteOAuthSignup("google", "raw")
				if tc.want != nil && !errors.Is(err, tc.want) {
					t.Fatalf("got %v want %v", err, tc.want)
				}
				if tc.want == nil && err == nil {
					t.Fatal("expected error")
				}
			})
		}
	})

	t.Run("confirm email invalid states", func(t *testing.T) {
		baseToken := func(exp time.Time, used driver.Value) *fakeQ {
			return qrow([]string{"id", "user_id", "expires_at", "used_at"}, "t", "u", exp, used)
		}
		cases := []struct {
			name    string
			queries []*fakeQ
			want    error
		}{
			{"expired", []*fakeQ{baseToken(time.Now().Add(-time.Hour), nil)}, ErrInvalidToken},
			{"used", []*fakeQ{baseToken(time.Now().Add(time.Hour), time.Now())}, ErrInvalidToken},
			{"no pending", []*fakeQ{baseToken(time.Now().Add(time.Hour), nil), {}}, ErrInvalidToken},
			{"duplicate", []*fakeQ{baseToken(time.Now().Add(time.Hour), nil), qerr(&fakePGErr{code: "23505"})}, ErrEmailTaken},
			{"inactive", []*fakeQ{baseToken(time.Now().Add(time.Hour), nil), qrow([]string{"id", "email", "role", "is_active", "email_verified", "must_change_password", "created_at"}, "u", "a@b.c", models.RoleUser, false, true, false, time.Now())}, ErrUserInactive},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				db := openFakeDB(t)
				gFakeDrv.queries = tc.queries
				_, _, _, err := newSvc(t, db).ConfirmEmailChange("raw")
				if !errors.Is(err, tc.want) {
					t.Fatalf("got %v want %v", err, tc.want)
				}
			})
		}
	})

	t.Run("request email database errors", func(t *testing.T) {
		cases := [][]*fakeQ{
			{qerr(dbFailure)},
			{qrow([]string{"email", "is_active"}, "old@b.c", true), qerr(dbFailure)},
			{qrow([]string{"email", "is_active"}, "old@b.c", true), qrow([]string{"exists"}, false), qerr(dbFailure)},
			{qrow([]string{"email", "is_active"}, "old@b.c", true), qrow([]string{"exists"}, false), qerr(&fakePGErr{code: "23505"})},
		}
		for i, qs := range cases {
			db := openFakeDB(t)
			gFakeDrv.queries = qs
			if err := newSvc(t, db).RequestEmailChange("u", "new@b.c"); err == nil {
				t.Fatalf("case %d expected error", i)
			}
		}
	})
}

func TestTransactionAndRepositoryErrorCoverage(t *testing.T) {
	dbFailure := errors.New("db failure")

	t.Run("begin failures", func(t *testing.T) {
		tests := []func(*AuthService) error{
			func(s *AuthService) error { return s.cancelEmailChangeRequest("u", "a@b.c", "raw") },
			func(s *AuthService) error { _, _, _, err := s.ConfirmEmailChange("raw"); return err },
			func(s *AuthService) error { _, _, _, err := s.CompleteOAuthSignup("google", "raw"); return err },
			func(s *AuthService) error { _, _, _, err := s.VerifyMFA("raw", "000000"); return err },
		}
		for i, call := range tests {
			db := openFakeDB(t)
			gFakeDrv.beginErr = dbFailure
			s := newSvc(t, db)
			if i == 3 {
				s, _ = New(db, "secret", time.Minute, time.Hour, nil, "", "", false, validKey(t))
			}
			if err := call(s); err == nil {
				t.Fatalf("case %d expected error", i)
			}
		}
	})

	t.Run("commit failures", func(t *testing.T) {
		db := openFakeDB(t)
		gFakeDrv.commitErr = dbFailure
		gFakeDrv.queries = []*fakeQ{{}, {}}
		if err := newSvc(t, db).cancelEmailChangeRequest("u", "a@b.c", "raw"); err == nil {
			t.Fatal("cancel commit")
		}

		db = openFakeDB(t)
		gFakeDrv.commitErr = dbFailure
		gFakeDrv.queries = []*fakeQ{
			qrow([]string{"id", "user_id", "expires_at", "used_at"}, "t", "u", time.Now().Add(time.Hour), nil),
			qrow([]string{"id", "email", "role", "is_active", "email_verified", "must_change_password", "created_at"}, "u", "a@b.c", models.RoleUser, true, true, false, time.Now()), {}, {}, {},
		}
		if _, _, _, err := newSvc(t, db).ConfirmEmailChange("raw"); err == nil {
			t.Fatal("confirm commit")
		}
	})

	t.Run("confirm exec failures", func(t *testing.T) {
		token := qrow([]string{"id", "user_id", "expires_at", "used_at"}, "t", "u", time.Now().Add(time.Hour), nil)
		user := qrow([]string{"id", "email", "role", "is_active", "email_verified", "must_change_password", "created_at"}, "u", "a@b.c", models.RoleUser, true, true, false, time.Now())
		for i := 0; i < 3; i++ {
			db := openFakeDB(t)
			qs := []*fakeQ{token, user}
			for j := 0; j < 3; j++ {
				if j == i {
					qs = append(qs, qerr(dbFailure))
				} else {
					qs = append(qs, &fakeQ{})
				}
			}
			gFakeDrv.queries = qs
			if _, _, _, err := newSvc(t, db).ConfirmEmailChange("raw"); err == nil {
				t.Fatalf("exec case %d", i)
			}
		}
	})

	t.Run("password update failures", func(t *testing.T) {
		hash, _ := bcrypt.GenerateFromPassword([]byte("old"), bcrypt.MinCost)
		row := qrow([]string{"id", "email", "password", "role", "is_active", "email_verified", "must_change_password", "created_at"}, "u", "a@b.c", string(hash), models.RoleUser, true, true, false, time.Now())
		for i := 0; i < 2; i++ {
			db := openFakeDB(t)
			gFakeDrv.queries = []*fakeQ{row}
			if i == 0 {
				gFakeDrv.queries = append(gFakeDrv.queries, qerr(dbFailure))
			} else {
				gFakeDrv.queries = append(gFakeDrv.queries, &fakeQ{}, qerr(dbFailure))
			}
			if _, _, _, err := newSvc(t, db).ChangePassword("u", "old", "new"); err == nil {
				t.Fatalf("case %d", i)
			}
		}
	})

	t.Run("rows affected mappings", func(t *testing.T) {
		db := openFakeDB(t)
		s := newSvc(t, db)
		gFakeDrv.rowsAffected = 0
		gFakeDrv.queries = []*fakeQ{{}}
		if err := s.SetRole("missing", models.RoleUser); !errors.Is(err, ErrUserNotFound) {
			t.Fatalf("zero rows: %v", err)
		}
		db = openFakeDB(t)
		s = newSvc(t, db)
		gFakeDrv.rowsAffectedErr = dbFailure
		gFakeDrv.queries = []*fakeQ{{}}
		if err := s.SetRole("u", models.RoleUser); err == nil {
			t.Fatal("RowsAffected error")
		}
		db = openFakeDB(t)
		s = newSvc(t, db)
		gFakeDrv.queries = []*fakeQ{{}, qerr(dbFailure)}
		if err := s.DeleteAccount("u"); err == nil {
			t.Fatal("delete credentials")
		}
	})

	t.Run("scan errors", func(t *testing.T) {
		db := openFakeDB(t)
		gFakeDrv.queries = []*fakeQ{qrow([]string{"id"}, "u")}
		if _, err := newSvc(t, db).ListUsers(1, 0, ""); err == nil {
			t.Fatal("list scan")
		}
		db = openFakeDB(t)
		gFakeDrv.queries = []*fakeQ{qrow([]string{"id"}, "u")}
		if _, err := newSvc(t, db).ListBannedBefore(time.Now(), 1); err == nil {
			t.Fatal("banned scan")
		}
	})

	t.Run("JWT unexpected algorithm", func(t *testing.T) {
		s := newSvc(t, openFakeDB(t))
		tok := jwt.NewWithClaims(jwt.SigningMethodNone, Claims{})
		raw, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := s.ParseToken(raw); err == nil {
			t.Fatal("none algorithm accepted")
		}
	})
}

func TestRemainingServiceBranches(t *testing.T) {
	dbFailure := errors.New("db failure")

	t.Run("reset and issue token failures", func(t *testing.T) {
		base := func() *fakeQ {
			return qrow([]string{"id", "user_id", "expires_at", "used_at"}, "t", "u", time.Now().Add(time.Hour), nil)
		}
		for i := 0; i < 2; i++ {
			db := openFakeDB(t)
			qs := []*fakeQ{base(), {}}
			if i == 0 {
				qs = append(qs, qerr(dbFailure))
			} else {
				qs = append(qs, &fakeQ{}, qerr(dbFailure))
			}
			gFakeDrv.queries = qs
			if err := newSvc(t, db).ResetPassword("raw", "new-password"); err == nil {
				t.Fatalf("reset case %d", i)
			}
		}
		db := openFakeDB(t)
		gFakeDrv.queries = []*fakeQ{qrow([]string{"terms_accepted_version"}, 0), qerr(dbFailure)}
		if _, _, _, err := newSvc(t, db).issueTokens(&models.User{ID: "u"}); err == nil {
			t.Fatal("issue refresh")
		}
	})

	t.Run("mail helper failures", func(t *testing.T) {
		db := openFakeDB(t)
		s := newSvc(t, db)
		gFakeDrv.queries = []*fakeQ{qerr(dbFailure)}
		if err := s.clearPendingEmail("u", "a@b.c"); err == nil {
			t.Fatal("clear pending")
		}

		mailer := &flowMailer{err: errors.New("smtp")}
		s, _ = New(db, "secret", time.Minute, time.Hour, mailer, "https://app.test", "", false, "")
		gFakeDrv.queries = []*fakeQ{qerr(dbFailure)}
		s.sendVerificationMail(&models.User{ID: "u", Email: "a@b.c"})
		gFakeDrv.queries = []*fakeQ{{}, {}}
		s.sendVerificationMail(&models.User{ID: "u", Email: "a@b.c"})
		gFakeDrv.queries = []*fakeQ{qerr(dbFailure)}
		s.sendResetMail(&models.User{ID: "u", Email: "a@b.c"})
		gFakeDrv.queries = []*fakeQ{{}, {}}
		s.sendResetMail(&models.User{ID: "u", Email: "a@b.c"})
		s.sendAdminWelcomeMail(&models.User{ID: "u", Email: "a@b.c", EmailVerified: true}, "A", "pw")
		gFakeDrv.queries = []*fakeQ{qerr(dbFailure)}
		s.sendAdminWelcomeMail(&models.User{ID: "u", Email: "a@b.c"}, "A", "pw")
		if err := (&AuthService{}).sendEmailChangeMail("u", "a@b.c", "raw"); err == nil {
			t.Fatal("disabled mail")
		}
	})

	t.Run("admin and anti enumeration branches", func(t *testing.T) {
		db := openFakeDB(t)
		gFakeDrv.queries = []*fakeQ{qerr(&fakePGErr{code: "23505"})}
		if _, err := newSvc(t, db).AdminCreateUser("a@b.c", "pw", ""); !errors.Is(err, ErrEmailTaken) {
			t.Fatalf("duplicate: %v", err)
		}
		db = openFakeDB(t)
		gFakeDrv.queries = []*fakeQ{qerr(dbFailure)}
		_ = newSvc(t, db).ResendVerification("a@b.c")
		db = openFakeDB(t)
		gFakeDrv.queries = []*fakeQ{qerr(dbFailure)}
		_ = newSvc(t, db).ForgotPassword("a@b.c")
		db = openFakeDB(t)
		gFakeDrv.queries = []*fakeQ{{}}
		if _, err := newSvc(t, db).RoleOf("missing"); !errors.Is(err, ErrUserNotFound) {
			t.Fatalf("role missing: %v", err)
		}
	})

	t.Run("MFA verify guards", func(t *testing.T) {
		newMFA := func() *AuthService {
			db := openFakeDB(t)
			s, _ := New(db, "secret", time.Minute, time.Hour, nil, "", "", false, validKey(t))
			return s
		}
		token := func(exp time.Time, used driver.Value) *fakeQ {
			return qrow([]string{"id", "user_id", "expires_at", "used_at"}, "t", "u", exp, used)
		}
		user := func(active, enabled bool, secret driver.Value) *fakeQ {
			return qrow([]string{"id", "email", "role", "is_active", "email_verified", "must_change_password", "mfa_enabled", "created_at", "mfa_secret"}, "u", "a@b.c", models.RoleUser, active, true, false, enabled, time.Now(), secret)
		}
		cases := []struct {
			name string
			qs   []*fakeQ
			want error
		}{
			{"token query", []*fakeQ{qerr(dbFailure)}, nil},
			{"expired", []*fakeQ{token(time.Now().Add(-time.Hour), nil)}, ErrInvalidToken},
			{"used", []*fakeQ{token(time.Now().Add(time.Hour), time.Now())}, ErrInvalidToken},
			{"user missing", []*fakeQ{token(time.Now().Add(time.Hour), nil), {}}, ErrUserNotFound},
			{"user query", []*fakeQ{token(time.Now().Add(time.Hour), nil), qerr(dbFailure)}, nil},
			{"inactive", []*fakeQ{token(time.Now().Add(time.Hour), nil), user(false, true, "x")}, ErrUserInactive},
			{"disabled", []*fakeQ{token(time.Now().Add(time.Hour), nil), user(true, false, nil)}, ErrMFANotEnabled},
			{"decrypt", []*fakeQ{token(time.Now().Add(time.Hour), nil), user(true, true, "broken")}, nil},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				s := newMFA()
				gFakeDrv.queries = tc.qs
				_, _, _, err := s.VerifyMFA("challenge", "000000")
				if tc.want != nil && !errors.Is(err, tc.want) {
					t.Fatalf("got %v want %v", err, tc.want)
				}
				if tc.want == nil && err == nil {
					t.Fatal("expected error")
				}
			})
		}

		secret := "JBSWY3DPEHPK3PXP"
		code, _ := totp.GenerateCode(secret, time.Now())
		for i := 0; i < 2; i++ {
			s := newMFA()
			enc, _ := s.mfaCipher.encrypt(secret)
			gFakeDrv.queries = []*fakeQ{token(time.Now().Add(time.Hour), nil), user(true, true, enc), qerr(dbFailure)}
			if i == 1 {
				gFakeDrv.queries[2] = &fakeQ{}
				gFakeDrv.commitErr = dbFailure
			}
			if _, _, _, err := s.VerifyMFA("challenge", code); err == nil {
				t.Fatalf("finalize case %d", i)
			}
		}
	})

	t.Run("MFA disable and status errors", func(t *testing.T) {
		db := openFakeDB(t)
		s, _ := New(db, "secret", time.Minute, time.Hour, nil, "", "", false, validKey(t))
		gFakeDrv.queries = []*fakeQ{qerr(dbFailure)}
		if err := s.DisableMFA("u", "", ""); err == nil {
			t.Fatal("disable query")
		}
		secret := "JBSWY3DPEHPK3PXP"
		enc, _ := s.mfaCipher.encrypt(secret)
		code, _ := totp.GenerateCode(secret, time.Now())
		gFakeDrv.queries = []*fakeQ{qrow([]string{"mfa_secret", "mfa_enabled", "password"}, enc, true, nil), qerr(dbFailure)}
		if err := s.DisableMFA("u", code, ""); err == nil {
			t.Fatal("disable update")
		}
		gFakeDrv.queries = []*fakeQ{{}}
		if _, err := s.MFAStatus("missing"); !errors.Is(err, ErrUserNotFound) {
			t.Fatalf("status missing: %v", err)
		}
	})
}

func TestCoverageThresholdBranches(t *testing.T) {
	dbFailure := errors.New("db failure")

	t.Run("complete OAuth write errors", func(t *testing.T) {
		token := func() *fakeQ {
			return qrow([]string{"id", "provider", "provider_subject", "email", "expires_at", "used_at"}, "t", "google", "sub", "a@b.c", time.Now().Add(time.Hour), nil)
		}
		cases := [][]*fakeQ{
			{token(), {}, qerr(&fakePGErr{code: "23505"})},
			{token(), {}, qerr(dbFailure)},
			{token(), {}, qrow([]string{"id", "email", "role", "is_active", "email_verified", "must_change_password", "created_at", "provider"}, "u", "a@b.c", models.RoleUser, true, true, false, time.Now(), "google"), qerr(dbFailure)},
		}
		for i, qs := range cases {
			db := openFakeDB(t)
			gFakeDrv.queries = qs
			if _, _, _, err := newSvc(t, db).CompleteOAuthSignup("google", "raw"); err == nil {
				t.Fatalf("case %d", i)
			}
		}
		db := openFakeDB(t)
		gFakeDrv.commitErr = dbFailure
		gFakeDrv.queries = []*fakeQ{token(), {}, qrow([]string{"id", "email", "role", "is_active", "email_verified", "must_change_password", "created_at", "provider"}, "u", "a@b.c", models.RoleUser, true, true, false, time.Now(), "google"), {}}
		if _, _, _, err := newSvc(t, db).CompleteOAuthSignup("google", "raw"); err == nil {
			t.Fatal("commit")
		}
	})

	t.Run("request email compensation failures", func(t *testing.T) {
		prefix := func() []*fakeQ {
			return []*fakeQ{qrow([]string{"email", "is_active"}, "old@b.c", true), qrow([]string{"exists"}, false), {}}
		}
		db := openFakeDB(t)
		gFakeDrv.queries = append(prefix(), qerr(dbFailure), &fakeQ{})
		if err := newSvc(t, db).RequestEmailChange("u", "new@b.c"); err == nil {
			t.Fatal("token creation")
		}
		db = openFakeDB(t)
		gFakeDrv.queries = append(prefix(), qerr(dbFailure), qerr(errors.New("cleanup")))
		if err := newSvc(t, db).RequestEmailChange("u", "new@b.c"); err == nil {
			t.Fatal("token cleanup")
		}
		db = openFakeDB(t)
		s, _ := New(db, "secret", time.Minute, time.Hour, &flowMailer{err: errors.New("smtp")}, "", "", false, "")
		gFakeDrv.queries = append(prefix(), &fakeQ{}, &fakeQ{}, qerr(errors.New("cancel")))
		if err := s.RequestEmailChange("u", "new@b.c"); err == nil {
			t.Fatal("mail cleanup")
		}
	})

	t.Run("repository edge errors", func(t *testing.T) {
		db := openFakeDB(t)
		gFakeDrv.queries = []*fakeQ{{}}
		if _, _, _, err := newSvc(t, db).AcceptTerms("missing"); !errors.Is(err, ErrUserNotFound) {
			t.Fatalf("terms missing: %v", err)
		}
		db = openFakeDB(t)
		gFakeDrv.queries = []*fakeQ{qerr(dbFailure)}
		if err := newSvc(t, db).SetRole("u", models.RoleUser); err == nil {
			t.Fatal("set role")
		}
		db = openFakeDB(t)
		s := newSvc(t, db)
		gFakeDrv.rowsAffected = 0
		gFakeDrv.queries = []*fakeQ{{}}
		if err := s.SetActive("u", true); !errors.Is(err, ErrUserNotFound) {
			t.Fatalf("active missing: %v", err)
		}
	})

	t.Run("OAuth onboarding token failure", func(t *testing.T) {
		db := openFakeDB(t)
		gFakeDrv.queries = []*fakeQ{{}, qerr(dbFailure)}
		if _, err := newSvc(t, db).LoginWithOAuth("google", "sub", "new@b.c"); err == nil {
			t.Fatal("expected token creation error")
		}
	})

	t.Run("MFA enable update error", func(t *testing.T) {
		db := openFakeDB(t)
		s, _ := New(db, "secret", time.Minute, time.Hour, nil, "", "", false, validKey(t))
		secret := "JBSWY3DPEHPK3PXP"
		enc, _ := s.mfaCipher.encrypt(secret)
		code, _ := totp.GenerateCode(secret, time.Now())
		gFakeDrv.queries = []*fakeQ{qrow([]string{"mfa_secret", "mfa_enabled"}, enc, false), qerr(dbFailure)}
		if err := s.EnableMFA("u", code); err == nil {
			t.Fatal("enable update")
		}
	})

	t.Run("sweep query error and active loop", func(t *testing.T) {
		db := openFakeDB(t)
		gFakeDrv.queries = []*fakeQ{qerr(dbFailure)}
		if _, err := newSvc(t, db).SweepBannedAccounts(context.Background(), eraser.New(eraser.Targets{}), time.Hour); err == nil {
			t.Fatal("sweep query")
		}
		db = openFakeDB(t)
		s := newSvc(t, db)
		gFakeDrv.queries = []*fakeQ{
			qrow([]string{"id", "email", "role", "is_active", "deactivated_at", "created_at"}, "u", "a@b.c", models.RoleUser, false, time.Now().Add(-time.Hour), time.Now()), {}, {},
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		s.RunAccountPurgeSweeper(ctx, eraser.New(eraser.Targets{}), time.Hour, time.Millisecond)
	})
}
