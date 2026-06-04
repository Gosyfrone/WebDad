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

	p := r.Group("/profils")
	{
		// Lecture publique du profil d'un utilisateur.
		p.GET("/:userId", h.GetByUserID)

		// Routes authentifiées (JWT requis).
		p.POST("", auth, h.Create)
		p.GET("/me", auth, h.GetMe)
		p.PATCH("/me", auth, h.UpdateMe)
		p.DELETE("/:userId", auth, h.Delete) // admin (vérifié dans le handler)
	}
}
