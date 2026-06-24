package service

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/webdad/user-service/internal/client"
	"github.com/webdad/user-service/internal/repository"
)

const serviceDriverName = "user_service_scripted_driver"

type serviceResponse struct {
	columns  []string
	rows     [][]driver.Value
	err      error
	affected int64
}

var serviceSQL struct {
	sync.Mutex
	once  sync.Once
	queue []*serviceResponse
}

func scriptedService(t *testing.T, responses ...*serviceResponse) *UserService {
	t.Helper()
	serviceSQL.once.Do(func() { sql.Register(serviceDriverName, serviceDriver{}) })
	serviceSQL.Lock()
	serviceSQL.queue = responses
	serviceSQL.Unlock()
	db, err := sql.Open(serviceDriverName, "")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return New(repository.New(db), 0)
}

func servicePop() *serviceResponse {
	serviceSQL.Lock()
	defer serviceSQL.Unlock()
	if len(serviceSQL.queue) == 0 {
		return &serviceResponse{affected: 1}
	}
	r := serviceSQL.queue[0]
	serviceSQL.queue = serviceSQL.queue[1:]
	return r
}

type serviceDriver struct{}
type serviceConn struct{}
type serviceStmt struct{}
type serviceTx struct{}
type serviceRows struct {
	r *serviceResponse
	i int
}
type serviceResult int64

func (serviceDriver) Open(string) (driver.Conn, error)  { return serviceConn{}, nil }
func (serviceConn) Prepare(string) (driver.Stmt, error) { return serviceStmt{}, nil }
func (serviceConn) Close() error                        { return nil }
func (serviceConn) Begin() (driver.Tx, error)           { return serviceTx{}, nil }
func (serviceStmt) Close() error                        { return nil }
func (serviceStmt) NumInput() int                       { return -1 }
func (serviceStmt) Exec([]driver.Value) (driver.Result, error) {
	r := servicePop()
	if r.err != nil {
		return nil, r.err
	}
	return serviceResult(r.affected), nil
}
func (serviceStmt) Query([]driver.Value) (driver.Rows, error) {
	r := servicePop()
	if r.err != nil {
		return nil, r.err
	}
	return &serviceRows{r: r}, nil
}
func (serviceTx) Commit() error          { return nil }
func (serviceTx) Rollback() error        { return nil }
func (r *serviceRows) Columns() []string { return r.r.columns }
func (r *serviceRows) Close() error      { return nil }
func (r *serviceRows) Next(dest []driver.Value) error {
	if r.i >= len(r.r.rows) {
		return io.EOF
	}
	copy(dest, r.r.rows[r.i])
	r.i++
	return nil
}
func (serviceResult) LastInsertId() (int64, error)   { return 0, nil }
func (r serviceResult) RowsAffected() (int64, error) { return int64(r), nil }

func boolResponse(v bool) *serviceResponse {
	return &serviceResponse{columns: []string{"exists"}, rows: [][]driver.Value{{v}}}
}
func execService() *serviceResponse { return &serviceResponse{affected: 1} }
func serviceUser(id, name string) *serviceResponse {
	now := time.Now().UTC()
	return &serviceResponse{
		columns: []string{"id", "username", "is_active", "created_at", "updated_at", "username_changed_at", "username_pending", "preferred_locale"},
		rows:    [][]driver.Value{{id, name, true, now, now, nil, false, nil}},
	}
}
func serviceDetails(id, name string) *serviceResponse {
	r := serviceUser(id, name)
	r.columns = append(r.columns, "follower_count", "following_count")
	r.rows[0] = append(r.rows[0], int64(0), int64(0))
	return r
}
func serviceIDs(ids ...string) *serviceResponse {
	rows := make([][]driver.Value, len(ids))
	for i, id := range ids {
		rows[i] = []driver.Value{id}
	}
	return &serviceResponse{columns: []string{"id"}, rows: rows}
}

func TestService_ScriptedProvisioning(t *testing.T) {
	t.Run("existing", func(t *testing.T) {
		s := scriptedService(t, boolResponse(true), serviceDetails(idA, "alice"))
		if p, err := s.ProvisionFromClaims(idA, "alice@test"); err != nil || p.Username != "alice" {
			t.Fatalf("profil=%+v err=%v", p, err)
		}
	})
	t.Run("create", func(t *testing.T) {
		s := scriptedService(t, boolResponse(false), serviceUser(idA, "alice"), serviceDetails(idA, "alice"))
		if _, err := s.ProvisionFromClaims(idA, "alice@test"); err != nil {
			t.Fatal(err)
		}
	})
}

func TestService_ScriptedFollowPaths(t *testing.T) {
	t.Run("public", func(t *testing.T) {
		n := &fakeNotifier{}
		s := scriptedService(t, boolResponse(true), serviceDetails(idA, "alice"), boolResponse(true), boolResponse(false), execService())
		s.profilClient = &fakeProfil{visibility: "public"}
		s.notification = n
		status, err := s.Follow(context.Background(), idA, "a@test", idB)
		if err != nil || status != FollowStatusFollowing || len(n.types()) != 1 {
			t.Fatalf("status=%q err=%v events=%v", status, err, n.types())
		}
	})
	t.Run("private", func(t *testing.T) {
		s := scriptedService(t, boolResponse(true), serviceDetails(idA, "alice"), boolResponse(true), boolResponse(false), execService())
		s.profilClient = &fakeProfil{visibility: client.VisibilityPrivate}
		status, err := s.Follow(context.Background(), idA, "a@test", idB)
		if err != nil || status != FollowStatusPending {
			t.Fatalf("status=%q err=%v", status, err)
		}
	})
	t.Run("already following", func(t *testing.T) {
		s := scriptedService(t, boolResponse(true), serviceDetails(idA, "alice"), boolResponse(true), boolResponse(true))
		status, err := s.Follow(context.Background(), idA, "a@test", idB)
		if err != nil || status != FollowStatusFollowing {
			t.Fatalf("status=%q err=%v", status, err)
		}
	})
}

func TestService_ScriptedSocialGraph(t *testing.T) {
	s := scriptedService(t,
		boolResponse(true), boolResponse(true), execService(), execService(), execService(),
		boolResponse(true), serviceUser(idB, "bob"),
		boolResponse(true), serviceUser(idB, "bob"),
		boolResponse(true), boolResponse(true), boolResponse(true),
	)
	if err := s.Block(idA, idB); err != nil {
		t.Fatal(err)
	}
	if users, err := s.ListFollowers(idA, 10, 0); err != nil || len(users) != 1 {
		t.Fatalf("followers=%v err=%v", users, err)
	}
	if users, err := s.ListFollowing(idA, 10, 0); err != nil || len(users) != 1 {
		t.Fatalf("following=%v err=%v", users, err)
	}
	if !s.IsFollowing(idA, idB) {
		t.Fatal("IsFollowing devrait être true")
	}
}

func TestService_ScriptedFollowRequests(t *testing.T) {
	n := &fakeNotifier{}
	s := scriptedService(t, execService(), execService(), serviceIDs(idB), execService(), execService())
	s.notification = n
	if err := s.AcceptFollowRequest(idA, idB); err != nil {
		t.Fatal(err)
	}
	if err := s.AcceptAllFollowRequests(idA); err != nil {
		t.Fatal(err)
	}
	if len(n.types()) < 3 {
		t.Fatalf("événements=%v", n.types())
	}
}
