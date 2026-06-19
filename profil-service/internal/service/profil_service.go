// Package service porte la logique métier du service profil : provisioning
// paresseux du profil, validation, règles d'édition (birth_date set-once,
// cooldown du display_name), mapping des erreurs.
package service

import (
	"context"
	"errors"
	"log/slog"
	"regexp"
	"strings"
	"time"
	"unicode"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/webdad/profil-service/internal/models"
	"github.com/webdad/profil-service/internal/repository"
)

// Erreurs métier (mappées vers des codes HTTP par les handlers).
var (
	// ErrProfilNotFound : aucun profil pour cet utilisateur → 404.
	ErrProfilNotFound = errors.New("profil introuvable")
	// ErrProfilExists : profil déjà présent (POST sur un profil existant) → 409.
	ErrProfilExists = errors.New("profil déjà existant")
	// ErrBirthDateLocked : tentative de modifier une date de naissance déjà
	// posée (set-once) → 409.
	ErrBirthDateLocked = errors.New("la date de naissance ne peut pas être modifiée")
	// ErrDisplayNameCooldown : changement de display_name trop rapproché du
	// précédent (cooldown actif) → 429.
	ErrDisplayNameCooldown = errors.New("nom d'affichage modifié trop récemment")
	// ErrInvalidDisplayName : le nom contient un caractère autre qu'une lettre,
	// un chiffre, un espace, un tiret, un underscore ou un point → 400.
	ErrInvalidDisplayName = errors.New("le nom ne peut contenir que des lettres, chiffres, espaces, tirets, underscores et points")
)

// ProfilService regroupe les dépendances et la config métier.
type ProfilService struct {
	repo                *repository.ProfilRepository
	displayNameCooldown time.Duration
	follows             FollowChecker
}

const onlineGracePeriod = 45 * time.Second

// FollowChecker interroge le graphe social dans user-service : vérifie la
// relation follower -> following et accepte en masse les demandes en attente
// (quand un profil privé repasse public).
type FollowChecker interface {
	IsFollowing(ctx context.Context, followerID, followingID string) (bool, error)
	AcceptAllFollowRequests(ctx context.Context, ownerID string) error
}

// Option configure les dépendances optionnelles du service.
type Option func(*ProfilService)

// WithFollowChecker active les règles de lecture liées aux profils privés.
func WithFollowChecker(checker FollowChecker) Option {
	return func(s *ProfilService) {
		s.follows = checker
	}
}

// New construit le service. displayNameCooldown=0 désactive l'enforcement du
// cooldown (le timestamp de changement reste enregistré dans tous les cas).
func New(repo *repository.ProfilRepository, displayNameCooldown time.Duration, opts ...Option) *ProfilService {
	s := &ProfilService{repo: repo, displayNameCooldown: displayNameCooldown}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// GetByUserID retourne le profil public d'un utilisateur (404 si absent).
func (s *ProfilService) GetByUserID(ctx context.Context, userID string) (*models.Profil, error) {
	p, err := s.repo.GetByUserID(ctx, userID)
	p, err = mapGet(p, err)
	if err != nil {
		return nil, err
	}
	normalizeActivity(p, time.Now().UTC())
	return p, nil
}

// Search retourne les profils dont le display_name contient `term` (insensible
// à la casse). Un terme vide renvoie une liste vide (pas de balayage complet).
// Les métacaractères regex de `term` sont neutralisés (recherche littérale).
func (s *ProfilService) Search(ctx context.Context, term string, limit int64) ([]models.Profil, error) {
	term = strings.TrimSpace(term)
	if term == "" {
		return []models.Profil{}, nil
	}
	profils, err := s.repo.SearchByDisplayName(ctx, regexp.QuoteMeta(term), limit)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	for i := range profils {
		normalizeActivity(&profils[i], now)
	}
	return profils, nil
}

// Create est l'UNIQUE voie de création d'un profil (POST /profils). L'id vient
// du JWT ; display_name est obligatoire (le BFF y met le username au register).
// birth_date et gender peuvent être posés dès l'inscription. Aucune valeur n'est
// dérivée de l'email, et la lecture (GET /profils/me) ne crée RIEN : un profil
// n'existe que parce qu'il a été explicitement créé ici (ou par le seed admin).
// 409 si le profil existe déjà.
func (s *ProfilService) Create(ctx context.Context, userID string, req models.CreateProfilRequest) (*models.Profil, error) {
	displayName := strings.TrimSpace(req.DisplayName)
	if !validDisplayName(displayName) {
		return nil, ErrInvalidDisplayName
	}
	now := time.Now().UTC()
	p := &models.Profil{
		UserID:             userID,
		DisplayName:        displayName,
		BirthDate:          req.BirthDate,
		CreatedAt:          now,
		UpdatedAt:          now,
		Visibility:         models.VisibilityPublic,
		LikesVisibility:    models.VisibilityPublic,
		ActivityVisibility: models.VisibilityPublic,
	}
	if req.Gender != nil {
		p.Gender = *req.Gender
	}
	err := s.repo.Insert(ctx, p)
	if err == nil {
		return p, nil
	}
	if mongo.IsDuplicateKeyError(err) {
		return nil, ErrProfilExists
	}
	return nil, err
}

// Update applique une modification partielle au profil de l'utilisateur
// courant. Charge le profil actuel (404 si absent), applique les règles
// (birth_date set-once, cooldown display_name) puis persiste.
func (s *ProfilService) Update(ctx context.Context, userID string, req models.UpdateProfilRequest) (*models.Profil, error) {
	current, err := s.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	set, err := planUpdate(current, req, time.Now().UTC(), s.displayNameCooldown)
	if err != nil {
		return nil, err
	}
	if len(set) == 1 { // uniquement updated_at → rien à changer, on renvoie l'existant
		return current, nil
	}

	p, err := s.repo.Update(ctx, userID, set)
	if err == nil && p != nil &&
		current.Visibility == models.VisibilityPrivate &&
		p.Visibility == models.VisibilityPublic &&
		s.follows != nil {
		// Privé → public : on accepte d'un coup les demandes d'abonnement en
		// attente. Best-effort (le changement de visibilité, lui, a réussi) :
		// une panne user-service ne doit pas faire échouer le PATCH.
		slog.Info("profil privé → public : acceptation des demandes en attente", "user_id", userID)
		if aerr := s.follows.AcceptAllFollowRequests(ctx, userID); aerr != nil {
			slog.Warn("acceptation en masse des demandes échouée",
				"user_id", userID, "error", aerr)
		}
	}
	return mapGet(p, err)
}

// TouchActivity enregistre une vraie entrée en session. Le champ peut rester
// privé côté lecture publique selon ActivityVisibility / Visibility.
func (s *ProfilService) TouchActivity(ctx context.Context, userID string, online bool) (*models.Profil, error) {
	now := time.Now().UTC()
	p, err := s.repo.Update(ctx, userID, bson.M{
		"last_login_at": now,
		"is_online":     online,
		"updated_at":    now,
	})
	return mapGet(p, err)
}

// CanViewActivity applique la barrière de lecture de l'activité :
// préférence publique + profil public, ou propriétaire, ou abonné accepté.
func (s *ProfilService) CanViewActivity(ctx context.Context, viewerID string, profil *models.Profil) bool {
	if profil == nil || profil.ActivityVisibility != models.VisibilityPublic {
		return false
	}
	if viewerID != "" && viewerID == profil.UserID {
		return true
	}
	if profil.Visibility != models.VisibilityPrivate {
		return true
	}
	if viewerID == "" || s.follows == nil {
		return false
	}
	ok, err := s.follows.IsFollowing(ctx, viewerID, profil.UserID)
	return err == nil && ok
}

// Delete supprime le profil d'un utilisateur (réservé admin, vérifié en amont).
func (s *ProfilService) Delete(ctx context.Context, userID string) error {
	err := s.repo.Delete(ctx, userID)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return ErrProfilNotFound
	}
	return err
}

// planUpdate construit le $set Mongo d'un PATCH à partir du profil courant et
// de la requête. Fonction PURE (pas d'I/O) → testable unitairement. Applique :
//   - birth_date set-once : autorisée tant que vide ; rejet si on tente de la
//     changer une fois posée (valeur identique = no-op toléré) ;
//   - display_name : si la valeur change réellement, on enregistre
//     display_name_changed_at=now ; si un cooldown>0 est actif et que le
//     dernier changement est trop récent, on refuse.
//
// updated_at est toujours posé (le set contient donc au minimum cette clé).
func planUpdate(current *models.Profil, req models.UpdateProfilRequest, now time.Time, cooldown time.Duration) (bson.M, error) {
	set := bson.M{"updated_at": now}

	if req.DisplayName != nil {
		displayName := strings.TrimSpace(*req.DisplayName)
		if displayName != current.DisplayName {
			if !validDisplayName(displayName) {
				return nil, ErrInvalidDisplayName
			}
			if cooldown > 0 && current.DisplayNameChangedAt != nil &&
				now.Sub(*current.DisplayNameChangedAt) < cooldown {
				return nil, ErrDisplayNameCooldown
			}
			set["display_name"] = displayName
			set["display_name_changed_at"] = now
		}
	}

	if req.BirthDate != nil {
		switch {
		case current.BirthDate == nil:
			set["birth_date"] = *req.BirthDate // 1er renseignement autorisé
		case !req.BirthDate.Equal(*current.BirthDate):
			return nil, ErrBirthDateLocked // déjà posée + valeur différente → refus
		}
	}

	if req.Bio != nil {
		set["bio"] = *req.Bio
	}
	if req.AvatarURL != nil {
		set["avatar_url"] = *req.AvatarURL
	}
	if req.BannerURL != nil {
		set["banner_url"] = *req.BannerURL
	}
	if req.Website != nil {
		set["website"] = *req.Website
	}
	if req.Location != nil {
		set["location"] = *req.Location
	}
	if req.Gender != nil {
		set["gender"] = *req.Gender
	}
	if req.Nationality != nil {
		set["nationality"] = strings.ToUpper(strings.TrimSpace(*req.Nationality))
	}
	if req.Visibility != nil {
		if *req.Visibility != current.Visibility {
			set["visibility"] = *req.Visibility
		}
	}
	if req.LikesVisibility != nil && *req.LikesVisibility != current.LikesVisibility {
		set["likes_visibility"] = *req.LikesVisibility
	}
	if req.ActivityVisibility != nil && *req.ActivityVisibility != current.ActivityVisibility {
		set["activity_visibility"] = *req.ActivityVisibility
	}

	return set, nil
}

func validDisplayName(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsMark(r) || unicode.IsNumber(r) || r == ' ' || r == '-' || r == '_' || r == '.' {
			continue
		}
		return false
	}
	return true
}

// mapGet mappe mongo.ErrNoDocuments vers ErrProfilNotFound.
func mapGet(p *models.Profil, err error) (*models.Profil, error) {
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrProfilNotFound
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

func normalizeActivity(p *models.Profil, now time.Time) {
	if p == nil || !p.IsOnline || p.LastLoginAt == nil {
		return
	}
	if now.Sub(*p.LastLoginAt) > onlineGracePeriod {
		p.IsOnline = false
	}
}
