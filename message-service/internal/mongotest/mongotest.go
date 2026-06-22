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

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Mot de passe de l'utilisateur Mongo EN LECTURE SEULE créé par ReadOnlyDB (le
// NOM est unique par appel — cf. roUserName — pour éviter toute collision entre
// binaires de test exécutés en parallèle par `go test ./...`).
const roPass = "msgtest_ro_pass"

// roUserName fabrique un nom d'utilisateur unique (pid + compteur atomique) →
// pas de drop/create concurrent sur un nom partagé.
func roUserName() string {
	return fmt.Sprintf("msgtest_ro_%d_%d", os.Getpid(), atomic.AddInt64(&counter, 1))
}

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
	// Petit pool + sélection serveur tolérante : `go test ./...` lance les
	// binaires de package en parallèle (× -race) → on limite le nombre total de
	// connexions ouvertes sur la même Mongo pour éviter les coupures côté serveur.
	opts := options.Client().ApplyURI(uri).
		SetServerSelectionTimeout(15 * time.Second).
		SetConnectTimeout(15 * time.Second).
		SetMaxPoolSize(5).
		SetRetryReads(true).
		SetRetryWrites(true)
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

// ReadOnlyDB renvoie DEUX vues d'une même base jetable :
//   - root : client partagé, en LECTURE/ÉCRITURE → sert à PEUPLER les données ;
//   - ro   : client dédié authentifié comme un utilisateur en LECTURE SEULE →
//     les lectures réussissent mais toute ÉCRITURE échoue (« Unauthorized »).
//
// Cela permet d'exercer, avec une vraie Mongo et sans toucher au code de prod,
// les branches d'erreur situées sur un appel repo EN ÉCRITURE qui suit des
// lectures réussies (contrôles d'appartenance OK, puis l'écriture échoue).
func ReadOnlyDB(t *testing.T) (root *mongo.Database, ro *mongo.Database) {
	t.Helper()
	root = DB(t) // base writable sur le client partagé (drop au cleanup)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	admin := sharedClient.Database("admin")
	user := roUserName() // unique → pas de collision entre binaires de test parallèles
	if err := admin.RunCommand(ctx, bson.D{
		{Key: "createUser", Value: user},
		{Key: "pwd", Value: roPass},
		{Key: "roles", Value: bson.A{bson.D{{Key: "role", Value: "readAnyDatabase"}, {Key: "db", Value: "admin"}}}},
	}).Err(); err != nil {
		// Droits insuffisants pour gérer les utilisateurs → on ignore proprement
		// (le test dépendant de ReadOnlyDB est skip, pas en échec).
		t.Skipf("création de l'utilisateur Mongo lecture seule impossible : %v", err)
	}

	roClient, cerr := mongo.Connect(options.Client().
		ApplyURI(uri(t)).
		SetAuth(options.Credential{Username: user, Password: roPass, AuthSource: "admin"}).
		SetServerSelectionTimeout(15 * time.Second).
		SetMaxPoolSize(3))
	if cerr != nil {
		t.Fatalf("connexion Mongo lecture seule : %v", cerr)
	}
	if perr := roClient.Ping(ctx, nil); perr != nil {
		t.Fatalf("ping Mongo lecture seule : %v", perr)
	}
	t.Cleanup(func() {
		c, cc := context.WithTimeout(context.Background(), 10*time.Second)
		defer cc()
		_ = roClient.Disconnect(c)
		_ = admin.RunCommand(c, bson.D{{Key: "dropUser", Value: user}}).Err()
	})
	return root, roClient.Database(root.Name())
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
