package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/webdad/user-service/internal/middleware"
)

// Follow : POST /users/:id/follow — l'utilisateur authentifié suit `:id`.
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
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"status": status}})
}

// Unfollow : DELETE /users/:id/follow — l'utilisateur authentifié ne suit plus `:id`.
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
	c.Status(http.StatusNoContent)
}

// RemoveFollower : DELETE /users/me/followers/:id — retire `:id` de mes abonnés.
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
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"status": "accepted"}})
}

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
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"status": "rejected"}})
}

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
