package handler

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/gin-gonic/gin"
	"github.com/webdad/post-service/internal/models"
	"github.com/webdad/post-service/internal/realtime"
	"github.com/webdad/post-service/internal/repository"
	"github.com/webdad/post-service/internal/service"
)

// ─── errRepo ─────────────────────────────────────────────────────────────────
//
// errRepo embeds emptyRepo and overrides the methods most commonly called by
// handlers. It returns either mongo.ErrNoDocuments (so translateNotFound maps
// it to ErrPostNotFound → handler gets 404) or a generic sentinel error
// (hits respondPostError default → 500). Both cover the `if err != nil {
// respondPostError; return }` branches that emptyRepo and nilRepo leave uncovered.

var errGeneric = errors.New("simulated db error")

type errRepo struct{ emptyRepo }

// ─── Posts ───────────────────────────────────────────────────────────────────

func (r *errRepo) Get(_ context.Context, _ bson.ObjectID) (*models.Post, error) {
	return nil, mongo.ErrNoDocuments
}
func (r *errRepo) GetAll(_ context.Context, _, _ int64) ([]models.Post, error) {
	return nil, errGeneric
}
func (r *errRepo) GetAllByHashtag(_ context.Context, _, _ string, _, _ int64) ([]models.Post, error) {
	return nil, errGeneric
}
func (r *errRepo) GetAllWithHashtags(_ context.Context, _, _ int64) ([]models.Post, error) {
	return nil, errGeneric
}

// ─── Modération ──────────────────────────────────────────────────────────────

func (r *errRepo) ListHidden(_ context.Context, _ repository.HiddenFilter, _, _ int64) ([]models.Post, error) {
	return nil, errGeneric
}
func (r *errRepo) PurgeByAuthor(_ context.Context, _ string) (int64, error) {
	return 0, errGeneric
}

// ─── Likes ───────────────────────────────────────────────────────────────────

func (r *errRepo) AddLike(_ context.Context, _, _ string) (bool, error) {
	return false, errGeneric
}
func (r *errRepo) RemoveLike(_ context.Context, _, _ string) (bool, error) {
	return false, errGeneric
}
func (r *errRepo) LikedPostIDs(_ context.Context, _ string) ([]string, error) {
	return nil, errGeneric
}
func (r *errRepo) LikersByPost(_ context.Context, _ string) ([]string, error) {
	return nil, errGeneric
}
func (r *errRepo) LikedPostsByUser(_ context.Context, _ string, _, _ int64) ([]*models.Post, error) {
	return nil, errGeneric
}

// ─── Reposts ──────────────────────────────────────────────────────────────────

func (r *errRepo) AddRepost(_ context.Context, _, _ string) (*models.Repost, bool, error) {
	return nil, false, errGeneric
}
func (r *errRepo) RemoveRepost(_ context.Context, _, _ string) (bool, error) {
	return false, errGeneric
}
func (r *errRepo) RepostedPostIDs(_ context.Context, _ string) ([]string, error) {
	return nil, errGeneric
}

// ─── Commentaires ─────────────────────────────────────────────────────────────

func (r *errRepo) GetComment(_ context.Context, _ bson.ObjectID) (*models.Comment, error) {
	return nil, mongo.ErrNoDocuments
}
func (r *errRepo) ListComments(_ context.Context, _ string, _, _ int64) ([]models.Comment, error) {
	return nil, errGeneric
}
func (r *errRepo) ListReplies(_ context.Context, _ string, _, _ int64) ([]models.Comment, error) {
	return nil, errGeneric
}
func (r *errRepo) ListCommentsByAuthor(_ context.Context, _ string, _, _ int64) ([]models.Comment, error) {
	return nil, errGeneric
}
func (r *errRepo) CommentStatsByIDs(_ context.Context, _ []bson.ObjectID) ([]models.Comment, error) {
	return nil, errGeneric
}
func (r *errRepo) AddComment(_ context.Context, _ *models.Comment) error {
	return errGeneric
}
func (r *errRepo) AddCommentLike(_ context.Context, _, _ string) (bool, error) {
	return false, errGeneric
}
func (r *errRepo) RemoveCommentLike(_ context.Context, _, _ string) (bool, error) {
	return false, errGeneric
}

// ─── Sondages ─────────────────────────────────────────────────────────────────

func (r *errRepo) AddPollVote(_ context.Context, _, _, _ string) (bool, error) {
	return false, errGeneric
}
func (r *errRepo) PollVoteChoice(_ context.Context, _, _ string) (string, error) {
	return "", errGeneric
}

// ─── Signets ──────────────────────────────────────────────────────────────────

func (r *errRepo) AllBookmarkedPostIDs(_ context.Context, _ string, _, _ int64) ([]string, error) {
	return nil, errGeneric
}
func (r *errRepo) BookmarkedPostIDs(_ context.Context, _ string) ([]string, error) {
	return nil, errGeneric
}
func (r *errRepo) ListCollections(_ context.Context, _ string) ([]models.BookmarkCollection, error) {
	return nil, errGeneric
}
func (r *errRepo) PostBookmarkCollectionIDs(_ context.Context, _, _ string) ([]string, error) {
	return nil, errGeneric
}
func (r *errRepo) GetCollection(_ context.Context, _ bson.ObjectID) (*models.BookmarkCollection, error) {
	return nil, mongo.ErrNoDocuments
}
func (r *errRepo) EnsureDefaultCollection(_ context.Context, _ string) (*models.BookmarkCollection, error) {
	return nil, errGeneric
}
func (r *errRepo) RemoveAllBookmarksForPost(_ context.Context, _, _ string) (int64, error) {
	return 0, errGeneric
}
func (r *errRepo) RemoveBookmark(_ context.Context, _, _, _ string) (bool, error) {
	return false, errGeneric
}
func (r *errRepo) CreateCollection(_ context.Context, _ *models.BookmarkCollection) error {
	return errGeneric
}
func (r *errRepo) AddBookmark(_ context.Context, _, _, _ string) (bool, error) {
	return false, errGeneric
}

// ─── GetHashtagPosts / tendances ─────────────────────────────────────────────

func (r *errRepo) StatsByIDs(_ context.Context, _ []bson.ObjectID) ([]models.Post, error) {
	return nil, errGeneric
}
func (r *errRepo) GetByProfileHashtag(_ context.Context, _, _ string, _, _ int64) ([]models.Post, error) {
	return nil, errGeneric
}
func (r *errRepo) GetByAuthorsHashtag(_ context.Context, _ []string, _, _ string, _, _ int64) ([]models.Post, error) {
	return nil, errGeneric
}

// ─── Épingles ─────────────────────────────────────────────────────────────────

func (r *errRepo) Hide(_ context.Context, _ bson.ObjectID, _ string, _ time.Time) (*models.Post, error) {
	return nil, errGeneric
}
func (r *errRepo) RestoreHidden(_ context.Context, _ bson.ObjectID) (*models.Post, error) {
	return nil, mongo.ErrNoDocuments
}
func (r *errRepo) SetNsfw(_ context.Context, _ bson.ObjectID, _ bool, _ string, _ time.Time) (*models.Post, error) {
	return nil, errGeneric
}
func (r *errRepo) SetAutoHidden(_ context.Context, _ bson.ObjectID, _ bool) (*models.Post, error) {
	return nil, errGeneric
}

// ─── Router avec errRepo ─────────────────────────────────────────────────────

func newErrRepoRouter(t interface{ Helper() }) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	svc := service.NewPostService(&errRepo{})
	RegisterRoutes(r, "post-test", svc, nilTestSecret, realtime.NewHub(), nil, nilInternalSec)
	return r
}
