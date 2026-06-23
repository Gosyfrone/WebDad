package database

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var dbSeq int64

// freshDB connecte une base Mongo isolée via ConnectMongo (couvre mongo.go), ou
// skip si MONGO_TEST_URI n'est pas défini / la base est injoignable.
func freshDB(t *testing.T) (*mongo.Database, context.Context) {
	t.Helper()
	uri := os.Getenv("MONGO_TEST_URI")
	if uri == "" {
		t.Skip("MONGO_TEST_URI non défini — test d'intégration Mongo ignoré")
	}
	client, err := ConnectMongo(uri)
	if err != nil {
		t.Skipf("Mongo injoignable (%s) : %v — test ignoré", uri, err)
	}
	name := fmt.Sprintf("reportdbtest_%d_%d", time.Now().UnixNano(), atomic.AddInt64(&dbSeq, 1))
	db := client.Database(name)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = db.Drop(ctx)
		_ = client.Disconnect(ctx)
	})
	return db, context.Background()
}

// TestEnsureSchema_Idempotent applique le schéma DEUX fois : la 1re crée les
// collections (branche CreateCollection), la 2nde les retrouve et resynchronise
// le validateur (branche collMod). Vérifie collections, index et singleton seedé.
func TestEnsureSchema_Idempotent(t *testing.T) {
	db, ctx := freshDB(t)

	if err := EnsureSchema(ctx, db); err != nil {
		t.Fatalf("1er EnsureSchema : %v", err)
	}
	// 2e passage → branche collMod (collection déjà présente) + index idempotents.
	if err := EnsureSchema(ctx, db); err != nil {
		t.Fatalf("2e EnsureSchema : %v", err)
	}

	names, err := db.ListCollectionNames(ctx, bson.M{})
	if err != nil {
		t.Fatalf("liste collections : %v", err)
	}
	have := map[string]bool{}
	for _, n := range names {
		have[n] = true
	}
	for _, want := range []string{"tickets", "warnings", "settings"} {
		if !have[want] {
			t.Errorf("collection %q absente après EnsureSchema", want)
		}
	}

	// Le singleton de configuration a été seedé avec le défaut.
	var s struct {
		AutoHideThreshold int32 `bson:"auto_hide_threshold"`
	}
	if err := db.Collection("settings").FindOne(ctx, bson.M{"_id": "global"}).Decode(&s); err != nil {
		t.Fatalf("settings singleton : %v", err)
	}
	if s.AutoHideThreshold != 5 {
		t.Errorf("seuil seedé = %d, attendu 5 (défaut)", s.AutoHideThreshold)
	}

	// Index unique partiel sur (entity_type, entity_id) présent.
	cur, err := db.Collection("tickets").Indexes().List(ctx)
	if err != nil {
		t.Fatalf("liste index : %v", err)
	}
	var idx []bson.M
	if err := cur.All(ctx, &idx); err != nil {
		t.Fatalf("decode index : %v", err)
	}
	if len(idx) < 2 {
		t.Errorf("attendu plusieurs index sur 'tickets', obtenu %d", len(idx))
	}
}

// TestConnectMongo_URIInvalide couvre la branche d'erreur de ConnectMongo (ping
// échoue rapidement sur un hôte injoignable).
func TestConnectMongo_URIInvalide(t *testing.T) {
	if os.Getenv("MONGO_TEST_URI") == "" {
		t.Skip("MONGO_TEST_URI non défini — test ignoré")
	}
	// Port fermé → la connexion s'ouvre mais le ping échoue (timeout court).
	_, err := ConnectMongo("mongodb://127.0.0.1:1/?serverSelectionTimeoutMS=500&connectTimeoutMS=500")
	if err == nil {
		t.Error("ConnectMongo sur un port fermé devrait échouer au ping")
	}
}
