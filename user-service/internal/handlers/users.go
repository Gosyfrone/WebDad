package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/webdad/user-service/internal/logging"
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
// @Summary     Créer l'entrée utilisateur
// @Tags        users
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body models.CreateUserRequest true "Username"
// @Success     201 {object} models.User "Utilisateur créé"
// @Failure     400 {object} map[string]string "Payload invalide"
// @Failure     401 {object} map[string]string "Non authentifié"
// @Failure     409 {object} map[string]string "Username pris"
// @Router      /users [post]
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

	user, err := h.users.Create(claims.UserID, req.Username)
	if err != nil {
		respondUserError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": user})
}

// AdminCreate : POST /users/admin — crée la ligne `users` pour un id donné (admin).
// Réservé aux admins (middleware) : sert au provisioning d'un compte créé de
// force via auth-service. Si le username demandé est pris, il est suffixé et
// username_pending passe à true.
// @Summary     Créer un utilisateur pour un id donné (admin)
// @Tags        users
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body models.AdminCreateUserRequest true "ID (= credentials.id) + username souhaité"
// @Success     201 {object} models.User "Utilisateur créé (username éventuellement suffixé)"
// @Failure     400 {object} map[string]string "Payload invalide"
// @Failure     401 {object} map[string]string "Non authentifié"
// @Failure     403 {object} map[string]string "Réservé admin"
// @Failure     409 {object} map[string]string "Username pris"
// @Router      /users/admin [post]
func (h *Handler) AdminCreate(c *gin.Context) {
	var req models.AdminCreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}

	user, err := h.users.AdminCreate(req.ID, req.Username)
	if err != nil {
		respondUserError(c, err)
		return
	}

	logging.FromGin(c).Info("compte utilisateur créé (admin)", "new_user_id", user.ID)
	c.JSON(http.StatusCreated, gin.H{"data": user})
}

// List : GET /users — liste paginée (?limit=&offset=).
// @Summary     Lister les utilisateurs (paginé)
// @Tags        users
// @Produce     json
// @Param       limit  query int false "Nb résultats (défaut 20, max 100)"
// @Param       offset query int false "Décalage"
// @Success     200 {array} models.User
// @Failure     500 {object} map[string]string
// @Router      /users [get]
func (h *Handler) List(c *gin.Context) {
	limit, offset := paginate(c)
	users, err := h.users.List(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "liste des utilisateurs impossible"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": users})
}

// Search : GET /users/search?q= — recherche d'utilisateurs par username (public).
// @Summary     Rechercher des utilisateurs par username
// @Tags        users
// @Produce     json
// @Param       q      query string true  "Terme de recherche"
// @Param       limit  query int    false "Nb résultats"
// @Param       offset query int    false "Décalage"
// @Success     200 {array} models.User
// @Failure     500 {object} map[string]string
// @Router      /users/search [get]
func (h *Handler) Search(c *gin.Context) {
	limit, offset := paginate(c)
	users, err := h.users.Search(c.Query("q"), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "recherche impossible"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": users})
}

// Suggestions : GET /users/suggestions — comptes les plus suivis (public).
// @Summary     Suggestions (comptes les plus suivis)
// @Tags        users
// @Produce     json
// @Param       limit  query int false "Nb résultats"
// @Param       offset query int false "Décalage"
// @Success     200 {array} models.User
// @Failure     500 {object} map[string]string
// @Router      /users/suggestions [get]
func (h *Handler) Suggestions(c *gin.Context) {
	limit, offset := paginate(c)
	users, err := h.users.Suggestions(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "suggestions impossibles"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": users})
}

// GetByID : GET /users/:id — détail d'un utilisateur + compteurs (public).
// @Summary     Détail d'un utilisateur par ID
// @Tags        users
// @Produce     json
// @Param       id path string true "User ID"
// @Success     200 {object} models.UserDetails
// @Failure     404 {object} map[string]string
// @Router      /users/{id} [get]
func (h *Handler) GetByID(c *gin.Context) {
	user, err := h.users.GetDetailsByID(c.Param("id"))
	if err != nil {
		respondUserError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": user})
}

// GetByUsername : GET /users/by-username/:username — détail par handle (public).
// @Summary     Détail d'un utilisateur par username
// @Tags        users
// @Produce     json
// @Param       username path string true "Username (handle)"
// @Success     200 {object} models.UserDetails
// @Failure     404 {object} map[string]string
// @Router      /users/by-username/{username} [get]
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
// @Summary     Utilisateur courant (provisioning paresseux)
// @Tags        users
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} models.User
// @Failure     401 {object} map[string]string
// @Failure     500 {object} map[string]string
// @Router      /users/me [get]
func (h *Handler) GetMe(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	user, err := h.users.ProvisionFromClaims(claims.UserID, claims.Email)
	if err != nil {
		logging.FromGin(c).Error("provisioning utilisateur échoué", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "récupération de l'utilisateur impossible"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": user})
}

// UpdateMe : PATCH /users/me — modifie l'utilisateur courant (protégé).
// @Summary     Modifier le compte courant
// @Tags        users
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body models.UpdateUserRequest true "Username et/ou langue préférée"
// @Success     200 {object} models.User
// @Failure     400 {object} map[string]string
// @Failure     401 {object} map[string]string
// @Failure     409 {object} map[string]string "Username pris"
// @Failure     429 {object} map[string]string "Cooldown actif"
// @Router      /users/me [patch]
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

	user, err := h.users.Update(claims.UserID, req.Username, req.PreferredLocale)
	if err != nil {
		respondUserError(c, err)
		return
	}

	if req.Username != nil {
		logging.FromGin(c).Info("username modifié")
	}
	if req.PreferredLocale != nil {
		logging.FromGin(c).Info("langue préférée modifiée", "preferred_locale", *req.PreferredLocale)
	}
	c.JSON(http.StatusOK, gin.H{"data": user})
}

// Delete : DELETE /users/:id — désactive un compte (protégé, admin).
// @Summary     Désactiver un compte (admin)
// @Tags        users
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "User ID"
// @Success     204
// @Failure     401 {object} map[string]string
// @Failure     403 {object} map[string]string "Réservé admin"
// @Failure     404 {object} map[string]string
// @Router      /users/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	if err := h.users.SoftDelete(c.Param("id")); err != nil {
		respondUserError(c, err)
		return
	}

	logging.FromGin(c).Info("compte désactivé (admin)", "target_id", c.Param("id"))
	c.Status(http.StatusNoContent)
}

// PurgeUser : DELETE /users/:id/hard — efface DÉFINITIVEMENT un compte et son
// graphe social (effacement RGPD, admin via middleware). Idempotent.
func (h *Handler) PurgeUser(c *gin.Context) {
	if err := h.users.PurgeUser(c.Param("id")); err != nil {
		respondUserError(c, err)
		return
	}
	logging.FromGin(c).Info("compte purgé RGPD (admin)", "target_id", c.Param("id"))
	c.Status(http.StatusNoContent)
}

// SetStatus : PATCH /users/:id/status — bannit/réactive un compte (admin via
// middleware). Bascule la visibilité publique `users.is_active` ; le blocage
// de connexion est porté par auth-service (PATCH /auth/users/:id/status).
func (h *Handler) SetStatus(c *gin.Context) {
	var req models.UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}

	if err := h.users.SetActive(c.Param("id"), *req.IsActive); err != nil {
		respondUserError(c, err)
		return
	}

	logging.FromGin(c).Info("statut compte modifié (admin)", "target_id", c.Param("id"), "is_active", *req.IsActive)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"id": c.Param("id"), "is_active": *req.IsActive}})
}

// respondUserError mappe les erreurs métier vers des codes HTTP.
func respondUserError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrUserNotFound), errors.Is(err, service.ErrFollowRequestNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrUsernameTaken):
		logging.FromGin(c).Warn("conflit de username", "reason", "username_taken")
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrInvalidUsername), errors.Is(err, service.ErrInvalidLocale), errors.Is(err, service.ErrSelfFollow):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrUsernameCooldown):
		logging.FromGin(c).Warn("changement de username refusé", "reason", "cooldown")
		c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
	default:
		logging.FromGin(c).Error("erreur utilisateur inattendue", "error", err)
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
