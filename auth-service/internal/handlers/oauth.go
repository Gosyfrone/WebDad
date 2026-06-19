package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/webdad/auth-service/internal/models"
	"github.com/webdad/auth-service/internal/oauth"
	"github.com/webdad/auth-service/internal/services"
)

// OAuthURL : GET /auth/oauth/{provider}/url — renvoie l'URL d'autorisation OIDC
// et un state anti-CSRF. Le front stocke le state, redirige l'utilisateur vers
// `url`, puis revérifie le state au callback avant d'appeler /exchange.
// @Summary     Démarrer une connexion OAuth (Google, GitHub)
// @Tags        auth
// @Produce     json
// @Param       provider path string true "Fournisseur OAuth" Enums(google, github)
// @Success     200 {object} models.OAuthURLData "data: {url, state}"
// @Failure     404 {object} map[string]string "Provider non supporté/configuré"
// @Failure     500 {object} map[string]string "Erreur interne"
// @Router      /auth/oauth/{provider}/url [get]
func (h *Handler) OAuthURL(c *gin.Context) {
	provider, ok := h.resolveProvider(c)
	if !ok {
		return
	}

	state, err := oauth.NewState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "génération du state impossible"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"url":   provider.AuthURL(state),
		"state": state,
	}})
}

// OAuthExchange : POST /auth/oauth/{provider}/exchange — échange le code
// d'autorisation contre les tokens du provider, vérifie l'identité provider, puis :
//   - compte existant : émet NOS tokens (même format que /auth/login) ;
//   - nouveau compte : renvoie un pending_token, sans créer de credential/JWT.
//
// @Summary     Finaliser une connexion OAuth (Google, GitHub)
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       provider path string true "Fournisseur OAuth" Enums(google, github)
// @Param       body body models.OAuthExchangeRequest true "Code d'autorisation"
// @Success     200 {object} models.OAuthExchangeData "Connexion réussie ou inscription à finaliser"
// @Failure     400 {object} map[string]string "Payload invalide"
// @Failure     401 {object} map[string]string "Échange/vérification OAuth échoué ou email non vérifié"
// @Failure     403 {object} map[string]string "Compte désactivé"
// @Failure     404 {object} map[string]string "Provider non supporté/configuré"
// @Failure     500 {object} map[string]string "Erreur interne"
// @Router      /auth/oauth/{provider}/exchange [post]
func (h *Handler) OAuthExchange(c *gin.Context) {
	name := c.Param("provider")
	provider, ok := h.resolveProvider(c)
	if !ok {
		return
	}

	var req models.OAuthExchangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}

	identity, err := provider.Exchange(c.Request.Context(), req.Code)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "échange OAuth échoué"})
		return
	}
	// Garde-fou : sans email vérifié, on refuse (un email non vérifié pourrait
	// permettre une prise de contrôle de compte par rapprochement).
	if identity.Email == "" || !identity.EmailVerified {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "email absent ou non vérifié chez le fournisseur"})
		return
	}

	result, err := h.auth.LoginWithOAuth(name, identity.Subject, identity.Email)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrUserInactive):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "connexion impossible"})
		}
		return
	}

	if result.OnboardingRequired {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{
			"onboarding_required": true,
			"pending_token":       result.PendingToken,
			"email":               result.Email,
		}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"token":         result.Token,
		"refresh_token": result.RefreshToken,
		"user":          models.NewAuthUser(result.User),
	}})
}

// OAuthComplete : POST /auth/oauth/{provider}/complete — consomme un pending
// token OAuth et crée le credential, après validation username/date/CGU côté BFF.
// @Summary     Créer un compte après onboarding OAuth
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       provider path string true "Fournisseur OAuth" Enums(google, github)
// @Param       body body models.OAuthCompleteRequest true "Token d'inscription OAuth"
// @Success     200 {object} models.AuthUser "Compte créé — data: {token, refresh_token, user}"
// @Failure     400 {object} map[string]string "Payload invalide"
// @Failure     401 {object} map[string]string "Token invalide ou expiré"
// @Failure     409 {object} map[string]string "Compte déjà créé"
// @Failure     500 {object} map[string]string "Erreur interne"
// @Router      /auth/oauth/{provider}/complete [post]
func (h *Handler) OAuthComplete(c *gin.Context) {
	name := c.Param("provider")
	if _, ok := h.resolveProvider(c); !ok {
		return
	}

	var req models.OAuthCompleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}
	if req.AcceptedTerms == nil || !*req.AcceptedTerms {
		c.JSON(http.StatusBadRequest, gin.H{"error": "CGU non acceptées"})
		return
	}

	token, refresh, user, err := h.auth.CompleteOAuthSignup(name, req.PendingToken)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidToken):
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		case errors.Is(err, services.ErrOAuthAccountExists):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "création du compte OAuth impossible"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"token":         token,
		"refresh_token": refresh,
		"user":          models.NewAuthUser(user),
	}})
}

// resolveProvider récupère le provider de la route et écrit une 404 si absent.
func (h *Handler) resolveProvider(c *gin.Context) (*oauth.Provider, bool) {
	p, err := h.oauth.Get(c.Param("provider"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "provider OAuth non supporté ou non configuré"})
		return nil, false
	}
	return p, true
}
