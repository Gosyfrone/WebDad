package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

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
	comments, err := h.service.ListComments(c.Request.Context(), c.Param("id"), pageLimit(c), pageOffset(c))
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
	replies, err := h.service.ListReplies(c.Request.Context(), c.Param("commentId"), pageLimit(c), pageOffset(c))
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
// @Param       body body  models.CreateCommentRequest   true "Contenu + parent_id optionnel"
// @Success     201 {object} models.Comment
// @Failure     400 {object} map[string]string
// @Failure     401 {object} map[string]string
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

	comment, err := h.service.CreateComment(c.Request.Context(), c.Param("id"), claims.UserID, strings.TrimSpace(req.Content), req.ParentID, req.Media)
	if err != nil {
		respondPostError(c, err)
		return
	}
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
	c.Status(http.StatusNoContent)
}
