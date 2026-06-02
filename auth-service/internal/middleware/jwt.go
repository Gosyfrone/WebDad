// Package middleware contient les middlewares Gin du service auth.
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/webdad/auth-service/internal/services"
)

// JWTAuth valide le bearer token de l'en-tête Authorization et injecte
// les claims dans le contexte Gin (`c.Set("claims", ...)`) pour les
// handlers en aval. Stoppe la requête (401) si le token est absent,
// mal formé, invalide ou expiré.
func JWTAuth(auth *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token manquant"})
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "format Authorization invalide (attendu : Bearer <token>)",
			})
			return
		}

		claims, err := auth.ParseToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token invalide ou expiré"})
			return
		}

		c.Set("claims", claims)
		c.Next()
	}
}
