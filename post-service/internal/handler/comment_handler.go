package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/webdad/post-service/internal/logging"
	"github.com/webdad/post-service/internal/middleware"
	"github.com/webdad/post-service/internal/models"
	"github.com/webdad/post-service/internal/service"
)

type CommentHandler struct {
	service *service.PostService
	name    string
}

func NewCommentHandler(svc *service.PostService, serviceName string) *CommentHandler {
	return &CommentHandler{
		service: svc,
		name:    serviceName,
	}
}

// ListCommentsByAuthor : GET /posts/comments?author_id=<id> — commentaires écrits
// par un utilisateur, enrichis du post parent visible (onglet « Réponses » du profil).
// JWT optionnel : utilisé pour la barrière de visibilité sur le post parent.
// @Summary     Réponses écrites par un utilisateur (onglet profil)
// @Tags        comments
// @Produce     json
// @Param       author_id query string true  "ID de l'auteur des commentaires"
// @Param       limit     query int    false "Nb résultats"
// @Param       offset    query int    false "Décalage"
// @Success     200 {array} models.CommentWithPost
// @Failure     400 {object} map[string]string
// @Router      /posts/comments [get]
func (h *CommentHandler) ListCommentsByAuthor(c *gin.Context) {
	authorID := c.Query("author_id")
	if authorID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "author_id requis"})
		return
	}
	viewerID := ""
	if claims, ok := middleware.ClaimsFrom(c); ok {
		viewerID = claims.UserID
	}
	results, err := h.service.ListCommentsByAuthor(c.Request.Context(), authorID, viewerID, pageLimit(c), pageOffset(c))
	if err != nil {
		respondPostError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": results})
}

// CommentStats : GET /posts/comments/stats?ids=a,b,c — compteurs de likes des
// commentaires demandés, pour le rafraîchissement périodique des cœurs côté
// front sans recharger tout le thread.
// @Summary     Compteurs de commentaires (rafraîchissement)
// @Tags        comments
// @Produce     json
// @Param       ids query string true "IDs de commentaires séparés par des virgules (max 100)"
// @Success     200 {array} models.CommentStat
// @Failure     500 {object} map[string]string
// @Router      /posts/comments/stats [get]
func (h *CommentHandler) CommentStats(c *gin.Context) {
	ids := splitIDs(c.Query("ids"))
	if len(ids) == 0 {
		c.JSON(http.StatusOK, gin.H{"data": []models.CommentStat{}})
		return
	}
	stats, err := h.service.CommentStats(c.Request.Context(), ids)
	if err != nil {
		respondPostError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": stats})
}

// ListPostComments : GET /posts/:id/comments (public) — commentaires RACINE,
// chronologiques, paginés (les réponses sont chargées via ListCommentReplies).
// @Summary     Commentaires d'un post
// @Tags        comments
// @Produce     json
// @Param       id     path  string true  "Post ID"
// @Param       limit  query int    false "Nb résultats"
// @Param       offset query int    false "Décalage"
// @Success     200 {array} models.Comment
// @Failure     404 {object} map[string]string
// @Router      /posts/{id}/comments [get]
func (h *CommentHandler) ListPostComments(c *gin.Context) {
	viewerID := ""
	if claims, ok := middleware.ClaimsFrom(c); ok {
		viewerID = claims.UserID
	}
	comments, err := h.service.ListComments(c.Request.Context(), c.Param("id"), viewerID, pageLimit(c), pageOffset(c))
	if err != nil {
		respondPostError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": comments})
}

// ListCommentReplies : GET /posts/:id/comments/:commentId/replies (public) —
// réponses d'un commentaire, chronologiques, paginées.
// @Summary     Réponses à un commentaire
// @Tags        comments
// @Produce     json
// @Param       id        path  string true  "Post ID"
// @Param       commentId path  string true  "Comment ID"
// @Param       limit     query int    false "Nb résultats"
// @Param       offset    query int    false "Décalage"
// @Success     200 {array} models.Comment
// @Failure     404 {object} map[string]string
// @Router      /posts/{id}/comments/{commentId}/replies [get]
func (h *CommentHandler) ListCommentReplies(c *gin.Context) {
	viewerID := ""
	if claims, ok := middleware.ClaimsFrom(c); ok {
		viewerID = claims.UserID
	}
	replies, err := h.service.ListReplies(c.Request.Context(), c.Param("commentId"), viewerID, pageLimit(c), pageOffset(c))
	if err != nil {
		respondPostError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": replies})
}

// CreatPostComment : POST /posts/:id/comments — l'auteur est dérivé du JWT.
// @Summary     Commenter un post
// @Tags        comments
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id   path  string                        true "Post ID"
// @Param       body body  models.CreateCommentRequest   true "Contenu texte et/ou médias + parent_id optionnel"
// @Success     201 {object} models.Comment
// @Failure     400 {object} map[string]string
// @Failure     401 {object} map[string]string
// @Failure     403 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /posts/{id}/comments [post]
func (h *CommentHandler) CreatPostComment(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	var req models.CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}
	if strings.TrimSpace(req.Content) == "" && len(req.Media) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "contenu ou média requis"})
		return
	}

	comment, err := h.service.CreateComment(c.Request.Context(), c.Param("id"), claims.UserID, claims.Role, strings.TrimSpace(req.Content), req.ParentID, req.Media)
	if err != nil {
		respondPostError(c, err)
		return
	}
	logging.FromGin(c).Info("commentaire créé", "post_id", c.Param("id"), "comment_id", comment.ID)
	c.JSON(http.StatusCreated, gin.H{"data": comment})
}

// DeletePostComment : DELETE /posts/:id/comments/:commentId — auteur du
// commentaire (ou modérateur/admin).
// @Summary     Supprimer un commentaire (auteur/modérateur/admin)
// @Tags        comments
// @Security    BearerAuth
// @Param       id        path string true "Post ID"
// @Param       commentId path string true "Comment ID"
// @Success     204
// @Failure     401 {object} map[string]string
// @Failure     403 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /posts/{id}/comments/{commentId} [delete]
func (h *CommentHandler) DeletePostComment(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	if err := h.service.DeleteComment(c.Request.Context(), c.Param("commentId"), claims.UserID, claims.Role); err != nil {
		respondPostError(c, err)
		return
	}
	logging.FromGin(c).Info("commentaire supprimé", "comment_id", c.Param("commentId"))
	c.Status(http.StatusNoContent)
}

// LikeComment : POST /posts/:id/comments/:commentId/like — like du commentaire
// courant par l'utilisateur authentifié (idempotent).
// @Summary     Liker un commentaire
// @Tags        comments
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Post ID"
// @Param       commentId path string true "Comment ID"
// @Success     200 {object} map[string]string "likes_count"
// @Failure     401 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /posts/{id}/comments/{commentId}/like [post]
func (h *CommentHandler) LikeComment(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}
	count, err := h.service.LikeComment(c.Request.Context(), c.Param("id"), c.Param("commentId"), claims.UserID)
	if err != nil {
		respondPostError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"liked": true, "likes_count": count}})
}

// UnlikeComment : DELETE /posts/:id/comments/:commentId/like — retrait du like
// du commentaire courant (idempotent).
// @Summary     Retirer son like d'un commentaire
// @Tags        comments
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Post ID"
// @Param       commentId path string true "Comment ID"
// @Success     200 {object} map[string]string "likes_count"
// @Failure     401 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /posts/{id}/comments/{commentId}/like [delete]
func (h *CommentHandler) UnlikeComment(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}
	count, err := h.service.UnlikeComment(c.Request.Context(), c.Param("id"), c.Param("commentId"), claims.UserID)
	if err != nil {
		respondPostError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"liked": false, "likes_count": count}})
}
