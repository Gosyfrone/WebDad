package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/webdad/report-service/internal/middleware"
	"github.com/webdad/report-service/internal/service"
)

// RegisterRoutes enregistre les routes du report-service sous `/reports` (seul
// préfixe que l'API Gateway mappe vers ce service).
//
// Trois niveaux d'accès :
//   - tout utilisateur authentifié : déposer un signalement, lire/acquitter SES
//     avertissements ;
//   - modérateur OU admin : lister/consulter les tickets, répondre, changer le
//     statut, émettre un avertissement ;
//   - admin seul : transférer un ticket de bug vers la modération.
func RegisterRoutes(r *gin.Engine, serviceName string, svc *service.ReportService, jwtSecret string) {
	auth := middleware.JWTAuth(jwtSecret)
	mod := middleware.ModeratorOnly()
	admin := middleware.AdminOnly()

	r.GET("/health", Health(serviceName))

	h := NewReportHandler(svc)

	reports := r.Group("/reports", auth)
	{
		// Dépôt de signalement — ouvert à tout utilisateur authentifié.
		reports.POST("", h.Create)

		// Avertissements de l'utilisateur courant (interception côté front).
		reports.GET("/warnings/pending", h.PendingWarnings)
		reports.POST("/warnings/:id/ack", h.AckWarning)

		// Émission d'un avertissement — modération.
		reports.POST("/warnings", mod, h.IssueWarning)

		// Configuration de la modération : lecture mod/admin, écriture admin seul.
		reports.GET("/settings", mod, h.GetSettings)
		reports.PATCH("/settings", admin, h.UpdateSettings)

		// Tickets — modération.
		reports.GET("/tickets", mod, h.ListTickets)
		reports.GET("/tickets/:id", mod, h.GetTicket)
		reports.POST("/tickets/:id/replies", mod, h.Reply)
		reports.PATCH("/tickets/:id/status", mod, h.ChangeStatus)
		reports.POST("/tickets/:id/removal", mod, h.RecordRemoval)
		// Validation « entité conforme » (terminal : démasque + verrouille).
		reports.POST("/tickets/:id/approve", mod, h.Approve)

		// Transfert bug → modération — gouvernance admin.
		reports.POST("/tickets/:id/transfer", admin, h.Transfer)
	}
}
