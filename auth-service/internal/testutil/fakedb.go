package testutil

import (
	"database/sql"
	"database/sql/driver"
	"io"
	"sync"
)

const DriverName = "auth_shared_test_driver"

type Response struct {
	Columns  []string
	Rows     [][]driver.Value
	Err      error
	Affected int64
}

var (
	once  sync.Once
	mu    sync.Mutex
	queue []*Response
)

func Open(responses ...*Response) *sql.DB {
	once.Do(func() { sql.Register(DriverName, fakeDriver{}) })
	mu.Lock()
	queue = responses
	mu.Unlock()
	db, _ := sql.Open(DriverName, "")
	db.SetMaxOpenConns(1)
	return db
}

func Set(responses ...*Response) { mu.Lock(); queue = responses; mu.Unlock() }
func Empty() *Response           { return &Response{Affected: 1} }
func Error(err error) *Response  { return &Response{Err: err} }
func Row(columns []string, values ...driver.Value) *Response {
	return &Response{Columns: columns, Rows: [][]driver.Value{values}, Affected: 1}
}

func pop() *Response {
	mu.Lock()
	defer mu.Unlock()
	if len(queue) == 0 {
		return Empty()
	}
	r := queue[0]
	queue = queue[1:]
	return r
}

type fakeDriver struct{}
type fakeConn struct{}
type fakeStmt struct{}
type fakeTx struct{}
type fakeRows struct {
	response *Response
	index    int
}
type fakeResult struct{ affected int64 }

func (fakeDriver) Open(string) (driver.Conn, error)  { return fakeConn{}, nil }
func (fakeConn) Prepare(string) (driver.Stmt, error) { return fakeStmt{}, nil }
func (fakeConn) Close() error                        { return nil }
func (fakeConn) Begin() (driver.Tx, error)           { return fakeTx{}, nil }
func (fakeStmt) Close() error                        { return nil }
func (fakeStmt) NumInput() int                       { return -1 }
func (fakeStmt) Exec([]driver.Value) (driver.Result, error) {
	r := pop()
	if r.Err != nil {
		return nil, r.Err
	}
	affected := r.Affected
	if affected == 0 {
		affected = 1
	}
	return fakeResult{affected}, nil
}
func (fakeStmt) Query([]driver.Value) (driver.Rows, error) {
	r := pop()
	if r.Err != nil {
		return nil, r.Err
	}
	return &fakeRows{response: r}, nil
}
func (fakeTx) Commit() error          { return nil }
func (fakeTx) Rollback() error        { return nil }
func (r *fakeRows) Columns() []string { return r.response.Columns }
func (r *fakeRows) Close() error      { return nil }
func (r *fakeRows) Next(dest []driver.Value) error {
	if r.index >= len(r.response.Rows) {
		return io.EOF
	}
	copy(dest, r.response.Rows[r.index])
	r.index++
	return nil
}
func (r fakeResult) LastInsertId() (int64, error) { return 0, nil }
func (r fakeResult) RowsAffected() (int64, error) { return r.affected, nil }
