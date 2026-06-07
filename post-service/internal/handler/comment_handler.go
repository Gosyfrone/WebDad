package handler

import (
	"net/http"

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

// ListPostComments : GET /posts/:id/comments (public) — fil chronologique paginé.
func (h *CommentHandler) ListPostComments(c *gin.Context) {
	comments, err := h.service.ListComments(c.Request.Context(), c.Param("id"), pageLimit(c), pageOffset(c))
	if err != nil {
		respondPostError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": comments})
}

// CreatPostComment : POST /posts/:id/comments — l'auteur est dérivé du JWT.
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

	comment, err := h.service.CreateComment(c.Request.Context(), c.Param("id"), claims.UserID, req.Content)
	if err != nil {
		respondPostError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": comment})
}

// DeletePostComment : DELETE /posts/:id/comments/:commentId — auteur du
// commentaire (ou modérateur/admin).
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
