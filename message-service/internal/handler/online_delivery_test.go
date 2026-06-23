package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/webdad/message-service/internal/models"
)

// UpdateConversation avec un corps JSON invalide → 400 (garde de binding).
func TestHandler_UpdateConversation_BadJSON(t *testing.T) {
	e := newLiveEnv(t)
	tok := e.token(t, hA, "user")
	req := httptest.NewRequest(http.MethodPatch, "/messages/conversations/"+"507f1f77bcf86cd799439011", strings.NewReader(`{bad`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("UpdateConversation(JSON invalide) = %d, attendu 400", w.Code)
	}
}

// SendMessage à un destinataire EN LIGNE : son accusé « remis » est posé et
// diffusé instantanément → couvre la branche de livraison aux membres connectés.
func TestHandler_SendMessage_OnlineDelivery(t *testing.T) {
	e := newLiveEnv(t)
	tokA := e.token(t, hA, "user")

	// DM hA ↔ hB.
	w := e.do(t, http.MethodPost, "/messages/conversations", tokA, models.CreateConversationRequest{
		Type: models.TypeDM, PeerID: hB, Envelopes: envM(hA, hB),
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("CreateDM = %d (%s)", w.Code, w.Body.String())
	}
	dmID := dataID(t, w)

	// Connecte hB au hub du routeur via un vrai WebSocket.
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
		ws, err := up.Upgrade(rw, r, nil)
		if err != nil {
			return
		}
		go e.hub.Register(hB, ws)
	}))
	defer srv.Close()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial WS : %v", err)
	}
	defer func() { _ = conn.Close() }()

	// Attend que le hub enregistre hB.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && len(e.hub.OnlineFrom([]string{hB})) == 0 {
		time.Sleep(5 * time.Millisecond)
	}
	if len(e.hub.OnlineFrom([]string{hB})) == 0 {
		t.Fatal("hB devrait être en ligne")
	}

	// hA envoie : hB étant en ligne, le serveur le marque « remis » + diffuse.
	if w := e.do(t, http.MethodPost, "/messages/conversations/"+dmID+"/messages", tokA,
		models.SendMessageRequest{Ciphertext: "c", Nonce: "n"}); w.Code != http.StatusCreated {
		t.Fatalf("SendMessage = %d (%s)", w.Code, w.Body.String())
	}
}
