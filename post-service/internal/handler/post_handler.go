package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/webdad/post-service/internal/middleware"
	"github.com/webdad/post-service/internal/models"
	"github.com/webdad/post-service/internal/service"
)

type PostHandler struct {
	service *service.PostService
	name    string
}

func NewPostHandler(svc *service.PostService, serviceName string) *PostHandler {
	return &PostHandler{
		service: svc,
		name:    serviceName,
	}
}

// CreatePost : POST /posts — l'auteur est dérivé du JWT (jamais du corps).
func (h *PostHandler) CreatePost(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	var req models.CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}

	post, err := h.service.CreatePost(c.Request.Context(), claims.UserID, req.Content)
	if err != nil {
		respondPostError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": post})
}

// ListPosts : GET /posts — fil paginé (public). Avec ?author_id=<id>, renvoie
// le fil d'un auteur précis (onglet « Posts » d'un profil) ; sans, le fil global.
func (h *PostHandler) ListPosts(c *gin.Context) {
	var (
		posts []models.Post
		err   error
	)
	if authorID := c.Query("author_id"); authorID != "" {
		posts, err = h.service.GetByProfile(c.Request.Context(), authorID, pageLimit(c), pageOffset(c))
	} else {
		posts, err = h.service.GetPosts(c.Request.Context(), pageLimit(c), pageOffset(c))
	}
	if err != nil {
		respondPostError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": posts})
}

// GetPost : GET /posts/:id (public).
func (h *PostHandler) GetPost(c *gin.Context) {
	post, err := h.service.GetPost(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondPostError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": post})
}

// UpdatePost : PATCH /posts/:id — réservé à l'auteur (ou modérateur/admin).
func (h *PostHandler) UpdatePost(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	var req models.UpdatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}

	post, err := h.service.UpdatePost(c.Request.Context(), c.Param("id"), req.Content, claims.UserID, claims.Role)
	if err != nil {
		respondPostError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": post})
}

// DeletePost : DELETE /posts/:id — réservé à l'auteur (ou modérateur/admin).
func (h *PostHandler) DeletePost(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	if err := h.service.DeletePost(c.Request.Context(), c.Param("id"), claims.UserID, claims.Role); err != nil {
		respondPostError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// respondPostError mappe les erreurs métier vers des codes HTTP.
func respondPostError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrPostNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrInvalidID):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur interne"})
	}
}

// pageLimit lit ?limit (défaut/borne appliqués côté service).
func pageLimit(c *gin.Context) int64 {
	n, err := strconv.ParseInt(c.Query("limit"), 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// pageOffset lit ?offset (défaut 0).
func pageOffset(c *gin.Context) int64 {
	n, err := strconv.ParseInt(c.Query("offset"), 10, 64)
	if err != nil {
		return 0
	}
	return n
}
