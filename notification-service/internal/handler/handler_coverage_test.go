package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/webdad/notification-service/internal/models"
	"github.com/webdad/notification-service/internal/realtime"
	"github.com/webdad/notification-service/internal/service"
)

type handlerFakeRepository struct {
	items          []models.Notification
	count          int64
	listErr        error
	countErr       error
	markAllReadErr error
	markReadErr    error
}

func (f *handlerFakeRepository) Upsert(context.Context, *models.Notification) (*models.Notification, error) {
	return nil, errors.New("not implemented")
}
func (f *handlerFakeRepository) UpsertUniqueActor(context.Context, *models.Notification) (*models.Notification, error) {
	return nil, errors.New("not implemented")
}
func (f *handlerFakeRepository) Decrement(context.Context, string, string) (*models.Notification, bool, error) {
	return nil, false, errors.New("not implemented")
}
func (f *handlerFakeRepository) DeleteByPost(context.Context, string) ([]string, error) {
	return nil, errors.New("not implemented")
}
func (f *handlerFakeRepository) List(context.Context, string, int64, *bson.ObjectID) ([]models.Notification, error) {
	return f.items, f.listErr
}
func (f *handlerFakeRepository) CountUnread(context.Context, string) (int64, error) {
	return f.count, f.countErr
}
func (f *handlerFakeRepository) MarkAllRead(context.Context, string) error {
	return f.markAllReadErr
}
func (f *handlerFakeRepository) MarkRead(context.Context, string, bson.ObjectID) error {
	return f.markReadErr
}

func routerWithRepository(repo service.Repository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r, "notification-test", service.NewNotificationService(repo, realtime.NewHub(), nil), realtime.NewHub(), testSecret, "internal-secret", nil)
	return r
}

func authenticatedRequest(t *testing.T, r http.Handler, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Authorization", "Bearer "+makeNotifToken(t))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestNotificationHandlersRejectMissingClaims(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewNotificationHandler(service.NewNotificationService(nil, nil, nil))
	tests := []struct {
		name   string
		method string
		path   string
		handle gin.HandlerFunc
	}{
		{"list", http.MethodGet, "/notifications", h.List},
		{"unread count", http.MethodGet, "/notifications/unread-count", h.UnreadCount},
		{"mark all read", http.MethodPost, "/notifications/read", h.MarkAllRead},
		{"mark read", http.MethodPost, "/notifications/id/read", h.MarkRead},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Handle(tt.method, tt.path, tt.handle)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(tt.method, tt.path, nil))
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, attendu 401", w.Code)
			}
		})
	}
}

func TestInternalEventsValidationAndAcceptedEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := service.NewNotificationService(&handlerFakeRepository{}, nil, nil)
	h := NewInternalHandler(svc, "internal-secret")
	r := gin.New()
	r.POST("/internal/events", h.Events)

	tests := []struct {
		name   string
		body   string
		secret string
		want   int
	}{
		{"secret absent", `{"type":"unknown","actor_id":"actor"}`, "", http.StatusUnauthorized},
		{"json invalide", `{`, "internal-secret", http.StatusBadRequest},
		{"type requis", `{"actor_id":"actor"}`, "internal-secret", http.StatusBadRequest},
		{"acteur requis", `{"type":"unknown"}`, "internal-secret", http.StatusBadRequest},
		{"événement inconnu accepté", `{"type":"unknown","actor_id":"actor"}`, "internal-secret", http.StatusAccepted},
		{"traitement impossible", `{"type":"like","actor_id":"actor","recipient_id":"recipient","post_id":"post"}`, "internal-secret", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/internal/events", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if tt.secret != "" {
				req.Header.Set("X-Internal-Secret", tt.secret)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tt.want {
				t.Fatalf("status = %d, attendu %d; body=%s", w.Code, tt.want, w.Body.String())
			}
		})
	}
}

func TestWSOriginPolicy(t *testing.T) {
	h := NewWSHandler(realtime.NewHub(), "secret", []string{"https://allowed.test"})
	tests := []struct {
		origin string
		want   bool
	}{
		{"", true},
		{"https://allowed.test", true},
		{"https://denied.test", false},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, "/notifications/ws", nil)
		if tt.origin != "" {
			req.Header.Set("Origin", tt.origin)
		}
		if got := h.upgrader.CheckOrigin(req); got != tt.want {
			t.Errorf("CheckOrigin(%q) = %v, attendu %v", tt.origin, got, tt.want)
		}
	}
}

func TestWSValidTokenWithoutUpgradeHeaders(t *testing.T) {
	r := routerWithRepository(&handlerFakeRepository{})
	req := httptest.NewRequest(http.MethodGet, "/notifications/ws?access_token="+makeNotifToken(t), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, attendu 400", w.Code)
	}
}

func TestNotificationHandlersSuccess(t *testing.T) {
	id := bson.NewObjectID()
	repo := &handlerFakeRepository{
		items: []models.Notification{{ID: id, RecipientID: "u1"}},
		count: 3,
	}
	r := routerWithRepository(repo)

	tests := []struct {
		method string
		path   string
		want   int
	}{
		{http.MethodGet, "/notifications?limit=12", http.StatusOK},
		{http.MethodGet, "/notifications/unread-count", http.StatusOK},
		{http.MethodPost, "/notifications/read", http.StatusNoContent},
		{http.MethodPost, "/notifications/" + id.Hex() + "/read", http.StatusNoContent},
	}
	for _, tt := range tests {
		w := authenticatedRequest(t, r, tt.method, tt.path)
		if w.Code != tt.want {
			t.Errorf("%s %s = %d, attendu %d; body=%s", tt.method, tt.path, w.Code, tt.want, w.Body.String())
		}
	}
}

func TestNotificationHandlersRepositoryErrors(t *testing.T) {
	id := bson.NewObjectID().Hex()
	tests := []struct {
		name   string
		repo   *handlerFakeRepository
		method string
		path   string
		want   int
	}{
		{"list", &handlerFakeRepository{listErr: errors.New("list")}, http.MethodGet, "/notifications", http.StatusInternalServerError},
		{"count", &handlerFakeRepository{countErr: errors.New("count")}, http.MethodGet, "/notifications/unread-count", http.StatusInternalServerError},
		{"mark all", &handlerFakeRepository{markAllReadErr: errors.New("mark all")}, http.MethodPost, "/notifications/read", http.StatusInternalServerError},
		{"mark read absent", &handlerFakeRepository{markReadErr: mongo.ErrNoDocuments}, http.MethodPost, "/notifications/" + id + "/read", http.StatusNotFound},
		{"mark read erreur", &handlerFakeRepository{markReadErr: errors.New("mark")}, http.MethodPost, "/notifications/" + id + "/read", http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := authenticatedRequest(t, routerWithRepository(tt.repo), tt.method, tt.path)
			if w.Code != tt.want {
				t.Fatalf("status = %d, attendu %d; body=%s", w.Code, tt.want, w.Body.String())
			}
		})
	}
}

func TestParseLimitDirect(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		query string
		want  int64
	}{
		{"", 0},
		{"abc", 0},
		{"42", 42},
		{"-3", -3},
	}
	for _, tt := range tests {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodGet, "/?limit="+tt.query, nil)
		if got := parseLimit(c); got != tt.want {
			t.Errorf("parseLimit(%q) = %d, attendu %d", tt.query, got, tt.want)
		}
	}
}
