package repository

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"sync"
	"testing"
	"time"
)

const scriptedDriverName = "user_repo_scripted_driver"

type scriptedResponse struct {
	columns  []string
	rows     [][]driver.Value
	err      error
	affected int64
}

var scriptedState struct {
	sync.Mutex
	once  sync.Once
	queue []*scriptedResponse
}

func scriptedDB(t *testing.T, responses ...*scriptedResponse) *sql.DB {
	t.Helper()
	scriptedState.once.Do(func() { sql.Register(scriptedDriverName, scriptedDriver{}) })
	scriptedState.Lock()
	scriptedState.queue = responses
	scriptedState.Unlock()
	db, err := sql.Open(scriptedDriverName, "")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func scriptedPop() *scriptedResponse {
	scriptedState.Lock()
	defer scriptedState.Unlock()
	if len(scriptedState.queue) == 0 {
		return &scriptedResponse{affected: 1}
	}
	r := scriptedState.queue[0]
	scriptedState.queue = scriptedState.queue[1:]
	return r
}

type scriptedDriver struct{}
type scriptedConn struct{}
type scriptedStmt struct{}
type scriptedTx struct{}
type scriptedRows struct {
	response *scriptedResponse
	index    int
}
type scriptedResult struct{ affected int64 }

func (scriptedDriver) Open(string) (driver.Conn, error)  { return scriptedConn{}, nil }
func (scriptedConn) Prepare(string) (driver.Stmt, error) { return scriptedStmt{}, nil }
func (scriptedConn) Close() error                        { return nil }
func (scriptedConn) Begin() (driver.Tx, error)           { return scriptedTx{}, nil }
func (scriptedStmt) Close() error                        { return nil }
func (scriptedStmt) NumInput() int                       { return -1 }
func (scriptedStmt) Exec([]driver.Value) (driver.Result, error) {
	r := scriptedPop()
	if r.err != nil {
		return nil, r.err
	}
	return scriptedResult{r.affected}, nil
}
func (scriptedStmt) Query([]driver.Value) (driver.Rows, error) {
	r := scriptedPop()
	if r.err != nil {
		return nil, r.err
	}
	return &scriptedRows{response: r}, nil
}
func (scriptedTx) Commit() error          { return nil }
func (scriptedTx) Rollback() error        { return nil }
func (r *scriptedRows) Columns() []string { return r.response.columns }
func (r *scriptedRows) Close() error      { return nil }
func (r *scriptedRows) Next(dest []driver.Value) error {
	if r.index >= len(r.response.rows) {
		return io.EOF
	}
	copy(dest, r.response.rows[r.index])
	r.index++
	return nil
}
func (scriptedResult) LastInsertId() (int64, error)   { return 0, nil }
func (r scriptedResult) RowsAffected() (int64, error) { return r.affected, nil }

func execResponse(affected int64) *scriptedResponse { return &scriptedResponse{affected: affected} }

func idRows(ids ...string) *scriptedResponse {
	rows := make([][]driver.Value, len(ids))
	for i, id := range ids {
		rows[i] = []driver.Value{id}
	}
	return &scriptedResponse{columns: []string{"id"}, rows: rows}
}

func userRows() *scriptedResponse {
	now := time.Now().UTC()
	return &scriptedResponse{
		columns: []string{"id", "username", "is_active", "created_at", "updated_at", "username_changed_at", "username_pending", "preferred_locale"},
		rows: [][]driver.Value{
			{"u1", "alice", true, now, now, nil, false, "fr"},
			{"u2", "bob", true, now, now, nil, false, nil},
		},
	}
}

func TestRepository_ScriptedListAndIDQueries(t *testing.T) {
	repo := New(scriptedDB(t, userRows(), idRows("u2", "u3"), idRows("u4"), idRows("u5", "u6")))
	users, err := repo.List(10, 0)
	if err != nil || len(users) != 2 || users[0].Username != "alice" {
		t.Fatalf("List = %+v, %v", users, err)
	}
	pending, err := repo.PendingFollowRequestIDs("u1")
	if err != nil || len(pending) != 2 {
		t.Fatalf("Pending = %v, %v", pending, err)
	}
	incoming, err := repo.IncomingFollowRequestFollowerIDs("u1")
	if err != nil || len(incoming) != 1 {
		t.Fatalf("Incoming = %v, %v", incoming, err)
	}
	blocked, err := repo.BlockedIDs("u1")
	if err != nil || len(blocked) != 2 {
		t.Fatalf("Blocked = %v, %v", blocked, err)
	}
}

func TestRepository_ScriptedMutations(t *testing.T) {
	repo := New(scriptedDB(t,
		execResponse(1),
		execResponse(0),
		execResponse(1), execResponse(1), execResponse(1),
		execResponse(1), execResponse(1), execResponse(1),
		execResponse(1), execResponse(1),
	))
	if err := repo.SetActive("u1", false); err != nil {
		t.Fatal(err)
	}
	if err := repo.SetActive("missing", false); err != sql.ErrNoRows {
		t.Fatalf("SetActive absent = %v", err)
	}
	if err := repo.PurgeUser("u1"); err != nil {
		t.Fatal(err)
	}
	if err := repo.Block("u1", "u2"); err != nil {
		t.Fatal(err)
	}
	if err := repo.Unfollow("u1", "u2"); err != nil {
		t.Fatal(err)
	}
}

func TestRepository_ScriptedAcceptFollowRequest(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		repo := New(scriptedDB(t, execResponse(0)))
		accepted, err := repo.AcceptFollowRequest("u1", "u2")
		if err != nil || accepted {
			t.Fatalf("accepted=%v err=%v", accepted, err)
		}
	})
	t.Run("accepted", func(t *testing.T) {
		repo := New(scriptedDB(t, execResponse(1), execResponse(1)))
		accepted, err := repo.AcceptFollowRequest("u1", "u2")
		if err != nil || !accepted {
			t.Fatalf("accepted=%v err=%v", accepted, err)
		}
	})
}

func TestRepository_QueryUsersScanError(t *testing.T) {
	bad := &scriptedResponse{
		columns: userRows().columns,
		rows:    [][]driver.Value{{"u1", "alice", "not-a-bool", time.Now(), time.Now(), nil, false, nil}},
	}
	repo := New(scriptedDB(t, bad))
	if _, err := repo.queryUsers("SELECT"); err == nil {
		t.Fatal("une erreur de scan était attendue")
	}
}

func TestScriptedDriverContextCompatibility(t *testing.T) {
	db := scriptedDB(t, idRows("u1"))
	rows, err := db.QueryContext(context.Background(), "SELECT id")
	if err != nil {
		t.Fatal(err)
	}
	_ = rows.Close()
}
