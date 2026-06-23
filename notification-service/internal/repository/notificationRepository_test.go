package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/webdad/notification-service/internal/models"
)

type collectionMock struct {
	findOneResults []*mongo.SingleResult
	deleteOne      *mongo.DeleteResult
	deleteOneErr   error
	deleteMany     *mongo.DeleteResult
	deleteManyErr  error
	findCursor     *mongo.Cursor
	findErr        error
	count          int64
	countErr       error
	updateMany     *mongo.UpdateResult
	updateManyErr  error
	updateOne      *mongo.UpdateResult
	updateOneErr   error
	findCalls      int
}

func (m *collectionMock) FindOneAndUpdate(context.Context, any, any, ...options.Lister[options.FindOneAndUpdateOptions]) *mongo.SingleResult {
	result := m.findOneResults[0]
	m.findOneResults = m.findOneResults[1:]
	return result
}

func (m *collectionMock) DeleteOne(context.Context, any, ...options.Lister[options.DeleteOneOptions]) (*mongo.DeleteResult, error) {
	return m.deleteOne, m.deleteOneErr
}

func (m *collectionMock) DeleteMany(context.Context, any, ...options.Lister[options.DeleteManyOptions]) (*mongo.DeleteResult, error) {
	return m.deleteMany, m.deleteManyErr
}

func (m *collectionMock) Find(context.Context, any, ...options.Lister[options.FindOptions]) (*mongo.Cursor, error) {
	m.findCalls++
	return m.findCursor, m.findErr
}

func (m *collectionMock) CountDocuments(context.Context, any, ...options.Lister[options.CountOptions]) (int64, error) {
	return m.count, m.countErr
}

func (m *collectionMock) UpdateMany(context.Context, any, any, ...options.Lister[options.UpdateManyOptions]) (*mongo.UpdateResult, error) {
	return m.updateMany, m.updateManyErr
}

func (m *collectionMock) UpdateOne(context.Context, any, any, ...options.Lister[options.UpdateOneOptions]) (*mongo.UpdateResult, error) {
	return m.updateOne, m.updateOneErr
}

func notificationDocument(overrides bson.M) bson.M {
	doc := bson.M{
		"_id": bson.NewObjectID(), "recipient_id": "recipient", "group_key": "like:post",
		"type": models.TypeLike, "last_actor_id": "actor", "count": int32(1),
		"is_read": false, "created_at": time.Now(), "updated_at": time.Now(),
	}
	for key, value := range overrides {
		doc[key] = value
	}
	return doc
}

func singleResult(doc bson.M, err error) *mongo.SingleResult {
	return mongo.NewSingleResultFromDocument(doc, err, nil)
}

func cursorFrom(t *testing.T, docs []any, err error) *mongo.Cursor {
	t.Helper()
	cursor, createErr := mongo.NewCursorFromDocuments(docs, err, nil)
	if createErr != nil {
		t.Fatalf("NewCursorFromDocuments: %v", createErr)
	}
	return cursor
}

func TestConstructors(t *testing.T) {
	client, err := mongo.Connect(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Fatalf("mongo.Connect: %v", err)
	}
	t.Cleanup(func() { _ = client.Disconnect(context.Background()) })
	repo := NewNotificationRepository(client.Database("test"))
	if repo == nil || repo.notifications == nil || repo.distinctRecipientIDs == nil {
		t.Fatal("NewNotificationRepository retourne un dépôt invalide")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := repo.distinctRecipientIDs(ctx, bson.M{}); err == nil {
		t.Fatal("distinctRecipientIDs devait relayer l'erreur du contexte annulé")
	}
	mock := &collectionMock{}
	if repo := newNotificationRepository(mock); repo.notifications != mock || repo.distinctRecipientIDs == nil {
		t.Fatal("newNotificationRepository n'a pas conservé la collection")
	}
}

func TestUpsert(t *testing.T) {
	wantErr := errors.New("upsert")
	tests := []struct {
		name string
		doc  bson.M
		err  error
	}{
		{"succès", notificationDocument(nil), nil},
		{"erreur", bson.M{}, wantErr},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &collectionMock{findOneResults: []*mongo.SingleResult{singleResult(tt.doc, tt.err)}}
			got, err := newNotificationRepository(mock).Upsert(context.Background(), &models.Notification{
				RecipientID: "recipient", GroupKey: "like:post", Type: models.TypeLike, LastActorID: "actor",
			})
			if tt.err != nil {
				if !errors.Is(err, tt.err) || got != nil {
					t.Fatalf("Upsert = (%#v, %v)", got, err)
				}
				return
			}
			if err != nil || got.Count != 1 {
				t.Fatalf("Upsert = (%#v, %v)", got, err)
			}
		})
	}
}

func TestUpsertUniqueActor(t *testing.T) {
	t.Run("compte déjà synchronisé", func(t *testing.T) {
		mock := &collectionMock{findOneResults: []*mongo.SingleResult{singleResult(notificationDocument(bson.M{
			"actor_ids": bson.A{"actor"}, "count": int32(1),
		}), nil)}}
		got, err := newNotificationRepository(mock).UpsertUniqueActor(context.Background(), &models.Notification{
			RecipientID: "recipient", GroupKey: "message", Type: models.TypeMessage, LastActorID: "actor",
		})
		if err != nil || got.Count != 1 || len(mock.findOneResults) != 0 {
			t.Fatalf("UpsertUniqueActor = (%#v, %v)", got, err)
		}
	})

	t.Run("resynchronise le compte", func(t *testing.T) {
		mock := &collectionMock{findOneResults: []*mongo.SingleResult{
			singleResult(notificationDocument(bson.M{"actor_ids": bson.A{"a", "b"}, "count": int32(1)}), nil),
			singleResult(notificationDocument(bson.M{"actor_ids": bson.A{"a", "b"}, "count": int32(2)}), nil),
		}}
		got, err := newNotificationRepository(mock).UpsertUniqueActor(context.Background(), &models.Notification{RecipientID: "recipient"})
		if err != nil || got.Count != 2 {
			t.Fatalf("UpsertUniqueActor = (%#v, %v)", got, err)
		}
	})

	t.Run("minimum un", func(t *testing.T) {
		mock := &collectionMock{findOneResults: []*mongo.SingleResult{
			singleResult(notificationDocument(bson.M{"actor_ids": bson.A{}, "count": int32(0)}), nil),
			singleResult(notificationDocument(bson.M{"count": int32(1)}), nil),
		}}
		got, err := newNotificationRepository(mock).UpsertUniqueActor(context.Background(), &models.Notification{})
		if err != nil || got.Count != 1 {
			t.Fatalf("UpsertUniqueActor = (%#v, %v)", got, err)
		}
	})

	wantErr := errors.New("unique")
	for _, results := range [][]*mongo.SingleResult{
		{singleResult(bson.M{}, wantErr)},
		{singleResult(notificationDocument(bson.M{"actor_ids": bson.A{"a", "b"}, "count": int32(1)}), nil), singleResult(bson.M{}, wantErr)},
	} {
		mock := &collectionMock{findOneResults: results}
		if got, err := newNotificationRepository(mock).UpsertUniqueActor(context.Background(), &models.Notification{}); !errors.Is(err, wantErr) || got != nil {
			t.Fatalf("UpsertUniqueActor erreur = (%#v, %v)", got, err)
		}
	}
}

func TestDecrement(t *testing.T) {
	id := bson.NewObjectID()
	wantErr := errors.New("decrement")
	tests := []struct {
		name      string
		result    *mongo.SingleResult
		deleteErr error
		deleted   bool
		wantErr   error
	}{
		{"erreur update", singleResult(bson.M{}, wantErr), nil, false, wantErr},
		{"reste positif", singleResult(notificationDocument(bson.M{"_id": id, "count": int32(1)}), nil), nil, false, nil},
		{"supprimé", singleResult(notificationDocument(bson.M{"_id": id, "count": int32(0)}), nil), nil, true, nil},
		{"erreur suppression", singleResult(notificationDocument(bson.M{"_id": id, "count": int32(0)}), nil), wantErr, false, wantErr},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &collectionMock{findOneResults: []*mongo.SingleResult{tt.result}, deleteOne: &mongo.DeleteResult{DeletedCount: 1}, deleteOneErr: tt.deleteErr}
			n, deleted, err := newNotificationRepository(mock).Decrement(context.Background(), "recipient", "like:post")
			if !errors.Is(err, tt.wantErr) || deleted != tt.deleted {
				t.Fatalf("Decrement = (%#v, %v, %v)", n, deleted, err)
			}
		})
	}
}

func TestDeleteByPost(t *testing.T) {
	mock := &collectionMock{deleteMany: &mongo.DeleteResult{DeletedCount: 4}}
	repo := newNotificationRepository(mock)
	repo.distinctRecipientIDs = func(context.Context, bson.M) ([]string, error) {
		return []string{"u1", "u2"}, nil
	}
	recipients, err := repo.DeleteByPost(context.Background(), "post")
	if err != nil || len(recipients) != 2 || recipients[0] != "u1" || recipients[1] != "u2" {
		t.Fatalf("DeleteByPost = (%#v, %v)", recipients, err)
	}

	wantErr := errors.New("delete post")
	for _, tc := range []struct {
		name        string
		distinctErr error
		deleteErr   error
	}{
		{"distinct", wantErr, nil},
		{"delete", nil, wantErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mock := &collectionMock{deleteManyErr: tc.deleteErr}
			repo := newNotificationRepository(mock)
			repo.distinctRecipientIDs = func(context.Context, bson.M) ([]string, error) {
				return []string{"u1"}, tc.distinctErr
			}
			if _, err := repo.DeleteByPost(context.Background(), "post"); err == nil {
				t.Fatalf("DeleteByPost = %v", err)
			}
		})
	}
}

func TestList(t *testing.T) {
	docs := []any{notificationDocument(nil), notificationDocument(bson.M{"group_key": "comment:post"})}
	mock := &collectionMock{findCursor: cursorFrom(t, docs, nil)}
	before := bson.NewObjectID()
	items, err := newNotificationRepository(mock).List(context.Background(), "recipient", 20, &before)
	if err != nil || len(items) != 2 || mock.findCalls != 1 {
		t.Fatalf("List = (%#v, %v)", items, err)
	}

	wantErr := errors.New("list")
	for _, failing := range []*collectionMock{
		{findErr: wantErr},
		{findCursor: cursorFrom(t, []any{bson.M{"_id": "invalid-object-id"}}, nil)},
	} {
		if _, err := newNotificationRepository(failing).List(context.Background(), "recipient", 20, nil); err == nil {
			t.Fatalf("List erreur = %v", err)
		}
	}
}

func TestReadStateOperations(t *testing.T) {
	wantErr := errors.New("mongo")
	repo := newNotificationRepository(&collectionMock{count: 4, updateMany: &mongo.UpdateResult{}, updateOne: &mongo.UpdateResult{MatchedCount: 1}})
	if count, err := repo.CountUnread(context.Background(), "recipient"); err != nil || count != 4 {
		t.Fatalf("CountUnread = (%d, %v)", count, err)
	}
	if err := repo.MarkAllRead(context.Background(), "recipient"); err != nil {
		t.Fatalf("MarkAllRead = %v", err)
	}
	if err := repo.MarkRead(context.Background(), "recipient", bson.NewObjectID()); err != nil {
		t.Fatalf("MarkRead = %v", err)
	}

	if _, err := newNotificationRepository(&collectionMock{countErr: wantErr}).CountUnread(context.Background(), "recipient"); !errors.Is(err, wantErr) {
		t.Fatalf("CountUnread erreur = %v", err)
	}
	if err := newNotificationRepository(&collectionMock{updateManyErr: wantErr}).MarkAllRead(context.Background(), "recipient"); !errors.Is(err, wantErr) {
		t.Fatalf("MarkAllRead erreur = %v", err)
	}
	if err := newNotificationRepository(&collectionMock{updateOneErr: wantErr}).MarkRead(context.Background(), "recipient", bson.NewObjectID()); !errors.Is(err, wantErr) {
		t.Fatalf("MarkRead erreur = %v", err)
	}
	if err := newNotificationRepository(&collectionMock{updateOne: &mongo.UpdateResult{}}).MarkRead(context.Background(), "recipient", bson.NewObjectID()); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("MarkRead absent = %v", err)
	}
}
