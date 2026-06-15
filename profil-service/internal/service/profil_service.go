// Package service porte la logique métier du service profil : provisioning
// paresseux du profil, validation, règles d'édition (birth_date set-once,
// cooldown du display_name), mapping des erreurs.
package service

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

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
)

// ProfilService regroupe les dépendances et la config métier.
type ProfilService struct {
	repo                *repository.ProfilRepository
	displayNameCooldown time.Duration
}

// New construit le service. displayNameCooldown=0 désactive l'enforcement du
// cooldown (le timestamp de changement reste enregistré dans tous les cas).
func New(repo *repository.ProfilRepository, displayNameCooldown time.Duration) *ProfilService {
	return &ProfilService{repo: repo, displayNameCooldown: displayNameCooldown}
}

// GetByUserID retourne le profil public d'un utilisateur (404 si absent).
func (s *ProfilService) GetByUserID(ctx context.Context, userID string) (*models.Profil, error) {
	p, err := s.repo.GetByUserID(ctx, userID)
	return mapGet(p, err)
}

// Search retourne les profils dont le display_name contient `term` (insensible
// à la casse). Un terme vide renvoie une liste vide (pas de balayage complet).
// Les métacaractères regex de `term` sont neutralisés (recherche littérale).
func (s *ProfilService) Search(ctx context.Context, term string, limit int64) ([]models.Profil, error) {
	term = strings.TrimSpace(term)
	if term == "" {
		return []models.Profil{}, nil
	}
	return s.repo.SearchByDisplayName(ctx, regexp.QuoteMeta(term), limit)
}

// Create est l'UNIQUE voie de création d'un profil (POST /profils). L'id vient
// du JWT ; display_name est obligatoire (le BFF y met le username au register).
// birth_date et gender peuvent être posés dès l'inscription. Aucune valeur n'est
// dérivée de l'email, et la lecture (GET /profils/me) ne crée RIEN : un profil
// n'existe que parce qu'il a été explicitement créé ici (ou par le seed admin).
// 409 si le profil existe déjà.
func (s *ProfilService) Create(ctx context.Context, userID string, req models.CreateProfilRequest) (*models.Profil, error) {
	now := time.Now().UTC()
	p := &models.Profil{
		UserID:          userID,
		DisplayName:     strings.TrimSpace(req.DisplayName),
		BirthDate:       req.BirthDate,
		CreatedAt:       now,
		UpdatedAt:       now,
		Visibility:      models.VisibilityPublic,
		LikesVisibility: models.VisibilityPublic,
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
	return mapGet(p, err)
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

	if req.DisplayName != nil && *req.DisplayName != current.DisplayName {
		if cooldown > 0 && current.DisplayNameChangedAt != nil &&
			now.Sub(*current.DisplayNameChangedAt) < cooldown {
			return nil, ErrDisplayNameCooldown
		}
		set["display_name"] = *req.DisplayName
		set["display_name_changed_at"] = now
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
	if req.Visibility != nil && *req.Visibility != current.Visibility {
		set["visibility"] = *req.Visibility
	}
	if req.LikesVisibility != nil && *req.LikesVisibility != current.LikesVisibility {
		set["likes_visibility"] = *req.LikesVisibility
	}

	return set, nil
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
