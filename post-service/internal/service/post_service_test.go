package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/webdad/post-service/internal/models"
)

// TestCanModify : un post n'est modifiable que par son auteur, un modérateur
// ou un administrateur. Fonction PURE.
func TestCanModify(t *testing.T) {
	post := &models.Post{AuthorID: "author-1"}

	cases := []struct {
		name      string
		actorID   string
		actorRole string
		want      bool
	}{
		{"auteur", "author-1", models.RoleUser, true},
		{"autre utilisateur", "author-2", models.RoleUser, false},
		{"modérateur tiers", "mod-9", models.RoleModerator, true},
		{"admin tiers", "admin-9", models.RoleAdmin, true},
		{"non-auteur sans rôle", "x", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := canModify(post, tc.actorID, tc.actorRole); got != tc.want {
				t.Fatalf("canModify(%s,%s) = %v, attendu %v", tc.actorID, tc.actorRole, got, tc.want)
			}
		})
	}
}

// TestCanModifyNilPost : un post nil n'est jamais modifiable (pas de panic).
func TestCanModifyNilPost(t *testing.T) {
	if canModify(nil, "anyone", models.RoleAdmin) {
		t.Fatal("canModify(nil, …) doit être false")
	}
}

// TestCanPin : l'épinglage est réservé à l'auteur, même si l'acteur tiers est
// modérateur/admin (personnalisation du profil, pas modération).
func TestCanPin(t *testing.T) {
	post := &models.Post{AuthorID: "author-1"}
	if !canPin(post, "author-1") {
		t.Fatal("l'auteur doit pouvoir épingler son post")
	}
	if canPin(post, "admin-1") {
		t.Fatal("un tiers ne doit pas pouvoir épingler le post d'un autre")
	}
	if canPin(nil, "author-1") {
		t.Fatal("canPin(nil, …) doit être false")
	}
}

// TestCanAct : règle commune posts/commentaires — auteur ou mod/admin. PURE.
func TestCanAct(t *testing.T) {
	cases := []struct {
		name              string
		authorID, actorID string
		actorRole         string
		want              bool
	}{
		{"auteur du commentaire", "a1", "a1", models.RoleUser, true},
		{"tiers sans rôle", "a1", "a2", models.RoleUser, false},
		{"modérateur tiers", "a1", "mod", models.RoleModerator, true},
		{"admin tiers", "a1", "adm", models.RoleAdmin, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := canAct(tc.authorID, tc.actorID, tc.actorRole); got != tc.want {
				t.Fatalf("canAct(%s,%s,%s) = %v, attendu %v", tc.authorID, tc.actorID, tc.actorRole, got, tc.want)
			}
		})
	}
}

// TestResolveParentID : threading 2 niveaux — répondre à une racine garde son
// id, répondre à une réponse rattache à la racine de cette réponse. PURE.
func TestResolveParentID(t *testing.T) {
	root := &models.Comment{} // parent_id vide = racine
	if got := resolveParentID(root, "root-id"); got != "root-id" {
		t.Fatalf("réponse à une racine = %q, attendu \"root-id\"", got)
	}

	reply := &models.Comment{ParentID: "root-id"} // une réponse
	if got := resolveParentID(reply, "reply-id"); got != "root-id" {
		t.Fatalf("réponse à une réponse = %q, attendu \"root-id\" (rattachement racine)", got)
	}
}

// TestWithinSessionWindow : un clic court range automatiquement (sans
// redemander la collection) seulement si l'utilisateur a une dernière collection et
// que son dernier signet est récent (< fenêtre). Fonction PURE.
func TestWithinSessionWindow(t *testing.T) {
	now := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)
	window := 5 * time.Minute
	recent := now.Add(-2 * time.Minute) // dans la fenêtre
	stale := now.Add(-10 * time.Minute) // hors fenêtre

	cases := []struct {
		name  string
		prefs *models.BookmarkPrefs
		win   time.Duration
		want  bool
	}{
		{"nil prefs (1er signet)", nil, window, false},
		{"jamais signé (date nil)", &models.BookmarkPrefs{LastCollectionID: "c1"}, window, false},
		{"sans dernière collection", &models.BookmarkPrefs{LastBookmarkAt: &recent}, window, false},
		{"rafale active", &models.BookmarkPrefs{LastCollectionID: "c1", LastBookmarkAt: &recent}, window, true},
		{"fenêtre expirée", &models.BookmarkPrefs{LastCollectionID: "c1", LastBookmarkAt: &stale}, window, false},
		{"auto désactivé (window 0)", &models.BookmarkPrefs{LastCollectionID: "c1", LastBookmarkAt: &recent}, 0, false},
	}
	for _, tc := range cases {
		if got := withinSessionWindow(tc.prefs, tc.win, now); got != tc.want {
			t.Fatalf("%s : withinSessionWindow = %v, attendu %v", tc.name, got, tc.want)
		}
	}
}

// TestGetFeedEmpty : sans aucun id suivi, GetFeed renvoie une liste vide SANS
// toucher au dépôt (court-circuit) — d'où le repo nil sans panic.
func TestGetFeedEmpty(t *testing.T) {
	s := NewPostService(nil, 5*time.Minute)
	posts, err := s.GetFeed(context.Background(), nil, 20, 0)
	if err != nil {
		t.Fatalf("GetFeed(nil) erreur inattendue : %v", err)
	}
	if len(posts) != 0 {
		t.Fatalf("GetFeed(nil) = %d posts, attendu 0", len(posts))
	}
}

// TestWithoutProfilePins : les feeds ne doivent pas divulguer l'état
// d'épinglage d'un profil, et la liste source ne doit pas être mutée.
func TestWithoutProfilePins(t *testing.T) {
	pinnedAt := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)
	posts := []models.Post{
		{AuthorID: "author-1", PinnedAt: &pinnedAt},
		{AuthorID: "author-2"},
	}

	cleaned := withoutProfilePins(posts)
	if cleaned[0].PinnedAt != nil {
		t.Fatal("un post de feed ne doit pas exposer pinned_at")
	}
	if posts[0].PinnedAt == nil {
		t.Fatal("withoutProfilePins ne doit pas muter la liste source")
	}
}

// TestParseID : un hex valide passe, le reste donne ErrInvalidID.
func TestParseID(t *testing.T) {
	if _, err := parseID("507f1f77bcf86cd799439011"); err != nil {
		t.Fatalf("ObjectID valide refusé : %v", err)
	}
	for _, bad := range []string{"", "xyz", "123"} {
		if _, err := parseID(bad); !errors.Is(err, ErrInvalidID) {
			t.Fatalf("parseID(%q) = %v, attendu ErrInvalidID", bad, err)
		}
	}
}

// TestClampLimit : défaut quand <=0, plafond à MaxLimit, valeur valide gardée.
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
			t.Fatalf("clampLimit(%d) = %d, attendu %d", tc.in, got, tc.want)
		}
	}
}

// TestClampOffset : un offset négatif est ramené à 0.
func TestClampOffset(t *testing.T) {
	if got := clampOffset(-1); got != 0 {
		t.Fatalf("clampOffset(-1) = %d, attendu 0", got)
	}
	if got := clampOffset(5); got != 5 {
		t.Fatalf("clampOffset(5) = %d, attendu 5", got)
	}
}
