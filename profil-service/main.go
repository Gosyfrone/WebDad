package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/webdad/profil-service/internal/client"
	"github.com/webdad/profil-service/internal/config"
	"github.com/webdad/profil-service/internal/database"
	"github.com/webdad/profil-service/internal/handler"
	"github.com/webdad/profil-service/internal/logging"
	"github.com/webdad/profil-service/internal/middleware"
	"github.com/webdad/profil-service/internal/repository"
	"github.com/webdad/profil-service/internal/service"
)

const serviceName = "profil-service"

func main() {
	logging.Setup(serviceName)
	if err := runWithDeps(defaultAppDeps()); err != nil {
		slog.Error("arrêt du service", "error", err)
		os.Exit(1)
	}
}

type appMongoClient interface {
	Disconnect(ctx context.Context) error
}

type appDeps struct {
	loadConfig       func() *config.Config
	connectMongo     func(uri string) (appMongoClient, error)
	database         func(client appMongoClient, name string) any
	ensureSchema     func(ctx context.Context, db any) error
	newProfilService func(db any, cfg *config.Config) *service.ProfilService
	registerRoutes   func(r *gin.Engine, serviceName string, profils *service.ProfilService, jwtSecret string, emitters ...handler.IdentityEmitter)
	runServer        func(r *gin.Engine, addr string) error
}

func defaultAppDeps() appDeps {
	return appDeps{
		loadConfig: config.Load,
		connectMongo: func(uri string) (appMongoClient, error) {
			client, err := database.ConnectMongo(uri)
			if err != nil {
				return nil, err
			}
			return &mongoClientAdapter{Client: client}, nil
		},
		database: func(client appMongoClient, name string) any {
			return client.(*mongoClientAdapter).Database(name)
		},
		ensureSchema: func(ctx context.Context, db any) error {
			return database.EnsureSchema(ctx, db.(*mongoDatabaseAdapter).Database)
		},
		newProfilService: func(db any, cfg *config.Config) *service.ProfilService {
			return service.New(
				repository.NewProfilRepository(db.(*mongoDatabaseAdapter).Database),
				cfg.DisplayNameCooldown,
				service.WithFollowChecker(client.NewUserClient(cfg.UserURL)),
			)
		},
		registerRoutes: handler.RegisterRoutes,
		runServer: func(r *gin.Engine, addr string) error {
			return r.Run(addr)
		},
	}
}

type mongoClientAdapter struct {
	*mongo.Client
}

func (m *mongoClientAdapter) Database(name string) *mongoDatabaseAdapter {
	return &mongoDatabaseAdapter{Database: m.Client.Database(name)}
}

type mongoDatabaseAdapter struct {
	*mongo.Database
}

func runWithDeps(deps appDeps) error {
	cfg := deps.loadConfig()
	gin.SetMode(ginMode(cfg.GinMode))

	mongoClient, err := deps.connectMongo(cfg.MongoURI)
	if err != nil {
		return err
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = mongoClient.Disconnect(ctx)
	}()

	db := deps.database(mongoClient, cfg.MongoDB)

	// Le service applique son propre schéma (collection + validateur + index +
	// seed admin) au démarrage, de façon idempotente → autonome, sans script
	// d'init externe.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := deps.ensureSchema(ctx, db); err != nil {
		return err
	}

	// On logue l'URL user-service résolue : un `http://localhost:...` ici en
	// conteneur = USER_SERVICE_URL absent de l'environnement (conteneur à recréer)
	// → les appels inter-services (is-following, accept-all) échoueraient.
	slog.Info("user-service", "url", cfg.UserURL)
	slog.Info("notification-service", "url", cfg.NotificationURL)
	profils := deps.newProfilService(db, cfg)
	notifications := client.NewNotificationClient(cfg.NotificationURL, cfg.InternalEventSecret)

	r := gin.New()
	r.Use(middleware.RequestID(), middleware.Recovery(), middleware.RequestLogger())
	deps.registerRoutes(r, serviceName, profils, cfg.JWTSecret, notifications)

	slog.Info("en écoute", "port", cfg.Port)
	return deps.runServer(r, ":"+cfg.Port)
}

// ginMode borne la valeur de GIN_MODE aux modes connus (défaut : debug).
func ginMode(mode string) string {
	switch mode {
	case gin.ReleaseMode, gin.TestMode:
		return mode
	default:
		return gin.DebugMode
	}
}
