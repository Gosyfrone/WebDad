// Package repository encapsule l'accès Mongo à la collection `profiles`.
//
// SQUELETTE : la structure et les signatures sont posées (chaîne
// handler→service→repository câblée), mais les corps renvoient encore
// ErrNotImplemented. L'implémentation réelle (lecture/écriture + provisioning)
// fera l'objet d'une issue dédiée.
package repository

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/webdad/profil-service/internal/models"
)

// ErrNotImplemented : sentinelle du squelette (mappée vers 501 par les
// handlers). Réexposée par le package service.
var ErrNotImplemented = errors.New("non implémenté")

// ProfilRepository : accès à la collection `profiles`.
type ProfilRepository struct {
	collection *mongo.Collection
}

// NewProfilRepository construit le repository sur la collection `profiles`.
func NewProfilRepository(db *mongo.Database) *ProfilRepository {
	return &ProfilRepository{collection: db.Collection("profiles")}
}

// GetByUserID retourne le profil d'un utilisateur (TODO).
func (r *ProfilRepository) GetByUserID(ctx context.Context, userID string) (*models.Profil, error) {
	return nil, ErrNotImplemented
}

// Upsert crée ou met à jour le profil d'un utilisateur (TODO).
func (r *ProfilRepository) Upsert(ctx context.Context, p *models.Profil) (*models.Profil, error) {
	return nil, ErrNotImplemented
}

// Update applique une modification partielle au profil d'un utilisateur (TODO).
func (r *ProfilRepository) Update(ctx context.Context, userID string, req models.UpdateProfilRequest) (*models.Profil, error) {
	return nil, ErrNotImplemented
}

// Delete supprime le profil d'un utilisateur (TODO).
func (r *ProfilRepository) Delete(ctx context.Context, userID string) error {
	return ErrNotImplemented
}
