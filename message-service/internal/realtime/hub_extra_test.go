package realtime

import (
	"testing"
	"time"
)

// injectConn insère une connexion factice dans le hub sans WebSocket.
func injectConn(h *Hub, userID string) *Conn {
	c := &Conn{hub: h, userID: userID, ws: nil, send: make(chan []byte, sendBuffer)}
	h.mu.Lock()
	if h.conns[userID] == nil {
		h.conns[userID] = make(map[*Conn]struct{})
	}
	h.conns[userID][c] = struct{}{}
	h.mu.Unlock()
	return c
}

// ─── remove — connexion présente ─────────────────────────────────────────────

func TestRemove_ConnPrésente_FermeCanal(t *testing.T) {
	h := NewHub()
	c := injectConn(h, "u1")

	h.remove(c)

	// Le canal doit être fermé (receive retourne immédiatement la valeur zéro).
	select {
	case _, ok := <-c.send:
		if ok {
			t.Fatal("remove doit fermer le canal send")
		}
	default:
		t.Fatal("canal non fermé après remove")
	}
}

func TestRemove_DernièreConn_SupprimeUtilisateur(t *testing.T) {
	h := NewHub()
	c := injectConn(h, "u2")

	h.remove(c)

	h.mu.RLock()
	_, exists := h.conns["u2"]
	h.mu.RUnlock()
	if exists {
		t.Fatal("remove de la dernière connexion doit supprimer l'entrée utilisateur")
	}
}

func TestRemove_DeuxConns_GardeUtilisateur(t *testing.T) {
	h := NewHub()
	c1 := injectConn(h, "u3")
	_ = injectConn(h, "u3") // 2ème connexion pour le même utilisateur

	h.remove(c1) // retire seulement c1

	h.mu.RLock()
	n := len(h.conns["u3"])
	h.mu.RUnlock()
	if n != 1 {
		t.Fatalf("après remove de c1, h.conns[u3] = %d connexions, attendu 1", n)
	}
}

func TestRemove_Idempotent(t *testing.T) {
	h := NewHub()
	c := injectConn(h, "u4")

	h.remove(c)
	// 2ème appel sur une connexion déjà retirée ne doit pas paniquer
	// (le canal est déjà fermé, mais le code vérifie l'existence avant de closer).
	h.remove(c)
}

// ─── Publish — connexion dans le hub ─────────────────────────────────────────

func TestPublish_AvecConn_EnvoiDansCanal(t *testing.T) {
	h := NewHub()
	c := injectConn(h, "u5")

	h.Publish([]string{"u5"}, map[string]string{"type": "msg", "text": "hello"})

	select {
	case data := <-c.send:
		if len(data) == 0 {
			t.Fatal("message reçu vide")
		}
	case <-time.After(time.Second):
		t.Fatal("Publish n'a pas livré le message dans le canal")
	}
}

func TestPublish_MultipleUsers_SeulsVisésReçoivent(t *testing.T) {
	h := NewHub()
	c1 := injectConn(h, "u6")
	c2 := injectConn(h, "u7")

	h.Publish([]string{"u6"}, map[string]string{"type": "test"})

	// u6 doit recevoir, u7 ne doit pas.
	select {
	case <-c1.send:
	case <-time.After(time.Second):
		t.Fatal("u6 n'a pas reçu le message")
	}
	select {
	case <-c2.send:
		t.Fatal("u7 ne devait pas recevoir le message")
	default:
	}
}

// ─── writePump — fermeture du canal (sortie de boucle) ───────────────────────

func TestWritePump_CanalFermé_SortImmédiatement(t *testing.T) {
	h := NewHub()
	c := &Conn{hub: h, userID: "u8", ws: nil, send: make(chan []byte, 1)}
	close(c.send) // boucle `for data := range c.send` sort immédiatement

	done := make(chan struct{})
	go func() {
		c.writePump()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("writePump ne s'est pas terminé après fermeture du canal")
	}
}
