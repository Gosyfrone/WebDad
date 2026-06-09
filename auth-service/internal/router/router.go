// Package router enregistre les routes Gin du service auth.
package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/webdad/auth-service/internal/handlers"
	"github.com/webdad/auth-service/internal/middleware"
	"github.com/webdad/auth-service/internal/services"
)

const serviceName = "auth-service"

// New construit le routeur Gin avec toutes les routes du service.
func New(auth *services.AuthService) *gin.Engine {
	r := gin.Default()
	h := handlers.New(auth)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": serviceName})
	})

	authGroup := r.Group("/auth")
	{
		authGroup.POST("/register", h.Register)
		authGroup.POST("/login", h.Login)
		// /auth/refresh : échange le refresh token (cookie httpOnly relayé par
		// le BFF) contre une nouvelle paire access+refresh (rotation).
		authGroup.POST("/refresh", h.Refresh)
		// /auth/logout : révoque le refresh token (best-effort).
		authGroup.POST("/logout", h.Logout)
		// /auth/validate est protégée : le middleware valide le JWT et
		// pose les claims avant que le handler ne les renvoie.
		authGroup.GET("/validate", middleware.JWTAuth(auth), h.Validate)

		// Administration des comptes (rôle + bannissement). Garde : JWT valide
		// + rôle admin. Source de vérité du rôle et de l'état du compte.
		admin := authGroup.Group("/users", middleware.JWTAuth(auth), middleware.AdminOnly())
		{
			admin.GET("", h.ListUsers)
			admin.PATCH("/:id/role", h.SetRole)
			admin.PATCH("/:id/status", h.SetStatus)
		}
	}

	return r
}
