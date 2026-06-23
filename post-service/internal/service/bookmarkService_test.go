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

func collOf(userID string, isDefault bool) *models.BookmarkCollection {
	return &models.BookmarkCollection{ID: newOID(), UserID: userID, IsDefault: isDefault, Name: "c"}
}

func TestCreateBookmarkCollection(t *testing.T) {
	repo := &fakeRepo{fnCreateCollection: func(context.Context, *models.BookmarkCollection) error { return nil }}
	s := NewPostService(repo)
	coll, err := s.CreateBookmarkCollection(bg(), "u1", "Voyages")
	if err != nil || coll.Name != "Voyages" || coll.UserID != "u1" {
		t.Fatalf("CreateBookmarkCollection: %v / %+v", err, coll)
	}
}

func TestListBookmarkCollections(t *testing.T) {
	repo := &fakeRepo{
		fnEnsureDefaultCollection: func(context.Context, string) (*models.BookmarkCollection, error) { return collOf("u1", true), nil },
		fnListCollections: func(context.Context, string) ([]models.BookmarkCollection, error) {
			return []models.BookmarkCollection{*collOf("u1", true), *collOf("u1", false)}, nil
		},
		fnCountBookmarks: func(context.Context, string, string) (int64, error) { return 3, nil },
	}
	s := NewPostService(repo)
	colls, err := s.ListBookmarkCollections(bg(), "u1")
	if err != nil || len(colls) != 2 || colls[0].ItemsCount != 3 {
		t.Fatalf("ListBookmarkCollections: %v / %+v", err, colls)
	}
}

func TestRenameBookmarkCollection_Success(t *testing.T) {
	coll := collOf("u1", false)
	repo := &fakeRepo{
		fnGetCollection:    func(context.Context, bson.ObjectID) (*models.BookmarkCollection, error) { return coll, nil },
		fnRenameCollection: func(context.Context, bson.ObjectID, string) (*models.BookmarkCollection, error) { return coll, nil },
		fnCountBookmarks:   func(context.Context, string, string) (int64, error) { return 1, nil },
	}
	s := NewPostService(repo)
	if _, err := s.RenameBookmarkCollection(bg(), coll.ID.Hex(), "u1", "Nouveau"); err != nil {
		t.Fatalf("RenameBookmarkCollection: %v", err)
	}
}

func TestRenameBookmarkCollection_DefaultForbidden(t *testing.T) {
	coll := collOf("u1", true)
	repo := &fakeRepo{fnGetCollection: func(context.Context, bson.ObjectID) (*models.BookmarkCollection, error) { return coll, nil }}
	s := NewPostService(repo)
	if _, err := s.RenameBookmarkCollection(bg(), coll.ID.Hex(), "u1", "x"); !errors.Is(err, ErrDefaultCollection) {
		t.Fatalf("la collection par défaut n'est pas renommable, got %v", err)
	}
}

func TestRenameBookmarkCollection_NotOwned(t *testing.T) {
	coll := collOf("autre", false)
	repo := &fakeRepo{fnGetCollection: func(context.Context, bson.ObjectID) (*models.BookmarkCollection, error) { return coll, nil }}
	s := NewPostService(repo)
	if _, err := s.RenameBookmarkCollection(bg(), coll.ID.Hex(), "u1", "x"); !errors.Is(err, ErrCollectionNotFound) {
		t.Fatalf("collection d'autrui → ErrCollectionNotFound, got %v", err)
	}
}

func TestDeleteBookmarkCollection_Success(t *testing.T) {
	coll := collOf("u1", false)
	repo := &fakeRepo{
		fnGetCollection:               func(context.Context, bson.ObjectID) (*models.BookmarkCollection, error) { return coll, nil },
		fnDeleteCollection:            func(context.Context, bson.ObjectID) error { return nil },
		fnDeleteBookmarksByCollection: func(context.Context, string) error { return nil },
	}
	s := NewPostService(repo)
	if err := s.DeleteBookmarkCollection(bg(), coll.ID.Hex(), "u1"); err != nil {
		t.Fatalf("DeleteBookmarkCollection: %v", err)
	}
}

func TestDeleteBookmarkCollection_DefaultForbidden(t *testing.T) {
	coll := collOf("u1", true)
	repo := &fakeRepo{fnGetCollection: func(context.Context, bson.ObjectID) (*models.BookmarkCollection, error) { return coll, nil }}
	s := NewPostService(repo)
	if err := s.DeleteBookmarkCollection(bg(), coll.ID.Hex(), "u1"); !errors.Is(err, ErrDefaultCollection) {
		t.Fatalf("la collection par défaut n'est pas supprimable, got %v", err)
	}
}

func TestBookmark_ExplicitCollection(t *testing.T) {
	post := postWith("a1")
	coll := collOf("u1", false)
	repo := &fakeRepo{
		fnGet:            func(context.Context, bson.ObjectID) (*models.Post, error) { return post, nil },
		fnGetCollection:  func(context.Context, bson.ObjectID) (*models.BookmarkCollection, error) { return coll, nil },
		fnAddBookmark:    func(context.Context, string, string, string) (bool, error) { return true, nil },
		fnUpsertPrefs:    func(context.Context, string, string, time.Time) error { return nil },
		fnCountBookmarks: func(context.Context, string, string) (int64, error) { return 1, nil },
	}
	s := NewPostService(repo)
	res, err := s.Bookmark(bg(), "u1", post.ID.Hex(), coll.ID.Hex())
	if err != nil || res.Status != BookmarkStatusFiled {
		t.Fatalf("Bookmark explicite → filed, got %v / %+v", err, res)
	}
}

func TestBookmark_ShortClick_NeedsChoice(t *testing.T) {
	post := postWith("a1")
	repo := &fakeRepo{
		fnGet:                     func(context.Context, bson.ObjectID) (*models.Post, error) { return post, nil },
		fnGetPrefs:                func(context.Context, string) (*models.BookmarkPrefs, error) { return &models.BookmarkPrefs{}, nil },
		fnEnsureDefaultCollection: func(context.Context, string) (*models.BookmarkCollection, error) { return collOf("u1", true), nil },
		fnListCollections: func(context.Context, string) ([]models.BookmarkCollection, error) {
			return []models.BookmarkCollection{*collOf("u1", true)}, nil
		},
	}
	s := NewPostService(repo, WithBookmarkWindow(5*time.Minute))
	res, err := s.Bookmark(bg(), "u1", post.ID.Hex(), "")
	if err != nil || res.Status != BookmarkStatusNeedsChoice {
		t.Fatalf("clic court hors fenêtre → needs_choice, got %v / %+v", err, res)
	}
}

func TestBookmark_ShortClick_WithinWindow(t *testing.T) {
	post := postWith("a1")
	coll := collOf("u1", false)
	now := time.Now()
	repo := &fakeRepo{
		fnGet: func(context.Context, bson.ObjectID) (*models.Post, error) { return post, nil },
		fnGetPrefs: func(context.Context, string) (*models.BookmarkPrefs, error) {
			return &models.BookmarkPrefs{LastCollectionID: coll.ID.Hex(), LastBookmarkAt: &now}, nil
		},
		fnGetCollection:  func(context.Context, bson.ObjectID) (*models.BookmarkCollection, error) { return coll, nil },
		fnAddBookmark:    func(context.Context, string, string, string) (bool, error) { return true, nil },
		fnUpsertPrefs:    func(context.Context, string, string, time.Time) error { return nil },
		fnCountBookmarks: func(context.Context, string, string) (int64, error) { return 2, nil },
	}
	s := NewPostService(repo, WithBookmarkWindow(5*time.Minute))
	res, err := s.Bookmark(bg(), "u1", post.ID.Hex(), "")
	if err != nil || res.Status != BookmarkStatusFiled {
		t.Fatalf("clic court dans la fenêtre → filed, got %v / %+v", err, res)
	}
}

func TestBookmark_PostNotFound(t *testing.T) {
	repo := &fakeRepo{fnGet: func(context.Context, bson.ObjectID) (*models.Post, error) { return nil, mongo.ErrNoDocuments }}
	s := NewPostService(repo)
	if _, err := s.Bookmark(bg(), "u1", newOID().Hex(), ""); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("attendu ErrPostNotFound, got %v", err)
	}
}

func TestUnbookmark_AllCollections(t *testing.T) {
	repo := &fakeRepo{fnRemoveAllBookmarksForPost: func(context.Context, string, string) (int64, error) { return 2, nil }}
	s := NewPostService(repo)
	if err := s.Unbookmark(bg(), "u1", newOID().Hex(), ""); err != nil {
		t.Fatalf("Unbookmark (toutes): %v", err)
	}
}

func TestUnbookmark_SpecificCollection(t *testing.T) {
	coll := collOf("u1", false)
	repo := &fakeRepo{
		fnGetCollection:  func(context.Context, bson.ObjectID) (*models.BookmarkCollection, error) { return coll, nil },
		fnRemoveBookmark: func(context.Context, string, string, string) (bool, error) { return true, nil },
	}
	s := NewPostService(repo)
	if err := s.Unbookmark(bg(), "u1", newOID().Hex(), coll.ID.Hex()); err != nil {
		t.Fatalf("Unbookmark (collection): %v", err)
	}
}

func TestBookmarkedPostIDs(t *testing.T) {
	repo := &fakeRepo{fnBookmarkedPostIDs: func(context.Context, string) ([]string, error) { return []string{"a"}, nil }}
	s := NewPostService(repo)
	if ids, err := s.BookmarkedPostIDs(bg(), "u1"); err != nil || len(ids) != 1 {
		t.Fatalf("BookmarkedPostIDs: %v / %v", ids, err)
	}
}

func TestPostBookmarkCollectionIDs(t *testing.T) {
	repo := &fakeRepo{fnPostBookmarkCollectionIDs: func(context.Context, string, string) ([]string, error) { return []string{"c1"}, nil }}
	s := NewPostService(repo)
	if _, err := s.PostBookmarkCollectionIDs(bg(), "u1", newOID().Hex()); err != nil {
		t.Fatalf("PostBookmarkCollectionIDs: %v", err)
	}
	if _, err := s.PostBookmarkCollectionIDs(bg(), "u1", "bad"); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("attendu ErrInvalidID, got %v", err)
	}
}

func TestListAllBookmarkedPosts(t *testing.T) {
	post := postWith("a1")
	repo := &fakeRepo{
		fnAllBookmarkedPostIDs: func(context.Context, string, int64, int64) ([]string, error) {
			return []string{post.ID.Hex(), "bad"}, nil
		},
		fnGet: func(context.Context, bson.ObjectID) (*models.Post, error) { return post, nil },
	}
	s := NewPostService(repo)
	posts, err := s.ListAllBookmarkedPosts(bg(), "u1", 0, 0)
	if err != nil || len(posts) != 1 {
		t.Fatalf("ListAllBookmarkedPosts: %v / %d (ids invalides ignorés)", err, len(posts))
	}
}

func TestListBookmarksInCollection(t *testing.T) {
	post := postWith("a1")
	coll := collOf("u1", false)
	repo := &fakeRepo{
		fnGetCollection: func(context.Context, bson.ObjectID) (*models.BookmarkCollection, error) { return coll, nil },
		fnBookmarksByCollection: func(context.Context, string, string, int64, int64) ([]string, error) {
			return []string{post.ID.Hex()}, nil
		},
		fnGet: func(context.Context, bson.ObjectID) (*models.Post, error) { return post, nil },
	}
	s := NewPostService(repo)
	posts, err := s.ListBookmarksInCollection(bg(), coll.ID.Hex(), "u1", 0, 0)
	if err != nil || len(posts) != 1 {
		t.Fatalf("ListBookmarksInCollection: %v / %d", err, len(posts))
	}
}

func TestGetOwnedCollection_InvalidID(t *testing.T) {
	s := NewPostService(&fakeRepo{})
	if _, err := s.getOwnedCollection(bg(), "not-hex", "u1"); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("attendu ErrInvalidID, got %v", err)
	}
}

func TestGetOwnedCollection_NotFound(t *testing.T) {
	repo := &fakeRepo{fnGetCollection: func(context.Context, bson.ObjectID) (*models.BookmarkCollection, error) {
		return nil, mongo.ErrNoDocuments
	}}
	s := NewPostService(repo)
	if _, err := s.getOwnedCollection(bg(), newOID().Hex(), "u1"); !errors.Is(err, ErrCollectionNotFound) {
		t.Fatalf("attendu ErrCollectionNotFound, got %v", err)
	}
}
