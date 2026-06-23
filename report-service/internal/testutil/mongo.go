// Package testutil fournit des aides partagées aux tests d'intégration du
// report-service (connexion Mongo isolée par test). Importé UNIQUEMENT par des
// fichiers _test.go : aucun binaire de production n'en dépend.
package testutil

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/webdad/report-service/internal/database"
)

// dbSeq garantit un nom de base unique même pour des tests créés à la même
// nanoseconde (tests parallèles).
var dbSeq int64

// MongoURI renvoie l'URI Mongo de test (MONGO_TEST_URI), ou "" si non défini.
func MongoURI() string { return os.Getenv("MONGO_TEST_URI") }

// MongoDB renvoie une base de données Mongo FRAÎCHE et isolée pour le test, avec
// le schéma appliqué (EnsureSchema : collections + validateurs + index). La base
// est supprimée et la connexion fermée au nettoyage du test.
//
// Si MONGO_TEST_URI n'est pas défini (run local sans Mongo), le test est ignoré
// (t.Skip) plutôt qu'en échec — l'intégration ne tourne qu'avec une base réelle
// (cf. service mongo de la CI). Une base injoignable est aussi un skip.
func MongoDB(t *testing.T) *mongo.Database {
	t.Helper()
	uri := MongoURI()
	if uri == "" {
		t.Skip("MONGO_TEST_URI non défini — test d'intégration Mongo ignoré")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatalf("connexion Mongo (%s) : %v", uri, err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		t.Skipf("Mongo injoignable (%s) : %v — test ignoré", uri, err)
	}

	name := fmt.Sprintf("reporttest_%d_%d", time.Now().UnixNano(), atomic.AddInt64(&dbSeq, 1))
	db := client.Database(name)
	if err := database.EnsureSchema(ctx, db); err != nil {
		_ = client.Disconnect(ctx)
		t.Fatalf("EnsureSchema : %v", err)
	}

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = db.Drop(ctx)
		_ = client.Disconnect(ctx)
	})
	return db
}
