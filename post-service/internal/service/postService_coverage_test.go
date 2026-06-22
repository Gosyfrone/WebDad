package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/webdad/post-service/internal/models"
)

// Ce fichier vise les branches d'erreur (dépendances repo/clients en échec) et
// quelques chemins logiques restés non couverts, pour pousser le package service
// au-delà de 95 %. Il s'appuie sur les helpers de fakerepo_test.go / postService_repo_test.go.

// commentWith renvoie un commentaire minimal avec un id valide.
func commentWith(postID, authorID string) *models.Comment {
	return &models.Comment{ID: newOID(), PostID: postID, AuthorID: authorID}
}

// getOK renvoie un fakeRepo dont Get renvoie toujours le post fourni.
func getOK(p *models.Post) func(context.Context, bson.ObjectID) (*models.Post, error) {
	return func(context.Context, bson.ObjectID) (*models.Post, error) { return p, nil }
}

// ─── canReadAuthor / hasBlocked ──────────────────────────────────────────────

func TestCanReadAuthor_SelfAlwaysAllowed(t *testing.T) {
	s := NewPostService(&fakeRepo{})
	ok, err := s.canReadAuthor(bg(), "u1", "u1")
	if err != nil || !ok {
		t.Fatalf("auteur == visiteur doit toujours être lisible: %v / %v", ok, err)
	}
}

func TestCanReadAuthor_BlockedHidesAuthor(t *testing.T) {
	s := NewPostService(&fakeRepo{}, WithFollowClient(&stubFollow{blocked: true}))
	ok, err := s.canReadAuthor(bg(), "viewer", "author")
	if err != nil || ok {
		t.Fatalf("un auteur bloqué doit être invisible: %v / %v", ok, err)
	}
}

func TestCanReadAuthor_HasBlockedError(t *testing.T) {
	s := NewPostService(&fakeRepo{}, WithFollowClient(&stubFollow{err: errBoom}))
	if _, err := s.canReadAuthor(bg(), "viewer", "author"); !errors.Is(err, ErrDependencyUnavailable) {
		t.Fatalf("erreur de blocage doit remonter ErrDependencyUnavailable, got %v", err)
	}
}

func TestCanReadAuthor_NoProfilClientAllows(t *testing.T) {
	s := NewPostService(&fakeRepo{})
	ok, err := s.canReadAuthor(bg(), "viewer", "author")
	if err != nil || !ok {
		t.Fatalf("sans profilClient, lisible par défaut: %v / %v", ok, err)
	}
}

func TestCanReadAuthor_VisibilityError(t *testing.T) {
	s := NewPostService(&fakeRepo{}, WithProfilClient(&stubProfil{visErr: errBoom}))
	if _, err := s.canReadAuthor(bg(), "viewer", "author"); !errors.Is(err, ErrDependencyUnavailable) {
		t.Fatalf("erreur visibilité → ErrDependencyUnavailable, got %v", err)
	}
}

func TestCanReadAuthor_PrivateAnonymous(t *testing.T) {
	s := NewPostService(&fakeRepo{}, WithProfilClient(&stubProfil{visibility: "private"}))
	ok, err := s.canReadAuthor(bg(), "", "author")
	if err != nil || ok {
		t.Fatalf("profil privé + visiteur anonyme → invisible: %v / %v", ok, err)
	}
}

func TestCanReadAuthor_PrivateFollowerAllowed(t *testing.T) {
	s := NewPostService(&fakeRepo{},
		WithProfilClient(&stubProfil{visibility: "private"}),
		WithFollowClient(&stubFollow{follows: true}))
	ok, err := s.canReadAuthor(bg(), "viewer", "author")
	if err != nil || !ok {
		t.Fatalf("profil privé + abonné → lisible: %v / %v", ok, err)
	}
}

func TestCanReadAuthor_PrivateFollowError(t *testing.T) {
	s := NewPostService(&fakeRepo{},
		WithProfilClient(&stubProfil{visibility: "private"}),
		WithFollowClient(&stubFollow{err: errBoom}))
	if _, err := s.canReadAuthor(bg(), "viewer", "author"); !errors.Is(err, ErrDependencyUnavailable) {
		t.Fatalf("erreur abonnement → ErrDependencyUnavailable, got %v", err)
	}
}

// ─── GetPosts ────────────────────────────────────────────────────────────────

func TestGetPosts_HashtagAny(t *testing.T) {
	post := postWith("a1")
	repo := &fakeRepo{
		fnGetAllWithHashtags: func(context.Context, int64, int64) ([]models.Post, error) {
			return []models.Post{*post}, nil
		},
	}
	s := NewPostService(repo)
	posts, err := s.GetPosts(bg(), "", "", "", true, 0, 0)
	if err != nil || len(posts) != 1 {
		t.Fatalf("GetPosts hashtagAny: %v / %d", err, len(posts))
	}
}

func TestGetPosts_FetchError(t *testing.T) {
	repo := &fakeRepo{fnGetAll: func(context.Context, int64, int64) ([]models.Post, error) { return nil, errBoom }}
	s := NewPostService(repo)
	if _, err := s.GetPosts(bg(), "", "", "", false, 0, 0); !errors.Is(err, errBoom) {
		t.Fatalf("erreur fetch doit remonter, got %v", err)
	}
}

// ─── visiblePage : filtrage, pagination, erreurs ─────────────────────────────

func TestVisiblePage_SkipsBlockedAndPaginates(t *testing.T) {
	visible := postWith("ok")
	hidden := postWith("blocked-author")
	calls := 0
	repo := &fakeRepo{
		fnGetAll: func(_ context.Context, _, off int64) ([]models.Post, error) {
			calls++
			if calls == 1 {
				return []models.Post{*hidden, *visible}, nil
			}
			return nil, nil // 2e page vide → arrêt
		},
	}
	// canReadAuthor: l'auteur "blocked-author" est privé sans followClient → invisible.
	s := NewPostService(repo, WithProfilClient(profilStub(func(_ context.Context, id string) (string, error) {
		if id == "blocked-author" {
			return "private", nil
		}
		return "public", nil
	})))
	posts, err := s.GetPosts(bg(), "viewer", "", "", false, 10, 0)
	if err != nil {
		t.Fatalf("GetPosts: %v", err)
	}
	if len(posts) != 1 || posts[0].AuthorID != "ok" {
		t.Fatalf("seul l'auteur public doit être visible, got %+v", posts)
	}
}

func TestVisiblePage_CanReadAuthorError(t *testing.T) {
	repo := &fakeRepo{fnGetAll: func(context.Context, int64, int64) ([]models.Post, error) {
		return []models.Post{*postWith("a1")}, nil
	}}
	s := NewPostService(repo, WithProfilClient(&stubProfil{visErr: errBoom}))
	if _, err := s.GetPosts(bg(), "viewer", "", "", false, 10, 0); !errors.Is(err, ErrDependencyUnavailable) {
		t.Fatalf("erreur de visibilité doit remonter, got %v", err)
	}
}

func TestVisiblePage_OffsetSkips(t *testing.T) {
	a := postWith("a1")
	b := postWith("a1")
	repo := &fakeRepo{fnGetAll: func(_ context.Context, _, off int64) ([]models.Post, error) {
		if off == 0 {
			return []models.Post{*a, *b}, nil
		}
		return nil, nil
	}}
	s := NewPostService(repo)
	posts, err := s.GetPosts(bg(), "", "", "", false, 10, 1)
	if err != nil || len(posts) != 1 {
		t.Fatalf("offset=1 doit sauter le premier visible: %v / %d", err, len(posts))
	}
}

// ─── TrendingHashtags ────────────────────────────────────────────────────────

func TestTrendingHashtags_GetAllError(t *testing.T) {
	repo := &fakeRepo{fnGetAll: func(context.Context, int64, int64) ([]models.Post, error) { return nil, errBoom }}
	s := NewPostService(repo)
	if _, err := s.TrendingHashtags(bg(), "viewer", "", 5); !errors.Is(err, errBoom) {
		t.Fatalf("erreur GetAll doit remonter, got %v", err)
	}
}

func TestTrendingHashtags_CanReadAuthorError(t *testing.T) {
	post := postWith("a1")
	post.Hashtags = []string{"go"}
	repo := &fakeRepo{fnGetAll: func(_ context.Context, _, off int64) ([]models.Post, error) {
		if off == 0 {
			return []models.Post{*post}, nil
		}
		return nil, nil
	}}
	s := NewPostService(repo, WithProfilClient(&stubProfil{visErr: errBoom}))
	if _, err := s.TrendingHashtags(bg(), "viewer", "", 5); !errors.Is(err, ErrDependencyUnavailable) {
		t.Fatalf("erreur visibilité doit remonter, got %v", err)
	}
}

func TestTrendingHashtags_FiltersBlockedAndQuery(t *testing.T) {
	pub := postWith("pub")
	pub.Hashtags = []string{"golang", "rust"}
	priv := postWith("priv")
	priv.Hashtags = []string{"golang"}
	repo := &fakeRepo{fnGetAll: func(_ context.Context, _, off int64) ([]models.Post, error) {
		if off == 0 {
			return []models.Post{*pub, *priv}, nil
		}
		return nil, nil
	}}
	s := NewPostService(repo, WithProfilClient(profilStub(func(_ context.Context, id string) (string, error) {
		if id == "priv" {
			return "private", nil
		}
		return "public", nil
	})))
	trends, err := s.TrendingHashtags(bg(), "viewer", "go", 5)
	if err != nil {
		t.Fatalf("TrendingHashtags: %v", err)
	}
	// "golang" du post privé est exclu ; "rust" ne matche pas le préfixe "go".
	if len(trends) != 1 || trends[0].Tag != "golang" || trends[0].Count != 1 {
		t.Fatalf("tendances inattendues: %+v", trends)
	}
}

// ─── VotePoll : branches d'erreur ────────────────────────────────────────────

func pollPostAud(authorID string, audience string) *models.Post {
	p := postWith(authorID)
	p.Poll = &models.Poll{
		Audience: audience,
		EndsAt:   time.Now().Add(time.Hour),
		Choices:  []models.PollChoice{{ID: "c1"}, {ID: "c2"}},
	}
	return p
}

func TestVotePoll_GetError(t *testing.T) {
	repo := &fakeRepo{fnGet: func(context.Context, bson.ObjectID) (*models.Post, error) { return nil, mongo.ErrNoDocuments }}
	s := NewPostService(repo)
	if _, err := s.VotePoll(bg(), newOID().Hex(), "u1", "c1"); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("attendu ErrPostNotFound, got %v", err)
	}
}

func TestVotePoll_CanReadAuthorError(t *testing.T) {
	p := pollPostAud("author", models.PollAudienceEveryone)
	repo := &fakeRepo{fnGet: getOK(p)}
	s := NewPostService(repo, WithProfilClient(&stubProfil{visErr: errBoom}))
	if _, err := s.VotePoll(bg(), p.ID.Hex(), "viewer", "c1"); !errors.Is(err, ErrDependencyUnavailable) {
		t.Fatalf("erreur visibilité doit remonter, got %v", err)
	}
}

func TestVotePoll_PrivateForbidden(t *testing.T) {
	p := pollPostAud("author", models.PollAudienceEveryone)
	repo := &fakeRepo{fnGet: getOK(p)}
	s := NewPostService(repo, WithProfilClient(&stubProfil{visibility: "private"}))
	if _, err := s.VotePoll(bg(), p.ID.Hex(), "viewer", "c1"); !errors.Is(err, ErrPrivateProfil) {
		t.Fatalf("profil privé → ErrPrivateProfil, got %v", err)
	}
}

func TestVotePoll_FollowersNoClient(t *testing.T) {
	p := pollPostAud("author", models.PollAudienceFollowers)
	repo := &fakeRepo{fnGet: getOK(p)}
	s := NewPostService(repo)
	if _, err := s.VotePoll(bg(), p.ID.Hex(), "viewer", "c1"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("sondage followers sans followClient → ErrForbidden, got %v", err)
	}
}

func TestVotePoll_FollowersIsFollowingError(t *testing.T) {
	p := pollPostAud("author", models.PollAudienceFollowers)
	repo := &fakeRepo{fnGet: getOK(p)}
	s := NewPostService(repo, WithFollowClient(&stubFollow{err: errBoom}))
	if _, err := s.VotePoll(bg(), p.ID.Hex(), "viewer", "c1"); !errors.Is(err, ErrDependencyUnavailable) {
		t.Fatalf("erreur abonnement → ErrDependencyUnavailable, got %v", err)
	}
}

func TestVotePoll_AddVoteError(t *testing.T) {
	p := pollPostAud("author", models.PollAudienceEveryone)
	repo := &fakeRepo{
		fnGet:         getOK(p),
		fnAddPollVote: func(context.Context, string, string, string) (bool, error) { return false, errBoom },
	}
	s := NewPostService(repo)
	if _, err := s.VotePoll(bg(), p.ID.Hex(), "author", "c1"); !errors.Is(err, errBoom) {
		t.Fatalf("erreur AddPollVote doit remonter, got %v", err)
	}
}

func TestVotePoll_IncChoiceError(t *testing.T) {
	p := pollPostAud("author", models.PollAudienceEveryone)
	repo := &fakeRepo{
		fnGet:           getOK(p),
		fnAddPollVote:   func(context.Context, string, string, string) (bool, error) { return true, nil },
		fnIncPollChoice: func(context.Context, bson.ObjectID, string) (*models.Post, error) { return nil, mongo.ErrNoDocuments },
	}
	s := NewPostService(repo)
	if _, err := s.VotePoll(bg(), p.ID.Hex(), "author", "c1"); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("erreur IncPollChoice → ErrPostNotFound, got %v", err)
	}
}

// ─── ClosePoll : branches d'erreur ───────────────────────────────────────────

func TestClosePoll_GetError(t *testing.T) {
	repo := &fakeRepo{fnGet: func(context.Context, bson.ObjectID) (*models.Post, error) { return nil, mongo.ErrNoDocuments }}
	s := NewPostService(repo)
	if _, err := s.ClosePoll(bg(), newOID().Hex(), "u1"); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("attendu ErrPostNotFound, got %v", err)
	}
}

func TestClosePoll_NoPoll(t *testing.T) {
	repo := &fakeRepo{fnGet: getOK(postWith("author"))}
	s := NewPostService(repo)
	if _, err := s.ClosePoll(bg(), newOID().Hex(), "author"); !errors.Is(err, ErrInvalidPoll) {
		t.Fatalf("post sans sondage → ErrInvalidPoll, got %v", err)
	}
}

func TestClosePoll_NotAuthor(t *testing.T) {
	p := pollPostAud("author", models.PollAudienceEveryone)
	repo := &fakeRepo{fnGet: getOK(p)}
	s := NewPostService(repo)
	if _, err := s.ClosePoll(bg(), p.ID.Hex(), "autre"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("seul l'auteur ferme le sondage → ErrForbidden, got %v", err)
	}
}

func TestClosePoll_RepoError(t *testing.T) {
	p := pollPostAud("author", models.PollAudienceEveryone)
	repo := &fakeRepo{
		fnGet: getOK(p),
		fnClosePoll: func(context.Context, bson.ObjectID, time.Time) (*models.Post, error) {
			return nil, mongo.ErrNoDocuments
		},
	}
	s := NewPostService(repo)
	if _, err := s.ClosePoll(bg(), p.ID.Hex(), "author"); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("erreur ClosePoll → ErrPostNotFound, got %v", err)
	}
}

// ─── LikePost / UnlikePost ───────────────────────────────────────────────────

func TestLikePost_InvalidID(t *testing.T) {
	s := NewPostService(&fakeRepo{})
	if _, err := s.LikePost(bg(), "bad", "u1"); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("attendu ErrInvalidID, got %v", err)
	}
}

func TestLikePost_GetError(t *testing.T) {
	repo := &fakeRepo{fnGet: func(context.Context, bson.ObjectID) (*models.Post, error) { return nil, mongo.ErrNoDocuments }}
	s := NewPostService(repo)
	if _, err := s.LikePost(bg(), newOID().Hex(), "u1"); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("attendu ErrPostNotFound, got %v", err)
	}
}

func TestLikePost_AddLikeError(t *testing.T) {
	repo := &fakeRepo{
		fnGet:     getOK(postWith("a1")),
		fnAddLike: func(context.Context, string, string) (bool, error) { return false, errBoom },
	}
	s := NewPostService(repo)
	if _, err := s.LikePost(bg(), newOID().Hex(), "u1"); !errors.Is(err, errBoom) {
		t.Fatalf("erreur AddLike doit remonter, got %v", err)
	}
}

func TestLikePost_IncCounterError(t *testing.T) {
	repo := &fakeRepo{
		fnGet:        getOK(postWith("a1")),
		fnAddLike:    func(context.Context, string, string) (bool, error) { return true, nil },
		fnIncCounter: func(context.Context, bson.ObjectID, string, int32) (*models.Post, error) { return nil, errBoom },
	}
	s := NewPostService(repo)
	if _, err := s.LikePost(bg(), newOID().Hex(), "u1"); !errors.Is(err, errBoom) {
		t.Fatalf("erreur IncCounter doit remonter, got %v", err)
	}
}

func TestUnlikePost_InvalidID(t *testing.T) {
	s := NewPostService(&fakeRepo{})
	if _, err := s.UnlikePost(bg(), "bad", "u1"); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("attendu ErrInvalidID, got %v", err)
	}
}

func TestUnlikePost_GetError(t *testing.T) {
	repo := &fakeRepo{fnGet: func(context.Context, bson.ObjectID) (*models.Post, error) { return nil, mongo.ErrNoDocuments }}
	s := NewPostService(repo)
	if _, err := s.UnlikePost(bg(), newOID().Hex(), "u1"); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("attendu ErrPostNotFound, got %v", err)
	}
}

func TestUnlikePost_RemoveError(t *testing.T) {
	repo := &fakeRepo{
		fnGet:        getOK(postWith("a1")),
		fnRemoveLike: func(context.Context, string, string) (bool, error) { return false, errBoom },
	}
	s := NewPostService(repo)
	if _, err := s.UnlikePost(bg(), newOID().Hex(), "u1"); !errors.Is(err, errBoom) {
		t.Fatalf("erreur RemoveLike doit remonter, got %v", err)
	}
}

func TestUnlikePost_IncCounterError(t *testing.T) {
	repo := &fakeRepo{
		fnGet:        getOK(postWith("a1")),
		fnRemoveLike: func(context.Context, string, string) (bool, error) { return true, nil },
		fnIncCounter: func(context.Context, bson.ObjectID, string, int32) (*models.Post, error) { return nil, errBoom },
	}
	s := NewPostService(repo)
	if _, err := s.UnlikePost(bg(), newOID().Hex(), "u1"); !errors.Is(err, errBoom) {
		t.Fatalf("erreur IncCounter doit remonter, got %v", err)
	}
}

// ─── LikeComment / UnlikeComment ─────────────────────────────────────────────

func TestLikeComment_InvalidPostID(t *testing.T) {
	s := NewPostService(&fakeRepo{})
	if _, err := s.LikeComment(bg(), "bad", newOID().Hex(), "u1"); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("attendu ErrInvalidID (post), got %v", err)
	}
}

func TestLikeComment_InvalidCommentID(t *testing.T) {
	s := NewPostService(&fakeRepo{})
	if _, err := s.LikeComment(bg(), newOID().Hex(), "bad", "u1"); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("attendu ErrInvalidID (commentaire), got %v", err)
	}
}

func TestLikeComment_GetCommentError(t *testing.T) {
	repo := &fakeRepo{fnGetComment: func(context.Context, bson.ObjectID) (*models.Comment, error) { return nil, mongo.ErrNoDocuments }}
	s := NewPostService(repo)
	if _, err := s.LikeComment(bg(), newOID().Hex(), newOID().Hex(), "u1"); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("attendu ErrPostNotFound, got %v", err)
	}
}

func TestLikeComment_AddError(t *testing.T) {
	postID := newOID().Hex()
	c := commentWith(postID, "author")
	repo := &fakeRepo{
		fnGetComment:     func(context.Context, bson.ObjectID) (*models.Comment, error) { return c, nil },
		fnAddCommentLike: func(context.Context, string, string) (bool, error) { return false, errBoom },
	}
	s := NewPostService(repo)
	if _, err := s.LikeComment(bg(), postID, c.ID.Hex(), "u1"); !errors.Is(err, errBoom) {
		t.Fatalf("erreur AddCommentLike doit remonter, got %v", err)
	}
}

func TestLikeComment_Idempotent(t *testing.T) {
	postID := newOID().Hex()
	c := commentWith(postID, "author")
	c.LikesCount = 7
	repo := &fakeRepo{
		fnGetComment:     func(context.Context, bson.ObjectID) (*models.Comment, error) { return c, nil },
		fnAddCommentLike: func(context.Context, string, string) (bool, error) { return false, nil },
	}
	s := NewPostService(repo)
	n, err := s.LikeComment(bg(), postID, c.ID.Hex(), "u1")
	if err != nil || n != 7 {
		t.Fatalf("reliker ne change pas le compteur: %v / %d", err, n)
	}
}

func TestLikeComment_IncError(t *testing.T) {
	postID := newOID().Hex()
	c := commentWith(postID, "author")
	repo := &fakeRepo{
		fnGetComment:        func(context.Context, bson.ObjectID) (*models.Comment, error) { return c, nil },
		fnAddCommentLike:    func(context.Context, string, string) (bool, error) { return true, nil },
		fnIncCommentCounter: func(context.Context, bson.ObjectID, string, int32) (*models.Comment, error) { return nil, errBoom },
	}
	s := NewPostService(repo)
	if _, err := s.LikeComment(bg(), postID, c.ID.Hex(), "u1"); !errors.Is(err, errBoom) {
		t.Fatalf("erreur IncCommentCounter doit remonter, got %v", err)
	}
}

func TestUnlikeComment_InvalidPostID(t *testing.T) {
	s := NewPostService(&fakeRepo{})
	if _, err := s.UnlikeComment(bg(), "bad", newOID().Hex(), "u1"); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("attendu ErrInvalidID (post), got %v", err)
	}
}

func TestUnlikeComment_InvalidCommentID(t *testing.T) {
	s := NewPostService(&fakeRepo{})
	if _, err := s.UnlikeComment(bg(), newOID().Hex(), "bad", "u1"); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("attendu ErrInvalidID (commentaire), got %v", err)
	}
}

func TestUnlikeComment_GetError(t *testing.T) {
	repo := &fakeRepo{fnGetComment: func(context.Context, bson.ObjectID) (*models.Comment, error) { return nil, mongo.ErrNoDocuments }}
	s := NewPostService(repo)
	if _, err := s.UnlikeComment(bg(), newOID().Hex(), newOID().Hex(), "u1"); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("attendu ErrPostNotFound, got %v", err)
	}
}

func TestUnlikeComment_WrongPost(t *testing.T) {
	c := commentWith(newOID().Hex(), "author")
	repo := &fakeRepo{fnGetComment: func(context.Context, bson.ObjectID) (*models.Comment, error) { return c, nil }}
	s := NewPostService(repo)
	if _, err := s.UnlikeComment(bg(), newOID().Hex(), c.ID.Hex(), "u1"); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("commentaire rattaché à un autre post → ErrPostNotFound, got %v", err)
	}
}

func TestUnlikeComment_RemoveError(t *testing.T) {
	postID := newOID().Hex()
	c := commentWith(postID, "author")
	repo := &fakeRepo{
		fnGetComment:        func(context.Context, bson.ObjectID) (*models.Comment, error) { return c, nil },
		fnRemoveCommentLike: func(context.Context, string, string) (bool, error) { return false, errBoom },
	}
	s := NewPostService(repo)
	if _, err := s.UnlikeComment(bg(), postID, c.ID.Hex(), "u1"); !errors.Is(err, errBoom) {
		t.Fatalf("erreur RemoveCommentLike doit remonter, got %v", err)
	}
}

func TestUnlikeComment_IncError(t *testing.T) {
	postID := newOID().Hex()
	c := commentWith(postID, "author")
	repo := &fakeRepo{
		fnGetComment:        func(context.Context, bson.ObjectID) (*models.Comment, error) { return c, nil },
		fnRemoveCommentLike: func(context.Context, string, string) (bool, error) { return true, nil },
		fnIncCommentCounter: func(context.Context, bson.ObjectID, string, int32) (*models.Comment, error) { return nil, errBoom },
	}
	s := NewPostService(repo)
	if _, err := s.UnlikeComment(bg(), postID, c.ID.Hex(), "u1"); !errors.Is(err, errBoom) {
		t.Fatalf("erreur IncCommentCounter doit remonter, got %v", err)
	}
}

// ─── CommentStats ────────────────────────────────────────────────────────────

func TestCommentStats_Truncates(t *testing.T) {
	ids := make([]string, MaxStatsIDs+5)
	for i := range ids {
		ids[i] = newOID().Hex()
	}
	var got int
	repo := &fakeRepo{fnCommentStatsByIDs: func(_ context.Context, oids []bson.ObjectID) ([]models.Comment, error) {
		got = len(oids)
		return nil, nil
	}}
	s := NewPostService(repo)
	if _, err := s.CommentStats(bg(), ids); err != nil {
		t.Fatalf("CommentStats: %v", err)
	}
	if got != MaxStatsIDs {
		t.Fatalf("la liste doit être bornée à %d, got %d", MaxStatsIDs, got)
	}
}

func TestCommentStats_RepoError(t *testing.T) {
	repo := &fakeRepo{fnCommentStatsByIDs: func(context.Context, []bson.ObjectID) ([]models.Comment, error) { return nil, errBoom }}
	s := NewPostService(repo)
	if _, err := s.CommentStats(bg(), []string{newOID().Hex()}); !errors.Is(err, errBoom) {
		t.Fatalf("erreur repo doit remonter, got %v", err)
	}
}

// ─── ListLikedByUser ─────────────────────────────────────────────────────────

func TestListLikedByUser_Blocked(t *testing.T) {
	s := NewPostService(&fakeRepo{}, WithFollowClient(&stubFollow{blocked: true}))
	posts, err := s.ListLikedByUser(bg(), "author", "caller", 0, 0)
	if err != nil || len(posts) != 0 {
		t.Fatalf("caller bloqué → liste vide: %v / %d", err, len(posts))
	}
}

func TestListLikedByUser_BlockedError(t *testing.T) {
	s := NewPostService(&fakeRepo{}, WithFollowClient(&stubFollow{err: errBoom}))
	if _, err := s.ListLikedByUser(bg(), "author", "caller", 0, 0); !errors.Is(err, ErrDependencyUnavailable) {
		t.Fatalf("erreur de blocage doit remonter, got %v", err)
	}
}

func TestListLikedByUser_VisibilityError(t *testing.T) {
	s := NewPostService(&fakeRepo{}, WithProfilClient(&stubProfil{likesErr: errBoom}))
	if _, err := s.ListLikedByUser(bg(), "author", "caller", 0, 0); !errors.Is(err, ErrDependencyUnavailable) {
		t.Fatalf("erreur likes-visibility doit remonter, got %v", err)
	}
}

// ─── RepostPost / UnrepostPost ───────────────────────────────────────────────

func TestRepostPost_InvalidID(t *testing.T) {
	s := NewPostService(&fakeRepo{})
	if _, err := s.RepostPost(bg(), "bad", "u1"); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("attendu ErrInvalidID, got %v", err)
	}
}

func TestRepostPost_GetError(t *testing.T) {
	repo := &fakeRepo{fnGet: func(context.Context, bson.ObjectID) (*models.Post, error) { return nil, mongo.ErrNoDocuments }}
	s := NewPostService(repo)
	if _, err := s.RepostPost(bg(), newOID().Hex(), "u1"); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("attendu ErrPostNotFound, got %v", err)
	}
}

func TestRepostPost_AddError(t *testing.T) {
	repo := &fakeRepo{
		fnGet:       getOK(postWith("a1")),
		fnAddRepost: func(context.Context, string, string) (*models.Repost, bool, error) { return nil, false, errBoom },
	}
	s := NewPostService(repo)
	if _, err := s.RepostPost(bg(), newOID().Hex(), "u1"); !errors.Is(err, errBoom) {
		t.Fatalf("erreur AddRepost doit remonter, got %v", err)
	}
}

func TestRepostPost_IncError(t *testing.T) {
	repo := &fakeRepo{
		fnGet: getOK(postWith("a1")),
		fnAddRepost: func(context.Context, string, string) (*models.Repost, bool, error) {
			return &models.Repost{}, true, nil
		},
		fnIncCounter: func(context.Context, bson.ObjectID, string, int32) (*models.Post, error) { return nil, errBoom },
	}
	s := NewPostService(repo)
	if _, err := s.RepostPost(bg(), newOID().Hex(), "u1"); !errors.Is(err, errBoom) {
		t.Fatalf("erreur IncCounter doit remonter, got %v", err)
	}
}

func TestUnrepostPost_InvalidID(t *testing.T) {
	s := NewPostService(&fakeRepo{})
	if _, err := s.UnrepostPost(bg(), "bad", "u1"); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("attendu ErrInvalidID, got %v", err)
	}
}

func TestUnrepostPost_GetError(t *testing.T) {
	repo := &fakeRepo{fnGet: func(context.Context, bson.ObjectID) (*models.Post, error) { return nil, mongo.ErrNoDocuments }}
	s := NewPostService(repo)
	if _, err := s.UnrepostPost(bg(), newOID().Hex(), "u1"); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("attendu ErrPostNotFound, got %v", err)
	}
}

func TestUnrepostPost_RemoveError(t *testing.T) {
	repo := &fakeRepo{
		fnGet:          getOK(postWith("a1")),
		fnRemoveRepost: func(context.Context, string, string) (bool, error) { return false, errBoom },
	}
	s := NewPostService(repo)
	if _, err := s.UnrepostPost(bg(), newOID().Hex(), "u1"); !errors.Is(err, errBoom) {
		t.Fatalf("erreur RemoveRepost doit remonter, got %v", err)
	}
}

func TestUnrepostPost_IncError(t *testing.T) {
	repo := &fakeRepo{
		fnGet:          getOK(postWith("a1")),
		fnRemoveRepost: func(context.Context, string, string) (bool, error) { return true, nil },
		fnIncCounter:   func(context.Context, bson.ObjectID, string, int32) (*models.Post, error) { return nil, errBoom },
	}
	s := NewPostService(repo)
	if _, err := s.UnrepostPost(bg(), newOID().Hex(), "u1"); !errors.Is(err, errBoom) {
		t.Fatalf("erreur IncCounter doit remonter, got %v", err)
	}
}

// ─── CreateComment : branches d'erreur ───────────────────────────────────────

func TestCreateComment_InvalidPostID(t *testing.T) {
	s := NewPostService(&fakeRepo{})
	if _, err := s.CreateComment(bg(), "bad", "u1", models.RoleUser, "hi", "", nil); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("attendu ErrInvalidID, got %v", err)
	}
}

func TestCreateComment_PostNotFound(t *testing.T) {
	repo := &fakeRepo{fnGet: func(context.Context, bson.ObjectID) (*models.Post, error) { return nil, mongo.ErrNoDocuments }}
	s := NewPostService(repo)
	if _, err := s.CreateComment(bg(), newOID().Hex(), "u1", models.RoleUser, "hi", "", nil); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("attendu ErrPostNotFound, got %v", err)
	}
}

func TestCreateComment_ParentInvalidID(t *testing.T) {
	repo := &fakeRepo{fnGet: getOK(postWith("a1"))}
	s := NewPostService(repo)
	if _, err := s.CreateComment(bg(), newOID().Hex(), "u1", models.RoleUser, "hi", "bad", nil); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("parentID invalide → ErrInvalidID, got %v", err)
	}
}

func TestCreateComment_ParentNotFound(t *testing.T) {
	repo := &fakeRepo{
		fnGet:        getOK(postWith("a1")),
		fnGetComment: func(context.Context, bson.ObjectID) (*models.Comment, error) { return nil, mongo.ErrNoDocuments },
	}
	s := NewPostService(repo)
	if _, err := s.CreateComment(bg(), newOID().Hex(), "u1", models.RoleUser, "hi", newOID().Hex(), nil); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("parent introuvable → ErrPostNotFound, got %v", err)
	}
}

func TestCreateComment_ParentWrongPost(t *testing.T) {
	postID := newOID().Hex()
	parent := commentWith(newOID().Hex(), "root") // rattaché à un AUTRE post
	repo := &fakeRepo{
		fnGet: func(_ context.Context, id bson.ObjectID) (*models.Post, error) {
			return &models.Post{ID: id, AuthorID: "a1", ReplyAudience: models.ReplyAudienceEveryone}, nil
		},
		fnGetComment: func(context.Context, bson.ObjectID) (*models.Comment, error) { return parent, nil },
	}
	s := NewPostService(repo)
	if _, err := s.CreateComment(bg(), postID, "u1", models.RoleUser, "hi", parent.ID.Hex(), nil); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("parent d'un autre post → ErrPostNotFound, got %v", err)
	}
}

func TestCreateComment_AddError(t *testing.T) {
	repo := &fakeRepo{
		fnGet:        getOK(postWith("a1")),
		fnAddComment: func(context.Context, *models.Comment) error { return errBoom },
	}
	s := NewPostService(repo)
	if _, err := s.CreateComment(bg(), newOID().Hex(), "u1", models.RoleUser, "hi", "", nil); !errors.Is(err, errBoom) {
		t.Fatalf("erreur AddComment doit remonter, got %v", err)
	}
}

func TestCreateComment_IncCounterError(t *testing.T) {
	repo := &fakeRepo{
		fnGet:        getOK(postWith("a1")),
		fnAddComment: func(context.Context, *models.Comment) error { return nil },
		fnIncCounter: func(context.Context, bson.ObjectID, string, int32) (*models.Post, error) { return nil, errBoom },
	}
	s := NewPostService(repo)
	if _, err := s.CreateComment(bg(), newOID().Hex(), "u1", models.RoleUser, "hi", "", nil); !errors.Is(err, errBoom) {
		t.Fatalf("erreur IncCounter doit remonter, got %v", err)
	}
}

// CreateComment : réponse à une réponse → commentAuthor résout l'auteur de la racine.
func TestCreateComment_ReplyToReply(t *testing.T) {
	postID := newOID().Hex()
	rootID := newOID()
	mid := commentWith(postID, "mid-author") // réponse intermédiaire
	mid.ParentID = rootID.Hex()
	root := &models.Comment{ID: rootID, PostID: postID, AuthorID: "root-author"}
	repo := &fakeRepo{
		fnGet: func(_ context.Context, id bson.ObjectID) (*models.Post, error) {
			return &models.Post{ID: id, AuthorID: "a1", ReplyAudience: models.ReplyAudienceEveryone}, nil
		},
		fnGetComment: func(_ context.Context, id bson.ObjectID) (*models.Comment, error) {
			if id == rootID {
				return root, nil
			}
			return mid, nil
		},
		fnAddComment:    func(context.Context, *models.Comment) error { return nil },
		fnIncCounter:    func(context.Context, bson.ObjectID, string, int32) (*models.Post, error) { return nil, nil },
		fnIncReplyCount: func(context.Context, bson.ObjectID, int32) error { return nil },
	}
	notif := &capNotifier{}
	s := NewPostService(repo, WithNotifier(notif))
	if _, err := s.CreateComment(bg(), postID, "replier", models.RoleUser, "re", mid.ID.Hex(), nil); err != nil {
		t.Fatalf("CreateComment réponse-à-réponse: %v", err)
	}
	if notif.typeCount("reply") != 1 {
		t.Fatalf("une notif reply attendue, events=%v", notif.events)
	}
}

// ─── commentAuthor / postAuthor ──────────────────────────────────────────────

func TestCommentAuthor_InvalidAndMissing(t *testing.T) {
	s := NewPostService(&fakeRepo{fnGetComment: func(context.Context, bson.ObjectID) (*models.Comment, error) { return nil, errBoom }})
	if a := s.commentAuthor(bg(), "bad"); a != "" {
		t.Fatalf("id invalide → \"\", got %q", a)
	}
	if a := s.commentAuthor(bg(), newOID().Hex()); a != "" {
		t.Fatalf("erreur repo → \"\", got %q", a)
	}
}

func TestPostAuthor_InvalidAndMissing(t *testing.T) {
	s := NewPostService(&fakeRepo{fnGet: func(context.Context, bson.ObjectID) (*models.Post, error) { return nil, errBoom }})
	if a := s.postAuthor(bg(), "bad"); a != "" {
		t.Fatalf("id invalide → \"\", got %q", a)
	}
	if a := s.postAuthor(bg(), newOID().Hex()); a != "" {
		t.Fatalf("erreur repo → \"\", got %q", a)
	}
}

func TestPostAuthor_Found(t *testing.T) {
	s := NewPostService(&fakeRepo{fnGet: getOK(postWith("alice"))})
	if a := s.postAuthor(bg(), newOID().Hex()); a != "alice" {
		t.Fatalf("auteur attendu alice, got %q", a)
	}
}

// ─── ListComments / ListReplies ──────────────────────────────────────────────

func TestListComments_InvalidID(t *testing.T) {
	s := NewPostService(&fakeRepo{})
	if _, err := s.ListComments(bg(), "bad", "", 0, 0); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("attendu ErrInvalidID, got %v", err)
	}
}

func TestListComments_RepoError(t *testing.T) {
	repo := &fakeRepo{fnListComments: func(context.Context, string, int64, int64) ([]models.Comment, error) { return nil, errBoom }}
	s := NewPostService(repo)
	if _, err := s.ListComments(bg(), newOID().Hex(), "", 0, 0); !errors.Is(err, errBoom) {
		t.Fatalf("erreur repo doit remonter, got %v", err)
	}
}

func TestListReplies_InvalidID(t *testing.T) {
	s := NewPostService(&fakeRepo{})
	if _, err := s.ListReplies(bg(), "bad", "", 0, 0); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("attendu ErrInvalidID, got %v", err)
	}
}

func TestListReplies_RepoError(t *testing.T) {
	repo := &fakeRepo{fnListReplies: func(context.Context, string, int64, int64) ([]models.Comment, error) { return nil, errBoom }}
	s := NewPostService(repo)
	if _, err := s.ListReplies(bg(), newOID().Hex(), "", 0, 0); !errors.Is(err, errBoom) {
		t.Fatalf("erreur repo doit remonter, got %v", err)
	}
}

// ListReplies hydrate l'état "liked" du visiteur.
func TestListReplies_HydratesLiked(t *testing.T) {
	c := commentWith(newOID().Hex(), "author")
	repo := &fakeRepo{
		fnListReplies: func(context.Context, string, int64, int64) ([]models.Comment, error) {
			return []models.Comment{*c}, nil
		},
		fnLikedCommentIDsByUser: func(_ context.Context, _ string, _ []string) ([]string, error) { return []string{c.ID.Hex()}, nil },
	}
	s := NewPostService(repo)
	replies, err := s.ListReplies(bg(), newOID().Hex(), "viewer", 0, 0)
	if err != nil || len(replies) != 1 || !replies[0].Liked {
		t.Fatalf("la réponse likée par le visiteur doit être marquée: %v / %+v", err, replies)
	}
}

// ─── ListCommentsByAuthor ────────────────────────────────────────────────────

func TestListCommentsByAuthor_Blocked(t *testing.T) {
	s := NewPostService(&fakeRepo{}, WithFollowClient(&stubFollow{blocked: true}))
	got, err := s.ListCommentsByAuthor(bg(), "author", "viewer", 0, 0)
	if err != nil || len(got) != 0 {
		t.Fatalf("viewer bloqué → liste vide: %v / %d", err, len(got))
	}
}

func TestListCommentsByAuthor_BlockedError(t *testing.T) {
	s := NewPostService(&fakeRepo{}, WithFollowClient(&stubFollow{err: errBoom}))
	if _, err := s.ListCommentsByAuthor(bg(), "author", "viewer", 0, 0); !errors.Is(err, ErrDependencyUnavailable) {
		t.Fatalf("erreur de blocage doit remonter, got %v", err)
	}
}

func TestListCommentsByAuthor_RepoError(t *testing.T) {
	repo := &fakeRepo{fnListCommentsByAuthor: func(context.Context, string, int64, int64) ([]models.Comment, error) { return nil, errBoom }}
	s := NewPostService(repo)
	if _, err := s.ListCommentsByAuthor(bg(), "author", "", 0, 0); !errors.Is(err, errBoom) {
		t.Fatalf("erreur repo doit remonter, got %v", err)
	}
}

func TestListCommentsByAuthor_FiltersHiddenAndPrivate(t *testing.T) {
	visiblePostID := newOID()
	hiddenPostID := newOID()
	badPostComment := commentWith("not-hex", "author") // PostID non parseable → ignoré
	hiddenComment := commentWith(hiddenPostID.Hex(), "author")
	rootID := newOID()
	reply := commentWith(visiblePostID.Hex(), "author")
	reply.ParentID = rootID.Hex()
	root := &models.Comment{ID: rootID, PostID: visiblePostID.Hex(), AuthorID: "root"}

	repo := &fakeRepo{
		fnListCommentsByAuthor: func(_ context.Context, _ string, _, off int64) ([]models.Comment, error) {
			if off == 0 {
				return []models.Comment{*badPostComment, *hiddenComment, *reply}, nil
			}
			return nil, nil
		},
		fnGet: func(_ context.Context, id bson.ObjectID) (*models.Post, error) {
			if id == hiddenPostID {
				return &models.Post{ID: id, AuthorID: "a1", IsHidden: true}, nil
			}
			return &models.Post{ID: id, AuthorID: "a1"}, nil
		},
		fnGetComment: func(_ context.Context, id bson.ObjectID) (*models.Comment, error) {
			if id == rootID {
				return root, nil
			}
			return nil, mongo.ErrNoDocuments
		},
	}
	s := NewPostService(repo)
	got, err := s.ListCommentsByAuthor(bg(), "author", "", 0, 0)
	if err != nil {
		t.Fatalf("ListCommentsByAuthor: %v", err)
	}
	// Seule la réponse au post visible passe ; son commentaire parent est hydraté.
	if len(got) != 1 || got[0].ParentComment == nil || got[0].ParentComment.ID != rootID {
		t.Fatalf("résultat inattendu: %+v", got)
	}
}

func TestListCommentsByAuthor_CanReadError(t *testing.T) {
	postID := newOID()
	c := commentWith(postID.Hex(), "author")
	repo := &fakeRepo{
		fnListCommentsByAuthor: func(_ context.Context, _ string, _, off int64) ([]models.Comment, error) {
			if off == 0 {
				return []models.Comment{*c}, nil
			}
			return nil, nil
		},
		fnGet: func(_ context.Context, id bson.ObjectID) (*models.Post, error) {
			return &models.Post{ID: id, AuthorID: "a1"}, nil
		},
	}
	s := NewPostService(repo, WithProfilClient(&stubProfil{visErr: errBoom}))
	if _, err := s.ListCommentsByAuthor(bg(), "author", "viewer", 0, 0); !errors.Is(err, ErrDependencyUnavailable) {
		t.Fatalf("erreur de visibilité doit remonter, got %v", err)
	}
}

// ─── DeleteComment ───────────────────────────────────────────────────────────

func TestDeleteComment_InvalidID(t *testing.T) {
	s := NewPostService(&fakeRepo{})
	if err := s.DeleteComment(bg(), "bad", "u1", models.RoleUser); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("attendu ErrInvalidID, got %v", err)
	}
}

func TestDeleteComment_NotFound(t *testing.T) {
	repo := &fakeRepo{fnGetComment: func(context.Context, bson.ObjectID) (*models.Comment, error) { return nil, mongo.ErrNoDocuments }}
	s := NewPostService(repo)
	if err := s.DeleteComment(bg(), newOID().Hex(), "u1", models.RoleUser); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("attendu ErrPostNotFound, got %v", err)
	}
}

// ─── Pin / Unpin : branches d'erreur ─────────────────────────────────────────

func TestPinPost_NotFound(t *testing.T) {
	repo := &fakeRepo{fnGet: func(context.Context, bson.ObjectID) (*models.Post, error) { return nil, mongo.ErrNoDocuments }}
	s := NewPostService(repo)
	if _, err := s.PinPost(bg(), newOID().Hex(), "u1"); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("attendu ErrPostNotFound, got %v", err)
	}
}

func TestPinPost_UnpinByAuthorError(t *testing.T) {
	repo := &fakeRepo{
		fnGet:           getOK(postWith("a1")),
		fnUnpinByAuthor: func(context.Context, string) error { return errBoom },
	}
	s := NewPostService(repo)
	if _, err := s.PinPost(bg(), newOID().Hex(), "a1"); !errors.Is(err, errBoom) {
		t.Fatalf("erreur UnpinByAuthor doit remonter, got %v", err)
	}
}

func TestUnpinPost_NotFound(t *testing.T) {
	repo := &fakeRepo{fnGet: func(context.Context, bson.ObjectID) (*models.Post, error) { return nil, mongo.ErrNoDocuments }}
	s := NewPostService(repo)
	if _, err := s.UnpinPost(bg(), newOID().Hex(), "u1"); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("attendu ErrPostNotFound, got %v", err)
	}
}

func TestUnpinPost_Forbidden(t *testing.T) {
	repo := &fakeRepo{fnGet: getOK(postWith("a1"))}
	s := NewPostService(repo)
	if _, err := s.UnpinPost(bg(), newOID().Hex(), "intrus"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("un tiers ne peut pas désépingler → ErrForbidden, got %v", err)
	}
}

// ─── GetByProfile : branches d'erreur ────────────────────────────────────────

func TestGetByProfile_CanReadError(t *testing.T) {
	s := NewPostService(&fakeRepo{}, WithProfilClient(&stubProfil{visErr: errBoom}))
	if _, err := s.GetByProfile(bg(), "author", "viewer", "", 0, 0); !errors.Is(err, ErrDependencyUnavailable) {
		t.Fatalf("erreur de visibilité doit remonter, got %v", err)
	}
}

func TestGetByProfile_RepoError(t *testing.T) {
	repo := &fakeRepo{fnGetByProfileHashtag: func(context.Context, string, string, int64, int64) ([]models.Post, error) { return nil, errBoom }}
	s := NewPostService(repo)
	if _, err := s.GetByProfile(bg(), "author", "author", "", 0, 0); !errors.Is(err, errBoom) {
		t.Fatalf("erreur GetByProfileHashtag doit remonter, got %v", err)
	}
}
