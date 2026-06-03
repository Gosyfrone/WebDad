package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/webdad/user-service/internal/middleware"
	"github.com/webdad/user-service/internal/models"
	"github.com/webdad/user-service/internal/service"
)

// Bornes de pagination pour GET /users.
const (
	defaultLimit = 20
	maxLimit     = 100
)

// Create : POST /users — crée l'enregistrement user (protégé).
// L'id provient du JWT (claims), pas du corps : un utilisateur ne crée que
// son propre enregistrement. Endpoint surtout utile pour les tests/l'admin ;
// le flux normal repose sur le provisioning paresseux (cf. GetMe).
func (h *Handler) Create(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	var req models.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}

	user, err := h.users.Create(claims.UserID, req.Username, req.DisplayName)
	if err != nil {
		respondUserError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": user})
}

// List : GET /users — liste paginée (?limit=&offset=).
func (h *Handler) List(c *gin.Context) {
	limit, offset := paginate(c)
	users, err := h.users.List(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "liste des utilisateurs impossible"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": users})
}

// GetByID : GET /users/:id — détail d'un utilisateur + compteurs (public).
func (h *Handler) GetByID(c *gin.Context) {
	user, err := h.users.GetDetailsByID(c.Param("id"))
	if err != nil {
		respondUserError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": user})
}

// GetByUsername : GET /users/by-username/:username — détail par handle (public).
func (h *Handler) GetByUsername(c *gin.Context) {
	user, err := h.users.GetDetailsByUsername(c.Param("username"))
	if err != nil {
		respondUserError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": user})
}

// GetMe : GET /users/me — utilisateur courant (protégé).
// Provisioning paresseux : si l'enregistrement n'existe pas encore, il est
// créé à la volée à partir des claims du JWT.
func (h *Handler) GetMe(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	user, err := h.users.ProvisionFromClaims(claims.UserID, claims.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "récupération de l'utilisateur impossible"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": user})
}

// UpdateMe : PATCH /users/me — modifie l'utilisateur courant (protégé).
func (h *Handler) UpdateMe(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}

	user, err := h.users.Update(claims.UserID, req.Username, req.DisplayName)
	if err != nil {
		respondUserError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": user})
}

// Delete : DELETE /users/:id — désactive un compte (protégé, admin).
func (h *Handler) Delete(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}
	if claims.Role != models.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "réservé aux administrateurs"})
		return
	}

	if err := h.users.SoftDelete(c.Param("id")); err != nil {
		respondUserError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// respondUserError mappe les erreurs métier vers des codes HTTP.
func respondUserError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrUserNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrUsernameTaken):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrInvalidUsername), errors.Is(err, service.ErrSelfFollow):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur interne"})
	}
}

// paginate lit et borne les paramètres de pagination (?limit=&offset=).
func paginate(c *gin.Context) (limit, offset int) {
	limit = parseQueryInt(c, "limit", defaultLimit)
	if limit <= 0 || limit > maxLimit {
		limit = defaultLimit
	}
	offset = parseQueryInt(c, "offset", 0)
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

// parseQueryInt lit un paramètre de requête entier avec valeur par défaut.
func parseQueryInt(c *gin.Context, key string, fallback int) int {
	raw := c.Query(key)
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}
