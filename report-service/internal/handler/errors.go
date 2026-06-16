package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/webdad/report-service/internal/logging"
	"github.com/webdad/report-service/internal/service"
)

// respondError mappe les erreurs métier du service vers des codes HTTP.
func respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrInvalidID), errors.Is(err, service.ErrValidation):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrNotBug), errors.Is(err, service.ErrAlreadyReported):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		logging.FromGin(c).Error("erreur report inattendue", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur interne"})
	}
}
