package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/webdad/profil-service/internal/middleware"
	"github.com/webdad/profil-service/internal/models"
	"github.com/webdad/profil-service/internal/service"
)

// Bornes de pagination de la recherche de profils.
const (
	defaultSearchLimit = 20
	maxSearchLimit     = 50
)

// ProfilHandler regroupe les handlers HTTP du profil.
type ProfilHandler struct {
	profils *service.ProfilService
}

// NewProfilHandler construit le handler.
func NewProfilHandler(profils *service.ProfilService) *ProfilHandler {
	return &ProfilHandler{profils: profils}
}

// Search : GET /profils/search?q=&limit= — recherche par display_name (public).
func (h *ProfilHandler) Search(c *gin.Context) {
	profils, err := h.profils.Search(c.Request.Context(), c.Query("q"), searchLimit(c))
	if err != nil {
		respondProfilError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": profils})
}

// searchLimit lit ?limit (défaut 20, borné à 50).
func searchLimit(c *gin.Context) int64 {
	n, err := strconv.Atoi(c.Query("limit"))
	if err != nil || n <= 0 {
		return defaultSearchLimit
	}
	if n > maxSearchLimit {
		return maxSearchLimit
	}
	return int64(n)
}

// GetByUserID : GET /profils/:userId — profil public d'un utilisateur.
func (h *ProfilHandler) GetByUserID(c *gin.Context) {
	profil, err := h.profils.GetByUserID(c.Request.Context(), c.Param("userId"))
	if err != nil {
		respondProfilError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": profil})
}

// GetMe : GET /profils/me — profil de l'utilisateur courant (protégé).
// Lecture seule : ne crée RIEN. 404 si le profil n'existe pas encore (le front
// le crée alors via POST /profils). La création est l'apanage exclusif du POST.
func (h *ProfilHandler) GetMe(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}
	profil, err := h.profils.GetByUserID(c.Request.Context(), claims.UserID)
	if err != nil {
		respondProfilError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": profil})
}

// UpdateMe : PATCH /profils/me — modifie le profil de l'utilisateur courant.
func (h *ProfilHandler) UpdateMe(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}
	var req models.UpdateProfilRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}
	profil, err := h.profils.Update(c.Request.Context(), claims.UserID, req)
	if err != nil {
		respondProfilError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": profil})
}

// Create : POST /profils — crée le profil de l'utilisateur courant (protégé).
// L'id provient du JWT, pas du corps. Surtout utile aux tests/à l'admin ; le
// flux normal repose sur le provisioning paresseux (cf. GetMe).
func (h *ProfilHandler) Create(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}
	// display_name obligatoire (= username au register, posé par le BFF).
	var req models.CreateProfilRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}
	profil, err := h.profils.Create(c.Request.Context(), claims.UserID, req.DisplayName)
	if err != nil {
		respondProfilError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": profil})
}

// Delete : DELETE /profils/:userId — supprime un profil (protégé, admin).
func (h *ProfilHandler) Delete(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}
	if claims.Role != models.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "réservé aux administrateurs"})
		return
	}
	if err := h.profils.Delete(c.Request.Context(), c.Param("userId")); err != nil {
		respondProfilError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// respondProfilError mappe les erreurs métier vers des codes HTTP.
func respondProfilError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrProfilNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrProfilExists), errors.Is(err, service.ErrBirthDateLocked):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrDisplayNameCooldown):
		c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur interne"})
	}
}
