package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/webdad/report-service/internal/middleware"
	"github.com/webdad/report-service/internal/repository"
	"github.com/webdad/report-service/internal/service"
)

// ReportHandler sert l'API du report-service (signalements + tickets + warns).
type ReportHandler struct {
	service *service.ReportService
}

func NewReportHandler(svc *service.ReportService) *ReportHandler {
	return &ReportHandler{service: svc}
}

// ── Signalement (tout utilisateur authentifié) ──────────────────────────────

type createReportRequest struct {
	Category         string `json:"category" binding:"required"`
	EntityType       string `json:"entity_type"`
	EntityID         string `json:"entity_id"`
	EntityOwnerID    string `json:"entity_owner_id"`
	Reason           string `json:"reason" binding:"required"`
	Text             string `json:"text"`
	DisclosedContent string `json:"disclosed_content"`
	AttachmentID     string `json:"attachment_id"`
}

// Create : POST /reports — dépose un signalement (modération agrégée par entité,
// ou bug autonome). Valide le motif, l'entité et la longueur du texte.
// @Summary  Déposer un signalement
// @Tags     reports
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    body body createReportRequest true "Signalement"
// @Success  201 {object} map[string]interface{} "data: ticket"
// @Failure  400 {object} map[string]string
// @Failure  401 {object} map[string]string
// @Failure  409 {object} map[string]string "déjà signalé par cet utilisateur"
// @Router   /reports [post]
func (h *ReportHandler) Create(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}
	var req createReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "corps invalide"})
		return
	}
	ticket, err := h.service.CreateReport(c.Request.Context(), claims.UserID, service.CreateReportInput{
		Category:         req.Category,
		EntityType:       req.EntityType,
		EntityID:         req.EntityID,
		EntityOwnerID:    req.EntityOwnerID,
		Reason:           req.Reason,
		Text:             req.Text,
		DisclosedContent: req.DisclosedContent,
		AttachmentID:     req.AttachmentID,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": ticket})
}

// ── Tickets (modération + admin) ─────────────────────────────────────────────

// ListTickets : GET /reports/tickets — liste filtrée et triée par volume.
// @Summary  Lister les tickets (filtres cumulables)
// @Tags     reports
// @Produce  json
// @Security BearerAuth
// @Param    category    query string false "moderation|bug"
// @Param    status      query string false "open|closed|reopened"
// @Param    min_reports query int    false "Nb minimal de signalements"
// @Param    since       query string false "RFC3339 — borne basse du dernier signalement"
// @Param    until       query string false "RFC3339 — borne haute du dernier signalement"
// @Param    limit       query int    false "Nb résultats"
// @Success  200 {object} map[string]interface{} "data: [tickets]"
// @Failure  401 {object} map[string]string
// @Router   /reports/tickets [get]
func (h *ReportHandler) ListTickets(c *gin.Context) {
	f := repository.TicketFilter{
		Category: c.Query("category"),
		Status:   c.Query("status"),
	}
	if n, err := strconv.Atoi(c.Query("min_reports")); err == nil && n > 0 {
		f.MinReports = n
	}
	if ts := parseTime(c.Query("since")); ts != nil {
		f.Since = ts
	}
	if ts := parseTime(c.Query("until")); ts != nil {
		f.Until = ts
	}
	limit, _ := strconv.ParseInt(c.Query("limit"), 10, 64)

	tickets, err := h.service.ListTickets(c.Request.Context(), f, limit)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": tickets})
}

// GetTicket : GET /reports/tickets/:id — détail (fil enfant + actions).
// @Summary  Détail d'un ticket
// @Tags     reports
// @Produce  json
// @Security BearerAuth
// @Param    id path string true "Ticket ID"
// @Success  200 {object} map[string]interface{} "data: ticket"
// @Failure  404 {object} map[string]string
// @Router   /reports/tickets/{id} [get]
func (h *ReportHandler) GetTicket(c *gin.Context) {
	ticket, err := h.service.GetTicket(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": ticket})
}

type replyRequest struct {
	Text string `json:"text" binding:"required"`
}

// Reply : POST /reports/tickets/:id/replies — réponse interne (trace le mod).
// @Summary  Réponse interne d'un modérateur
// @Tags     reports
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    id   path string       true "Ticket ID"
// @Param    body body replyRequest true "Réponse"
// @Success  200 {object} map[string]interface{} "data: ticket"
// @Failure  400 {object} map[string]string
// @Router   /reports/tickets/{id}/replies [post]
func (h *ReportHandler) Reply(c *gin.Context) {
	claims, _ := middleware.ClaimsFrom(c)
	var req replyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "corps invalide"})
		return
	}
	ticket, err := h.service.Reply(c.Request.Context(), c.Param("id"), claims.UserID, req.Text)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": ticket})
}

type statusRequest struct {
	Status string `json:"status" binding:"required"`
}

// ChangeStatus : PATCH /reports/tickets/:id/status — open|closed|reopened.
// @Summary  Changer le statut d'un ticket
// @Tags     reports
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    id   path string        true "Ticket ID"
// @Param    body body statusRequest true "Nouveau statut"
// @Success  200 {object} map[string]interface{} "data: ticket"
// @Failure  400 {object} map[string]string
// @Router   /reports/tickets/{id}/status [patch]
func (h *ReportHandler) ChangeStatus(c *gin.Context) {
	claims, _ := middleware.ClaimsFrom(c)
	var req statusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "corps invalide"})
		return
	}
	ticket, err := h.service.ChangeStatus(c.Request.Context(), c.Param("id"), claims.UserID, req.Status)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": ticket})
}

// RecordRemoval : POST /reports/tickets/:id/removal — journalise le retrait du
// contenu signalé par la modération (sans changer le statut du ticket).
// @Summary  Journaliser le retrait du contenu
// @Tags     reports
// @Produce  json
// @Security BearerAuth
// @Param    id path string true "Ticket ID"
// @Success  200 {object} map[string]interface{} "data: ticket"
// @Failure  404 {object} map[string]string
// @Router   /reports/tickets/{id}/removal [post]
func (h *ReportHandler) RecordRemoval(c *gin.Context) {
	claims, _ := middleware.ClaimsFrom(c)
	ticket, err := h.service.LogRemoval(c.Request.Context(), c.Param("id"), claims.UserID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": ticket})
}

// Transfer : POST /reports/tickets/:id/transfer — bug → modération (admin).
// @Summary  Transférer un ticket bug vers la modération
// @Tags     reports
// @Produce  json
// @Security BearerAuth
// @Param    id path string true "Ticket ID"
// @Success  200 {object} map[string]interface{} "data: ticket"
// @Failure  409 {object} map[string]string
// @Router   /reports/tickets/{id}/transfer [post]
func (h *ReportHandler) Transfer(c *gin.Context) {
	claims, _ := middleware.ClaimsFrom(c)
	ticket, err := h.service.Transfer(c.Request.Context(), c.Param("id"), claims.UserID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": ticket})
}

// ── Avertissements (warns) ───────────────────────────────────────────────────

type warningRequest struct {
	TargetUserID string `json:"target_user_id" binding:"required"`
	TicketID     string `json:"ticket_id"`
	Message      string `json:"message" binding:"required"`
}

// IssueWarning : POST /reports/warnings — émet un avertissement (mod/admin).
// @Summary  Émettre un avertissement
// @Tags     reports
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    body body warningRequest true "Avertissement"
// @Success  201 {object} map[string]interface{} "data: warning"
// @Failure  400 {object} map[string]string
// @Router   /reports/warnings [post]
func (h *ReportHandler) IssueWarning(c *gin.Context) {
	claims, _ := middleware.ClaimsFrom(c)
	var req warningRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "corps invalide"})
		return
	}
	w, err := h.service.IssueWarning(c.Request.Context(), claims.UserID, req.TargetUserID, req.TicketID, req.Message)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": w})
}

// PendingWarnings : GET /reports/warnings/pending — warns non acquittés du compte.
// @Summary  Mes avertissements en attente
// @Tags     reports
// @Produce  json
// @Security BearerAuth
// @Success  200 {object} map[string]interface{} "data: [warnings]"
// @Failure  401 {object} map[string]string
// @Router   /reports/warnings/pending [get]
func (h *ReportHandler) PendingWarnings(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}
	warns, err := h.service.PendingWarnings(c.Request.Context(), claims.UserID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": warns})
}

// AckWarning : POST /reports/warnings/:id/ack — acquitte un avertissement.
// @Summary  Acquitter un avertissement
// @Tags     reports
// @Produce  json
// @Security BearerAuth
// @Param    id path string true "Warning ID"
// @Success  204
// @Failure  404 {object} map[string]string
// @Router   /reports/warnings/{id}/ack [post]
func (h *ReportHandler) AckWarning(c *gin.Context) {
	claims, _ := middleware.ClaimsFrom(c)
	if err := h.service.AckWarning(c.Request.Context(), c.Param("id"), claims.UserID); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// parseTime lit une date RFC3339 (filtres temporels). nil si vide/invalide.
func parseTime(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil
	}
	return &t
}
