package repository

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/webdad/notification-service/internal/models"
)

func integrationRepository(t *testing.T) (*NotificationRepository, *mongo.Database) {
	t.Helper()
	if os.Getenv("RUN_MONGO_INTEGRATION") != "1" {
		t.Skip("définir RUN_MONGO_INTEGRATION=1 pour les tests Mongo")
	}
	host := envOr("MONGO_HOST", "localhost")
	port := envOr("MONGO_PORT", "27017")
	user := os.Getenv("MONGO_INITDB_ROOT_USERNAME")
	password := os.Getenv("MONGO_INITDB_ROOT_PASSWORD")
	uri := fmt.Sprintf("mongodb://%s:%s", host, port)
	if user != "" {
		uri = fmt.Sprintf("mongodb://%s:%s@%s:%s/?authSource=admin", user, password, host, port)
	}
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatalf("mongo.Connect: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	if err := client.Ping(ctx, nil); err != nil {
		t.Fatalf("mongo ping: %v", err)
	}
	db := client.Database(fmt.Sprintf("notification_repository_test_%d", time.Now().UnixNano()))
	t.Cleanup(func() {
		_ = db.Drop(context.Background())
		_ = client.Disconnect(context.Background())
	})
	return NewNotificationRepository(db), db
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func TestRepositoryAggregationAndReads(t *testing.T) {
	repo, _ := integrationRepository(t)
	ctx := context.Background()
	base := &models.Notification{
		RecipientID: "recipient", GroupKey: "like:post", Type: models.TypeLike,
		PostID: "post", LastActorID: "actor",
	}
	first, err := repo.Upsert(ctx, base)
	if err != nil || first.Count != 1 || first.IsRead {
		t.Fatalf("premier Upsert = (%#v, %v)", first, err)
	}
	second, err := repo.Upsert(ctx, base)
	if err != nil || second.Count != 2 {
		t.Fatalf("second Upsert = (%#v, %v)", second, err)
	}

	items, err := repo.List(ctx, "recipient", 10, nil)
	if err != nil || len(items) != 1 {
		t.Fatalf("List = (%#v, %v)", items, err)
	}
	before := bson.NewObjectID()
	items, err = repo.List(ctx, "recipient", 10, &before)
	if err != nil || len(items) != 1 {
		t.Fatalf("List avec curseur = (%#v, %v)", items, err)
	}
	count, err := repo.CountUnread(ctx, "recipient")
	if err != nil || count != 1 {
		t.Fatalf("CountUnread = (%d, %v)", count, err)
	}
	if err := repo.MarkRead(ctx, "recipient", second.ID); err != nil {
		t.Fatalf("MarkRead = %v", err)
	}
	count, err = repo.CountUnread(ctx, "recipient")
	if err != nil || count != 0 {
		t.Fatalf("CountUnread après lecture = (%d, %v)", count, err)
	}
	if err := repo.MarkRead(ctx, "other", second.ID); err != mongo.ErrNoDocuments {
		t.Fatalf("MarkRead autre destinataire = %v", err)
	}

	third, err := repo.Upsert(ctx, &models.Notification{
		RecipientID: "recipient", GroupKey: "comment:post", Type: models.TypeComment,
		PostID: "post", LastActorID: "actor2",
	})
	if err != nil || third == nil {
		t.Fatalf("troisième Upsert = (%#v, %v)", third, err)
	}
	if err := repo.MarkAllRead(ctx, "recipient"); err != nil {
		t.Fatalf("MarkAllRead = %v", err)
	}
	count, _ = repo.CountUnread(ctx, "recipient")
	if count != 0 {
		t.Fatalf("non lues après MarkAllRead = %d", count)
	}
}

func TestRepositoryUniqueActorsDecrementAndDeleteByPost(t *testing.T) {
	repo, _ := integrationRepository(t)
	ctx := context.Background()
	message := &models.Notification{
		RecipientID: "recipient", GroupKey: "message", Type: models.TypeMessage,
		ConversationID: "conversation", LastActorID: "actor1",
	}
	first, err := repo.UpsertUniqueActor(ctx, message)
	if err != nil || first.Count != 1 || len(first.ActorIDs) != 1 {
		t.Fatalf("premier acteur = (%#v, %v)", first, err)
	}
	same, err := repo.UpsertUniqueActor(ctx, message)
	if err != nil || same.Count != 1 {
		t.Fatalf("même acteur = (%#v, %v)", same, err)
	}
	message.LastActorID = "actor2"
	second, err := repo.UpsertUniqueActor(ctx, message)
	if err != nil || second.Count != 2 || len(second.ActorIDs) != 2 {
		t.Fatalf("second acteur = (%#v, %v)", second, err)
	}

	like := &models.Notification{
		RecipientID: "recipient", GroupKey: "like:post", Type: models.TypeLike,
		PostID: "post", LastActorID: "actor",
	}
	_, _ = repo.Upsert(ctx, like)
	_, _ = repo.Upsert(ctx, like)
	decremented, deleted, err := repo.Decrement(ctx, "recipient", "like:post")
	if err != nil || deleted || decremented.Count != 1 {
		t.Fatalf("premier Decrement = (%#v, %v, %v)", decremented, deleted, err)
	}
	decremented, deleted, err = repo.Decrement(ctx, "recipient", "like:post")
	if err != nil || !deleted || decremented.Count != 0 {
		t.Fatalf("second Decrement = (%#v, %v, %v)", decremented, deleted, err)
	}
	if _, _, err := repo.Decrement(ctx, "recipient", "like:post"); err != mongo.ErrNoDocuments {
		t.Fatalf("Decrement absent = %v", err)
	}

	for _, recipient := range []string{"u1", "u2", "u2"} {
		_, err := repo.Upsert(ctx, &models.Notification{
			RecipientID: recipient, GroupKey: "mention:post:" + recipient, Type: models.TypeMention,
			PostID: "post-delete", LastActorID: "actor",
		})
		if err != nil {
			t.Fatalf("préparation DeleteByPost: %v", err)
		}
	}
	recipients, err := repo.DeleteByPost(ctx, "post-delete")
	if err != nil || len(recipients) != 2 {
		t.Fatalf("DeleteByPost = (%#v, %v)", recipients, err)
	}
}

func TestRepositoryCanceledContextErrors(t *testing.T) {
	repo, _ := integrationRepository(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	n := &models.Notification{RecipientID: "u", GroupKey: "like:p", Type: models.TypeLike, LastActorID: "a"}
	id := bson.NewObjectID()

	checks := []struct {
		name string
		call func() error
	}{
		{"upsert", func() error { _, err := repo.Upsert(ctx, n); return err }},
		{"unique", func() error { _, err := repo.UpsertUniqueActor(ctx, n); return err }},
		{"decrement", func() error { _, _, err := repo.Decrement(ctx, "u", "like:p"); return err }},
		{"delete post", func() error { _, err := repo.DeleteByPost(ctx, "p"); return err }},
		{"distinct", func() error { _, err := repo.distinctRecipients(ctx, bson.M{}); return err }},
		{"list", func() error { _, err := repo.List(ctx, "u", 10, nil); return err }},
		{"count", func() error { _, err := repo.CountUnread(ctx, "u"); return err }},
		{"mark all", func() error { return repo.MarkAllRead(ctx, "u") }},
		{"mark read", func() error { return repo.MarkRead(ctx, "u", id) }},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			if err := check.call(); err == nil {
				t.Fatal("erreur attendue avec contexte annulé")
			}
		})
	}
}
