package database

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func mongoIntegrationURI(t *testing.T) string {
	t.Helper()
	if os.Getenv("RUN_MONGO_INTEGRATION") != "1" {
		t.Skip("définir RUN_MONGO_INTEGRATION=1 pour les tests Mongo")
	}
	host, port := os.Getenv("MONGO_HOST"), os.Getenv("MONGO_PORT")
	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = "27017"
	}
	user, password := os.Getenv("MONGO_INITDB_ROOT_USERNAME"), os.Getenv("MONGO_INITDB_ROOT_PASSWORD")
	if user == "" {
		return fmt.Sprintf("mongodb://%s:%s", host, port)
	}
	return fmt.Sprintf("mongodb://%s:%s@%s:%s/?authSource=admin", user, password, host, port)
}

func integrationDatabase(t *testing.T) (*mongo.Client, *mongo.Database) {
	t.Helper()
	client, err := mongo.Connect(options.Client().ApplyURI(mongoIntegrationURI(t)))
	if err != nil {
		t.Fatalf("mongo.Connect: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	if err := client.Ping(ctx, nil); err != nil {
		t.Fatalf("mongo ping: %v", err)
	}
	db := client.Database(fmt.Sprintf("notification_database_test_%d", time.Now().UnixNano()))
	t.Cleanup(func() {
		_ = db.Drop(context.Background())
		_ = client.Disconnect(context.Background())
	})
	return client, db
}

func TestEnsureSchemaCreatesAndUpdatesSchema(t *testing.T) {
	_, db := integrationDatabase(t)
	ctx := context.Background()
	if err := EnsureSchema(ctx, db); err != nil {
		t.Fatalf("premier EnsureSchema: %v", err)
	}
	if err := EnsureSchema(ctx, db); err != nil {
		t.Fatalf("second EnsureSchema: %v", err)
	}
	names, err := db.ListCollectionNames(ctx, bson.M{})
	if err != nil || len(names) != 1 || names[0] != "notifications" {
		t.Fatalf("collections = (%#v, %v)", names, err)
	}
	cursor, err := db.Collection("notifications").Indexes().List(ctx)
	if err != nil {
		t.Fatalf("liste index: %v", err)
	}
	defer cursor.Close(ctx)
	var specs []bson.M
	if err := cursor.All(ctx, &specs); err != nil {
		t.Fatalf("décodage index: %v", err)
	}
	if len(specs) != 5 {
		t.Fatalf("nombre d'index = %d, attendu 5", len(specs))
	}
}

func TestSchemaCanceledContextErrors(t *testing.T) {
	_, db := integrationDatabase(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := EnsureSchema(ctx, db); err == nil {
		t.Fatal("EnsureSchema devait échouer")
	}
	if err := ensureCollections(ctx, db); err == nil {
		t.Fatal("ensureCollections devait échouer")
	}
	if err := ensureIndexes(ctx, db); err == nil {
		t.Fatal("ensureIndexes devait échouer")
	}
}

func TestConnectMongo(t *testing.T) {
	uri := mongoIntegrationURI(t)
	client, err := ConnectMongo(uri)
	if err != nil {
		t.Fatalf("ConnectMongo valide: %v", err)
	}
	if err := client.Disconnect(context.Background()); err != nil {
		t.Fatalf("Disconnect: %v", err)
	}
	if _, err := ConnectMongo("://invalid"); err == nil {
		t.Fatal("ConnectMongo URI invalide devait échouer")
	}
	if _, err := ConnectMongo("mongodb://127.0.0.1:1/?serverSelectionTimeoutMS=10"); err == nil {
		t.Fatal("ConnectMongo serveur absent devait échouer au ping")
	}
}
