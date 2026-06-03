// Package router enregistre les routes Gin du service user.
package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/webdad/user-service/internal/handlers"
	"github.com/webdad/user-service/internal/middleware"
	"github.com/webdad/user-service/internal/service"
)

const serviceName = "user-service"

// New construit le routeur Gin avec toutes les routes du service.
// jwtSecret protège les routes mutables / personnelles (validation locale
// du token émis par auth-service, même secret partagé).
func New(users *service.UserService, jwtSecret string) *gin.Engine {
	r := gin.Default()
	h := handlers.New(users)
	auth := middleware.JWTAuth(jwtSecret)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": serviceName})
	})

	u := r.Group("/users")
	{
		// Lecture publique.
		u.GET("", h.List)
		u.GET("/by-username/:username", h.GetByUsername)
		u.GET("/:id", h.GetByID)
		u.GET("/:id/followers", h.Followers)
		u.GET("/:id/following", h.Following)

		// Routes authentifiées (JWT requis).
		u.POST("", auth, h.Create)
		u.GET("/me", auth, h.GetMe)
		u.PATCH("/me", auth, h.UpdateMe)
		u.DELETE("/:id", auth, h.Delete) // admin (vérifié dans le handler)
		u.POST("/:id/follow", auth, h.Follow)
		u.DELETE("/:id/follow", auth, h.Unfollow)
	}

	return r
}
