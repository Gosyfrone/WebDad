package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/webdad/post-service/internal/models"
)

// --- Playlists de signets (bookmark_collections) -----------------------------

// CreateCollection insère une playlist et renseigne coll.ID.
func (r *PostRepository) CreateCollection(ctx context.Context, coll *models.BookmarkCollection) error {
	res, err := r.bookmarkCollections.InsertOne(ctx, coll)
	if err != nil {
		return err
	}
	if oid, ok := res.InsertedID.(bson.ObjectID); ok {
		coll.ID = oid
	}
	return nil
}

// GetCollection renvoie une playlist par son ObjectID (mongo.ErrNoDocuments si absente).
func (r *PostRepository) GetCollection(ctx context.Context, id bson.ObjectID) (*models.BookmarkCollection, error) {
	var coll models.BookmarkCollection
	if err := r.bookmarkCollections.FindOne(ctx, bson.M{"_id": id}).Decode(&coll); err != nil {
		return nil, err
	}
	return &coll, nil
}

// DefaultCollectionName : nom stocké de la playlist par défaut. Le front affiche
// un libellé localisé pour les playlists `is_default` (cf. `bookmarks.default_name`),
// cette valeur n'est qu'un repli.
const DefaultCollectionName = "Mes signets"

// EnsureDefaultCollection renvoie la playlist par défaut de l'utilisateur, en la
// créant si elle n'existe pas encore (idempotent, protégé par l'index unique
// partiel sur `is_default`). Permet d'enregistrer sans avoir à créer de playlist.
func (r *PostRepository) EnsureDefaultCollection(ctx context.Context, userID string) (*models.BookmarkCollection, error) {
	existing, err := r.findDefaultCollection(ctx, userID)
	if err == nil {
		return existing, nil
	}
	if err != mongo.ErrNoDocuments {
		return nil, err
	}
	now := time.Now()
	coll := &models.BookmarkCollection{
		UserID:    userID,
		Name:      DefaultCollectionName,
		IsDefault: true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := r.CreateCollection(ctx, coll); err != nil {
		// Course : un autre appel l'a créée entre-temps (index unique partiel).
		if mongo.IsDuplicateKeyError(err) {
			return r.findDefaultCollection(ctx, userID)
		}
		return nil, err
	}
	return coll, nil
}

func (r *PostRepository) findDefaultCollection(ctx context.Context, userID string) (*models.BookmarkCollection, error) {
	var coll models.BookmarkCollection
	if err := r.bookmarkCollections.FindOne(ctx, bson.M{"user_id": userID, "is_default": true}).Decode(&coll); err != nil {
		return nil, err
	}
	return &coll, nil
}

// ListCollections renvoie les playlists d'un utilisateur : la playlist par défaut
// d'abord (`is_default`), puis de la plus récente à la plus ancienne. (Le
// compteur d'items est calculé par la couche service.)
func (r *PostRepository) ListCollections(ctx context.Context, userID string) ([]models.BookmarkCollection, error) {
	opts := options.Find().SetSort(bson.D{{Key: "is_default", Value: -1}, {Key: "created_at", Value: -1}})
	cursor, err := r.bookmarkCollections.Find(ctx, bson.M{"user_id": userID}, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	colls := []models.BookmarkCollection{}
	if err := cursor.All(ctx, &colls); err != nil {
		return nil, err
	}
	return colls, nil
}

// RenameCollection renomme une playlist et renvoie le document à jour.
func (r *PostRepository) RenameCollection(ctx context.Context, id bson.ObjectID, name string) (*models.BookmarkCollection, error) {
	update := bson.M{"$set": bson.M{"name": name, "updated_at": time.Now()}}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var coll models.BookmarkCollection
	if err := r.bookmarkCollections.FindOneAndUpdate(ctx, bson.M{"_id": id}, update, opts).Decode(&coll); err != nil {
		return nil, err
	}
	return &coll, nil
}

// DeleteCollection supprime une playlist (mongo.ErrNoDocuments si rien supprimé).
func (r *PostRepository) DeleteCollection(ctx context.Context, id bson.ObjectID) error {
	res, err := r.bookmarkCollections.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// CountBookmarks compte les signets d'une playlist (compteur calculé à la lecture).
func (r *PostRepository) CountBookmarks(ctx context.Context, userID, collectionID string) (int64, error) {
	return r.bookmarks.CountDocuments(ctx, bson.M{"user_id": userID, "collection_id": collectionID})
}

// --- Signets (bookmarks) -----------------------------------------------------

// AddBookmark range un post dans une playlist (idempotent grâce à l'index
// unique user_id+post_id+collection_id). Renvoie true si le signet a été créé,
// false s'il existait déjà.
func (r *PostRepository) AddBookmark(ctx context.Context, userID, postID, collectionID string) (bool, error) {
	_, err := r.bookmarks.InsertOne(ctx, &models.Bookmark{
		UserID:       userID,
		PostID:       postID,
		CollectionID: collectionID,
		CreatedAt:    time.Now(),
	})
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// RemoveBookmark retire un post d'une playlist précise. Renvoie true si un
// signet a effectivement été supprimé.
func (r *PostRepository) RemoveBookmark(ctx context.Context, userID, postID, collectionID string) (bool, error) {
	res, err := r.bookmarks.DeleteOne(ctx, bson.M{
		"user_id":       userID,
		"post_id":       postID,
		"collection_id": collectionID,
	})
	if err != nil {
		return false, err
	}
	return res.DeletedCount > 0, nil
}

// RemoveAllBookmarksForPost retire un post de TOUTES les playlists de
// l'utilisateur (« dé-signer » complet). Renvoie le nombre de signets supprimés.
func (r *PostRepository) RemoveAllBookmarksForPost(ctx context.Context, userID, postID string) (int64, error) {
	res, err := r.bookmarks.DeleteMany(ctx, bson.M{"user_id": userID, "post_id": postID})
	if err != nil {
		return 0, err
	}
	return res.DeletedCount, nil
}

// BookmarkedPostIDs renvoie les ids des posts signés (au moins une playlist) par
// un utilisateur — initialise l'état des boutons signet côté front, façon
// LikedPostIDs. Dédupliqué : un post rangé dans plusieurs playlists ne doit
// apparaître qu'une fois (Distinct côté Mongo, contrairement à distinctStrings
// qui projette sans dédupliquer).
func (r *PostRepository) BookmarkedPostIDs(ctx context.Context, userID string) ([]string, error) {
	var ids []string
	if err := r.bookmarks.Distinct(ctx, "post_id", bson.M{"user_id": userID}).Decode(&ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// PostBookmarkCollectionIDs renvoie les ids des playlists d'un utilisateur qui
// contiennent un post (coche le sélecteur « Ranger dans… »).
func (r *PostRepository) PostBookmarkCollectionIDs(ctx context.Context, userID, postID string) ([]string, error) {
	return r.distinctStrings(ctx, r.bookmarks, bson.M{"user_id": userID, "post_id": postID}, "collection_id")
}

// BookmarksByCollection renvoie les post_ids d'une playlist, du plus récemment
// rangé au plus ancien, paginés.
func (r *PostRepository) BookmarksByCollection(ctx context.Context, userID, collectionID string, limit, skip int64) ([]string, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(limit).
		SetSkip(skip).
		SetProjection(bson.M{"post_id": 1})

	cursor, err := r.bookmarks.Find(ctx, bson.M{"user_id": userID, "collection_id": collectionID}, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var docs []bson.M
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	return postIDsFromDocs(docs), nil
}

// AllBookmarkedPostIDs renvoie la vue « Tous mes signets » : les posts signés au
// moins une fois (peu importe la playlist), dédupliqués et triés par signet le
// plus récent, paginés. Un post rangé dans 2 playlists n'apparaît qu'une fois.
func (r *PostRepository) AllBookmarkedPostIDs(ctx context.Context, userID string, limit, skip int64) ([]string, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"user_id": userID}}},
		{{Key: "$sort", Value: bson.D{{Key: "created_at", Value: -1}}}},
		{{Key: "$group", Value: bson.M{"_id": "$post_id", "created_at": bson.M{"$first": "$created_at"}}}},
		{{Key: "$sort", Value: bson.D{{Key: "created_at", Value: -1}}}},
		{{Key: "$skip", Value: skip}},
		{{Key: "$limit", Value: limit}},
	}
	cursor, err := r.bookmarks.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var docs []bson.M
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(docs))
	for _, d := range docs {
		if v, ok := d["_id"].(string); ok {
			ids = append(ids, v)
		}
	}
	return ids, nil
}

// DeleteBookmarksByPost purge les signets d'un post pour TOUS les utilisateurs
// (nettoyage à la suppression du post).
func (r *PostRepository) DeleteBookmarksByPost(ctx context.Context, postID string) error {
	_, err := r.bookmarks.DeleteMany(ctx, bson.M{"post_id": postID})
	return err
}

// DeleteBookmarksByCollection purge les signets d'une playlist (à sa suppression).
func (r *PostRepository) DeleteBookmarksByCollection(ctx context.Context, collectionID string) error {
	_, err := r.bookmarks.DeleteMany(ctx, bson.M{"collection_id": collectionID})
	return err
}

// --- Préférences de rafale (bookmark_prefs) ----------------------------------

// GetPrefs renvoie l'état de rafale d'un utilisateur, ou nil s'il n'a jamais
// signé (mongo.ErrNoDocuments traité comme « pas de prefs »).
func (r *PostRepository) GetPrefs(ctx context.Context, userID string) (*models.BookmarkPrefs, error) {
	var prefs models.BookmarkPrefs
	err := r.bookmarkPrefs.FindOne(ctx, bson.M{"user_id": userID}).Decode(&prefs)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &prefs, nil
}

// UpsertPrefs pose la dernière playlist utilisée + la date du dernier signet
// (repousse la fenêtre glissante).
func (r *PostRepository) UpsertPrefs(ctx context.Context, userID, collectionID string, at time.Time) error {
	_, err := r.bookmarkPrefs.UpdateOne(
		ctx,
		bson.M{"user_id": userID},
		bson.M{"$set": bson.M{"last_collection_id": collectionID, "last_bookmark_at": at}},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

// postIDsFromDocs extrait le champ post_id d'une projection, en préservant l'ordre.
func postIDsFromDocs(docs []bson.M) []string {
	ids := make([]string, 0, len(docs))
	for _, d := range docs {
		if v, ok := d["post_id"].(string); ok {
			ids = append(ids, v)
		}
	}
	return ids
}
