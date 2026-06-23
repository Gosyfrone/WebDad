package main

import (
	"context"
	"errors"
	"testing"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/webdad/profil-service/internal/config"
	"github.com/webdad/profil-service/internal/service"
)

func TestGinMode(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "release", in: gin.ReleaseMode, want: gin.ReleaseMode},
		{name: "test", in: gin.TestMode, want: gin.TestMode},
		{name: "empty defaults debug", in: "", want: gin.DebugMode},
		{name: "unknown defaults debug", in: "prod", want: gin.DebugMode},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ginMode(tc.in); got != tc.want {
				t.Fatalf("ginMode(%q) = %q, attendu %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestDefaultAppDeps(t *testing.T) {
	deps := defaultAppDeps()
	if deps.loadConfig == nil || deps.connectMongo == nil || deps.database == nil ||
		deps.ensureSchema == nil || deps.newProfilService == nil ||
		deps.registerRoutes == nil || deps.runServer == nil {
		t.Fatalf("default deps incompletes: %#v", deps)
	}

	if _, err := deps.connectMongo("://bad-uri"); err == nil {
		t.Fatal("connectMongo default devrait remonter une URI invalide")
	}

	client, err := mongo.Connect(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Fatalf("mongo.Connect: %v", err)
	}
	db := deps.database(&mongoClientAdapter{Client: client}, "profil_test")
	if db == nil {
		t.Fatal("database default retourne nil")
	}
	svc := deps.newProfilService(db, &config.Config{UserURL: "http://user-service"})
	if svc == nil {
		t.Fatal("newProfilService default retourne nil")
	}
}

type fakeAppMongoClient struct {
	disconnected bool
}

func (f *fakeAppMongoClient) Disconnect(context.Context) error {
	f.disconnected = true
	return nil
}

func testAppDeps(client *fakeAppMongoClient) appDeps {
	return appDeps{
		loadConfig: func() *config.Config {
			return &config.Config{
				Port:      "8083",
				GinMode:   gin.TestMode,
				MongoURI:  "mongodb://test",
				MongoDB:   "webdad_test",
				JWTSecret: "secret",
				UserURL:   "http://user-service",
			}
		},
		connectMongo: func(string) (appMongoClient, error) {
			return client, nil
		},
		database: func(appMongoClient, string) any {
			return "db"
		},
		ensureSchema: func(context.Context, any) error {
			return nil
		},
		newProfilService: func(any, *config.Config) *service.ProfilService {
			return nil
		},
		registerRoutes: func(*gin.Engine, string, *service.ProfilService, string) {},
		runServer: func(*gin.Engine, string) error {
			return nil
		},
	}
}

func TestRunWithDepsSuccess(t *testing.T) {
	client := &fakeAppMongoClient{}
	deps := testAppDeps(client)
	called := false
	deps.runServer = func(_ *gin.Engine, addr string) error {
		called = true
		if addr != ":8083" {
			t.Fatalf("addr = %q", addr)
		}
		return nil
	}

	if err := runWithDeps(deps); err != nil {
		t.Fatalf("runWithDeps erreur: %v", err)
	}
	if !called {
		t.Fatal("runServer non appele")
	}
	if !client.disconnected {
		t.Fatal("client Mongo non deconnecte")
	}
}

func TestRunWithDepsErrors(t *testing.T) {
	t.Run("connect", func(t *testing.T) {
		deps := testAppDeps(&fakeAppMongoClient{})
		sentinel := errors.New("connect")
		deps.connectMongo = func(string) (appMongoClient, error) { return nil, sentinel }
		if err := runWithDeps(deps); !errors.Is(err, sentinel) {
			t.Fatalf("attendu connect, obtenu %v", err)
		}
	})

	t.Run("schema", func(t *testing.T) {
		client := &fakeAppMongoClient{}
		deps := testAppDeps(client)
		sentinel := errors.New("schema")
		deps.ensureSchema = func(context.Context, any) error { return sentinel }
		if err := runWithDeps(deps); !errors.Is(err, sentinel) {
			t.Fatalf("attendu schema, obtenu %v", err)
		}
		if !client.disconnected {
			t.Fatal("client Mongo doit etre deconnecte meme en erreur schema")
		}
	})

	t.Run("server", func(t *testing.T) {
		deps := testAppDeps(&fakeAppMongoClient{})
		sentinel := errors.New("server")
		deps.runServer = func(*gin.Engine, string) error { return sentinel }
		if err := runWithDeps(deps); !errors.Is(err, sentinel) {
			t.Fatalf("attendu server, obtenu %v", err)
		}
	})
}
