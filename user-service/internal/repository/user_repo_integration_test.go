package repository

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/webdad/user-service/internal/testutil"
)

// UUID figés pour les tests (forme canonique exigée par la colonne `uuid`).
const (
	uA = "aaaaaaaa-0000-0000-0000-000000000001"
	uB = "bbbbbbbb-0000-0000-0000-000000000002"
	uC = "cccccccc-0000-0000-0000-000000000003"
)

// newRepo construit un repository adossé à la base de test (skip si absente).
func newRepo(t *testing.T) *UserRepository {
	t.Helper()
	return New(testutil.DB(t))
}

// seed insère un utilisateur et échoue le test en cas d'erreur.
func seed(t *testing.T, r *UserRepository, id, username string) {
	t.Helper()
	if _, err := r.Create(id, username); err != nil {
		t.Fatalf("seed Create(%s) : %v", username, err)
	}
}

func TestRepo_Create_GetByID_ExistsByID(t *testing.T) {
	r := newRepo(t)

	u, err := r.Create(uA, "alice")
	if err != nil {
		t.Fatalf("Create : %v", err)
	}
	if u.ID != uA || u.Username != "alice" || !u.IsActive || u.UsernamePending {
		t.Fatalf("Create renvoie un user inattendu : %+v", u)
	}

	got, err := r.GetByID(uA)
	if err != nil {
		t.Fatalf("GetByID : %v", err)
	}
	if got.Username != "alice" {
		t.Fatalf("GetByID username = %q", got.Username)
	}

	exists, err := r.ExistsByID(uA)
	if err != nil || !exists {
		t.Fatalf("ExistsByID = %v, %v ; attendu true, nil", exists, err)
	}
	absent, err := r.ExistsByID(uB)
	if err != nil || absent {
		t.Fatalf("ExistsByID(absent) = %v, %v ; attendu false, nil", absent, err)
	}
}

func TestRepo_Create_DuplicateUsername_UniqueViolation(t *testing.T) {
	r := newRepo(t)
	seed(t, r, uA, "alice")
	_, err := r.Create(uB, "alice")
	if err == nil {
		t.Fatal("Create avec username dupliqué doit échouer (contrainte UNIQUE)")
	}
}

func TestRepo_GetByID_NoRows(t *testing.T) {
	r := newRepo(t)
	_, err := r.GetByID(uA)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("GetByID(absent) = %v, attendu sql.ErrNoRows", err)
	}
}

func TestRepo_CreateWithPending(t *testing.T) {
	r := newRepo(t)
	u, err := r.CreateWithPending(uA, "alice_1a2b3c4d", true)
	if err != nil {
		t.Fatalf("CreateWithPending : %v", err)
	}
	if !u.UsernamePending {
		t.Fatalf("CreateWithPending pending = false, attendu true")
	}
}

func TestRepo_GetDetailsByID_AndUsername_WithCounts(t *testing.T) {
	r := newRepo(t)
	seed(t, r, uA, "alice")
	seed(t, r, uB, "bob")
	seed(t, r, uC, "carol")
	// bob et carol suivent alice ; alice suit bob.
	mustFollow(t, r, uB, uA)
	mustFollow(t, r, uC, uA)
	mustFollow(t, r, uA, uB)

	d, err := r.GetDetailsByID(uA)
	if err != nil {
		t.Fatalf("GetDetailsByID : %v", err)
	}
	if d.FollowerCount != 2 || d.FollowingCount != 1 {
		t.Fatalf("compteurs alice = %d/%d, attendu 2/1", d.FollowerCount, d.FollowingCount)
	}

	d2, err := r.GetDetailsByUsername("alice")
	if err != nil {
		t.Fatalf("GetDetailsByUsername : %v", err)
	}
	if d2.ID != uA {
		t.Fatalf("GetDetailsByUsername id = %q", d2.ID)
	}

	if _, err := r.GetDetailsByID(uB); err != nil {
		t.Fatalf("GetDetailsByID(bob) : %v", err)
	}
	if _, err := r.GetDetailsByUsername("inconnu"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("GetDetailsByUsername(inconnu) = %v, attendu ErrNoRows", err)
	}
}

func TestRepo_List_Search_ListByFollowers(t *testing.T) {
	r := newRepo(t)
	seed(t, r, uA, "alice")
	seed(t, r, uB, "bob")
	seed(t, r, uC, "carol")
	mustFollow(t, r, uB, uA)
	mustFollow(t, r, uC, uA) // alice = la plus suivie

	list, err := r.List(10, 0)
	if err != nil || len(list) != 3 {
		t.Fatalf("List = %d users, %v ; attendu 3", len(list), err)
	}

	// SetActive false → masqué de List.
	if err := r.SetActive(uC, false); err != nil {
		t.Fatalf("SetActive : %v", err)
	}
	list, _ = r.List(10, 0)
	if len(list) != 2 {
		t.Fatalf("List après désactivation = %d, attendu 2", len(list))
	}

	found, err := r.Search("ali", 10, 0)
	if err != nil || len(found) != 1 || found[0].Username != "alice" {
		t.Fatalf("Search(ali) = %+v, %v", found, err)
	}
	// Échappement LIKE : un terme avec % ne doit pas tout matcher.
	none, err := r.Search("%", 10, 0)
	if err != nil || len(none) != 0 {
		t.Fatalf("Search(%%) = %d, %v ; attendu 0 (littéral)", len(none), err)
	}

	sugg, err := r.ListByFollowers(10, 0)
	if err != nil || len(sugg) == 0 || sugg[0].Username != "alice" {
		t.Fatalf("ListByFollowers premier = %+v, %v ; attendu alice en tête", sugg, err)
	}
}

func TestRepo_Update(t *testing.T) {
	r := newRepo(t)
	seed(t, r, uA, "alice")

	newName := "alice2"
	locale := "fr"
	u, err := r.Update(uA, &newName, &locale)
	if err != nil {
		t.Fatalf("Update : %v", err)
	}
	if u.Username != "alice2" || u.PreferredLocale == nil || *u.PreferredLocale != "fr" {
		t.Fatalf("Update renvoie %+v", u)
	}
	if u.UsernameChangedAt == nil {
		t.Fatal("UsernameChangedAt doit être posé après un changement effectif")
	}

	// PATCH sans-op (mêmes valeurs nil) ne touche rien.
	u2, err := r.Update(uA, nil, nil)
	if err != nil {
		t.Fatalf("Update no-op : %v", err)
	}
	if u2.Username != "alice2" {
		t.Fatalf("Update no-op a modifié le username : %q", u2.Username)
	}

	if _, err := r.Update(uB, &newName, nil); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("Update(absent) = %v, attendu ErrNoRows", err)
	}
}

func TestRepo_SetActive_SoftDelete_NoRows(t *testing.T) {
	r := newRepo(t)
	if err := r.SetActive(uA, false); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("SetActive(absent) = %v, attendu ErrNoRows", err)
	}
	if err := r.SoftDelete(uA); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("SoftDelete(absent) = %v, attendu ErrNoRows", err)
	}
	seed(t, r, uA, "alice")
	if err := r.SoftDelete(uA); err != nil {
		t.Fatalf("SoftDelete : %v", err)
	}
	got, _ := r.GetByID(uA)
	if got.IsActive {
		t.Fatal("SoftDelete doit désactiver le compte")
	}
}

func TestRepo_FollowGraph(t *testing.T) {
	r := newRepo(t)
	seed(t, r, uA, "alice")
	seed(t, r, uB, "bob")

	mustFollow(t, r, uA, uB)
	// Idempotent.
	mustFollow(t, r, uA, uB)

	yes, err := r.IsFollowing(uA, uB)
	if err != nil || !yes {
		t.Fatalf("IsFollowing = %v, %v ; attendu true", yes, err)
	}
	no, _ := r.IsFollowing(uB, uA)
	if no {
		t.Fatal("IsFollowing(bob, alice) doit être false")
	}

	followers, err := r.ListFollowers(uB, 10, 0)
	if err != nil || len(followers) != 1 || followers[0].ID != uA {
		t.Fatalf("ListFollowers(bob) = %+v, %v", followers, err)
	}
	following, err := r.ListFollowing(uA, 10, 0)
	if err != nil || len(following) != 1 || following[0].ID != uB {
		t.Fatalf("ListFollowing(alice) = %+v, %v", following, err)
	}

	if err := r.Unfollow(uA, uB); err != nil {
		t.Fatalf("Unfollow : %v", err)
	}
	yes, _ = r.IsFollowing(uA, uB)
	if yes {
		t.Fatal("Unfollow doit supprimer la relation")
	}
}

func TestRepo_FollowRequests(t *testing.T) {
	r := newRepo(t)
	seed(t, r, uA, "alice")
	seed(t, r, uB, "bob")

	if err := r.RequestFollow(uA, uB); err != nil {
		t.Fatalf("RequestFollow : %v", err)
	}
	// Idempotent.
	if err := r.RequestFollow(uA, uB); err != nil {
		t.Fatalf("RequestFollow idempotent : %v", err)
	}

	has, err := r.HasFollowRequest(uA, uB)
	if err != nil || !has {
		t.Fatalf("HasFollowRequest = %v, %v ; attendu true", has, err)
	}

	out, err := r.PendingFollowRequestIDs(uA)
	if err != nil || len(out) != 1 || out[0] != uB {
		t.Fatalf("PendingFollowRequestIDs = %+v, %v", out, err)
	}
	in, err := r.IncomingFollowRequestFollowerIDs(uB)
	if err != nil || len(in) != 1 || in[0] != uA {
		t.Fatalf("IncomingFollowRequestFollowerIDs = %+v, %v", in, err)
	}

	// AcceptFollowRequest : convertit la demande en follow.
	accepted, err := r.AcceptFollowRequest(uA, uB)
	if err != nil || !accepted {
		t.Fatalf("AcceptFollowRequest = %v, %v ; attendu true", accepted, err)
	}
	yes, _ := r.IsFollowing(uA, uB)
	if !yes {
		t.Fatal("après acceptation, la relation follow doit exister")
	}
	// Accepter une demande inexistante → false.
	again, err := r.AcceptFollowRequest(uA, uB)
	if err != nil || again {
		t.Fatalf("AcceptFollowRequest(absent) = %v, %v ; attendu false", again, err)
	}

	// DeleteFollowRequest (rejet).
	if err := r.RequestFollow(uB, uA); err != nil {
		t.Fatalf("RequestFollow(b→a) : %v", err)
	}
	if err := r.DeleteFollowRequest(uB, uA); err != nil {
		t.Fatalf("DeleteFollowRequest : %v", err)
	}
	has, _ = r.HasFollowRequest(uB, uA)
	if has {
		t.Fatal("DeleteFollowRequest doit retirer la demande")
	}
}

func TestRepo_Blocks(t *testing.T) {
	r := newRepo(t)
	seed(t, r, uA, "alice")
	seed(t, r, uB, "bob")
	// Relations sociales préexistantes : le blocage doit les couper.
	mustFollow(t, r, uA, uB)
	mustFollow(t, r, uB, uA)
	if err := r.RequestFollow(uA, uB); err != nil {
		// follows existant déjà, mais on force une demande dans l'autre sens
		t.Logf("RequestFollow : %v", err)
	}

	if err := r.Block(uA, uB); err != nil {
		t.Fatalf("Block : %v", err)
	}
	// Idempotent.
	if err := r.Block(uA, uB); err != nil {
		t.Fatalf("Block idempotent : %v", err)
	}

	has, err := r.HasBlocked(uA, uB)
	if err != nil || !has {
		t.Fatalf("HasBlocked = %v, %v ; attendu true", has, err)
	}
	// Le blocage a coupé les follows dans les deux sens.
	if yes, _ := r.IsFollowing(uA, uB); yes {
		t.Fatal("Block doit supprimer le follow a→b")
	}
	if yes, _ := r.IsFollowing(uB, uA); yes {
		t.Fatal("Block doit supprimer le follow b→a")
	}

	ids, err := r.BlockedIDs(uA)
	if err != nil || len(ids) != 1 || ids[0] != uB {
		t.Fatalf("BlockedIDs = %+v, %v", ids, err)
	}

	if err := r.Unblock(uA, uB); err != nil {
		t.Fatalf("Unblock : %v", err)
	}
	has, _ = r.HasBlocked(uA, uB)
	if has {
		t.Fatal("Unblock doit retirer le blocage")
	}
}

func TestRepo_PurgeUser(t *testing.T) {
	r := newRepo(t)
	seed(t, r, uA, "alice")
	seed(t, r, uB, "bob")
	mustFollow(t, r, uA, uB)
	if err := r.RequestFollow(uB, uA); err != nil {
		t.Fatalf("RequestFollow : %v", err)
	}

	if err := r.PurgeUser(uA); err != nil {
		t.Fatalf("PurgeUser : %v", err)
	}
	if exists, _ := r.ExistsByID(uA); exists {
		t.Fatal("PurgeUser doit supprimer la ligne users")
	}
	// Idempotent : purger un compte déjà absent ne renvoie pas d'erreur.
	if err := r.PurgeUser(uA); err != nil {
		t.Fatalf("PurgeUser idempotent : %v", err)
	}
}

// mustFollow crée une relation de follow et échoue le test en cas d'erreur.
func mustFollow(t *testing.T, r *UserRepository, followerID, followingID string) {
	t.Helper()
	if err := r.Follow(followerID, followingID); err != nil {
		t.Fatalf("Follow(%s→%s) : %v", followerID, followingID, err)
	}
}
