// Package service porte la logique métier du service user : validation,
// mapping des erreurs DB vers des erreurs métier, provisioning paresseux,
// graphe social.
package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/webdad/user-service/internal/client"
	"github.com/webdad/user-service/internal/models"
	"github.com/webdad/user-service/internal/repository"
)

// Erreurs métier (mappées vers des codes HTTP par les handlers).
var (
	ErrUserNotFound          = errors.New("utilisateur introuvable")
	ErrUsernameTaken         = errors.New("nom d'utilisateur déjà utilisé")
	ErrInvalidUsername       = errors.New("nom d'utilisateur invalide (3-50 caractères : lettres, chiffres, _)")
	ErrSelfFollow            = errors.New("impossible de se suivre soi-même")
	ErrUsernameCooldown      = errors.New("nom d'utilisateur modifié trop récemment")
	ErrFollowRequestNotFound = errors.New("demande de suivi introuvable")
)

const (
	FollowStatusFollowing = "following"
	FollowStatusPending   = "pending"
)

// usernamePattern : charset autorisé pour un username (3-50, alphanum + _).
var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]{3,50}$`)

// reservedUsernames : handles interdits (mots structurants / sensibles).
var reservedUsernames = map[string]bool{
	"me": true, "admin": true, "root": true, "users": true,
	"by-username": true, "null": true, "undefined": true,
	"search": true, "suggestions": true,
}

// UserService regroupe les dépendances et la config métier.
type UserService struct {
	repo             *repository.UserRepository
	usernameCooldown time.Duration
	profilClient     profilVisibilityClient
	notification     notificationEmitter
}

type profilVisibilityClient interface {
	Visibility(ctx context.Context, userID string) (string, error)
}

type notificationEmitter interface {
	Emit(ev client.Event)
}

type Option func(*UserService)

func WithProfilClient(c profilVisibilityClient) Option {
	return func(s *UserService) { s.profilClient = c }
}

func WithNotificationClient(c notificationEmitter) Option {
	return func(s *UserService) { s.notification = c }
}

// New construit le service. usernameCooldown=0 désactive l'enforcement du
// cooldown (le timestamp de changement reste enregistré dans tous les cas).
func New(repo *repository.UserRepository, usernameCooldown time.Duration, opts ...Option) *UserService {
	s := &UserService{repo: repo, usernameCooldown: usernameCooldown}
	for _, opt := range opts {
		opt(s)
	}
	return s
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

// Search retourne les utilisateurs dont le username contient `term`. Un terme
// vide renvoie une liste vide (pas de balayage complet de la table).
func (s *UserService) Search(term string, limit, offset int) ([]models.User, error) {
	term = strings.TrimSpace(term)
	if term == "" {
		return []models.User{}, nil
	}
	users, err := s.repo.Search(term, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("recherche utilisateurs : %w", err)
	}
	return users, nil
}

// Suggestions retourne les utilisateurs les plus suivis (« Qui suivre »).
func (s *UserService) Suggestions(limit, offset int) ([]models.User, error) {
	users, err := s.repo.ListByFollowers(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("suggestions utilisateurs : %w", err)
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
		// Cooldown : refuse un changement effectif trop rapproché du précédent.
		// Désactivé si usernameCooldown=0 (le timestamp reste enregistré côté
		// repo dans tous les cas, pour servir de baseline future).
		if s.usernameCooldown > 0 {
			current, err := s.repo.GetByID(id)
			if errors.Is(err, sql.ErrNoRows) {
				return nil, ErrUserNotFound
			}
			if err != nil {
				return nil, fmt.Errorf("lecture utilisateur : %w", err)
			}
			if *username != current.Username && current.UsernameChangedAt != nil &&
				time.Since(*current.UsernameChangedAt) < s.usernameCooldown {
				return nil, ErrUsernameCooldown
			}
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
func (s *UserService) Follow(ctx context.Context, followerID, followerEmail, followingID string) (string, error) {
	if followerID == followingID {
		return "", ErrSelfFollow
	}
	if _, err := s.ProvisionFromClaims(followerID, followerEmail); err != nil {
		return "", err
	}
	exists, err := s.repo.ExistsByID(followingID)
	if err != nil {
		return "", fmt.Errorf("vérification cible : %w", err)
	}
	if !exists {
		return "", ErrUserNotFound
	}
	already, err := s.repo.IsFollowing(followerID, followingID)
	if err != nil {
		return "", fmt.Errorf("vérification follow : %w", err)
	}
	if already {
		return FollowStatusFollowing, nil
	}
	visibility := ""
	if s.profilClient != nil {
		visibility, err = s.profilClient.Visibility(ctx, followingID)
		if err != nil {
			return "", fmt.Errorf("vérification visibilité : %w", err)
		}
	}
	if visibility == client.VisibilityPrivate {
		if err := s.repo.RequestFollow(followerID, followingID); err != nil {
			return "", fmt.Errorf("demande follow : %w", err)
		}
		s.emitFollowRequest(followerID, followingID, false)
		return FollowStatusPending, nil
	}
	if err := s.repo.Follow(followerID, followingID); err != nil {
		return "", fmt.Errorf("follow : %w", err)
	}
	s.emitFollow(followerID, followingID)
	return FollowStatusFollowing, nil
}

// Unfollow supprime la relation (idempotent).
func (s *UserService) Unfollow(followerID, followingID string) error {
	if err := s.repo.Unfollow(followerID, followingID); err != nil {
		return fmt.Errorf("unfollow : %w", err)
	}
	s.emitFollowRequest(followerID, followingID, true)
	return nil
}

func (s *UserService) AcceptFollowRequest(ownerID, followerID string) error {
	accepted, err := s.repo.AcceptFollowRequest(followerID, ownerID)
	if err != nil {
		return fmt.Errorf("accept follow request : %w", err)
	}
	if !accepted {
		return ErrFollowRequestNotFound
	}
	s.emitFollowRequest(followerID, ownerID, true)
	s.emitFollowRequestAcceptConfirm(followerID, ownerID)
	s.emitFollowRequestDecision(ownerID, followerID, client.TypeFollowRequestAccepted)
	return nil
}

func (s *UserService) RejectFollowRequest(ownerID, followerID string) error {
	exists, err := s.repo.HasFollowRequest(followerID, ownerID)
	if err != nil {
		return fmt.Errorf("vérification follow request : %w", err)
	}
	if !exists {
		return ErrFollowRequestNotFound
	}
	if err := s.repo.DeleteFollowRequest(followerID, ownerID); err != nil {
		return fmt.Errorf("reject follow request : %w", err)
	}
	s.emitFollowRequest(followerID, ownerID, true)
	return nil
}

func (s *UserService) PendingFollowRequestIDs(followerID string) ([]string, error) {
	ids, err := s.repo.PendingFollowRequestIDs(followerID)
	if err != nil {
		return nil, fmt.Errorf("demandes follow pending : %w", err)
	}
	return ids, nil
}

func (s *UserService) emitFollowRequest(followerID, followingID string, retract bool) {
	if s.notification == nil {
		return
	}
	s.notification.Emit(client.Event{
		Type:        client.TypeFollowRequest,
		ActorID:     followerID,
		RecipientID: followingID,
		Retract:     retract,
	})
}

func (s *UserService) emitFollow(followerID, followingID string) {
	if s.notification == nil {
		return
	}
	s.notification.Emit(client.Event{
		Type:        client.TypeFollow,
		ActorID:     followerID,
		RecipientID: followingID,
	})
}

func (s *UserService) emitFollowRequestAcceptConfirm(followerID, ownerID string) {
	if s.notification == nil {
		return
	}
	s.notification.Emit(client.Event{
		Type:        client.TypeFollowRequestAcceptConfirm,
		ActorID:     followerID,
		RecipientID: ownerID,
	})
}

func (s *UserService) emitFollowRequestDecision(ownerID, followerID, eventType string) {
	if s.notification == nil {
		return
	}
	s.notification.Emit(client.Event{
		Type:        eventType,
		ActorID:     ownerID,
		RecipientID: followerID,
	})
}

// RemoveFollower supprime la relation followerID → currentUserID (idempotent).
func (s *UserService) RemoveFollower(currentUserID, followerID string) error {
	if currentUserID == followerID {
		return ErrSelfFollow
	}
	if err := s.repo.Unfollow(followerID, currentUserID); err != nil {
		return fmt.Errorf("retrait follower : %w", err)
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

func (s *UserService) IsFollowing(userId string, followingId string) bool {
	if err := s.requireExists(userId); err != nil {
		return false
	}
	if err := s.requireExists(followingId); err != nil {
		return false
	}
	exists, err := s.repo.IsFollowing(userId, followingId)
	if err != nil {
		return false
	}
	return exists
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
