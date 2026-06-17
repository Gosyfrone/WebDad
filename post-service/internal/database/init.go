package database

import (
	"context"
	"fmt"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// collectionOrder fige l'ordre de création (déterministe pour les logs/tests).
var collectionOrder = []string{"posts", "comments", "likes", "comment_likes", "reposts", "poll_votes", "reports", "bookmark_collections", "bookmarks", "bookmark_prefs"}

// EnsureSchema crée les collections (avec validateurs $jsonSchema) et les
// index du post-service, de façon idempotente. Le service possède ainsi son
// schéma et le maintient au démarrage — il fonctionne sans script d'init
// externe, en local comme en stack Docker.
func EnsureSchema(ctx context.Context, db *mongo.Database) error {
	if err := ensureCollections(ctx, db); err != nil {
		return err
	}
	if err := backfillReplyAudience(ctx, db); err != nil {
		return err
	}
	if err := backfillCommentLikesCount(ctx, db); err != nil {
		return err
	}
	return ensureIndexes(ctx, db)
}

// backfillReplyAudience pose `reply_audience: "everyone"` sur les posts écrits
// avant l'introduction du champ (audience des réponses absente). Bien qu'un champ
// absent reste valide vis-à-vis de l'enum optionnelle du $jsonSchema (donc pas de
// crash en l'état), on matérialise la valeur par défaut sur les vieux documents
// pour rester explicite et homogène — et conforme à la règle 5b (toute feature à
// champ contraint fournit une migration idempotente au boot, vu les cas vécus
// `visibility`/`likes_visibility`). Idempotent : no-op une fois les docs corrigés
// (filtre sur champ absent OU chaîne vide).
func backfillReplyAudience(ctx context.Context, db *mongo.Database) error {
	coll := db.Collection("posts")
	filter := bson.M{"$or": bson.A{
		bson.M{"reply_audience": bson.M{"$exists": false}},
		bson.M{"reply_audience": ""},
	}}
	res, err := coll.UpdateMany(ctx, filter, bson.M{"$set": bson.M{"reply_audience": "everyone"}})
	if err != nil {
		return fmt.Errorf("backfill reply_audience : %w", err)
	}
	if res.ModifiedCount > 0 {
		slog.Info("migration: posts legacy normalisés", "field", "reply_audience", "count", res.ModifiedCount)
	}
	return nil
}

// backfillCommentLikesCount pose `likes_count: 0` sur les commentaires hérités
// d'avant l'introduction des likes de commentaire. Idempotent : une fois les
// documents corrigés, le filtre (champ absent OU vide) ne rematche plus rien.
func backfillCommentLikesCount(ctx context.Context, db *mongo.Database) error {
	coll := db.Collection("comments")
	filter := bson.M{"$or": bson.A{
		bson.M{"likes_count": bson.M{"$exists": false}},
		bson.M{"likes_count": nil},
	}}
	res, err := coll.UpdateMany(ctx, filter, bson.M{"$set": bson.M{"likes_count": 0}})
	if err != nil {
		return fmt.Errorf("backfill comments.likes_count : %w", err)
	}
	if res.ModifiedCount > 0 {
		slog.Info("migration: comments legacy normalisés", "field", "likes_count", "count", res.ModifiedCount)
	}
	return nil
}

// ensureCollections crée les collections manquantes avec leur validateur, et
// resynchronise le validateur des collections déjà présentes (`collMod`).
//
// Le service possède son schéma : il ne se contente pas de le poser à la
// création, il le MAINTIENT à chaque démarrage. Sans le `collMod`, un volume
// créé par une version antérieure conserverait indéfiniment son ancien
// validateur (ex. un schéma `posts` sans `pinned_at`) → les écritures conformes
// au schéma courant échoueraient.
func ensureCollections(ctx context.Context, db *mongo.Database) error {
	existing, err := db.ListCollectionNames(ctx, bson.M{})
	if err != nil {
		return fmt.Errorf("liste des collections : %w", err)
	}
	have := make(map[string]bool, len(existing))
	for _, n := range existing {
		have[n] = true
	}

	for _, name := range collectionOrder {
		if have[name] {
			if err := updateValidator(ctx, db, name); err != nil {
				return err
			}
			continue
		}
		opts := options.CreateCollection().SetValidator(validators[name])
		if err := db.CreateCollection(ctx, name, opts); err != nil {
			return fmt.Errorf("création collection %q : %w", name, err)
		}
	}
	return nil
}

// updateValidator aligne le validateur d'une collection existante avec le
// schéma courant. `collMod` est idempotent si le validateur est déjà à jour.
func updateValidator(ctx context.Context, db *mongo.Database, name string) error {
	if _, err := db.RunCommand(ctx, bson.D{
		{Key: "collMod", Value: name},
		{Key: "validator", Value: validators[name]},
	}).Raw(); err != nil {
		return fmt.Errorf("mise à jour validateur %q : %w", name, err)
	}
	return nil
}

// ensureIndexes crée les index (CreateMany est idempotent pour un index
// de spécification identique).
func ensureIndexes(ctx context.Context, db *mongo.Database) error {
	for _, coll := range collectionOrder {
		models := indexes[coll]
		if len(models) == 0 {
			continue
		}
		if _, err := db.Collection(coll).Indexes().CreateMany(ctx, models); err != nil {
			return fmt.Errorf("index sur %q : %w", coll, err)
		}
	}
	return nil
}

// validators : schémas $jsonSchema par collection (portés depuis l'ancien
// scripts/init-db/post-init.js, désormais supprimé). Champs en snake_case.
var validators = map[string]bson.M{
	"posts": {
		"$jsonSchema": bson.M{
			"bsonType": "object",
			"required": bson.A{"author_id", "content", "created_at"},
			"properties": bson.M{
				"author_id": bson.M{"bsonType": "string"},
				"content":   bson.M{"bsonType": "string", "maxLength": 280},
				"hashtags": bson.M{
					"bsonType": "array",
					"items":    bson.M{"bsonType": "string"},
				},
				"media": bson.M{
					"bsonType": "array",
					"maxItems": 4,
					"items": bson.M{
						"bsonType": "object",
						"required": bson.A{"url", "type"},
						"properties": bson.M{
							"url":  bson.M{"bsonType": "string"},
							"type": bson.M{"enum": bson.A{"image", "video"}},
						},
					},
				},
				"poll": bson.M{
					"bsonType": "object",
					"required": bson.A{"choices", "ends_at", "audience", "total_votes"},
					"properties": bson.M{
						"choices": bson.M{
							"bsonType": "array",
							"minItems": 2,
							"maxItems": 4,
							"items": bson.M{
								"bsonType": "object",
								"required": bson.A{"id", "label", "votes_count"},
								"properties": bson.M{
									"id":          bson.M{"bsonType": "string"},
									"label":       bson.M{"bsonType": "string", "maxLength": 80},
									"votes_count": bson.M{"bsonType": "int", "minimum": 0},
									// Optionnel : illustration du choix (chemin relatif /media/<id>).
									// Champ non requis → aucun backfill nécessaire (cf. Règle 5b).
									"image_url": bson.M{"bsonType": "string", "maxLength": 512},
								},
							},
						},
						"ends_at":     bson.M{"bsonType": "date"},
						"closed_at":   bson.M{"bsonType": bson.A{"date", "null"}},
						"audience":    bson.M{"enum": bson.A{"everyone", "followers"}},
						"total_votes": bson.M{"bsonType": "int", "minimum": 0},
					},
				},
				"reply_audience":  bson.M{"enum": bson.A{"everyone", "followers"}},
				"is_hidden":       bson.M{"bsonType": "bool"},
				"hidden_by":       bson.M{"bsonType": bson.A{"string", "null"}},
				"hidden_at":       bson.M{"bsonType": bson.A{"date", "null"}},
				"purge_warned_at": bson.M{"bsonType": bson.A{"date", "null"}},
				"likes_count":     bson.M{"bsonType": "int", "minimum": 0},
				"comments_count":  bson.M{"bsonType": "int", "minimum": 0},
				"reposts_count":   bson.M{"bsonType": "int", "minimum": 0},
				"quote_post_id":   bson.M{"bsonType": bson.A{"string", "null"}},
				"reports_count":   bson.M{"bsonType": "int", "minimum": 0},
				"pinned_at":       bson.M{"bsonType": bson.A{"date", "null"}},
				"created_at":      bson.M{"bsonType": "date"},
				"updated_at":      bson.M{"bsonType": "date"},
			},
		},
	},
	"comments": {
		"$jsonSchema": bson.M{
			"bsonType": "object",
			"required": bson.A{"post_id", "author_id", "content", "created_at"},
			"properties": bson.M{
				"post_id":   bson.M{"bsonType": "string"},
				"parent_id": bson.M{"bsonType": bson.A{"string", "null"}},
				"author_id": bson.M{"bsonType": "string"},
				"content":   bson.M{"bsonType": "string", "maxLength": 280},
				"media": bson.M{
					"bsonType": "array",
					"maxItems": 4,
					"items": bson.M{
						"bsonType": "object",
						"required": bson.A{"url", "type"},
						"properties": bson.M{
							"url":  bson.M{"bsonType": "string"},
							"type": bson.M{"enum": bson.A{"image", "video"}},
						},
					},
				},
				"likes_count": bson.M{"bsonType": "int", "minimum": 0},
				"reply_count": bson.M{"bsonType": "int", "minimum": 0},
				"is_hidden":   bson.M{"bsonType": "bool"},
				"created_at":  bson.M{"bsonType": "date"},
				"updated_at":  bson.M{"bsonType": "date"},
			},
		},
	},
	"likes": {
		"$jsonSchema": bson.M{
			"bsonType": "object",
			"required": bson.A{"post_id", "user_id", "created_at"},
			"properties": bson.M{
				"post_id":    bson.M{"bsonType": "string"},
				"user_id":    bson.M{"bsonType": "string"},
				"created_at": bson.M{"bsonType": "date"},
			},
		},
	},
	"reposts": {
		"$jsonSchema": bson.M{
			"bsonType": "object",
			"required": bson.A{"post_id", "user_id", "created_at"},
			"properties": bson.M{
				"post_id":    bson.M{"bsonType": "string"},
				"user_id":    bson.M{"bsonType": "string"},
				"created_at": bson.M{"bsonType": "date"},
			},
		},
	},
	"comment_likes": {
		"$jsonSchema": bson.M{
			"bsonType": "object",
			"required": bson.A{"comment_id", "user_id", "created_at"},
			"properties": bson.M{
				"comment_id": bson.M{"bsonType": "string"},
				"user_id":    bson.M{"bsonType": "string"},
				"created_at": bson.M{"bsonType": "date"},
			},
		},
	},
	"poll_votes": {
		"$jsonSchema": bson.M{
			"bsonType": "object",
			"required": bson.A{"post_id", "user_id", "choice_id", "created_at"},
			"properties": bson.M{
				"post_id":    bson.M{"bsonType": "string"},
				"user_id":    bson.M{"bsonType": "string"},
				"choice_id":  bson.M{"bsonType": "string"},
				"created_at": bson.M{"bsonType": "date"},
			},
		},
	},
	"reports": {
		"$jsonSchema": bson.M{
			"bsonType": "object",
			"required": bson.A{"post_id", "reporter_id", "reason", "created_at"},
			"properties": bson.M{
				"post_id":     bson.M{"bsonType": "string"},
				"reporter_id": bson.M{"bsonType": "string"},
				"reason":      bson.M{"bsonType": "string", "enum": bson.A{"spam", "harassment", "misinformation", "other"}},
				"status":      bson.M{"bsonType": "string", "enum": bson.A{"pending", "reviewed", "dismissed"}},
				"created_at":  bson.M{"bsonType": "date"},
			},
		},
	},
	// Collections de signets (par utilisateur). Pas de compteur stocké : il est
	// calculé à la lecture (cf. models.BookmarkCollection).
	"bookmark_collections": {
		"$jsonSchema": bson.M{
			"bsonType": "object",
			"required": bson.A{"user_id", "name", "created_at"},
			"properties": bson.M{
				"user_id":    bson.M{"bsonType": "string"},
				"name":       bson.M{"bsonType": "string", "maxLength": 60},
				"is_default": bson.M{"bsonType": "bool"},
				"created_at": bson.M{"bsonType": "date"},
				"updated_at": bson.M{"bsonType": "date"},
			},
		},
	},
	// Signets (appartenance many-to-many post↔collection).
	"bookmarks": {
		"$jsonSchema": bson.M{
			"bsonType": "object",
			"required": bson.A{"user_id", "post_id", "collection_id", "created_at"},
			"properties": bson.M{
				"user_id":       bson.M{"bsonType": "string"},
				"post_id":       bson.M{"bsonType": "string"},
				"collection_id": bson.M{"bsonType": "string"},
				"created_at":    bson.M{"bsonType": "date"},
			},
		},
	},
	// Préférences de rafale (un doc par utilisateur).
	"bookmark_prefs": {
		"$jsonSchema": bson.M{
			"bsonType": "object",
			"required": bson.A{"user_id"},
			"properties": bson.M{
				"user_id":            bson.M{"bsonType": "string"},
				"last_collection_id": bson.M{"bsonType": bson.A{"string", "null"}},
				"last_bookmark_at":   bson.M{"bsonType": bson.A{"date", "null"}},
			},
		},
	},
}

// indexes : index par collection (portés depuis post-init.js).
var indexes = map[string][]mongo.IndexModel{
	"posts": {
		{Keys: bson.D{{Key: "author_id", Value: 1}}},
		{Keys: bson.D{{Key: "created_at", Value: -1}}}, // tri fil d'actu
		{Keys: bson.D{{Key: "author_id", Value: 1}, {Key: "pinned_at", Value: -1}, {Key: "created_at", Value: -1}}},
		{Keys: bson.D{{Key: "is_hidden", Value: 1}}},
		{Keys: bson.D{{Key: "hashtags", Value: 1}, {Key: "created_at", Value: -1}}},
	},
	"comments": {
		{Keys: bson.D{{Key: "author_id", Value: 1}}},
		// Commentaires racine d'un post (parent_id null), triés chronologiquement.
		{Keys: bson.D{{Key: "post_id", Value: 1}, {Key: "parent_id", Value: 1}, {Key: "created_at", Value: 1}}},
		// Réponses d'un commentaire (parent_id = id racine), chronologiques.
		{Keys: bson.D{{Key: "parent_id", Value: 1}, {Key: "created_at", Value: 1}}},
	},
	"likes": {
		{Keys: bson.D{{Key: "post_id", Value: 1}, {Key: "user_id", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "user_id", Value: 1}}},
	},
	"comment_likes": {
		{Keys: bson.D{{Key: "comment_id", Value: 1}, {Key: "user_id", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "comment_id", Value: 1}}},
	},
	"reposts": {
		{Keys: bson.D{{Key: "post_id", Value: 1}, {Key: "user_id", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}}},
	},
	"poll_votes": {
		{Keys: bson.D{{Key: "post_id", Value: 1}, {Key: "user_id", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "post_id", Value: 1}}},
	},
	"reports": {
		{Keys: bson.D{{Key: "post_id", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}}},
	},
	"bookmark_collections": {
		// Collections d'un utilisateur, triées par date de création.
		{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}}},
		// Au plus UNE collection par défaut par utilisateur (index unique partiel).
		{
			Keys: bson.D{{Key: "user_id", Value: 1}},
			Options: options.Index().
				SetUnique(true).
				SetPartialFilterExpression(bson.M{"is_default": true}),
		},
	},
	"bookmarks": {
		// Idempotence : un post ne peut être rangé qu'une fois par collection.
		{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "post_id", Value: 1}, {Key: "collection_id", Value: 1}}, Options: options.Index().SetUnique(true)},
		// Contenu d'une collection, du plus récemment rangé au plus ancien.
		{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "collection_id", Value: 1}, {Key: "created_at", Value: -1}}},
		// Vue « Tous mes signets » + état des boutons (tous les signets d'un user).
		{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}}},
		// Nettoyage à la suppression d'un post (tous utilisateurs).
		{Keys: bson.D{{Key: "post_id", Value: 1}}},
	},
	"bookmark_prefs": {
		{Keys: bson.D{{Key: "user_id", Value: 1}}, Options: options.Index().SetUnique(true)},
	},
}
