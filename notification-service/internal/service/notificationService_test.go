package service

import (
	"context"
	"testing"

	"github.com/webdad/notification-service/internal/models"
)

type capturePublisher struct {
	userIDs []string
	event   any
}

func (p *capturePublisher) Publish(userIDs []string, event any) {
	p.userIDs = userIDs
	p.event = event
}

func TestGroupKeyFor(t *testing.T) {
	cases := []struct {
		name string
		ev   models.Event
		want string
		ok   bool
	}{
		{"like agrège par post", models.Event{Type: models.TypeLike, PostID: "p1"}, "like:p1", true},
		{"comment agrège par post", models.Event{Type: models.TypeComment, PostID: "p1"}, "comment:p1", true},
		{"reply agrège par commentaire", models.Event{Type: models.TypeReply, CommentID: "c1", PostID: "p1"}, "reply:c1", true},
		{"mention dans un commentaire → source = commentaire", models.Event{Type: models.TypeMention, CommentID: "c9", PostID: "p1"}, "mention:c9", true},
		{"mention dans un post → source = post", models.Event{Type: models.TypeMention, PostID: "p1"}, "mention:p1", true},
		{"like sans post → invalide", models.Event{Type: models.TypeLike}, "", false},
		{"reply sans commentaire → invalide", models.Event{Type: models.TypeReply, PostID: "p1"}, "", false},
		{"repost agrège par post original", models.Event{Type: models.TypeRepost, PostID: "p1"}, "repost:p1", true},
		{"quote agrège par post citant", models.Event{Type: models.TypeQuote, PostID: "p2"}, "quote:p2", true},
		{"mention en message agrège par conversation", models.Event{Type: models.TypeMessageMention, ConversationID: "cv1"}, "message_mention:cv1", true},
		{"mention en message sans conversation → invalide", models.Event{Type: models.TypeMessageMention}, "", false},
		{"type inconnu → invalide", models.Event{Type: "bogus"}, "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := groupKeyFor(tc.ev)
			if got != tc.want || ok != tc.ok {
				t.Errorf("groupKeyFor(%+v) = (%q, %v) ; attendu (%q, %v)", tc.ev, got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestNormalizeHandle(t *testing.T) {
	cases := map[string]string{
		"@bob":    "bob",
		"  @bob ": "bob",
		"alice":   "alice",
		"@":       "",
		"   ":     "",
		"":        "",
	}
	for in, want := range cases {
		if got := normalizeHandle(in); got != want {
			t.Errorf("normalizeHandle(%q) = %q ; attendu %q", in, got, want)
		}
	}
}

func TestClampLimit(t *testing.T) {
	cases := []struct{ in, want int64 }{
		{0, DefaultLimit},
		{-5, DefaultLimit},
		{10, 10},
		{MaxLimit, MaxLimit},
		{MaxLimit + 1, MaxLimit},
	}
	for _, tc := range cases {
		if got := clampLimit(tc.in); got != tc.want {
			t.Errorf("clampLimit(%d) = %d ; attendu %d", tc.in, got, tc.want)
		}
	}
}

func TestHandleEventPublishesFollowRequestDecision(t *testing.T) {
	pub := &capturePublisher{}
	svc := NewNotificationService(nil, pub, nil)

	err := svc.HandleEvent(context.Background(), models.Event{
		Type:        models.EventFollowRequestAccepted,
		ActorID:     "private-user",
		RecipientID: "requester",
	})
	if err != nil {
		t.Fatalf("HandleEvent() erreur inattendue: %v", err)
	}
	if len(pub.userIDs) != 1 || pub.userIDs[0] != "requester" {
		t.Fatalf("destinataires = %#v ; attendu requester", pub.userIDs)
	}
	payload, ok := pub.event.(map[string]any)
	if !ok {
		t.Fatalf("payload type = %T ; attendu map[string]any", pub.event)
	}
	if payload["type"] != "follow_request_decision" {
		t.Fatalf("type = %#v ; attendu follow_request_decision", payload["type"])
	}
	data, ok := payload["data"].(map[string]string)
	if !ok {
		t.Fatalf("data type = %T ; attendu map[string]string", payload["data"])
	}
	if data["actor_id"] != "private-user" || data["status"] != "accepted" {
		t.Fatalf("data = %#v ; attendu actor/status accepted", data)
	}
}
