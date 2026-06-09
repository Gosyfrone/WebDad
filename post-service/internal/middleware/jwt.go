// Package middleware contient les middlewares Gin du service post.
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

// Claims : contenu du JWT, identique à celui émis par auth-service.
// post-service ne signe pas de token, il se contente de les VALIDER avec le
// même JWT_SECRET partagé.
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// JWTAuth valide le bearer token de l'en-tête Authorization et injecte les
// claims dans le contexte Gin. Stoppe la requête (401) si le token est
// absent, mal formé, invalide ou expiré.
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

		claims, err := parseToken(parts[1], key)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token invalide ou expiré"})
			return
		}

		c.Set(contextKey, claims)
		c.Next()
	}
}

// OptionalJWTAuth valide le bearer token s'il est présent, sans rendre la route
// protégée. Utile pour les lectures publiques qui peuvent exposer un peu plus
// de contenu à l'utilisateur connecté (ex. profils privés suivis).
func OptionalJWTAuth(secret string) gin.HandlerFunc {
	key := []byte(secret)
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.Next()
			return
		}

		claims, err := parseToken(parts[1], key)
		if err == nil {
			c.Set(contextKey, claims)
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

// parseToken valide la signature (HS256) et l'expiration, puis retourne les claims.
func parseToken(tokenStr string, key []byte) (*Claims, error) {
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
