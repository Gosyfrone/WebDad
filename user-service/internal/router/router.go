// Package router enregistre les routes Gin du service user.
package router

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/webdad/user-service/internal/handlers"
	"github.com/webdad/user-service/internal/middleware"
	"github.com/webdad/user-service/internal/service"
)

const serviceName = "user-service"

// startedAt : instant d'init du package (≈ démarrage du process), exposé en
// uptime dans /health (consommé par le monitoring admin du gateway).
var startedAt = time.Now()

// New construit le routeur Gin avec toutes les routes du service.
// jwtSecret protège les routes mutables / personnelles (validation locale
// du token émis par auth-service, même secret partagé).
func New(users *service.UserService, jwtSecret string) *gin.Engine {
	r := gin.New()
	r.Use(middleware.RequestID(), middleware.Recovery(), middleware.RequestLogger())
	h := handlers.New(users)
	auth := middleware.JWTAuth(jwtSecret)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":         "ok",
			"service":        serviceName,
			"uptime_seconds": int(time.Since(startedAt).Seconds()),
		})
	})

	u := r.Group("/users")
	{
		// RIV-003 : l'annuaire complet (énumération de toute la base) exige
		// désormais une session. Le front ne consomme pas cette liste (il passe
		// par /search, /suggestions, /by-username) → restriction sans impact UX.
		u.GET("", auth, h.List)
		// Lecture publique (volontairement) : nécessaire au login (résolution
		// username→id), aux aperçus de profil et à la recherche de l'explorer.
		u.GET("/search", h.Search)           // ?q= : recherche par username
		u.GET("/suggestions", h.Suggestions) // comptes les plus suivis
		u.GET("/by-username/:username", h.GetByUsername)
		u.GET("/:id", h.GetByID)
		u.GET("/:id/followers", h.Followers)
		u.GET("/:id/following", h.Following)

		// Routes authentifiées (JWT requis).
		u.POST("", auth, h.Create)
		// Provisioning d'un compte créé par un admin (id imposé, suffixe si pris) → admin.
		u.POST("/admin", auth, middleware.AdminOnly(), h.AdminCreate)
		u.GET("/me", auth, h.GetMe)
		u.PATCH("/me", auth, h.UpdateMe)
		u.GET("/me/blocks", auth, h.BlockedUsers)
		u.GET("/me/follow-requests/outgoing", auth, h.PendingFollowRequests)
		u.DELETE("/me/followers/:id", auth, h.RemoveFollower)
		u.POST("/:id/follow", auth, h.Follow)
		u.DELETE("/:id/follow", auth, h.Unfollow)
		u.POST("/:id/block", auth, h.Block)
		u.DELETE("/:id/block", auth, h.Unblock)
		u.POST("/follow-requests/:followerId/accept", auth, h.AcceptFollowRequest)
		u.POST("/follow-requests/:followerId/reject", auth, h.RejectFollowRequest)

		// Administration / modération (JWT requis).
		//   - bannir/réactiver = visibilité publique du compte (volet du
		//     bannissement, le blocage de connexion vit dans auth-service) →
		//     MODÉRATION (mod ou admin) ;
		//   - suppression douce du compte (DELETE) = action de gouvernance →
		//     admin uniquement.
		u.PATCH("/:id/status", auth, middleware.ModeratorOnly(), h.SetStatus)
		u.DELETE("/:id", auth, middleware.AdminOnly(), h.Delete)
		// Effacement RGPD : purge définitive du compte + graphe social (admin).
		u.DELETE("/:id/hard", auth, middleware.AdminOnly(), h.PurgeUser)
	}
	internal := r.Group("/internal")
	{
		internal.GET("/:userId/is-following/:followingId", h.IsFollowing)
		internal.GET("/follows/:userId/is-following/:followingId", h.IsFollowing)
		internal.GET("/users/:blockerId/has-blocked/:blockedId", h.HasBlocked)
		// Acceptation en masse (privé → public), émise par profil-service.
		internal.POST("/users/:ownerId/accept-all-follow-requests", h.AcceptAllFollowRequests)
	}

	return r
}
