package db

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

// testDSN renvoie le DSN d'administration de test, ou skip si absent.
func testDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("USER_TEST_DSN")
	if dsn == "" {
		t.Skip("USER_TEST_DSN non défini — test d'intégration PostgreSQL ignoré")
	}
	return dsn
}

func TestConnect_AndEnsureSchema(t *testing.T) {
	dsn := testDSN(t)

	conn, err := Connect(dsn)
	if err != nil {
		t.Skipf("Postgres injoignable (%s) : %v — test ignoré", dsn, err)
	}
	defer func() { _ = conn.Close() }()

	// EnsureSchema est idempotent : on l'applique deux fois.
	if err := EnsureSchema(conn); err != nil {
		t.Fatalf("EnsureSchema#1 : %v", err)
	}
	if err := EnsureSchema(conn); err != nil {
		t.Fatalf("EnsureSchema#2 (idempotence) : %v", err)
	}

	// Les tables attendues existent.
	var n int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM information_schema.tables
		WHERE table_name IN ('users','follows','follow_requests','blocks')`).Scan(&n); err != nil {
		t.Fatalf("vérification des tables : %v", err)
	}
	if n != 4 {
		t.Fatalf("tables créées = %d, attendu 4", n)
	}
}

func TestEnsureSchema_ClosedConn(t *testing.T) {
	conn, err := sql.Open("postgres", "host=127.0.0.1 port=1 dbname=x sslmode=disable")
	if err != nil {
		t.Fatalf("sql.Open : %v", err)
	}
	_ = conn.Close()
	if err := EnsureSchema(conn); err == nil {
		t.Fatal("EnsureSchema doit échouer sur une connexion fermée")
	}
}
