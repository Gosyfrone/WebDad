package database

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// mongoURI renvoie l'URI Mongo de test (CI : service conteneur ; local : défaut).
func mongoURI() string {
	if u := os.Getenv("MONGO_TEST_URI"); u != "" {
		return u
	}
	return "mongodb://localhost:27017"
}

// testDB ouvre une base de test éphémère et la supprime en fin de test. Si aucun
// Mongo n'est joignable, le test est ignoré (skip) — les développeurs sans base
// locale ne sont pas bloqués ; la CI fournit un conteneur Mongo (cf. ci-go.yml).
func testDB(t *testing.T) (*mongo.Database, context.Context) {
	t.Helper()
	ctx := context.Background()
	client, err := ConnectMongo(mongoURI())
	if err != nil {
		t.Skipf("Mongo indisponible (%v) — test d'intégration ignoré", err)
	}
	name := fmt.Sprintf("webdad_post_test_%d", time.Now().UnixNano())
	db := client.Database(name)
	// Sonde : un Mongo joignable mais nécessitant une authentification (cas du
	// stack de dev local) rejette les opérations → on ignore plutôt que d'échouer.
	// En CI, le conteneur Mongo dédié est sans auth : la sonde passe.
	if err := db.Collection("__probe").Drop(ctx); err != nil {
		_ = client.Disconnect(ctx)
		t.Skipf("Mongo joignable mais opérations refusées (%v) — test ignoré", err)
	}
	t.Cleanup(func() {
		_ = db.Drop(ctx)
		_ = client.Disconnect(ctx)
	})
	return db, ctx
}

func TestConnectMongo_Success(t *testing.T) {
	client, err := ConnectMongo(mongoURI())
	if err != nil {
		t.Skipf("Mongo indisponible (%v)", err)
	}
	_ = client.Disconnect(context.Background())
}

func TestConnectMongo_Unreachable(t *testing.T) {
	if _, err := ConnectMongo("mongodb://127.0.0.1:1/?serverSelectionTimeoutMS=500"); err == nil {
		t.Fatal("une URI injoignable doit renvoyer une erreur")
	}
}

func TestEnsureSchema_Idempotent(t *testing.T) {
	db, ctx := testDB(t)
	if err := EnsureSchema(ctx, db); err != nil {
		t.Fatalf("EnsureSchema (1er passage): %v", err)
	}
	// Deuxième passage : collMod + CreateMany d'index identiques = no-op.
	if err := EnsureSchema(ctx, db); err != nil {
		t.Fatalf("EnsureSchema (2e passage, idempotence): %v", err)
	}
	names, err := db.ListCollectionNames(ctx, bson.M{})
	if err != nil {
		t.Fatalf("ListCollectionNames: %v", err)
	}
	if len(names) < len(collectionOrder) {
		t.Fatalf("collections créées = %d, attendu >= %d", len(names), len(collectionOrder))
	}
}

func TestEnsureSchema_Backfills(t *testing.T) {
	db, ctx := testDB(t)
	// Documents « legacy » : post sans reply_audience, commentaire sans likes_count.
	if _, err := db.Collection("posts").InsertOne(ctx, bson.M{
		"author_id":  "u1",
		"content":    "vieux post",
		"created_at": time.Now(),
	}); err != nil {
		t.Fatalf("insert post legacy: %v", err)
	}
	if _, err := db.Collection("comments").InsertOne(ctx, bson.M{
		"post_id":    "p1",
		"author_id":  "u1",
		"content":    "vieux commentaire",
		"created_at": time.Now(),
	}); err != nil {
		t.Fatalf("insert comment legacy: %v", err)
	}

	if err := EnsureSchema(ctx, db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}

	var post bson.M
	if err := db.Collection("posts").FindOne(ctx, bson.M{"author_id": "u1"}).Decode(&post); err != nil {
		t.Fatalf("relecture post: %v", err)
	}
	if post["reply_audience"] != "everyone" {
		t.Fatalf("backfill reply_audience attendu 'everyone', got %v", post["reply_audience"])
	}
	var comment bson.M
	if err := db.Collection("comments").FindOne(ctx, bson.M{"author_id": "u1"}).Decode(&comment); err != nil {
		t.Fatalf("relecture comment: %v", err)
	}
	if v, ok := comment["likes_count"]; !ok || v == nil {
		t.Fatalf("backfill likes_count attendu, got %v", v)
	}
}
