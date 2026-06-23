package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/webdad/post-service/internal/service"
)

// newNoClaimsCtx fabrique un *gin.Context SANS claims posés (le middleware JWT
// n'a pas tourné). Permet de couvrir la garde `claims, ok := ClaimsFrom(c); if
// !ok` de chaque handler protégé — branche inatteignable via le routeur réel,
// où le middleware abort en 401 AVANT le handler.
func newNoClaimsCtx() (*httptest.ResponseRecorder, *gin.Context) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
	c.Request.Header.Set("Content-Type", "application/json")
	return w, c
}

// TestHandlers_ClaimsAbsents_401 : chaque handler protégé répond 401 quand les
// claims sont absents du contexte (garde de défense en profondeur).
func TestHandlers_ClaimsAbsents_401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := service.NewPostService(&emptyRepo{})
	ph := NewPostHandler(svc, "t")
	lh := NewLikeHandler(svc, "t")
	ch := NewCommentHandler(svc, "t")
	bh := NewBookmarkHandler(svc, "t")

	cases := []struct {
		name string
		fn   gin.HandlerFunc
	}{
		{"CreatePost", ph.CreatePost},
		{"VotePoll", ph.VotePoll},
		{"ClosePoll", ph.ClosePoll},
		{"UpdatePost", ph.UpdatePost},
		{"SetNsfw", ph.SetNsfw},
		{"PinPost", ph.PinPost},
		{"UnpinPost", ph.UnpinPost},
		{"RepostPost", ph.RepostPost},
		{"UnrepostPost", ph.UnrepostPost},
		{"RepostedByMe", ph.RepostedByMe},
		{"DeletePost", ph.DeletePost},
		{"ListHidden", ph.ListHidden},
		{"RestorePost", ph.RestorePost},
		{"PurgePost", ph.PurgePost},
		{"PurgeUserData", ph.PurgeUserData},
		{"LikePost", lh.LikePost},
		{"UnlikePost", lh.UnlikePost},
		{"LikedByMe", lh.LikedByMe},
		{"CreatPostComment", ch.CreatPostComment},
		{"DeletePostComment", ch.DeletePostComment},
		{"LikeComment", ch.LikeComment},
		{"UnlikeComment", ch.UnlikeComment},
		{"ListCollections", bh.ListCollections},
		{"CreateCollection", bh.CreateCollection},
		{"RenameCollection", bh.RenameCollection},
		{"DeleteCollection", bh.DeleteCollection},
		{"ListCollectionPosts", bh.ListCollectionPosts},
		{"ListAll", bh.ListAll},
		{"Bookmark", bh.Bookmark},
		{"Unbookmark", bh.Unbookmark},
		{"PostCollections", bh.PostCollections},
		{"BookmarkedByMe", bh.BookmarkedByMe},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, c := newNoClaimsCtx()
			tc.fn(c)
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("%s sans claims = %d, attendu 401", tc.name, w.Code)
			}
		})
	}
}

// ─── Routes publiques de stats — chemins erreur & succès ─────────────────────

// TestPublicStats_Routes couvre les branches restantes des lectures publiques :
//   - succès (repo vide → 200) sur PostStats / CommentStats avec ids non vides ;
//   - erreur repo → 500 (branche respondPostError) ;
//   - ListPosts avec viewer authentifié (la branche `if claims, ok` vraie).
func TestPublicStats_Routes(t *testing.T) {
	tok := makePostToken(t, "user")
	cases := []struct {
		name   string
		router *gin.Engine
		path   string
		authed bool
		want   int
	}{
		{"post_stats_ok", newEmptyRepoRouter(t), "/posts/stats?ids=abc", false, http.StatusOK},
		{"post_stats_err", newErrRepoRouter(t), "/posts/stats?ids=abc", false, http.StatusInternalServerError},
		{"comment_stats_ok", newEmptyRepoRouter(t), "/posts/comments/stats?ids=abc", false, http.StatusOK},
		{"comment_stats_err", newErrRepoRouter(t), "/posts/comments/stats?ids=abc", false, http.StatusInternalServerError},
		{"list_posts_viewer", newErrRepoRouter(t), "/posts", true, http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			if tc.authed {
				req.Header.Set("Authorization", "Bearer "+tok)
			}
			tc.router.ServeHTTP(w, req)
			if w.Code != tc.want {
				t.Fatalf("GET %s = %d, attendu %d", tc.path, w.Code, tc.want)
			}
		})
	}
}

// ─── respondPostError — branches sentinelles spécifiques ──────────────────────

// TestRespondPostError_Mapping couvre les branches de mapping erreur→HTTP qui ne
// sont pas atteintes par les chemins repo (profil privé → 403, dépendance
// inter-services indisponible → 503).
func TestRespondPostError_Mapping(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"profil_prive", service.ErrPrivateProfil, http.StatusForbidden},
		{"dependance_indispo", service.ErrDependencyUnavailable, http.StatusServiceUnavailable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			respondPostError(c, tc.err)
			if w.Code != tc.want {
				t.Fatalf("respondPostError(%v) = %d, attendu %d", tc.err, w.Code, tc.want)
			}
		})
	}
}
