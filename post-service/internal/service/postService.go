// Package service porte la logique métier du post-service : validation des
// identifiants, contrôle de propriété et traduction des erreurs du dépôt en
// erreurs métier (mappées vers des codes HTTP par les handlers).
package service

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/webdad/post-service/internal/models"
	"github.com/webdad/post-service/internal/repository"
)

// Erreurs métier — traduites en codes HTTP par les handlers.
var (
	// ErrPostNotFound : aucun post pour cet id → 404.
	ErrPostNotFound = errors.New("post introuvable")
	// ErrInvalidID : id de post mal formé (pas un ObjectID hexadécimal) → 400.
	ErrInvalidID = errors.New("identifiant de post invalide")
	// ErrForbidden : l'utilisateur n'est ni l'auteur ni un modérateur/admin → 403.
	ErrForbidden = errors.New("action non autorisée sur ce post")
)

// Bornes de pagination des listes.
const (
	DefaultLimit = 20
	MaxLimit     = 100
)

type PostService struct {
	repo *repository.PostRepository
}

func NewPostService(r *repository.PostRepository) *PostService {
	return &PostService{repo: r}
}

// CreatePost crée un post pour authorID (dérivé du JWT) et renvoie le document
// créé (avec son id généré).
func (s *PostService) CreatePost(ctx context.Context, authorID, content string) (*models.Post, error) {
	now := time.Now()
	post := &models.Post{
		AuthorID:  authorID,
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.Create(ctx, post); err != nil {
		return nil, err
	}
	return post, nil
}

// GetPosts renvoie le fil global, du plus récent au plus ancien, paginé.
func (s *PostService) GetPosts(ctx context.Context, limit, offset int64) ([]models.Post, error) {
	return s.repo.GetAll(ctx, clampLimit(limit), clampOffset(offset))
}

// GetPost renvoie un post par son id.
func (s *PostService) GetPost(ctx context.Context, id string) (*models.Post, error) {
	oid, err := parseID(id)
	if err != nil {
		return nil, err
	}
	post, err := s.repo.Get(ctx, oid)
	return post, translateNotFound(err)
}

// UpdatePost modifie le contenu d'un post si l'acteur en a le droit (auteur,
// modérateur ou admin).
func (s *PostService) UpdatePost(ctx context.Context, id, content, actorID, actorRole string) (*models.Post, error) {
	oid, err := parseID(id)
	if err != nil {
		return nil, err
	}
	post, err := s.repo.Get(ctx, oid)
	if err != nil {
		return nil, translateNotFound(err)
	}
	if !canModify(post, actorID, actorRole) {
		return nil, ErrForbidden
	}
	updated, err := s.repo.Update(ctx, oid, content)
	return updated, translateNotFound(err)
}

// DeletePost supprime un post si l'acteur en a le droit.
func (s *PostService) DeletePost(ctx context.Context, id, actorID, actorRole string) error {
	oid, err := parseID(id)
	if err != nil {
		return err
	}
	post, err := s.repo.Get(ctx, oid)
	if err != nil {
		return translateNotFound(err)
	}
	if !canModify(post, actorID, actorRole) {
		return ErrForbidden
	}
	return translateNotFound(s.repo.Delete(ctx, oid))
}

// GetByProfile renvoie les posts d'un auteur, du plus récent au plus ancien.
func (s *PostService) GetByProfile(ctx context.Context, authorID string, limit, offset int64) ([]models.Post, error) {
	return s.repo.GetByProfile(ctx, authorID, clampLimit(limit), clampOffset(offset))
}

// canModify : un post n'est modifiable/supprimable que par son auteur ou par un
// modérateur / administrateur. Fonction PURE (testée unitairement).
func canModify(post *models.Post, actorID, actorRole string) bool {
	if post == nil {
		return false
	}
	return post.AuthorID == actorID ||
		actorRole == models.RoleModerator ||
		actorRole == models.RoleAdmin
}

// parseID valide qu'un id est bien un ObjectID hexadécimal.
func parseID(id string) (bson.ObjectID, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return bson.ObjectID{}, ErrInvalidID
	}
	return oid, nil
}

// translateNotFound mappe l'absence de document Mongo vers ErrPostNotFound ;
// laisse les autres erreurs (et nil) intactes.
func translateNotFound(err error) error {
	if errors.Is(err, mongo.ErrNoDocuments) {
		return ErrPostNotFound
	}
	return err
}

// clampLimit borne la taille de page (défaut 20, max 100).
func clampLimit(limit int64) int64 {
	if limit <= 0 {
		return DefaultLimit
	}
	if limit > MaxLimit {
		return MaxLimit
	}
	return limit
}

// clampOffset interdit un décalage négatif.
func clampOffset(offset int64) int64 {
	if offset < 0 {
		return 0
	}
	return offset
}
