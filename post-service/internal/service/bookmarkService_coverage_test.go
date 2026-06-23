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

// Branches d'erreur des opérations signets (dépendances repo en échec).

func TestCreateBookmarkCollection_Error(t *testing.T) {
	repo := &fakeRepo{fnCreateCollection: func(context.Context, *models.BookmarkCollection) error { return errBoom }}
	s := NewPostService(repo)
	if _, err := s.CreateBookmarkCollection(bg(), "u1", "x"); !errors.Is(err, errBoom) {
		t.Fatalf("erreur CreateCollection doit remonter, got %v", err)
	}
}

func TestListBookmarkCollections_EnsureError(t *testing.T) {
	repo := &fakeRepo{fnEnsureDefaultCollection: func(context.Context, string) (*models.BookmarkCollection, error) { return nil, errBoom }}
	s := NewPostService(repo)
	if _, err := s.ListBookmarkCollections(bg(), "u1"); !errors.Is(err, errBoom) {
		t.Fatalf("erreur EnsureDefaultCollection doit remonter, got %v", err)
	}
}

func TestListBookmarkCollections_ListError(t *testing.T) {
	repo := &fakeRepo{
		fnEnsureDefaultCollection: func(context.Context, string) (*models.BookmarkCollection, error) { return collOf("u1", true), nil },
		fnListCollections:         func(context.Context, string) ([]models.BookmarkCollection, error) { return nil, errBoom },
	}
	s := NewPostService(repo)
	if _, err := s.ListBookmarkCollections(bg(), "u1"); !errors.Is(err, errBoom) {
		t.Fatalf("erreur ListCollections doit remonter, got %v", err)
	}
}

func TestListBookmarkCollections_CountError(t *testing.T) {
	repo := &fakeRepo{
		fnEnsureDefaultCollection: func(context.Context, string) (*models.BookmarkCollection, error) { return collOf("u1", true), nil },
		fnListCollections: func(context.Context, string) ([]models.BookmarkCollection, error) {
			return []models.BookmarkCollection{*collOf("u1", true)}, nil
		},
		fnCountBookmarks: func(context.Context, string, string) (int64, error) { return 0, errBoom },
	}
	s := NewPostService(repo)
	if _, err := s.ListBookmarkCollections(bg(), "u1"); !errors.Is(err, errBoom) {
		t.Fatalf("erreur CountBookmarks doit remonter, got %v", err)
	}
}

func TestRenameBookmarkCollection_RenameError(t *testing.T) {
	coll := collOf("u1", false)
	repo := &fakeRepo{
		fnGetCollection: func(context.Context, bson.ObjectID) (*models.BookmarkCollection, error) { return coll, nil },
		fnRenameCollection: func(context.Context, bson.ObjectID, string) (*models.BookmarkCollection, error) {
			return nil, mongo.ErrNoDocuments
		},
	}
	s := NewPostService(repo)
	if _, err := s.RenameBookmarkCollection(bg(), coll.ID.Hex(), "u1", "x"); !errors.Is(err, ErrCollectionNotFound) {
		t.Fatalf("erreur RenameCollection → ErrCollectionNotFound, got %v", err)
	}
}

func TestRenameBookmarkCollection_CountError(t *testing.T) {
	coll := collOf("u1", false)
	repo := &fakeRepo{
		fnGetCollection:    func(context.Context, bson.ObjectID) (*models.BookmarkCollection, error) { return coll, nil },
		fnRenameCollection: func(context.Context, bson.ObjectID, string) (*models.BookmarkCollection, error) { return coll, nil },
		fnCountBookmarks:   func(context.Context, string, string) (int64, error) { return 0, errBoom },
	}
	s := NewPostService(repo)
	if _, err := s.RenameBookmarkCollection(bg(), coll.ID.Hex(), "u1", "x"); !errors.Is(err, errBoom) {
		t.Fatalf("erreur CountBookmarks doit remonter, got %v", err)
	}
}

func TestDeleteBookmarkCollection_NotOwned(t *testing.T) {
	coll := collOf("autre", false)
	repo := &fakeRepo{fnGetCollection: func(context.Context, bson.ObjectID) (*models.BookmarkCollection, error) { return coll, nil }}
	s := NewPostService(repo)
	if err := s.DeleteBookmarkCollection(bg(), coll.ID.Hex(), "u1"); !errors.Is(err, ErrCollectionNotFound) {
		t.Fatalf("collection d'autrui → ErrCollectionNotFound, got %v", err)
	}
}

func TestDeleteBookmarkCollection_DeleteError(t *testing.T) {
	coll := collOf("u1", false)
	repo := &fakeRepo{
		fnGetCollection:    func(context.Context, bson.ObjectID) (*models.BookmarkCollection, error) { return coll, nil },
		fnDeleteCollection: func(context.Context, bson.ObjectID) error { return mongo.ErrNoDocuments },
	}
	s := NewPostService(repo)
	if err := s.DeleteBookmarkCollection(bg(), coll.ID.Hex(), "u1"); !errors.Is(err, ErrCollectionNotFound) {
		t.Fatalf("erreur DeleteCollection → ErrCollectionNotFound, got %v", err)
	}
}

func TestBookmark_FileBookmarkError(t *testing.T) {
	post := postWith("a1")
	coll := collOf("u1", false)
	repo := &fakeRepo{
		fnGet:           getOK(post),
		fnGetCollection: func(context.Context, bson.ObjectID) (*models.BookmarkCollection, error) { return coll, nil },
		fnAddBookmark:   func(context.Context, string, string, string) (bool, error) { return false, errBoom },
	}
	s := NewPostService(repo)
	if _, err := s.Bookmark(bg(), "u1", post.ID.Hex(), coll.ID.Hex()); !errors.Is(err, errBoom) {
		t.Fatalf("erreur fileBookmark doit remonter, got %v", err)
	}
}

func TestBookmark_GetPrefsError(t *testing.T) {
	post := postWith("a1")
	repo := &fakeRepo{
		fnGet:      getOK(post),
		fnGetPrefs: func(context.Context, string) (*models.BookmarkPrefs, error) { return nil, errBoom },
	}
	s := NewPostService(repo)
	if _, err := s.Bookmark(bg(), "u1", post.ID.Hex(), ""); !errors.Is(err, errBoom) {
		t.Fatalf("erreur GetPrefs doit remonter, got %v", err)
	}
}

// Clic court dans la fenêtre mais la dernière collection a disparu → on retombe
// sur le choix (needs_choice) au lieu de classer automatiquement.
func TestBookmark_WindowCollectionGone(t *testing.T) {
	post := postWith("a1")
	now := time.Now()
	repo := &fakeRepo{
		fnGet: getOK(post),
		fnGetPrefs: func(context.Context, string) (*models.BookmarkPrefs, error) {
			return &models.BookmarkPrefs{LastCollectionID: newOID().Hex(), LastBookmarkAt: &now}, nil
		},
		fnGetCollection: func(context.Context, bson.ObjectID) (*models.BookmarkCollection, error) {
			return nil, mongo.ErrNoDocuments
		},
		fnEnsureDefaultCollection: func(context.Context, string) (*models.BookmarkCollection, error) { return collOf("u1", true), nil },
		fnListCollections: func(context.Context, string) ([]models.BookmarkCollection, error) {
			return []models.BookmarkCollection{*collOf("u1", true)}, nil
		},
	}
	s := NewPostService(repo, WithBookmarkWindow(5*time.Minute))
	res, err := s.Bookmark(bg(), "u1", post.ID.Hex(), "")
	if err != nil || res.Status != BookmarkStatusNeedsChoice {
		t.Fatalf("collection disparue → needs_choice, got %v / %+v", err, res)
	}
}

// Clic court dans la fenêtre : la collection existe mais le classement échoue.
func TestBookmark_WindowFileError(t *testing.T) {
	post := postWith("a1")
	coll := collOf("u1", false)
	now := time.Now()
	repo := &fakeRepo{
		fnGet: getOK(post),
		fnGetPrefs: func(context.Context, string) (*models.BookmarkPrefs, error) {
			return &models.BookmarkPrefs{LastCollectionID: coll.ID.Hex(), LastBookmarkAt: &now}, nil
		},
		fnGetCollection: func(context.Context, bson.ObjectID) (*models.BookmarkCollection, error) { return coll, nil },
		fnAddBookmark:   func(context.Context, string, string, string) (bool, error) { return false, errBoom },
	}
	s := NewPostService(repo, WithBookmarkWindow(5*time.Minute))
	if _, err := s.Bookmark(bg(), "u1", post.ID.Hex(), ""); !errors.Is(err, errBoom) {
		t.Fatalf("erreur de classement dans la fenêtre doit remonter, got %v", err)
	}
}

func TestBookmark_ListCollectionsError(t *testing.T) {
	post := postWith("a1")
	repo := &fakeRepo{
		fnGet:                     getOK(post),
		fnGetPrefs:                func(context.Context, string) (*models.BookmarkPrefs, error) { return &models.BookmarkPrefs{}, nil },
		fnEnsureDefaultCollection: func(context.Context, string) (*models.BookmarkCollection, error) { return nil, errBoom },
	}
	s := NewPostService(repo)
	if _, err := s.Bookmark(bg(), "u1", post.ID.Hex(), ""); !errors.Is(err, errBoom) {
		t.Fatalf("erreur ListBookmarkCollections doit remonter, got %v", err)
	}
}

func TestFileBookmark_UpsertError(t *testing.T) {
	coll := collOf("u1", false)
	repo := &fakeRepo{
		fnGetCollection: func(context.Context, bson.ObjectID) (*models.BookmarkCollection, error) { return coll, nil },
		fnAddBookmark:   func(context.Context, string, string, string) (bool, error) { return true, nil },
		fnUpsertPrefs:   func(context.Context, string, string, time.Time) error { return errBoom },
	}
	s := NewPostService(repo)
	if _, err := s.fileBookmark(bg(), "u1", newOID().Hex(), coll.ID.Hex()); !errors.Is(err, errBoom) {
		t.Fatalf("erreur UpsertPrefs doit remonter, got %v", err)
	}
}

func TestFileBookmark_CountError(t *testing.T) {
	coll := collOf("u1", false)
	repo := &fakeRepo{
		fnGetCollection:  func(context.Context, bson.ObjectID) (*models.BookmarkCollection, error) { return coll, nil },
		fnAddBookmark:    func(context.Context, string, string, string) (bool, error) { return true, nil },
		fnUpsertPrefs:    func(context.Context, string, string, time.Time) error { return nil },
		fnCountBookmarks: func(context.Context, string, string) (int64, error) { return 0, errBoom },
	}
	s := NewPostService(repo)
	if _, err := s.fileBookmark(bg(), "u1", newOID().Hex(), coll.ID.Hex()); !errors.Is(err, errBoom) {
		t.Fatalf("erreur CountBookmarks doit remonter, got %v", err)
	}
}

func TestUnbookmark_NotOwned(t *testing.T) {
	coll := collOf("autre", false)
	repo := &fakeRepo{fnGetCollection: func(context.Context, bson.ObjectID) (*models.BookmarkCollection, error) { return coll, nil }}
	s := NewPostService(repo)
	if err := s.Unbookmark(bg(), "u1", newOID().Hex(), coll.ID.Hex()); !errors.Is(err, ErrCollectionNotFound) {
		t.Fatalf("collection d'autrui → ErrCollectionNotFound, got %v", err)
	}
}

func TestListAllBookmarkedPosts_Error(t *testing.T) {
	repo := &fakeRepo{fnAllBookmarkedPostIDs: func(context.Context, string, int64, int64) ([]string, error) { return nil, errBoom }}
	s := NewPostService(repo)
	if _, err := s.ListAllBookmarkedPosts(bg(), "u1", 0, 0); !errors.Is(err, errBoom) {
		t.Fatalf("erreur AllBookmarkedPostIDs doit remonter, got %v", err)
	}
}

func TestListBookmarksInCollection_NotOwned(t *testing.T) {
	coll := collOf("autre", false)
	repo := &fakeRepo{fnGetCollection: func(context.Context, bson.ObjectID) (*models.BookmarkCollection, error) { return coll, nil }}
	s := NewPostService(repo)
	if _, err := s.ListBookmarksInCollection(bg(), coll.ID.Hex(), "u1", 0, 0); !errors.Is(err, ErrCollectionNotFound) {
		t.Fatalf("collection d'autrui → ErrCollectionNotFound, got %v", err)
	}
}

func TestListBookmarksInCollection_BookmarksError(t *testing.T) {
	coll := collOf("u1", false)
	repo := &fakeRepo{
		fnGetCollection:         func(context.Context, bson.ObjectID) (*models.BookmarkCollection, error) { return coll, nil },
		fnBookmarksByCollection: func(context.Context, string, string, int64, int64) ([]string, error) { return nil, errBoom },
	}
	s := NewPostService(repo)
	if _, err := s.ListBookmarksInCollection(bg(), coll.ID.Hex(), "u1", 0, 0); !errors.Is(err, errBoom) {
		t.Fatalf("erreur BookmarksByCollection doit remonter, got %v", err)
	}
}

// resolvePosts ignore les posts disparus (Get en erreur).
func TestResolvePosts_SkipsMissing(t *testing.T) {
	ok := postWith("a1")
	repo := &fakeRepo{fnGet: func(_ context.Context, id bson.ObjectID) (*models.Post, error) {
		if id == ok.ID {
			return ok, nil
		}
		return nil, mongo.ErrNoDocuments
	}}
	s := NewPostService(repo)
	posts := s.resolvePosts(bg(), []string{ok.ID.Hex(), newOID().Hex(), "bad"})
	if len(posts) != 1 || posts[0].AuthorID != "a1" {
		t.Fatalf("seuls les posts existants doivent être résolus, got %+v", posts)
	}
}

func TestRequirePost_InvalidID(t *testing.T) {
	s := NewPostService(&fakeRepo{})
	if _, err := s.requirePost(bg(), "bad"); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("attendu ErrInvalidID, got %v", err)
	}
}

// translateCollectionNotFound laisse passer une erreur non-mongo telle quelle.
func TestTranslateCollectionNotFound_Passthrough(t *testing.T) {
	if got := translateCollectionNotFound(errBoom); !errors.Is(got, errBoom) {
		t.Fatalf("erreur arbitraire doit passer telle quelle, got %v", got)
	}
	if got := translateCollectionNotFound(nil); got != nil {
		t.Fatalf("nil doit rester nil, got %v", got)
	}
}
