package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/webdad/user-service/internal/client"
	"github.com/webdad/user-service/internal/repository"
	"github.com/webdad/user-service/internal/testutil"
)

const (
	idA = "aaaaaaaa-1111-0000-0000-000000000001"
	idB = "bbbbbbbb-1111-0000-0000-000000000002"
	idC = "cccccccc-1111-0000-0000-000000000003"
)

// fakeProfil simule profil-service : visibilité configurable + erreur optionnelle.
type fakeProfil struct {
	visibility string
	err        error
}

func (f *fakeProfil) Visibility(_ context.Context, _ string) (string, error) {
	return f.visibility, f.err
}

// fakeNotifier capture les événements émis (Emit est best-effort, synchrone ici).
type fakeNotifier struct {
	mu     sync.Mutex
	events []client.Event
}

func (f *fakeNotifier) Emit(ev client.Event) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, ev)
}

func (f *fakeNotifier) types() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.events))
	for i, e := range f.events {
		out[i] = e.Type
	}
	return out
}

// newSvcDB construit un service adossé à la base de test (skip si absente),
// avec un repo réel et des fakes profil/notification.
func newSvcDB(t *testing.T, cooldown time.Duration, opts ...Option) (*UserService, *repository.UserRepository) {
	t.Helper()
	repo := repository.New(testutil.DB(t))
	return New(repo, cooldown, opts...), repo
}

func TestService_Create_AndConflict(t *testing.T) {
	svc, _ := newSvcDB(t, 0)

	u, err := svc.Create(idA, "alice")
	if err != nil {
		t.Fatalf("Create : %v", err)
	}
	if u.Username != "alice" {
		t.Fatalf("Create username = %q", u.Username)
	}

	// Username déjà pris → ErrUsernameTaken.
	if _, err := svc.Create(idB, "alice"); !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("Create(dup) = %v, attendu ErrUsernameTaken", err)
	}
}

func TestService_AdminCreate_Nominal_Suffix_Idempotent(t *testing.T) {
	svc, _ := newSvcDB(t, 0)

	// 1) Handle libre.
	u, err := svc.AdminCreate(idA, "alice")
	if err != nil {
		t.Fatalf("AdminCreate : %v", err)
	}
	if u.Username != "alice" || u.UsernamePending {
		t.Fatalf("AdminCreate nominal = %+v", u)
	}

	// 2) Même id → idempotent (renvoie la ligne existante).
	again, err := svc.AdminCreate(idA, "whatever")
	if err != nil || again.ID != idA {
		t.Fatalf("AdminCreate idempotent = %+v, %v", again, err)
	}

	// 3) Handle pris par un autre id → suffixe + pending.
	sfx, err := svc.AdminCreate(idB, "alice")
	if err != nil {
		t.Fatalf("AdminCreate(suffix) : %v", err)
	}
	if !sfx.UsernamePending || sfx.Username == "alice" {
		t.Fatalf("AdminCreate(suffix) = %+v ; attendu pending + handle suffixé", sfx)
	}
}

func TestService_GetDetails_List_Search_Suggestions(t *testing.T) {
	svc, _ := newSvcDB(t, 0)
	mustCreate(t, svc, idA, "alice")
	mustCreate(t, svc, idB, "bob")

	d, err := svc.GetDetailsByID(idA)
	if err != nil || d.Username != "alice" {
		t.Fatalf("GetDetailsByID = %+v, %v", d, err)
	}
	if _, err := svc.GetDetailsByID(idC); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("GetDetailsByID(absent) = %v, attendu ErrUserNotFound", err)
	}
	d2, err := svc.GetDetailsByUsername("bob")
	if err != nil || d2.ID != idB {
		t.Fatalf("GetDetailsByUsername = %+v, %v", d2, err)
	}
	if _, err := svc.GetDetailsByUsername("zzz"); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("GetDetailsByUsername(absent) = %v, attendu ErrUserNotFound", err)
	}

	list, err := svc.List(10, 0)
	if err != nil || len(list) != 2 {
		t.Fatalf("List = %d, %v", len(list), err)
	}
	found, err := svc.Search("ali", 10, 0)
	if err != nil || len(found) != 1 {
		t.Fatalf("Search = %d, %v", len(found), err)
	}
	sugg, err := svc.Suggestions(10, 0)
	if err != nil || len(sugg) != 2 {
		t.Fatalf("Suggestions = %d, %v", len(sugg), err)
	}
}

func TestService_Update_Cooldown(t *testing.T) {
	svc, _ := newSvcDB(t, time.Hour)
	mustCreate(t, svc, idA, "alice")

	// Premier changement : pas de baseline → autorisé.
	n1 := "alice1"
	if _, err := svc.Update(idA, &n1, nil); err != nil {
		t.Fatalf("Update#1 : %v", err)
	}
	// Deuxième changement immédiat : cooldown actif → refus.
	n2 := "alice2"
	if _, err := svc.Update(idA, &n2, nil); !errors.Is(err, ErrUsernameCooldown) {
		t.Fatalf("Update#2 = %v, attendu ErrUsernameCooldown", err)
	}

	// Utilisateur inexistant → ErrUserNotFound.
	if _, err := svc.Update(idC, &n2, nil); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("Update(absent) = %v, attendu ErrUserNotFound", err)
	}
}

func TestService_Update_UsernameTaken(t *testing.T) {
	svc, _ := newSvcDB(t, 0)
	mustCreate(t, svc, idA, "alice")
	mustCreate(t, svc, idB, "bob")

	taken := "bob"
	if _, err := svc.Update(idA, &taken, nil); !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("Update(taken) = %v, attendu ErrUsernameTaken", err)
	}
}

func TestService_SetActive_SoftDelete_Purge(t *testing.T) {
	svc, _ := newSvcDB(t, 0)
	mustCreate(t, svc, idA, "alice")

	if err := svc.SoftDelete(idA); err != nil {
		t.Fatalf("SoftDelete : %v", err)
	}
	if err := svc.SetActive(idA, true); err != nil {
		t.Fatalf("SetActive(true) : %v", err)
	}
	if err := svc.SetActive(idC, true); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("SetActive(absent) = %v, attendu ErrUserNotFound", err)
	}
	if err := svc.PurgeUser(idA); err != nil {
		t.Fatalf("PurgeUser : %v", err)
	}
	if _, err := svc.GetDetailsByID(idA); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("après purge, GetDetailsByID = %v, attendu ErrUserNotFound", err)
	}
}

func TestService_ProvisionFromClaims(t *testing.T) {
	svc, _ := newSvcDB(t, 0)

	// Premier appel : crée la ligne à partir de l'email.
	d, err := svc.ProvisionFromClaims(idA, "alice@breezy.dev")
	if err != nil {
		t.Fatalf("ProvisionFromClaims#1 : %v", err)
	}
	if d.Username != "alice" {
		t.Fatalf("provision username = %q, attendu alice", d.Username)
	}
	// Deuxième appel : la ligne existe déjà → simple lecture.
	d2, err := svc.ProvisionFromClaims(idA, "alice@breezy.dev")
	if err != nil || d2.ID != idA {
		t.Fatalf("ProvisionFromClaims#2 = %+v, %v", d2, err)
	}
}

func TestService_ProvisionFromClaims_UsernameCollision(t *testing.T) {
	svc, _ := newSvcDB(t, 0)
	// Occupe le handle "bob" pour forcer le provisioning à suffixer.
	mustCreate(t, svc, idC, "bob")

	d, err := svc.ProvisionFromClaims(idA, "bob@example.com")
	if err != nil {
		t.Fatalf("ProvisionFromClaims(collision) : %v", err)
	}
	if d.Username == "bob" {
		t.Fatal("provision aurait dû éviter le handle déjà pris")
	}
}

func TestService_Follow_Public(t *testing.T) {
	notif := &fakeNotifier{}
	svc, _ := newSvcDB(t, 0, WithProfilClient(&fakeProfil{visibility: "public"}), WithNotificationClient(notif))
	mustCreate(t, svc, idA, "alice")
	mustCreate(t, svc, idB, "bob")

	status, err := svc.Follow(context.Background(), idA, "alice@x.com", idB)
	if err != nil {
		t.Fatalf("Follow : %v", err)
	}
	if status != FollowStatusFollowing {
		t.Fatalf("Follow status = %q, attendu following", status)
	}
	// Idempotent : déjà abonné → following sans réémission obligatoire.
	if status, err = svc.Follow(context.Background(), idA, "alice@x.com", idB); err != nil || status != FollowStatusFollowing {
		t.Fatalf("Follow(idempotent) = %q, %v", status, err)
	}
	if !contains(notif.types(), client.TypeFollow) {
		t.Fatalf("événement follow attendu, obtenu %v", notif.types())
	}
}

func TestService_Follow_Private_RequestThenAccept(t *testing.T) {
	notif := &fakeNotifier{}
	svc, _ := newSvcDB(t, 0, WithProfilClient(&fakeProfil{visibility: client.VisibilityPrivate}), WithNotificationClient(notif))
	mustCreate(t, svc, idA, "alice")
	mustCreate(t, svc, idB, "bob") // bob privé

	status, err := svc.Follow(context.Background(), idA, "alice@x.com", idB)
	if err != nil || status != FollowStatusPending {
		t.Fatalf("Follow(privé) = %q, %v ; attendu pending", status, err)
	}

	pending, err := svc.PendingFollowRequestIDs(idA)
	if err != nil || len(pending) != 1 || pending[0] != idB {
		t.Fatalf("PendingFollowRequestIDs = %+v, %v", pending, err)
	}

	// bob accepte la demande de alice.
	if err := svc.AcceptFollowRequest(idB, idA); err != nil {
		t.Fatalf("AcceptFollowRequest : %v", err)
	}
	if !svc.IsFollowing(idA, idB) {
		t.Fatal("après acceptation, alice suit bob")
	}
	// Demande inexistante.
	if err := svc.AcceptFollowRequest(idB, idA); !errors.Is(err, ErrFollowRequestNotFound) {
		t.Fatalf("AcceptFollowRequest(absent) = %v, attendu ErrFollowRequestNotFound", err)
	}
}

func TestService_Follow_TargetNotFound_AndSelf(t *testing.T) {
	svc, _ := newSvcDB(t, 0, WithProfilClient(&fakeProfil{visibility: "public"}))
	mustCreate(t, svc, idA, "alice")

	if _, err := svc.Follow(context.Background(), idA, "alice@x.com", idA); !errors.Is(err, ErrSelfFollow) {
		t.Fatalf("Follow(self) = %v, attendu ErrSelfFollow", err)
	}
	if _, err := svc.Follow(context.Background(), idA, "alice@x.com", idC); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("Follow(cible absente) = %v, attendu ErrUserNotFound", err)
	}
}

func TestService_RejectFollowRequest(t *testing.T) {
	svc, _ := newSvcDB(t, 0, WithProfilClient(&fakeProfil{visibility: client.VisibilityPrivate}))
	mustCreate(t, svc, idA, "alice")
	mustCreate(t, svc, idB, "bob")

	if _, err := svc.Follow(context.Background(), idA, "alice@x.com", idB); err != nil {
		t.Fatalf("Follow(privé) : %v", err)
	}
	if err := svc.RejectFollowRequest(idB, idA); err != nil {
		t.Fatalf("RejectFollowRequest : %v", err)
	}
	if err := svc.RejectFollowRequest(idB, idA); !errors.Is(err, ErrFollowRequestNotFound) {
		t.Fatalf("RejectFollowRequest(absent) = %v, attendu ErrFollowRequestNotFound", err)
	}
}

func TestService_AcceptAllFollowRequests(t *testing.T) {
	svc, _ := newSvcDB(t, 0, WithProfilClient(&fakeProfil{visibility: client.VisibilityPrivate}))
	mustCreate(t, svc, idA, "alice") // owner privé
	mustCreate(t, svc, idB, "bob")
	mustCreate(t, svc, idC, "carol")

	// bob et carol demandent à suivre alice (privée).
	if _, err := svc.Follow(context.Background(), idB, "bob@x.com", idA); err != nil {
		t.Fatalf("Follow(bob→alice) : %v", err)
	}
	if _, err := svc.Follow(context.Background(), idC, "carol@x.com", idA); err != nil {
		t.Fatalf("Follow(carol→alice) : %v", err)
	}

	if err := svc.AcceptAllFollowRequests(idA); err != nil {
		t.Fatalf("AcceptAllFollowRequests : %v", err)
	}
	if !svc.IsFollowing(idB, idA) || !svc.IsFollowing(idC, idA) {
		t.Fatal("AcceptAllFollowRequests doit convertir toutes les demandes")
	}
	// Aucune demande restante : no-op sans erreur.
	if err := svc.AcceptAllFollowRequests(idA); err != nil {
		t.Fatalf("AcceptAllFollowRequests(vide) : %v", err)
	}
}

func TestService_Unfollow_RemoveFollower_Lists(t *testing.T) {
	svc, _ := newSvcDB(t, 0, WithProfilClient(&fakeProfil{visibility: "public"}))
	mustCreate(t, svc, idA, "alice")
	mustCreate(t, svc, idB, "bob")

	if _, err := svc.Follow(context.Background(), idA, "alice@x.com", idB); err != nil {
		t.Fatalf("Follow : %v", err)
	}

	followers, err := svc.ListFollowers(idB, 10, 0)
	if err != nil || len(followers) != 1 {
		t.Fatalf("ListFollowers = %d, %v", len(followers), err)
	}
	following, err := svc.ListFollowing(idA, 10, 0)
	if err != nil || len(following) != 1 {
		t.Fatalf("ListFollowing = %d, %v", len(following), err)
	}
	if _, err := svc.ListFollowers(idC, 10, 0); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("ListFollowers(absent) = %v, attendu ErrUserNotFound", err)
	}
	if _, err := svc.ListFollowing(idC, 10, 0); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("ListFollowing(absent) = %v, attendu ErrUserNotFound", err)
	}

	// RemoveFollower : bob retire alice ? Non — alice suit bob, donc bob retire
	// alice de SES abonnés.
	if err := svc.RemoveFollower(idB, idA); err != nil {
		t.Fatalf("RemoveFollower : %v", err)
	}
	if svc.IsFollowing(idA, idB) {
		t.Fatal("RemoveFollower doit retirer la relation")
	}

	if err := svc.Unfollow(idA, idB); err != nil {
		t.Fatalf("Unfollow : %v", err)
	}
}

func TestService_IsFollowing_AbsentUsers(t *testing.T) {
	svc, _ := newSvcDB(t, 0)
	mustCreate(t, svc, idA, "alice")
	// follower inexistant → false.
	if svc.IsFollowing(idC, idA) {
		t.Fatal("IsFollowing(absent, alice) doit être false")
	}
	// following inexistant → false.
	if svc.IsFollowing(idA, idC) {
		t.Fatal("IsFollowing(alice, absent) doit être false")
	}
}

func TestService_Block_Unblock_BlockedIDs_HasBlocked(t *testing.T) {
	svc, _ := newSvcDB(t, 0)
	mustCreate(t, svc, idA, "alice")
	mustCreate(t, svc, idB, "bob")

	if err := svc.Block(idA, idA); !errors.Is(err, ErrSelfBlock) {
		t.Fatalf("Block(self) = %v, attendu ErrSelfBlock", err)
	}
	if err := svc.Block(idA, idC); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("Block(cible absente) = %v, attendu ErrUserNotFound", err)
	}
	if err := svc.Block(idA, idB); err != nil {
		t.Fatalf("Block : %v", err)
	}
	if !svc.HasBlocked(idA, idB) {
		t.Fatal("HasBlocked doit être true")
	}
	if svc.HasBlocked(idA, idA) {
		t.Fatal("HasBlocked(self) doit être false")
	}

	ids, err := svc.BlockedIDs(idA)
	if err != nil || len(ids) != 1 || ids[0] != idB {
		t.Fatalf("BlockedIDs = %+v, %v", ids, err)
	}
	if _, err := svc.BlockedIDs(idC); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("BlockedIDs(absent) = %v, attendu ErrUserNotFound", err)
	}

	if err := svc.Unblock(idA, idA); !errors.Is(err, ErrSelfBlock) {
		t.Fatalf("Unblock(self) = %v, attendu ErrSelfBlock", err)
	}
	if err := svc.Unblock(idA, idB); err != nil {
		t.Fatalf("Unblock : %v", err)
	}
	if svc.HasBlocked(idA, idB) {
		t.Fatal("après Unblock, HasBlocked doit être false")
	}
}

// mustCreate insère un utilisateur via le service (validation incluse).
func mustCreate(t *testing.T, svc *UserService, id, username string) {
	t.Helper()
	if _, err := svc.Create(id, username); err != nil {
		t.Fatalf("Create(%s) : %v", username, err)
	}
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
