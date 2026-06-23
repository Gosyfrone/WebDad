package database

import (
	"context"
	"testing"
)

// canceled renvoie un contexte déjà annulé : les commandes Mongo (listCollections,
// CreateMany, UpdateByID) échouent, ce qui exerce les branches `return fmt.Errorf`
// du schéma — jamais atteintes contre une base saine.
func canceled() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

// TestEnsureSchema_CtxAnnule couvre les remontées d'erreur du schéma. La base est
// réelle (freshDB) ; seul le contexte est annulé. On vérifie chaque sous-étape
// directement pour isoler sa branche d'erreur, puis EnsureSchema bout en bout.
func TestEnsureSchema_CtxAnnule(t *testing.T) {
	db, _ := freshDB(t) // skip si MONGO_TEST_URI absent
	cctx := canceled()

	if err := ensureCollections(cctx, db); err == nil {
		t.Error("ensureCollections(ctx annulé) → nil, attendu erreur")
	}
	if err := ensureIndexes(cctx, db); err == nil {
		t.Error("ensureIndexes(ctx annulé) → nil, attendu erreur")
	}
	if err := ensureSettings(cctx, db); err == nil {
		t.Error("ensureSettings(ctx annulé) → nil, attendu erreur")
	}
	// Bout en bout : EnsureSchema doit propager l'erreur de la 1re sous-étape.
	if err := EnsureSchema(cctx, db); err == nil {
		t.Error("EnsureSchema(ctx annulé) → nil, attendu erreur")
	}
}

// TestConnectMongo_URIMalformee couvre la branche d'erreur de mongo.Connect
// (parsing de l'URI), distincte de l'échec au ping déjà testé : un schéma
// d'URI invalide est rejeté par ApplyURI avant toute connexion.
func TestConnectMongo_URIMalformee(t *testing.T) {
	if _, err := ConnectMongo("://pas-une-uri"); err == nil {
		t.Error("ConnectMongo sur une URI malformée → nil, attendu erreur")
	}
}
