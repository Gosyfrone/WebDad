// Package db gère la connexion PostgreSQL et l'application du schéma.
//
// Le schéma (schema.sql, embarqué via go:embed) est appliqué au démarrage
// de façon idempotente : le service est autonome (il bootstrappe sa propre
// base), aussi bien en local qu'en stack Docker.
package db

import (
	"database/sql"
	_ "embed"
	"fmt"
	"time"

	_ "github.com/lib/pq" // driver PostgreSQL (database/sql)
)

//go:embed schema.sql
var schemaSQL string

var (
	driverName    = "postgres"
	pingAttempts  = 10
	pingRetryWait = 2 * time.Second
)

// Connect ouvre un pool de connexions et attend que la base réponde.
// Le retry couvre le cas où le service démarre avant que PostgreSQL
// accepte les connexions (le healthcheck compose limite déjà ce risque).
func Connect(dsn string) (*sql.DB, error) {
	conn, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("ouverture connexion : %w", err)
	}

	conn.SetMaxOpenConns(10)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(time.Hour)

	var pingErr error
	for i := 0; i < pingAttempts; i++ {
		if pingErr = conn.Ping(); pingErr == nil {
			return conn, nil
		}
		time.Sleep(pingRetryWait)
	}
	return nil, fmt.Errorf("ping DB après plusieurs tentatives : %w", pingErr)
}

// EnsureSchema applique le schéma embarqué (idempotent).
func EnsureSchema(conn *sql.DB) error {
	if _, err := conn.Exec(schemaSQL); err != nil {
		return fmt.Errorf("application du schéma : %w", err)
	}
	return nil
}
