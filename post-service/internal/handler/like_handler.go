package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/webdad/post-service/internal/middleware"
	"github.com/webdad/post-service/internal/service"
)

type LikeHandler struct {
	service *service.PostService
	name    string
}

func NewLikeHandler(svc *service.PostService, serviceName string) *LikeHandler {
	return &LikeHandler{
		service: svc,
		name:    serviceName,
	}
}

// LikePost : POST /posts/:id/like — like de l'utilisateur courant (idempotent).
// @Summary     Liker un post
// @Tags        likes
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Post ID"
// @Success     200 {object} map[string]string "likes_count"
// @Failure     401 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /posts/{id}/like [post]
func (h *LikeHandler) LikePost(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}
	count, err := h.service.LikePost(c.Request.Context(), c.Param("id"), claims.UserID)
	if err != nil {
		respondPostError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"liked": true, "likes_count": count}})
}

// UnlikePost : DELETE /posts/:id/like — retrait du like (idempotent).
// @Summary     Retirer son like
// @Tags        likes
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Post ID"
// @Success     200 {object} map[string]string "likes_count"
// @Failure     401 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /posts/{id}/like [delete]
func (h *LikeHandler) UnlikePost(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}
	count, err := h.service.UnlikePost(c.Request.Context(), c.Param("id"), claims.UserID)
	if err != nil {
		respondPostError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"liked": false, "likes_count": count}})
}

// ListPostLikes : GET /posts/:id/likes (public) — ids des utilisateurs ayant liké.
// @Summary     Utilisateurs ayant liké un post
// @Tags        likes
// @Produce     json
// @Param       id path string true "Post ID"
// @Success     200 {array} string "Liste d'user IDs"
// @Failure     404 {object} map[string]string
// @Router      /posts/{id}/likes [get]
func (h *LikeHandler) ListPostLikes(c *gin.Context) {
	ids, err := h.service.PostLikers(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondPostError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": ids})
}

// ListLikedByUser : GET /posts/liked?author_id=<id> — posts likés par un
// utilisateur, antéchronologiques. Retourne 403 si les likes sont privés et
// l'appelant n'est pas le propriétaire.
// @Summary     Posts likés par un utilisateur
// @Tags        likes
// @Produce     json
// @Param       author_id query string true  "User ID"
// @Param       limit     query int    false "Nb résultats"
// @Param       offset    query int    false "Décalage"
// @Success     200 {array} models.Post
// @Failure     400 {object} map[string]string
// @Failure     403 {object} map[string]string "Likes privés"
// @Router      /posts/liked [get]
func (h *LikeHandler) ListLikedByUser(c *gin.Context) {
	authorID := c.Query("author_id")
	if authorID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "author_id requis"})
		return
	}
	callerID := ""
	if claims, ok := middleware.ClaimsFrom(c); ok {
		callerID = claims.UserID
	}
	posts, err := h.service.ListLikedByUser(c.Request.Context(), authorID, callerID, pageLimit(c), pageOffset(c))
	if err != nil {
		respondPostError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": posts})
}

// LikedByMe : GET /posts/me/liked-ids — ids des posts likés par l'utilisateur
// courant (initialise l'état des cœurs côté front, façon getFollowingIds).
// @Summary     IDs des posts que j'ai likés
// @Tags        likes
// @Produce     json
// @Security    BearerAuth
// @Success     200 {array} string "Liste d'IDs"
// @Failure     401 {object} map[string]string
// @Router      /posts/me/liked-ids [get]
func (h *LikeHandler) LikedByMe(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}
	ids, err := h.service.LikedPostIDs(c.Request.Context(), claims.UserID)
	if err != nil {
		respondPostError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": ids})
}
