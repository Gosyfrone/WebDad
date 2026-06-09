package service

import (
	"testing"

	"github.com/webdad/notification-service/internal/models"
)

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
