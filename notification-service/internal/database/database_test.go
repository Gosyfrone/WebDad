package database

import (
	"context"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func restoreSchemaFunctions(t *testing.T) {
	t.Helper()
	originalList := listCollectionNames
	originalRun := runCommand
	originalCreate := createCollection
	originalIndexes := createIndexes
	t.Cleanup(func() {
		listCollectionNames = originalList
		runCommand = originalRun
		createCollection = originalCreate
		createIndexes = originalIndexes
	})
}

func successfulSchemaFunctions() {
	listCollectionNames = func(context.Context, *mongo.Database) ([]string, error) { return nil, nil }
	runCommand = func(context.Context, *mongo.Database, bson.D) error { return nil }
	createCollection = func(context.Context, *mongo.Database, string, *options.CreateCollectionOptionsBuilder) error {
		return nil
	}
	createIndexes = func(context.Context, *mongo.Database, string, []mongo.IndexModel) error { return nil }
}

func TestEnsureSchemaCreatesMissingCollection(t *testing.T) {
	restoreSchemaFunctions(t)
	successfulSchemaFunctions()
	created, indexed := "", ""
	createCollection = func(_ context.Context, _ *mongo.Database, name string, _ *options.CreateCollectionOptionsBuilder) error {
		created = name
		return nil
	}
	createIndexes = func(_ context.Context, _ *mongo.Database, name string, models []mongo.IndexModel) error {
		indexed = name
		if len(models) != 4 {
			t.Fatalf("index = %d, attendu 4", len(models))
		}
		return nil
	}
	if err := EnsureSchema(context.Background(), nil); err != nil {
		t.Fatalf("EnsureSchema = %v", err)
	}
	if created != "notifications" || indexed != "notifications" {
		t.Fatalf("created=%q indexed=%q", created, indexed)
	}
}

func TestEnsureSchemaUpdatesExistingCollection(t *testing.T) {
	restoreSchemaFunctions(t)
	successfulSchemaFunctions()
	listCollectionNames = func(context.Context, *mongo.Database) ([]string, error) {
		return []string{"notifications", "unrelated"}, nil
	}
	commands := 0
	runCommand = func(_ context.Context, _ *mongo.Database, command bson.D) error {
		commands++
		if len(command) == 0 || command[0].Key != "collMod" || command[0].Value != "notifications" {
			t.Fatalf("commande = %#v", command)
		}
		return nil
	}
	if err := EnsureSchema(context.Background(), nil); err != nil {
		t.Fatalf("EnsureSchema = %v", err)
	}
	if commands != 1 {
		t.Fatalf("commandes = %d", commands)
	}
}

func TestSchemaErrors(t *testing.T) {
	wantErr := errors.New("mongo")
	tests := []struct {
		name  string
		setup func()
		call  func() error
	}{
		{"liste", func() {
			listCollectionNames = func(context.Context, *mongo.Database) ([]string, error) { return nil, wantErr }
		}, func() error { return ensureCollections(context.Background(), nil) }},
		{"collMod", func() {
			listCollectionNames = func(context.Context, *mongo.Database) ([]string, error) { return []string{"notifications"}, nil }
			runCommand = func(context.Context, *mongo.Database, bson.D) error { return wantErr }
		}, func() error { return ensureCollections(context.Background(), nil) }},
		{"création", func() {
			createCollection = func(context.Context, *mongo.Database, string, *options.CreateCollectionOptionsBuilder) error {
				return wantErr
			}
		}, func() error { return ensureCollections(context.Background(), nil) }},
		{"index", func() {
			createIndexes = func(context.Context, *mongo.Database, string, []mongo.IndexModel) error { return wantErr }
		}, func() error { return ensureIndexes(context.Background(), nil) }},
		{"propagation collections", func() {
			listCollectionNames = func(context.Context, *mongo.Database) ([]string, error) { return nil, wantErr }
		}, func() error { return EnsureSchema(context.Background(), nil) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restoreSchemaFunctions(t)
			successfulSchemaFunctions()
			tt.setup()
			if err := tt.call(); !errors.Is(err, wantErr) {
				t.Fatalf("erreur = %v, attendu %v", err, wantErr)
			}
		})
	}
}

func TestEnsureIndexesSkipsEmptySpecification(t *testing.T) {
	restoreSchemaFunctions(t)
	successfulSchemaFunctions()
	original := indexes["notifications"]
	indexes["notifications"] = nil
	t.Cleanup(func() { indexes["notifications"] = original })
	called := false
	createIndexes = func(context.Context, *mongo.Database, string, []mongo.IndexModel) error {
		called = true
		return nil
	}
	if err := ensureIndexes(context.Background(), nil); err != nil || called {
		t.Fatalf("ensureIndexes = %v, called=%v", err, called)
	}
}

func TestDefaultMongoSchemaFunctions(t *testing.T) {
	client, err := mongo.Connect(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Fatalf("mongo.Connect: %v", err)
	}
	t.Cleanup(func() { _ = client.Disconnect(context.Background()) })
	db := client.Database("notification_test")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := listCollectionNames(ctx, db); err == nil {
		t.Error("listCollectionNames devait échouer")
	}
	if err := runCommand(ctx, db, bson.D{{Key: "ping", Value: 1}}); err == nil {
		t.Error("runCommand devait échouer")
	}
	if err := createCollection(ctx, db, "notifications", options.CreateCollection()); err == nil {
		t.Error("createCollection devait échouer")
	}
	if err := createIndexes(ctx, db, "notifications", indexes["notifications"]); err == nil {
		t.Error("createIndexes devait échouer")
	}
}

func TestConnectMongoUnit(t *testing.T) {
	originalConnect, originalPing := connectClient, pingClient
	t.Cleanup(func() { connectClient, pingClient = originalConnect, originalPing })
	wantErr := errors.New("mongo")

	connectClient = func(*options.ClientOptions) (*mongo.Client, error) { return nil, wantErr }
	if client, err := ConnectMongo("mongodb://unused"); !errors.Is(err, wantErr) || client != nil {
		t.Fatalf("ConnectMongo connect = (%#v, %v)", client, err)
	}

	dummy := &mongo.Client{}
	connectClient = func(*options.ClientOptions) (*mongo.Client, error) { return dummy, nil }
	pingClient = func(context.Context, *mongo.Client) error { return wantErr }
	if client, err := ConnectMongo("mongodb://unused"); !errors.Is(err, wantErr) || client != nil {
		t.Fatalf("ConnectMongo ping = (%#v, %v)", client, err)
	}

	pingClient = func(context.Context, *mongo.Client) error { return nil }
	if client, err := ConnectMongo("mongodb://unused"); err != nil || client != dummy {
		t.Fatalf("ConnectMongo succès = (%#v, %v)", client, err)
	}
}

func TestDefaultConnectFunctions(t *testing.T) {
	client, err := connectClient(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Fatalf("connectClient = %v", err)
	}
	t.Cleanup(func() { _ = client.Disconnect(context.Background()) })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := pingClient(ctx, client); err == nil {
		t.Fatal("pingClient devait échouer avec un contexte annulé")
	}
}
