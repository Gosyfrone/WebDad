package database

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/webdad/report-service/internal/models"
)

// collectionOrder fige l'ordre de création (déterministe pour les logs/tests).
var collectionOrder = []string{"tickets", "warnings", "settings"}

// EnsureSchema crée les collections `tickets` et `warnings` (avec validateur
// $jsonSchema) et leurs index, de façon idempotente. Le service possède ainsi
// son schéma et le maintient au démarrage — autonome, sans script d'init externe
// (même pattern que notification-service / post-service).
//
// Modèle d'agrégation : un signalement de MODÉRATION crée OU enrichit un
// « ticket parent » unique par entité signalée (clé (entity_type, entity_id)) ;
// les signalements suivants sont empilés dans `reports[]` et `report_count` est
// incrémenté. Les rapports de BUG sont des tickets autonomes (pas de dédup).
func EnsureSchema(ctx context.Context, db *mongo.Database) error {
	if err := ensureCollections(ctx, db); err != nil {
		return err
	}
	if err := ensureIndexes(ctx, db); err != nil {
		return err
	}
	return ensureSettings(ctx, db)
}

// ensureSettings garantit l'existence du document SINGLETON de configuration
// (`_id="global"`) avec le seuil d'auto-masquage par défaut. Idempotent :
// `$setOnInsert` ne pose la valeur QUE si le document n'existe pas encore — un
// seuil déjà réglé par l'administrateur n'est jamais écrasé au redémarrage.
func ensureSettings(ctx context.Context, db *mongo.Database) error {
	opts := options.UpdateOne().SetUpsert(true)
	_, err := db.Collection("settings").UpdateByID(ctx, models.SettingsSingletonID,
		bson.M{"$setOnInsert": bson.M{"auto_hide_threshold": models.DefaultAutoHideThreshold}},
		opts,
	)
	if err != nil {
		return fmt.Errorf("seed settings : %w", err)
	}
	return nil
}

// ensureCollections crée/maintient les collections + leur validateur. Le
// validateur est resynchronisé via collMod sur une collection existante.
// validationLevel "moderate" : les sous-documents (`reports[]`, `actions[]`)
// restent souples (pas de revalidation profonde à chaque update).
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
	"tickets": {
		"$jsonSchema": bson.M{
			"bsonType": "object",
			"required": bson.A{"category", "entity_type", "status", "report_count", "last_reported_at", "created_at", "updated_at"},
			"properties": bson.M{
				// moderation = signalement de contenu/compte (onglet Modération) ;
				// bug = rapport technique (onglet Administration).
				"category":    bson.M{"bsonType": "string", "enum": bson.A{"moderation", "bug"}},
				"entity_type": bson.M{"bsonType": "string", "enum": bson.A{"post", "message", "group_message", "profile", "app"}},
				// entity_id : id de l'entité signalée (post, message, profil…).
				// Vide pour un bug (entity_type=app) → autonome, pas d'agrégation.
				"entity_id":       bson.M{"bsonType": bson.A{"string", "null"}},
				"entity_owner_id": bson.M{"bsonType": bson.A{"string", "null"}},
				// `approved` = entité jugée conforme (terminal, verrou de re-signalement).
				"status": bson.M{"bsonType": "string", "enum": bson.A{"open", "closed", "reopened", "approved"}},
				// report_count = nombre de signalements empilés. int32 (bsonType
				// "int" pour le validateur Mongo), maintenu par $inc.
				"report_count": bson.M{"bsonType": "int"},
				// reports_since_closed : compteur pour la réouverture auto (optionnel).
				"reports_since_closed": bson.M{"bsonType": "int"},
				// reason_tags : map motif → compteur (étiquettes récurrentes).
				"reason_tags":      bson.M{"bsonType": bson.A{"object", "null"}},
				"reports":          bson.M{"bsonType": bson.A{"array", "null"}},
				"actions":          bson.M{"bsonType": bson.A{"array", "null"}},
				"last_reported_at": bson.M{"bsonType": "date"},
				"created_at":       bson.M{"bsonType": "date"},
				"updated_at":       bson.M{"bsonType": "date"},
			},
		},
	},
	"settings": {
		"$jsonSchema": bson.M{
			"bsonType": "object",
			"required": bson.A{"auto_hide_threshold"},
			"properties": bson.M{
				// Seuil d'auto-masquage d'un post (int32). 0 = désactivé.
				"auto_hide_threshold": bson.M{"bsonType": "int"},
			},
		},
	},
	"warnings": {
		"$jsonSchema": bson.M{
			"bsonType": "object",
			"required": bson.A{"target_user_id", "message", "issued_by", "acknowledged", "created_at"},
			"properties": bson.M{
				"target_user_id":  bson.M{"bsonType": "string"},
				"ticket_id":       bson.M{"bsonType": bson.A{"string", "null"}},
				"message":         bson.M{"bsonType": "string"},
				"issued_by":       bson.M{"bsonType": "string"},
				"acknowledged":    bson.M{"bsonType": "bool"},
				"created_at":      bson.M{"bsonType": "date"},
				"acknowledged_at": bson.M{"bsonType": bson.A{"date", "null"}},
			},
		},
	},
}

// indexes : index par collection.
var indexes = map[string][]mongo.IndexModel{
	"tickets": {
		// Un seul ticket parent par entité signalée — UNIQUEMENT pour la
		// modération (index unique PARTIEL). Les rapports de bug ne sont pas
		// dédupliqués (chaque bug = un ticket autonome) → exclus du filtre.
		{
			Keys: bson.D{{Key: "entity_type", Value: 1}, {Key: "entity_id", Value: 1}},
			Options: options.Index().SetUnique(true).
				SetPartialFilterExpression(bson.M{"category": "moderation"}),
		},
		// Liste triée par volume de signalements décroissant (tri par défaut).
		{Keys: bson.D{{Key: "category", Value: 1}, {Key: "report_count", Value: -1}}},
		// Filtre par statut.
		{Keys: bson.D{{Key: "category", Value: 1}, {Key: "status", Value: 1}}},
		// Filtre par plage temporelle du dernier signalement.
		{Keys: bson.D{{Key: "last_reported_at", Value: -1}}},
	},
	"warnings": {
		// Avertissements en attente d'un utilisateur (interception à la connexion).
		{Keys: bson.D{{Key: "target_user_id", Value: 1}, {Key: "acknowledged", Value: 1}}},
	},
}
