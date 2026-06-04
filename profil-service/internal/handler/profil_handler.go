package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/webdad/profil-service/internal/middleware"
	"github.com/webdad/profil-service/internal/models"
	"github.com/webdad/profil-service/internal/service"
)

// ProfilHandler regroupe les handlers HTTP du profil.
type ProfilHandler struct {
	profils *service.ProfilService
}

// NewProfilHandler construit le handler.
func NewProfilHandler(profils *service.ProfilService) *ProfilHandler {
	return &ProfilHandler{profils: profils}
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
