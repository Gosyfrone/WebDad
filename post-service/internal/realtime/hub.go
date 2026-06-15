// Package realtime diffuse en temps réel les évènements du fil (nouveaux posts)
// via WebSocket.
//
// Modèle : un hub de DIFFUSION (broadcast). Contrairement au temps réel des
// notifications (une connexion par destinataire), le fil « pour toi » est public
// — un nouveau post public concerne potentiellement tout le monde. Le hub garde
// donc un simple ensemble de connexions actives et pousse à toutes un évènement
// léger (id du post + id de l'auteur). Le client résout l'auteur (avatar/nom) et
// va chercher le contenu via le fil authentifié normal : la barrière de
// visibilité reste server-side, le WS n'est qu'un « ping ».
package realtime

import (
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// writeWait : délai max d'écriture d'un message sur la socket.
const writeWait = 10 * time.Second

// sendBuffer : taille du tampon d'envoi par connexion (messages en attente
// d'écriture). Au-delà, on considère le client trop lent et on ferme.
const sendBuffer = 32

// Hub garde l'ensemble des connexions WebSocket actives du fil.
type Hub struct {
	mu    sync.RWMutex
	conns map[*Conn]struct{}
}

// NewHub crée un hub vide.
func NewHub() *Hub {
	return &Hub{conns: make(map[*Conn]struct{})}
}

// Conn représente une connexion WebSocket. Les écritures sur la socket passent
// TOUTES par la goroutine writePump (une seule écrivaine) via le canal send — un
// *websocket.Conn ne supporte pas les écritures concurrentes.
type Conn struct {
	hub  *Hub
	ws   *websocket.Conn
	send chan []byte
}

// postCreatedEvent : charge utile poussée aux clients à la création d'un post
// public racine. Volontairement minimale (le client enrichit l'auteur et
// refetch le contenu via le fil normal).
type postCreatedEvent struct {
	Type     string `json:"type"`
	PostID   string `json:"post_id"`
	AuthorID string `json:"author_id"`
}

// Register crée une connexion gérée, l'enregistre et lance ses pompes de
// lecture/écriture. La socket est fermée et désenregistrée à la fin.
func (h *Hub) Register(ws *websocket.Conn) {
	c := &Conn{hub: h, ws: ws, send: make(chan []byte, sendBuffer)}

	h.mu.Lock()
	h.conns[c] = struct{}{}
	h.mu.Unlock()

	slog.Debug("hub fil ws enregistré")
	go c.writePump()
	c.readPump() // bloque jusqu'à fermeture de la socket
}

// remove désenregistre une connexion.
func (h *Hub) remove(c *Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.conns[c]; ok {
		delete(h.conns, c)
		close(c.send)
		slog.Debug("hub fil ws déconnecté")
	}
}

// PostCreated diffuse à toutes les connexions actives qu'un nouveau post (public
// racine) vient d'être publié. Satisfait l'interface attendue par le service.
func (h *Hub) PostCreated(postID, authorID string) {
	h.broadcast(postCreatedEvent{Type: "post_created", PostID: postID, AuthorID: authorID})
}

// broadcast pousse un évènement (sérialisé une seule fois) à toutes les
// connexions. Non bloquant : une connexion saturée est fermée plutôt que de
// ralentir la diffusion.
func (h *Hub) broadcast(event any) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.conns {
		select {
		case c.send <- data:
		default:
			// Client trop lent : on ferme sa socket (readPump nettoiera).
			_ = c.ws.Close()
		}
	}
}

// readPump draine les messages entrants (on n'attend rien du client). Sert
// surtout à détecter la fermeture de la socket.
func (c *Conn) readPump() {
	defer func() {
		c.hub.remove(c)
		_ = c.ws.Close()
	}()
	for {
		if _, _, err := c.ws.ReadMessage(); err != nil {
			return
		}
	}
}

// writePump écrit les évènements diffusés sur la socket (seule goroutine
// écrivaine de cette connexion).
func (c *Conn) writePump() {
	for data := range c.send {
		_ = c.ws.SetWriteDeadline(time.Now().Add(writeWait))
		if err := c.ws.WriteMessage(websocket.TextMessage, data); err != nil {
			_ = c.ws.Close()
			return
		}
	}
}
