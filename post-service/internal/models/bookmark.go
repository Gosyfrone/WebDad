package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// BookmarkCollection — une « playlist » de signets, propre à un utilisateur
// (collection `bookmark_collections`). Champs en snake_case, alignés sur le
// validateur $jsonSchema (cf. internal/database/init.go).
//
// ItemsCount n'est PAS stocké : il est calculé à la lecture (CountDocuments sur
// `bookmarks`), comme les compteurs de followers du user-service — pas de
// dénormalisation à maintenir, donc pas de dérive possible. Le champ porte
// `bson:"-"` pour ne jamais être persisté.
type BookmarkCollection struct {
	ID     bson.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID string        `bson:"user_id" json:"user_id"`
	Name   string        `bson:"name" json:"name"`
	// IsDefault : playlist « par défaut » de l'utilisateur (une seule par compte,
	// créée à la volée). Toujours en tête de liste, NON supprimable et NON
	// renommable — mais on peut y ajouter/retirer des posts comme les autres.
	IsDefault  bool      `bson:"is_default" json:"is_default"`
	ItemsCount int64     `bson:"-" json:"items_count"`
	CreatedAt  time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt  time.Time `bson:"updated_at" json:"updated_at"`
}

// Bookmark — un post rangé dans une playlist (collection `bookmarks`).
// Many-to-many : un même post peut figurer dans plusieurs playlists, soit un
// document par triplet (user_id, post_id, collection_id). L'index unique sur ce
// triplet rend l'ajout idempotent.
type Bookmark struct {
	ID           bson.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID       string        `bson:"user_id" json:"user_id"`
	PostID       string        `bson:"post_id" json:"post_id"`
	CollectionID string        `bson:"collection_id" json:"collection_id"`
	CreatedAt    time.Time     `bson:"created_at" json:"created_at"`
}

// BookmarkPrefs — état « rafale » par utilisateur (collection `bookmark_prefs`,
// un document par user, `user_id` unique). Mémorise la dernière playlist
// utilisée et la date du dernier signet : tant que l'utilisateur enchaîne les
// signets dans la fenêtre glissante (cf. BOOKMARK_SESSION_WINDOW), un clic court
// les range automatiquement dans `last_collection_id` ; passé ce délai, le
// serveur redemande la playlist (« ouverture de rafale »).
type BookmarkPrefs struct {
	UserID           string     `bson:"user_id" json:"-"`
	LastCollectionID string     `bson:"last_collection_id" json:"last_collection_id"`
	LastBookmarkAt   *time.Time `bson:"last_bookmark_at" json:"last_bookmark_at"`
}

// CreateCollectionRequest : corps de POST /posts/bookmarks/collections.
type CreateCollectionRequest struct {
	Name string `json:"name" binding:"required,max=60"`
}

// RenameCollectionRequest : corps de PATCH /posts/bookmarks/collections/:cid.
type RenameCollectionRequest struct {
	Name string `json:"name" binding:"required,max=60"`
}

// BookmarkRequest : corps (optionnel) de POST/DELETE /posts/:id/bookmark.
// `collection_id` absent au clic court → le serveur résout la fenêtre de rafale
// (range dans la dernière playlist, ou propose un choix). Fourni → ajout/retrait
// explicite dans cette playlist précise (appui long / sélecteur).
type BookmarkRequest struct {
	CollectionID string `json:"collection_id"`
}
