package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/webdad/message-service/internal/middleware"
	"github.com/webdad/message-service/internal/models"
	"github.com/webdad/message-service/internal/realtime"
	"github.com/webdad/message-service/internal/service"
)

// ConversationHandler : conversations + messages (le hub sert la diffusion WS).
type ConversationHandler struct {
	service *service.MessageService
	hub     *realtime.Hub
}

func NewConversationHandler(svc *service.MessageService, hub *realtime.Hub) *ConversationHandler {
	return &ConversationHandler{service: svc, hub: hub}
}

// CreateConversation : POST /messages/conversations — crée (ou retrouve) une
// conversation. `type:"group"` → groupe ; sinon (défaut) → DM. Le client a
// généré la clé de contenu et fournit les enveloppes par membre.
func (h *ConversationHandler) CreateConversation(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	var req models.CreateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}

	var (
		view *models.ConversationView
		err  error
	)
	switch req.Type {
	case models.TypeGroup:
		view, err = h.service.CreateGroup(c.Request.Context(), claims.UserID, req.Title, req.TitleNonce, req.Envelopes)
	case models.TypeCommunity:
		view, err = h.service.CreateCommunity(c.Request.Context(), claims.UserID, req.Title, req.ContentKey)
	default:
		view, err = h.service.CreateDM(c.Request.Context(), claims.UserID, req.PeerID, req.Envelopes)
	}
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": view})
}

// ListCommunities : GET /messages/communities — annuaire public (nom en clair,
// nb de membres, déjà-membre). Pagination + recherche `?q=`. Jamais la clé.
func (h *ConversationHandler) ListCommunities(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	items, err := h.service.ListCommunities(
		c.Request.Context(), claims.UserID, pageLimit(c), pageOffset(c), c.Query("q"),
	)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

// JoinCommunity : POST /messages/conversations/:id/join — auto-join en viewer ;
// renvoie la vue AVEC la clé de contenu (remise par le serveur).
func (h *ConversationHandler) JoinCommunity(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	view, notify, err := h.service.JoinCommunity(c.Request.Context(), c.Param("id"), claims.UserID)
	if err != nil {
		respondError(c, err)
		return
	}
	if len(notify) > 0 {
		h.hub.Publish(notify, gin.H{"type": "member_added", "data": gin.H{
			"conversation_id": c.Param("id"), "user_id": claims.UserID,
		}})
	}
	c.JSON(http.StatusOK, gin.H{"data": view})
}

// SetMemberRole : PATCH /messages/conversations/:id/members/:userId — promouvoir
// / rétrograder talker↔viewer dans une communauté (owner uniquement).
func (h *ConversationHandler) SetMemberRole(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	var req models.SetMemberRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}

	notify, err := h.service.SetMemberRole(c.Request.Context(), c.Param("id"), claims.UserID, c.Param("userId"), req.Role)
	if err != nil {
		respondError(c, err)
		return
	}
	h.hub.Publish(notify, gin.H{"type": "member_role_changed", "data": gin.H{
		"conversation_id": c.Param("id"), "user_id": c.Param("userId"), "role": req.Role,
	}})
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"user_id": c.Param("userId"), "role": req.Role}})
}

// UpdateConversation : PATCH /messages/conversations/:id — renomme un groupe
// (owner uniquement, nom re-chiffré côté client).
func (h *ConversationHandler) UpdateConversation(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	var req models.UpdateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}

	view, err := h.service.UpdateGroup(c.Request.Context(), c.Param("id"), claims.UserID, req.Title, req.TitleNonce)
	if err != nil {
		respondError(c, err)
		return
	}
	h.hub.Publish(view.MemberIDs, gin.H{"type": "conversation_updated", "data": view})
	c.JSON(http.StatusOK, gin.H{"data": view})
}

// DeleteConversation : DELETE /messages/conversations/:id — supprime un groupe
// (owner uniquement) + cascade membres/messages.
func (h *ConversationHandler) DeleteConversation(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	convID := c.Param("id")
	notify, err := h.service.DeleteGroup(c.Request.Context(), convID, claims.UserID)
	if err != nil {
		respondError(c, err)
		return
	}
	h.hub.Publish(notify, gin.H{"type": "conversation_deleted", "data": gin.H{"id": convID}})
	c.Status(http.StatusNoContent)
}

// PinConversation : PATCH /messages/conversations/:id/pin — épingle la
// conversation en tête de MA liste (état par-utilisateur, pas de diffusion).
func (h *ConversationHandler) PinConversation(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	view, err := h.service.PinConversation(c.Request.Context(), c.Param("id"), claims.UserID, true)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": view})
}

// UnpinConversation : DELETE /messages/conversations/:id/pin — retire l'épinglage.
func (h *ConversationHandler) UnpinConversation(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	view, err := h.service.PinConversation(c.Request.Context(), c.Param("id"), claims.UserID, false)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": view})
}

// MuteConversation : PATCH /messages/conversations/:id/mute — met en sourdine.
func (h *ConversationHandler) MuteConversation(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	view, err := h.service.MuteConversation(c.Request.Context(), c.Param("id"), claims.UserID, true)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": view})
}

// UnmuteConversation : DELETE /messages/conversations/:id/mute — réactive.
func (h *ConversationHandler) UnmuteConversation(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	view, err := h.service.MuteConversation(c.Request.Context(), c.Param("id"), claims.UserID, false)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": view})
}

// ClearConversation : DELETE /messages/conversations/:id/me — « supprime » la
// conversation côté user (masque + coupe l'historique). N'affecte pas les autres.
func (h *ConversationHandler) ClearConversation(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	if err := h.service.ClearConversation(c.Request.Context(), c.Param("id"), claims.UserID); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// MarkRead : PUT /messages/conversations/:id/read — avance le curseur de lecture
// du membre courant (la conversation est lue jusqu'à maintenant). Personnel,
// pas de diffusion WS.
func (h *ConversationHandler) MarkRead(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	if err := h.service.MarkRead(c.Request.Context(), c.Param("id"), claims.UserID); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// UnreadCount : GET /messages/unread-count — nombre de conversations ayant au
// moins un message non lu (badge app-wide). Calcul serveur, sans lire le contenu.
func (h *ConversationHandler) UnreadCount(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	count, err := h.service.UnreadCount(c.Request.Context(), claims.UserID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"count": count}})
}

// ListMembers : GET /messages/conversations/:id/members — membres (id + rôle),
// sans les enveloppes des autres (membre requis).
func (h *ConversationHandler) ListMembers(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	members, err := h.service.ListMembers(c.Request.Context(), c.Param("id"), claims.UserID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": members})
}

// AddMember : POST /messages/conversations/:id/members — invite (tout membre).
func (h *ConversationHandler) AddMember(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	var req models.AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}

	notify, err := h.service.AddMember(c.Request.Context(), c.Param("id"), claims.UserID, req.UserID, req.Envelope)
	if err != nil {
		respondError(c, err)
		return
	}
	h.hub.Publish(notify, gin.H{"type": "member_added", "data": gin.H{
		"conversation_id": c.Param("id"), "user_id": req.UserID,
	}})
	c.JSON(http.StatusCreated, gin.H{"data": gin.H{"user_id": req.UserID, "role": models.MemberTalker}})
}

// RemoveMember : DELETE /messages/conversations/:id/members/:userId — exclure
// (owner) ou quitter (soi-même). `:userId` = "me" cible l'utilisateur courant.
func (h *ConversationHandler) RemoveMember(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	targetID := c.Param("userId")
	if targetID == "me" {
		targetID = claims.UserID
	}

	notify, err := h.service.RemoveMember(c.Request.Context(), c.Param("id"), claims.UserID, targetID)
	if err != nil {
		respondError(c, err)
		return
	}
	h.hub.Publish(notify, gin.H{"type": "member_removed", "data": gin.H{
		"conversation_id": c.Param("id"), "user_id": targetID,
	}})
	c.Status(http.StatusNoContent)
}

// ListConversations : GET /messages/conversations — mes conversations (avec mon
// enveloppe + mon rôle), triées par activité.
func (h *ConversationHandler) ListConversations(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	views, err := h.service.ListConversations(c.Request.Context(), claims.UserID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": views})
}

// GetConversation : GET /messages/conversations/:id — détail (membre requis).
func (h *ConversationHandler) GetConversation(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	view, err := h.service.GetConversation(c.Request.Context(), c.Param("id"), claims.UserID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": view})
}

// ListMessages : GET /messages/conversations/:id/messages — historique chiffré
// (membre requis). ?before=<messageId> pagine vers le haut ; ?limit borne la page.
func (h *ConversationHandler) ListMessages(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	msgs, err := h.service.ListMessages(
		c.Request.Context(), c.Param("id"), claims.UserID, pageLimit(c), c.Query("before"),
	)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": msgs})
}

// SendMessage : POST /messages/conversations/:id/messages — poste un message
// DÉJÀ chiffré (membre + droit d'écriture requis), puis le diffuse en temps réel
// aux membres connectés.
func (h *ConversationHandler) SendMessage(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	var req models.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}

	msg, memberIDs, err := h.service.SendMessage(
		c.Request.Context(), c.Param("id"), claims.UserID, req.Ciphertext, req.Nonce,
	)
	if err != nil {
		respondError(c, err)
		return
	}

	// Diffusion temps réel (le payload reste chiffré ; le serveur ne lit rien).
	h.hub.Publish(memberIDs, gin.H{"type": "message", "data": msg})

	c.JSON(http.StatusCreated, gin.H{"data": msg})
}

// pageLimit lit ?limit (défaut/borne appliqués côté service).
func pageLimit(c *gin.Context) int64 {
	n, err := strconv.ParseInt(c.Query("limit"), 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// pageOffset lit ?offset (défaut 0).
func pageOffset(c *gin.Context) int64 {
	n, err := strconv.ParseInt(c.Query("offset"), 10, 64)
	if err != nil {
		return 0
	}
	return n
}
