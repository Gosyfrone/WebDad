// Package service porte la logique métier du service profil : provisioning
// paresseux du profil, validation, mapping des erreurs.
//
// SQUELETTE : les méthodes délèguent au repository, qui renvoie pour l'instant
// ErrNotImplemented. La logique réelle (provisioning depuis les claims,
// édition) sera ajoutée dans une issue dédiée.
package service

import (
	"context"
	"errors"

	"github.com/webdad/profil-service/internal/models"
	"github.com/webdad/profil-service/internal/repository"
)

// Erreurs métier (mappées vers des codes HTTP par les handlers).
var (
	// ErrNotImplemented : route encore au stade squelette → 501.
	ErrNotImplemented = repository.ErrNotImplemented
	// ErrProfilNotFound : aucun profil pour cet utilisateur → 404.
	ErrProfilNotFound = errors.New("profil introuvable")
)

// ProfilService regroupe les dépendances.
type ProfilService struct {
	repo *repository.ProfilRepository
}

// New construit le service.
func New(repo *repository.ProfilRepository) *ProfilService {
	return &ProfilService{repo: repo}
}

// GetByUserID retourne le profil public d'un utilisateur.
func (s *ProfilService) GetByUserID(ctx context.Context, userID string) (*models.Profil, error) {
	return s.repo.GetByUserID(ctx, userID)
}

// ProvisionFromClaims garantit l'existence du profil de l'utilisateur courant
// (provisioning paresseux depuis le JWT), puis le retourne.
func (s *ProfilService) ProvisionFromClaims(ctx context.Context, userID, email string) (*models.Profil, error) {
	return s.repo.GetByUserID(ctx, userID)
}

// Create crée le profil de l'utilisateur courant (id issu du JWT).
func (s *ProfilService) Create(ctx context.Context, userID string) (*models.Profil, error) {
	return s.repo.Upsert(ctx, &models.Profil{UserID: userID})
}

// Update applique une modification partielle au profil de l'utilisateur courant.
func (s *ProfilService) Update(ctx context.Context, userID string, req models.UpdateProfilRequest) (*models.Profil, error) {
	return s.repo.Update(ctx, userID, req)
}

// Delete supprime le profil d'un utilisateur (réservé admin, vérifié en amont).
func (s *ProfilService) Delete(ctx context.Context, userID string) error {
	return s.repo.Delete(ctx, userID)
}
