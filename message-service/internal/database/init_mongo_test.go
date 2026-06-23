package database

import (
	"context"
	"os"
	"testing"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// TestConnectMongo couvre l'ouverture de connexion (cas nominal + erreur d'URI).
func TestConnectMongo(t *testing.T) {
	// URI invalide → erreur (pas de ping possible).
	if _, err := ConnectMongo("mongodb://127.0.0.1:1/?serverSelectionTimeoutMS=200&connectTimeoutMS=200"); err == nil {
		t.Error("URI injoignable doit échouer au ping")
	}

	uri := os.Getenv("MONGO_TEST_URI")
	if uri == "" {
		t.Skip("MONGO_TEST_URI non défini — connexion réelle ignorée")
	}
	client, err := ConnectMongo(uri)
	if err != nil {
		t.Fatalf("ConnectMongo : %v", err)
	}
	t.Cleanup(func() { _ = client.Disconnect(context.Background()) })
}

// TestEnsureSchema_Idempotent vérifie que le schéma se (re)pose sans erreur, y
// compris sur une base déjà initialisée (resync collMod + index CreateMany).
func TestEnsureSchema_Idempotent(t *testing.T) {
	uri := os.Getenv("MONGO_TEST_URI")
	if uri == "" {
		t.Skip("MONGO_TEST_URI non défini — schéma Mongo ignoré")
	}
	ctx := context.Background()
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatalf("connect : %v", err)
	}
	db := client.Database("msgtest_schema_idem")
	t.Cleanup(func() {
		_ = db.Drop(ctx)
		_ = client.Disconnect(ctx)
	})

	if err := EnsureSchema(ctx, db); err != nil {
		t.Fatalf("EnsureSchema (1) : %v", err)
	}
	// 2e passage : collections existent déjà → branche collMod + CreateMany idempotent.
	if err := EnsureSchema(ctx, db); err != nil {
		t.Fatalf("EnsureSchema (2) : %v", err)
	}
}
