package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/webdad/post-service/internal/middleware"
	"github.com/webdad/post-service/internal/models"
	"github.com/webdad/post-service/internal/service"
)

// BookmarkHandler porte les routes des signets : collections (CRUD), ajout/retrait
// d'un post dans une collection, et vues de lecture. Toutes les routes sont
// protégées par JWT (les signets sont strictement privés).
type BookmarkHandler struct {
	service *service.PostService
	name    string
}

func NewBookmarkHandler(svc *service.PostService, serviceName string) *BookmarkHandler {
	return &BookmarkHandler{service: svc, name: serviceName}
}

// --- Collections ---------------------------------------------------------------

// ListCollections : GET /posts/bookmarks/collections — collections de l'utilisateur.
// @Summary     Lister mes collections de signets
// @Tags        bookmarks
// @Produce     json
// @Security    BearerAuth
// @Success     200 {array} models.BookmarkCollection
// @Failure     401 {object} map[string]string
// @Router      /posts/bookmarks/collections [get]
func (h *BookmarkHandler) ListCollections(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}
	colls, err := h.service.ListBookmarkCollections(c.Request.Context(), claims.UserID)
	if err != nil {
		respondPostError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": colls})
}

// CreateCollection : POST /posts/bookmarks/collections — crée une collection.
// @Summary     Créer une collection de signets
// @Tags        bookmarks
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body models.CreateCollectionRequest true "Nom de la collection"
// @Success     201 {object} models.BookmarkCollection
// @Failure     400 {object} map[string]string
// @Failure     401 {object} map[string]string
// @Router      /posts/bookmarks/collections [post]
func (h *BookmarkHandler) CreateCollection(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}
	var req models.CreateCollectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}
	coll, err := h.service.CreateBookmarkCollection(c.Request.Context(), claims.UserID, req.Name)
	if err != nil {
		respondPostError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": coll})
}

// RenameCollection : PATCH /posts/bookmarks/collections/:cid — renomme une collection.
// @Summary     Renommer une collection de signets
// @Tags        bookmarks
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       cid  path string                        true "Collection ID"
// @Param       body body models.RenameCollectionRequest true "Nouveau nom"
// @Success     200 {object} models.BookmarkCollection
// @Failure     400 {object} map[string]string
// @Failure     401 {object} map[string]string
// @Failure     403 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /posts/bookmarks/collections/{cid} [patch]
func (h *BookmarkHandler) RenameCollection(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}
	var req models.RenameCollectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}
	coll, err := h.service.RenameBookmarkCollection(c.Request.Context(), c.Param("cid"), claims.UserID, req.Name)
	if err != nil {
		respondPostError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": coll})
}

// DeleteCollection : DELETE /posts/bookmarks/collections/:cid — supprime une collection.
// @Summary     Supprimer une collection de signets
// @Tags        bookmarks
// @Produce     json
// @Security    BearerAuth
// @Param       cid path string true "Collection ID"
// @Success     204
// @Failure     401 {object} map[string]string
// @Failure     403 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /posts/bookmarks/collections/{cid} [delete]
func (h *BookmarkHandler) DeleteCollection(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}
	if err := h.service.DeleteBookmarkCollection(c.Request.Context(), c.Param("cid"), claims.UserID); err != nil {
		respondPostError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ListCollectionPosts : GET /posts/bookmarks/collections/:cid/posts — posts d'une collection.
// @Summary     Posts d'une collection de signets
// @Tags        bookmarks
// @Produce     json
// @Security    BearerAuth
// @Param       cid    path  string true  "Collection ID"
// @Param       limit  query int    false "Nb résultats"
// @Param       offset query int    false "Décalage"
// @Success     200 {array} models.Post
// @Failure     401 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /posts/bookmarks/collections/{cid}/posts [get]
func (h *BookmarkHandler) ListCollectionPosts(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}
	posts, err := h.service.ListBookmarksInCollection(c.Request.Context(), c.Param("cid"), claims.UserID, pageLimit(c), pageOffset(c))
	if err != nil {
		respondPostError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": posts})
}

// --- Signets -----------------------------------------------------------------

// ListAll : GET /posts/bookmarks — vue « Tous mes signets » (union paginée).
// @Summary     Tous mes signets
// @Tags        bookmarks
// @Produce     json
// @Security    BearerAuth
// @Param       limit  query int false "Nb résultats"
// @Param       offset query int false "Décalage"
// @Success     200 {array} models.Post
// @Failure     401 {object} map[string]string
// @Router      /posts/bookmarks [get]
func (h *BookmarkHandler) ListAll(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}
	posts, err := h.service.ListAllBookmarkedPosts(c.Request.Context(), claims.UserID, pageLimit(c), pageOffset(c))
	if err != nil {
		respondPostError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": posts})
}

// Bookmark : POST /posts/:id/bookmark — range un post. Le corps `collection_id`
// est OPTIONNEL : absent (clic court) → résolution de la fenêtre de rafale
// (statut "filed" ou "needs_choice") ; fourni (sélecteur) → ajout explicite.
// @Summary     Signer (bookmarker) un post
// @Tags        bookmarks
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id   path  string               true  "Post ID"
// @Param       body body  models.BookmarkRequest false "Collection ID (optionnel)"
// @Success     200 {object} map[string]string
// @Failure     401 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /posts/{id}/bookmark [post]
func (h *BookmarkHandler) Bookmark(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}
	// Corps optionnel : un clic court n'envoie pas forcément de JSON.
	var req models.BookmarkRequest
	_ = c.ShouldBindJSON(&req)

	result, err := h.service.Bookmark(c.Request.Context(), claims.UserID, c.Param("id"), req.CollectionID)
	if err != nil {
		respondPostError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

// Unbookmark : DELETE /posts/:id/bookmark — retire un post. `collection_id`
// fourni → retire de cette collection ; absent → retire de toutes (dé-signer).
// @Summary     Retirer un signet
// @Tags        bookmarks
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id   path  string               true  "Post ID"
// @Param       body body  models.BookmarkRequest false "Collection ID (optionnel)"
// @Success     200 {object} map[string]string
// @Failure     401 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /posts/{id}/bookmark [delete]
func (h *BookmarkHandler) Unbookmark(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}
	var req models.BookmarkRequest
	_ = c.ShouldBindJSON(&req)

	if err := h.service.Unbookmark(c.Request.Context(), claims.UserID, c.Param("id"), req.CollectionID); err != nil {
		respondPostError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"bookmarked": false}})
}

// PostCollections : GET /posts/:id/bookmark/collections — ids des collections de
// l'utilisateur contenant ce post (coche le sélecteur).
// @Summary     Collections contenant un post signé
// @Tags        bookmarks
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Post ID"
// @Success     200 {array} string "Liste d'IDs de collections"
// @Failure     401 {object} map[string]string
// @Router      /posts/{id}/bookmark/collections [get]
func (h *BookmarkHandler) PostCollections(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}
	ids, err := h.service.PostBookmarkCollectionIDs(c.Request.Context(), claims.UserID, c.Param("id"))
	if err != nil {
		respondPostError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": ids})
}

// BookmarkedByMe : GET /posts/me/bookmarked-ids — ids des posts signés (état des boutons).
// @Summary     IDs des posts que j'ai signés
// @Tags        bookmarks
// @Produce     json
// @Security    BearerAuth
// @Success     200 {array} string "Liste d'IDs"
// @Failure     401 {object} map[string]string
// @Router      /posts/me/bookmarked-ids [get]
func (h *BookmarkHandler) BookmarkedByMe(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}
	ids, err := h.service.BookmarkedPostIDs(c.Request.Context(), claims.UserID)
	if err != nil {
		respondPostError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": ids})
}
