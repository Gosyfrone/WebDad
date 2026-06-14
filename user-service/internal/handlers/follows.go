package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/webdad/user-service/internal/logging"
	"github.com/webdad/user-service/internal/middleware"
)

// Follow : POST /users/:id/follow — l'utilisateur authentifié suit `:id`.
// @Summary     Suivre un utilisateur
// @Tags        users
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "ID de l'utilisateur à suivre"
// @Success     200 {object} map[string]string "status: following|requested"
// @Failure     400 {object} map[string]string
// @Failure     401 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /users/{id}/follow [post]
func (h *Handler) Follow(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	status, err := h.users.Follow(c.Request.Context(), claims.UserID, claims.Email, c.Param("id"))
	if err != nil {
		respondUserError(c, err)
		return
	}
	logging.FromGin(c).Info("follow", "target_id", c.Param("id"), "status", status)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"status": status}})
}

// Unfollow : DELETE /users/:id/follow — l'utilisateur authentifié ne suit plus `:id`.
// @Summary     Ne plus suivre un utilisateur
// @Tags        users
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "ID de l'utilisateur"
// @Success     204
// @Failure     401 {object} map[string]string
// @Router      /users/{id}/follow [delete]
func (h *Handler) Unfollow(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	if err := h.users.Unfollow(claims.UserID, c.Param("id")); err != nil {
		respondUserError(c, err)
		return
	}
	logging.FromGin(c).Info("unfollow", "target_id", c.Param("id"))
	c.Status(http.StatusNoContent)
}

// RemoveFollower : DELETE /users/me/followers/:id — retire `:id` de mes abonnés.
// @Summary     Retirer un abonné de sa liste
// @Tags        users
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "ID de l'abonné à retirer"
// @Success     204
// @Failure     401 {object} map[string]string
// @Router      /users/me/followers/{id} [delete]
func (h *Handler) RemoveFollower(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	if err := h.users.RemoveFollower(claims.UserID, c.Param("id")); err != nil {
		respondUserError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Followers : GET /users/:id/followers — abonnés de `:id` (public, paginé).
// @Summary     Lister les abonnés d'un utilisateur
// @Tags        users
// @Produce     json
// @Param       id     path  string true  "User ID"
// @Param       limit  query int    false "Nb résultats"
// @Param       offset query int    false "Décalage"
// @Success     200 {array} models.User
// @Failure     404 {object} map[string]string
// @Router      /users/{id}/followers [get]
func (h *Handler) Followers(c *gin.Context) {
	limit, offset := paginate(c)
	users, err := h.users.ListFollowers(c.Param("id"), limit, offset)
	if err != nil {
		respondUserError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": users})
}

// Following : GET /users/:id/following — abonnements de `:id` (public, paginé).
// @Summary     Lister les abonnements d'un utilisateur
// @Tags        users
// @Produce     json
// @Param       id     path  string true  "User ID"
// @Param       limit  query int    false "Nb résultats"
// @Param       offset query int    false "Décalage"
// @Success     200 {array} models.User
// @Failure     404 {object} map[string]string
// @Router      /users/{id}/following [get]
func (h *Handler) Following(c *gin.Context) {
	limit, offset := paginate(c)
	users, err := h.users.ListFollowing(c.Param("id"), limit, offset)
	if err != nil {
		respondUserError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": users})
}

func (h *Handler) IsFollowing(c *gin.Context) {
	userId := c.Param("userId")
	followingId := c.Param("followingId")
	isFollowing := h.users.IsFollowing(userId, followingId)

	c.JSON(http.StatusOK, gin.H{"isFollowing": isFollowing})
}

// AcceptFollowRequest : POST /users/follow-requests/:followerId/accept
// @Summary     Accepter une demande d'abonnement
// @Tags        users
// @Produce     json
// @Security    BearerAuth
// @Param       followerId path string true "ID du demandeur"
// @Success     200 {object} map[string]string "status: accepted"
// @Failure     401 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /users/follow-requests/{followerId}/accept [post]
func (h *Handler) AcceptFollowRequest(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}
	if err := h.users.AcceptFollowRequest(claims.UserID, c.Param("followerId")); err != nil {
		respondUserError(c, err)
		return
	}
	logging.FromGin(c).Info("demande de suivi acceptée", "follower_id", c.Param("followerId"))
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"status": "accepted"}})
}

// RejectFollowRequest : POST /users/follow-requests/:followerId/reject
// @Summary     Rejeter une demande d'abonnement
// @Tags        users
// @Produce     json
// @Security    BearerAuth
// @Param       followerId path string true "ID du demandeur"
// @Success     200 {object} map[string]string "status: rejected"
// @Failure     401 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /users/follow-requests/{followerId}/reject [post]
func (h *Handler) RejectFollowRequest(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}
	if err := h.users.RejectFollowRequest(claims.UserID, c.Param("followerId")); err != nil {
		respondUserError(c, err)
		return
	}
	logging.FromGin(c).Info("demande de suivi rejetée", "follower_id", c.Param("followerId"))
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"status": "rejected"}})
}

// PendingFollowRequests : GET /users/me/follow-requests/outgoing
// @Summary     Demandes d'abonnement sortantes (envoyées par moi)
// @Tags        users
// @Produce     json
// @Security    BearerAuth
// @Success     200 {array} string "Liste d'IDs"
// @Failure     401 {object} map[string]string
// @Router      /users/me/follow-requests/outgoing [get]
func (h *Handler) PendingFollowRequests(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}
	ids, err := h.users.PendingFollowRequestIDs(claims.UserID)
	if err != nil {
		respondUserError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": ids})
}
