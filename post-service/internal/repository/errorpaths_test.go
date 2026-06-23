package repository

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/webdad/post-service/internal/models"
)

// Couvre les branches d'erreur Mongo (`if err != nil { return ..., err }`) de
// chaque méthode du repository. Technique : un contexte ANNULÉ fait échouer
// immédiatement la première opération Mongo de la méthode, sans dépendre d'un
// état particulier de la base. On vérifie seulement qu'une erreur remonte (la
// nature de l'erreur — driver — n'est pas du ressort du repository).
func TestRepo_ErrorPaths(t *testing.T) {
	r, ctx := newRepo(t)

	// Contexte annulé : toute opération Mongo lancée avec lui échoue.
	cc, cancel := context.WithCancel(ctx)
	cancel()

	oid := bson.NewObjectID()
	now := time.Now()

	mustErr := func(name string, err error) {
		t.Helper()
		if err == nil {
			t.Errorf("%s: erreur attendue avec un contexte annulé, got nil", name)
		}
	}

	// --- Posts ---------------------------------------------------------------
	mustErr("Create", r.Create(cc, &models.Post{AuthorID: "a", Content: "x", ReplyAudience: models.ReplyAudienceEveryone, CreatedAt: now, UpdatedAt: now}))
	_, err := r.GetAll(cc, 1, 0)
	mustErr("GetAll", err)
	_, err = r.GetAllByHashtag(cc, "go", "recent", 1, 0)
	mustErr("GetAllByHashtag", err)
	_, err = r.GetAllWithHashtags(cc, 1, 0)
	mustErr("GetAllWithHashtags", err)
	_, err = r.GetByProfile(cc, "a", 1, 0)
	mustErr("GetByProfile", err)
	_, err = r.GetByProfileHashtag(cc, "a", "go", 1, 0)
	mustErr("GetByProfileHashtag", err)
	_, err = r.GetByAuthors(cc, []string{"a"}, 1, 0)
	mustErr("GetByAuthors", err)
	_, err = r.GetByAuthorsHashtag(cc, []string{"a"}, "go", "top", 1, 0)
	mustErr("GetByAuthorsHashtag", err)
	_, err = r.ListTopHashtags(cc, 1)
	mustErr("ListTopHashtags", err)
	_, err = r.StatsByIDs(cc, []bson.ObjectID{oid})
	mustErr("StatsByIDs", err)
	_, err = r.Get(cc, oid)
	mustErr("Get", err)
	mustErr("Delete", r.Delete(cc, oid))
	_, err = r.Hide(cc, oid, "mod", now)
	mustErr("Hide", err)
	_, err = r.RestoreHidden(cc, oid)
	mustErr("RestoreHidden", err)
	_, err = r.SetAutoHidden(cc, oid, true)
	mustErr("SetAutoHidden", err)
	_, err = r.SetNsfw(cc, oid, true, "mod", now)
	mustErr("SetNsfw", err)
	since := now.Add(-time.Hour)
	_, err = r.ListHidden(cc, HiddenFilter{AuthorID: "a", Since: &since, Until: &now}, 1, 0)
	mustErr("ListHidden", err)
	_, err = r.ListPurgeable(cc, now, 1)
	mustErr("ListPurgeable", err)
	_, err = r.ListPurgeWarnable(cc, now, 1)
	mustErr("ListPurgeWarnable", err)
	mustErr("MarkPurgeWarned", r.MarkPurgeWarned(cc, oid, now))
	_, err = r.PurgeByAuthor(cc, "a")
	mustErr("PurgeByAuthor", err)
	_, err = r.Update(cc, oid, "x", []string{})
	mustErr("Update", err)
	mustErr("UnpinByAuthor", r.UnpinByAuthor(cc, "a"))
	_, err = r.Pin(cc, oid, now)
	mustErr("Pin", err)
	_, err = r.Unpin(cc, oid)
	mustErr("Unpin", err)
	_, err = r.IncCounter(cc, oid, "likes_count", 1)
	mustErr("IncCounter", err)

	// --- Sondages ------------------------------------------------------------
	_, err = r.AddPollVote(cc, "p", "u", "c")
	mustErr("AddPollVote", err)
	_, err = r.PollVoteChoice(cc, "p", "u")
	mustErr("PollVoteChoice", err)
	_, err = r.IncPollChoice(cc, oid, "c")
	mustErr("IncPollChoice", err)
	_, err = r.ClosePoll(cc, oid, now)
	mustErr("ClosePoll", err)
	mustErr("DeletePollVotesByPost", r.DeletePollVotesByPost(cc, "p"))

	// --- Likes ---------------------------------------------------------------
	_, err = r.AddLike(cc, "p", "u")
	mustErr("AddLike", err)
	_, err = r.RemoveLike(cc, "p", "u")
	mustErr("RemoveLike", err)
	_, err = r.LikedPostIDs(cc, "u")
	mustErr("LikedPostIDs", err)
	_, err = r.LikersByPost(cc, "p")
	mustErr("LikersByPost", err)
	_, err = r.LikedPostsByUser(cc, "u", 1, 0)
	mustErr("LikedPostsByUser", err)
	mustErr("DeleteLikesByPost", r.DeleteLikesByPost(cc, "p"))

	// --- Reposts -------------------------------------------------------------
	_, _, err = r.AddRepost(cc, "p", "u")
	mustErr("AddRepost", err)
	_, err = r.GetRepost(cc, "p", "u")
	mustErr("GetRepost", err)
	_, err = r.RemoveRepost(cc, "p", "u")
	mustErr("RemoveRepost", err)
	_, err = r.RepostedPostIDs(cc, "u")
	mustErr("RepostedPostIDs", err)
	_, err = r.ListRepostsByUser(cc, "u", 1, 0)
	mustErr("ListRepostsByUser", err)
	mustErr("DeleteRepostsByPost", r.DeleteRepostsByPost(cc, "p"))

	// --- Commentaires --------------------------------------------------------
	mustErr("AddComment", r.AddComment(cc, &models.Comment{PostID: "p", AuthorID: "u", Content: "x", CreatedAt: now, UpdatedAt: now}))
	_, err = r.CommentStatsByIDs(cc, []bson.ObjectID{oid})
	mustErr("CommentStatsByIDs", err)
	_, err = r.ListComments(cc, "p", 1, 0)
	mustErr("ListComments", err)
	_, err = r.ListReplies(cc, "parent", 1, 0)
	mustErr("ListReplies", err)
	mustErr("IncReplyCount", r.IncReplyCount(cc, oid, 1))
	_, err = r.IncCommentCounter(cc, oid, "likes_count", 1)
	mustErr("IncCommentCounter", err)
	_, err = r.DeleteRepliesByParent(cc, "parent")
	mustErr("DeleteRepliesByParent", err)
	_, err = r.CommentIDsByParent(cc, "parent")
	mustErr("CommentIDsByParent", err)
	_, err = r.ListCommentsByAuthor(cc, "a", 1, 0)
	mustErr("ListCommentsByAuthor", err)
	_, err = r.GetComment(cc, oid)
	mustErr("GetComment", err)
	mustErr("DeleteComment", r.DeleteComment(cc, oid))
	_, err = r.AddCommentLike(cc, "c", "u")
	mustErr("AddCommentLike", err)
	_, err = r.RemoveCommentLike(cc, "c", "u")
	mustErr("RemoveCommentLike", err)
	_, err = r.LikedCommentIDsByUser(cc, "u", []string{"c"})
	mustErr("LikedCommentIDsByUser", err)
	mustErr("DeleteCommentLikesByComment", r.DeleteCommentLikesByComment(cc, "c"))
	mustErr("DeleteCommentLikesByComments", r.DeleteCommentLikesByComments(cc, []string{"c"}))
	mustErr("DeleteCommentsByPost", r.DeleteCommentsByPost(cc, "p"))

	// --- Signets -------------------------------------------------------------
	mustErr("CreateCollection", r.CreateCollection(cc, &models.BookmarkCollection{UserID: "u", Name: "n", CreatedAt: now, UpdatedAt: now}))
	_, err = r.GetCollection(cc, oid)
	mustErr("GetCollection", err)
	_, err = r.EnsureDefaultCollection(cc, "u")
	mustErr("EnsureDefaultCollection", err)
	_, err = r.ListCollections(cc, "u")
	mustErr("ListCollections", err)
	_, err = r.RenameCollection(cc, oid, "n")
	mustErr("RenameCollection", err)
	mustErr("DeleteCollection", r.DeleteCollection(cc, oid))
	_, err = r.CountBookmarks(cc, "u", "c")
	mustErr("CountBookmarks", err)
	_, err = r.AddBookmark(cc, "u", "p", "c")
	mustErr("AddBookmark", err)
	_, err = r.RemoveBookmark(cc, "u", "p", "c")
	mustErr("RemoveBookmark", err)
	_, err = r.RemoveAllBookmarksForPost(cc, "u", "p")
	mustErr("RemoveAllBookmarksForPost", err)
	_, err = r.BookmarkedPostIDs(cc, "u")
	mustErr("BookmarkedPostIDs", err)
	_, err = r.PostBookmarkCollectionIDs(cc, "u", "p")
	mustErr("PostBookmarkCollectionIDs", err)
	_, err = r.BookmarksByCollection(cc, "u", "c", 1, 0)
	mustErr("BookmarksByCollection", err)
	_, err = r.AllBookmarkedPostIDs(cc, "u", 1, 0)
	mustErr("AllBookmarkedPostIDs", err)
	mustErr("DeleteBookmarksByPost", r.DeleteBookmarksByPost(cc, "p"))
	mustErr("DeleteBookmarksByCollection", r.DeleteBookmarksByCollection(cc, "c"))
	_, err = r.GetPrefs(cc, "u")
	mustErr("GetPrefs", err)
	mustErr("UpsertPrefs", r.UpsertPrefs(cc, "u", "c", now))
}
