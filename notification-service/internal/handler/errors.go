package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/webdad/notification-service/internal/service"
)

// respondError mappe les erreurs métier du service vers des codes HTTP.
func respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrInvalidID):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur interne"})
	}
}
