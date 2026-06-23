package client

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

// chanHandler est un slog.Handler minimal qui signale chaque log émis sur un
// canal — utilisé pour SYNCHRONISER avec la goroutine fire-and-forget de `call`
// (sinon le test pourrait finir avant que la branche d'erreur ne s'exécute).
type chanHandler struct{ ch chan struct{} }

func (h chanHandler) Enabled(context.Context, slog.Level) bool  { return true }
func (h chanHandler) Handle(context.Context, slog.Record) error { h.ch <- struct{}{}; return nil }
func (h chanHandler) WithAttrs([]slog.Attr) slog.Handler        { return h }
func (h chanHandler) WithGroup(string) slog.Handler             { return h }

// withLogCapture installe temporairement un logger qui signale sur `ch`, restauré
// en fin de test.
func withLogCapture(t *testing.T) (ch chan struct{}) {
	t.Helper()
	ch = make(chan struct{}, 1)
	prev := slog.Default()
	slog.SetDefault(slog.New(chanHandler{ch: ch}))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return ch
}

func waitLog(t *testing.T, ch chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(3 * time.Second):
		t.Fatalf("timeout : %s n'a pas journalisé d'erreur", what)
	}
}

// TestHTTPPostClient_DoErreur_Journalise couvre la branche d'erreur d'ENVOI
// (client.Do échoue) : un port fermé refuse la connexion, l'erreur est
// journalisée sans impacter l'appelant (fire-and-forget).
func TestHTTPPostClient_DoErreur_Journalise(t *testing.T) {
	ch := withLogCapture(t)
	c := NewPostClient("http://127.0.0.1:1", "s") // port 1 → connexion refusée
	c.AutoHide("post-injoignable")
	waitLog(t, ch, "AutoHide vers port fermé")
}

// TestHTTPPostClient_RequeteInvalide_Journalise couvre la branche d'erreur de
// CONSTRUCTION de requête : une base d'URL contenant un caractère de contrôle
// fait échouer http.NewRequestWithContext (parsing d'URL).
func TestHTTPPostClient_RequeteInvalide_Journalise(t *testing.T) {
	ch := withLogCapture(t)
	c := NewPostClient("http://bad\x7fhost", "s") // \x7f → URL invalide
	c.AutoUnhide("post-x")
	waitLog(t, ch, "AutoUnhide avec URL invalide")
}
