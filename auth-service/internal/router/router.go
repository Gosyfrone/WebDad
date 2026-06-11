// Package router enregistre les routes Gin du service auth.
package router

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/webdad/auth-service/internal/handlers"
	"github.com/webdad/auth-service/internal/middleware"
	"github.com/webdad/auth-service/internal/oauth"
	"github.com/webdad/auth-service/internal/services"
)

const serviceName = "auth-service"

// startedAt : instant d'init du package (≈ démarrage du process), exposé en
// uptime dans /health (consommé par le monitoring admin du gateway).
var startedAt = time.Now()

// New construit le routeur Gin avec toutes les routes du service.
func New(auth *services.AuthService, oauthReg *oauth.Registry) *gin.Engine {
	r := gin.Default()
	h := handlers.New(auth, oauthReg)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":         "ok",
			"service":        serviceName,
			"uptime_seconds": int(time.Since(startedAt).Seconds()),
		})
	})

	authGroup := r.Group("/auth")
	{
		authGroup.POST("/register", h.Register)
		authGroup.POST("/login", h.Login)
		// Vérification d'e-mail (PUBLIQUES, pas de middleware JWT) :
		// confirm consomme le token du lien ; request (re)envoie le mail.
		authGroup.POST("/verify-email/confirm", h.ConfirmVerifyEmail)
		authGroup.POST("/verify-email/request", h.RequestVerifyEmail)
		// Mot de passe oublié (PUBLIQUES, pas de middleware JWT) : forgot
		// déclenche le mail (anti-énumération) ; reset consomme le token.
		authGroup.POST("/password/forgot", h.ForgotPassword)
		authGroup.POST("/password/reset", h.ResetPassword)
		// /auth/refresh : échange le refresh token (cookie httpOnly relayé par
		// le BFF) contre une nouvelle paire access+refresh (rotation).
		authGroup.POST("/refresh", h.Refresh)
		// /auth/logout : révoque le refresh token (best-effort).
		authGroup.POST("/logout", h.Logout)
		// /auth/validate est protégée : le middleware valide le JWT et
		// pose les claims avant que le handler ne les renvoie.
		authGroup.GET("/validate", middleware.JWTAuth(auth), h.Validate)

		// Administration des comptes. Source de vérité du rôle et de l'état du
		// compte. Gardes différenciées (JWT valide d'abord) :
		//   - annuaire + bannissement/réactivation = MODÉRATION → mod ou admin
		//     (un modérateur « fait régner l'ordre » : il voit les comptes et peut
		//     bannir/réactiver) ;
		//   - changement de rôle = GOUVERNANCE → admin uniquement.
		admin := authGroup.Group("/users", middleware.JWTAuth(auth))
		{
			admin.GET("", middleware.ModeratorOnly(), h.ListUsers)
			admin.PATCH("/:id/status", middleware.ModeratorOnly(), h.SetStatus)
			admin.PATCH("/:id/role", middleware.AdminOnly(), h.SetRole)
			// Effacement RGPD : purge définitive des identifiants (admin).
			admin.DELETE("/:id", middleware.AdminOnly(), h.DeleteUser)
		}

		// OAuth OIDC (Login with Google) — l'échange code→tokens et
		// la vérification de l'ID token se font côté serveur.
		oauthGroup := authGroup.Group("/oauth/:provider")
		{
			oauthGroup.GET("/url", h.OAuthURL)
			oauthGroup.POST("/exchange", h.OAuthExchange)
		}
	}

	return r
}
