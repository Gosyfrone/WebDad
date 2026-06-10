// Package middleware : middlewares du gateway (CORS, garde JWT admin).
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// claims : sous-ensemble du JWT émis par auth-service (même secret partagé).
type claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// AdminJWT valide le bearer token (HS256, secret partagé avec auth-service) et
// exige le rôle `admin`. Garde des routes d'administration servies DIRECTEMENT
// par le gateway (ex. monitoring), distinctes du reverse proxy. 401 si le token
// est absent/invalide/expiré, 403 si le rôle n'est pas admin. Fail-closed : un
// secret vide ne validera aucun token légitime → accès refusé.
func AdminJWT(secret string) gin.HandlerFunc {
	key := []byte(secret)
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token manquant"})
			return
		}
		cl := &claims{}
		token, err := jwt.ParseWithClaims(parts[1], cl, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return key, nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token invalide ou expiré"})
			return
		}
		if cl.Role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "réservé aux administrateurs"})
			return
		}
		c.Next()
	}
}
