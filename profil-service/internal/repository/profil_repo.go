// Package repository encapsule l'accès Mongo à la collection `profiles`.
// Aucune logique métier ici : uniquement des requêtes Mongo. Les erreurs
// brutes du driver (mongo.ErrNoDocuments, duplicate key) sont remontées
// telles quelles et interprétées par la couche service.
package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/webdad/profil-service/internal/models"
)

// ProfilRepository : accès à la collection `profiles`.
type ProfilRepository struct {
	collection collectionAPI
}

// NewProfilRepository construit le repository sur la collection `profiles`.
func NewProfilRepository(db *mongo.Database) *ProfilRepository {
	return &ProfilRepository{collection: mongoCollection{collection: db.Collection("profiles")}}
}

type collectionAPI interface {
	FindOne(ctx context.Context, filter any, opts ...options.Lister[options.FindOneOptions]) singleResult
	InsertOne(ctx context.Context, document any, opts ...options.Lister[options.InsertOneOptions]) (*mongo.InsertOneResult, error)
	Find(ctx context.Context, filter any, opts ...options.Lister[options.FindOptions]) (cursorAPI, error)
	FindOneAndUpdate(ctx context.Context, filter any, update any, opts ...options.Lister[options.FindOneAndUpdateOptions]) singleResult
	DeleteOne(ctx context.Context, filter any, opts ...options.Lister[options.DeleteOneOptions]) (*mongo.DeleteResult, error)
}

type singleResult interface {
	Decode(v any) error
}

type cursorAPI interface {
	All(ctx context.Context, results any) error
	Close(ctx context.Context) error
}

type mongoCollection struct {
	collection *mongo.Collection
}

func (m mongoCollection) FindOne(ctx context.Context, filter any, opts ...options.Lister[options.FindOneOptions]) singleResult {
	return m.collection.FindOne(ctx, filter, opts...)
}

func (m mongoCollection) InsertOne(ctx context.Context, document any, opts ...options.Lister[options.InsertOneOptions]) (*mongo.InsertOneResult, error) {
	return m.collection.InsertOne(ctx, document, opts...)
}

func (m mongoCollection) Find(ctx context.Context, filter any, opts ...options.Lister[options.FindOptions]) (cursorAPI, error) {
	return m.collection.Find(ctx, filter, opts...)
}

func (m mongoCollection) FindOneAndUpdate(ctx context.Context, filter any, update any, opts ...options.Lister[options.FindOneAndUpdateOptions]) singleResult {
	return m.collection.FindOneAndUpdate(ctx, filter, update, opts...)
}

func (m mongoCollection) DeleteOne(ctx context.Context, filter any, opts ...options.Lister[options.DeleteOneOptions]) (*mongo.DeleteResult, error) {
	return m.collection.DeleteOne(ctx, filter, opts...)
}

// GetByUserID retourne le profil d'un utilisateur. mongo.ErrNoDocuments si
// aucun profil (mappé en 404 par le service).
func (r *ProfilRepository) GetByUserID(ctx context.Context, userID string) (*models.Profil, error) {
	var p models.Profil
	err := r.collection.FindOne(ctx, bson.M{"user_id": userID}).Decode(&p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// Insert crée le profil. Renvoie une erreur duplicate key si le user_id existe
// déjà (index unique) — le service la traite comme une course de provisioning.
func (r *ProfilRepository) Insert(ctx context.Context, p *models.Profil) error {
	_, err := r.collection.InsertOne(ctx, p)
	return err
}

// SearchByDisplayName retourne les profils dont le display_name matche le motif
// `pattern` (déjà échappé par le service), insensible à la casse, triés par nom.
// ⚠️ Regex non ancrée = balayage de collection (acceptable à l'échelle du
// projet ; perspective : index `$text` ou collation pour passer à l'échelle).
func (r *ProfilRepository) SearchByDisplayName(ctx context.Context, pattern string, limit int64) ([]models.Profil, error) {
	filter := bson.M{"display_name": bson.M{"$regex": pattern, "$options": "i"}}
	opts := options.Find().
		SetLimit(limit).
		SetSort(bson.D{{Key: "display_name", Value: 1}})

	cur, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cur.Close(ctx) }()

	profils := make([]models.Profil, 0)
	if err := cur.All(ctx, &profils); err != nil {
		return nil, err
	}
	return profils, nil
}

// Update applique un $set au profil et retourne le document à jour.
// mongo.ErrNoDocuments si le profil n'existe pas. set ne doit contenir que
// des champs déjà résolus par le service (validés, règles appliquées).
func (r *ProfilRepository) Update(ctx context.Context, userID string, set bson.M) (*models.Profil, error) {
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var p models.Profil
	err := r.collection.
		FindOneAndUpdate(ctx, bson.M{"user_id": userID}, bson.M{"$set": set}, opts).
		Decode(&p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// Delete supprime le profil. Renvoie mongo.ErrNoDocuments si aucun document
// n'a été supprimé (mappé en 404 par le service).
func (r *ProfilRepository) Delete(ctx context.Context, userID string) error {
	res, err := r.collection.DeleteOne(ctx, bson.M{"user_id": userID})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}
