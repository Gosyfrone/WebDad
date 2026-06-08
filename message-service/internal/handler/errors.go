package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/webdad/message-service/internal/service"
)

// respondError mappe les erreurs métier du service vers des codes HTTP.
func respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrConversationNotFound), errors.Is(err, service.ErrKeyNotFound),
		errors.Is(err, service.ErrTargetNotMember):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrInvalidID), errors.Is(err, service.ErrSelfConversation),
		errors.Is(err, service.ErrMissingEnvelope), errors.Is(err, service.ErrInvalidGroup):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrNotMember), errors.Is(err, service.ErrCannotWrite),
		errors.Is(err, service.ErrOwnerOnly), errors.Is(err, service.ErrOwnerCannotLeave):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrNotGroup), errors.Is(err, service.ErrAlreadyMember),
		errors.Is(err, service.ErrGroupFull):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur interne"})
	}
}
