// Package service porte la logique métier du service user : validation,
// mapping des erreurs DB vers des erreurs métier, provisioning paresseux.
package service

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/webdad/user-service/internal/models"
	"github.com/webdad/user-service/internal/repository"
)

// Erreurs métier (mappées vers des codes HTTP par les handlers).
var (
	ErrUserNotFound  = errors.New("utilisateur introuvable")
	ErrUsernameTaken = errors.New("nom d'utilisateur déjà utilisé")
)

// UserService regroupe les dépendances.
type UserService struct {
	repo *repository.UserRepository
}

// New construit le service.
func New(repo *repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

// Create crée un utilisateur avec l'id fourni (= credentials.id du JWT).
func (s *UserService) Create(id, username, displayName string) (*models.User, error) {
	u, err := s.repo.Create(id, username, displayName)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrUsernameTaken
		}
		return nil, fmt.Errorf("création utilisateur : %w", err)
	}
	return u, nil
}

// GetByID retourne un utilisateur ou ErrUserNotFound.
func (s *UserService) GetByID(id string) (*models.User, error) {
	u, err := s.repo.GetByID(id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lecture utilisateur : %w", err)
	}
	return u, nil
}

// List retourne une page d'utilisateurs.
func (s *UserService) List(limit, offset int) ([]models.User, error) {
	users, err := s.repo.List(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("liste utilisateurs : %w", err)
	}
	return users, nil
}

// Update modifie l'utilisateur (champs nil = inchangés).
func (s *UserService) Update(id string, username, displayName *string) (*models.User, error) {
	u, err := s.repo.Update(id, username, displayName)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrUsernameTaken
		}
		return nil, fmt.Errorf("mise à jour utilisateur : %w", err)
	}
	return u, nil
}

// SoftDelete désactive un compte.
func (s *UserService) SoftDelete(id string) error {
	err := s.repo.SoftDelete(id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrUserNotFound
	}
	if err != nil {
		return fmt.Errorf("désactivation utilisateur : %w", err)
	}
	return nil
}

// ProvisionFromClaims garantit l'existence de l'enregistrement user pour un
// utilisateur authentifié (provisioning paresseux). Au premier appel
// authentifié, la ligne est créée à partir des claims du JWT ; ensuite elle
// est simplement retournée. C'est ce qui relie « inscription (auth) » à
// « existence du user » sans couplage direct entre les services.
func (s *UserService) ProvisionFromClaims(id, email string) (*models.User, error) {
	u, err := s.repo.Upsert(id, usernameFromEmail(email), "")
	if err != nil {
		return nil, fmt.Errorf("provisioning utilisateur : %w", err)
	}
	return u, nil
}

// usernameFromEmail dérive un username par défaut depuis l'email (partie
// locale). Provisoire : l'utilisateur pourra le changer via PATCH /users/me.
func usernameFromEmail(email string) string {
	if i := strings.IndexByte(email, '@'); i > 0 {
		return email[:i]
	}
	return email
}

// isUniqueViolation détecte l'erreur PostgreSQL 23505 (contrainte UNIQUE)
// sans dépendre directement du type concret du driver.
func isUniqueViolation(err error) bool {
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.SQLState() == "23505"
	}
	return false
}
