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
// @Summary     Rechercher des profils par display_name
// @Tags        profils
// @Produce     json
// @Param       q     query string true  "Terme de recherche"
// @Param       limit query int    false "Nb résultats (défaut 20, max 50)"
// @Success     200 {array} models.Profil
// @Failure     500 {object} map[string]string
// @Router      /profils/search [get]
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
// @Summary     Profil public d'un utilisateur
// @Tags        profils
// @Produce     json
// @Param       userId path string true "User ID"
// @Success     200 {object} models.Profil
// @Failure     404 {object} map[string]string
// @Router      /profils/{userId} [get]
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
// @Summary     Profil de l'utilisateur courant
// @Tags        profils
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} models.Profil
// @Failure     401 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /profils/me [get]
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
// @Summary     Modifier son profil
// @Tags        profils
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body models.UpdateProfilRequest true "Champs à modifier (tous optionnels)"
// @Success     200 {object} models.Profil
// @Failure     400 {object} map[string]string
// @Failure     401 {object} map[string]string
// @Failure     409 {object} map[string]string "birth_date déjà définie"
// @Failure     429 {object} map[string]string "Cooldown display_name"
// @Router      /profils/me [patch]
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
// @Summary     Créer son profil
// @Tags        profils
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body models.CreateProfilRequest true "display_name obligatoire"
// @Success     201 {object} models.Profil
// @Failure     400 {object} map[string]string
// @Failure     401 {object} map[string]string
// @Failure     409 {object} map[string]string "Profil déjà existant"
// @Router      /profils [post]
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
	profil, err := h.profils.Create(c.Request.Context(), claims.UserID, req)
	if err != nil {
		respondProfilError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": profil})
}

// AdminCreate : POST /profils/admin — crée le profil d'un utilisateur donné (admin).
// Sert au provisioning d'un compte créé de force via auth-service : l'id vient du
// corps (pas du JWT). display_name = username effectif. Réservé aux admins.
// @Summary     Créer le profil d'un utilisateur (admin)
// @Tags        profils
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body models.AdminCreateProfilRequest true "ID (= credentials.id) + display_name"
// @Success     201 {object} models.Profil
// @Failure     400 {object} map[string]string
// @Failure     401 {object} map[string]string
// @Failure     403 {object} map[string]string "Réservé admin"
// @Failure     409 {object} map[string]string "Profil déjà existant"
// @Router      /profils/admin [post]
func (h *ProfilHandler) AdminCreate(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}
	if claims.Role != models.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "réservé aux administrateurs"})
		return
	}
	var req models.AdminCreateProfilRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}
	profil, err := h.profils.Create(c.Request.Context(), req.ID, models.CreateProfilRequest{
		DisplayName: req.DisplayName,
	})
	if err != nil {
		respondProfilError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": profil})
}

// Delete : DELETE /profils/:userId — supprime un profil (protégé, admin).
// @Summary     Supprimer un profil (admin)
// @Tags        profils
// @Produce     json
// @Security    BearerAuth
// @Param       userId path string true "User ID"
// @Success     204
// @Failure     401 {object} map[string]string
// @Failure     403 {object} map[string]string "Réservé admin"
// @Failure     404 {object} map[string]string
// @Router      /profils/{userId} [delete]
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

// GetVisibility : GET /profils/:userId/visibility — visibilité d'un profil.
// @Summary     Visibilité d'un profil (public/private)
// @Tags        profils
// @Produce     json
// @Param       userId path string true "User ID"
// @Success     200 {object} map[string]string "visibility: public|private"
// @Failure     404 {object} map[string]string
// @Router      /profils/{userId}/visibility [get]
func (h *ProfilHandler) GetVisibility(c *gin.Context) {
	userID := c.Param("userId")

	profil, err := h.profils.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "profil introuvable"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"visibility": profil.Visibility})
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
