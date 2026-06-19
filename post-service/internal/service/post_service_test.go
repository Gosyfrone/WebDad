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

// TestIsModerator : seuls les rôles modérateur et admin ouvrent la corbeille de
// modération (restauration / purge). PURE.
func TestIsModerator(t *testing.T) {
	cases := []struct {
		role string
		want bool
	}{
		{models.RoleUser, false},
		{models.RoleModerator, true},
		{models.RoleAdmin, true},
		{"", false},
		{"Admin", false}, // sensible à la casse (le JWT émet en minuscules)
	}
	for _, tc := range cases {
		t.Run(tc.role, func(t *testing.T) {
			if got := isModerator(tc.role); got != tc.want {
				t.Fatalf("isModerator(%q) = %v, attendu %v", tc.role, got, tc.want)
			}
		})
	}
}

// TestPurgeCutoffs : la borne de préavis précède toujours la borne de purge
// (fenêtre de préavis = avant la purge), et les deux sont dans le passé. PURE.
func TestPurgeCutoffs(t *testing.T) {
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	after := 43800 * time.Hour // ~5 ans
	warn := 720 * time.Hour    // 30 jours

	purge, warnAt := purgeCutoffs(now, after, warn)
	if !purge.Equal(now.Add(-after)) {
		t.Fatalf("purge cutoff = %v, attendu %v", purge, now.Add(-after))
	}
	if !warnAt.After(purge) {
		t.Fatalf("la borne de préavis (%v) doit être postérieure à la borne de purge (%v)", warnAt, purge)
	}
	// Un tweet masqué il y a (after - warn/2) doit être en zone de préavis mais
	// pas encore purgeable : hidden_at > purge cutoff ET hidden_at < warn cutoff.
	hiddenAt := now.Add(-(after - warn/2))
	if !hiddenAt.After(purge) || !hiddenAt.Before(warnAt) {
		t.Fatalf("tweet en préavis mal classé : hidden=%v purge=%v warn=%v", hiddenAt, purge, warnAt)
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
	s := NewPostService(nil, WithBookmarkWindow(5*time.Minute))
	posts, err := s.GetFeed(context.Background(), nil, "", "", "", 20, 0)
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

func TestExtractHashtags(t *testing.T) {
	got := ExtractHashtags("Go #Breezy #breezy #Dev_2026 #123 #école fin#tag")
	want := []string{"breezy", "dev_2026", "école"}
	if len(got) != len(want) {
		t.Fatalf("ExtractHashtags len = %d (%v), attendu %d (%v)", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ExtractHashtags[%d] = %q, attendu %q (liste %v)", i, got[i], want[i], got)
		}
	}
}

// TestReplyAudienceOf : audience vide/absente = everyone (rétrocompat vieux docs).
func TestReplyAudienceOf(t *testing.T) {
	cases := []struct {
		name string
		post *models.Post
		want string
	}{
		{"nil", nil, models.ReplyAudienceEveryone},
		{"vide", &models.Post{}, models.ReplyAudienceEveryone},
		{"everyone", &models.Post{ReplyAudience: models.ReplyAudienceEveryone}, models.ReplyAudienceEveryone},
		{"followers", &models.Post{ReplyAudience: models.ReplyAudienceFollowers}, models.ReplyAudienceFollowers},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := models.ReplyAudienceOf(tc.post); got != tc.want {
				t.Fatalf("ReplyAudienceOf = %q, attendu %q", got, tc.want)
			}
		})
	}
}

// stubFollow implémente followStatusClient pour les tests de la barrière.
type stubFollow struct {
	follows      bool
	blocked      bool
	err          error
	calls        int
	blockedCalls int
}

func (s *stubFollow) IsFollowing(context.Context, string, string) (bool, error) {
	s.calls++
	return s.follows, s.err
}

func (s *stubFollow) HasBlocked(context.Context, string, string) (bool, error) {
	s.blockedCalls++
	return s.blocked, s.err
}

// TestCanReplyTo : barrière « qui peut répondre ». everyone → toujours OK sans
// appel user-service ; followers → auteur/mod/admin bypass, sinon check abonnement.
func TestCanReplyTo(t *testing.T) {
	postEveryone := &models.Post{AuthorID: "author-1", ReplyAudience: models.ReplyAudienceEveryone}
	postFollowers := &models.Post{AuthorID: "author-1", ReplyAudience: models.ReplyAudienceFollowers}

	cases := []struct {
		name      string
		post      *models.Post
		actorID   string
		actorRole string
		follows   bool
		want      bool
		wantCalls int // appels IsFollowing attendus
	}{
		{"everyone tiers", postEveryone, "x", models.RoleUser, false, true, 0},
		{"followers auteur", postFollowers, "author-1", models.RoleUser, false, true, 0},
		{"followers modérateur", postFollowers, "mod-9", models.RoleModerator, false, true, 0},
		{"followers admin", postFollowers, "admin-9", models.RoleAdmin, false, true, 0},
		{"followers abonné", postFollowers, "x", models.RoleUser, true, true, 1},
		{"followers non-abonné", postFollowers, "x", models.RoleUser, false, false, 1},
		{"followers visiteur", postFollowers, "", "", false, false, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			follow := &stubFollow{follows: tc.follows}
			s := NewPostService(nil, WithFollowClient(follow))
			got, err := s.canReplyTo(context.Background(), tc.post, tc.actorID, tc.actorRole)
			if err != nil {
				t.Fatalf("canReplyTo erreur inattendue : %v", err)
			}
			if got != tc.want {
				t.Fatalf("canReplyTo = %v, attendu %v", got, tc.want)
			}
			if follow.calls != tc.wantCalls {
				t.Fatalf("IsFollowing appelé %d fois, attendu %d", follow.calls, tc.wantCalls)
			}
		})
	}
}

func TestTrendQueryMatching(t *testing.T) {
	query := normalizeTrendQuery(" #Br")
	if query != "br" {
		t.Fatalf("normalizeTrendQuery = %q, attendu br", query)
	}
	if !matchesTrendQuery("breezy", query) {
		t.Fatal("breezy doit matcher le préfixe br")
	}
	if matchesTrendQuery("dev", query) {
		t.Fatal("dev ne doit pas matcher le préfixe br")
	}
	if !matchesTrendQuery("dev", "") {
		t.Fatal("une requête vide doit tout matcher")
	}
}
