package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Le graphe social (follows) est volontairement laissé en STUB dans ce
// squelette : routes + handlers en place (contrat figé), implémentation à
// brancher dans une issue dédiée. Le schéma SQL (table `follows`) existe déjà.

// Follow : POST /users/:id/follow — suivre un utilisateur (stub).
func (h *Handler) Follow(c *gin.Context) {
	notImplemented(c, "follow")
}

// Unfollow : DELETE /users/:id/follow — ne plus suivre (stub).
func (h *Handler) Unfollow(c *gin.Context) {
	notImplemented(c, "unfollow")
}

// Followers : GET /users/:id/followers — abonnés d'un utilisateur (stub).
func (h *Handler) Followers(c *gin.Context) {
	notImplemented(c, "followers")
}

// Following : GET /users/:id/following — abonnements d'un utilisateur (stub).
func (h *Handler) Following(c *gin.Context) {
	notImplemented(c, "following")
}

func notImplemented(c *gin.Context, feature string) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": feature + " : non implémenté (squelette)"})
}
