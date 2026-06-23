package main

import (
	"context"
	"errors"
	"testing"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/webdad/notification-service/internal/config"
)

func testConfig() *config.Config {
	return &config.Config{
		Port: "8086", GinMode: gin.TestMode, MongoURI: "mongodb://unused", MongoDB: "notification_test",
		JWTSecret: "jwt", InternalSecret: "internal", UserServiceURL: "http://users.test",
	}
}

func disconnectedClient(t *testing.T) *mongo.Client {
	t.Helper()
	client, err := mongo.Connect(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Fatalf("mongo.Connect: %v", err)
	}
	return client
}

func TestDefaultRuntimeDependencies(t *testing.T) {
	deps := defaultRuntimeDependencies()
	if deps.connectMongo == nil || deps.ensureSchema == nil || deps.serve == nil {
		t.Fatal("dépendances runtime incomplètes")
	}
}

func TestRun(t *testing.T) {
	client := disconnectedClient(t)
	ensureCalled, serveCalled := false, false
	deps := runtimeDependencies{
		connectMongo: func(uri string) (*mongo.Client, error) {
			if uri != "mongodb://unused" {
				t.Fatalf("URI = %q", uri)
			}
			return client, nil
		},
		ensureSchema: func(_ context.Context, db *mongo.Database) error {
			ensureCalled = true
			if db.Name() != "notification_test" {
				t.Fatalf("database = %q", db.Name())
			}
			return nil
		},
		serve: func(engine *gin.Engine, addresses ...string) error {
			serveCalled = true
			if len(addresses) != 1 || addresses[0] != ":8086" {
				t.Fatalf("addresses = %#v", addresses)
			}
			if len(engine.Routes()) == 0 {
				t.Fatal("aucune route enregistrée")
			}
			return nil
		},
	}
	if err := run(testConfig(), deps); err != nil {
		t.Fatalf("run = %v", err)
	}
	if !ensureCalled || !serveCalled {
		t.Fatalf("ensure=%v serve=%v", ensureCalled, serveCalled)
	}
}

func TestRunErrors(t *testing.T) {
	wantErr := errors.New("runtime")
	t.Run("connexion", func(t *testing.T) {
		deps := runtimeDependencies{connectMongo: func(string) (*mongo.Client, error) { return nil, wantErr }}
		if err := run(testConfig(), deps); !errors.Is(err, wantErr) {
			t.Fatalf("run = %v", err)
		}
	})
	t.Run("schéma", func(t *testing.T) {
		deps := runtimeDependencies{
			connectMongo: func(string) (*mongo.Client, error) { return disconnectedClient(t), nil },
			ensureSchema: func(context.Context, *mongo.Database) error { return wantErr },
		}
		if err := run(testConfig(), deps); !errors.Is(err, wantErr) {
			t.Fatalf("run = %v", err)
		}
	})
	t.Run("serveur", func(t *testing.T) {
		deps := runtimeDependencies{
			connectMongo: func(string) (*mongo.Client, error) { return disconnectedClient(t), nil },
			ensureSchema: func(context.Context, *mongo.Database) error { return nil },
			serve:        func(*gin.Engine, ...string) error { return wantErr },
		}
		if err := run(testConfig(), deps); !errors.Is(err, wantErr) {
			t.Fatalf("run = %v", err)
		}
	})
}

func TestGinMode(t *testing.T) {
	for _, tt := range []struct {
		input string
		want  string
	}{
		{gin.ReleaseMode, gin.ReleaseMode},
		{gin.TestMode, gin.TestMode},
		{"invalid", gin.DebugMode},
	} {
		if got := ginMode(tt.input); got != tt.want {
			t.Errorf("ginMode(%q) = %q, attendu %q", tt.input, got, tt.want)
		}
	}
}
