package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

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
func (h *Handler) ListUsers(c *gin.Context) {
	limit, offset := paginate(c)
	users, err := h.auth.ListUsers(limit, offset, c.Query("q"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "liste des comptes impossible"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": users})
}

// SetRole : PATCH /auth/users/:id/role — change le rôle d'un compte (admin).
// Un admin ne peut pas changer son PROPRE rôle (anti-verrouillage / anti
// auto-rétrogradation accidentelle).
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
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"id": id, "role": req.Role}})
}

// SetStatus : PATCH /auth/users/:id/status — bannit/réactive un compte (admin).
// is_active=false bloque login + refresh et révoque les refresh tokens. Un
// admin ne peut pas se bannir lui-même.
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
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"id": id, "is_active": *req.IsActive}})
}

// DeleteUser : DELETE /auth/users/:id — efface DÉFINITIVEMENT les identifiants
// d'un compte (effacement RGPD, admin). Un admin ne peut pas s'effacer lui-même.
// La purge des autres services est orchestrée côté appelant.
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
