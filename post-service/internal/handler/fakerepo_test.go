package handler

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/webdad/post-service/internal/models"
	"github.com/webdad/post-service/internal/realtime"
	"github.com/webdad/post-service/internal/repository"
	"github.com/webdad/post-service/internal/service"

	"github.com/gin-gonic/gin"
)

// emptyRepo implémente service.Repository en retournant des valeurs zéro (nil,
// nil) pour toutes les méthodes. Convient aux chemins de succès des handlers
// qui retournent des listes vides (pas de boucle → pas de nil-deref dans le
// service). Pour les méthodes retournant un pointeur consommé ensuite (Get,
// IncCounter…), le service panique → gin.Recovery → 500 ; ces cas sont couverts
// par les tests NilRepo et IDInvalide existants.
type emptyRepo struct{}

// ─── Posts ───────────────────────────────────────────────────────────────────

func (r *emptyRepo) Create(ctx context.Context, post *models.Post) error { return nil }
func (r *emptyRepo) Get(ctx context.Context, id bson.ObjectID) (*models.Post, error) {
	return nil, nil
}
func (r *emptyRepo) Delete(ctx context.Context, id bson.ObjectID) error { return nil }
func (r *emptyRepo) Update(ctx context.Context, id bson.ObjectID, content string, hashtags []string) (*models.Post, error) {
	return nil, nil
}
func (r *emptyRepo) GetAll(ctx context.Context, limit, skip int64) ([]models.Post, error) {
	return nil, nil
}
func (r *emptyRepo) GetAllByHashtag(ctx context.Context, hashtag, sortMode string, limit, skip int64) ([]models.Post, error) {
	return nil, nil
}
func (r *emptyRepo) GetAllWithHashtags(ctx context.Context, limit, skip int64) ([]models.Post, error) {
	return nil, nil
}
func (r *emptyRepo) GetByProfileHashtag(ctx context.Context, authorID, hashtag string, limit, skip int64) ([]models.Post, error) {
	return nil, nil
}
func (r *emptyRepo) GetByAuthorsHashtag(ctx context.Context, authorIDs []string, hashtag, sortMode string, limit, skip int64) ([]models.Post, error) {
	return nil, nil
}
func (r *emptyRepo) StatsByIDs(ctx context.Context, oids []bson.ObjectID) ([]models.Post, error) {
	return nil, nil
}

// ─── Modération ──────────────────────────────────────────────────────────────

func (r *emptyRepo) Hide(ctx context.Context, id bson.ObjectID, byUserID string, at time.Time) (*models.Post, error) {
	return nil, nil
}
func (r *emptyRepo) RestoreHidden(ctx context.Context, id bson.ObjectID) (*models.Post, error) {
	return nil, nil
}
func (r *emptyRepo) SetAutoHidden(ctx context.Context, id bson.ObjectID, hidden bool) (*models.Post, error) {
	return nil, nil
}
func (r *emptyRepo) SetNsfw(ctx context.Context, id bson.ObjectID, nsfw bool, byUserID string, at time.Time) (*models.Post, error) {
	return nil, nil
}
func (r *emptyRepo) ListHidden(ctx context.Context, f repository.HiddenFilter, limit, skip int64) ([]models.Post, error) {
	return nil, nil
}
func (r *emptyRepo) ListPurgeable(ctx context.Context, before time.Time, limit int64) ([]models.Post, error) {
	return nil, nil
}
func (r *emptyRepo) ListPurgeWarnable(ctx context.Context, before time.Time, limit int64) ([]models.Post, error) {
	return nil, nil
}
func (r *emptyRepo) MarkPurgeWarned(ctx context.Context, id bson.ObjectID, at time.Time) error {
	return nil
}
func (r *emptyRepo) PurgeByAuthor(ctx context.Context, userID string) (int64, error) {
	return 0, nil
}

// ─── Épingles ─────────────────────────────────────────────────────────────────

func (r *emptyRepo) Pin(ctx context.Context, id bson.ObjectID, pinnedAt time.Time) (*models.Post, error) {
	return nil, nil
}
func (r *emptyRepo) Unpin(ctx context.Context, id bson.ObjectID) (*models.Post, error) {
	return nil, nil
}
func (r *emptyRepo) UnpinByAuthor(ctx context.Context, authorID string) error { return nil }

// ─── Compteurs et sondages ───────────────────────────────────────────────────

func (r *emptyRepo) IncCounter(ctx context.Context, id bson.ObjectID, field string, delta int32) (*models.Post, error) {
	return nil, nil
}
func (r *emptyRepo) AddPollVote(ctx context.Context, postID, userID, choiceID string) (bool, error) {
	return false, nil
}
func (r *emptyRepo) PollVoteChoice(ctx context.Context, postID, userID string) (string, error) {
	return "", nil
}
func (r *emptyRepo) IncPollChoice(ctx context.Context, id bson.ObjectID, choiceID string) (*models.Post, error) {
	return nil, nil
}
func (r *emptyRepo) ClosePoll(ctx context.Context, id bson.ObjectID, at time.Time) (*models.Post, error) {
	return nil, nil
}
func (r *emptyRepo) DeletePollVotesByPost(ctx context.Context, postID string) error { return nil }

// ─── Likes de posts ───────────────────────────────────────────────────────────

func (r *emptyRepo) AddLike(ctx context.Context, postID, userID string) (bool, error) {
	return false, nil
}
func (r *emptyRepo) RemoveLike(ctx context.Context, postID, userID string) (bool, error) {
	return false, nil
}
func (r *emptyRepo) LikedPostIDs(ctx context.Context, userID string) ([]string, error) {
	return nil, nil
}
func (r *emptyRepo) LikersByPost(ctx context.Context, postID string) ([]string, error) {
	return nil, nil
}
func (r *emptyRepo) LikedPostsByUser(ctx context.Context, userID string, limit, offset int64) ([]*models.Post, error) {
	return nil, nil
}
func (r *emptyRepo) DeleteLikesByPost(ctx context.Context, postID string) error { return nil }

// ─── Reposts ──────────────────────────────────────────────────────────────────

func (r *emptyRepo) AddRepost(ctx context.Context, postID, userID string) (*models.Repost, bool, error) {
	return nil, false, nil
}
func (r *emptyRepo) RemoveRepost(ctx context.Context, postID, userID string) (bool, error) {
	return false, nil
}
func (r *emptyRepo) RepostedPostIDs(ctx context.Context, userID string) ([]string, error) {
	return nil, nil
}
func (r *emptyRepo) DeleteRepostsByPost(ctx context.Context, postID string) error { return nil }

// ─── Commentaires ─────────────────────────────────────────────────────────────

func (r *emptyRepo) AddComment(ctx context.Context, comment *models.Comment) error { return nil }
func (r *emptyRepo) GetComment(ctx context.Context, id bson.ObjectID) (*models.Comment, error) {
	return nil, nil
}
func (r *emptyRepo) DeleteComment(ctx context.Context, id bson.ObjectID) error { return nil }
func (r *emptyRepo) CommentStatsByIDs(ctx context.Context, oids []bson.ObjectID) ([]models.Comment, error) {
	return nil, nil
}
func (r *emptyRepo) ListComments(ctx context.Context, postID string, limit, skip int64) ([]models.Comment, error) {
	return nil, nil
}
func (r *emptyRepo) ListReplies(ctx context.Context, parentID string, limit, skip int64) ([]models.Comment, error) {
	return nil, nil
}
func (r *emptyRepo) ListCommentsByAuthor(ctx context.Context, authorID string, limit, skip int64) ([]models.Comment, error) {
	return nil, nil
}
func (r *emptyRepo) IncReplyCount(ctx context.Context, id bson.ObjectID, delta int32) error {
	return nil
}
func (r *emptyRepo) IncCommentCounter(ctx context.Context, id bson.ObjectID, field string, delta int32) (*models.Comment, error) {
	return nil, nil
}
func (r *emptyRepo) DeleteRepliesByParent(ctx context.Context, parentID string) (int64, error) {
	return 0, nil
}
func (r *emptyRepo) CommentIDsByParent(ctx context.Context, parentID string) ([]string, error) {
	return nil, nil
}
func (r *emptyRepo) DeleteCommentsByPost(ctx context.Context, postID string) error { return nil }

// ─── Likes de commentaires ───────────────────────────────────────────────────

func (r *emptyRepo) AddCommentLike(ctx context.Context, commentID, userID string) (bool, error) {
	return false, nil
}
func (r *emptyRepo) RemoveCommentLike(ctx context.Context, commentID, userID string) (bool, error) {
	return false, nil
}
func (r *emptyRepo) LikedCommentIDsByUser(ctx context.Context, userID string, commentIDs []string) ([]string, error) {
	return nil, nil
}
func (r *emptyRepo) DeleteCommentLikesByComment(ctx context.Context, commentID string) error {
	return nil
}
func (r *emptyRepo) DeleteCommentLikesByComments(ctx context.Context, commentIDs []string) error {
	return nil
}

// ─── Signets — collections ───────────────────────────────────────────────────

func (r *emptyRepo) CreateCollection(ctx context.Context, coll *models.BookmarkCollection) error {
	return nil
}
func (r *emptyRepo) GetCollection(ctx context.Context, id bson.ObjectID) (*models.BookmarkCollection, error) {
	return nil, nil
}
func (r *emptyRepo) EnsureDefaultCollection(ctx context.Context, userID string) (*models.BookmarkCollection, error) {
	return nil, nil
}
func (r *emptyRepo) ListCollections(ctx context.Context, userID string) ([]models.BookmarkCollection, error) {
	return nil, nil
}
func (r *emptyRepo) RenameCollection(ctx context.Context, id bson.ObjectID, name string) (*models.BookmarkCollection, error) {
	return nil, nil
}
func (r *emptyRepo) DeleteCollection(ctx context.Context, id bson.ObjectID) error { return nil }

// ─── Signets — entrées ───────────────────────────────────────────────────────

func (r *emptyRepo) CountBookmarks(ctx context.Context, userID, collectionID string) (int64, error) {
	return 0, nil
}
func (r *emptyRepo) AddBookmark(ctx context.Context, userID, postID, collectionID string) (bool, error) {
	return false, nil
}
func (r *emptyRepo) RemoveBookmark(ctx context.Context, userID, postID, collectionID string) (bool, error) {
	return false, nil
}
func (r *emptyRepo) RemoveAllBookmarksForPost(ctx context.Context, userID, postID string) (int64, error) {
	return 0, nil
}
func (r *emptyRepo) BookmarkedPostIDs(ctx context.Context, userID string) ([]string, error) {
	return nil, nil
}
func (r *emptyRepo) PostBookmarkCollectionIDs(ctx context.Context, userID, postID string) ([]string, error) {
	return nil, nil
}
func (r *emptyRepo) BookmarksByCollection(ctx context.Context, userID, collectionID string, limit, skip int64) ([]string, error) {
	return nil, nil
}
func (r *emptyRepo) AllBookmarkedPostIDs(ctx context.Context, userID string, limit, skip int64) ([]string, error) {
	return nil, nil
}
func (r *emptyRepo) DeleteBookmarksByPost(ctx context.Context, postID string) error { return nil }
func (r *emptyRepo) DeleteBookmarksByCollection(ctx context.Context, collectionID string) error {
	return nil
}

// ─── Signets — préférences ───────────────────────────────────────────────────

func (r *emptyRepo) GetPrefs(ctx context.Context, userID string) (*models.BookmarkPrefs, error) {
	return nil, nil
}
func (r *emptyRepo) UpsertPrefs(ctx context.Context, userID, collectionID string, at time.Time) error {
	return nil
}

// ─── Router avec emptyRepo ───────────────────────────────────────────────────

// newEmptyRepoRouter monte les routes avec un PostService à emptyRepo.
// Les méthodes retournant des listes vides permettent de couvrir les chemins
// de succès (c.JSON 200 / c.Status 204) sans MongoDB.
// ATTENTION : les méthodes retournant un *models.Post (Get, IncCounter…) renvoient
// nil, ce qui provoque un nil-deref dans le service → gin.Recovery → 500.
func newEmptyRepoRouter(t interface{ Helper() }) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	svc := service.NewPostService(&emptyRepo{})
	RegisterRoutes(r, "post-test", svc, nilTestSecret, realtime.NewHub(), nil, nilInternalSec)
	return r
}
