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
		// /auth/validate est protégée : le middleware valide le JWT et
		// pose les claims avant que le handler ne les renvoie.
		authGroup.GET("/validate", middleware.JWTAuth(auth), h.Validate)
	}

	return r
}
