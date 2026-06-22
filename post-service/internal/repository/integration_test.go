package repository

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/webdad/post-service/internal/database"
	"github.com/webdad/post-service/internal/models"
)

func mongoURI() string {
	if u := os.Getenv("MONGO_TEST_URI"); u != "" {
		return u
	}
	return "mongodb://localhost:27017"
}

// newRepo ouvre une base de test éphémère, applique le schéma et renvoie un
// repository câblé. Skip si Mongo est indisponible ou refuse les opérations
// (auth du stack de dev local) — la CI fournit un Mongo dédié sans auth.
func newRepo(t *testing.T) (*PostRepository, context.Context) {
	t.Helper()
	ctx := context.Background()
	client, err := mongo.Connect(options.Client().ApplyURI(mongoURI()))
	if err != nil {
		t.Skipf("Mongo indisponible (%v)", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		t.Skipf("Mongo injoignable (%v)", err)
	}
	db := client.Database(fmt.Sprintf("webdad_post_test_%d", time.Now().UnixNano()))
	if err := db.Collection("__probe").Drop(ctx); err != nil {
		_ = client.Disconnect(ctx)
		t.Skipf("Mongo refuse les opérations (%v)", err)
	}
	if err := database.EnsureSchema(ctx, db); err != nil {
		_ = client.Disconnect(ctx)
		t.Skipf("EnsureSchema impossible (%v)", err)
	}
	t.Cleanup(func() {
		_ = db.Drop(ctx)
		_ = client.Disconnect(ctx)
	})
	return NewPostRepository(db), ctx
}

func mkPost(t *testing.T, r *PostRepository, ctx context.Context, author, content string, hashtags ...string) *models.Post {
	t.Helper()
	now := time.Now()
	p := &models.Post{
		AuthorID:      author,
		Content:       content,
		Hashtags:      hashtags,
		ReplyAudience: models.ReplyAudienceEveryone,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := r.Create(ctx, p); err != nil {
		t.Fatalf("Create post: %v", err)
	}
	return p
}

func TestRepo_PostCRUDAndLists(t *testing.T) {
	r, ctx := newRepo(t)
	p1 := mkPost(t, r, ctx, "a1", "hello #go", "go")
	p2 := mkPost(t, r, ctx, "a2", "deux #go #test", "go", "test")

	if got, err := r.Get(ctx, p1.ID); err != nil || got.Content != "hello #go" {
		t.Fatalf("Get: %v / %+v", err, got)
	}
	if _, err := r.Get(ctx, bson.NewObjectID()); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("Get inexistant doit renvoyer ErrNoDocuments, got %v", err)
	}
	if up, err := r.Update(ctx, p1.ID, "édité", []string{}); err != nil || up.Content != "édité" {
		t.Fatalf("Update: %v / %+v", err, up)
	}

	if all, err := r.GetAll(ctx, 10, 0); err != nil || len(all) != 2 {
		t.Fatalf("GetAll: %v / %d", err, len(all))
	}
	if byTag, err := r.GetAllByHashtag(ctx, "go", "recent", 10, 0); err != nil || len(byTag) != 1 {
		// p1 a perdu son hashtag à l'update → seul p2 reste taggé "go".
		t.Fatalf("GetAllByHashtag: %v / %d", err, len(byTag))
	}
	if withTags, err := r.GetAllWithHashtags(ctx, 10, 0); err != nil || len(withTags) != 1 {
		t.Fatalf("GetAllWithHashtags: %v / %d", err, len(withTags))
	}
	if byProfile, err := r.GetByProfile(ctx, "a2", 10, 0); err != nil || len(byProfile) != 1 {
		t.Fatalf("GetByProfile: %v / %d", err, len(byProfile))
	}
	if byProfileTag, err := r.GetByProfileHashtag(ctx, "a2", "test", 10, 0); err != nil || len(byProfileTag) != 1 {
		t.Fatalf("GetByProfileHashtag: %v / %d", err, len(byProfileTag))
	}
	if byAuthors, err := r.GetByAuthors(ctx, []string{"a1", "a2"}, 10, 0); err != nil || len(byAuthors) != 2 {
		t.Fatalf("GetByAuthors: %v / %d", err, len(byAuthors))
	}
	if byAuthorsTag, err := r.GetByAuthorsHashtag(ctx, []string{"a1", "a2"}, "go", "top", 10, 0); err != nil || len(byAuthorsTag) != 1 {
		t.Fatalf("GetByAuthorsHashtag: %v / %d", err, len(byAuthorsTag))
	}
	if stats, err := r.StatsByIDs(ctx, []bson.ObjectID{p1.ID, p2.ID}); err != nil || len(stats) != 2 {
		t.Fatalf("StatsByIDs: %v / %d", err, len(stats))
	}
	if trends, err := r.ListTopHashtags(ctx, 10); err != nil || len(trends) == 0 {
		t.Fatalf("ListTopHashtags: %v / %d", err, len(trends))
	}

	if err := r.Delete(ctx, p1.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := r.Delete(ctx, p1.ID); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("Delete déjà supprimé → ErrNoDocuments, got %v", err)
	}
}

func TestRepo_Counters(t *testing.T) {
	r, ctx := newRepo(t)
	p := mkPost(t, r, ctx, "a1", "x")
	up, err := r.IncCounter(ctx, p.ID, "likes_count", 1)
	if err != nil || up.LikesCount != 1 {
		t.Fatalf("IncCounter: %v / %+v", err, up)
	}
}

func TestRepo_ModerationLifecycle(t *testing.T) {
	r, ctx := newRepo(t)
	p := mkPost(t, r, ctx, "a1", "litige")

	if _, err := r.Hide(ctx, p.ID, "mod", time.Now()); err != nil {
		t.Fatalf("Hide: %v", err)
	}
	hidden, err := r.ListHidden(ctx, HiddenFilter{}, 10, 0)
	if err != nil || len(hidden) != 1 {
		t.Fatalf("ListHidden: %v / %d", err, len(hidden))
	}
	if _, err := r.ListHidden(ctx, HiddenFilter{AuthorID: "a1"}, 10, 0); err != nil {
		t.Fatalf("ListHidden filtré: %v", err)
	}
	if _, err := r.RestoreHidden(ctx, p.ID); err != nil {
		t.Fatalf("RestoreHidden: %v", err)
	}
	if _, err := r.SetAutoHidden(ctx, p.ID, true); err != nil {
		t.Fatalf("SetAutoHidden: %v", err)
	}
	if _, err := r.SetNsfw(ctx, p.ID, true, "mod", time.Now()); err != nil {
		t.Fatalf("SetNsfw: %v", err)
	}
}

func TestRepo_Pins(t *testing.T) {
	r, ctx := newRepo(t)
	p := mkPost(t, r, ctx, "a1", "à épingler")
	if _, err := r.Pin(ctx, p.ID, time.Now()); err != nil {
		t.Fatalf("Pin: %v", err)
	}
	if _, err := r.Unpin(ctx, p.ID); err != nil {
		t.Fatalf("Unpin: %v", err)
	}
	if _, err := r.Pin(ctx, p.ID, time.Now()); err != nil {
		t.Fatalf("Pin2: %v", err)
	}
	if err := r.UnpinByAuthor(ctx, "a1"); err != nil {
		t.Fatalf("UnpinByAuthor: %v", err)
	}
}

func TestRepo_Likes(t *testing.T) {
	r, ctx := newRepo(t)
	p := mkPost(t, r, ctx, "a1", "likeable")
	pid := p.ID.Hex()

	if created, err := r.AddLike(ctx, pid, "u1"); err != nil || !created {
		t.Fatalf("AddLike: %v / %v", err, created)
	}
	if created, err := r.AddLike(ctx, pid, "u1"); err != nil || created {
		t.Fatalf("AddLike idempotent attendu created=false, got %v / %v", err, created)
	}
	if ids, err := r.LikedPostIDs(ctx, "u1"); err != nil || len(ids) != 1 {
		t.Fatalf("LikedPostIDs: %v / %v", err, ids)
	}
	if likers, err := r.LikersByPost(ctx, pid); err != nil || len(likers) != 1 {
		t.Fatalf("LikersByPost: %v / %v", err, likers)
	}
	if posts, err := r.LikedPostsByUser(ctx, "u1", 10, 0); err != nil || len(posts) != 1 {
		t.Fatalf("LikedPostsByUser: %v / %d", err, len(posts))
	}
	if removed, err := r.RemoveLike(ctx, pid, "u1"); err != nil || !removed {
		t.Fatalf("RemoveLike: %v / %v", err, removed)
	}
	if removed, err := r.RemoveLike(ctx, pid, "u1"); err != nil || removed {
		t.Fatalf("RemoveLike idempotent: %v / %v", err, removed)
	}
	_, _ = r.AddLike(ctx, pid, "u2")
	if err := r.DeleteLikesByPost(ctx, pid); err != nil {
		t.Fatalf("DeleteLikesByPost: %v", err)
	}
}

func TestRepo_Reposts(t *testing.T) {
	r, ctx := newRepo(t)
	p := mkPost(t, r, ctx, "a1", "repostable")
	pid := p.ID.Hex()

	if _, created, err := r.AddRepost(ctx, pid, "u1"); err != nil || !created {
		t.Fatalf("AddRepost: %v / %v", err, created)
	}
	if _, created, err := r.AddRepost(ctx, pid, "u1"); err != nil || created {
		t.Fatalf("AddRepost idempotent attendu created=false, got %v / %v", err, created)
	}
	if rep, err := r.GetRepost(ctx, pid, "u1"); err != nil || rep == nil {
		t.Fatalf("GetRepost: %v / %v", err, rep)
	}
	if ids, err := r.RepostedPostIDs(ctx, "u1"); err != nil || len(ids) != 1 {
		t.Fatalf("RepostedPostIDs: %v / %v", err, ids)
	}
	if list, err := r.ListRepostsByUser(ctx, "u1", 10, 0); err != nil || len(list) != 1 {
		t.Fatalf("ListRepostsByUser: %v / %d", err, len(list))
	}
	if removed, err := r.RemoveRepost(ctx, pid, "u1"); err != nil || !removed {
		t.Fatalf("RemoveRepost: %v / %v", err, removed)
	}
	_, _, _ = r.AddRepost(ctx, pid, "u2")
	if err := r.DeleteRepostsByPost(ctx, pid); err != nil {
		t.Fatalf("DeleteRepostsByPost: %v", err)
	}
}

func TestRepo_Polls(t *testing.T) {
	r, ctx := newRepo(t)
	now := time.Now()
	p := &models.Post{
		AuthorID:      "a1",
		Content:       "sondage",
		ReplyAudience: models.ReplyAudienceEveryone,
		Poll: &models.Poll{
			Choices:    []models.PollChoice{{ID: "c1", Label: "A", VotesCount: 0}, {ID: "c2", Label: "B", VotesCount: 0}},
			EndsAt:     now.Add(time.Hour),
			Audience:   models.PollAudienceEveryone,
			TotalVotes: 0,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := r.Create(ctx, p); err != nil {
		t.Fatalf("Create poll: %v", err)
	}
	pid := p.ID.Hex()
	if created, err := r.AddPollVote(ctx, pid, "u1", "c1"); err != nil || !created {
		t.Fatalf("AddPollVote: %v / %v", err, created)
	}
	if created, err := r.AddPollVote(ctx, pid, "u1", "c1"); err != nil || created {
		t.Fatalf("AddPollVote idempotent: %v / %v", err, created)
	}
	if choice, err := r.PollVoteChoice(ctx, pid, "u1"); err != nil || choice != "c1" {
		t.Fatalf("PollVoteChoice: %v / %q", err, choice)
	}
	if _, err := r.IncPollChoice(ctx, p.ID, "c1"); err != nil {
		t.Fatalf("IncPollChoice: %v", err)
	}
	if _, err := r.ClosePoll(ctx, p.ID, time.Now()); err != nil {
		t.Fatalf("ClosePoll: %v", err)
	}
	if err := r.DeletePollVotesByPost(ctx, pid); err != nil {
		t.Fatalf("DeletePollVotesByPost: %v", err)
	}
}

func mkComment(t *testing.T, r *PostRepository, ctx context.Context, postID, parentID, author string) *models.Comment {
	t.Helper()
	now := time.Now()
	c := &models.Comment{
		PostID:    postID,
		ParentID:  parentID,
		AuthorID:  author,
		Content:   "un commentaire",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := r.AddComment(ctx, c); err != nil {
		t.Fatalf("AddComment: %v", err)
	}
	return c
}

func TestRepo_Comments(t *testing.T) {
	r, ctx := newRepo(t)
	p := mkPost(t, r, ctx, "a1", "post commenté")
	pid := p.ID.Hex()

	root := mkComment(t, r, ctx, pid, "", "u1")
	reply := mkComment(t, r, ctx, pid, root.ID.Hex(), "u2")

	if got, err := r.GetComment(ctx, root.ID); err != nil || got.AuthorID != "u1" {
		t.Fatalf("GetComment: %v / %+v", err, got)
	}
	if list, err := r.ListComments(ctx, pid, 10, 0); err != nil || len(list) != 1 {
		t.Fatalf("ListComments (racines): %v / %d", err, len(list))
	}
	if replies, err := r.ListReplies(ctx, root.ID.Hex(), 10, 0); err != nil || len(replies) != 1 {
		t.Fatalf("ListReplies: %v / %d", err, len(replies))
	}
	if byAuthor, err := r.ListCommentsByAuthor(ctx, "u1", 10, 0); err != nil || len(byAuthor) != 1 {
		t.Fatalf("ListCommentsByAuthor: %v / %d", err, len(byAuthor))
	}
	if stats, err := r.CommentStatsByIDs(ctx, []bson.ObjectID{root.ID, reply.ID}); err != nil || len(stats) != 2 {
		t.Fatalf("CommentStatsByIDs: %v / %d", err, len(stats))
	}
	if err := r.IncReplyCount(ctx, root.ID, 1); err != nil {
		t.Fatalf("IncReplyCount: %v", err)
	}
	if _, err := r.IncCommentCounter(ctx, root.ID, "likes_count", 1); err != nil {
		t.Fatalf("IncCommentCounter: %v", err)
	}
	if ids, err := r.CommentIDsByParent(ctx, root.ID.Hex()); err != nil || len(ids) != 1 {
		t.Fatalf("CommentIDsByParent: %v / %v", err, ids)
	}
	if n, err := r.DeleteRepliesByParent(ctx, root.ID.Hex()); err != nil || n != 1 {
		t.Fatalf("DeleteRepliesByParent: %v / %d", err, n)
	}
	if err := r.DeleteComment(ctx, root.ID); err != nil {
		t.Fatalf("DeleteComment: %v", err)
	}
	mkComment(t, r, ctx, pid, "", "u3")
	if err := r.DeleteCommentsByPost(ctx, pid); err != nil {
		t.Fatalf("DeleteCommentsByPost: %v", err)
	}
}

func TestRepo_CommentLikes(t *testing.T) {
	r, ctx := newRepo(t)
	p := mkPost(t, r, ctx, "a1", "post")
	c := mkComment(t, r, ctx, p.ID.Hex(), "", "u1")
	cid := c.ID.Hex()

	if created, err := r.AddCommentLike(ctx, cid, "u2"); err != nil || !created {
		t.Fatalf("AddCommentLike: %v / %v", err, created)
	}
	if created, err := r.AddCommentLike(ctx, cid, "u2"); err != nil || created {
		t.Fatalf("AddCommentLike idempotent: %v / %v", err, created)
	}
	if liked, err := r.LikedCommentIDsByUser(ctx, "u2", []string{cid}); err != nil || len(liked) != 1 {
		t.Fatalf("LikedCommentIDsByUser: %v / %v", err, liked)
	}
	if removed, err := r.RemoveCommentLike(ctx, cid, "u2"); err != nil || !removed {
		t.Fatalf("RemoveCommentLike: %v / %v", err, removed)
	}
	_, _ = r.AddCommentLike(ctx, cid, "u3")
	if err := r.DeleteCommentLikesByComment(ctx, cid); err != nil {
		t.Fatalf("DeleteCommentLikesByComment: %v", err)
	}
	_, _ = r.AddCommentLike(ctx, cid, "u4")
	if err := r.DeleteCommentLikesByComments(ctx, []string{cid}); err != nil {
		t.Fatalf("DeleteCommentLikesByComments: %v", err)
	}
}

func TestRepo_Bookmarks(t *testing.T) {
	r, ctx := newRepo(t)
	p := mkPost(t, r, ctx, "a1", "à ranger")
	pid := p.ID.Hex()

	def, err := r.EnsureDefaultCollection(ctx, "u1")
	if err != nil {
		t.Fatalf("EnsureDefaultCollection: %v", err)
	}
	if def2, err := r.EnsureDefaultCollection(ctx, "u1"); err != nil || def2.ID != def.ID {
		t.Fatalf("EnsureDefaultCollection idempotent: %v / %+v", err, def2)
	}

	coll := &models.BookmarkCollection{UserID: "u1", Name: "Voyages", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := r.CreateCollection(ctx, coll); err != nil {
		t.Fatalf("CreateCollection: %v", err)
	}
	if got, err := r.GetCollection(ctx, coll.ID); err != nil || got.Name != "Voyages" {
		t.Fatalf("GetCollection: %v / %+v", err, got)
	}
	if colls, err := r.ListCollections(ctx, "u1"); err != nil || len(colls) != 2 {
		t.Fatalf("ListCollections: %v / %d", err, len(colls))
	}
	if _, err := r.RenameCollection(ctx, coll.ID, "Vacances"); err != nil {
		t.Fatalf("RenameCollection: %v", err)
	}

	cid := coll.ID.Hex()
	if added, err := r.AddBookmark(ctx, "u1", pid, cid); err != nil || !added {
		t.Fatalf("AddBookmark: %v / %v", err, added)
	}
	if added, err := r.AddBookmark(ctx, "u1", pid, cid); err != nil || added {
		t.Fatalf("AddBookmark idempotent: %v / %v", err, added)
	}
	if n, err := r.CountBookmarks(ctx, "u1", cid); err != nil || n != 1 {
		t.Fatalf("CountBookmarks: %v / %d", err, n)
	}
	if cids, err := r.PostBookmarkCollectionIDs(ctx, "u1", pid); err != nil || len(cids) != 1 {
		t.Fatalf("PostBookmarkCollectionIDs: %v / %v", err, cids)
	}
	if ids, err := r.BookmarkedPostIDs(ctx, "u1"); err != nil || len(ids) != 1 {
		t.Fatalf("BookmarkedPostIDs: %v / %v", err, ids)
	}
	if ids, err := r.BookmarksByCollection(ctx, "u1", cid, 10, 0); err != nil || len(ids) != 1 {
		t.Fatalf("BookmarksByCollection: %v / %v", err, ids)
	}
	if ids, err := r.AllBookmarkedPostIDs(ctx, "u1", 10, 0); err != nil || len(ids) != 1 {
		t.Fatalf("AllBookmarkedPostIDs: %v / %v", err, ids)
	}
	if removed, err := r.RemoveBookmark(ctx, "u1", pid, cid); err != nil || !removed {
		t.Fatalf("RemoveBookmark: %v / %v", err, removed)
	}
	_, _ = r.AddBookmark(ctx, "u1", pid, cid)
	if _, err := r.RemoveAllBookmarksForPost(ctx, "u1", pid); err != nil {
		t.Fatalf("RemoveAllBookmarksForPost: %v", err)
	}
	_, _ = r.AddBookmark(ctx, "u1", pid, cid)
	if err := r.DeleteBookmarksByPost(ctx, pid); err != nil {
		t.Fatalf("DeleteBookmarksByPost: %v", err)
	}
	_, _ = r.AddBookmark(ctx, "u1", pid, cid)
	if err := r.DeleteBookmarksByCollection(ctx, cid); err != nil {
		t.Fatalf("DeleteBookmarksByCollection: %v", err)
	}
	if err := r.DeleteCollection(ctx, coll.ID); err != nil {
		t.Fatalf("DeleteCollection: %v", err)
	}
}

func TestRepo_Prefs(t *testing.T) {
	r, ctx := newRepo(t)
	if prefs, err := r.GetPrefs(ctx, "u1"); err != nil || prefs != nil {
		t.Fatalf("GetPrefs absent → (nil, nil), got %+v / %v", prefs, err)
	}
	if err := r.UpsertPrefs(ctx, "u1", "c1", time.Now()); err != nil {
		t.Fatalf("UpsertPrefs: %v", err)
	}
	if prefs, err := r.GetPrefs(ctx, "u1"); err != nil || prefs.LastCollectionID != "c1" {
		t.Fatalf("GetPrefs: %v / %+v", err, prefs)
	}
}

func TestRepo_Purge(t *testing.T) {
	r, ctx := newRepo(t)
	p := mkPost(t, r, ctx, "a1", "à purger")
	// Masque le post dans le passé pour le rendre purgeable.
	past := time.Now().Add(-48 * time.Hour)
	if _, err := r.Hide(ctx, p.ID, "mod", past); err != nil {
		t.Fatalf("Hide: %v", err)
	}
	cutoff := time.Now().Add(-24 * time.Hour)
	if purgeable, err := r.ListPurgeable(ctx, cutoff, 10); err != nil || len(purgeable) != 1 {
		t.Fatalf("ListPurgeable: %v / %d", err, len(purgeable))
	}
	if warnable, err := r.ListPurgeWarnable(ctx, cutoff, 10); err != nil || len(warnable) != 1 {
		t.Fatalf("ListPurgeWarnable: %v / %d", err, len(warnable))
	}
	if err := r.MarkPurgeWarned(ctx, p.ID, time.Now()); err != nil {
		t.Fatalf("MarkPurgeWarned: %v", err)
	}
	// Une fois prévenu, le post sort de la liste des préavis.
	if warnable, err := r.ListPurgeWarnable(ctx, cutoff, 10); err != nil || len(warnable) != 0 {
		t.Fatalf("ListPurgeWarnable après préavis: %v / %d", err, len(warnable))
	}
	if n, err := r.PurgeByAuthor(ctx, "a1"); err != nil || n != 1 {
		t.Fatalf("PurgeByAuthor: %v / %d", err, n)
	}
}
