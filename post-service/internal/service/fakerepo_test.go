package service

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/webdad/post-service/internal/models"
	"github.com/webdad/post-service/internal/notifier"
	"github.com/webdad/post-service/internal/repository"
)

// fakeRepo est une implémentation en mémoire de service.Repository, pilotée par
// des champs fonction. Un champ non renseigné renvoie la valeur zéro et nil :
// chaque test ne câble que les méthodes qu'il exerce. Permet de couvrir la
// logique métier de PostService sans MongoDB.
type fakeRepo struct {
	// Posts.
	fnCreate              func(context.Context, *models.Post) error
	fnGet                 func(context.Context, bson.ObjectID) (*models.Post, error)
	fnDelete              func(context.Context, bson.ObjectID) error
	fnUpdate              func(context.Context, bson.ObjectID, string, []string) (*models.Post, error)
	fnGetAll              func(context.Context, int64, int64) ([]models.Post, error)
	fnGetAllByHashtag     func(context.Context, string, string, int64, int64) ([]models.Post, error)
	fnGetAllWithHashtags  func(context.Context, int64, int64) ([]models.Post, error)
	fnGetByProfileHashtag func(context.Context, string, string, int64, int64) ([]models.Post, error)
	fnGetByAuthorsHashtag func(context.Context, []string, string, string, int64, int64) ([]models.Post, error)
	fnStatsByIDs          func(context.Context, []bson.ObjectID) ([]models.Post, error)

	// Modération / cycle de vie.
	fnHide              func(context.Context, bson.ObjectID, string, time.Time) (*models.Post, error)
	fnRestoreHidden     func(context.Context, bson.ObjectID) (*models.Post, error)
	fnSetAutoHidden     func(context.Context, bson.ObjectID, bool) (*models.Post, error)
	fnSetNsfw           func(context.Context, bson.ObjectID, bool, string, time.Time) (*models.Post, error)
	fnListHidden        func(context.Context, repository.HiddenFilter, int64, int64) ([]models.Post, error)
	fnListPurgeable     func(context.Context, time.Time, int64) ([]models.Post, error)
	fnListPurgeWarnable func(context.Context, time.Time, int64) ([]models.Post, error)
	fnMarkPurgeWarned   func(context.Context, bson.ObjectID, time.Time) error
	fnPurgeByAuthor     func(context.Context, string) (int64, error)

	// Épingles.
	fnPin           func(context.Context, bson.ObjectID, time.Time) (*models.Post, error)
	fnUnpin         func(context.Context, bson.ObjectID) (*models.Post, error)
	fnUnpinByAuthor func(context.Context, string) error

	// Compteurs / sondages.
	fnIncCounter            func(context.Context, bson.ObjectID, string, int32) (*models.Post, error)
	fnAddPollVote           func(context.Context, string, string, string) (bool, error)
	fnPollVoteChoice        func(context.Context, string, string) (string, error)
	fnIncPollChoice         func(context.Context, bson.ObjectID, string) (*models.Post, error)
	fnClosePoll             func(context.Context, bson.ObjectID, time.Time) (*models.Post, error)
	fnDeletePollVotesByPost func(context.Context, string) error

	// Likes de posts.
	fnAddLike           func(context.Context, string, string) (bool, error)
	fnRemoveLike        func(context.Context, string, string) (bool, error)
	fnLikedPostIDs      func(context.Context, string) ([]string, error)
	fnLikersByPost      func(context.Context, string) ([]string, error)
	fnLikedPostsByUser  func(context.Context, string, int64, int64) ([]*models.Post, error)
	fnDeleteLikesByPost func(context.Context, string) error

	// Reposts.
	fnAddRepost           func(context.Context, string, string) (*models.Repost, bool, error)
	fnRemoveRepost        func(context.Context, string, string) (bool, error)
	fnRepostedPostIDs     func(context.Context, string) ([]string, error)
	fnDeleteRepostsByPost func(context.Context, string) error

	// Commentaires.
	fnAddComment            func(context.Context, *models.Comment) error
	fnGetComment            func(context.Context, bson.ObjectID) (*models.Comment, error)
	fnDeleteComment         func(context.Context, bson.ObjectID) error
	fnCommentStatsByIDs     func(context.Context, []bson.ObjectID) ([]models.Comment, error)
	fnListComments          func(context.Context, string, int64, int64) ([]models.Comment, error)
	fnListReplies           func(context.Context, string, int64, int64) ([]models.Comment, error)
	fnListCommentsByAuthor  func(context.Context, string, int64, int64) ([]models.Comment, error)
	fnIncReplyCount         func(context.Context, bson.ObjectID, int32) error
	fnIncCommentCounter     func(context.Context, bson.ObjectID, string, int32) (*models.Comment, error)
	fnDeleteRepliesByParent func(context.Context, string) (int64, error)
	fnCommentIDsByParent    func(context.Context, string) ([]string, error)
	fnDeleteCommentsByPost  func(context.Context, string) error

	// Likes de commentaires.
	fnAddCommentLike               func(context.Context, string, string) (bool, error)
	fnRemoveCommentLike            func(context.Context, string, string) (bool, error)
	fnLikedCommentIDsByUser        func(context.Context, string, []string) ([]string, error)
	fnDeleteCommentLikesByComment  func(context.Context, string) error
	fnDeleteCommentLikesByComments func(context.Context, []string) error

	// Signets — collections.
	fnCreateCollection        func(context.Context, *models.BookmarkCollection) error
	fnGetCollection           func(context.Context, bson.ObjectID) (*models.BookmarkCollection, error)
	fnEnsureDefaultCollection func(context.Context, string) (*models.BookmarkCollection, error)
	fnListCollections         func(context.Context, string) ([]models.BookmarkCollection, error)
	fnRenameCollection        func(context.Context, bson.ObjectID, string) (*models.BookmarkCollection, error)
	fnDeleteCollection        func(context.Context, bson.ObjectID) error

	// Signets — entrées.
	fnCountBookmarks              func(context.Context, string, string) (int64, error)
	fnAddBookmark                 func(context.Context, string, string, string) (bool, error)
	fnRemoveBookmark              func(context.Context, string, string, string) (bool, error)
	fnRemoveAllBookmarksForPost   func(context.Context, string, string) (int64, error)
	fnBookmarkedPostIDs           func(context.Context, string) ([]string, error)
	fnPostBookmarkCollectionIDs   func(context.Context, string, string) ([]string, error)
	fnBookmarksByCollection       func(context.Context, string, string, int64, int64) ([]string, error)
	fnAllBookmarkedPostIDs        func(context.Context, string, int64, int64) ([]string, error)
	fnDeleteBookmarksByPost       func(context.Context, string) error
	fnDeleteBookmarksByCollection func(context.Context, string) error

	// Signets — préférences.
	fnGetPrefs    func(context.Context, string) (*models.BookmarkPrefs, error)
	fnUpsertPrefs func(context.Context, string, string, time.Time) error
}

func (f *fakeRepo) Create(ctx context.Context, p *models.Post) error {
	if f.fnCreate != nil {
		return f.fnCreate(ctx, p)
	}
	return nil
}

func (f *fakeRepo) Get(ctx context.Context, id bson.ObjectID) (*models.Post, error) {
	if f.fnGet != nil {
		return f.fnGet(ctx, id)
	}
	return nil, nil
}

func (f *fakeRepo) Delete(ctx context.Context, id bson.ObjectID) error {
	if f.fnDelete != nil {
		return f.fnDelete(ctx, id)
	}
	return nil
}

func (f *fakeRepo) Update(ctx context.Context, id bson.ObjectID, c string, h []string) (*models.Post, error) {
	if f.fnUpdate != nil {
		return f.fnUpdate(ctx, id, c, h)
	}
	return nil, nil
}

func (f *fakeRepo) GetAll(ctx context.Context, l, s int64) ([]models.Post, error) {
	if f.fnGetAll != nil {
		return f.fnGetAll(ctx, l, s)
	}
	return nil, nil
}

func (f *fakeRepo) GetAllByHashtag(ctx context.Context, h, sm string, l, s int64) ([]models.Post, error) {
	if f.fnGetAllByHashtag != nil {
		return f.fnGetAllByHashtag(ctx, h, sm, l, s)
	}
	return nil, nil
}

func (f *fakeRepo) GetAllWithHashtags(ctx context.Context, l, s int64) ([]models.Post, error) {
	if f.fnGetAllWithHashtags != nil {
		return f.fnGetAllWithHashtags(ctx, l, s)
	}
	return nil, nil
}

func (f *fakeRepo) GetByProfileHashtag(ctx context.Context, a, h string, l, s int64) ([]models.Post, error) {
	if f.fnGetByProfileHashtag != nil {
		return f.fnGetByProfileHashtag(ctx, a, h, l, s)
	}
	return nil, nil
}

func (f *fakeRepo) GetByAuthorsHashtag(ctx context.Context, a []string, h, sm string, l, s int64) ([]models.Post, error) {
	if f.fnGetByAuthorsHashtag != nil {
		return f.fnGetByAuthorsHashtag(ctx, a, h, sm, l, s)
	}
	return nil, nil
}

func (f *fakeRepo) StatsByIDs(ctx context.Context, o []bson.ObjectID) ([]models.Post, error) {
	if f.fnStatsByIDs != nil {
		return f.fnStatsByIDs(ctx, o)
	}
	return nil, nil
}

func (f *fakeRepo) Hide(ctx context.Context, id bson.ObjectID, by string, at time.Time) (*models.Post, error) {
	if f.fnHide != nil {
		return f.fnHide(ctx, id, by, at)
	}
	return nil, nil
}

func (f *fakeRepo) RestoreHidden(ctx context.Context, id bson.ObjectID) (*models.Post, error) {
	if f.fnRestoreHidden != nil {
		return f.fnRestoreHidden(ctx, id)
	}
	return nil, nil
}

func (f *fakeRepo) SetAutoHidden(ctx context.Context, id bson.ObjectID, h bool) (*models.Post, error) {
	if f.fnSetAutoHidden != nil {
		return f.fnSetAutoHidden(ctx, id, h)
	}
	return nil, nil
}

func (f *fakeRepo) SetNsfw(ctx context.Context, id bson.ObjectID, n bool, by string, at time.Time) (*models.Post, error) {
	if f.fnSetNsfw != nil {
		return f.fnSetNsfw(ctx, id, n, by, at)
	}
	return nil, nil
}

func (f *fakeRepo) ListHidden(ctx context.Context, ft repository.HiddenFilter, l, s int64) ([]models.Post, error) {
	if f.fnListHidden != nil {
		return f.fnListHidden(ctx, ft, l, s)
	}
	return nil, nil
}

func (f *fakeRepo) ListPurgeable(ctx context.Context, b time.Time, l int64) ([]models.Post, error) {
	if f.fnListPurgeable != nil {
		return f.fnListPurgeable(ctx, b, l)
	}
	return nil, nil
}

func (f *fakeRepo) ListPurgeWarnable(ctx context.Context, b time.Time, l int64) ([]models.Post, error) {
	if f.fnListPurgeWarnable != nil {
		return f.fnListPurgeWarnable(ctx, b, l)
	}
	return nil, nil
}

func (f *fakeRepo) MarkPurgeWarned(ctx context.Context, id bson.ObjectID, at time.Time) error {
	if f.fnMarkPurgeWarned != nil {
		return f.fnMarkPurgeWarned(ctx, id, at)
	}
	return nil
}

func (f *fakeRepo) PurgeByAuthor(ctx context.Context, u string) (int64, error) {
	if f.fnPurgeByAuthor != nil {
		return f.fnPurgeByAuthor(ctx, u)
	}
	return 0, nil
}

func (f *fakeRepo) Pin(ctx context.Context, id bson.ObjectID, at time.Time) (*models.Post, error) {
	if f.fnPin != nil {
		return f.fnPin(ctx, id, at)
	}
	return nil, nil
}

func (f *fakeRepo) Unpin(ctx context.Context, id bson.ObjectID) (*models.Post, error) {
	if f.fnUnpin != nil {
		return f.fnUnpin(ctx, id)
	}
	return nil, nil
}

func (f *fakeRepo) UnpinByAuthor(ctx context.Context, a string) error {
	if f.fnUnpinByAuthor != nil {
		return f.fnUnpinByAuthor(ctx, a)
	}
	return nil
}

func (f *fakeRepo) IncCounter(ctx context.Context, id bson.ObjectID, fld string, d int32) (*models.Post, error) {
	if f.fnIncCounter != nil {
		return f.fnIncCounter(ctx, id, fld, d)
	}
	return nil, nil
}

func (f *fakeRepo) AddPollVote(ctx context.Context, p, u, c string) (bool, error) {
	if f.fnAddPollVote != nil {
		return f.fnAddPollVote(ctx, p, u, c)
	}
	return false, nil
}

func (f *fakeRepo) PollVoteChoice(ctx context.Context, p, u string) (string, error) {
	if f.fnPollVoteChoice != nil {
		return f.fnPollVoteChoice(ctx, p, u)
	}
	return "", nil
}

func (f *fakeRepo) IncPollChoice(ctx context.Context, id bson.ObjectID, c string) (*models.Post, error) {
	if f.fnIncPollChoice != nil {
		return f.fnIncPollChoice(ctx, id, c)
	}
	return nil, nil
}

func (f *fakeRepo) ClosePoll(ctx context.Context, id bson.ObjectID, at time.Time) (*models.Post, error) {
	if f.fnClosePoll != nil {
		return f.fnClosePoll(ctx, id, at)
	}
	return nil, nil
}

func (f *fakeRepo) DeletePollVotesByPost(ctx context.Context, p string) error {
	if f.fnDeletePollVotesByPost != nil {
		return f.fnDeletePollVotesByPost(ctx, p)
	}
	return nil
}

func (f *fakeRepo) AddLike(ctx context.Context, p, u string) (bool, error) {
	if f.fnAddLike != nil {
		return f.fnAddLike(ctx, p, u)
	}
	return false, nil
}

func (f *fakeRepo) RemoveLike(ctx context.Context, p, u string) (bool, error) {
	if f.fnRemoveLike != nil {
		return f.fnRemoveLike(ctx, p, u)
	}
	return false, nil
}

func (f *fakeRepo) LikedPostIDs(ctx context.Context, u string) ([]string, error) {
	if f.fnLikedPostIDs != nil {
		return f.fnLikedPostIDs(ctx, u)
	}
	return nil, nil
}

func (f *fakeRepo) LikersByPost(ctx context.Context, p string) ([]string, error) {
	if f.fnLikersByPost != nil {
		return f.fnLikersByPost(ctx, p)
	}
	return nil, nil
}

func (f *fakeRepo) LikedPostsByUser(ctx context.Context, u string, l, o int64) ([]*models.Post, error) {
	if f.fnLikedPostsByUser != nil {
		return f.fnLikedPostsByUser(ctx, u, l, o)
	}
	return nil, nil
}

func (f *fakeRepo) DeleteLikesByPost(ctx context.Context, p string) error {
	if f.fnDeleteLikesByPost != nil {
		return f.fnDeleteLikesByPost(ctx, p)
	}
	return nil
}

func (f *fakeRepo) AddRepost(ctx context.Context, p, u string) (*models.Repost, bool, error) {
	if f.fnAddRepost != nil {
		return f.fnAddRepost(ctx, p, u)
	}
	return nil, false, nil
}

func (f *fakeRepo) RemoveRepost(ctx context.Context, p, u string) (bool, error) {
	if f.fnRemoveRepost != nil {
		return f.fnRemoveRepost(ctx, p, u)
	}
	return false, nil
}

func (f *fakeRepo) RepostedPostIDs(ctx context.Context, u string) ([]string, error) {
	if f.fnRepostedPostIDs != nil {
		return f.fnRepostedPostIDs(ctx, u)
	}
	return nil, nil
}

func (f *fakeRepo) DeleteRepostsByPost(ctx context.Context, p string) error {
	if f.fnDeleteRepostsByPost != nil {
		return f.fnDeleteRepostsByPost(ctx, p)
	}
	return nil
}

func (f *fakeRepo) AddComment(ctx context.Context, c *models.Comment) error {
	if f.fnAddComment != nil {
		return f.fnAddComment(ctx, c)
	}
	return nil
}

func (f *fakeRepo) GetComment(ctx context.Context, id bson.ObjectID) (*models.Comment, error) {
	if f.fnGetComment != nil {
		return f.fnGetComment(ctx, id)
	}
	return nil, nil
}

func (f *fakeRepo) DeleteComment(ctx context.Context, id bson.ObjectID) error {
	if f.fnDeleteComment != nil {
		return f.fnDeleteComment(ctx, id)
	}
	return nil
}

func (f *fakeRepo) CommentStatsByIDs(ctx context.Context, o []bson.ObjectID) ([]models.Comment, error) {
	if f.fnCommentStatsByIDs != nil {
		return f.fnCommentStatsByIDs(ctx, o)
	}
	return nil, nil
}

func (f *fakeRepo) ListComments(ctx context.Context, p string, l, s int64) ([]models.Comment, error) {
	if f.fnListComments != nil {
		return f.fnListComments(ctx, p, l, s)
	}
	return nil, nil
}

func (f *fakeRepo) ListReplies(ctx context.Context, p string, l, s int64) ([]models.Comment, error) {
	if f.fnListReplies != nil {
		return f.fnListReplies(ctx, p, l, s)
	}
	return nil, nil
}

func (f *fakeRepo) ListCommentsByAuthor(ctx context.Context, a string, l, s int64) ([]models.Comment, error) {
	if f.fnListCommentsByAuthor != nil {
		return f.fnListCommentsByAuthor(ctx, a, l, s)
	}
	return nil, nil
}

func (f *fakeRepo) IncReplyCount(ctx context.Context, id bson.ObjectID, d int32) error {
	if f.fnIncReplyCount != nil {
		return f.fnIncReplyCount(ctx, id, d)
	}
	return nil
}

func (f *fakeRepo) IncCommentCounter(ctx context.Context, id bson.ObjectID, fld string, d int32) (*models.Comment, error) {
	if f.fnIncCommentCounter != nil {
		return f.fnIncCommentCounter(ctx, id, fld, d)
	}
	return nil, nil
}

func (f *fakeRepo) DeleteRepliesByParent(ctx context.Context, p string) (int64, error) {
	if f.fnDeleteRepliesByParent != nil {
		return f.fnDeleteRepliesByParent(ctx, p)
	}
	return 0, nil
}

func (f *fakeRepo) CommentIDsByParent(ctx context.Context, p string) ([]string, error) {
	if f.fnCommentIDsByParent != nil {
		return f.fnCommentIDsByParent(ctx, p)
	}
	return nil, nil
}

func (f *fakeRepo) DeleteCommentsByPost(ctx context.Context, p string) error {
	if f.fnDeleteCommentsByPost != nil {
		return f.fnDeleteCommentsByPost(ctx, p)
	}
	return nil
}

func (f *fakeRepo) AddCommentLike(ctx context.Context, c, u string) (bool, error) {
	if f.fnAddCommentLike != nil {
		return f.fnAddCommentLike(ctx, c, u)
	}
	return false, nil
}

func (f *fakeRepo) RemoveCommentLike(ctx context.Context, c, u string) (bool, error) {
	if f.fnRemoveCommentLike != nil {
		return f.fnRemoveCommentLike(ctx, c, u)
	}
	return false, nil
}

func (f *fakeRepo) LikedCommentIDsByUser(ctx context.Context, u string, ids []string) ([]string, error) {
	if f.fnLikedCommentIDsByUser != nil {
		return f.fnLikedCommentIDsByUser(ctx, u, ids)
	}
	return nil, nil
}

func (f *fakeRepo) DeleteCommentLikesByComment(ctx context.Context, c string) error {
	if f.fnDeleteCommentLikesByComment != nil {
		return f.fnDeleteCommentLikesByComment(ctx, c)
	}
	return nil
}

func (f *fakeRepo) DeleteCommentLikesByComments(ctx context.Context, ids []string) error {
	if f.fnDeleteCommentLikesByComments != nil {
		return f.fnDeleteCommentLikesByComments(ctx, ids)
	}
	return nil
}

func (f *fakeRepo) CreateCollection(ctx context.Context, c *models.BookmarkCollection) error {
	if f.fnCreateCollection != nil {
		return f.fnCreateCollection(ctx, c)
	}
	return nil
}

func (f *fakeRepo) GetCollection(ctx context.Context, id bson.ObjectID) (*models.BookmarkCollection, error) {
	if f.fnGetCollection != nil {
		return f.fnGetCollection(ctx, id)
	}
	return nil, nil
}

func (f *fakeRepo) EnsureDefaultCollection(ctx context.Context, u string) (*models.BookmarkCollection, error) {
	if f.fnEnsureDefaultCollection != nil {
		return f.fnEnsureDefaultCollection(ctx, u)
	}
	return nil, nil
}

func (f *fakeRepo) ListCollections(ctx context.Context, u string) ([]models.BookmarkCollection, error) {
	if f.fnListCollections != nil {
		return f.fnListCollections(ctx, u)
	}
	return nil, nil
}

func (f *fakeRepo) RenameCollection(ctx context.Context, id bson.ObjectID, n string) (*models.BookmarkCollection, error) {
	if f.fnRenameCollection != nil {
		return f.fnRenameCollection(ctx, id, n)
	}
	return nil, nil
}

func (f *fakeRepo) DeleteCollection(ctx context.Context, id bson.ObjectID) error {
	if f.fnDeleteCollection != nil {
		return f.fnDeleteCollection(ctx, id)
	}
	return nil
}

func (f *fakeRepo) CountBookmarks(ctx context.Context, u, c string) (int64, error) {
	if f.fnCountBookmarks != nil {
		return f.fnCountBookmarks(ctx, u, c)
	}
	return 0, nil
}

func (f *fakeRepo) AddBookmark(ctx context.Context, u, p, c string) (bool, error) {
	if f.fnAddBookmark != nil {
		return f.fnAddBookmark(ctx, u, p, c)
	}
	return false, nil
}

func (f *fakeRepo) RemoveBookmark(ctx context.Context, u, p, c string) (bool, error) {
	if f.fnRemoveBookmark != nil {
		return f.fnRemoveBookmark(ctx, u, p, c)
	}
	return false, nil
}

func (f *fakeRepo) RemoveAllBookmarksForPost(ctx context.Context, u, p string) (int64, error) {
	if f.fnRemoveAllBookmarksForPost != nil {
		return f.fnRemoveAllBookmarksForPost(ctx, u, p)
	}
	return 0, nil
}

func (f *fakeRepo) BookmarkedPostIDs(ctx context.Context, u string) ([]string, error) {
	if f.fnBookmarkedPostIDs != nil {
		return f.fnBookmarkedPostIDs(ctx, u)
	}
	return nil, nil
}

func (f *fakeRepo) PostBookmarkCollectionIDs(ctx context.Context, u, p string) ([]string, error) {
	if f.fnPostBookmarkCollectionIDs != nil {
		return f.fnPostBookmarkCollectionIDs(ctx, u, p)
	}
	return nil, nil
}

func (f *fakeRepo) BookmarksByCollection(ctx context.Context, u, c string, l, s int64) ([]string, error) {
	if f.fnBookmarksByCollection != nil {
		return f.fnBookmarksByCollection(ctx, u, c, l, s)
	}
	return nil, nil
}

func (f *fakeRepo) AllBookmarkedPostIDs(ctx context.Context, u string, l, s int64) ([]string, error) {
	if f.fnAllBookmarkedPostIDs != nil {
		return f.fnAllBookmarkedPostIDs(ctx, u, l, s)
	}
	return nil, nil
}

func (f *fakeRepo) DeleteBookmarksByPost(ctx context.Context, p string) error {
	if f.fnDeleteBookmarksByPost != nil {
		return f.fnDeleteBookmarksByPost(ctx, p)
	}
	return nil
}

func (f *fakeRepo) DeleteBookmarksByCollection(ctx context.Context, c string) error {
	if f.fnDeleteBookmarksByCollection != nil {
		return f.fnDeleteBookmarksByCollection(ctx, c)
	}
	return nil
}

func (f *fakeRepo) GetPrefs(ctx context.Context, u string) (*models.BookmarkPrefs, error) {
	if f.fnGetPrefs != nil {
		return f.fnGetPrefs(ctx, u)
	}
	return nil, nil
}

func (f *fakeRepo) UpsertPrefs(ctx context.Context, u, c string, at time.Time) error {
	if f.fnUpsertPrefs != nil {
		return f.fnUpsertPrefs(ctx, u, c, at)
	}
	return nil
}

// --- Stubs de clients & notifier -------------------------------------------

// stubProfil implémente profilVisibilityClient.
type stubProfil struct {
	visibility      string
	visErr          error
	likesVisibility string
	likesErr        error
}

func (s *stubProfil) Visibility(context.Context, string) (string, error) {
	v := s.visibility
	if v == "" {
		v = "public"
	}
	return v, s.visErr
}

func (s *stubProfil) LikesVisibility(context.Context, string) (string, error) {
	v := s.likesVisibility
	if v == "" {
		v = "public"
	}
	return v, s.likesErr
}

// profilFnClient permet de piloter la visibilité par auteur (cas multi-auteurs).
type profilFnClient struct {
	vis func(context.Context, string) (string, error)
}

func (p profilFnClient) Visibility(ctx context.Context, id string) (string, error) {
	return p.vis(ctx, id)
}

func (p profilFnClient) LikesVisibility(context.Context, string) (string, error) {
	return "public", nil
}

func profilStub(fn func(context.Context, string) (string, error)) profilFnClient {
	return profilFnClient{vis: fn}
}

// capNotifier capture les événements émis pour vérification.
type capNotifier struct {
	events []notifier.Event
}

func (c *capNotifier) Emit(ev notifier.Event) { c.events = append(c.events, ev) }

func (c *capNotifier) typeCount(t string) int {
	n := 0
	for _, e := range c.events {
		if e.Type == t {
			n++
		}
	}
	return n
}

// newOID renvoie un ObjectID neuf (helper de lisibilité des tests).
func newOID() bson.ObjectID { return bson.NewObjectID() }
