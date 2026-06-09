package service

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/webdad/post-service/internal/models"
)

// BookmarkResult décrit l'issue d'un clic court (ou explicite) sur le bouton
// signet, renvoyé au front.
//   - Status "filed"        : le post a été rangé dans Collection.
//   - Status "needs_choice" : ouverture de rafale → le front ouvre le sélecteur
//     (Collections = collections existantes proposées) ; rien n'a été rangé.
type BookmarkResult struct {
	Status      string                      `json:"status"`
	Collection  *models.BookmarkCollection  `json:"collection,omitempty"`
	Collections []models.BookmarkCollection `json:"collections,omitempty"`
}

// --- Collections ---------------------------------------------------------------

// CreateBookmarkCollection crée une collection pour l'utilisateur.
func (s *PostService) CreateBookmarkCollection(ctx context.Context, userID, name string) (*models.BookmarkCollection, error) {
	now := time.Now()
	coll := &models.BookmarkCollection{
		UserID:    userID,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.CreateCollection(ctx, coll); err != nil {
		return nil, err
	}
	coll.ItemsCount = 0
	return coll, nil
}

// ListBookmarkCollections renvoie les collections de l'utilisateur, chacune
// enrichie de son compteur d'items (calculé à la lecture). La collection par
// défaut est garantie présente (créée à la volée) et placée en tête.
func (s *PostService) ListBookmarkCollections(ctx context.Context, userID string) ([]models.BookmarkCollection, error) {
	if _, err := s.repo.EnsureDefaultCollection(ctx, userID); err != nil {
		return nil, err
	}
	colls, err := s.repo.ListCollections(ctx, userID)
	if err != nil {
		return nil, err
	}
	for i := range colls {
		n, err := s.repo.CountBookmarks(ctx, userID, colls[i].ID.Hex())
		if err != nil {
			return nil, err
		}
		colls[i].ItemsCount = n
	}
	return colls, nil
}

// RenameBookmarkCollection renomme une collection appartenant à l'utilisateur.
// La collection par défaut n'est pas renommable.
func (s *PostService) RenameBookmarkCollection(ctx context.Context, collectionID, userID, name string) (*models.BookmarkCollection, error) {
	coll, err := s.getOwnedCollection(ctx, collectionID, userID)
	if err != nil {
		return nil, err
	}
	if coll.IsDefault {
		return nil, ErrDefaultCollection
	}
	updated, err := s.repo.RenameCollection(ctx, coll.ID, name)
	if err != nil {
		return nil, translateCollectionNotFound(err)
	}
	n, err := s.repo.CountBookmarks(ctx, userID, updated.ID.Hex())
	if err != nil {
		return nil, err
	}
	updated.ItemsCount = n
	return updated, nil
}

// DeleteBookmarkCollection supprime une collection de l'utilisateur et ses signets.
// La collection par défaut n'est pas supprimable.
func (s *PostService) DeleteBookmarkCollection(ctx context.Context, collectionID, userID string) error {
	coll, err := s.getOwnedCollection(ctx, collectionID, userID)
	if err != nil {
		return err
	}
	if coll.IsDefault {
		return ErrDefaultCollection
	}
	if err := s.repo.DeleteCollection(ctx, coll.ID); err != nil {
		return translateCollectionNotFound(err)
	}
	_ = s.repo.DeleteBookmarksByCollection(ctx, collectionID)
	return nil
}

// --- Signets -----------------------------------------------------------------

// Bookmark range un post côté utilisateur. Deux modes :
//   - collectionID fourni (appui long / sélecteur) → ajout explicite dans cette
//     collection, statut "filed".
//   - collectionID vide (clic court) → résolution de la fenêtre de rafale :
//     fenêtre active → range dans la dernière collection (statut "filed") ; sinon
//     statut "needs_choice" (le front ouvre le sélecteur, rien n'est rangé).
func (s *PostService) Bookmark(ctx context.Context, userID, postID, collectionID string) (*BookmarkResult, error) {
	if _, err := s.requirePost(ctx, postID); err != nil {
		return nil, err
	}

	if collectionID != "" {
		coll, err := s.fileBookmark(ctx, userID, postID, collectionID)
		if err != nil {
			return nil, err
		}
		return &BookmarkResult{Status: BookmarkStatusFiled, Collection: coll}, nil
	}

	// Clic court : la fenêtre de rafale décide.
	prefs, err := s.repo.GetPrefs(ctx, userID)
	if err != nil {
		return nil, err
	}
	if withinSessionWindow(prefs, s.bookmarkWindow, time.Now()) {
		// La dernière collection doit toujours exister et appartenir à l'utilisateur.
		if coll, err := s.getOwnedCollection(ctx, prefs.LastCollectionID, userID); err == nil {
			filed, err := s.fileBookmark(ctx, userID, postID, coll.ID.Hex())
			if err != nil {
				return nil, err
			}
			return &BookmarkResult{Status: BookmarkStatusFiled, Collection: filed}, nil
		}
		// Collection disparue → on retombe sur le choix.
	}

	colls, err := s.ListBookmarkCollections(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &BookmarkResult{Status: BookmarkStatusNeedsChoice, Collections: colls}, nil
}

// fileBookmark range le post dans une collection (idempotent) et repousse la
// fenêtre de rafale (UpsertPrefs). Renvoie la collection avec son compteur à jour.
func (s *PostService) fileBookmark(ctx context.Context, userID, postID, collectionID string) (*models.BookmarkCollection, error) {
	coll, err := s.getOwnedCollection(ctx, collectionID, userID)
	if err != nil {
		return nil, err
	}
	if _, err := s.repo.AddBookmark(ctx, userID, postID, coll.ID.Hex()); err != nil {
		return nil, err
	}
	if err := s.repo.UpsertPrefs(ctx, userID, coll.ID.Hex(), time.Now()); err != nil {
		return nil, err
	}
	n, err := s.repo.CountBookmarks(ctx, userID, coll.ID.Hex())
	if err != nil {
		return nil, err
	}
	coll.ItemsCount = n
	return coll, nil
}

// Unbookmark retire un post d'une collection précise (collectionID fourni) ou de
// TOUTES les collections (collectionID vide → dé-signer complet).
func (s *PostService) Unbookmark(ctx context.Context, userID, postID, collectionID string) error {
	if collectionID == "" {
		_, err := s.repo.RemoveAllBookmarksForPost(ctx, userID, postID)
		return err
	}
	coll, err := s.getOwnedCollection(ctx, collectionID, userID)
	if err != nil {
		return err
	}
	_, err = s.repo.RemoveBookmark(ctx, userID, postID, coll.ID.Hex())
	return err
}

// BookmarkedPostIDs renvoie les ids des posts signés par l'utilisateur (état des
// boutons signet côté front).
func (s *PostService) BookmarkedPostIDs(ctx context.Context, userID string) ([]string, error) {
	return s.repo.BookmarkedPostIDs(ctx, userID)
}

// PostBookmarkCollectionIDs renvoie les ids des collections contenant un post
// (coche le sélecteur « Ranger dans… »).
func (s *PostService) PostBookmarkCollectionIDs(ctx context.Context, userID, postID string) ([]string, error) {
	if _, err := parseID(postID); err != nil {
		return nil, err
	}
	return s.repo.PostBookmarkCollectionIDs(ctx, userID, postID)
}

// ListAllBookmarkedPosts renvoie la vue « Tous mes signets » (union dédupliquée,
// triée par signet le plus récent), résolue en posts.
func (s *PostService) ListAllBookmarkedPosts(ctx context.Context, userID string, limit, offset int64) ([]models.Post, error) {
	ids, err := s.repo.AllBookmarkedPostIDs(ctx, userID, clampLimit(limit), clampOffset(offset))
	if err != nil {
		return nil, err
	}
	return s.resolvePosts(ctx, ids), nil
}

// ListBookmarksInCollection renvoie les posts d'une collection de l'utilisateur
// (du plus récemment rangé au plus ancien).
func (s *PostService) ListBookmarksInCollection(ctx context.Context, collectionID, userID string, limit, offset int64) ([]models.Post, error) {
	coll, err := s.getOwnedCollection(ctx, collectionID, userID)
	if err != nil {
		return nil, err
	}
	ids, err := s.repo.BookmarksByCollection(ctx, userID, coll.ID.Hex(), clampLimit(limit), clampOffset(offset))
	if err != nil {
		return nil, err
	}
	return s.resolvePosts(ctx, ids), nil
}

// resolvePosts charge les posts pour une liste ordonnée d'ids, en préservant
// l'ordre et en ignorant les posts supprimés entre-temps (best-effort).
func (s *PostService) resolvePosts(ctx context.Context, ids []string) []models.Post {
	posts := make([]models.Post, 0, len(ids))
	for _, id := range ids {
		oid, err := parseID(id)
		if err != nil {
			continue
		}
		post, err := s.repo.Get(ctx, oid)
		if err != nil {
			continue
		}
		posts = append(posts, *post)
	}
	return posts
}

// --- Helpers -----------------------------------------------------------------

// requirePost valide l'id et confirme l'existence du post (404 sinon).
func (s *PostService) requirePost(ctx context.Context, postID string) (*models.Post, error) {
	oid, err := parseID(postID)
	if err != nil {
		return nil, err
	}
	post, err := s.repo.Get(ctx, oid)
	if err != nil {
		return nil, translateNotFound(err)
	}
	return post, nil
}

// getOwnedCollection valide l'id, charge la collection et vérifie qu'elle
// appartient à l'utilisateur (ErrCollectionNotFound si absente OU non possédée :
// on ne divulgue pas l'existence d'une collection d'autrui).
func (s *PostService) getOwnedCollection(ctx context.Context, collectionID, userID string) (*models.BookmarkCollection, error) {
	oid, err := bson.ObjectIDFromHex(collectionID)
	if err != nil {
		return nil, ErrInvalidID
	}
	coll, err := s.repo.GetCollection(ctx, oid)
	if err != nil {
		return nil, translateCollectionNotFound(err)
	}
	if coll.UserID != userID {
		return nil, ErrCollectionNotFound
	}
	return coll, nil
}

// withinSessionWindow : un clic court range automatiquement (sans redemander la
// collection) si l'utilisateur a déjà une dernière collection et que son dernier
// signet date de moins de `window`. Fonction PURE (testée). window <= 0
// désactive l'auto-classement (toujours proposer).
func withinSessionWindow(prefs *models.BookmarkPrefs, window time.Duration, now time.Time) bool {
	if window <= 0 || prefs == nil || prefs.LastCollectionID == "" || prefs.LastBookmarkAt == nil {
		return false
	}
	return now.Sub(*prefs.LastBookmarkAt) <= window
}

// translateCollectionNotFound mappe l'absence de document Mongo vers
// ErrCollectionNotFound ; laisse les autres erreurs (et nil) intactes.
func translateCollectionNotFound(err error) error {
	if errors.Is(err, mongo.ErrNoDocuments) {
		return ErrCollectionNotFound
	}
	return err
}
