// Package middleware contient les middlewares Gin du message-service.
package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// contextKey : clé sous laquelle les claims sont stockés dans le contexte Gin.
const contextKey = "claims"

// Claims : contenu du JWT, identique à celui émis par auth-service. Le service
// ne signe pas de token, il les VALIDE avec le même JWT_SECRET partagé.
type Claims struct {
	UserID        string `json:"user_id"`
	Email         string `json:"email"`
	Role          string `json:"role"`
	EmailVerified bool   `json:"email_verified"`
	jwt.RegisteredClaims
}

// JWTAuth valide le bearer token de l'en-tête Authorization et injecte les
// claims dans le contexte Gin. Stoppe la requête (401) si le token est absent,
// mal formé, invalide ou expiré.
func JWTAuth(secret string) gin.HandlerFunc {
	key := []byte(secret)
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

		claims, err := ParseToken(parts[1], key)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token invalide ou expiré"})
			return
		}

		c.Set(contextKey, claims)
		c.Set("user_id", claims.UserID)
		c.Next()
	}
}

// AdminOnly exige un JWT dont le rôle est `admin`. À chaîner APRÈS JWTAuth.
// Garde de l'effacement RGPD (purge de la participation d'un utilisateur).
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := ClaimsFrom(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token manquant"})
			return
		}
		if claims.Role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "réservé aux administrateurs"})
			return
		}
		c.Next()
	}
}

// ModeratorOnly exige un JWT dont le rôle est `moderator` ou `admin` (l'admin
// est un sur-ensemble). À chaîner APRÈS JWTAuth. Garde la suppression de message
// par la modération de plateforme (par id de message, hors appartenance).
func ModeratorOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := ClaimsFrom(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token manquant"})
			return
		}
		if claims.Role != "moderator" && claims.Role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "réservé à la modération"})
			return
		}
		c.Next()
	}
}

// VerifiedOnly exige une adresse e-mail vérifiée (claim email_verified). À
// chaîner APRÈS JWTAuth. Empêche un compte non vérifié (RIV-002) de créer des
// conversations ou d'envoyer des messages. 403 sinon.
func VerifiedOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := ClaimsFrom(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token manquant"})
			return
		}
		if !claims.EmailVerified {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "adresse e-mail non vérifiée",
				"code":  "email_not_verified",
			})
			return
		}
		c.Next()
	}
}

// ClaimsFrom récupère les claims posés par JWTAuth dans le contexte.
func ClaimsFrom(c *gin.Context) (*Claims, bool) {
	val, exists := c.Get(contextKey)
	if !exists {
		return nil, false
	}
	claims, ok := val.(*Claims)
	return claims, ok
}

// ParseToken valide la signature (HS256) et l'expiration, puis retourne les
// claims. Exporté car la poignée de main WebSocket ne peut pas porter d'en-tête
// Authorization (le navigateur ne le permet pas) : le token y transite en query
// param et est validé directement via cette fonction.
func ParseToken(tokenStr string, key []byte) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("méthode de signature inattendue : %v", t.Header["alg"])
		}
		return key, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("token invalide")
	}
	return claims, nil
}
