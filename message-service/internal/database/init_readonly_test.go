package database

import (
	"context"
	"testing"

	"github.com/webdad/message-service/internal/mongotest"
)

// EnsureSchema sur un client DÉCONNECTÉ : ListCollectionNames échoue → branche
// d'erreur de ensureCollections (remontée par EnsureSchema).
func TestEnsureSchema_Disconnected(t *testing.T) {
	db := mongotest.DisposableDB(t)
	if err := db.Client().Disconnect(context.Background()); err != nil {
		t.Fatalf("Disconnect : %v", err)
	}
	if err := EnsureSchema(context.Background(), db); err == nil {
		t.Fatal("EnsureSchema sur client déconnecté doit échouer")
	}
}

// EnsureSchema via un client LECTURE SEULE sur une base vide : la création de
// collection (write) échoue → branche d'erreur de CreateCollection.
func TestEnsureSchema_ReadOnly_CreateFails(t *testing.T) {
	_, ro := mongotest.ReadOnlyDB(t)
	if err := EnsureSchema(context.Background(), ro); err == nil {
		t.Fatal("EnsureSchema en lecture seule (création) doit échouer")
	}
}

// EnsureSchema via un client LECTURE SEULE sur une base dont les collections
// EXISTENT déjà (peuplées via root) : le resync du validateur (collMod, write)
// échoue → branche d'erreur du collMod.
func TestEnsureSchema_ReadOnly_CollModFails(t *testing.T) {
	root, ro := mongotest.ReadOnlyDB(t)
	if err := EnsureSchema(context.Background(), root); err != nil {
		t.Fatalf("seed schéma via root : %v", err)
	}
	if err := EnsureSchema(context.Background(), ro); err == nil {
		t.Fatal("EnsureSchema en lecture seule (collMod) doit échouer")
	}
}
