package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/webdad/user-service/internal/client"
)

// Follow : une erreur de profil-service (visibilité) est propagée.
func TestService_Follow_VisibilityError(t *testing.T) {
	profilErr := errors.New("profil indisponible")
	svc, _ := newSvcDB(t, 0, WithProfilClient(&fakeProfil{err: profilErr}))
	mustCreate(t, svc, idA, "alice")
	mustCreate(t, svc, idB, "bob")

	if _, err := svc.Follow(context.Background(), idA, "alice@x.com", idB); err == nil {
		t.Fatal("Follow doit propager l'erreur de visibilité du profil")
	}
}

// Block / Unblock : la cible inexistante (2e requireExists) → ErrUserNotFound.
func TestService_BlockUnblock_TargetNotFound(t *testing.T) {
	svc, _ := newSvcDB(t, 0)
	mustCreate(t, svc, idA, "alice") // bloqueur existe, cible (idC) absente

	if err := svc.Block(idA, idC); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("Block(cible absente) = %v, attendu ErrUserNotFound", err)
	}
	if err := svc.Unblock(idA, idC); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("Unblock(cible absente) = %v, attendu ErrUserNotFound", err)
	}
}

// defaultUsername : la partie locale trop longue est tronquée à 50 caractères.
func TestDefaultUsername_Truncation(t *testing.T) {
	long := strings.Repeat("a", 80) + "@example.com"
	got := defaultUsername(long)
	if len(got) != 50 {
		t.Fatalf("defaultUsername len = %d, attendu 50", len(got))
	}
}

// defaultUsername : partie locale vide/illisible → repli "user".
func TestDefaultUsername_Fallback(t *testing.T) {
	if got := defaultUsername("@@@@@@@@@"); got != "user" {
		t.Fatalf("defaultUsername(illisible) = %q, attendu user", got)
	}
}

// Update (cooldown désactivé) sur un utilisateur inexistant : repo.Update
// renvoie sql.ErrNoRows → ErrUserNotFound.
func TestService_Update_NotFound_NoCooldown(t *testing.T) {
	svc, _ := newSvcDB(t, 0)
	name := "ghost"
	if _, err := svc.Update(idC, &name, nil); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("Update(absent) = %v, attendu ErrUserNotFound", err)
	}
}

// provision : épuisement de tous les candidats (base, base_<short>, user_<short>
// tous pris) → ProvisionFromClaims propage une erreur.
func TestService_Provision_AllCandidatesTaken(t *testing.T) {
	svc, repo := newSvcDB(t, 0)
	// short = 8 premiers caractères de l'UUID idA sans tirets = "aaaaaaaa".
	// base dérivée de "bob@x.com" = "bob".
	for i, name := range []string{"bob", "bob_aaaaaaaa", "user_aaaaaaaa"} {
		id := []string{idB, idC, "dddddddd-1111-0000-0000-000000000004"}[i]
		if _, err := repo.Create(id, name); err != nil {
			t.Fatalf("seed %s : %v", name, err)
		}
	}
	if _, err := svc.ProvisionFromClaims(idA, "bob@x.com"); err == nil {
		t.Fatal("ProvisionFromClaims doit échouer quand tous les candidats sont pris")
	}
}

// Emit (notification) sans client configuré : no-op silencieux.
func TestService_Emit_NoNotifier(t *testing.T) {
	svc, _ := newSvcDB(t, 0) // pas de WithNotificationClient
	// emitFollow & co ne paniquent pas et ne font rien.
	svc.emitFollow(idA, idB)
	svc.emitFollowRequest(idA, idB, false)
	svc.emitFollowRequestAcceptConfirm(idA, idB)
	svc.emitFollowRequestDecision(idA, idB, client.TypeFollowRequestAccepted)
}

func TestService_Emit_WithNotifier(t *testing.T) {
	notifier := &fakeNotifier{}
	svc := New(nil, 0, WithNotificationClient(notifier))
	svc.emitFollow(idA, idB)
	svc.emitFollowRequest(idA, idB, false)
	svc.emitFollowRequestAcceptConfirm(idA, idB)
	svc.emitFollowRequestDecision(idA, idB, client.TypeFollowRequestRejected)

	want := []string{
		client.TypeFollow,
		client.TypeFollowRequest,
		client.TypeFollowRequestAcceptConfirm,
		client.TypeFollowRequestRejected,
	}
	got := notifier.types()
	if len(got) != len(want) {
		t.Fatalf("événements = %v, attendu %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("événement %d = %q, attendu %q", i, got[i], want[i])
		}
	}
}

func TestService_SoftDelete_DelegatesAndWrapsError(t *testing.T) {
	svc := closedSvc(t)
	if err := svc.SoftDelete(idA); err == nil {
		t.Fatal("SoftDelete doit propager l'erreur du repository")
	}
}
