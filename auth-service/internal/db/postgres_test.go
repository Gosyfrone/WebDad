package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"
	"time"
)

const fakeDriverName = "auth_db_test"

var fakeDBErr error

type fakeDriver struct{}
type fakeConn struct{}
type fakeResult struct{}

func (fakeDriver) Open(string) (driver.Conn, error)  { return fakeConn{}, nil }
func (fakeConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("unused") }
func (fakeConn) Close() error                        { return nil }
func (fakeConn) Begin() (driver.Tx, error)           { return nil, errors.New("unused") }
func (fakeConn) Ping(context.Context) error          { return fakeDBErr }
func (fakeConn) ExecContext(context.Context, string, []driver.NamedValue) (driver.Result, error) {
	return fakeResult{}, fakeDBErr
}
func (fakeResult) LastInsertId() (int64, error) { return 0, nil }
func (fakeResult) RowsAffected() (int64, error) { return 1, nil }

func init() { sql.Register(fakeDriverName, fakeDriver{}) }

func TestConnectAndEnsureSchema(t *testing.T) {
	oldDriver, oldAttempts, oldWait := driverName, pingAttempts, pingRetryWait
	t.Cleanup(func() { driverName, pingAttempts, pingRetryWait = oldDriver, oldAttempts, oldWait })
	driverName, pingAttempts, pingRetryWait = fakeDriverName, 2, time.Nanosecond

	fakeDBErr = nil
	conn, err := Connect("ignored")
	if err != nil {
		t.Fatalf("Connect success: %v", err)
	}
	defer conn.Close()
	if err := EnsureSchema(conn); err != nil {
		t.Fatalf("EnsureSchema success: %v", err)
	}

	fakeDBErr = errors.New("offline")
	if _, err := Connect("ignored"); err == nil {
		t.Fatal("expected ping failure")
	}
	if err := EnsureSchema(conn); err == nil {
		t.Fatal("expected schema failure")
	}

	driverName = "missing-driver"
	if _, err := Connect("ignored"); err == nil {
		t.Fatal("expected open failure")
	}
}
