package handlers

import (
	"errors"
	"net/http"
	"testing"

	"github.com/webdad/user-service/internal/service"
)

func TestRespondUserError_StatusMapping(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"user not found", service.ErrUserNotFound, http.StatusNotFound},
		{"follow request not found", service.ErrFollowRequestNotFound, http.StatusNotFound},
		{"username taken", service.ErrUsernameTaken, http.StatusConflict},
		{"invalid username", service.ErrInvalidUsername, http.StatusBadRequest},
		{"invalid locale", service.ErrInvalidLocale, http.StatusBadRequest},
		{"self follow", service.ErrSelfFollow, http.StatusBadRequest},
		{"self block", service.ErrSelfBlock, http.StatusBadRequest},
		{"cooldown", service.ErrUsernameCooldown, http.StatusTooManyRequests},
		{"unexpected", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, w := newCtx(http.MethodGet, "/", "")
			respondUserError(c, tc.err)
			if w.Code != tc.want {
				t.Fatalf("status = %d, attendu %d", w.Code, tc.want)
			}
		})
	}
}
