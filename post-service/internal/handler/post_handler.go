package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

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
// @Summary     Créer un post
// @Tags        posts
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body models.CreatePostRequest true "Contenu (texte et/ou médias)"
// @Success     201 {object} models.Post
// @Failure     400 {object} map[string]string
// @Failure     401 {object} map[string]string
// @Router      /posts [post]
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
	// Un post doit porter du texte OU au moins un média (pas les deux vides).
	if strings.TrimSpace(req.Content) == "" && len(req.Media) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "post vide : texte ou média requis"})
		return
	}

	post, err := h.service.CreatePost(c.Request.Context(), claims.UserID, req.Content, req.QuotePostID, req.Media)
	if err != nil {
		respondPostError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": post})
}

// ListPosts : GET /posts — fil paginé (public). Trois modes :
//   - ?author_id=<id>        : fil d'un auteur (onglet « Posts » d'un profil) ;
//   - ?author_ids=<id,id,…>  : fil « Abonnements » (posts des comptes suivis,
//     le front fournit les ids — seul user-service connaît le graphe) ;
//   - sans paramètre         : fil global.
//
// @Summary     Fil de posts (global, profil ou abonnements)
// @Tags        posts
// @Produce     json
// @Param       author_id  query string false "Fil d'un auteur"
// @Param       author_ids query string false "Fil abonnements (IDs séparés par virgule)"
// @Param       limit      query int    false "Nb résultats"
// @Param       offset     query int    false "Décalage"
// @Success     200 {array} models.Post
// @Failure     500 {object} map[string]string
// @Router      /posts [get]
func (h *PostHandler) ListPosts(c *gin.Context) {
	var (
		posts []models.Post
		err   error
	)
	viewerID := ""
	if claims, ok := middleware.ClaimsFrom(c); ok {
		viewerID = claims.UserID
	}
	switch {
	case c.Query("author_ids") != "":
		posts, err = h.service.GetFeed(c.Request.Context(), splitIDs(c.Query("author_ids")), viewerID, pageLimit(c), pageOffset(c))
	case c.Query("author_id") != "":
		posts, err = h.service.GetByProfile(c.Request.Context(), c.Query("author_id"), viewerID, pageLimit(c), pageOffset(c))
	default:
		posts, err = h.service.GetPosts(c.Request.Context(), viewerID, pageLimit(c), pageOffset(c))
	}
	if err != nil {
		respondPostError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": posts})
}

// GetPost : GET /posts/:id (public). Un post masqué par la modération n'est
// visible qu'à un modérateur/admin (rôle lu dans le JWT optionnel).
// @Summary     Détail d'un post
// @Tags        posts
// @Produce     json
// @Param       id path string true "Post ID (ObjectID hex)"
// @Success     200 {object} models.Post
// @Failure     404 {object} map[string]string
// @Router      /posts/{id} [get]
func (h *PostHandler) GetPost(c *gin.Context) {
	viewerID := ""
	viewerRole := ""
	if claims, ok := middleware.ClaimsFrom(c); ok {
		viewerID = claims.UserID
		viewerRole = claims.Role
	}
	post, err := h.service.GetPost(c.Request.Context(), c.Param("id"), viewerID, viewerRole)
	if err != nil {
		respondPostError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": post})
}

// UpdatePost : PATCH /posts/:id — réservé à l'auteur (ou modérateur/admin).
// @Summary     Modifier un post (auteur/modérateur/admin)
// @Tags        posts
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id   path string                    true "Post ID"
// @Param       body body models.UpdatePostRequest  true "Nouveau contenu"
// @Success     200 {object} models.Post
// @Failure     400 {object} map[string]string
// @Failure     401 {object} map[string]string
// @Failure     403 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /posts/{id} [patch]
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

// PinPost : PATCH /posts/:id/pin — réservé à l'auteur du post. L'épinglage
// est persistant et visible par tous sur le profil public.
// @Summary     Épingler un post (auteur uniquement)
// @Tags        posts
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Post ID"
// @Success     200 {object} models.Post
// @Failure     401 {object} map[string]string
// @Failure     403 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /posts/{id}/pin [patch]
func (h *PostHandler) PinPost(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	post, err := h.service.PinPost(c.Request.Context(), c.Param("id"), claims.UserID)
	if err != nil {
		respondPostError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": post})
}

// UnpinPost : DELETE /posts/:id/pin — réservé à l'auteur du post.
// @Summary     Désépingler un post
// @Tags        posts
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Post ID"
// @Success     200 {object} models.Post
// @Failure     401 {object} map[string]string
// @Failure     403 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /posts/{id}/pin [delete]
func (h *PostHandler) UnpinPost(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	post, err := h.service.UnpinPost(c.Request.Context(), c.Param("id"), claims.UserID)
	if err != nil {
		respondPostError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": post})
}

// RepostPost : POST /posts/:id/repost — repost simple, visible sur le profil
// de l'acteur. Idempotent côté service.
// @Summary     Reposter un post
// @Tags        posts
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Post ID"
// @Success     200 {object} models.Post
// @Failure     401 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /posts/{id}/repost [post]
func (h *PostHandler) RepostPost(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	post, err := h.service.RepostPost(c.Request.Context(), c.Param("id"), claims.UserID)
	if err != nil {
		respondPostError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": post})
}

// UnrepostPost : DELETE /posts/:id/repost — retire le repost simple.
// @Summary     Retirer un repost
// @Tags        posts
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Post ID"
// @Success     200 {object} map[string]string "reposts_count"
// @Failure     401 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /posts/{id}/repost [delete]
func (h *PostHandler) UnrepostPost(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	count, err := h.service.UnrepostPost(c.Request.Context(), c.Param("id"), claims.UserID)
	if err != nil {
		respondPostError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"reposts_count": count}})
}

// RepostedByMe : GET /posts/me/reposted-ids — état initial des boutons repost.
// @Summary     IDs des posts que j'ai repostés
// @Tags        posts
// @Produce     json
// @Security    BearerAuth
// @Success     200 {array} string "Liste d'IDs"
// @Failure     401 {object} map[string]string
// @Router      /posts/me/reposted-ids [get]
func (h *PostHandler) RepostedByMe(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	ids, err := h.service.RepostedPostIDs(c.Request.Context(), claims.UserID)
	if err != nil {
		respondPostError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": ids})
}

// DeletePost : DELETE /posts/:id — réservé à l'auteur (ou modérateur/admin).
// @Summary     Supprimer un post (auteur/modérateur/admin)
// @Tags        posts
// @Security    BearerAuth
// @Param       id path string true "Post ID"
// @Success     204
// @Failure     401 {object} map[string]string
// @Failure     403 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /posts/{id} [delete]
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

// ListHidden : GET /posts/moderation/deleted — corbeille de modération (posts
// retirés en suppression douce, partagée mod/admin). Réservé via ModeratorOnly.
func (h *PostHandler) ListHidden(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}
	posts, err := h.service.ListHiddenPosts(c.Request.Context(), claims.Role, pageLimit(c), pageOffset(c))
	if err != nil {
		respondPostError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": posts})
}

// RestorePost : POST /posts/:id/restore — restaure un post masqué depuis la
// corbeille (mod/admin). Pas de confirmation : action réversible.
func (h *PostHandler) RestorePost(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}
	post, err := h.service.RestorePost(c.Request.Context(), c.Param("id"), claims.Role)
	if err != nil {
		respondPostError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": post})
}

// PurgePost : DELETE /posts/:id/purge — efface DÉFINITIVEMENT un post de la
// corbeille (mod/admin). Irréversible.
func (h *PostHandler) PurgePost(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}
	if err := h.service.PurgePost(c.Request.Context(), c.Param("id"), claims.UserID, claims.Role); err != nil {
		respondPostError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// PurgeUserData : DELETE /posts/by-author/:id — efface toutes les données d'un
// utilisateur (effacement RGPD, admin). Renvoie le nombre de posts supprimés.
func (h *PostHandler) PurgeUserData(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}
	n, err := h.service.PurgeUserData(c.Request.Context(), c.Param("id"), claims.Role)
	if err != nil {
		respondPostError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"posts_deleted": n}})
}

// respondPostError mappe les erreurs métier vers des codes HTTP.
func respondPostError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrPostNotFound), errors.Is(err, service.ErrCollectionNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrInvalidID):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrForbidden), errors.Is(err, service.ErrDefaultCollection):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrPrivateProfil):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrDependencyUnavailable):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
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

// splitIDs découpe une liste d'ids séparés par des virgules, en ignorant les
// segments vides (ex. « a,,b, » → ["a","b"]).
func splitIDs(raw string) []string {
	parts := strings.Split(raw, ",")
	ids := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			ids = append(ids, p)
		}
	}
	return ids
}
