package service

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/lib/pq"

	"github.com/webdad/user-service/internal/repository"
)

// closedSvc construit un service dont le repo est adossé à un pool FERMÉ : toute
// requête SQL échoue, ce qui exerce les branches de mapping d'erreur DB du
// service (sans dépendre d'un serveur PostgreSQL).
func closedSvc(t *testing.T) *UserService {
	t.Helper()
	conn, err := sql.Open("postgres", "host=127.0.0.1 port=1 dbname=x sslmode=disable")
	if err != nil {
		t.Fatalf("sql.Open : %v", err)
	}
	_ = conn.Close()
	return New(repository.New(conn), 0, WithProfilClient(&fakeProfil{visibility: "public"}), WithNotificationClient(&fakeNotifier{}))
}

func TestService_ErrorPaths_ClosedDB(t *testing.T) {
	s := closedSvc(t)
	ctx := context.Background()
	const id1 = "11111111-0000-0000-0000-000000000001"
	const id2 = "22222222-0000-0000-0000-000000000002"

	if _, err := s.Create(id1, "validname"); err == nil {
		t.Error("Create doit propager l'erreur DB")
	}
	if _, err := s.AdminCreate(id1, "validname"); err == nil {
		t.Error("AdminCreate doit propager l'erreur DB")
	}
	if _, err := s.GetDetailsByID(id1); err == nil {
		t.Error("GetDetailsByID doit propager l'erreur DB")
	}
	if _, err := s.GetDetailsByUsername("validname"); err == nil {
		t.Error("GetDetailsByUsername doit propager l'erreur DB")
	}
	if _, err := s.List(10, 0); err == nil {
		t.Error("List doit propager l'erreur DB")
	}
	if _, err := s.Search("term", 10, 0); err == nil {
		t.Error("Search doit propager l'erreur DB")
	}
	if _, err := s.Suggestions(10, 0); err == nil {
		t.Error("Suggestions doit propager l'erreur DB")
	}
	valid := "validname"
	if _, err := s.Update(id1, &valid, nil); err == nil {
		t.Error("Update doit propager l'erreur DB")
	}
	if err := s.SetActive(id1, false); err == nil {
		t.Error("SetActive doit propager l'erreur DB")
	}
	if err := s.PurgeUser(id1); err == nil {
		t.Error("PurgeUser doit propager l'erreur DB")
	}
	if _, err := s.ProvisionFromClaims(id1, "a@b.com"); err == nil {
		t.Error("ProvisionFromClaims doit propager l'erreur DB")
	}
	if _, err := s.Follow(ctx, id1, "a@b.com", id2); err == nil {
		t.Error("Follow doit propager l'erreur DB")
	}
	if err := s.Unfollow(id1, id2); err == nil {
		t.Error("Unfollow doit propager l'erreur DB")
	}
	if err := s.AcceptFollowRequest(id1, id2); err == nil {
		t.Error("AcceptFollowRequest doit propager l'erreur DB")
	}
	if err := s.AcceptAllFollowRequests(id1); err == nil {
		t.Error("AcceptAllFollowRequests doit propager l'erreur DB")
	}
	if err := s.RejectFollowRequest(id1, id2); err == nil {
		t.Error("RejectFollowRequest doit propager l'erreur DB")
	}
	if _, err := s.PendingFollowRequestIDs(id1); err == nil {
		t.Error("PendingFollowRequestIDs doit propager l'erreur DB")
	}
	if err := s.RemoveFollower(id1, id2); err == nil {
		t.Error("RemoveFollower doit propager l'erreur DB")
	}
	if _, err := s.ListFollowers(id1, 10, 0); err == nil {
		t.Error("ListFollowers doit propager l'erreur DB")
	}
	if _, err := s.ListFollowing(id1, 10, 0); err == nil {
		t.Error("ListFollowing doit propager l'erreur DB")
	}
	if s.IsFollowing(id1, id2) {
		t.Error("IsFollowing doit être false sur erreur DB")
	}
	if err := s.Block(id1, id2); err == nil {
		t.Error("Block doit propager l'erreur DB")
	}
	if err := s.Unblock(id1, id2); err == nil {
		t.Error("Unblock doit propager l'erreur DB")
	}
	if _, err := s.BlockedIDs(id1); err == nil {
		t.Error("BlockedIDs doit propager l'erreur DB")
	}
	if s.HasBlocked(id1, id2) {
		t.Error("HasBlocked doit être false sur erreur DB")
	}
}

// TestService_Update_CooldownReadError : avec cooldown>0, Update lit le user
// courant ; sur pool fermé, la lecture échoue (≠ ErrNoRows) → erreur propagée.
func TestService_Update_CooldownReadError(t *testing.T) {
	conn, _ := sql.Open("postgres", "host=127.0.0.1 port=1 dbname=x sslmode=disable")
	_ = conn.Close()
	s := New(repository.New(conn), 60*1e9) // 60s de cooldown
	valid := "validname"
	if _, err := s.Update("11111111-0000-0000-0000-000000000001", &valid, nil); err == nil {
		t.Error("Update(cooldown) doit propager l'erreur de lecture DB")
	}
}
