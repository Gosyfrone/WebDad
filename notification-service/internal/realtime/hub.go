// Package realtime gère la diffusion temps réel des notifications via WebSocket.
//
// Modèle : une connexion WS par utilisateur (authentifiée par JWT). Le hub
// indexe les connexions par user_id. Quand une notification est créée ou mise à
// jour, le service fournit l'id du destinataire et le hub pousse l'événement à
// chacune de ses connexions actives.
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

// Hub indexe les connexions WebSocket actives par utilisateur.
type Hub struct {
	mu    sync.RWMutex
	conns map[string]map[*Conn]struct{} // user_id -> ensemble de connexions
}

// NewHub crée un hub vide.
func NewHub() *Hub {
	return &Hub{conns: make(map[string]map[*Conn]struct{})}
}

// Conn représente une connexion WebSocket d'un utilisateur. Les écritures sur la
// socket passent TOUTES par la goroutine writePump (une seule écrivaine) via le
// canal send — un *websocket.Conn ne supporte pas les écritures concurrentes.
type Conn struct {
	hub    *Hub
	userID string
	ws     *websocket.Conn
	send   chan []byte
}

// Register crée une connexion gérée pour userID, l'enregistre et lance ses
// pompes de lecture/écriture. La socket est fermée et désenregistrée à la fin.
func (h *Hub) Register(userID string, ws *websocket.Conn) {
	c := &Conn{hub: h, userID: userID, ws: ws, send: make(chan []byte, sendBuffer)}

	h.mu.Lock()
	if h.conns[userID] == nil {
		h.conns[userID] = make(map[*Conn]struct{})
	}
	h.conns[userID][c] = struct{}{}
	h.mu.Unlock()

	slog.Debug("hub ws enregistré", "user_id", userID)
	go c.writePump()
	c.readPump() // bloque jusqu'à fermeture de la socket
}

// remove désenregistre une connexion (et nettoie l'entrée utilisateur vide).
func (h *Hub) remove(c *Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set, ok := h.conns[c.userID]; ok {
		if _, exists := set[c]; exists {
			delete(set, c)
			close(c.send)
			slog.Debug("hub ws déconnecté", "user_id", c.userID)
		}
		if len(set) == 0 {
			delete(h.conns, c.userID)
		}
	}
}

// Publish pousse un événement (sérialisé une seule fois) à toutes les connexions
// des utilisateurs visés. Non bloquant : une connexion saturée est fermée plutôt
// que de ralentir la diffusion.
func (h *Hub) Publish(userIDs []string, event any) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()
	if len(userIDs) == 0 {
		for _, set := range h.conns {
			for c := range set {
				select {
				case c.send <- data:
				default:
					_ = c.ws.Close()
				}
			}
		}
		return
	}
	for _, uid := range userIDs {
		for c := range h.conns[uid] {
			select {
			case c.send <- data:
			default:
				// Client trop lent : on ferme sa socket (readPump nettoiera).
				_ = c.ws.Close()
			}
		}
	}
}

// readPump draine les messages entrants (on n'attend rien du client : l'envoi
// se fait via l'API REST). Sert surtout à détecter la fermeture de la socket.
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

// writePump écrit les événements diffusés sur la socket (seule goroutine
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
