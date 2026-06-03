// Package middleware contient les middlewares Gin du service auth.
package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

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
			// On distingue l'expiration (cas nominal : le front déclenche un
			// refresh) des autres erreurs (token malformé/signature invalide :
			// pas de refresh, on reste en 401). Le `code` guide le front.
			if errors.Is(err, jwt.ErrTokenExpired) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "token expiré",
					"code":  "token_expired",
				})
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token invalide"})
			return
		}

		c.Set("claims", claims)
		c.Next()
	}
}
