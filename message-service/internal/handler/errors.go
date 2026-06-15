package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/webdad/message-service/internal/logging"
	"github.com/webdad/message-service/internal/service"
)

// respondError mappe les erreurs métier du service vers des codes HTTP.
func respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrConversationNotFound), errors.Is(err, service.ErrKeyNotFound),
		errors.Is(err, service.ErrBackupNotFound),
		errors.Is(err, service.ErrMessageNotFound),
		errors.Is(err, service.ErrTargetNotMember):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrInvalidID), errors.Is(err, service.ErrSelfConversation),
		errors.Is(err, service.ErrMissingEnvelope), errors.Is(err, service.ErrInvalidGroup),
		errors.Is(err, service.ErrInvalidCommunity), errors.Is(err, service.ErrInvalidRole):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrNotMember), errors.Is(err, service.ErrCannotWrite),
		errors.Is(err, service.ErrOwnerOnly), errors.Is(err, service.ErrOwnerCannotLeave),
		errors.Is(err, service.ErrNotMessageOwner):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrNotGroup), errors.Is(err, service.ErrNotCommunity),
		errors.Is(err, service.ErrNotManageable), errors.Is(err, service.ErrAlreadyMember),
		errors.Is(err, service.ErrTalkersFull):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		logging.FromGin(c).Error("erreur messagerie inattendue", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur interne"})
	}
}
