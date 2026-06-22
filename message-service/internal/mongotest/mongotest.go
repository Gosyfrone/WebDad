// Package mongotest fournit un helper partagé pour les tests d'intégration Mongo
// du message-service. Les tests qui en dépendent sont GATED par la variable
// d'environnement MONGO_TEST_URI : sans elle, ils s'auto-ignorent (t.Skip) — la
// suite reste donc verte hors d'un environnement disposant d'une MongoDB (un
// service `mongo` est fourni en CI, cf. .github/workflows/ci-go.yml).
//
// Chaque appel à DB crée une base JETABLE au nom unique et l'efface en fin de
// test (t.Cleanup) : les tests sont isolés les uns des autres et ne polluent
// jamais une base existante. Un client Mongo unique est partagé par tout le
// binaire de test (évite la churn de connexions) ; DisposableDB ouvre au
// contraire un client dédié pour les tests qui le DÉCONNECTENT volontairement
// (simulation de panne).
package mongotest

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	counter      int64
	sharedOnce   sync.Once
	sharedClient *mongo.Client
	sharedErr    error
)

// uri renvoie l'URI de test ou ignore le test si elle est absente.
func uri(t *testing.T) string {
	t.Helper()
	v := os.Getenv("MONGO_TEST_URI")
	if v == "" {
		t.Skip("MONGO_TEST_URI non défini — test d'intégration Mongo ignoré")
	}
	return v
}

func connect(uri string) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	opts := options.Client().ApplyURI(uri).
		SetServerSelectionTimeout(5 * time.Second).
		SetMaxPoolSize(20)
	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, err
	}
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}
	return client, nil
}

func dbName() string {
	return fmt.Sprintf("msgtest_%d_%d", time.Now().UnixNano(), atomic.AddInt64(&counter, 1))
}

// DB renvoie une base jetable (vide) sur le client partagé, effacée en fin de
// test. À utiliser pour tous les tests qui ne déconnectent PAS le client.
func DB(t *testing.T) *mongo.Database {
	t.Helper()
	u := uri(t)
	sharedOnce.Do(func() { sharedClient, sharedErr = connect(u) })
	if sharedErr != nil {
		t.Fatalf("connexion Mongo partagée : %v", sharedErr)
	}
	db := sharedClient.Database(dbName())
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = db.Drop(ctx)
	})
	return db
}

// DisposableDB renvoie une base jetable sur un client DÉDIÉ (propre au test),
// déconnecté au nettoyage. À utiliser par les tests qui déconnectent le client
// pour simuler une panne du dépôt.
func DisposableDB(t *testing.T) *mongo.Database {
	t.Helper()
	client, err := connect(uri(t))
	if err != nil {
		t.Fatalf("connexion Mongo dédiée : %v", err)
	}
	db := client.Database(dbName())
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = db.Drop(ctx)
		_ = client.Disconnect(ctx)
	})
	return db
}
