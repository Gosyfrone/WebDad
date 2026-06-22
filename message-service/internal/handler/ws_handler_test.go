package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/webdad/message-service/internal/realtime"
)

// wsServer monte un routeur Gin servant /messages/ws via le WSHandler réel.
func wsServer(t *testing.T) *httptest.Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	wsH := NewWSHandler(realtime.NewHub(), msgTestSecret, []string{"http://localhost:3000"})
	r.GET("/messages/ws", wsH.Connect)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

func TestWSConnect_ValidToken(t *testing.T) {
	srv := wsServer(t)
	tok := makeMsgToken(t, "user")
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/messages/ws?access_token=" + tok

	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial WS valide : %v", err)
	}
	_ = c.Close()
}

func TestWSConnect_InvalidToken_401(t *testing.T) {
	srv := wsServer(t)
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/messages/ws?access_token=garbage"

	_, resp, err := websocket.DefaultDialer.Dial(url, nil)
	if err == nil {
		t.Fatal("un token invalide doit refuser l'upgrade")
	}
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("statut = %v, attendu 401", resp)
	}
}

// CheckOrigin : une origine non autorisée doit faire échouer l'upgrade.
func TestWSConnect_BadOrigin(t *testing.T) {
	srv := wsServer(t)
	tok := makeMsgToken(t, "user")
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/messages/ws?access_token=" + tok

	h := http.Header{}
	h.Set("Origin", "http://evil.example")
	_, _, err := websocket.DefaultDialer.Dial(url, h)
	if err == nil {
		t.Fatal("une origine non autorisée doit être rejetée")
	}
}
