package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/webdad/auth-service/internal/logging"
	"github.com/webdad/auth-service/internal/middleware"
	"github.com/webdad/auth-service/internal/models"
	"github.com/webdad/auth-service/internal/services"
)

// Bornes de pagination pour l'annuaire admin.
const (
	defaultLimit = 20
	maxLimit     = 100
)

// ListUsers : GET /auth/users — annuaire des comptes (admin).
// Source faisant autorité pour le rôle + l'état du compte ; le front enrichit
// avec username/avatar (user-service + profil-service). `?q=` filtre par email.
// @Summary     Lister les comptes (modérateur/admin)
// @Tags        admin
// @Produce     json
// @Security    BearerAuth
// @Param       q      query  string false "Filtre email (sous-chaîne)"
// @Param       limit  query  int    false "Nombre de résultats (défaut 20, max 100)"
// @Param       offset query  int    false "Décalage de pagination"
// @Success     200 {object} map[string]interface{} "data: []User"
// @Failure     401 {object} map[string]string "Non authentifié"
// @Failure     403 {object} map[string]string "Rôle insuffisant"
// @Failure     500 {object} map[string]string "Erreur interne"
// @Router      /auth/users [get]
func (h *Handler) ListUsers(c *gin.Context) {
	limit, offset := paginate(c)
	users, err := h.auth.ListUsers(limit, offset, c.Query("q"))
	if err != nil {
		logging.FromGin(c).Error("liste des comptes : erreur DB", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "liste des comptes impossible"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": users})
}

// PublicRoles : GET /auth/users/roles?ids=... — rôles d'affichage publics.
// @Summary     Rôles publics d'affichage
// @Tags        auth
// @Produce     json
// @Security    BearerAuth
// @Param       ids query string true "IDs utilisateurs séparés par des virgules"
// @Success     200 {object} map[string]interface{} "data: []PublicRole"
// @Failure     401 {object} map[string]string "Non authentifié"
// @Failure     500 {object} map[string]string "Erreur interne"
// @Router      /auth/users/roles [get]
func (h *Handler) PublicRoles(c *gin.Context) {
	ids := strings.Split(c.Query("ids"), ",")
	roles, err := h.auth.PublicRoles(ids)
	if err != nil {
		logging.FromGin(c).Error("rôles publics : erreur DB", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "rôles publics impossibles"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": roles})
}

// AdminCreateUser : POST /auth/users — crée un compte de force (admin).
// L'admin fournit email + mot de passe temporaire (+ username pour l'e-mail).
// Le compte est créé vérifié (l'admin se porte garant) avec un mot de passe
// temporaire (changement imposé à la 1re connexion). L'identité username/profil
// est ensuite provisionnée par l'appelant (user-service + profil-service).
// @Summary     Créer un compte avec mot de passe temporaire (admin)
// @Tags        admin
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body models.AdminCreateUserRequest true "Email + mot de passe temporaire (+ username pour l'e-mail)"
// @Success     201 {object} map[string]interface{} "data: {id, email}"
// @Failure     400 {object} map[string]string "Payload invalide"
// @Failure     401 {object} map[string]string "Non authentifié"
// @Failure     403 {object} map[string]string "Rôle insuffisant"
// @Failure     409 {object} map[string]string "Email déjà utilisé"
// @Failure     500 {object} map[string]string "Erreur interne"
// @Router      /auth/users [post]
func (h *Handler) AdminCreateUser(c *gin.Context) {
	var req models.AdminCreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}

	user, err := h.auth.AdminCreateUser(req.Email, req.Password, req.Username)
	if err != nil {
		if errors.Is(err, services.ErrEmailTaken) {
			logging.FromGin(c).Warn("création compte (admin) refusée", "reason", "email_taken")
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		logging.FromGin(c).Error("création compte (admin) : erreur inattendue", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "création du compte impossible"})
		return
	}

	logging.FromGin(c).Info("compte créé par admin", "new_user_id", user.ID)
	c.JSON(http.StatusCreated, gin.H{"data": gin.H{"id": user.ID, "email": user.Email}})
}

// SetRole : PATCH /auth/users/:id/role — change le rôle d'un compte (admin).
// Un admin ne peut pas changer son PROPRE rôle (anti-verrouillage / anti
// auto-rétrogradation accidentelle).
// @Summary     Changer le rôle d'un compte (admin)
// @Tags        admin
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id   path string                         true "ID de l'utilisateur cible"
// @Param       body body models.UpdateRoleRequest       true "Nouveau rôle"
// @Success     200 {object} map[string]interface{} "data: {id, role}"
// @Failure     400 {object} map[string]string "Payload invalide ou auto-modification"
// @Failure     401 {object} map[string]string "Non authentifié"
// @Failure     403 {object} map[string]string "Rôle insuffisant"
// @Failure     404 {object} map[string]string "Compte introuvable"
// @Failure     500 {object} map[string]string "Erreur interne"
// @Router      /auth/users/{id}/role [patch]
func (h *Handler) SetRole(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}
	id := c.Param("id")
	if id == claims.UserID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "impossible de changer son propre rôle"})
		return
	}

	var req models.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}

	if err := h.auth.SetRole(id, req.Role); err != nil {
		respondAdminError(c, err)
		return
	}
	logging.FromGin(c).Info("rôle modifié", "target_id", id, "new_role", req.Role)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"id": id, "role": req.Role}})
}

// SetStatus : PATCH /auth/users/:id/status — bannit/réactive un compte (admin).
// is_active=false bloque login + refresh et révoque les refresh tokens. Un
// admin ne peut pas se bannir lui-même.
// @Summary     Bannir / réactiver un compte (modérateur/admin)
// @Tags        admin
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id   path string                           true "ID de l'utilisateur cible"
// @Param       body body models.UpdateStatusRequest       true "Nouvel état actif"
// @Success     200 {object} map[string]interface{} "data: {id, is_active}"
// @Failure     400 {object} map[string]string "Payload invalide ou auto-modification"
// @Failure     401 {object} map[string]string "Non authentifié"
// @Failure     403 {object} map[string]string "Rôle insuffisant"
// @Failure     404 {object} map[string]string "Compte introuvable"
// @Failure     500 {object} map[string]string "Erreur interne"
// @Router      /auth/users/{id}/status [patch]
func (h *Handler) SetStatus(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}
	id := c.Param("id")
	if id == claims.UserID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "impossible de changer son propre état de compte"})
		return
	}

	// Protection de la hiérarchie : un modérateur ne peut bannir/réactiver qu'un
	// simple utilisateur (pas un autre modérateur ni un admin) — sinon un mod
	// pourrait neutraliser un admin. Un admin agit sur n'importe quelle cible.
	if claims.Role == models.RoleModerator {
		targetRole, err := h.auth.RoleOf(id)
		if err != nil {
			respondAdminError(c, err)
			return
		}
		if targetRole != models.RoleUser {
			respondAdminError(c, services.ErrInsufficientPrivilege)
			return
		}
	}

	var req models.UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}

	if err := h.auth.SetActive(id, *req.IsActive); err != nil {
		respondAdminError(c, err)
		return
	}
	action := "réactivé"
	if !*req.IsActive {
		action = "banni"
	}
	logging.FromGin(c).Info("statut compte modifié", "target_id", id, "action", action)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"id": id, "is_active": *req.IsActive}})
}

// DeleteUser : DELETE /auth/users/:id — efface DÉFINITIVEMENT les identifiants
// d'un compte (effacement RGPD, admin). Un admin ne peut pas s'effacer lui-même.
// La purge des autres services est orchestrée côté appelant.
// @Summary     Supprimer définitivement un compte (admin, RGPD)
// @Tags        admin
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "ID de l'utilisateur cible"
// @Success     204 "Compte supprimé"
// @Failure     400 {object} map[string]string "Auto-suppression interdite"
// @Failure     401 {object} map[string]string "Non authentifié"
// @Failure     403 {object} map[string]string "Rôle insuffisant"
// @Failure     404 {object} map[string]string "Compte introuvable"
// @Failure     500 {object} map[string]string "Erreur interne"
// @Router      /auth/users/{id} [delete]
func (h *Handler) DeleteUser(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}
	id := c.Param("id")
	if id == claims.UserID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "impossible de supprimer son propre compte"})
		return
	}
	if err := h.auth.DeleteAccount(id); err != nil {
		respondAdminError(c, err)
		return
	}
	logging.FromGin(c).Info("compte supprimé (RGPD)", "target_id", id)
	c.Status(http.StatusNoContent)
}

// respondAdminError mappe les erreurs métier admin vers des codes HTTP.
func respondAdminError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrUserNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrInvalidRole):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrInsufficientPrivilege):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	default:
		logging.FromGin(c).Error("erreur admin inattendue", "error", err)
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
