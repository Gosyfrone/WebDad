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
	UserID        string `json:"user_id"`
	Email         string `json:"email"`
	Role          string `json:"role"`
	EmailVerified bool   `json:"email_verified"`
	jwt.RegisteredClaims
}

// En-têtes d'identité posés par le gateway après validation du JWT, à
// destination des services backend. Ils ne sont JAMAIS recopiés depuis la
// requête entrante (cf. stripIdentityHeaders) : seul le gateway peut les
// renseigner, sinon un client les usurperait trivialement.
const (
	HeaderUserID        = "X-User-Id"
	HeaderUserRole      = "X-User-Role"
	HeaderEmailVerified = "X-Email-Verified"
)

// parse valide la signature (HS256) et l'expiration d'un bearer token et
// retourne ses claims. Fail-closed : un secret vide ne validera aucun token.
func parse(tokenStr string, key []byte) (*claims, error) {
	cl := &claims{}
	token, err := jwt.ParseWithClaims(tokenStr, cl, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return key, nil
	})
	if err != nil || !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return cl, nil
}

// bearerToken extrait le token brut de l'en-tête `Authorization: Bearer <t>`.
// Retourne ("", false) si l'en-tête est absent ou mal formé.
func bearerToken(c *gin.Context) (string, bool) {
	header := c.GetHeader("Authorization")
	if header == "" {
		return "", false
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}
	return parts[1], true
}

// stripIdentityHeaders supprime toute copie entrante des en-têtes d'identité :
// garde-fou anti-spoofing. Appelé systématiquement, même sans token.
func stripIdentityHeaders(c *gin.Context) {
	c.Request.Header.Del(HeaderUserID)
	c.Request.Header.Del(HeaderUserRole)
	c.Request.Header.Del(HeaderEmailVerified)
}

// PropagateJWT est le premier filtre JWT du gateway, appliqué globalement
// AVANT le reverse proxy. Politique « valider-si-présent » :
//   - aucun token        → passe-plat (login, refresh par cookie, vue visiteur) ;
//     le service tranche s'il exige une authentification ;
//   - token présent mais absent/mal formé/invalide/expiré → 401 au plus tôt,
//     la requête n'atteint jamais le service ;
//   - token valide       → injecte X-User-Id / X-User-Role / X-Email-Verified
//     depuis les claims, à destination des services backend.
//
// Les en-têtes d'identité entrants sont TOUJOURS strippés d'abord (anti-spoof).
// La validation par service est conservée (défense en profondeur) : ces en-têtes
// sont propagés pour usage futur, pas comme unique source de confiance.
func PropagateJWT(secret string) gin.HandlerFunc {
	key := []byte(secret)
	return func(c *gin.Context) {
		stripIdentityHeaders(c)

		tokenStr, ok := bearerToken(c)
		if !ok {
			if c.GetHeader("Authorization") != "" {
				// En-tête présent mais format invalide : on rejette au plus tôt.
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "format Authorization invalide (attendu : Bearer <token>)",
				})
				return
			}
			c.Next()
			return
		}

		cl, err := parse(tokenStr, key)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token invalide ou expiré"})
			return
		}

		c.Request.Header.Set(HeaderUserID, cl.UserID)
		c.Request.Header.Set(HeaderUserRole, cl.Role)
		if cl.EmailVerified {
			c.Request.Header.Set(HeaderEmailVerified, "true")
		}
		c.Next()
	}
}

// AdminJWT valide le bearer token (HS256, secret partagé avec auth-service) et
// exige le rôle `admin`. Garde des routes d'administration servies DIRECTEMENT
// par le gateway (ex. monitoring), distinctes du reverse proxy. 401 si le token
// est absent/invalide/expiré, 403 si le rôle n'est pas admin. Fail-closed : un
// secret vide ne validera aucun token légitime → accès refusé.
func AdminJWT(secret string) gin.HandlerFunc {
	key := []byte(secret)
	return func(c *gin.Context) {
		tokenStr, ok := bearerToken(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token manquant"})
			return
		}
		cl, err := parse(tokenStr, key)
		if err != nil {
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
