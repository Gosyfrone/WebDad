package database

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// collectionOrder fige l'ordre de création (déterministe pour les logs/tests).
var collectionOrder = []string{"user_keys", "key_backups", "conversations", "members", "messages"}

// EnsureSchema crée les collections (avec validateurs $jsonSchema) et les index
// du message-service, de façon idempotente. Le service possède ainsi son schéma
// et le maintient au démarrage — autonome, sans script d'init externe (même
// pattern que post-service / profil-service).
//
// Note E2EE : le serveur ne stocke JAMAIS de clair. Les messages ne portent que
// `ciphertext`+`nonce` (chiffrés côté client) ; la clé de contenu d'une
// conversation est stockée emballée par membre (`key_envelope`), déchiffrable
// uniquement par le membre avec sa clé privée. Le serveur reste aveugle au
// contenu — il ne contrôle que l'appartenance et les rôles.
func EnsureSchema(ctx context.Context, db *mongo.Database) error {
	if err := ensureCollections(ctx, db); err != nil {
		return err
	}
	return ensureIndexes(ctx, db)
}

// ensureCollections crée/maintient les collections + leur validateur. Le
// validateur est resynchronisé via collMod sur une collection existante (comme
// profil-service) → un volume créé par une version antérieure est corrigé.
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
			// Resync du validateur (idempotent).
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
	// Registre des clés publiques d'identité (X25519, base64). La clé PRIVÉE ne
	// quitte jamais le navigateur : elle n'est jamais stockée ici.
	"user_keys": {
		"$jsonSchema": bson.M{
			"bsonType": "object",
			"required": bson.A{"user_id", "public_key", "created_at"},
			"properties": bson.M{
				"user_id":    bson.M{"bsonType": "string"},
				"public_key": bson.M{"bsonType": "string"},
				"created_at": bson.M{"bsonType": "date"},
				"updated_at": bson.M{"bsonType": "date"},
			},
		},
	},
	// Sauvegarde CHIFFRÉE de la clé privée d'identité (zero-knowledge). Le serveur
	// ne stocke que des blobs opaques : clé privée emballée par une clé dérivée
	// (Argon2id) d'une phrase de passe que seul l'utilisateur connaît.
	"key_backups": {
		"$jsonSchema": bson.M{
			"bsonType": "object",
			"required": bson.A{"user_id", "salt", "nonce", "wrapped_private_key", "kdf_params", "public_key", "created_at"},
			"properties": bson.M{
				"user_id":             bson.M{"bsonType": "string"},
				"salt":                bson.M{"bsonType": "string"},
				"nonce":               bson.M{"bsonType": "string"},
				"wrapped_private_key": bson.M{"bsonType": "string"},
				"kdf_params":          bson.M{"bsonType": "string"},
				"public_key":          bson.M{"bsonType": "string"},
				"created_at":          bson.M{"bsonType": "date"},
				"updated_at":          bson.M{"bsonType": "date"},
			},
		},
	},
	// Conversations : dm (2 membres), group (≤32 talkers), community (talkers +
	// viewers). `dm_key` = paire d'ids triée « a:b » (dédup des DM, index unique).
	"conversations": {
		"$jsonSchema": bson.M{
			"bsonType": "object",
			"required": bson.A{"type", "member_ids", "created_by", "created_at"},
			"properties": bson.M{
				"type":        bson.M{"bsonType": "string", "enum": bson.A{"dm", "group", "community"}},
				"member_ids":  bson.M{"bsonType": "array", "items": bson.M{"bsonType": "string"}},
				"dm_key":      bson.M{"bsonType": bson.A{"string", "null"}},
				"title":       bson.M{"bsonType": bson.A{"string", "null"}},
				"title_nonce": bson.M{"bsonType": bson.A{"string", "null"}},
				// content_key : clé de contenu d'une COMMUNAUTÉ, détenue par le
				// serveur pour la remettre aux nouveaux arrivants (auto-join). Les
				// communautés ne sont donc PAS admin-proof (compromis hybride
				// assumé) ; DM/groupes n'ont jamais ce champ (clé jamais côté serveur).
				"content_key": bson.M{"bsonType": bson.A{"string", "null"}},
				"created_by":  bson.M{"bsonType": "string"},
				"created_at":  bson.M{"bsonType": "date"},
				"updated_at":  bson.M{"bsonType": "date"},
			},
		},
	},
	// Appartenance + rôle + enveloppe de clé (la clé de contenu de la
	// conversation, emballée pour CE membre avec sa clé publique).
	"members": {
		"$jsonSchema": bson.M{
			"bsonType": "object",
			"required": bson.A{"conversation_id", "user_id", "role", "created_at"},
			"properties": bson.M{
				"conversation_id": bson.M{"bsonType": "string"},
				"user_id":         bson.M{"bsonType": "string"},
				"role":            bson.M{"bsonType": "string", "enum": bson.A{"owner", "admin", "talker", "viewer"}},
				"key_envelope":    bson.M{"bsonType": bson.A{"string", "null"}},
				// État par-utilisateur (optionnel) : épinglage + suppression côté user
				// + curseur de lecture (non-lu calculé côté serveur).
				"pinned_at":    bson.M{"bsonType": bson.A{"date", "null"}},
				"cleared_at":   bson.M{"bsonType": bson.A{"date", "null"}},
				"last_read_at": bson.M{"bsonType": bson.A{"date", "null"}},
				"muted_at":     bson.M{"bsonType": bson.A{"date", "null"}},
				"created_at":   bson.M{"bsonType": "date"},
			},
		},
	},
	// Messages : UNIQUEMENT du chiffré. Pas de champ `content` en clair.
	"messages": {
		"$jsonSchema": bson.M{
			"bsonType": "object",
			"required": bson.A{"conversation_id", "sender_id", "ciphertext", "nonce", "created_at"},
			"properties": bson.M{
				"conversation_id":     bson.M{"bsonType": "string"},
				"sender_id":           bson.M{"bsonType": "string"},
				"ciphertext":          bson.M{"bsonType": "string"},
				"nonce":               bson.M{"bsonType": "string"},
				"original_ciphertext": bson.M{"bsonType": bson.A{"string", "null"}},
				"original_nonce":      bson.M{"bsonType": bson.A{"string", "null"}},
				"created_at":          bson.M{"bsonType": "date"},
				"edited_at":           bson.M{"bsonType": bson.A{"date", "null"}},
			},
		},
	},
}

// indexes : index par collection.
var indexes = map[string][]mongo.IndexModel{
	"user_keys": {
		{Keys: bson.D{{Key: "user_id", Value: 1}}, Options: options.Index().SetUnique(true)},
	},
	"key_backups": {
		{Keys: bson.D{{Key: "user_id", Value: 1}}, Options: options.Index().SetUnique(true)},
	},
	"conversations": {
		// Dédup des DM : un seul document par paire d'utilisateurs. Sparse car
		// seuls les DM portent un dm_key (groups/communities ne l'ont pas).
		{Keys: bson.D{{Key: "dm_key", Value: 1}}, Options: options.Index().SetUnique(true).SetSparse(true)},
		{Keys: bson.D{{Key: "updated_at", Value: -1}}},
		// Annuaire des communautés (liste publique, du plus actif au plus ancien).
		{Keys: bson.D{{Key: "type", Value: 1}, {Key: "updated_at", Value: -1}}},
	},
	"members": {
		// Un utilisateur n'apparaît qu'une fois par conversation.
		{Keys: bson.D{{Key: "conversation_id", Value: 1}, {Key: "user_id", Value: 1}}, Options: options.Index().SetUnique(true)},
		// Lister les conversations d'un utilisateur.
		{Keys: bson.D{{Key: "user_id", Value: 1}}},
	},
	"messages": {
		// Historique d'une conversation, du plus récent au plus ancien.
		{Keys: bson.D{{Key: "conversation_id", Value: 1}, {Key: "_id", Value: -1}}},
	},
}
