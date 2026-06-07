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
// créé (avec son id généré). Les compteurs sont posés à 0 explicitement.
func (s *PostService) CreatePost(ctx context.Context, authorID, content string) (*models.Post, error) {
	now := time.Now()
	post := &models.Post{
		AuthorID:      authorID,
		Content:       content,
		LikesCount:    0,
		CommentsCount: 0,
		CreatedAt:     now,
		UpdatedAt:     now,
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

// DeletePost supprime un post si l'acteur en a le droit, puis purge ses likes
// et commentaires (best-effort, pour ne pas laisser d'orphelins).
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
	if err := s.repo.Delete(ctx, oid); err != nil {
		return translateNotFound(err)
	}
	_ = s.repo.DeleteLikesByPost(ctx, id)
	_ = s.repo.DeleteCommentsByPost(ctx, id)
	return nil
}

// GetByProfile renvoie les posts d'un auteur, du plus récent au plus ancien.
func (s *PostService) GetByProfile(ctx context.Context, authorID string, limit, offset int64) ([]models.Post, error) {
	return s.repo.GetByProfile(ctx, authorID, clampLimit(limit), clampOffset(offset))
}

// GetFeed renvoie les posts d'un ensemble d'auteurs (fil « Abonnements »). Le
// front fournit les ids suivis (seul le user-service connaît le graphe) ; la
// sélection + le tri + la pagination sont faits côté DB ($in indexé). Liste
// vide → aucun post (pas de requête inutile).
func (s *PostService) GetFeed(ctx context.Context, authorIDs []string, limit, offset int64) ([]models.Post, error) {
	if len(authorIDs) == 0 {
		return []models.Post{}, nil
	}
	return s.repo.GetByAuthors(ctx, authorIDs, clampLimit(limit), clampOffset(offset))
}

// LikePost enregistre un like de actorID sur un post et renvoie le nombre de
// likes à jour. Idempotent : reliker ne double pas le compteur.
func (s *PostService) LikePost(ctx context.Context, id, actorID string) (int32, error) {
	oid, err := parseID(id)
	if err != nil {
		return 0, err
	}
	post, err := s.repo.Get(ctx, oid)
	if err != nil {
		return 0, translateNotFound(err)
	}
	created, err := s.repo.AddLike(ctx, id, actorID)
	if err != nil {
		return 0, err
	}
	if !created {
		return post.LikesCount, nil
	}
	updated, err := s.repo.IncCounter(ctx, oid, "likes_count", 1)
	if err != nil {
		return 0, err
	}
	return updated.LikesCount, nil
}

// UnlikePost retire le like de actorID et renvoie le nombre de likes à jour.
// Idempotent : déliker un post non liké ne décrémente pas.
func (s *PostService) UnlikePost(ctx context.Context, id, actorID string) (int32, error) {
	oid, err := parseID(id)
	if err != nil {
		return 0, err
	}
	post, err := s.repo.Get(ctx, oid)
	if err != nil {
		return 0, translateNotFound(err)
	}
	removed, err := s.repo.RemoveLike(ctx, id, actorID)
	if err != nil {
		return 0, err
	}
	if !removed {
		return post.LikesCount, nil
	}
	updated, err := s.repo.IncCounter(ctx, oid, "likes_count", -1)
	if err != nil {
		return 0, err
	}
	return updated.LikesCount, nil
}

// LikedPostIDs renvoie les ids des posts likés par actorID.
func (s *PostService) LikedPostIDs(ctx context.Context, actorID string) ([]string, error) {
	return s.repo.LikedPostIDs(ctx, actorID)
}

// PostLikers renvoie les ids des utilisateurs ayant liké un post.
func (s *PostService) PostLikers(ctx context.Context, id string) ([]string, error) {
	if _, err := parseID(id); err != nil {
		return nil, err
	}
	return s.repo.LikersByPost(ctx, id)
}

// CreateComment ajoute un commentaire (auteur dérivé du JWT) sur un post
// existant et incrémente son compteur.
func (s *PostService) CreateComment(ctx context.Context, postID, authorID, content string) (*models.Comment, error) {
	oid, err := parseID(postID)
	if err != nil {
		return nil, err
	}
	if _, err := s.repo.Get(ctx, oid); err != nil {
		return nil, translateNotFound(err)
	}
	now := time.Now()
	comment := &models.Comment{
		PostID:    postID,
		AuthorID:  authorID,
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.AddComment(ctx, comment); err != nil {
		return nil, err
	}
	if _, err := s.repo.IncCounter(ctx, oid, "comments_count", 1); err != nil {
		return nil, err
	}
	return comment, nil
}

// ListComments renvoie les commentaires d'un post (chronologiques, paginés).
func (s *PostService) ListComments(ctx context.Context, postID string, limit, offset int64) ([]models.Comment, error) {
	if _, err := parseID(postID); err != nil {
		return nil, err
	}
	return s.repo.ListComments(ctx, postID, clampLimit(limit), clampOffset(offset))
}

// DeleteComment supprime un commentaire si l'acteur en a le droit (auteur du
// commentaire ou modérateur/admin) et décrémente le compteur du post.
func (s *PostService) DeleteComment(ctx context.Context, commentID, actorID, actorRole string) error {
	coid, err := parseID(commentID)
	if err != nil {
		return err
	}
	comment, err := s.repo.GetComment(ctx, coid)
	if err != nil {
		return translateNotFound(err)
	}
	if !canAct(comment.AuthorID, actorID, actorRole) {
		return ErrForbidden
	}
	if err := s.repo.DeleteComment(ctx, coid); err != nil {
		return translateNotFound(err)
	}
	// Décrémente le compteur du post visé (id pris sur le commentaire, source
	// de vérité). Best-effort : un post déjà supprimé n'a plus de compteur.
	if poid, err := parseID(comment.PostID); err == nil {
		_, _ = s.repo.IncCounter(ctx, poid, "comments_count", -1)
	}
	return nil
}

// canModify : un post n'est modifiable/supprimable que par son auteur ou par un
// modérateur / administrateur. Fonction PURE (testée unitairement).
func canModify(post *models.Post, actorID, actorRole string) bool {
	if post == nil {
		return false
	}
	return canAct(post.AuthorID, actorID, actorRole)
}

// canAct : règle d'autorisation commune (posts ET commentaires) — l'auteur, un
// modérateur ou un admin peut agir. Fonction PURE.
func canAct(authorID, actorID, actorRole string) bool {
	return authorID == actorID ||
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
