package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/webdad/message-service/internal/middleware"
	"github.com/webdad/message-service/internal/realtime"
	"github.com/webdad/message-service/internal/service"
)

// RegisterRoutes enregistre les routes du message-service. Toutes vivent sous le
// préfixe `/messages` : c'est le seul préfixe que l'API Gateway mappe vers ce
// service. Tout est authentifié (pas de messagerie publique). `jwtSecret` valide
// localement les tokens émis par auth-service (secret partagé).
func RegisterRoutes(
	r *gin.Engine,
	serviceName string,
	svc *service.MessageService,
	hub *realtime.Hub,
	jwtSecret string,
	allowedOrigins []string,
) {
	auth := middleware.JWTAuth(jwtSecret)

	r.GET("/health", Health(serviceName))

	keyH := NewKeyHandler(svc)
	convH := NewConversationHandler(svc, hub)
	wsH := NewWSHandler(hub, jwtSecret, allowedOrigins)

	messages := r.Group("/messages")
	{
		// Temps réel : la poignée de main porte le token en query param (le
		// navigateur n'autorise pas d'en-tête sur un upgrade WS) → pas de
		// middleware `auth`, validation interne au handler.
		messages.GET("/ws", wsH.Connect)

		// Registre de clés publiques (E2EE).
		messages.PUT("/keys", auth, keyH.PublishKey)
		messages.GET("/keys/:userId", auth, keyH.GetKey)

		// Annuaire public des communautés (découverte + recherche).
		messages.GET("/communities", auth, convH.ListCommunities)

		// Compteur de conversations non lues (badge app-wide).
		messages.GET("/unread-count", auth, convH.UnreadCount)

		// Effacement RGPD : purge la participation d'un utilisateur (admin).
		messages.DELETE("/users/:id", auth, middleware.AdminOnly(), convH.PurgeUser)

		// Conversations + messages.
		conversations := messages.Group("/conversations", auth)
		{
			conversations.GET("", convH.ListConversations)
			conversations.POST("", convH.CreateConversation)

			conv := conversations.Group("/:id")
			{
				conv.GET("", convH.GetConversation)
				conv.PATCH("", convH.UpdateConversation)  // renommer (owner)
				conv.DELETE("", convH.DeleteConversation) // supprimer (owner)

				conv.POST("/join", convH.JoinCommunity) // rejoindre une communauté (viewer)

				// État par-utilisateur (membre requis, pas de diffusion).
				conv.PATCH("/pin", convH.PinConversation)      // épingler
				conv.DELETE("/pin", convH.UnpinConversation)   // désépingler
				conv.PATCH("/mute", convH.MuteConversation)    // mettre en sourdine
				conv.DELETE("/mute", convH.UnmuteConversation) // réactiver
				conv.DELETE("/me", convH.ClearConversation)    // supprimer côté user
				conv.PUT("/read", convH.MarkRead)              // marquer lu (curseur de lecture)

				conv.GET("/messages", convH.ListMessages)
				conv.POST("/messages", convH.SendMessage)
				conv.PATCH("/messages/:messageId", convH.EditMessage)

				members := conv.Group("/members")
				{
					members.GET("", convH.ListMembers)
					members.POST("", convH.AddMember)              // inviter un groupe (tout membre)
					members.PATCH("/:userId", convH.SetMemberRole) // promouvoir/rétrograder (owner, communauté)
					members.DELETE("/:userId", convH.RemoveMember) // exclure (owner) / quitter (soi)
				}
			}
		}
	}
}
