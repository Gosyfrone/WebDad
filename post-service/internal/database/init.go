package database

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// collectionOrder fige l'ordre de création (déterministe pour les logs/tests).
var collectionOrder = []string{"posts", "comments", "likes", "reports"}

// EnsureSchema crée les collections (avec validateurs $jsonSchema) et les
// index du post-service, de façon idempotente. Le service possède ainsi son
// schéma et le maintient au démarrage — il fonctionne sans script d'init
// externe, en local comme en stack Docker.
func EnsureSchema(ctx context.Context, db *mongo.Database) error {
	if err := ensureCollections(ctx, db); err != nil {
		return err
	}
	return ensureIndexes(ctx, db)
}

// ensureCollections crée les collections manquantes avec leur validateur.
// (Une collection déjà présente est laissée telle quelle.)
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
			continue
		}
		opts := options.CreateCollection().SetValidator(validators[name])
		if err := db.CreateCollection(ctx, name, opts); err != nil {
			return fmt.Errorf("création collection %q : %w", name, err)
		}
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
				"author_id":      bson.M{"bsonType": "string"},
				"content":        bson.M{"bsonType": "string", "maxLength": 280},
				"is_hidden":      bson.M{"bsonType": "bool"},
				"hidden_by":      bson.M{"bsonType": bson.A{"string", "null"}},
				"hidden_at":      bson.M{"bsonType": bson.A{"date", "null"}},
				"likes_count":    bson.M{"bsonType": "int", "minimum": 0},
				"comments_count": bson.M{"bsonType": "int", "minimum": 0},
				"reports_count":  bson.M{"bsonType": "int", "minimum": 0},
				"created_at":     bson.M{"bsonType": "date"},
				"updated_at":     bson.M{"bsonType": "date"},
			},
		},
	},
	"comments": {
		"$jsonSchema": bson.M{
			"bsonType": "object",
			"required": bson.A{"post_id", "author_id", "content", "created_at"},
			"properties": bson.M{
				"post_id":     bson.M{"bsonType": "string"},
				"parent_id":   bson.M{"bsonType": bson.A{"string", "null"}},
				"author_id":   bson.M{"bsonType": "string"},
				"content":     bson.M{"bsonType": "string", "maxLength": 280},
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
}

// indexes : index par collection (portés depuis post-init.js).
var indexes = map[string][]mongo.IndexModel{
	"posts": {
		{Keys: bson.D{{Key: "author_id", Value: 1}}},
		{Keys: bson.D{{Key: "created_at", Value: -1}}}, // tri fil d'actu
		{Keys: bson.D{{Key: "is_hidden", Value: 1}}},
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
	"reports": {
		{Keys: bson.D{{Key: "post_id", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}}},
	},
}
