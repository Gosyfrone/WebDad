package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/webdad/mail-service/internal/mailer"
)

type failingMailer struct{ err error }

func (m failingMailer) Send(mailer.Message) error { return m.err }

func TestSendReturnsServiceUnavailableWithoutDeliveryTransport(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewSendHandler(failingMailer{err: mailer.ErrDeliveryUnavailable}, "secret")
	router := gin.New()
	router.POST("/internal/send", h.Send)

	req := httptest.NewRequest(http.MethodPost, "/internal/send", strings.NewReader(
		`{"to":"test@example.com","subject":"Test","text":"Message"}`,
	))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Secret", "secret")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d; body = %s", res.Code, http.StatusServiceUnavailable, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), `"code":"delivery_unavailable"`) {
		t.Fatalf("body = %s, want delivery_unavailable code", res.Body.String())
	}
}

func TestSendReturnsBadGatewayOnSMTPFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewSendHandler(failingMailer{err: errors.New("smtp failure")}, "secret")
	router := gin.New()
	router.POST("/internal/send", h.Send)

	req := httptest.NewRequest(http.MethodPost, "/internal/send", strings.NewReader(
		`{"to":"test@example.com","subject":"Test","text":"Message"}`,
	))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Secret", "secret")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d; body = %s", res.Code, http.StatusBadGateway, res.Body.String())
	}
}
