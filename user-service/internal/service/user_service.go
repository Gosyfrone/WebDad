// Package service porte la logique métier du service user : validation,
// mapping des erreurs DB vers des erreurs métier, provisioning paresseux,
// graphe social.
package service

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/webdad/user-service/internal/models"
	"github.com/webdad/user-service/internal/repository"
)

// Erreurs métier (mappées vers des codes HTTP par les handlers).
var (
	ErrUserNotFound    = errors.New("utilisateur introuvable")
	ErrUsernameTaken   = errors.New("nom d'utilisateur déjà utilisé")
	ErrInvalidUsername = errors.New("nom d'utilisateur invalide (3-50 caractères : lettres, chiffres, _)")
	ErrSelfFollow      = errors.New("impossible de se suivre soi-même")
)

// usernamePattern : charset autorisé pour un username (3-50, alphanum + _).
var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]{3,50}$`)

// reservedUsernames : handles interdits (mots structurants / sensibles).
var reservedUsernames = map[string]bool{
	"me": true, "admin": true, "root": true, "users": true,
	"by-username": true, "null": true, "undefined": true,
}

// UserService regroupe les dépendances.
type UserService struct {
	repo *repository.UserRepository
}

// New construit le service.
func New(repo *repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

// Create crée un utilisateur avec l'id fourni (= credentials.id du JWT).
func (s *UserService) Create(id, username string) (*models.User, error) {
	if err := validateUsername(username); err != nil {
		return nil, err
	}
	u, err := s.repo.Create(id, username)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrUsernameTaken
		}
		return nil, fmt.Errorf("création utilisateur : %w", err)
	}
	return u, nil
}

// GetDetailsByID retourne un utilisateur + compteurs ou ErrUserNotFound.
func (s *UserService) GetDetailsByID(id string) (*models.UserDetails, error) {
	d, err := s.repo.GetDetailsByID(id)
	return mapDetails(d, err)
}

// GetDetailsByUsername retourne un utilisateur + compteurs par son handle.
func (s *UserService) GetDetailsByUsername(username string) (*models.UserDetails, error) {
	d, err := s.repo.GetDetailsByUsername(username)
	return mapDetails(d, err)
}

// List retourne une page d'utilisateurs actifs.
func (s *UserService) List(limit, offset int) ([]models.User, error) {
	users, err := s.repo.List(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("liste utilisateurs : %w", err)
	}
	return users, nil
}

// Update modifie l'utilisateur (champs nil = inchangés). Valide le username
// s'il est fourni.
func (s *UserService) Update(id string, username *string) (*models.User, error) {
	if username != nil {
		if err := validateUsername(*username); err != nil {
			return nil, err
		}
	}
	u, err := s.repo.Update(id, username)
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
// utilisateur authentifié (provisioning paresseux), puis retourne sa vue
// détaillée. Au premier appel authentifié, la ligne est créée à partir des
// claims du JWT ; ensuite elle est simplement retournée. C'est ce qui relie
// « inscription (auth) » à « existence du user » sans couplage entre services.
func (s *UserService) ProvisionFromClaims(id, email string) (*models.UserDetails, error) {
	exists, err := s.repo.ExistsByID(id)
	if err != nil {
		return nil, fmt.Errorf("vérification utilisateur : %w", err)
	}
	if !exists {
		if err := s.provision(id, email); err != nil {
			return nil, err
		}
	}
	return s.GetDetailsByID(id)
}

// provision insère la ligne user. Le username dérivé de l'email peut entrer
// en collision avec un autre compte (deux emails `bob@x`/`bob@y` → `bob`) :
// on essaie donc une série de candidats de plus en plus spécifiques jusqu'à
// en trouver un libre.
func (s *UserService) provision(id, email string) error {
	short := strings.ReplaceAll(id, "-", "")
	if len(short) > 8 {
		short = short[:8]
	}
	base := defaultUsername(email)
	candidates := []string{base, base + "_" + short, "user_" + short}

	var lastErr error
	for _, c := range candidates {
		_, err := s.repo.Create(id, c)
		if err == nil {
			return nil
		}
		if isUniqueViolation(err) {
			// Course possible : la ligne (id) a pu être créée entre-temps.
			if exists, _ := s.repo.ExistsByID(id); exists {
				return nil
			}
			lastErr = ErrUsernameTaken
			continue // username pris → candidat suivant
		}
		return fmt.Errorf("provisioning utilisateur : %w", err)
	}
	return fmt.Errorf("provisioning utilisateur : %w", lastErr)
}

// ─── Graphe social (follows) ──────────────────────────────────────────────

// Follow : `followerID` (utilisateur authentifié, email pour provisioning si
// besoin) suit `followingID`. Idempotent. Provisionne d'abord le follower
// (il peut ne jamais avoir tapé /users/me) puis vérifie l'existence de la cible.
func (s *UserService) Follow(followerID, followerEmail, followingID string) error {
	if followerID == followingID {
		return ErrSelfFollow
	}
	if _, err := s.ProvisionFromClaims(followerID, followerEmail); err != nil {
		return err
	}
	exists, err := s.repo.ExistsByID(followingID)
	if err != nil {
		return fmt.Errorf("vérification cible : %w", err)
	}
	if !exists {
		return ErrUserNotFound
	}
	if err := s.repo.Follow(followerID, followingID); err != nil {
		return fmt.Errorf("follow : %w", err)
	}
	return nil
}

// Unfollow supprime la relation (idempotent).
func (s *UserService) Unfollow(followerID, followingID string) error {
	if err := s.repo.Unfollow(followerID, followingID); err != nil {
		return fmt.Errorf("unfollow : %w", err)
	}
	return nil
}

// ListFollowers retourne les abonnés de `id` (404 si l'utilisateur n'existe pas).
func (s *UserService) ListFollowers(id string, limit, offset int) ([]models.User, error) {
	if err := s.requireExists(id); err != nil {
		return nil, err
	}
	users, err := s.repo.ListFollowers(id, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("liste followers : %w", err)
	}
	return users, nil
}

// ListFollowing retourne les abonnements de `id` (404 si l'utilisateur n'existe pas).
func (s *UserService) ListFollowing(id string, limit, offset int) ([]models.User, error) {
	if err := s.requireExists(id); err != nil {
		return nil, err
	}
	users, err := s.repo.ListFollowing(id, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("liste following : %w", err)
	}
	return users, nil
}

// requireExists renvoie ErrUserNotFound si l'utilisateur n'existe pas.
func (s *UserService) requireExists(id string) error {
	exists, err := s.repo.ExistsByID(id)
	if err != nil {
		return fmt.Errorf("vérification utilisateur : %w", err)
	}
	if !exists {
		return ErrUserNotFound
	}
	return nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────

// mapDetails mappe sql.ErrNoRows vers ErrUserNotFound.
func mapDetails(d *models.UserDetails, err error) (*models.UserDetails, error) {
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lecture utilisateur : %w", err)
	}
	return d, nil
}

// validateUsername vérifie le charset et les mots réservés.
func validateUsername(username string) error {
	if !usernamePattern.MatchString(username) {
		return ErrInvalidUsername
	}
	if reservedUsernames[strings.ToLower(username)] {
		return ErrInvalidUsername
	}
	return nil
}

// defaultUsername dérive un username valide depuis l'email (partie locale,
// nettoyée au charset autorisé). Fallback "user" si le résultat est trop court.
func defaultUsername(email string) string {
	local := email
	if i := strings.IndexByte(email, '@'); i > 0 {
		local = email[:i]
	}
	var b strings.Builder
	for _, r := range strings.ToLower(local) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
		}
	}
	cleaned := b.String()
	if len(cleaned) < 3 {
		return "user"
	}
	if len(cleaned) > 50 {
		cleaned = cleaned[:50]
	}
	return cleaned
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
