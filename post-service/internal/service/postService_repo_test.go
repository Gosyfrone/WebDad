package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/webdad/post-service/internal/models"
	"github.com/webdad/post-service/internal/repository"
)

var errBoom = errors.New("boom")

func bg() context.Context { return context.Background() }

// postWith renvoie un post minimal avec un id valide.
func postWith(authorID string) *models.Post {
	return &models.Post{ID: newOID(), AuthorID: authorID, ReplyAudience: models.ReplyAudienceEveryone}
}

// ─── CreatePost ──────────────────────────────────────────────────────────────

func TestCreatePost_Success(t *testing.T) {
	repo := &fakeRepo{fnCreate: func(_ context.Context, p *models.Post) error { p.ID = newOID(); return nil }}
	notif := &capNotifier{}
	s := NewPostService(repo, WithNotifier(notif))
	post, err := s.CreatePost(bg(), "u1", "hello #go @bob", "", nil, nil, "", false)
	if err != nil {
		t.Fatalf("CreatePost: %v", err)
	}
	if post.AuthorID != "u1" || !post.CanReply {
		t.Fatalf("post inattendu: %+v", post)
	}
	if len(post.Hashtags) != 1 || post.Hashtags[0] != "go" {
		t.Fatalf("hashtags = %v", post.Hashtags)
	}
	if notif.typeCount("mention") != 1 {
		t.Fatalf("mention attendue, events=%v", notif.events)
	}
}

func TestCreatePost_NSFWByAuthor(t *testing.T) {
	repo := &fakeRepo{fnCreate: func(context.Context, *models.Post) error { return nil }}
	s := NewPostService(repo)
	post, err := s.CreatePost(bg(), "u1", "x", "", nil, nil, models.ReplyAudienceFollowers, true)
	if err != nil {
		t.Fatalf("CreatePost: %v", err)
	}
	if post.NsfwBy != "u1" || post.NsfwAt == nil {
		t.Fatal("le flag nsfw posé par l'auteur doit tracer NsfwBy/NsfwAt")
	}
}

func TestCreatePost_WithQuote(t *testing.T) {
	quoted := postWith("author-quoted")
	repo := &fakeRepo{
		fnGet:    func(context.Context, bson.ObjectID) (*models.Post, error) { return quoted, nil },
		fnCreate: func(context.Context, *models.Post) error { return nil },
	}
	notif := &capNotifier{}
	s := NewPostService(repo, WithNotifier(notif))
	_, err := s.CreatePost(bg(), "u1", "cite", quoted.ID.Hex(), nil, nil, "", false)
	if err != nil {
		t.Fatalf("CreatePost: %v", err)
	}
	if notif.typeCount("quote") != 1 {
		t.Fatalf("notif quote attendue, events=%v", notif.events)
	}
}

func TestCreatePost_QuoteInvalidID(t *testing.T) {
	s := NewPostService(&fakeRepo{})
	_, err := s.CreatePost(bg(), "u1", "x", "not-hex", nil, nil, "", false)
	if !errors.Is(err, ErrInvalidID) {
		t.Fatalf("attendu ErrInvalidID, got %v", err)
	}
}

func TestCreatePost_QuoteNotFound(t *testing.T) {
	repo := &fakeRepo{fnGet: func(context.Context, bson.ObjectID) (*models.Post, error) { return nil, mongo.ErrNoDocuments }}
	s := NewPostService(repo)
	_, err := s.CreatePost(bg(), "u1", "x", newOID().Hex(), nil, nil, "", false)
	if !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("attendu ErrPostNotFound, got %v", err)
	}
}

func TestCreatePost_CreateError(t *testing.T) {
	repo := &fakeRepo{fnCreate: func(context.Context, *models.Post) error { return errBoom }}
	s := NewPostService(repo)
	_, err := s.CreatePost(bg(), "u1", "x", "", nil, nil, "", false)
	if !errors.Is(err, errBoom) {
		t.Fatalf("attendu errBoom, got %v", err)
	}
}

func TestCreatePost_InvalidPoll(t *testing.T) {
	s := NewPostService(&fakeRepo{})
	_, err := s.CreatePost(bg(), "u1", "x", "", nil, &models.CreatePollRequest{DurationMinutes: 0}, "", false)
	if !errors.Is(err, ErrInvalidPoll) {
		t.Fatalf("attendu ErrInvalidPoll, got %v", err)
	}
}

// ─── GetPost ─────────────────────────────────────────────────────────────────

func TestGetPost_Success(t *testing.T) {
	p := postWith("a1")
	repo := &fakeRepo{fnGet: func(context.Context, bson.ObjectID) (*models.Post, error) { return p, nil }}
	s := NewPostService(repo)
	got, err := s.GetPost(bg(), p.ID.Hex(), "a1", models.RoleUser)
	if err != nil || got == nil {
		t.Fatalf("GetPost: %v / %v", got, err)
	}
}

func TestGetPost_HiddenForPublic(t *testing.T) {
	p := postWith("a1")
	p.IsHidden = true
	repo := &fakeRepo{fnGet: func(context.Context, bson.ObjectID) (*models.Post, error) { return p, nil }}
	s := NewPostService(repo)
	if _, err := s.GetPost(bg(), p.ID.Hex(), "viewer", models.RoleUser); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("post masqué doit être 404 pour le public, got %v", err)
	}
}

func TestGetPost_HiddenVisibleForModerator(t *testing.T) {
	p := postWith("a1")
	p.AutoHidden = true
	repo := &fakeRepo{fnGet: func(context.Context, bson.ObjectID) (*models.Post, error) { return p, nil }}
	s := NewPostService(repo)
	if _, err := s.GetPost(bg(), p.ID.Hex(), "mod", models.RoleModerator); err != nil {
		t.Fatalf("un modérateur doit voir un post masqué: %v", err)
	}
}

func TestGetPost_PrivateProfil(t *testing.T) {
	p := postWith("a1")
	repo := &fakeRepo{fnGet: func(context.Context, bson.ObjectID) (*models.Post, error) { return p, nil }}
	s := NewPostService(repo, WithProfilClient(&stubProfil{visibility: "private"}))
	if _, err := s.GetPost(bg(), p.ID.Hex(), "viewer", models.RoleUser); !errors.Is(err, ErrPrivateProfil) {
		t.Fatalf("attendu ErrPrivateProfil, got %v", err)
	}
}

func TestGetPost_NotFound(t *testing.T) {
	repo := &fakeRepo{fnGet: func(context.Context, bson.ObjectID) (*models.Post, error) { return nil, mongo.ErrNoDocuments }}
	s := NewPostService(repo)
	if _, err := s.GetPost(bg(), newOID().Hex(), "v", ""); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("attendu ErrPostNotFound, got %v", err)
	}
}

// ─── PostStats ───────────────────────────────────────────────────────────────

func TestPostStats_FiltersByVisibility(t *testing.T) {
	pubID, privID := newOID(), newOID()
	repo := &fakeRepo{fnStatsByIDs: func(context.Context, []bson.ObjectID) ([]models.Post, error) {
		return []models.Post{
			{ID: pubID, AuthorID: "pub", LikesCount: 3},
			{ID: privID, AuthorID: "priv", LikesCount: 9},
		}, nil
	}}
	profilFn := func(_ context.Context, id string) (string, error) {
		if id == "priv" {
			return "private", nil
		}
		return "public", nil
	}
	s := NewPostService(repo, WithProfilClient(profilStub(profilFn)))
	stats, err := s.PostStats(bg(), []string{pubID.Hex(), privID.Hex()}, "viewer")
	if err != nil {
		t.Fatalf("PostStats: %v", err)
	}
	if len(stats) != 1 || stats[0].LikesCount != 3 {
		t.Fatalf("seul le post public doit ressortir: %+v", stats)
	}
}

func TestPostStats_Truncates(t *testing.T) {
	var got int
	repo := &fakeRepo{fnStatsByIDs: func(_ context.Context, o []bson.ObjectID) ([]models.Post, error) {
		got = len(o)
		return nil, nil
	}}
	s := NewPostService(repo)
	ids := make([]string, MaxStatsIDs+10)
	for i := range ids {
		ids[i] = newOID().Hex()
	}
	if _, err := s.PostStats(bg(), ids, ""); err != nil {
		t.Fatalf("PostStats: %v", err)
	}
	if got != MaxStatsIDs {
		t.Fatalf("ids tronqués à %d, attendu %d", got, MaxStatsIDs)
	}
}

// ─── UpdatePost / Pin / Unpin ────────────────────────────────────────────────

func TestUpdatePost_Success(t *testing.T) {
	p := postWith("a1")
	repo := &fakeRepo{
		fnGet: func(context.Context, bson.ObjectID) (*models.Post, error) { return p, nil },
		fnUpdate: func(_ context.Context, _ bson.ObjectID, c string, _ []string) (*models.Post, error) {
			return &models.Post{Content: c}, nil
		},
	}
	s := NewPostService(repo)
	got, err := s.UpdatePost(bg(), p.ID.Hex(), "nouveau", "a1", models.RoleUser)
	if err != nil || got.Content != "nouveau" {
		t.Fatalf("UpdatePost: %v / %+v", err, got)
	}
}

func TestUpdatePost_Forbidden(t *testing.T) {
	p := postWith("a1")
	repo := &fakeRepo{fnGet: func(context.Context, bson.ObjectID) (*models.Post, error) { return p, nil }}
	s := NewPostService(repo)
	if _, err := s.UpdatePost(bg(), p.ID.Hex(), "x", "intrus", models.RoleUser); !errors.Is(err, ErrForbidden) {
		t.Fatalf("attendu ErrForbidden, got %v", err)
	}
}

func TestPinPost_Success(t *testing.T) {
	p := postWith("a1")
	repo := &fakeRepo{
		fnGet:           func(context.Context, bson.ObjectID) (*models.Post, error) { return p, nil },
		fnUnpinByAuthor: func(context.Context, string) error { return nil },
		fnPin:           func(context.Context, bson.ObjectID, time.Time) (*models.Post, error) { return p, nil },
	}
	s := NewPostService(repo)
	if _, err := s.PinPost(bg(), p.ID.Hex(), "a1"); err != nil {
		t.Fatalf("PinPost: %v", err)
	}
}

func TestPinPost_ForbiddenForThirdParty(t *testing.T) {
	p := postWith("a1")
	repo := &fakeRepo{fnGet: func(context.Context, bson.ObjectID) (*models.Post, error) { return p, nil }}
	s := NewPostService(repo)
	if _, err := s.PinPost(bg(), p.ID.Hex(), "admin"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("seul l'auteur peut épingler, got %v", err)
	}
}

func TestUnpinPost_Success(t *testing.T) {
	p := postWith("a1")
	repo := &fakeRepo{
		fnGet:   func(context.Context, bson.ObjectID) (*models.Post, error) { return p, nil },
		fnUnpin: func(context.Context, bson.ObjectID) (*models.Post, error) { return p, nil },
	}
	s := NewPostService(repo)
	if _, err := s.UnpinPost(bg(), p.ID.Hex(), "a1"); err != nil {
		t.Fatalf("UnpinPost: %v", err)
	}
}

// ─── DeletePost ──────────────────────────────────────────────────────────────

func TestDeletePost_HardByAuthor(t *testing.T) {
	p := postWith("a1")
	var deleted bool
	repo := &fakeRepo{
		fnGet:    func(context.Context, bson.ObjectID) (*models.Post, error) { return p, nil },
		fnDelete: func(context.Context, bson.ObjectID) error { deleted = true; return nil },
	}
	notif := &capNotifier{}
	s := NewPostService(repo, WithNotifier(notif))
	if err := s.DeletePost(bg(), p.ID.Hex(), "a1", models.RoleUser); err != nil {
		t.Fatalf("DeletePost: %v", err)
	}
	if !deleted || notif.typeCount("post_deleted") != 1 {
		t.Fatalf("suppression auteur = hard + cascade, deleted=%v events=%v", deleted, notif.events)
	}
}

func TestDeletePost_SoftByModerator(t *testing.T) {
	p := postWith("a1")
	var hidden bool
	repo := &fakeRepo{
		fnGet: func(context.Context, bson.ObjectID) (*models.Post, error) { return p, nil },
		fnHide: func(context.Context, bson.ObjectID, string, time.Time) (*models.Post, error) {
			hidden = true
			return p, nil
		},
	}
	s := NewPostService(repo)
	if err := s.DeletePost(bg(), p.ID.Hex(), "mod", models.RoleModerator); err != nil {
		t.Fatalf("DeletePost: %v", err)
	}
	if !hidden {
		t.Fatal("retrait par la modération = masquage doux")
	}
}

func TestDeletePost_Forbidden(t *testing.T) {
	p := postWith("a1")
	repo := &fakeRepo{fnGet: func(context.Context, bson.ObjectID) (*models.Post, error) { return p, nil }}
	s := NewPostService(repo)
	if err := s.DeletePost(bg(), p.ID.Hex(), "intrus", models.RoleUser); !errors.Is(err, ErrForbidden) {
		t.Fatalf("attendu ErrForbidden, got %v", err)
	}
}

// ─── Modération : corbeille, restore, nsfw, autohide, purge ──────────────────

func TestListHiddenPosts_ForbiddenForUser(t *testing.T) {
	s := NewPostService(&fakeRepo{})
	if _, err := s.ListHiddenPosts(bg(), models.RoleUser, repository.HiddenFilter{}, 0, 0); !errors.Is(err, ErrForbidden) {
		t.Fatalf("attendu ErrForbidden, got %v", err)
	}
}

func TestListHiddenPosts_SetsPurgeAt(t *testing.T) {
	hiddenAt := time.Now().Add(-time.Hour)
	repo := &fakeRepo{fnListHidden: func(context.Context, repository.HiddenFilter, int64, int64) ([]models.Post, error) {
		return []models.Post{{ID: newOID(), HiddenAt: &hiddenAt}}, nil
	}}
	s := NewPostService(repo, WithPurgeRetention(24*time.Hour, time.Hour))
	posts, err := s.ListHiddenPosts(bg(), models.RoleAdmin, repository.HiddenFilter{}, 0, 0)
	if err != nil {
		t.Fatalf("ListHiddenPosts: %v", err)
	}
	if posts[0].PurgeAt == nil {
		t.Fatal("PurgeAt doit être calculé quand une rétention est configurée")
	}
}

func TestRestorePost(t *testing.T) {
	repo := &fakeRepo{fnRestoreHidden: func(context.Context, bson.ObjectID) (*models.Post, error) { return postWith("a1"), nil }}
	s := NewPostService(repo)
	if _, err := s.RestorePost(bg(), newOID().Hex(), models.RoleModerator); err != nil {
		t.Fatalf("RestorePost: %v", err)
	}
	if _, err := s.RestorePost(bg(), newOID().Hex(), models.RoleUser); !errors.Is(err, ErrForbidden) {
		t.Fatalf("attendu ErrForbidden pour un user, got %v", err)
	}
}

func TestSetNsfw(t *testing.T) {
	repo := &fakeRepo{fnSetNsfw: func(context.Context, bson.ObjectID, bool, string, time.Time) (*models.Post, error) {
		return postWith("a1"), nil
	}}
	s := NewPostService(repo)
	if _, err := s.SetNsfw(bg(), newOID().Hex(), "mod", models.RoleModerator, true); err != nil {
		t.Fatalf("SetNsfw: %v", err)
	}
	if _, err := s.SetNsfw(bg(), newOID().Hex(), "u", models.RoleUser, true); !errors.Is(err, ErrForbidden) {
		t.Fatalf("attendu ErrForbidden, got %v", err)
	}
}

func TestAutoHideUnhide(t *testing.T) {
	repo := &fakeRepo{fnSetAutoHidden: func(_ context.Context, _ bson.ObjectID, h bool) (*models.Post, error) {
		return &models.Post{AutoHidden: h}, nil
	}}
	s := NewPostService(repo)
	if p, err := s.AutoHide(bg(), newOID().Hex()); err != nil || !p.AutoHidden {
		t.Fatalf("AutoHide: %v / %+v", err, p)
	}
	if p, err := s.AutoUnhide(bg(), newOID().Hex()); err != nil || p.AutoHidden {
		t.Fatalf("AutoUnhide: %v / %+v", err, p)
	}
	if _, err := s.AutoHide(bg(), "not-hex"); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("attendu ErrInvalidID, got %v", err)
	}
}

func TestPurgeUserData(t *testing.T) {
	repo := &fakeRepo{fnPurgeByAuthor: func(context.Context, string) (int64, error) { return 5, nil }}
	s := NewPostService(repo)
	if n, err := s.PurgeUserData(bg(), "u1", models.RoleAdmin); err != nil || n != 5 {
		t.Fatalf("PurgeUserData: %v / %d", err, n)
	}
	if _, err := s.PurgeUserData(bg(), "u1", models.RoleModerator); !errors.Is(err, ErrForbidden) {
		t.Fatalf("la purge RGPD est réservée admin, got %v", err)
	}
}

func TestPurgePost(t *testing.T) {
	p := postWith("a1")
	repo := &fakeRepo{
		fnGet:    func(context.Context, bson.ObjectID) (*models.Post, error) { return p, nil },
		fnDelete: func(context.Context, bson.ObjectID) error { return nil },
	}
	s := NewPostService(repo)
	if err := s.PurgePost(bg(), p.ID.Hex(), "mod", models.RoleAdmin); err != nil {
		t.Fatalf("PurgePost: %v", err)
	}
	if err := s.PurgePost(bg(), p.ID.Hex(), "u", models.RoleUser); !errors.Is(err, ErrForbidden) {
		t.Fatalf("attendu ErrForbidden, got %v", err)
	}
}

func TestSweepPurge(t *testing.T) {
	warnable := []models.Post{{ID: newOID(), AuthorID: "a1"}}
	purgeable := []models.Post{{ID: newOID(), AuthorID: "a2"}}
	repo := &fakeRepo{
		fnListPurgeWarnable: func(context.Context, time.Time, int64) ([]models.Post, error) { return warnable, nil },
		fnMarkPurgeWarned:   func(context.Context, bson.ObjectID, time.Time) error { return nil },
		fnListPurgeable:     func(context.Context, time.Time, int64) ([]models.Post, error) { return purgeable, nil },
		fnDelete:            func(context.Context, bson.ObjectID) error { return nil },
	}
	notif := &capNotifier{}
	s := NewPostService(repo, WithPurgeRetention(24*time.Hour, time.Hour), WithNotifier(notif))
	warned, purged, err := s.SweepPurge(bg())
	if err != nil || warned != 1 || purged != 1 {
		t.Fatalf("SweepPurge = (%d,%d,%v), attendu (1,1,nil)", warned, purged, err)
	}
}

// ─── Feeds / profil ──────────────────────────────────────────────────────────

func TestGetByProfile_PrivateReturnsEmpty(t *testing.T) {
	s := NewPostService(&fakeRepo{}, WithProfilClient(&stubProfil{visibility: "private"}))
	posts, err := s.GetByProfile(bg(), "author", "viewer", "", 0, 0)
	if err != nil || len(posts) != 0 {
		t.Fatalf("profil privé → liste vide, got %v / %v", posts, err)
	}
}

func TestGetByProfile_Success(t *testing.T) {
	repo := &fakeRepo{fnGetByProfileHashtag: func(context.Context, string, string, int64, int64) ([]models.Post, error) {
		return []models.Post{*postWith("author")}, nil
	}}
	s := NewPostService(repo)
	posts, err := s.GetByProfile(bg(), "author", "author", "", 0, 0)
	if err != nil || len(posts) != 1 {
		t.Fatalf("GetByProfile: %v / %d", err, len(posts))
	}
}

func TestGetFeed_EmptyAuthors(t *testing.T) {
	s := NewPostService(&fakeRepo{})
	posts, err := s.GetFeed(bg(), nil, "v", "", "", 0, 0)
	if err != nil || len(posts) != 0 {
		t.Fatalf("GetFeed(nil) → vide, got %v / %v", posts, err)
	}
}

func TestGetFeed_Success(t *testing.T) {
	repo := &fakeRepo{fnGetByAuthorsHashtag: func(context.Context, []string, string, string, int64, int64) ([]models.Post, error) {
		return []models.Post{*postWith("a1"), *postWith("a2")}, nil
	}}
	s := NewPostService(repo)
	posts, err := s.GetFeed(bg(), []string{"a1", "a2"}, "v", "", "top", 0, 0)
	if err != nil || len(posts) != 2 {
		t.Fatalf("GetFeed: %v / %d", err, len(posts))
	}
}

func TestGetPosts_WithHashtag(t *testing.T) {
	repo := &fakeRepo{fnGetAllByHashtag: func(context.Context, string, string, int64, int64) ([]models.Post, error) {
		return []models.Post{*postWith("a1")}, nil
	}}
	s := NewPostService(repo)
	posts, err := s.GetPosts(bg(), "v", "go", "recent", false, 0, 0)
	if err != nil || len(posts) != 1 {
		t.Fatalf("GetPosts(hashtag): %v / %d", err, len(posts))
	}
}

func TestTrendingHashtags(t *testing.T) {
	calls := 0
	repo := &fakeRepo{fnGetAll: func(context.Context, int64, int64) ([]models.Post, error) {
		calls++
		if calls > 1 {
			return nil, nil
		}
		return []models.Post{
			{ID: newOID(), AuthorID: "a", Hashtags: []string{"go", "test"}},
			{ID: newOID(), AuthorID: "a", Hashtags: []string{"go"}},
		}, nil
	}}
	s := NewPostService(repo)
	trends, err := s.TrendingHashtags(bg(), "v", "", 10)
	if err != nil {
		t.Fatalf("TrendingHashtags: %v", err)
	}
	if len(trends) != 2 || trends[0].Tag != "go" || trends[0].Count != 2 {
		t.Fatalf("classement inattendu: %+v", trends)
	}
}

// ─── Polls ───────────────────────────────────────────────────────────────────

func pollPost(authorID string) *models.Post {
	p := postWith(authorID)
	p.Poll = &models.Poll{
		Choices:  []models.PollChoice{{ID: "c1"}, {ID: "c2"}},
		EndsAt:   time.Now().Add(time.Hour),
		Audience: models.PollAudienceEveryone,
	}
	return p
}

func TestVotePoll_Success(t *testing.T) {
	p := pollPost("a1")
	repo := &fakeRepo{
		fnGet:           func(context.Context, bson.ObjectID) (*models.Post, error) { return p, nil },
		fnAddPollVote:   func(context.Context, string, string, string) (bool, error) { return true, nil },
		fnIncPollChoice: func(context.Context, bson.ObjectID, string) (*models.Post, error) { return p, nil },
	}
	s := NewPostService(repo)
	if _, err := s.VotePoll(bg(), p.ID.Hex(), "voter", "c1"); err != nil {
		t.Fatalf("VotePoll: %v", err)
	}
}

func TestVotePoll_AlreadyVoted(t *testing.T) {
	p := pollPost("a1")
	repo := &fakeRepo{
		fnGet:         func(context.Context, bson.ObjectID) (*models.Post, error) { return p, nil },
		fnAddPollVote: func(context.Context, string, string, string) (bool, error) { return false, nil },
	}
	s := NewPostService(repo)
	if _, err := s.VotePoll(bg(), p.ID.Hex(), "voter", "c1"); !errors.Is(err, ErrPollAlreadyVoted) {
		t.Fatalf("attendu ErrPollAlreadyVoted, got %v", err)
	}
}

func TestVotePoll_Closed(t *testing.T) {
	p := pollPost("a1")
	p.Poll.EndsAt = time.Now().Add(-time.Hour)
	repo := &fakeRepo{fnGet: func(context.Context, bson.ObjectID) (*models.Post, error) { return p, nil }}
	s := NewPostService(repo)
	if _, err := s.VotePoll(bg(), p.ID.Hex(), "v", "c1"); !errors.Is(err, ErrPollClosed) {
		t.Fatalf("attendu ErrPollClosed, got %v", err)
	}
}

func TestVotePoll_NoPoll(t *testing.T) {
	p := postWith("a1")
	repo := &fakeRepo{fnGet: func(context.Context, bson.ObjectID) (*models.Post, error) { return p, nil }}
	s := NewPostService(repo)
	if _, err := s.VotePoll(bg(), p.ID.Hex(), "v", "c1"); !errors.Is(err, ErrInvalidPoll) {
		t.Fatalf("attendu ErrInvalidPoll, got %v", err)
	}
}

func TestVotePoll_BadChoice(t *testing.T) {
	p := pollPost("a1")
	repo := &fakeRepo{fnGet: func(context.Context, bson.ObjectID) (*models.Post, error) { return p, nil }}
	s := NewPostService(repo)
	if _, err := s.VotePoll(bg(), p.ID.Hex(), "v", "inexistant"); !errors.Is(err, ErrInvalidPoll) {
		t.Fatalf("attendu ErrInvalidPoll, got %v", err)
	}
}

func TestVotePoll_FollowersOnlyForbidden(t *testing.T) {
	p := pollPost("a1")
	p.Poll.Audience = models.PollAudienceFollowers
	repo := &fakeRepo{fnGet: func(context.Context, bson.ObjectID) (*models.Post, error) { return p, nil }}
	s := NewPostService(repo, WithFollowClient(&stubFollow{follows: false}))
	if _, err := s.VotePoll(bg(), p.ID.Hex(), "v", "c1"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("non-abonné → ErrForbidden, got %v", err)
	}
}

func TestClosePoll(t *testing.T) {
	p := pollPost("a1")
	repo := &fakeRepo{
		fnGet:       func(context.Context, bson.ObjectID) (*models.Post, error) { return p, nil },
		fnClosePoll: func(context.Context, bson.ObjectID, time.Time) (*models.Post, error) { return p, nil },
	}
	s := NewPostService(repo)
	if _, err := s.ClosePoll(bg(), p.ID.Hex(), "a1"); err != nil {
		t.Fatalf("ClosePoll: %v", err)
	}
	if _, err := s.ClosePoll(bg(), p.ID.Hex(), "intrus"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("seul l'auteur ferme, got %v", err)
	}
}

// ─── Likes ───────────────────────────────────────────────────────────────────

func TestLikePost(t *testing.T) {
	p := postWith("a1")
	repo := &fakeRepo{
		fnGet:     func(context.Context, bson.ObjectID) (*models.Post, error) { return p, nil },
		fnAddLike: func(context.Context, string, string) (bool, error) { return true, nil },
		fnIncCounter: func(context.Context, bson.ObjectID, string, int32) (*models.Post, error) {
			return &models.Post{LikesCount: 1}, nil
		},
	}
	notif := &capNotifier{}
	s := NewPostService(repo, WithNotifier(notif))
	n, err := s.LikePost(bg(), p.ID.Hex(), "liker")
	if err != nil || n != 1 || notif.typeCount("like") != 1 {
		t.Fatalf("LikePost = %d / %v, events=%v", n, err, notif.events)
	}
}

func TestLikePost_Idempotent(t *testing.T) {
	p := postWith("a1")
	p.LikesCount = 7
	repo := &fakeRepo{
		fnGet:     func(context.Context, bson.ObjectID) (*models.Post, error) { return p, nil },
		fnAddLike: func(context.Context, string, string) (bool, error) { return false, nil },
	}
	s := NewPostService(repo)
	if n, err := s.LikePost(bg(), p.ID.Hex(), "liker"); err != nil || n != 7 {
		t.Fatalf("reliker ne double pas: %d / %v", n, err)
	}
}

func TestUnlikePost(t *testing.T) {
	p := postWith("a1")
	repo := &fakeRepo{
		fnGet:        func(context.Context, bson.ObjectID) (*models.Post, error) { return p, nil },
		fnRemoveLike: func(context.Context, string, string) (bool, error) { return true, nil },
		fnIncCounter: func(context.Context, bson.ObjectID, string, int32) (*models.Post, error) {
			return &models.Post{LikesCount: 0}, nil
		},
	}
	s := NewPostService(repo)
	if n, err := s.UnlikePost(bg(), p.ID.Hex(), "liker"); err != nil || n != 0 {
		t.Fatalf("UnlikePost = %d / %v", n, err)
	}
}

func TestUnlikePost_Idempotent(t *testing.T) {
	p := postWith("a1")
	p.LikesCount = 4
	repo := &fakeRepo{
		fnGet:        func(context.Context, bson.ObjectID) (*models.Post, error) { return p, nil },
		fnRemoveLike: func(context.Context, string, string) (bool, error) { return false, nil },
	}
	s := NewPostService(repo)
	if n, err := s.UnlikePost(bg(), p.ID.Hex(), "l"); err != nil || n != 4 {
		t.Fatalf("déliker un post non liké ne décrémente pas: %d / %v", n, err)
	}
}

func TestUnlikeComment_Idempotent(t *testing.T) {
	postID := newOID().Hex()
	comment := &models.Comment{ID: newOID(), PostID: postID, AuthorID: "c", LikesCount: 2}
	repo := &fakeRepo{
		fnGetComment:        func(context.Context, bson.ObjectID) (*models.Comment, error) { return comment, nil },
		fnRemoveCommentLike: func(context.Context, string, string) (bool, error) { return false, nil },
	}
	s := NewPostService(repo)
	if n, err := s.UnlikeComment(bg(), postID, comment.ID.Hex(), "l"); err != nil || n != 2 {
		t.Fatalf("déliker un commentaire non liké ne décrémente pas: %d / %v", n, err)
	}
}

func TestUnrepostPost_Idempotent(t *testing.T) {
	p := postWith("a1")
	p.RepostsCount = 6
	repo := &fakeRepo{
		fnGet:          func(context.Context, bson.ObjectID) (*models.Post, error) { return p, nil },
		fnRemoveRepost: func(context.Context, string, string) (bool, error) { return false, nil },
	}
	s := NewPostService(repo)
	if n, err := s.UnrepostPost(bg(), p.ID.Hex(), "r"); err != nil || n != 6 {
		t.Fatalf("dé-reposter sans repost ne décrémente pas: %d / %v", n, err)
	}
}

func TestRepostPost_Idempotent(t *testing.T) {
	p := postWith("a1")
	repo := &fakeRepo{
		fnGet: func(context.Context, bson.ObjectID) (*models.Post, error) { return p, nil },
		fnAddRepost: func(context.Context, string, string) (*models.Repost, bool, error) {
			return &models.Repost{CreatedAt: time.Now()}, false, nil
		},
	}
	s := NewPostService(repo)
	got, err := s.RepostPost(bg(), p.ID.Hex(), "r")
	if err != nil || got.RepostedByID != "r" {
		t.Fatalf("RepostPost idempotent: %v / %+v", err, got)
	}
}

func TestLikedPostIDs(t *testing.T) {
	repo := &fakeRepo{fnLikedPostIDs: func(context.Context, string) ([]string, error) { return []string{"a", "b"}, nil }}
	s := NewPostService(repo)
	ids, err := s.LikedPostIDs(bg(), "u")
	if err != nil || len(ids) != 2 {
		t.Fatalf("LikedPostIDs: %v / %v", ids, err)
	}
}

func TestPostLikers(t *testing.T) {
	repo := &fakeRepo{fnLikersByPost: func(context.Context, string) ([]string, error) { return []string{"u1"}, nil }}
	s := NewPostService(repo)
	if _, err := s.PostLikers(bg(), newOID().Hex()); err != nil {
		t.Fatalf("PostLikers: %v", err)
	}
	if _, err := s.PostLikers(bg(), "bad"); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("attendu ErrInvalidID, got %v", err)
	}
}

func TestListLikedByUser_PrivateForbidden(t *testing.T) {
	s := NewPostService(&fakeRepo{}, WithProfilClient(&stubProfil{likesVisibility: "private"}))
	if _, err := s.ListLikedByUser(bg(), "author", "caller", 0, 0); !errors.Is(err, ErrForbidden) {
		t.Fatalf("likes privés → ErrForbidden, got %v", err)
	}
}

func TestListLikedByUser_Success(t *testing.T) {
	repo := &fakeRepo{fnLikedPostsByUser: func(context.Context, string, int64, int64) ([]*models.Post, error) {
		return []*models.Post{postWith("author")}, nil
	}}
	s := NewPostService(repo)
	posts, err := s.ListLikedByUser(bg(), "author", "author", 0, 0)
	if err != nil || len(posts) != 1 {
		t.Fatalf("ListLikedByUser: %v / %d", err, len(posts))
	}
}

// ─── Reposts ─────────────────────────────────────────────────────────────────

func TestRepostPost(t *testing.T) {
	p := postWith("a1")
	repo := &fakeRepo{
		fnGet: func(context.Context, bson.ObjectID) (*models.Post, error) { return p, nil },
		fnAddRepost: func(context.Context, string, string) (*models.Repost, bool, error) {
			return &models.Repost{CreatedAt: time.Now()}, true, nil
		},
		fnIncCounter: func(context.Context, bson.ObjectID, string, int32) (*models.Post, error) { return p, nil },
	}
	notif := &capNotifier{}
	s := NewPostService(repo, WithNotifier(notif))
	got, err := s.RepostPost(bg(), p.ID.Hex(), "reposter")
	if err != nil || got.RepostedByID != "reposter" || notif.typeCount("repost") != 1 {
		t.Fatalf("RepostPost: %v / %+v / %v", err, got, notif.events)
	}
}

func TestUnrepostPost(t *testing.T) {
	p := postWith("a1")
	repo := &fakeRepo{
		fnGet:          func(context.Context, bson.ObjectID) (*models.Post, error) { return p, nil },
		fnRemoveRepost: func(context.Context, string, string) (bool, error) { return true, nil },
		fnIncCounter: func(context.Context, bson.ObjectID, string, int32) (*models.Post, error) {
			return &models.Post{RepostsCount: 0}, nil
		},
	}
	s := NewPostService(repo)
	if n, err := s.UnrepostPost(bg(), p.ID.Hex(), "r"); err != nil || n != 0 {
		t.Fatalf("UnrepostPost: %d / %v", n, err)
	}
}

func TestRepostedPostIDs(t *testing.T) {
	repo := &fakeRepo{fnRepostedPostIDs: func(context.Context, string) ([]string, error) { return []string{"x"}, nil }}
	s := NewPostService(repo)
	if ids, err := s.RepostedPostIDs(bg(), "u"); err != nil || len(ids) != 1 {
		t.Fatalf("RepostedPostIDs: %v / %v", ids, err)
	}
}

// ─── Commentaires ────────────────────────────────────────────────────────────

func TestCreateComment_Root(t *testing.T) {
	p := postWith("a1")
	repo := &fakeRepo{
		fnGet:        func(context.Context, bson.ObjectID) (*models.Post, error) { return p, nil },
		fnAddComment: func(_ context.Context, c *models.Comment) error { c.ID = newOID(); return nil },
		fnIncCounter: func(context.Context, bson.ObjectID, string, int32) (*models.Post, error) { return p, nil },
	}
	notif := &capNotifier{}
	s := NewPostService(repo, WithNotifier(notif))
	c, err := s.CreateComment(bg(), p.ID.Hex(), "commenter", models.RoleUser, "salut @bob", "", nil)
	if err != nil || c == nil {
		t.Fatalf("CreateComment: %v / %v", c, err)
	}
	if notif.typeCount("comment") != 1 || notif.typeCount("mention") != 1 {
		t.Fatalf("notifs commentaire+mention attendues, events=%v", notif.events)
	}
}

func TestCreateComment_Reply(t *testing.T) {
	p := postWith("a1")
	parent := &models.Comment{ID: newOID(), PostID: p.ID.Hex(), AuthorID: "root-author"}
	repo := &fakeRepo{
		fnGet:           func(context.Context, bson.ObjectID) (*models.Post, error) { return p, nil },
		fnGetComment:    func(context.Context, bson.ObjectID) (*models.Comment, error) { return parent, nil },
		fnAddComment:    func(_ context.Context, c *models.Comment) error { c.ID = newOID(); return nil },
		fnIncCounter:    func(context.Context, bson.ObjectID, string, int32) (*models.Post, error) { return p, nil },
		fnIncReplyCount: func(context.Context, bson.ObjectID, int32) error { return nil },
	}
	notif := &capNotifier{}
	s := NewPostService(repo, WithNotifier(notif))
	_, err := s.CreateComment(bg(), p.ID.Hex(), "replier", models.RoleUser, "réponse", parent.ID.Hex(), nil)
	if err != nil {
		t.Fatalf("CreateComment reply: %v", err)
	}
	if notif.typeCount("reply") != 1 {
		t.Fatalf("notif reply attendue, events=%v", notif.events)
	}
}

func TestCreateComment_ReplyNotAllowed(t *testing.T) {
	p := postWith("a1")
	p.ReplyAudience = models.ReplyAudienceFollowers
	repo := &fakeRepo{fnGet: func(context.Context, bson.ObjectID) (*models.Post, error) { return p, nil }}
	s := NewPostService(repo, WithFollowClient(&stubFollow{follows: false}))
	if _, err := s.CreateComment(bg(), p.ID.Hex(), "x", models.RoleUser, "c", "", nil); !errors.Is(err, ErrReplyNotAllowed) {
		t.Fatalf("attendu ErrReplyNotAllowed, got %v", err)
	}
}

func TestDeleteComment_Root(t *testing.T) {
	p := postWith("post-author")
	comment := &models.Comment{ID: newOID(), PostID: p.ID.Hex(), AuthorID: "c-author"}
	repo := &fakeRepo{
		fnGetComment:                   func(context.Context, bson.ObjectID) (*models.Comment, error) { return comment, nil },
		fnDeleteComment:                func(context.Context, bson.ObjectID) error { return nil },
		fnCommentIDsByParent:           func(context.Context, string) ([]string, error) { return []string{"r1"}, nil },
		fnDeleteRepliesByParent:        func(context.Context, string) (int64, error) { return 2, nil },
		fnDeleteCommentLikesByComments: func(context.Context, []string) error { return nil },
		fnDeleteCommentLikesByComment:  func(context.Context, string) error { return nil },
		fnIncCounter:                   func(context.Context, bson.ObjectID, string, int32) (*models.Post, error) { return p, nil },
		fnGet:                          func(context.Context, bson.ObjectID) (*models.Post, error) { return p, nil },
	}
	s := NewPostService(repo)
	if err := s.DeleteComment(bg(), comment.ID.Hex(), "c-author", models.RoleUser); err != nil {
		t.Fatalf("DeleteComment: %v", err)
	}
}

func TestDeleteComment_Forbidden(t *testing.T) {
	comment := &models.Comment{ID: newOID(), PostID: newOID().Hex(), AuthorID: "owner"}
	repo := &fakeRepo{fnGetComment: func(context.Context, bson.ObjectID) (*models.Comment, error) { return comment, nil }}
	s := NewPostService(repo)
	if err := s.DeleteComment(bg(), comment.ID.Hex(), "intrus", models.RoleUser); !errors.Is(err, ErrForbidden) {
		t.Fatalf("attendu ErrForbidden, got %v", err)
	}
}

func TestLikeComment(t *testing.T) {
	postID := newOID().Hex()
	comment := &models.Comment{ID: newOID(), PostID: postID, AuthorID: "c-author"}
	repo := &fakeRepo{
		fnGetComment:     func(context.Context, bson.ObjectID) (*models.Comment, error) { return comment, nil },
		fnAddCommentLike: func(context.Context, string, string) (bool, error) { return true, nil },
		fnIncCommentCounter: func(context.Context, bson.ObjectID, string, int32) (*models.Comment, error) {
			return &models.Comment{LikesCount: 1}, nil
		},
	}
	notif := &capNotifier{}
	s := NewPostService(repo, WithNotifier(notif))
	n, err := s.LikeComment(bg(), postID, comment.ID.Hex(), "liker")
	if err != nil || n != 1 || notif.typeCount("comment_like") != 1 {
		t.Fatalf("LikeComment = %d / %v / %v", n, err, notif.events)
	}
}

func TestLikeComment_WrongPost(t *testing.T) {
	comment := &models.Comment{ID: newOID(), PostID: newOID().Hex(), AuthorID: "x"}
	repo := &fakeRepo{fnGetComment: func(context.Context, bson.ObjectID) (*models.Comment, error) { return comment, nil }}
	s := NewPostService(repo)
	if _, err := s.LikeComment(bg(), newOID().Hex(), comment.ID.Hex(), "l"); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("commentaire d'un autre post → 404, got %v", err)
	}
}

func TestUnlikeComment(t *testing.T) {
	postID := newOID().Hex()
	comment := &models.Comment{ID: newOID(), PostID: postID, AuthorID: "c"}
	repo := &fakeRepo{
		fnGetComment:        func(context.Context, bson.ObjectID) (*models.Comment, error) { return comment, nil },
		fnRemoveCommentLike: func(context.Context, string, string) (bool, error) { return true, nil },
		fnIncCommentCounter: func(context.Context, bson.ObjectID, string, int32) (*models.Comment, error) {
			return &models.Comment{LikesCount: 0}, nil
		},
	}
	s := NewPostService(repo)
	if n, err := s.UnlikeComment(bg(), postID, comment.ID.Hex(), "l"); err != nil || n != 0 {
		t.Fatalf("UnlikeComment: %d / %v", n, err)
	}
}

func TestCommentStats(t *testing.T) {
	id := newOID()
	repo := &fakeRepo{fnCommentStatsByIDs: func(context.Context, []bson.ObjectID) ([]models.Comment, error) {
		return []models.Comment{{ID: id, LikesCount: 4}}, nil
	}}
	s := NewPostService(repo)
	stats, err := s.CommentStats(bg(), []string{id.Hex(), "bad"})
	if err != nil || len(stats) != 1 || stats[0].LikesCount != 4 {
		t.Fatalf("CommentStats: %v / %+v", err, stats)
	}
}

func TestListComments(t *testing.T) {
	repo := &fakeRepo{fnListComments: func(context.Context, string, int64, int64) ([]models.Comment, error) {
		return []models.Comment{{ID: newOID()}}, nil
	}}
	s := NewPostService(repo)
	if c, err := s.ListComments(bg(), newOID().Hex(), "", 0, 0); err != nil || len(c) != 1 {
		t.Fatalf("ListComments: %v / %d", err, len(c))
	}
	if _, err := s.ListComments(bg(), "bad", "", 0, 0); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("attendu ErrInvalidID, got %v", err)
	}
}

func TestListReplies(t *testing.T) {
	repo := &fakeRepo{fnListReplies: func(context.Context, string, int64, int64) ([]models.Comment, error) {
		return []models.Comment{{ID: newOID()}}, nil
	}}
	s := NewPostService(repo)
	if c, err := s.ListReplies(bg(), newOID().Hex(), "viewer", 0, 0); err != nil || len(c) != 1 {
		t.Fatalf("ListReplies: %v / %d", err, len(c))
	}
}

func TestListCommentsByAuthor(t *testing.T) {
	postID := newOID()
	repo := &fakeRepo{
		fnListCommentsByAuthor: func(context.Context, string, int64, int64) ([]models.Comment, error) {
			return []models.Comment{{ID: newOID(), PostID: postID.Hex(), AuthorID: "author"}}, nil
		},
		fnGet: func(context.Context, bson.ObjectID) (*models.Post, error) {
			return &models.Post{ID: postID, AuthorID: "post-author"}, nil
		},
	}
	s := NewPostService(repo)
	items, err := s.ListCommentsByAuthor(bg(), "author", "viewer", 0, 0)
	if err != nil || len(items) != 1 {
		t.Fatalf("ListCommentsByAuthor: %v / %d", err, len(items))
	}
}

// ─── Hydratation des likes de commentaires (viewer) ─────────────────────────

func TestListComments_HydratesLiked(t *testing.T) {
	cid := newOID()
	repo := &fakeRepo{
		fnListComments: func(context.Context, string, int64, int64) ([]models.Comment, error) {
			return []models.Comment{{ID: cid}}, nil
		},
		fnLikedCommentIDsByUser: func(context.Context, string, []string) ([]string, error) { return []string{cid.Hex()}, nil },
	}
	s := NewPostService(repo)
	c, err := s.ListComments(bg(), newOID().Hex(), "viewer", 0, 0)
	if err != nil || !c[0].Liked {
		t.Fatalf("le commentaire liké doit être marqué Liked: %v / %+v", err, c)
	}
}
