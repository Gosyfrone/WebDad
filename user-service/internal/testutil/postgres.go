// Package testutil fournit des aides partagées aux tests d'intégration du
// user-service : chaque test reçoit une base PostgreSQL FRAÎCHE et isolée (le
// schéma y est appliqué), supprimée au nettoyage. Importé UNIQUEMENT par des
// fichiers _test.go : aucun binaire de production n'en dépend.
package testutil

import (
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/lib/pq" // driver PostgreSQL (database/sql)

	"github.com/webdad/user-service/internal/db"
)

// dsnEnv : variable d'environnement portant le DSN PostgreSQL de test (vers une
// base d'administration existante, p. ex. `postgres`, depuis laquelle on crée
// et détruit une base éphémère par test).
const dsnEnv = "USER_TEST_DSN"

var (
	admin    *sql.DB
	adminErr error
	once     sync.Once
	dbSeq    int64
)

// dbNameRe repère le token dbname=<valeur> dans un DSN au format clé=valeur.
var dbNameRe = regexp.MustCompile(`dbname=\S+`)

// DSN renvoie le DSN PostgreSQL d'administration (USER_TEST_DSN), ou "" si non défini.
func DSN() string { return os.Getenv(dsnEnv) }

// DB renvoie une base PostgreSQL FRAÎCHE et isolée pour le test, schéma appliqué
// (EnsureSchema) et tables vidées (la ligne admin semée par le schéma est
// retirée pour partir d'une base réellement vierge). La base est supprimée et la
// connexion fermée au nettoyage du test.
//
// Si USER_TEST_DSN n'est pas défini (run local sans Postgres), le test est
// ignoré (t.Skip) plutôt qu'en échec. Une base injoignable est aussi un skip.
func DB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := DSN()
	if dsn == "" {
		t.Skip("USER_TEST_DSN non défini — test d'intégration PostgreSQL ignoré")
	}

	once.Do(func() {
		admin, adminErr = sql.Open("postgres", dsn)
		if adminErr == nil {
			adminErr = admin.Ping()
		}
	})
	if adminErr != nil {
		t.Skipf("Postgres injoignable (%s) : %v — test ignoré", dsn, adminErr)
	}

	name := fmt.Sprintf("usertest_%d_%d", time.Now().UnixNano(), atomic.AddInt64(&dbSeq, 1))
	if _, err := admin.Exec("CREATE DATABASE " + name); err != nil {
		t.Fatalf("CREATE DATABASE %s : %v", name, err)
	}

	childDSN := dbNameRe.ReplaceAllString(dsn, "dbname="+name)
	if !dbNameRe.MatchString(dsn) {
		childDSN = dsn + " dbname=" + name
	}

	conn, err := sql.Open("postgres", childDSN)
	if err != nil {
		dropDatabase(t, name)
		t.Fatalf("ouverture base de test : %v", err)
	}
	if err := db.EnsureSchema(conn); err != nil {
		_ = conn.Close()
		dropDatabase(t, name)
		t.Fatalf("EnsureSchema : %v", err)
	}
	// Le schéma sème un admin par défaut : on le retire pour une base vierge.
	if _, err := conn.Exec(`TRUNCATE users, follows, follow_requests, blocks RESTART IDENTITY CASCADE`); err != nil {
		_ = conn.Close()
		dropDatabase(t, name)
		t.Fatalf("TRUNCATE initial : %v", err)
	}

	t.Cleanup(func() {
		_ = conn.Close()
		dropDatabase(t, name)
	})
	return conn
}

// dropDatabase supprime la base éphémère (best-effort, après fermeture du pool).
func dropDatabase(t *testing.T, name string) {
	t.Helper()
	if _, err := admin.Exec("DROP DATABASE IF EXISTS " + name + " WITH (FORCE)"); err != nil {
		t.Logf("DROP DATABASE %s : %v", name, err)
	}
}
