package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
	return ensureSchema(ctx, mongoSchemaDatabase{db: db})
}

func ensureSchema(ctx context.Context, db schemaDatabase) error {
	if err := ensureCollections(ctx, db); err != nil {
		return err
	}
	if err := dropLegacyIndexes(ctx, db); err != nil {
		return err
	}
	if err := ensureIndexes(ctx, db); err != nil {
		return err
	}
	if err := backfillVisibility(ctx, db); err != nil {
		return err
	}
	if err := backfillCertification(ctx, db); err != nil {
		return err
	}
	if err := backfillBirthDate(ctx, db); err != nil {
		return err
	}
	return ensureSeed(ctx, db)
}

type schemaDatabase interface {
	ListCollectionNames(ctx context.Context, filter any, opts ...options.Lister[options.ListCollectionsOptions]) ([]string, error)
	CreateCollection(ctx context.Context, name string, opts ...options.Lister[options.CreateCollectionOptions]) error
	RunCommand(ctx context.Context, runCommand any, opts ...options.Lister[options.RunCmdOptions]) rawResult
	Collection(name string, opts ...options.Lister[options.CollectionOptions]) schemaCollection
}

type schemaCollection interface {
	UpdateMany(ctx context.Context, filter any, update any, opts ...options.Lister[options.UpdateManyOptions]) (*mongo.UpdateResult, error)
	UpdateOne(ctx context.Context, filter any, update any, opts ...options.Lister[options.UpdateOneOptions]) (*mongo.UpdateResult, error)
	Indexes() schemaIndexView
}

type schemaIndexView interface {
	DropOne(ctx context.Context, name string, opts ...options.Lister[options.DropIndexesOptions]) error
	CreateMany(ctx context.Context, models []mongo.IndexModel, opts ...options.Lister[options.CreateIndexesOptions]) ([]string, error)
}

type rawResult interface {
	Raw() (bson.Raw, error)
}

type mongoSchemaDatabase struct {
	db *mongo.Database
}

func (m mongoSchemaDatabase) ListCollectionNames(ctx context.Context, filter any, opts ...options.Lister[options.ListCollectionsOptions]) ([]string, error) {
	return m.db.ListCollectionNames(ctx, filter, opts...)
}

func (m mongoSchemaDatabase) CreateCollection(ctx context.Context, name string, opts ...options.Lister[options.CreateCollectionOptions]) error {
	return m.db.CreateCollection(ctx, name, opts...)
}

func (m mongoSchemaDatabase) RunCommand(ctx context.Context, runCommand any, opts ...options.Lister[options.RunCmdOptions]) rawResult {
	return m.db.RunCommand(ctx, runCommand, opts...)
}

func (m mongoSchemaDatabase) Collection(name string, opts ...options.Lister[options.CollectionOptions]) schemaCollection {
	return mongoSchemaCollection{collection: m.db.Collection(name, opts...)}
}

type mongoSchemaCollection struct {
	collection *mongo.Collection
}

func (m mongoSchemaCollection) UpdateMany(ctx context.Context, filter any, update any, opts ...options.Lister[options.UpdateManyOptions]) (*mongo.UpdateResult, error) {
	return m.collection.UpdateMany(ctx, filter, update, opts...)
}

func (m mongoSchemaCollection) UpdateOne(ctx context.Context, filter any, update any, opts ...options.Lister[options.UpdateOneOptions]) (*mongo.UpdateResult, error) {
	return m.collection.UpdateOne(ctx, filter, update, opts...)
}

func (m mongoSchemaCollection) Indexes() schemaIndexView {
	return m.collection.Indexes()
}

// backfillBirthDate pose une date de naissance par défaut (01/01/1999, traité
// comme MAJEUR) sur les profils antérieurs au champ (date absente). Sans elle,
// ces comptes seraient « mineurs par défaut » et verraient le contenu NSFW
// masqué alors qu'ils en disposaient en prod : on préserve l'expérience
// existante. Les nouveaux comptes (register Breezy + onboarding OAuth) saisissent
// TOUJOURS leur vraie date. Idempotent : filtre sur birth_date absente.
func backfillBirthDate(ctx context.Context, db schemaDatabase) error {
	defaultBirthDate := time.Date(1999, time.January, 1, 0, 0, 0, 0, time.UTC)
	filter := bson.M{"birth_date": bson.M{"$exists": false}}
	res, err := db.Collection("profiles").UpdateMany(ctx, filter,
		bson.M{"$set": bson.M{"birth_date": defaultBirthDate}})
	if err != nil {
		return fmt.Errorf("backfill birth_date : %w", err)
	}
	if res.ModifiedCount > 0 {
		slog.Info("migration: birth_date legacy posée au défaut", "count", res.ModifiedCount, "default", "1999-01-01")
	}
	return nil
}

// backfillVisibility remet à une valeur valide les champs `visibility`,
// `likes_visibility` et `activity_visibility` des documents écrits avant
// l'introduction de ces champs
// (ou avant leur valeur par défaut). Sans omitempty historique, la valeur zéro
// a pu être persistée comme chaîne vide : `""` ne respecte pas l'enum
// {public, private} du $jsonSchema, et la validation `strict` de Mongo rejette
// alors TOUTE mise à jour ultérieure du document (le doc complet est revalidé).
//
// ⚠️ Base de PROD : toute feature ajoutant un champ contraint (enum, required,
// index unique) doit fournir ce type de migration idempotente au boot, sinon les
// vieux documents bloquent leurs propres écritures. Idempotent : no-op une fois
// les documents corrigés (filtre sur champ absent OU chaîne vide).
func backfillVisibility(ctx context.Context, db schemaDatabase) error {
	coll := db.Collection("profiles")
	for _, field := range []string{"visibility", "likes_visibility", "activity_visibility"} {
		filter := bson.M{"$or": bson.A{
			bson.M{field: bson.M{"$exists": false}},
			bson.M{field: ""},
		}}
		res, err := coll.UpdateMany(ctx, filter, bson.M{"$set": bson.M{field: "public"}})
		if err != nil {
			return fmt.Errorf("backfill %q : %w", field, err)
		}
		if res.ModifiedCount > 0 {
			slog.Info("migration: profils legacy normalisés", "field", field, "count", res.ModifiedCount)
		}
	}
	return nil
}

// backfillCertification pose la valeur neutre sur les profils antérieurs à la
// certification. Le champ est enum-validé par Mongo ; les vieux documents sans
// valeur doivent donc être normalisés avant toute future mise à jour.
func backfillCertification(ctx context.Context, db schemaDatabase) error {
	coll := db.Collection("profiles")
	filter := bson.M{"$or": bson.A{
		bson.M{"certification": bson.M{"$exists": false}},
		bson.M{"certification": ""},
	}}
	res, err := coll.UpdateMany(ctx, filter, bson.M{"$set": bson.M{"certification": "none"}})
	if err != nil {
		return fmt.Errorf("backfill certification : %w", err)
	}
	if res.ModifiedCount > 0 {
		slog.Info("migration: certification profils legacy posée au défaut", "count", res.ModifiedCount)
	}
	return nil
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
func ensureCollections(ctx context.Context, db schemaDatabase) error {
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
// schéma courant. Sans ça, les volumes Mongo de dev gardent les anciens
// validateurs et peuvent refuser des documents pourtant valides côté code.
func updateValidator(ctx context.Context, db schemaDatabase, name string) error {
	if _, err := db.RunCommand(ctx, bson.D{
		{Key: "collMod", Value: name},
		{Key: "validator", Value: validators[name]},
	}).Raw(); err != nil {
		return fmt.Errorf("mise à jour validateur %q : %w", name, err)
	}
	return nil
}

// dropLegacyIndexes retire les index d'anciennes versions du schéma qui ne
// correspondent plus au modèle actuel. username vit maintenant dans
// user-service ; garder cet index unique dans profiles bloque tous les profils
// sans username après le premier document.
func dropLegacyIndexes(ctx context.Context, db schemaDatabase) error {
	if err := db.Collection("profiles").Indexes().DropOne(ctx, "username_1"); err != nil {
		if isIndexNotFound(err) {
			return nil
		}
		return fmt.Errorf("suppression index legacy profiles.username_1 : %w", err)
	}
	return nil
}

func isIndexNotFound(err error) bool {
	var commandErr mongo.CommandError
	return errors.As(err, &commandErr) && commandErr.Code == 27
}

// ensureIndexes crée les index (CreateMany est idempotent pour un index de
// spécification identique).
func ensureIndexes(ctx context.Context, db schemaDatabase) error {
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
func ensureSeed(ctx context.Context, db schemaDatabase) error {
	now := time.Now().UTC()
	filter := bson.M{"user_id": adminUserID}
	update := bson.M{
		"$setOnInsert": bson.M{
			"user_id":             adminUserID,
			"display_name":        "Administrateur Breezy",
			"bio":                 "Administrateur de Breezy",
			"avatar_url":          "",
			"banner_url":          "",
			"website":             "",
			"location":            "",
			"visibility":          "public",
			"likes_visibility":    "public",
			"activity_visibility": "public",
			"certification":       "none",
			"is_online":           false,
			"created_at":          now,
			"updated_at":          now,
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
				"user_id":             bson.M{"bsonType": "string"},
				"display_name":        bson.M{"bsonType": "string", "maxLength": 100},
				"bio":                 bson.M{"bsonType": "string", "maxLength": 160},
				"avatar_url":          bson.M{"bsonType": "string"},
				"banner_url":          bson.M{"bsonType": "string"},
				"website":             bson.M{"bsonType": "string"},
				"location":            bson.M{"bsonType": "string"},
				"visibility":          bson.M{"bsonType": "string", "enum": bson.A{"public", "private"}},
				"likes_visibility":    bson.M{"bsonType": "string", "enum": bson.A{"public", "private"}},
				"activity_visibility": bson.M{"bsonType": "string", "enum": bson.A{"public", "private"}},
				"certification":       bson.M{"bsonType": "string", "enum": bson.A{"none", "political", "public_figure"}},
				// Champs optionnels (validés uniquement s'ils sont présents).
				"is_online":               bson.M{"bsonType": "bool"},
				"nsfw_enabled":            bson.M{"bsonType": "bool"},
				"tutorial_done":           bson.M{"bsonType": "bool"},
				"birth_date":              bson.M{"bsonType": "date"},
				"gender":                  bson.M{"bsonType": "string", "enum": bson.A{"male", "female"}},
				"nationality":             bson.M{"bsonType": "string", "pattern": "^[A-Z]{2}$"},
				"display_name_changed_at": bson.M{"bsonType": "date"},
				"last_login_at":           bson.M{"bsonType": "date"},
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
