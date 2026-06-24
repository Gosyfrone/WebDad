package repository

import (
	"database/sql"
	"testing"

	_ "github.com/lib/pq"
)

// closedRepo renvoie un repository adossé à un pool FERMÉ : toute opération SQL
// échoue immédiatement (« sql: database is closed »). Cela exerce les branches
// d'erreur de chaque méthode sans dépendre d'un serveur PostgreSQL.
func closedRepo(t *testing.T) *UserRepository {
	t.Helper()
	conn, err := sql.Open("postgres", "host=127.0.0.1 port=1 dbname=x sslmode=disable")
	if err != nil {
		t.Fatalf("sql.Open : %v", err)
	}
	_ = conn.Close()
	return New(conn)
}

func TestRepo_ErrorPaths_ClosedDB(t *testing.T) {
	r := closedRepo(t)

	if _, err := r.Create(uA, "x"); err == nil {
		t.Error("Create doit échouer sur pool fermé")
	}
	if _, err := r.CreateWithPending(uA, "x", true); err == nil {
		t.Error("CreateWithPending doit échouer")
	}
	if _, err := r.ExistsByID(uA); err == nil {
		t.Error("ExistsByID doit échouer")
	}
	if _, err := r.GetByID(uA); err == nil {
		t.Error("GetByID doit échouer")
	}
	if _, err := r.GetDetailsByID(uA); err == nil {
		t.Error("GetDetailsByID doit échouer")
	}
	if _, err := r.GetDetailsByUsername("x"); err == nil {
		t.Error("GetDetailsByUsername doit échouer")
	}
	if _, err := r.List(10, 0); err == nil {
		t.Error("List doit échouer")
	}
	if _, err := r.Update(uA, nil, nil); err == nil {
		t.Error("Update doit échouer")
	}
	if err := r.SetActive(uA, false); err == nil {
		t.Error("SetActive doit échouer")
	}
	if err := r.SoftDelete(uA); err == nil {
		t.Error("SoftDelete doit échouer")
	}
	if err := r.PurgeUser(uA); err == nil {
		t.Error("PurgeUser doit échouer")
	}
	if err := r.Follow(uA, uB); err == nil {
		t.Error("Follow doit échouer")
	}
	if err := r.RequestFollow(uA, uB); err == nil {
		t.Error("RequestFollow doit échouer")
	}
	if err := r.DeleteFollowRequest(uA, uB); err == nil {
		t.Error("DeleteFollowRequest doit échouer")
	}
	if _, err := r.HasFollowRequest(uA, uB); err == nil {
		t.Error("HasFollowRequest doit échouer")
	}
	if _, err := r.PendingFollowRequestIDs(uA); err == nil {
		t.Error("PendingFollowRequestIDs doit échouer")
	}
	if _, err := r.IncomingFollowRequestFollowerIDs(uA); err == nil {
		t.Error("IncomingFollowRequestFollowerIDs doit échouer")
	}
	if _, err := r.AcceptFollowRequest(uA, uB); err == nil {
		t.Error("AcceptFollowRequest doit échouer")
	}
	if err := r.Unfollow(uA, uB); err == nil {
		t.Error("Unfollow doit échouer")
	}
	if _, err := r.ListFollowers(uA, 10, 0); err == nil {
		t.Error("ListFollowers doit échouer")
	}
	if _, err := r.ListFollowing(uA, 10, 0); err == nil {
		t.Error("ListFollowing doit échouer")
	}
	if _, err := r.Search("x", 10, 0); err == nil {
		t.Error("Search doit échouer")
	}
	if _, err := r.ListByFollowers(10, 0); err == nil {
		t.Error("ListByFollowers doit échouer")
	}
	if _, err := r.IsFollowing(uA, uB); err == nil {
		t.Error("IsFollowing doit échouer")
	}
	if err := r.Block(uA, uB); err == nil {
		t.Error("Block doit échouer")
	}
	if err := r.Unblock(uA, uB); err == nil {
		t.Error("Unblock doit échouer")
	}
	if _, err := r.BlockedIDs(uA); err == nil {
		t.Error("BlockedIDs doit échouer")
	}
	if _, err := r.HasBlocked(uA, uB); err == nil {
		t.Error("HasBlocked doit échouer")
	}
}
