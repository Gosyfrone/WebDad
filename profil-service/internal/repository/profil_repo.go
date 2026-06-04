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
	collection *mongo.Collection
}

// NewProfilRepository construit le repository sur la collection `profiles`.
func NewProfilRepository(db *mongo.Database) *ProfilRepository {
	return &ProfilRepository{collection: db.Collection("profiles")}
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
