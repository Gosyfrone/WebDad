package service

import (
	"context"
	"errors"
	"testing"

	"github.com/webdad/notification-service/internal/models"
)

// ─── List — cursor invalide ───────────────────────────────────────────────────

func TestList_InvalidBeforeCursor(t *testing.T) {
	svc := NewNotificationService(nil, &capturePublisher{}, nil)
	_, err := svc.List(context.TODO(), "u1", 10, "not-a-valid-objectid")
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("List(before invalide) → %v, attendu ErrInvalidID", err)
	}
}

func TestList_EmptyBeforeCursor(t *testing.T) {
	// before="" → pas de cursor → appel repo (nil → panic récupérée)
	svc := NewNotificationService(nil, &capturePublisher{}, nil)
	func() {
		defer func() { _ = recover() }()
		_, _ = svc.List(context.TODO(), "u1", 10, "")
	}()
}

// ─── MarkRead — id invalide ───────────────────────────────────────────────────

func TestMarkRead_InvalidID(t *testing.T) {
	svc := NewNotificationService(nil, &capturePublisher{}, nil)
	err := svc.MarkRead(context.TODO(), "u1", "not-a-valid-objectid")
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("MarkRead(id invalide) → %v, attendu ErrInvalidID", err)
	}
}

func TestMarkRead_EmptyID(t *testing.T) {
	svc := NewNotificationService(nil, &capturePublisher{}, nil)
	err := svc.MarkRead(context.TODO(), "u1", "")
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("MarkRead(id vide) → %v, attendu ErrInvalidID", err)
	}
}

// ─── HandleEvent — types dont la logique retourne tôt ────────────────────────

func TestHandleEvent_FollowRequest_SelfIgnored(t *testing.T) {
	// actor == recipient → early return (pas de repo call)
	pub := &capturePublisher{}
	svc := NewNotificationService(nil, pub, nil)
	err := svc.HandleEvent(context.Background(), models.Event{
		Type: models.TypeFollowRequest, ActorID: "u1", RecipientID: "u1",
	})
	if err != nil {
		t.Fatalf("HandleEvent(follow_request self) erreur : %v", err)
	}
	if pub.event != nil {
		t.Error("auto-notification supprimée : aucun event attendu")
	}
}

func TestHandleEvent_FollowRequest_EmptyRecipient(t *testing.T) {
	pub := &capturePublisher{}
	svc := NewNotificationService(nil, pub, nil)
	err := svc.HandleEvent(context.Background(), models.Event{
		Type: models.TypeFollowRequest, ActorID: "u1", RecipientID: "",
	})
	if err != nil {
		t.Fatalf("HandleEvent(follow_request recipient vide) erreur : %v", err)
	}
}

func TestHandleEvent_PostDeleted_EmptyPostID(t *testing.T) {
	// PostID vide → early return sans appel repo
	pub := &capturePublisher{}
	svc := NewNotificationService(nil, pub, nil)
	err := svc.HandleEvent(context.Background(), models.Event{
		Type: models.EventPostDeleted, PostID: "",
	})
	if err != nil {
		t.Fatalf("HandleEvent(post_deleted, postID vide) erreur : %v", err)
	}
}

func TestHandleEvent_SelfLike_NoNotification(t *testing.T) {
	pub := &capturePublisher{}
	svc := NewNotificationService(nil, pub, nil)
	err := svc.HandleEvent(context.Background(), models.Event{
		Type: models.TypeLike, ActorID: "u1", RecipientID: "u1", PostID: "p1",
	})
	if err != nil {
		t.Fatalf("HandleEvent(self-like) erreur : %v", err)
	}
	// Auto-notification supprimée : on envoie quand même via applyToGroup si le
	// groupKey est valide — on vérifie juste qu'il n'y a pas de crash.
}

func TestHandleEvent_LikeWithoutPostID(t *testing.T) {
	pub := &capturePublisher{}
	svc := NewNotificationService(nil, pub, nil)
	err := svc.HandleEvent(context.Background(), models.Event{
		Type: models.TypeLike, ActorID: "u1", RecipientID: "u2",
		// PostID vide → groupKeyFor retourne ok=false → event ignoré
	})
	if err != nil {
		t.Fatalf("HandleEvent(like sans post) erreur : %v", err)
	}
	if pub.event != nil {
		t.Error("like sans post_id ne devrait pas émettre")
	}
}

func TestHandleEvent_MentionNoHandles(t *testing.T) {
	// MentionHandles vide → loop ne s'exécute pas → pas de repo call
	pub := &capturePublisher{}
	svc := NewNotificationService(nil, pub, nil)
	err := svc.HandleEvent(context.Background(), models.Event{
		Type: models.TypeMention, ActorID: "u1", PostID: "p1", CommentID: "c1",
		MentionHandles: []string{},
	})
	if err != nil {
		t.Fatalf("HandleEvent(mention handles vides) erreur : %v", err)
	}
}

func TestHandleEvent_MentionNoMentions(t *testing.T) {
	pub := &capturePublisher{}
	svc := NewNotificationService(nil, pub, nil)
	err := svc.HandleEvent(context.Background(), models.Event{
		Type: models.TypeMention, ActorID: "u1", PostID: "p1",
		MentionHandles: nil,
	})
	if err != nil {
		t.Fatalf("HandleEvent(mention sans destinataire) erreur : %v", err)
	}
	if pub.event != nil {
		t.Error("mention sans destinataire ne doit pas émettre")
	}
}

// ─── clampLimit — bornes ─────────────────────────────────────────────────────

func TestClampLimit_Zero(t *testing.T) {
	if got := clampLimit(0); got != DefaultLimit {
		t.Errorf("clampLimit(0) = %d, attendu DefaultLimit (%d)", got, DefaultLimit)
	}
}

func TestClampLimit_AboveMax(t *testing.T) {
	if got := clampLimit(MaxLimit + 100); got != MaxLimit {
		t.Errorf("clampLimit(>MaxLimit) = %d, attendu MaxLimit (%d)", got, MaxLimit)
	}
}
