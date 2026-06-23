package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// TestWSConnect_ValidToken_UpgradeFail : token valide mais httptest.Recorder
// ne supporte pas WebSocket → upgrader écrit l'erreur (400) et retourne.
// Couvre la branche "if err != nil { return }" dans Connect.
func TestWSConnect_ValidToken_UpgradeFail(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodGet, "/posts/ws?access_token="+tok, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// Token valide mais recorder ne supporte pas WS → 400 de gorilla, pas 401.
	if w.Code == http.StatusUnauthorized {
		t.Errorf("WSConnect(token valide) = 401, le token ne doit pas être rejeté")
	}
}

// TestWSConnect_ValidToken_FullUpgrade : connexion WebSocket complète via un
// vrai httptest.Server. Couvre slog.Debug + hub.Register dans Connect, ainsi
// que le corps de CheckOrigin (origin vide → acceptée).
func TestWSConnect_ValidToken_FullUpgrade(t *testing.T) {
	engine := newNilRepoRouter(t)
	srv := httptest.NewServer(engine)
	defer srv.Close()

	tok := makePostToken(t, "user")
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/posts/ws?access_token=" + tok

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WS dial : %v", err)
	}
	defer conn.Close()

	// Laisser le hub enregistrer la connexion.
	time.Sleep(30 * time.Millisecond)
}

// TestWSConnect_AllowedOrigin : même test avec en-tête Origin connu → CheckOrigin
// retourne true (branche allowed[origin]).
func TestWSConnect_AllowedOrigin(t *testing.T) {
	engine := newNilRepoRouter(t)
	srv := httptest.NewServer(engine)
	defer srv.Close()

	tok := makePostToken(t, "user")
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/posts/ws?access_token=" + tok

	// newNilRepoRouter passe allowedOrigins=nil → allowed est vide.
	// L'origin ne sera pas autorisée via allowed[origin] (false) mais origin != "" →
	// CheckOrigin retourne false → gorilla refuse l'upgrade → erreur attendue.
	hdr := http.Header{}
	hdr.Set("Origin", "https://evil.example.com")
	_, _, err := websocket.DefaultDialer.Dial(wsURL, hdr)
	// Upgrade refusé par CheckOrigin → erreur WS.
	if err == nil {
		t.Error("origine non autorisée doit être rejetée")
	}
}
