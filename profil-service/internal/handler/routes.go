package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/webdad/profil-service/internal/middleware"
	"github.com/webdad/profil-service/internal/service"
)

// RegisterRoutes enregistre les routes du service profil. jwtSecret protège
// les routes mutables / personnelles (validation locale du token émis par
// auth-service, même secret partagé).
func RegisterRoutes(r *gin.Engine, serviceName string, profils *service.ProfilService, jwtSecret string) {
	auth := middleware.JWTAuth(jwtSecret)
	h := NewProfilHandler(profils)

	r.GET("/health", Health(serviceName))
	r.POST("/profils", auth, h.Create)

	p := r.Group("/profils")
	{
		// Routes authentifiées (JWT requis).
		p.POST("/", auth, h.Create)
		p.GET("/me", auth, h.GetMe)
		p.PATCH("/me", auth, h.UpdateMe)

		// Lecture publique.
		p.GET("/search", h.Search) // ?q= : recherche par display_name
		p.GET("/:userId/visibility", h.GetVisibility)
		p.GET("/:userId", h.GetByUserID)
		p.DELETE("/:userId", auth, h.Delete) // admin (vérifié dans le handler)
	}
}
