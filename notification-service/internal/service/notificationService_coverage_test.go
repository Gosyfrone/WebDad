package service

import (
	"context"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/webdad/notification-service/internal/models"
)

type fakeRepository struct {
	upsert             func(*models.Notification) (*models.Notification, error)
	upsertUniqueActor  func(*models.Notification) (*models.Notification, error)
	decrement          func(string, string) (*models.Notification, bool, error)
	deleteByPost       func(string) ([]string, error)
	list               func(string, int64, *bson.ObjectID) ([]models.Notification, error)
	countUnread        func(string) (int64, error)
	markAllRead        func(string) error
	markRead           func(string, bson.ObjectID) error
	lastNotification   *models.Notification
	lastRecipient      string
	lastGroupKey       string
	lastLimit          int64
	lastCursor         *bson.ObjectID
	lastNotificationID bson.ObjectID
}

func (f *fakeRepository) Upsert(_ context.Context, n *models.Notification) (*models.Notification, error) {
	f.lastNotification = n
	if f.upsert != nil {
		return f.upsert(n)
	}
	return n, nil
}

func (f *fakeRepository) UpsertUniqueActor(_ context.Context, n *models.Notification) (*models.Notification, error) {
	f.lastNotification = n
	if f.upsertUniqueActor != nil {
		return f.upsertUniqueActor(n)
	}
	return n, nil
}

func (f *fakeRepository) Decrement(_ context.Context, recipient, groupKey string) (*models.Notification, bool, error) {
	f.lastRecipient, f.lastGroupKey = recipient, groupKey
	if f.decrement != nil {
		return f.decrement(recipient, groupKey)
	}
	return &models.Notification{RecipientID: recipient}, false, nil
}

func (f *fakeRepository) DeleteByPost(_ context.Context, postID string) ([]string, error) {
	if f.deleteByPost != nil {
		return f.deleteByPost(postID)
	}
	return nil, nil
}

func (f *fakeRepository) List(_ context.Context, recipient string, limit int64, cursor *bson.ObjectID) ([]models.Notification, error) {
	f.lastRecipient, f.lastLimit, f.lastCursor = recipient, limit, cursor
	if f.list != nil {
		return f.list(recipient, limit, cursor)
	}
	return nil, nil
}

func (f *fakeRepository) CountUnread(_ context.Context, recipient string) (int64, error) {
	f.lastRecipient = recipient
	if f.countUnread != nil {
		return f.countUnread(recipient)
	}
	return 0, nil
}

func (f *fakeRepository) MarkAllRead(_ context.Context, recipient string) error {
	f.lastRecipient = recipient
	if f.markAllRead != nil {
		return f.markAllRead(recipient)
	}
	return nil
}

func (f *fakeRepository) MarkRead(_ context.Context, recipient string, id bson.ObjectID) error {
	f.lastRecipient, f.lastNotificationID = recipient, id
	if f.markRead != nil {
		return f.markRead(recipient, id)
	}
	return nil
}

type resolverResult struct {
	id    string
	found bool
	err   error
}

type fakeResolver map[string]resolverResult

func (r fakeResolver) ResolveHandle(_ context.Context, handle string) (string, bool, error) {
	result := r[handle]
	return result.id, result.found, result.err
}

type historyPublisher struct {
	userIDs [][]string
	events  []any
}

func (p *historyPublisher) Publish(userIDs []string, event any) {
	p.userIDs = append(p.userIDs, userIDs)
	p.events = append(p.events, event)
}

func TestPushEvents(t *testing.T) {
	pub := &historyPublisher{}
	svc := NewNotificationService(nil, pub, nil)
	n := &models.Notification{RecipientID: "recipient"}

	svc.pushNotification(n)
	svc.pushDeleted("recipient", "notification-id")
	svc.pushRefresh("recipient")
	svc.pushFollowRequestDecision("recipient", "actor", "accepted")
	svc.pushIdentityUpdate(models.Event{TargetUserID: "target", Certification: "political", Role: "moderator"})

	if len(pub.events) != 5 {
		t.Fatalf("nombre d'événements publiés = %d, attendu 5", len(pub.events))
	}
	for i, recipients := range pub.userIDs[:4] {
		if len(recipients) != 1 || recipients[0] != "recipient" {
			t.Errorf("publication %d vers %#v, attendu recipient", i, recipients)
		}
	}
	if pub.userIDs[4] != nil {
		t.Fatalf("identity update doit être broadcast, obtenu %#v", pub.userIDs[4])
	}

	notification := pub.events[0].(map[string]any)
	if notification["type"] != "notification" || notification["data"] != n {
		t.Errorf("notification publiée = %#v", notification)
	}
	deleted := pub.events[1].(map[string]any)
	if deleted["type"] != "notification_deleted" {
		t.Errorf("suppression publiée = %#v", deleted)
	}
	refresh := pub.events[2].(map[string]any)
	if refresh["type"] != "notification_refresh" {
		t.Errorf("refresh publié = %#v", refresh)
	}
	decision := pub.events[3].(map[string]any)
	if decision["type"] != "follow_request_decision" {
		t.Errorf("décision publiée = %#v", decision)
	}
	identity := pub.events[4].(map[string]any)
	if identity["type"] != "identity_updated" {
		t.Errorf("identité publiée = %#v", identity)
	}
}

func TestHandleEventEarlyReturns(t *testing.T) {
	tests := []struct {
		name string
		ev   models.Event
	}{
		{"type inconnu", models.Event{Type: "unknown"}},
		{"identity sans cible", models.Event{Type: models.EventIdentityUpdated}},
		{"message sans destinataire", models.Event{Type: models.TypeMessage, ActorID: "actor", ConversationID: "conv"}},
		{"message à soi-même", models.Event{Type: models.TypeMessage, ActorID: "actor", RecipientID: "actor", ConversationID: "conv"}},
		{"message sans conversation", models.Event{Type: models.TypeMessage, ActorID: "actor", RecipientID: "recipient"}},
		{"retract message ignoré", models.Event{Type: models.TypeMessage, ActorID: "actor", RecipientID: "recipient", ConversationID: "conv", Retract: true}},
		{"mention message sans destinataire", models.Event{Type: models.TypeMessageMention, ActorID: "actor", ConversationID: "conv"}},
		{"mention message à soi-même", models.Event{Type: models.TypeMessageMention, ActorID: "actor", RecipientID: "actor", ConversationID: "conv"}},
		{"mention message sans conversation", models.Event{Type: models.TypeMessageMention, ActorID: "actor", RecipientID: "recipient"}},
		{"mention post sans source", models.Event{Type: models.TypeMention, ActorID: "actor", MentionHandles: []string{"alice"}}},
		{"acceptation sans destinataire", models.Event{Type: models.TypeFollowRequestAccepted, ActorID: "actor"}},
		{"acceptation sans acteur", models.Event{Type: models.TypeFollowRequestAccepted, RecipientID: "recipient"}},
		{"acceptation de soi-même", models.Event{Type: models.TypeFollowRequestAccepted, ActorID: "actor", RecipientID: "actor"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pub := &historyPublisher{}
			svc := NewNotificationService(nil, pub, nil)
			if err := svc.HandleEvent(context.Background(), tt.ev); err != nil {
				t.Fatalf("HandleEvent() = %v", err)
			}
			if len(pub.events) != 0 {
				t.Fatalf("publications inattendues = %#v", pub.events)
			}
		})
	}
}

func TestHandleEventIdentityUpdateBroadcasts(t *testing.T) {
	pub := &historyPublisher{}
	svc := NewNotificationService(nil, pub, nil)
	if err := svc.HandleEvent(context.Background(), models.Event{
		Type:          models.EventIdentityUpdated,
		ActorID:       "actor",
		TargetUserID:  "target",
		Certification: "public_figure",
		Role:          "admin",
	}); err != nil {
		t.Fatalf("HandleEvent(identity) = %v", err)
	}
	if len(pub.events) != 1 || pub.userIDs[0] != nil {
		t.Fatalf("broadcast identity = recipients %#v events %#v", pub.userIDs, pub.events)
	}
	data := pub.events[0].(map[string]any)["data"].(map[string]string)
	if data["user_id"] != "target" || data["certification"] != "public_figure" || data["role"] != "admin" {
		t.Fatalf("payload identité = %#v", data)
	}
}

func TestGroupKeyForAdditionalBranches(t *testing.T) {
	tests := []struct {
		name string
		ev   models.Event
		want string
		ok   bool
	}{
		{"comment like", models.Event{Type: models.TypeCommentLike, CommentID: "comment"}, "comment_like:comment", true},
		{"comment like invalide", models.Event{Type: models.TypeCommentLike}, "", false},
		{"comment invalide", models.Event{Type: models.TypeComment}, "", false},
		{"repost invalide", models.Event{Type: models.TypeRepost}, "", false},
		{"quote invalide", models.Event{Type: models.TypeQuote}, "", false},
		{"follow request", models.Event{Type: models.TypeFollowRequest, ActorID: "actor"}, "follow_request:actor", true},
		{"follow request invalide", models.Event{Type: models.TypeFollowRequest}, "", false},
		{"acceptation invalide", models.Event{Type: models.TypeFollowRequestAccepted}, "", false},
		{"confirmation invalide", models.Event{Type: models.TypeFollowRequestAcceptConfirm}, "", false},
		{"purge warning", models.Event{Type: models.TypePostPurgeWarning, PostID: "post"}, "post_purge_warning:post", true},
		{"purge warning invalide", models.Event{Type: models.TypePostPurgeWarning}, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := groupKeyFor(tt.ev)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("groupKeyFor() = (%q, %v), attendu (%q, %v)", got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestHandleEventPersistsAndPublishes(t *testing.T) {
	tests := []models.Event{
		{Type: models.TypeLike, ActorID: "actor", RecipientID: "recipient", PostID: "post"},
		{Type: models.TypeCommentLike, ActorID: "actor", RecipientID: "recipient", CommentID: "comment"},
		{Type: models.TypeMessageMention, ActorID: "actor", RecipientID: "recipient", ConversationID: "conversation"},
		{Type: models.TypeFollowRequestAccepted, ActorID: "actor", RecipientID: "recipient"},
	}

	for _, ev := range tests {
		t.Run(ev.Type, func(t *testing.T) {
			repo := &fakeRepository{}
			pub := &historyPublisher{}
			svc := NewNotificationService(repo, pub, nil)
			if err := svc.HandleEvent(context.Background(), ev); err != nil {
				t.Fatalf("HandleEvent() = %v", err)
			}
			if repo.lastNotification == nil || repo.lastNotification.LastActorID != "actor" {
				t.Fatalf("notification persistée = %#v", repo.lastNotification)
			}
			wantPublications := 1
			if ev.Type == models.TypeFollowRequestAccepted {
				wantPublications = 2
			}
			if len(pub.events) != wantPublications {
				t.Fatalf("publications = %d, attendu %d", len(pub.events), wantPublications)
			}
		})
	}
}

func TestHandleEventPersistenceErrors(t *testing.T) {
	wantErr := errors.New("mongo indisponible")
	repo := &fakeRepository{upsert: func(*models.Notification) (*models.Notification, error) {
		return nil, wantErr
	}}
	svc := NewNotificationService(repo, &historyPublisher{}, nil)
	err := svc.HandleEvent(context.Background(), models.Event{
		Type: models.TypeLike, ActorID: "actor", RecipientID: "recipient", PostID: "post",
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("HandleEvent() = %v, attendu %v", err, wantErr)
	}

	repo.upsertUniqueActor = func(*models.Notification) (*models.Notification, error) { return nil, wantErr }
	err = svc.HandleEvent(context.Background(), models.Event{
		Type: models.TypeMessage, ActorID: "actor", RecipientID: "recipient", ConversationID: "conversation",
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("HandleEvent(message) = %v, attendu %v", err, wantErr)
	}
}

func TestHandleEventMessageUsesUniqueActorRepository(t *testing.T) {
	repo := &fakeRepository{}
	pub := &historyPublisher{}
	svc := NewNotificationService(repo, pub, nil)
	err := svc.HandleEvent(context.Background(), models.Event{
		Type: models.TypeMessage, ActorID: "actor", RecipientID: "recipient", ConversationID: "conversation",
	})
	if err != nil {
		t.Fatalf("HandleEvent() = %v", err)
	}
	if repo.lastNotification == nil || repo.lastNotification.ConversationID != "conversation" {
		t.Fatalf("notification = %#v", repo.lastNotification)
	}
	if len(pub.events) != 1 {
		t.Fatalf("publications = %d, attendu 1", len(pub.events))
	}
}

func TestHandleEventRetractBranches(t *testing.T) {
	id := bson.NewObjectID()
	tests := []struct {
		name       string
		result     *models.Notification
		deleted    bool
		err        error
		wantErr    error
		wantEvents int
		wantType   string
	}{
		{"groupe absent", nil, false, mongo.ErrNoDocuments, nil, 0, ""},
		{"erreur dépôt", nil, false, errors.New("decrement"), errors.New("decrement"), 0, ""},
		{"notification mise à jour", &models.Notification{ID: id, RecipientID: "recipient"}, false, nil, nil, 1, "notification"},
		{"notification supprimée", &models.Notification{ID: id, RecipientID: "recipient"}, true, nil, nil, 1, "notification_deleted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepository{decrement: func(string, string) (*models.Notification, bool, error) {
				return tt.result, tt.deleted, tt.err
			}}
			pub := &historyPublisher{}
			svc := NewNotificationService(repo, pub, nil)
			err := svc.HandleEvent(context.Background(), models.Event{
				Type: models.TypeLike, ActorID: "actor", RecipientID: "recipient", PostID: "post", Retract: true,
			})
			if tt.wantErr != nil && err == nil {
				t.Fatalf("erreur attendue, obtenu nil")
			}
			if tt.wantErr == nil && err != nil {
				t.Fatalf("erreur inattendue: %v", err)
			}
			if len(pub.events) != tt.wantEvents {
				t.Fatalf("publications = %d, attendu %d", len(pub.events), tt.wantEvents)
			}
			if tt.wantType != "" && pub.events[0].(map[string]any)["type"] != tt.wantType {
				t.Errorf("type publié = %#v", pub.events[0])
			}
		})
	}
}

func TestHandleEventPostDeleted(t *testing.T) {
	repo := &fakeRepository{deleteByPost: func(postID string) ([]string, error) {
		if postID != "post" {
			t.Fatalf("postID = %q", postID)
		}
		return []string{"u1", "u2"}, nil
	}}
	pub := &historyPublisher{}
	if err := NewNotificationService(repo, pub, nil).HandleEvent(context.Background(), models.Event{
		Type: models.EventPostDeleted, ActorID: "actor", PostID: "post",
	}); err != nil {
		t.Fatalf("HandleEvent() = %v", err)
	}
	if len(pub.events) != 2 {
		t.Fatalf("refresh publiés = %d, attendu 2", len(pub.events))
	}

	wantErr := errors.New("delete")
	repo.deleteByPost = func(string) ([]string, error) { return nil, wantErr }
	if err := NewNotificationService(repo, pub, nil).HandleEvent(context.Background(), models.Event{
		Type: models.EventPostDeleted, ActorID: "actor", PostID: "post",
	}); !errors.Is(err, wantErr) {
		t.Fatalf("HandleEvent() = %v, attendu %v", err, wantErr)
	}
}

func TestHandleMentionsFiltersAndContinues(t *testing.T) {
	repo := &fakeRepository{}
	pub := &historyPublisher{}
	resolver := fakeResolver{
		"alice":   {id: "alice-id", found: true},
		"missing": {found: false},
		"self":    {id: "actor", found: true},
		"broken":  {err: errors.New("user service")},
	}
	svc := NewNotificationService(repo, pub, resolver)
	err := svc.HandleEvent(context.Background(), models.Event{
		Type: models.TypeMention, ActorID: "actor", PostID: "post",
		MentionHandles: []string{" ", "@alice", "alice", "missing", "self", "broken"},
	})
	if err != nil {
		t.Fatalf("HandleEvent() = %v", err)
	}
	if len(pub.events) != 1 || repo.lastNotification.RecipientID != "alice-id" {
		t.Fatalf("publications = %#v, notification = %#v", pub.events, repo.lastNotification)
	}

	repo.upsert = func(*models.Notification) (*models.Notification, error) { return nil, errors.New("upsert") }
	pub.events = nil
	if err := svc.HandleEvent(context.Background(), models.Event{
		Type: models.TypeMention, ActorID: "actor", PostID: "post", MentionHandles: []string{"alice"},
	}); err != nil {
		t.Fatalf("le fan-out mention est best-effort: %v", err)
	}
	if len(pub.events) != 0 {
		t.Fatalf("publication inattendue: %#v", pub.events)
	}
}

func TestReadOperations(t *testing.T) {
	id := bson.NewObjectID()
	wantErr := errors.New("repository")
	repo := &fakeRepository{
		list: func(string, int64, *bson.ObjectID) ([]models.Notification, error) {
			return []models.Notification{{ID: id}}, nil
		},
		countUnread: func(string) (int64, error) { return 7, nil },
	}
	svc := NewNotificationService(repo, &historyPublisher{}, nil)

	items, err := svc.List(context.Background(), "recipient", 999, id.Hex())
	if err != nil || len(items) != 1 || repo.lastLimit != MaxLimit || repo.lastCursor == nil || *repo.lastCursor != id {
		t.Fatalf("List() = (%#v, %v), limit=%d cursor=%v", items, err, repo.lastLimit, repo.lastCursor)
	}
	count, err := svc.UnreadCount(context.Background(), "recipient")
	if err != nil || count != 7 {
		t.Fatalf("UnreadCount() = (%d, %v)", count, err)
	}
	if err := svc.MarkAllRead(context.Background(), "recipient"); err != nil {
		t.Fatalf("MarkAllRead() = %v", err)
	}
	if err := svc.MarkRead(context.Background(), "recipient", id.Hex()); err != nil {
		t.Fatalf("MarkRead() = %v", err)
	}
	if repo.lastNotificationID != id {
		t.Fatalf("id transmis = %s, attendu %s", repo.lastNotificationID, id)
	}

	repo.markAllRead = func(string) error { return wantErr }
	if err := svc.MarkAllRead(context.Background(), "recipient"); !errors.Is(err, wantErr) {
		t.Fatalf("MarkAllRead() = %v", err)
	}
	repo.markRead = func(string, bson.ObjectID) error { return mongo.ErrNoDocuments }
	if err := svc.MarkRead(context.Background(), "recipient", id.Hex()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("MarkRead(not found) = %v", err)
	}
	repo.markRead = func(string, bson.ObjectID) error { return wantErr }
	if err := svc.MarkRead(context.Background(), "recipient", id.Hex()); !errors.Is(err, wantErr) {
		t.Fatalf("MarkRead(repository) = %v", err)
	}
}
