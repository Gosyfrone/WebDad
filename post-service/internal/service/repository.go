package service

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/webdad/post-service/internal/models"
	"github.com/webdad/post-service/internal/repository"
)

// Repository abstrait le dépôt de persistance consommé par PostService. Définie
// côté consommateur (idiome Go) pour permettre l'injection d'un faux dépôt en
// test unitaire : la logique métier est ainsi couverte sans MongoDB. Le type
// concret *repository.PostRepository l'implémente structurellement.
//
// Les tests d'intégration (vrai Mongo) couvrent l'implémentation concrète.
type Repository interface {
	// Posts — CRUD et listes.
	Create(ctx context.Context, post *models.Post) error
	Get(ctx context.Context, id bson.ObjectID) (*models.Post, error)
	Delete(ctx context.Context, id bson.ObjectID) error
	Update(ctx context.Context, id bson.ObjectID, content string, hashtags []string) (*models.Post, error)
	GetAll(ctx context.Context, limit, skip int64) ([]models.Post, error)
	GetAllByHashtag(ctx context.Context, hashtag, sortMode string, limit, skip int64) ([]models.Post, error)
	GetAllWithHashtags(ctx context.Context, limit, skip int64) ([]models.Post, error)
	GetByProfileHashtag(ctx context.Context, authorID, hashtag string, limit, skip int64) ([]models.Post, error)
	GetByAuthorsHashtag(ctx context.Context, authorIDs []string, hashtag, sortMode string, limit, skip int64) ([]models.Post, error)
	StatsByIDs(ctx context.Context, oids []bson.ObjectID) ([]models.Post, error)

	// Modération / cycle de vie.
	Hide(ctx context.Context, id bson.ObjectID, byUserID string, at time.Time) (*models.Post, error)
	RestoreHidden(ctx context.Context, id bson.ObjectID) (*models.Post, error)
	SetAutoHidden(ctx context.Context, id bson.ObjectID, hidden bool) (*models.Post, error)
	SetNsfw(ctx context.Context, id bson.ObjectID, nsfw bool, byUserID string, at time.Time) (*models.Post, error)
	ListHidden(ctx context.Context, f repository.HiddenFilter, limit, skip int64) ([]models.Post, error)
	ListPurgeable(ctx context.Context, before time.Time, limit int64) ([]models.Post, error)
	ListPurgeWarnable(ctx context.Context, before time.Time, limit int64) ([]models.Post, error)
	MarkPurgeWarned(ctx context.Context, id bson.ObjectID, at time.Time) error
	PurgeByAuthor(ctx context.Context, userID string) (int64, error)

	// Épingles.
	Pin(ctx context.Context, id bson.ObjectID, pinnedAt time.Time) (*models.Post, error)
	Unpin(ctx context.Context, id bson.ObjectID) (*models.Post, error)
	UnpinByAuthor(ctx context.Context, authorID string) error

	// Compteurs et sondages.
	IncCounter(ctx context.Context, id bson.ObjectID, field string, delta int32) (*models.Post, error)
	AddPollVote(ctx context.Context, postID, userID, choiceID string) (bool, error)
	PollVoteChoice(ctx context.Context, postID, userID string) (string, error)
	IncPollChoice(ctx context.Context, id bson.ObjectID, choiceID string) (*models.Post, error)
	ClosePoll(ctx context.Context, id bson.ObjectID, at time.Time) (*models.Post, error)
	DeletePollVotesByPost(ctx context.Context, postID string) error

	// Likes de posts.
	AddLike(ctx context.Context, postID, userID string) (bool, error)
	RemoveLike(ctx context.Context, postID, userID string) (bool, error)
	LikedPostIDs(ctx context.Context, userID string) ([]string, error)
	LikersByPost(ctx context.Context, postID string) ([]string, error)
	LikedPostsByUser(ctx context.Context, userID string, limit, offset int64) ([]*models.Post, error)
	DeleteLikesByPost(ctx context.Context, postID string) error

	// Reposts.
	AddRepost(ctx context.Context, postID, userID string) (*models.Repost, bool, error)
	RemoveRepost(ctx context.Context, postID, userID string) (bool, error)
	RepostedPostIDs(ctx context.Context, userID string) ([]string, error)
	DeleteRepostsByPost(ctx context.Context, postID string) error

	// Commentaires.
	AddComment(ctx context.Context, comment *models.Comment) error
	GetComment(ctx context.Context, id bson.ObjectID) (*models.Comment, error)
	DeleteComment(ctx context.Context, id bson.ObjectID) error
	CommentStatsByIDs(ctx context.Context, oids []bson.ObjectID) ([]models.Comment, error)
	ListComments(ctx context.Context, postID string, limit, skip int64) ([]models.Comment, error)
	ListReplies(ctx context.Context, parentID string, limit, skip int64) ([]models.Comment, error)
	ListCommentsByAuthor(ctx context.Context, authorID string, limit, skip int64) ([]models.Comment, error)
	IncReplyCount(ctx context.Context, id bson.ObjectID, delta int32) error
	IncCommentCounter(ctx context.Context, id bson.ObjectID, field string, delta int32) (*models.Comment, error)
	DeleteRepliesByParent(ctx context.Context, parentID string) (int64, error)
	CommentIDsByParent(ctx context.Context, parentID string) ([]string, error)
	DeleteCommentsByPost(ctx context.Context, postID string) error

	// Likes de commentaires.
	AddCommentLike(ctx context.Context, commentID, userID string) (bool, error)
	RemoveCommentLike(ctx context.Context, commentID, userID string) (bool, error)
	LikedCommentIDsByUser(ctx context.Context, userID string, commentIDs []string) ([]string, error)
	DeleteCommentLikesByComment(ctx context.Context, commentID string) error
	DeleteCommentLikesByComments(ctx context.Context, commentIDs []string) error

	// Signets — collections.
	CreateCollection(ctx context.Context, coll *models.BookmarkCollection) error
	GetCollection(ctx context.Context, id bson.ObjectID) (*models.BookmarkCollection, error)
	EnsureDefaultCollection(ctx context.Context, userID string) (*models.BookmarkCollection, error)
	ListCollections(ctx context.Context, userID string) ([]models.BookmarkCollection, error)
	RenameCollection(ctx context.Context, id bson.ObjectID, name string) (*models.BookmarkCollection, error)
	DeleteCollection(ctx context.Context, id bson.ObjectID) error

	// Signets — entrées.
	CountBookmarks(ctx context.Context, userID, collectionID string) (int64, error)
	AddBookmark(ctx context.Context, userID, postID, collectionID string) (bool, error)
	RemoveBookmark(ctx context.Context, userID, postID, collectionID string) (bool, error)
	RemoveAllBookmarksForPost(ctx context.Context, userID, postID string) (int64, error)
	BookmarkedPostIDs(ctx context.Context, userID string) ([]string, error)
	PostBookmarkCollectionIDs(ctx context.Context, userID, postID string) ([]string, error)
	BookmarksByCollection(ctx context.Context, userID, collectionID string, limit, skip int64) ([]string, error)
	AllBookmarkedPostIDs(ctx context.Context, userID string, limit, skip int64) ([]string, error)
	DeleteBookmarksByPost(ctx context.Context, postID string) error
	DeleteBookmarksByCollection(ctx context.Context, collectionID string) error

	// Signets — préférences (fenêtre de rafale).
	GetPrefs(ctx context.Context, userID string) (*models.BookmarkPrefs, error)
	UpsertPrefs(ctx context.Context, userID, collectionID string, at time.Time) error
}

// Vérifie à la compilation que l'implémentation concrète satisfait l'interface.
var _ Repository = (*repository.PostRepository)(nil)
