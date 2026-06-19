package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/webdad/auth-service/internal/logging"
	"github.com/webdad/auth-service/internal/middleware"
	"github.com/webdad/auth-service/internal/models"
	"github.com/webdad/auth-service/internal/services"
)

// MFASetup : POST /auth/mfa/setup — génère un secret TOTP (non encore confirmé)
// et renvoie le QR + l'URI otpauth + le secret en clair. Authentifié.
// @Summary     Démarrer la configuration MFA (TOTP)
// @Description Génère un secret TOTP et renvoie un QR code (data URI) + l'URI otpauth:// à scanner dans une app d'authentification. Le secret n'est PAS encore actif : il faut le confirmer via /auth/mfa/enable.
// @Tags        auth
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} models.MFASetupData "data: {secret, otpauth_url, qr_data_uri}"
// @Failure     401 {object} map[string]string "Non authentifié"
// @Failure     409 {object} map[string]string "MFA déjà activée"
// @Failure     503 {object} map[string]string "MFA non configurée (code: mfa_not_configured)"
// @Router      /auth/mfa/setup [post]
func (h *Handler) MFASetup(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	data, err := h.auth.SetupMFA(claims.UserID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrMFANotConfigured):
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error(), "code": "mfa_not_configured"})
		case errors.Is(err, services.ErrMFAAlreadyEnabled):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error(), "code": "mfa_already_enabled"})
		case errors.Is(err, services.ErrUserNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			logging.FromGin(c).Error("mfa setup : erreur inattendue", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "configuration MFA impossible"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

// MFAEnable : POST /auth/mfa/enable — confirme la configuration avec un code TOTP.
// @Summary     Activer la MFA (confirmation du code)
// @Tags        auth
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body models.MFAEnableRequest true "Code TOTP affiché par l'app"
// @Success     200 {object} map[string]bool "data: {enabled: true}"
// @Failure     400 {object} map[string]string "Code invalide (code: invalid_mfa_code) ou aucun setup en attente (code: mfa_not_pending)"
// @Failure     401 {object} map[string]string "Non authentifié"
// @Failure     409 {object} map[string]string "MFA déjà activée"
// @Failure     503 {object} map[string]string "MFA non configurée"
// @Router      /auth/mfa/enable [post]
func (h *Handler) MFAEnable(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	var req models.MFAEnableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}

	if err := h.auth.EnableMFA(claims.UserID, req.Code); err != nil {
		switch {
		case errors.Is(err, services.ErrMFANotConfigured):
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error(), "code": "mfa_not_configured"})
		case errors.Is(err, services.ErrMFAAlreadyEnabled):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error(), "code": "mfa_already_enabled"})
		case errors.Is(err, services.ErrMFANotPending):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "mfa_not_pending"})
		case errors.Is(err, services.ErrInvalidMFACode):
			logging.FromGin(c).Warn("mfa enable refusé", "reason", "invalid_code")
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "invalid_mfa_code"})
		case errors.Is(err, services.ErrUserNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			logging.FromGin(c).Error("mfa enable : erreur inattendue", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "activation MFA impossible"})
		}
		return
	}

	logging.FromGin(c).Info("MFA activée", "user_id", claims.UserID)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"enabled": true}})
}

// MFADisable : POST /auth/mfa/disable — désactive la MFA (code TOTP OU mot de passe).
// @Summary     Désactiver la MFA
// @Tags        auth
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body models.MFADisableRequest true "Code TOTP courant OU mot de passe du compte"
// @Success     200 {object} map[string]bool "data: {enabled: false}"
// @Failure     400 {object} map[string]string "Preuve invalide (code: invalid_mfa_code) ou MFA non active (code: mfa_not_enabled)"
// @Failure     401 {object} map[string]string "Non authentifié"
// @Failure     503 {object} map[string]string "MFA non configurée"
// @Router      /auth/mfa/disable [post]
func (h *Handler) MFADisable(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	var req models.MFADisableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}

	if err := h.auth.DisableMFA(claims.UserID, req.Code, req.Password); err != nil {
		switch {
		case errors.Is(err, services.ErrMFANotConfigured):
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error(), "code": "mfa_not_configured"})
		case errors.Is(err, services.ErrMFANotEnabled):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "mfa_not_enabled"})
		case errors.Is(err, services.ErrInvalidMFACode):
			logging.FromGin(c).Warn("mfa disable refusé", "reason", "invalid_proof")
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "invalid_mfa_code"})
		case errors.Is(err, services.ErrUserNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			logging.FromGin(c).Error("mfa disable : erreur inattendue", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "désactivation MFA impossible"})
		}
		return
	}

	logging.FromGin(c).Info("MFA désactivée", "user_id", claims.UserID)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"enabled": false}})
}

// MFAStatus : GET /auth/mfa/status — état d'activation de la MFA du compte.
// @Summary     Statut MFA du compte
// @Tags        auth
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} map[string]bool "data: {enabled, configured}"
// @Failure     401 {object} map[string]string "Non authentifié"
// @Router      /auth/mfa/status [get]
func (h *Handler) MFAStatus(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	enabled, err := h.auth.MFAStatus(claims.UserID)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		logging.FromGin(c).Error("mfa status : erreur inattendue", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "lecture du statut MFA impossible"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"enabled":    enabled,
		"configured": h.auth.MFAConfigured(),
	}})
}

// MFAVerify : POST /auth/mfa/verify — valide le second facteur d'un login et
// émet la session. PUBLIC : le challenge (émis au login) remplace l'auth.
// @Summary     Valider le second facteur (login MFA)
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       body body models.MFAVerifyRequest true "Challenge (issu du login) + code TOTP"
// @Success     200 {object} models.AuthUser "Connexion réussie — data: {token, refresh_token, user}"
// @Failure     400 {object} map[string]string "Challenge invalide/expiré (code: invalid_challenge) ou code TOTP faux (code: invalid_mfa_code)"
// @Failure     403 {object} map[string]string "Compte désactivé"
// @Failure     503 {object} map[string]string "MFA non configurée"
// @Router      /auth/mfa/verify [post]
func (h *Handler) MFAVerify(c *gin.Context) {
	var req models.MFAVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}

	token, refresh, user, err := h.auth.VerifyMFA(req.Challenge, req.Code)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrMFANotConfigured):
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error(), "code": "mfa_not_configured"})
		case errors.Is(err, services.ErrInvalidToken):
			logging.FromGin(c).Warn("mfa verify refusé", "reason", "invalid_challenge")
			c.JSON(http.StatusBadRequest, gin.H{"error": "challenge invalide ou expiré", "code": "invalid_challenge"})
		case errors.Is(err, services.ErrInvalidMFACode):
			logging.FromGin(c).Warn("mfa verify refusé", "reason", "invalid_code")
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "invalid_mfa_code"})
		case errors.Is(err, services.ErrMFANotEnabled):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "mfa_not_enabled"})
		case errors.Is(err, services.ErrUserInactive):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			logging.FromGin(c).Error("mfa verify : erreur inattendue", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "vérification MFA impossible"})
		}
		return
	}

	logging.FromGin(c).Info("connexion MFA réussie", "user_id", user.ID)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"token":         token,
		"refresh_token": refresh,
		"user":          models.NewAuthUser(user),
	}})
}
