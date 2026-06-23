package repository

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/webdad/post-service/internal/models"
)

// Tests PURS des helpers du repository (aucun Mongo requis).

func TestPostHasHashtag(t *testing.T) {
	p := &models.Post{Hashtags: []string{"go", "test"}}
	if !postHasHashtag(p, "go") {
		t.Fatal("hashtag présent attendu true")
	}
	if postHasHashtag(p, "absent") {
		t.Fatal("hashtag absent attendu false")
	}
	if postHasHashtag(nil, "x") {
		t.Fatal("post nil attendu false")
	}
}

func TestProfileSortTime(t *testing.T) {
	created := time.Now().Add(-time.Hour)
	reposted := time.Now()
	if got := profileSortTime(models.Post{CreatedAt: created}); !got.Equal(created) {
		t.Fatalf("sans repost → created_at, got %v", got)
	}
	if got := profileSortTime(models.Post{CreatedAt: created, RepostedAt: &reposted}); !got.Equal(reposted) {
		t.Fatalf("avec repost → reposted_at, got %v", got)
	}
}

func TestSortProfilePosts(t *testing.T) {
	now := time.Now()
	older := now.Add(-2 * time.Hour)
	pinnedAt := now
	posts := []models.Post{
		{Content: "ancien", CreatedAt: older},
		{Content: "récent", CreatedAt: now},
		{Content: "épinglé", CreatedAt: older, PinnedAt: &pinnedAt},
	}
	sortProfilePosts(posts)
	if posts[0].Content != "épinglé" {
		t.Fatalf("l'épinglé doit être en tête, got %q", posts[0].Content)
	}
	if posts[1].Content != "récent" || posts[2].Content != "ancien" {
		t.Fatalf("ordre anté-chronologique attendu ensuite, got %q puis %q", posts[1].Content, posts[2].Content)
	}

	// Épinglé EN TÊTE de l'entrée : force le comparateur dans l'autre sens
	// (i non épinglé vs j épinglé → false), couvrant la branche symétrique.
	posts2 := []models.Post{
		{Content: "épinglé", CreatedAt: older, PinnedAt: &pinnedAt},
		{Content: "récent", CreatedAt: now},
	}
	sortProfilePosts(posts2)
	if posts2[0].Content != "épinglé" {
		t.Fatalf("l'épinglé doit rester en tête, got %q", posts2[0].Content)
	}
}

func TestNotHidden(t *testing.T) {
	f := notHidden()
	if f["is_hidden"] == nil || f["auto_hidden"] == nil {
		t.Fatalf("notHidden doit exclure is_hidden et auto_hidden, got %v", f)
	}
}

func TestWithHashtag(t *testing.T) {
	if f := withHashtag(bson.M{}, "go"); f["hashtags"] != "go" {
		t.Fatalf("withHashtag doit poser le filtre, got %v", f)
	}
	if f := withHashtag(bson.M{}, ""); f["hashtags"] != nil {
		t.Fatalf("withHashtag(\"\") ne doit rien poser, got %v", f)
	}
}

func TestWithAnyHashtag(t *testing.T) {
	if f := withAnyHashtag(bson.M{}); f["hashtags"] == nil {
		t.Fatalf("withAnyHashtag doit poser un filtre d'existence, got %v", f)
	}
}

func TestPostIDsFromDocs(t *testing.T) {
	docs := []bson.M{{"post_id": "a"}, {"post_id": "b"}, {"autre": "ignoré"}}
	ids := postIDsFromDocs(docs)
	if len(ids) != 2 || ids[0] != "a" || ids[1] != "b" {
		t.Fatalf("postIDsFromDocs = %v, attendu [a b]", ids)
	}
}
