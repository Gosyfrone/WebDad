package database

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// collectionOrder fige l'ordre de création (déterministe pour les logs/tests).
var collectionOrder = []string{"notifications"}

// EnsureSchema crée la collection `notifications` (avec validateur $jsonSchema)
// et ses index, de façon idempotente. Le service possède ainsi son schéma et le
// maintient au démarrage — autonome, sans script d'init externe (même pattern
// que post-service / message-service).
//
// Modèle d'agrégation (anti-spam) : une notification = un GROUPE identifié par
// `group_key` unique par destinataire (p. ex. `like:<post_id>`). 300 likes sur
// un post → un seul document, dont le compteur `count` est incrémenté. L'auteur
// (`last_actor_id`) + `count` suffisent à l'affichage « X et N autres ».
func EnsureSchema(ctx context.Context, db *mongo.Database) error {
	if err := ensureCollections(ctx, db); err != nil {
		return err
	}
	return ensureIndexes(ctx, db)
}

// ensureCollections crée/maintient les collections + leur validateur. Le
// validateur est resynchronisé via collMod sur une collection existante → un
// volume créé par une version antérieure est corrigé (même pattern qu'ailleurs).
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
			cmd := bson.D{
				{Key: "collMod", Value: name},
				{Key: "validator", Value: validators[name]},
				{Key: "validationLevel", Value: "moderate"},
			}
			if err := db.RunCommand(ctx, cmd).Err(); err != nil {
				return fmt.Errorf("collMod %q : %w", name, err)
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

// ensureIndexes crée les index (CreateMany est idempotent pour une spec identique).
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

// validators : schémas $jsonSchema par collection. Champs en snake_case.
var validators = map[string]bson.M{
	"notifications": {
		"$jsonSchema": bson.M{
			"bsonType": "object",
			"required": bson.A{"recipient_id", "group_key", "type", "last_actor_id", "count", "is_read", "created_at", "updated_at"},
			"properties": bson.M{
				"recipient_id":    bson.M{"bsonType": "string"},
				"group_key":       bson.M{"bsonType": "string"},
				"type":            bson.M{"bsonType": "string", "enum": bson.A{"like", "comment_like", "comment", "reply", "mention", "repost", "quote", "follow", "message", "message_mention", "follow_request", "follow_request_accepted", "follow_request_accept_confirm", "post_purge_warning"}},
				"post_id":         bson.M{"bsonType": bson.A{"string", "null"}},
				"comment_id":      bson.M{"bsonType": bson.A{"string", "null"}},
				"conversation_id": bson.M{"bsonType": bson.A{"string", "null"}},
				"last_actor_id":   bson.M{"bsonType": "string"},
				"actor_ids":       bson.M{"bsonType": "array", "items": bson.M{"bsonType": "string"}},
				// count = nombre d'événements agrégés (likes, commentaires…). int32
				// pour respecter le validateur Mongo (bsonType "int"). Pour `message`,
				// il représente les expéditeurs uniques (`actor_ids`) plutôt que les messages.
				"count":      bson.M{"bsonType": "int"},
				"is_read":    bson.M{"bsonType": "bool"},
				"created_at": bson.M{"bsonType": "date"},
				"updated_at": bson.M{"bsonType": "date"},
			},
		},
	},
}

// indexes : index par collection.
var indexes = map[string][]mongo.IndexModel{
	"notifications": {
		// Un seul document par (destinataire, groupe) → cible de l'upsert agrégé.
		{Keys: bson.D{{Key: "recipient_id", Value: 1}, {Key: "group_key", Value: 1}}, Options: options.Index().SetUnique(true)},
		// Liste d'un utilisateur, de la plus récente à la plus ancienne (curseur _id).
		{Keys: bson.D{{Key: "recipient_id", Value: 1}, {Key: "_id", Value: -1}}},
		// Compte des non-lues (badge).
		{Keys: bson.D{{Key: "recipient_id", Value: 1}, {Key: "is_read", Value: 1}}},
		// Cascade à la suppression d'un post.
		{Keys: bson.D{{Key: "post_id", Value: 1}}},
	},
}
