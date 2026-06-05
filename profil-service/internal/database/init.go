package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// adminUserID : UUID figé de l'administrateur, partagé avec auth/user-service.
const adminUserID = "00000000-0000-0000-0000-000000000001"

// collectionOrder fige l'ordre de création (déterministe pour les logs/tests).
var collectionOrder = []string{"profiles"}

// EnsureSchema crée les collections (avec validateurs $jsonSchema), les index
// et le profil admin de seed, de façon idempotente. Le service possède ainsi
// son schéma et le maintient au démarrage — il fonctionne sans script d'init
// externe, en local comme en stack Docker (même pattern qu'auth/user/post).
func EnsureSchema(ctx context.Context, db *mongo.Database) error {
	if err := ensureCollections(ctx, db); err != nil {
		return err
	}
	if err := ensureIndexes(ctx, db); err != nil {
		return err
	}
	return ensureSeed(ctx, db)
}

// ensureCollections crée les collections manquantes avec leur validateur, et
// resynchronise le validateur des collections déjà présentes (`collMod`).
//
// Le service possède son schéma : il ne se contente pas de le poser à la
// création, il le MAINTIENT à chaque démarrage. Sans le `collMod`, un volume
// créé par une version antérieure conserverait indéfiniment son ancien
// validateur (ex. un schéma qui exigeait encore `username`, supprimé depuis du
// profil) → les écritures conformes au schéma courant échoueraient. `collMod`
// avec le validateur courant est idempotent (no-op s'il est déjà à jour).
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
			}
			if err := db.RunCommand(ctx, cmd).Err(); err != nil {
				return fmt.Errorf("maj validateur %q : %w", name, err)
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

// ensureIndexes crée les index (CreateMany est idempotent pour un index de
// spécification identique).
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

// ensureSeed insère le profil de l'administrateur (idempotent : upsert sur
// user_id). Le username vit dans user-service ; ici on ne porte que le
// décoratif (display_name + champs éditables).
func ensureSeed(ctx context.Context, db *mongo.Database) error {
	now := time.Now().UTC()
	filter := bson.M{"user_id": adminUserID}
	update := bson.M{
		"$setOnInsert": bson.M{
			"user_id":      adminUserID,
			"display_name": "Administrateur Breezy",
			"bio":          "Administrateur de Breezy",
			"avatar_url":   "",
			"banner_url":   "",
			"website":      "",
			"location":     "",
			"created_at":   now,
			"updated_at":   now,
		},
	}
	_, err := db.Collection("profiles").UpdateOne(ctx, filter, update, options.UpdateOne().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("seed admin : %w", err)
	}
	return nil
}

// validators : schémas $jsonSchema par collection. Le profil-service ne
// possède QUE les champs décoratifs : pas de username (→ user-service) ni de
// compteurs followers/following (→ user-service) pour éviter toute double
// source de vérité. Champs en snake_case.
var validators = map[string]bson.M{
	"profiles": {
		"$jsonSchema": bson.M{
			"bsonType": "object",
			"required": bson.A{"user_id", "created_at"},
			"properties": bson.M{
				"user_id":      bson.M{"bsonType": "string"},
				"display_name": bson.M{"bsonType": "string", "maxLength": 100},
				"bio":          bson.M{"bsonType": "string", "maxLength": 160},
				"avatar_url":   bson.M{"bsonType": "string"},
				"banner_url":   bson.M{"bsonType": "string"},
				"website":      bson.M{"bsonType": "string"},
				"location":     bson.M{"bsonType": "string"},
				// Champs optionnels (validés uniquement s'ils sont présents).
				"birth_date":              bson.M{"bsonType": "date"},
				"gender":                  bson.M{"bsonType": "string", "enum": bson.A{"male", "female"}},
				"display_name_changed_at": bson.M{"bsonType": "date"},
				"created_at":              bson.M{"bsonType": "date"},
				"updated_at":              bson.M{"bsonType": "date"},
			},
		},
	},
}

// indexes : index par collection. user_id unique = un profil par utilisateur.
var indexes = map[string][]mongo.IndexModel{
	"profiles": {
		{Keys: bson.D{{Key: "user_id", Value: 1}}, Options: options.Index().SetUnique(true)},
	},
}
