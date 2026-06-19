// Package middleware : limitation de débit (anti brute-force / RIV-001).
package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter applique une fenêtre glissante simple « N requêtes par fenêtre,
// par clé » (clé = IP cliente). En mémoire, sans dépendance externe : suffisant
// pour une instance unique par service (cf. docker-compose). Pour un déploiement
// multi-instances, remplacer le backend par Redis (même interface).
//
// But : ralentir le brute-force de mot de passe (login), de code TOTP
// (mfa/verify) et l'abus des envois d'e-mails (register/forgot/resend), sans
// gêner un usage normal. Échec → 429 + en-tête Retry-After.
type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*window
	limit    int
	period   time.Duration
}

type window struct {
	count   int
	resetAt time.Time
}

// NewRateLimiter crée un limiteur (limit requêtes / period) et lance le balayage
// périodique des entrées expirées (anti-fuite mémoire).
func NewRateLimiter(limit int, period time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*window),
		limit:    limit,
		period:   period,
	}
	go rl.cleanupLoop()
	return rl
}

// allow incrémente le compteur de la clé et indique si la requête passe.
// Retourne aussi le temps restant avant réinitialisation (pour Retry-After).
func (rl *RateLimiter) allow(key string) (bool, time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	w, ok := rl.visitors[key]
	if !ok || now.After(w.resetAt) {
		rl.visitors[key] = &window{count: 1, resetAt: now.Add(rl.period)}
		return true, 0
	}
	if w.count >= rl.limit {
		return false, time.Until(w.resetAt)
	}
	w.count++
	return true, 0
}

// Middleware borne le débit par IP cliente. À placer en tête des routes
// sensibles. `c.ClientIP()` lit X-Forwarded-For (posé par Caddy puis la gateway)
// → vraie IP du client, pas celle du proxy.
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ok, retryAfter := rl.allow(c.ClientIP())
		if !ok {
			c.Header("Retry-After", strconv.Itoa(int(retryAfter.Seconds())+1))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "trop de tentatives, réessaie plus tard",
				"code":  "rate_limited",
			})
			return
		}
		c.Next()
	}
}

// cleanupLoop purge les fenêtres expirées toutes les `period` (au plus chaque
// minute) pour que la map ne grossisse pas indéfiniment.
func (rl *RateLimiter) cleanupLoop() {
	interval := rl.period
	if interval > time.Minute {
		interval = time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		rl.mu.Lock()
		for k, w := range rl.visitors {
			if now.After(w.resetAt) {
				delete(rl.visitors, k)
			}
		}
		rl.mu.Unlock()
	}
}
