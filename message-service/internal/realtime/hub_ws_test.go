package realtime

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// dialHub ouvre une vraie connexion WS vers un serveur httptest qui enregistre
// l'utilisateur dans le hub (Register lance read/writePump). Renvoie le client.
func dialHub(t *testing.T, h *Hub, userID string) *websocket.Conn {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
		ws, err := up.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade : %v", err)
			return
		}
		// Register bloque (readPump) jusqu'à fermeture → goroutine dédiée.
		go h.Register(userID, ws)
	}))
	t.Cleanup(srv.Close)

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial : %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

// Couvre Register + writePump (réception d'un Publish) + OnlineFrom + remove.
func TestHub_RegisterPublishReceive(t *testing.T) {
	h := NewHub()
	client := dialHub(t, h, "u1")

	// Laisse le hub enregistrer la connexion.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(h.OnlineFrom([]string{"u1"})) == 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if got := h.OnlineFrom([]string{"u1", "u2"}); len(got) != 1 || got[0] != "u1" {
		t.Fatalf("OnlineFrom = %v, attendu [u1]", got)
	}

	// Publish → writePump → le client reçoit le message.
	h.Publish([]string{"u1"}, map[string]string{"type": "ping"})
	_ = client.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := client.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage : %v", err)
	}
	if !strings.Contains(string(data), "ping") {
		t.Fatalf("message reçu = %s", data)
	}

	// Fermeture côté client → readPump détecte la fin → remove.
	_ = client.Close()
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(h.OnlineFrom([]string{"u1"})) == 0 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("la connexion fermée doit être désenregistrée du hub")
}

// Deux connexions pour le même user : l'entrée n'est nettoyée qu'à la dernière.
func TestHub_MultiConn(t *testing.T) {
	h := NewHub()
	c1 := dialHub(t, h, "dup")
	c2 := dialHub(t, h, "dup")

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(h.OnlineFrom([]string{"dup"})) == 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	h.Publish([]string{"dup"}, map[string]string{"type": "x"})
	_ = c1.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, err := c1.ReadMessage(); err != nil {
		t.Fatalf("c1 ReadMessage : %v", err)
	}
	_ = c2.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, err := c2.ReadMessage(); err != nil {
		t.Fatalf("c2 ReadMessage : %v", err)
	}
	_ = c1.Close()
	_ = c2.Close()
}
