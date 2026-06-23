package handlers

import (
	"context"
	"database/sql/driver"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/webdad/auth-service/internal/models"
	"github.com/webdad/auth-service/internal/oauth"
	"github.com/webdad/auth-service/internal/services"
	"github.com/webdad/auth-service/internal/testutil"
)

type handlerMailer struct{ err error }

func (m handlerMailer) Send(string, string, string, string) error { return m.err }

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func integrationHandler(t *testing.T, responses ...*testutil.Response) *Handler {
	return integrationHandlerWithMailer(t, handlerMailer{}, responses...)
}

func integrationHandlerWithMailer(t *testing.T, mailer services.Mailer, responses ...*testutil.Response) *Handler {
	t.Helper()
	db := testutil.Open(responses...)
	key := base64.StdEncoding.EncodeToString([]byte("01234567890123456789012345678901"))
	svc, err := services.New(db, handlerSecret, 15*time.Minute, 24*time.Hour, mailer, "https://app.test", "", false, key)
	if err != nil {
		t.Fatal(err)
	}
	reg := oauth.NewRegistry(context.Background(), oauth.Options{
		RedirectBaseURL: "https://app.test",
		Providers: map[string]oauth.Credentials{
			"github": {ClientID: "client", ClientSecret: "secret"},
		},
	})
	return New(svc, reg)
}

func integrationHandlerWithoutMFA(t *testing.T, responses ...*testutil.Response) *Handler {
	t.Helper()
	db := testutil.Open(responses...)
	svc, err := services.New(db, handlerSecret, time.Minute, time.Hour, nil, "", "", false, "")
	if err != nil {
		t.Fatal(err)
	}
	return New(svc, oauth.NewRegistry(context.Background(), oauth.Options{}))
}

func perform(t *testing.T, method, path, body string, handler gin.HandlerFunc, claims *services.Claims) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	if claims != nil {
		r.Use(func(c *gin.Context) { c.Set("claims", claims); c.Next() })
	}
	routePath := strings.SplitN(path, "?", 2)[0]
	r.Handle(method, routePath, handler)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func authUserValues(id string, active, verified bool) []driver.Value {
	return []driver.Value{id, "user@example.test", models.RoleUser, active, verified, false, time.Now()}
}

func assertStatus(t *testing.T, got *httptest.ResponseRecorder, want int) {
	t.Helper()
	if got.Code != want {
		t.Fatalf("status = %d, want %d; body=%s", got.Code, want, got.Body.String())
	}
}

func TestHandlerSuccessfulFlows(t *testing.T) {
	t.Run("register", func(t *testing.T) {
		h := integrationHandler(t,
			testutil.Row([]string{"id", "email", "role", "is_active", "created_at"}, "u", "user@example.test", models.RoleUser, true, time.Now()),
			testutil.Row([]string{"terms_accepted_version"}, int64(models.CurrentTermsVersion)), testutil.Empty(),
		)
		assertStatus(t, perform(t, http.MethodPost, "/", `{"email":"user@example.test","password":"password123"}`, h.Register, nil), http.StatusCreated)
	})

	t.Run("login by email and id", func(t *testing.T) {
		hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
		for _, body := range []string{`{"email":"user@example.test","password":"password123"}`, `{"user_id":"u","password":"password123"}`} {
			h := integrationHandler(t,
				testutil.Row([]string{"id", "email", "password", "role", "is_active", "email_verified", "must_change_password", "mfa_enabled", "created_at"}, "u", "user@example.test", string(hash), models.RoleUser, true, true, false, false, time.Now()),
				testutil.Row([]string{"terms_accepted_version"}, int64(models.CurrentTermsVersion)), testutil.Empty(),
			)
			assertStatus(t, perform(t, http.MethodPost, "/", body, h.Login, nil), http.StatusOK)
		}
	})

	t.Run("password and verify", func(t *testing.T) {
		h := integrationHandler(t, testutil.Empty())
		assertStatus(t, perform(t, http.MethodPost, "/", `{"email":"missing@example.test"}`, h.ForgotPassword, nil), http.StatusOK)
		h = integrationHandler(t, testutil.Empty())
		assertStatus(t, perform(t, http.MethodPost, "/", `{"email":"missing@example.test"}`, h.RequestVerifyEmail, nil), http.StatusOK)

		base := testutil.Row([]string{"id", "user_id", "expires_at", "used_at"}, "tok", "u", time.Now().Add(time.Hour), nil)
		h = integrationHandler(t, base, testutil.Empty(), testutil.Empty(), testutil.Empty())
		assertStatus(t, perform(t, http.MethodPost, "/", `{"token":"raw","new_password":"new-password"}`, h.ResetPassword, nil), http.StatusOK)

		h = integrationHandler(t, base, testutil.Empty(), testutil.Row([]string{"id", "email", "role", "is_active", "email_verified", "must_change_password", "created_at"}, authUserValues("u", true, true)...), testutil.Row([]string{"terms_accepted_version"}, 1), testutil.Empty())
		assertStatus(t, perform(t, http.MethodPost, "/", `{"token":"raw"}`, h.ConfirmVerifyEmail, nil), http.StatusOK)
	})

	t.Run("terms", func(t *testing.T) {
		h := integrationHandler(t,
			testutil.Row([]string{"id", "email", "role", "is_active", "email_verified", "must_change_password", "created_at"}, authUserValues("u", true, true)...),
			testutil.Row([]string{"terms_accepted_version"}, 1), testutil.Empty(),
		)
		assertStatus(t, perform(t, http.MethodPost, "/", `{}`, h.AcceptTerms, &services.Claims{UserID: "u"}), http.StatusOK)
	})

	t.Run("email request and confirm", func(t *testing.T) {
		h := integrationHandler(t,
			testutil.Row([]string{"email", "is_active"}, "old@example.test", true), testutil.Row([]string{"exists"}, false),
			testutil.Empty(), testutil.Empty(), testutil.Empty(),
		)
		assertStatus(t, perform(t, http.MethodPost, "/", `{"email":"NEW@example.test"}`, h.RequestEmailChange, &services.Claims{UserID: "u"}), http.StatusOK)

		h = integrationHandler(t,
			testutil.Row([]string{"id", "user_id", "expires_at", "used_at"}, "tok", "u", time.Now().Add(time.Hour), nil),
			testutil.Row([]string{"id", "email", "role", "is_active", "email_verified", "must_change_password", "created_at"}, authUserValues("u", true, true)...),
			testutil.Empty(), testutil.Empty(), testutil.Empty(), testutil.Row([]string{"terms_accepted_version"}, 1), testutil.Empty(),
		)
		assertStatus(t, perform(t, http.MethodPost, "/", `{"token":"raw"}`, h.ConfirmEmailChange, nil), http.StatusOK)
	})

	t.Run("admin operations", func(t *testing.T) {
		admin := &services.Claims{UserID: "admin", Role: models.RoleAdmin}
		h := integrationHandler(t, testutil.Row([]string{"id", "email", "role", "is_active", "deactivated_at", "created_at"}, "u", "user@example.test", models.RoleUser, true, nil, time.Now()))
		assertStatus(t, perform(t, http.MethodGet, "/?limit=999&offset=-2", "", h.ListUsers, admin), http.StatusOK)

		h = integrationHandler(t, testutil.Row([]string{"id", "email", "role", "is_active", "email_verified", "must_change_password", "created_at"}, "u", "user@example.test", models.RoleUser, true, true, true, time.Now()))
		assertStatus(t, perform(t, http.MethodPost, "/", `{"email":"user@example.test","password":"temporary123","username":"user"}`, h.AdminCreateUser, admin), http.StatusCreated)

		h = integrationHandler(t, testutil.Empty())
		assertStatus(t, perform(t, http.MethodPut, "/", `{"role":"moderator"}`, h.SetRole, admin), http.StatusOK)
		h = integrationHandler(t, testutil.Empty())
		assertStatus(t, perform(t, http.MethodPatch, "/", `{"is_active":false}`, h.SetStatus, admin), http.StatusOK)
		h = integrationHandler(t, testutil.Empty(), testutil.Empty())
		assertStatus(t, perform(t, http.MethodDelete, "/", "", h.DeleteUser, admin), http.StatusNoContent)
	})

	t.Run("MFA setup and status", func(t *testing.T) {
		claims := &services.Claims{UserID: "u"}
		h := integrationHandler(t, testutil.Row([]string{"email", "mfa_enabled"}, "user@example.test", false), testutil.Empty())
		assertStatus(t, perform(t, http.MethodPost, "/", `{}`, h.MFASetup, claims), http.StatusOK)
		h = integrationHandler(t, testutil.Row([]string{"mfa_enabled"}, false))
		assertStatus(t, perform(t, http.MethodGet, "/", "", h.MFAStatus, claims), http.StatusOK)
	})

	t.Run("OAuth URL and validation", func(t *testing.T) {
		h := integrationHandler(t)
		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.GET("/:provider", h.OAuthURL)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/github", nil))
		assertStatus(t, w, http.StatusOK)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/unknown", nil))
		assertStatus(t, w, http.StatusNotFound)
	})
}

func TestHandlerMappedErrors(t *testing.T) {
	dbErr := errors.New("db failure")

	t.Run("register conflict and internal", func(t *testing.T) {
		for _, tc := range []struct {
			err    error
			status int
		}{{&fakePGState{"23505"}, 409}, {dbErr, 500}} {
			h := integrationHandler(t, testutil.Error(tc.err))
			assertStatus(t, perform(t, http.MethodPost, "/", `{"email":"user@example.test","password":"password123"}`, h.Register, nil), tc.status)
		}
	})

	t.Run("login mappings", func(t *testing.T) {
		hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
		rows := []struct {
			values []driver.Value
			status int
		}{
			{[]driver.Value{"u", "user@example.test", string(hash), models.RoleUser, false, true, false, false, time.Now()}, 403},
			{[]driver.Value{"u", "user@example.test", nil, models.RoleUser, true, true, false, false, time.Now()}, 409},
			{[]driver.Value{"u", "user@example.test", string(hash), models.RoleUser, true, false, false, false, time.Now()}, 403},
		}
		for _, tc := range rows {
			h := integrationHandler(t, testutil.Row([]string{"id", "email", "password", "role", "is_active", "email_verified", "must_change_password", "mfa_enabled", "created_at"}, tc.values...))
			assertStatus(t, perform(t, http.MethodPost, "/", `{"email":"user@example.test","password":"password123"}`, h.Login, nil), tc.status)
		}
		h := integrationHandler(t, testutil.Empty())
		assertStatus(t, perform(t, http.MethodPost, "/", `{"email":"user@example.test","password":"password123"}`, h.Login, nil), 401)
		h = integrationHandler(t, testutil.Error(dbErr))
		assertStatus(t, perform(t, http.MethodPost, "/", `{"email":"user@example.test","password":"password123"}`, h.Login, nil), 500)
	})

	t.Run("email request mappings", func(t *testing.T) {
		claims := &services.Claims{UserID: "u"}
		for _, tc := range []struct {
			responses []*testutil.Response
			status    int
		}{
			{[]*testutil.Response{{}}, 404},
			{[]*testutil.Response{testutil.Row([]string{"email", "is_active"}, "old@example.test", false)}, 403},
			{[]*testutil.Response{testutil.Row([]string{"email", "is_active"}, "new@example.test", true)}, 400},
			{[]*testutil.Response{testutil.Row([]string{"email", "is_active"}, "old@example.test", true), testutil.Row([]string{"exists"}, true)}, 400},
			{[]*testutil.Response{testutil.Error(dbErr)}, 500},
		} {
			h := integrationHandler(t, tc.responses...)
			assertStatus(t, perform(t, http.MethodPost, "/", `{"email":"new@example.test"}`, h.RequestEmailChange, claims), tc.status)
		}
	})

	t.Run("refresh mappings", func(t *testing.T) {
		h := integrationHandler(t, testutil.Empty())
		assertStatus(t, perform(t, http.MethodPost, "/", `{"refresh_token":"raw"}`, h.Refresh, nil), 401)
		h = integrationHandler(t, testutil.Error(dbErr))
		assertStatus(t, perform(t, http.MethodPost, "/", `{"refresh_token":"raw"}`, h.Refresh, nil), 500)
		assertStatus(t, perform(t, http.MethodPost, "/", `{}`, h.Refresh, nil), 401)
	})

	t.Run("admin error mapper and pagination", func(t *testing.T) {
		for _, tc := range []struct {
			err    error
			status int
		}{{services.ErrUserNotFound, 404}, {services.ErrInvalidRole, 400}, {services.ErrInsufficientPrivilege, 403}, {dbErr, 500}} {
			gin.SetMode(gin.TestMode)
			r := gin.New()
			r.GET("/", func(c *gin.Context) { respondAdminError(c, tc.err) })
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
			assertStatus(t, w, tc.status)
		}
		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodGet, "/?limit=nope&offset=nope", nil)
		limit, offset := paginate(c)
		if limit != defaultLimit || offset != 0 {
			t.Fatalf("paginate = %d,%d", limit, offset)
		}
	})
}

func oauthPerform(t *testing.T, method, provider, body string, handler gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Handle(method, "/:provider", handler)
	req := httptest.NewRequest(method, "/"+provider, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestOAuthHandlers(t *testing.T) {
	t.Run("complete validation and errors", func(t *testing.T) {
		h := integrationHandler(t)
		assertStatus(t, oauthPerform(t, http.MethodPost, "unknown", `{}`, h.OAuthComplete), 404)
		assertStatus(t, oauthPerform(t, http.MethodPost, "github", `{bad}`, h.OAuthComplete), 400)
		assertStatus(t, oauthPerform(t, http.MethodPost, "github", `{"pending_token":"raw","accepted_terms":false}`, h.OAuthComplete), 400)

		h = integrationHandler(t, testutil.Empty())
		assertStatus(t, oauthPerform(t, http.MethodPost, "github", `{"pending_token":"raw","accepted_terms":true}`, h.OAuthComplete), 401)

		token := testutil.Row([]string{"id", "provider", "provider_subject", "email", "expires_at", "used_at"}, "t", "github", "sub", "user@example.test", time.Now().Add(time.Hour), nil)
		h = integrationHandler(t, token, testutil.Row([]string{"one"}, 1))
		assertStatus(t, oauthPerform(t, http.MethodPost, "github", `{"pending_token":"raw","accepted_terms":true}`, h.OAuthComplete), 409)

		h = integrationHandler(t, testutil.Error(errors.New("db")))
		assertStatus(t, oauthPerform(t, http.MethodPost, "github", `{"pending_token":"raw","accepted_terms":true}`, h.OAuthComplete), 500)
	})

	t.Run("complete success", func(t *testing.T) {
		token := testutil.Row([]string{"id", "provider", "provider_subject", "email", "expires_at", "used_at"}, "t", "github", "sub", "user@example.test", time.Now().Add(time.Hour), nil)
		h := integrationHandler(t,
			token, testutil.Empty(),
			testutil.Row([]string{"id", "email", "role", "is_active", "email_verified", "must_change_password", "created_at", "provider"}, "u", "user@example.test", models.RoleUser, true, true, false, time.Now(), "github"),
			testutil.Empty(), testutil.Row([]string{"terms_accepted_version"}, 1), testutil.Empty(),
		)
		assertStatus(t, oauthPerform(t, http.MethodPost, "github", `{"pending_token":"raw","accepted_terms":true}`, h.OAuthComplete), 200)
	})

	t.Run("exchange guards", func(t *testing.T) {
		h := integrationHandler(t)
		assertStatus(t, oauthPerform(t, http.MethodPost, "unknown", `{}`, h.OAuthExchange), 404)
		assertStatus(t, oauthPerform(t, http.MethodPost, "github", `{bad}`, h.OAuthExchange), 400)
		assertStatus(t, oauthPerform(t, http.MethodPost, "github", `{"code":"bad"}`, h.OAuthExchange), 401)
	})
}

func TestRemainingHandlerMappings(t *testing.T) {
	dbErr := errors.New("db failure")
	claims := &services.Claims{UserID: "u", Role: models.RoleUser}

	t.Run("terms", func(t *testing.T) {
		h := integrationHandler(t, testutil.Empty())
		assertStatus(t, perform(t, http.MethodPost, "/", `{}`, h.AcceptTerms, claims), 404)
		h = integrationHandler(t, testutil.Error(dbErr))
		assertStatus(t, perform(t, http.MethodPost, "/", `{}`, h.AcceptTerms, claims), 500)
		assertStatus(t, perform(t, http.MethodPost, "/", `{}`, h.AcceptTerms, nil), 401)
	})

	t.Run("change password", func(t *testing.T) {
		hash, _ := bcrypt.GenerateFromPassword([]byte("old-password"), bcrypt.MinCost)
		row := func(password driver.Value) *testutil.Response {
			return testutil.Row([]string{"id", "email", "password", "role", "is_active", "email_verified", "must_change_password", "created_at"}, "u", "user@example.test", password, models.RoleUser, true, true, false, time.Now())
		}
		h := integrationHandler(t, row(string(hash)))
		assertStatus(t, perform(t, http.MethodPost, "/", `{"current_password":"wrong","new_password":"new-password"}`, h.ChangePassword, claims), 400)
		h = integrationHandler(t, testutil.Empty())
		assertStatus(t, perform(t, http.MethodPost, "/", `{"current_password":"old-password","new_password":"new-password"}`, h.ChangePassword, claims), 404)
		h = integrationHandler(t, testutil.Error(dbErr))
		assertStatus(t, perform(t, http.MethodPost, "/", `{"current_password":"old-password","new_password":"new-password"}`, h.ChangePassword, claims), 500)

		h = integrationHandler(t, row(string(hash)), testutil.Empty(), testutil.Empty(), testutil.Row([]string{"terms_accepted_version"}, 1), testutil.Empty())
		assertStatus(t, perform(t, http.MethodPost, "/", `{"current_password":"old-password","new_password":"new-password"}`, h.ChangePassword, claims), 200)
	})

	t.Run("reset and confirm errors", func(t *testing.T) {
		h := integrationHandler(t, testutil.Empty())
		assertStatus(t, perform(t, http.MethodPost, "/", `{"token":"raw","new_password":"new-password"}`, h.ResetPassword, nil), 400)
		h = integrationHandler(t, testutil.Error(dbErr))
		assertStatus(t, perform(t, http.MethodPost, "/", `{"token":"raw","new_password":"new-password"}`, h.ResetPassword, nil), 500)

		h = integrationHandler(t, testutil.Empty())
		assertStatus(t, perform(t, http.MethodPost, "/", `{"token":"raw"}`, h.ConfirmVerifyEmail, nil), 400)
		h = integrationHandler(t, testutil.Error(dbErr))
		assertStatus(t, perform(t, http.MethodPost, "/", `{"token":"raw"}`, h.ConfirmVerifyEmail, nil), 500)
	})

	t.Run("confirm email mappings", func(t *testing.T) {
		validToken := func() *testutil.Response {
			return testutil.Row([]string{"id", "user_id", "expires_at", "used_at"}, "t", "u", time.Now().Add(time.Hour), nil)
		}
		cases := []struct {
			responses []*testutil.Response
			status    int
		}{
			{[]*testutil.Response{{}}, 400},
			{[]*testutil.Response{validToken(), testutil.Error(&fakePGState{"23505"})}, 400},
			{[]*testutil.Response{validToken(), testutil.Row([]string{"id", "email", "role", "is_active", "email_verified", "must_change_password", "created_at"}, authUserValues("u", false, true)...)}, 403},
			{[]*testutil.Response{testutil.Error(dbErr)}, 500},
		}
		for _, tc := range cases {
			h := integrationHandler(t, tc.responses...)
			assertStatus(t, perform(t, http.MethodPost, "/", `{"token":"raw"}`, h.ConfirmEmailChange, nil), tc.status)
		}
	})

	t.Run("MFA mappings", func(t *testing.T) {
		h := integrationHandler(t, testutil.Empty())
		assertStatus(t, perform(t, http.MethodPost, "/", `{"challenge":"raw","code":"000000"}`, h.MFAVerify, nil), 400)
		h = integrationHandler(t, testutil.Error(dbErr))
		assertStatus(t, perform(t, http.MethodPost, "/", `{"challenge":"raw","code":"000000"}`, h.MFAVerify, nil), 500)
		h = integrationHandler(t, testutil.Empty())
		assertStatus(t, perform(t, http.MethodPost, "/", `{"code":"000000"}`, h.MFAEnable, claims), 404)
		h = integrationHandler(t, testutil.Row([]string{"mfa_secret", "mfa_enabled"}, nil, false))
		assertStatus(t, perform(t, http.MethodPost, "/", `{"code":"000000"}`, h.MFAEnable, claims), 400)
		h = integrationHandler(t, testutil.Row([]string{"mfa_secret", "mfa_enabled", "password"}, nil, false, nil))
		assertStatus(t, perform(t, http.MethodPost, "/", `{"code":"000000"}`, h.MFADisable, claims), 400)
		h = integrationHandler(t, testutil.Error(dbErr))
		assertStatus(t, perform(t, http.MethodGet, "/", "", h.MFAStatus, claims), 500)
		h = integrationHandler(t, testutil.Empty())
		assertStatus(t, perform(t, http.MethodGet, "/", "", h.MFAStatus, claims), 404)
	})

	t.Run("admin DB mappings", func(t *testing.T) {
		admin := &services.Claims{UserID: "admin", Role: models.RoleAdmin}
		h := integrationHandler(t, testutil.Error(dbErr))
		assertStatus(t, perform(t, http.MethodGet, "/", "", h.ListUsers, admin), 500)
		h = integrationHandler(t, testutil.Error(&fakePGState{"23505"}))
		assertStatus(t, perform(t, http.MethodPost, "/", `{"email":"user@example.test","password":"temporary123"}`, h.AdminCreateUser, admin), 409)
		h = integrationHandler(t, testutil.Error(dbErr))
		assertStatus(t, perform(t, http.MethodPost, "/", `{"email":"user@example.test","password":"temporary123"}`, h.AdminCreateUser, admin), 500)
		h = integrationHandler(t, testutil.Empty())
		assertStatus(t, perform(t, http.MethodPut, "/", `{"role":"root"}`, h.SetRole, admin), 400)
	})
}

func TestHandlerGuardAndMFABranches(t *testing.T) {
	claims := &services.Claims{UserID: "u", Role: models.RoleUser}
	dbErr := errors.New("db")

	t.Run("validate", func(t *testing.T) {
		h := integrationHandler(t)
		assertStatus(t, perform(t, http.MethodGet, "/", "", h.Validate, nil), 401)
		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.Use(func(c *gin.Context) { c.Set("claims", "bad"); c.Next() })
		r.GET("/", h.Validate)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
		assertStatus(t, w, 500)
	})

	t.Run("logout error", func(t *testing.T) {
		h := integrationHandler(t, testutil.Error(dbErr))
		assertStatus(t, perform(t, http.MethodPost, "/", `{"refresh_token":"raw"}`, h.Logout, nil), 500)
	})

	t.Run("setup mappings", func(t *testing.T) {
		h := integrationHandler(t)
		assertStatus(t, perform(t, http.MethodPost, "/", `{}`, h.MFASetup, nil), 401)
		h = integrationHandlerWithoutMFA(t)
		assertStatus(t, perform(t, http.MethodPost, "/", `{}`, h.MFASetup, claims), 503)
		h = integrationHandler(t, testutil.Empty())
		assertStatus(t, perform(t, http.MethodPost, "/", `{}`, h.MFASetup, claims), 404)
		h = integrationHandler(t, testutil.Row([]string{"email", "mfa_enabled"}, "a@b.c", true))
		assertStatus(t, perform(t, http.MethodPost, "/", `{}`, h.MFASetup, claims), 409)
		h = integrationHandler(t, testutil.Error(dbErr))
		assertStatus(t, perform(t, http.MethodPost, "/", `{}`, h.MFASetup, claims), 500)
	})

	t.Run("enable mappings", func(t *testing.T) {
		h := integrationHandler(t)
		assertStatus(t, perform(t, http.MethodPost, "/", `{}`, h.MFAEnable, nil), 401)
		h = integrationHandler(t)
		assertStatus(t, perform(t, http.MethodPost, "/", `{bad}`, h.MFAEnable, claims), 400)
		h = integrationHandlerWithoutMFA(t)
		assertStatus(t, perform(t, http.MethodPost, "/", `{"code":"000000"}`, h.MFAEnable, claims), 503)
		h = integrationHandler(t, testutil.Row([]string{"mfa_secret", "mfa_enabled"}, nil, true))
		assertStatus(t, perform(t, http.MethodPost, "/", `{"code":"000000"}`, h.MFAEnable, claims), 409)
		h = integrationHandler(t, testutil.Error(dbErr))
		assertStatus(t, perform(t, http.MethodPost, "/", `{"code":"000000"}`, h.MFAEnable, claims), 500)
	})

	t.Run("disable mappings", func(t *testing.T) {
		h := integrationHandler(t)
		assertStatus(t, perform(t, http.MethodPost, "/", `{}`, h.MFADisable, nil), 401)
		h = integrationHandler(t)
		assertStatus(t, perform(t, http.MethodPost, "/", `{bad}`, h.MFADisable, claims), 400)
		h = integrationHandlerWithoutMFA(t)
		assertStatus(t, perform(t, http.MethodPost, "/", `{"code":"000000"}`, h.MFADisable, claims), 503)
		h = integrationHandler(t, testutil.Empty())
		assertStatus(t, perform(t, http.MethodPost, "/", `{"code":"000000"}`, h.MFADisable, claims), 404)
		h = integrationHandler(t, testutil.Row([]string{"mfa_secret", "mfa_enabled", "password"}, nil, true, nil))
		assertStatus(t, perform(t, http.MethodPost, "/", `{"code":"000000"}`, h.MFADisable, claims), 400)
		h = integrationHandler(t, testutil.Error(dbErr))
		assertStatus(t, perform(t, http.MethodPost, "/", `{"code":"000000"}`, h.MFADisable, claims), 500)
	})

	t.Run("verify mappings", func(t *testing.T) {
		h := integrationHandler(t)
		assertStatus(t, perform(t, http.MethodPost, "/", `{bad}`, h.MFAVerify, nil), 400)
		h = integrationHandlerWithoutMFA(t)
		assertStatus(t, perform(t, http.MethodPost, "/", `{"challenge":"raw","code":"000000"}`, h.MFAVerify, nil), 503)
		token := func() *testutil.Response {
			return testutil.Row([]string{"id", "user_id", "expires_at", "used_at"}, "t", "u", time.Now().Add(time.Hour), nil)
		}
		user := func(active, enabled bool, secret driver.Value) *testutil.Response {
			return testutil.Row([]string{"id", "email", "role", "is_active", "email_verified", "must_change_password", "mfa_enabled", "created_at", "mfa_secret"}, "u", "a@b.c", models.RoleUser, active, true, false, enabled, time.Now(), secret)
		}
		h = integrationHandler(t, token(), user(true, false, nil))
		assertStatus(t, perform(t, http.MethodPost, "/", `{"challenge":"raw","code":"000000"}`, h.MFAVerify, nil), 400)
		h = integrationHandler(t, token(), user(false, true, "x"))
		assertStatus(t, perform(t, http.MethodPost, "/", `{"challenge":"raw","code":"000000"}`, h.MFAVerify, nil), 403)
	})

	t.Run("status no claims", func(t *testing.T) {
		h := integrationHandler(t)
		assertStatus(t, perform(t, http.MethodGet, "/", "", h.MFAStatus, nil), 401)
	})

	t.Run("admin guards and moderator hierarchy", func(t *testing.T) {
		h := integrationHandler(t)
		assertStatus(t, perform(t, http.MethodPut, "/", `{"role":"user"}`, h.SetRole, nil), 401)
		assertStatus(t, perform(t, http.MethodPatch, "/", `{"is_active":true}`, h.SetStatus, nil), 401)
		assertStatus(t, perform(t, http.MethodDelete, "/", "", h.DeleteUser, nil), 401)

		moderator := &services.Claims{UserID: "mod", Role: models.RoleModerator}
		h = integrationHandler(t, testutil.Row([]string{"role"}, models.RoleAdmin))
		assertStatus(t, perform(t, http.MethodPatch, "/", `{"is_active":false}`, h.SetStatus, moderator), 403)
		h = integrationHandler(t, testutil.Row([]string{"role"}, models.RoleUser), testutil.Empty())
		assertStatus(t, perform(t, http.MethodPatch, "/", `{"is_active":true}`, h.SetStatus, moderator), 200)
		h = integrationHandler(t, testutil.Error(dbErr))
		assertStatus(t, perform(t, http.MethodPatch, "/", `{"is_active":true}`, h.SetStatus, moderator), 500)
	})
}

func TestHandlerOAuthExchangeSuccessBranches(t *testing.T) {
	oldTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := `{"access_token":"provider-token","token_type":"bearer"}`
		if strings.Contains(r.URL.Host, "api.github.com") {
			body = `{"id":42,"email":"user@example.test"}`
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = oldTransport })

	t.Run("onboarding", func(t *testing.T) {
		h := integrationHandler(t, testutil.Empty(), testutil.Empty(), testutil.Empty())
		w := oauthPerform(t, http.MethodPost, "github", `{"code":"good"}`, h.OAuthExchange)
		assertStatus(t, w, 200)
		if !strings.Contains(w.Body.String(), "onboarding_required") {
			t.Fatalf("body=%s", w.Body.String())
		}
	})

	t.Run("existing account", func(t *testing.T) {
		h := integrationHandler(t,
			testutil.Row([]string{"id", "email", "role", "is_active", "must_change_password", "created_at", "provider"}, "u", "user@example.test", models.RoleUser, true, false, time.Now(), "github"),
			testutil.Empty(), testutil.Row([]string{"terms_accepted_version"}, 1), testutil.Empty(),
		)
		w := oauthPerform(t, http.MethodPost, "github", `{"code":"good"}`, h.OAuthExchange)
		assertStatus(t, w, 200)
		if !strings.Contains(w.Body.String(), "refresh_token") {
			t.Fatalf("body=%s", w.Body.String())
		}
	})

	t.Run("inactive account", func(t *testing.T) {
		h := integrationHandler(t, testutil.Row([]string{"id", "email", "role", "is_active", "must_change_password", "created_at", "provider"}, "u", "user@example.test", models.RoleUser, false, false, time.Now(), "github"))
		assertStatus(t, oauthPerform(t, http.MethodPost, "github", `{"code":"good"}`, h.OAuthExchange), 403)
	})
}

func TestMoreHandlerSuccessAndErrors(t *testing.T) {
	t.Run("refresh success and inactive", func(t *testing.T) {
		row := func(active bool) *testutil.Response {
			return testutil.Row([]string{"id", "email", "role", "is_active", "must_change_password", "email_verified", "created_at", "expires_at"}, "u", "user@example.test", models.RoleUser, active, false, true, time.Now(), time.Now().Add(time.Hour))
		}
		h := integrationHandler(t, row(true), testutil.Empty(), testutil.Row([]string{"terms_accepted_version"}, 1), testutil.Empty())
		assertStatus(t, perform(t, http.MethodPost, "/", `{"refresh_token":"raw"}`, h.Refresh, nil), 200)
		h = integrationHandler(t, row(false))
		assertStatus(t, perform(t, http.MethodPost, "/", `{"refresh_token":"raw"}`, h.Refresh, nil), 403)
	})

	t.Run("MFA disable by password", func(t *testing.T) {
		hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
		h := integrationHandler(t, testutil.Row([]string{"mfa_secret", "mfa_enabled", "password"}, nil, true, string(hash)), testutil.Empty())
		assertStatus(t, perform(t, http.MethodPost, "/", `{"password":"password123"}`, h.MFADisable, &services.Claims{UserID: "u"}), 200)
	})

	t.Run("admin write errors", func(t *testing.T) {
		admin := &services.Claims{UserID: "admin", Role: models.RoleAdmin}
		h := integrationHandler(t, testutil.Error(errors.New("db")))
		assertStatus(t, perform(t, http.MethodPatch, "/", `{"is_active":true}`, h.SetStatus, admin), 500)
		h = integrationHandler(t, testutil.Empty(), testutil.Error(errors.New("db")))
		assertStatus(t, perform(t, http.MethodDelete, "/", "", h.DeleteUser, admin), 500)
	})
}

func TestFinalHandlerBranches(t *testing.T) {
	t.Run("email request guards and delivery failure", func(t *testing.T) {
		h := integrationHandler(t)
		assertStatus(t, perform(t, http.MethodPost, "/", `{}`, h.RequestEmailChange, nil), 401)
		assertStatus(t, perform(t, http.MethodPost, "/", `{bad}`, h.RequestEmailChange, &services.Claims{UserID: "u"}), 400)

		h = integrationHandlerWithMailer(t, handlerMailer{err: errors.New("smtp")},
			testutil.Row([]string{"email", "is_active"}, "old@example.test", true), testutil.Row([]string{"exists"}, false),
			testutil.Empty(), testutil.Empty(), testutil.Empty(), testutil.Empty(), testutil.Empty(),
		)
		assertStatus(t, perform(t, http.MethodPost, "/", `{"email":"new@example.test"}`, h.RequestEmailChange, &services.Claims{UserID: "u"}), 503)
	})

	t.Run("confirm invalid JSON", func(t *testing.T) {
		h := integrationHandler(t)
		assertStatus(t, perform(t, http.MethodPost, "/", `{bad}`, h.ConfirmEmailChange, nil), 400)
	})

	t.Run("login MFA challenge", func(t *testing.T) {
		hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
		h := integrationHandler(t,
			testutil.Row([]string{"id", "email", "password", "role", "is_active", "email_verified", "must_change_password", "mfa_enabled", "created_at"}, "u", "user@example.test", string(hash), models.RoleUser, true, true, false, true, time.Now()),
			testutil.Empty(), testutil.Empty(),
		)
		w := perform(t, http.MethodPost, "/", `{"email":"user@example.test","password":"password123"}`, h.Login, nil)
		assertStatus(t, w, 200)
		if !strings.Contains(w.Body.String(), "mfa_required") {
			t.Fatalf("body=%s", w.Body.String())
		}
	})

	t.Run("OAuth service error", func(t *testing.T) {
		oldTransport := http.DefaultTransport
		http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
			body := `{"access_token":"provider-token","token_type":"bearer"}`
			if strings.Contains(r.URL.Host, "api.github.com") {
				body = `{"id":42,"email":"user@example.test"}`
			}
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
		})
		defer func() { http.DefaultTransport = oldTransport }()
		h := integrationHandler(t, testutil.Error(errors.New("db")))
		assertStatus(t, oauthPerform(t, http.MethodPost, "github", `{"code":"good"}`, h.OAuthExchange), 500)
	})
}

type fakePGState struct{ state string }

func (e *fakePGState) Error() string    { return e.state }
func (e *fakePGState) SQLState() string { return e.state }
