package repository

import (
	"context"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/webdad/profil-service/internal/models"
)

type fakeCollection struct {
	one        *models.Profil
	many       []models.Profil
	err        error
	deleted    int64
	inserted   any
	filter     any
	update     any
	findFilter any
}

func (f *fakeCollection) FindOne(_ context.Context, filter any, _ ...options.Lister[options.FindOneOptions]) singleResult {
	f.filter = filter
	return fakeSingleResult{profil: f.one, err: f.err}
}

func (f *fakeCollection) InsertOne(_ context.Context, document any, _ ...options.Lister[options.InsertOneOptions]) (*mongo.InsertOneResult, error) {
	f.inserted = document
	if f.err != nil {
		return nil, f.err
	}
	return &mongo.InsertOneResult{InsertedID: "id"}, nil
}

func (f *fakeCollection) Find(_ context.Context, filter any, _ ...options.Lister[options.FindOptions]) (cursorAPI, error) {
	f.findFilter = filter
	if f.err != nil {
		return nil, f.err
	}
	return &fakeCursor{profils: f.many}, nil
}

func (f *fakeCollection) FindOneAndUpdate(_ context.Context, filter any, update any, _ ...options.Lister[options.FindOneAndUpdateOptions]) singleResult {
	f.filter = filter
	f.update = update
	return fakeSingleResult{profil: f.one, err: f.err}
}

func (f *fakeCollection) DeleteOne(_ context.Context, filter any, _ ...options.Lister[options.DeleteOneOptions]) (*mongo.DeleteResult, error) {
	f.filter = filter
	if f.err != nil {
		return nil, f.err
	}
	return &mongo.DeleteResult{DeletedCount: f.deleted}, nil
}

type fakeSingleResult struct {
	profil *models.Profil
	err    error
}

func (f fakeSingleResult) Decode(v any) error {
	if f.err != nil {
		return f.err
	}
	if f.profil == nil {
		return mongo.ErrNoDocuments
	}
	out := v.(*models.Profil)
	*out = *f.profil
	return nil
}

type fakeCursor struct {
	profils  []models.Profil
	allErr   error
	closed   bool
	closeErr error
}

func (f *fakeCursor) All(_ context.Context, results any) error {
	if f.allErr != nil {
		return f.allErr
	}
	out := results.(*[]models.Profil)
	*out = append((*out)[:0], f.profils...)
	return nil
}

func (f *fakeCursor) Close(context.Context) error {
	f.closed = true
	return f.closeErr
}

func TestProfilRepository_GetByUserID(t *testing.T) {
	coll := &fakeCollection{one: &models.Profil{UserID: "u1", DisplayName: "Alice"}}
	repo := &ProfilRepository{collection: coll}

	got, err := repo.GetByUserID(context.Background(), "u1")
	if err != nil {
		t.Fatalf("GetByUserID erreur: %v", err)
	}
	if got.UserID != "u1" {
		t.Fatalf("profil = %#v", got)
	}
	if filter := coll.filter.(bson.M); filter["user_id"] != "u1" {
		t.Fatalf("filtre inattendu: %#v", coll.filter)
	}
}

func TestMongoCollectionAdapter(t *testing.T) {
	client, err := mongo.Connect(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Fatalf("mongo.Connect: %v", err)
	}
	coll := mongoCollection{collection: client.Database("profil_adapter_test").Collection("profiles")}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := coll.FindOne(ctx, bson.M{"user_id": "u1"}).Decode(&models.Profil{}); err == nil {
		t.Fatal("FindOne avec contexte annule devrait echouer")
	}
	if _, err := coll.InsertOne(ctx, &models.Profil{UserID: "u1"}); err == nil {
		t.Fatal("InsertOne avec contexte annule devrait echouer")
	}
	if _, err := coll.Find(ctx, bson.M{}); err == nil {
		t.Fatal("Find avec contexte annule devrait echouer")
	}
	if err := coll.FindOneAndUpdate(ctx, bson.M{"user_id": "u1"}, bson.M{"$set": bson.M{"display_name": "Alice"}}).Decode(&models.Profil{}); err == nil {
		t.Fatal("FindOneAndUpdate avec contexte annule devrait echouer")
	}
	if _, err := coll.DeleteOne(ctx, bson.M{"user_id": "u1"}); err == nil {
		t.Fatal("DeleteOne avec contexte annule devrait echouer")
	}
}

func TestNewProfilRepository(t *testing.T) {
	client, err := mongo.Connect(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Fatal(err)
	}
	repo := NewProfilRepository(client.Database("profil_constructor_test"))
	if repo == nil || repo.collection == nil {
		t.Fatalf("repository mal construit: %#v", repo)
	}
}

func TestProfilRepository_GetByUserID_Error(t *testing.T) {
	sentinel := errors.New("db")
	repo := &ProfilRepository{collection: &fakeCollection{err: sentinel}}
	if _, err := repo.GetByUserID(context.Background(), "u1"); !errors.Is(err, sentinel) {
		t.Fatalf("attendu sentinel, obtenu %v", err)
	}
}

func TestProfilRepository_Insert(t *testing.T) {
	coll := &fakeCollection{}
	repo := &ProfilRepository{collection: coll}
	p := &models.Profil{UserID: "u1"}

	if err := repo.Insert(context.Background(), p); err != nil {
		t.Fatalf("Insert erreur: %v", err)
	}
	if coll.inserted != p {
		t.Fatalf("document insere inattendu: %#v", coll.inserted)
	}
}

func TestProfilRepository_SearchByDisplayName(t *testing.T) {
	coll := &fakeCollection{many: []models.Profil{{UserID: "u1"}, {UserID: "u2"}}}
	repo := &ProfilRepository{collection: coll}

	got, err := repo.SearchByDisplayName(context.Background(), "ali", 5)
	if err != nil {
		t.Fatalf("SearchByDisplayName erreur: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("resultats = %d", len(got))
	}
	filter := coll.findFilter.(bson.M)
	displayName := filter["display_name"].(bson.M)
	if displayName["$regex"] != "ali" || displayName["$options"] != "i" {
		t.Fatalf("filtre recherche inattendu: %#v", filter)
	}
}

func TestProfilRepository_SearchByDisplayName_FindError(t *testing.T) {
	sentinel := errors.New("find")
	repo := &ProfilRepository{collection: &fakeCollection{err: sentinel}}
	if _, err := repo.SearchByDisplayName(context.Background(), "ali", 5); !errors.Is(err, sentinel) {
		t.Fatalf("attendu sentinel, obtenu %v", err)
	}
}

func TestProfilRepository_SearchByDisplayName_DecodeError(t *testing.T) {
	sentinel := errors.New("decode")
	coll := &fakeCollection{}
	repo := &ProfilRepository{collection: collectionWithCursor{fakeCollection: coll, cursor: &fakeCursor{allErr: sentinel}}}
	if _, err := repo.SearchByDisplayName(context.Background(), "ali", 5); !errors.Is(err, sentinel) {
		t.Fatalf("attendu sentinel, obtenu %v", err)
	}
}

type collectionWithCursor struct {
	*fakeCollection
	cursor cursorAPI
}

func (f collectionWithCursor) Find(context.Context, any, ...options.Lister[options.FindOptions]) (cursorAPI, error) {
	return f.cursor, nil
}

func TestProfilRepository_Update(t *testing.T) {
	coll := &fakeCollection{one: &models.Profil{UserID: "u1", DisplayName: "Alice"}}
	repo := &ProfilRepository{collection: coll}
	set := bson.M{"display_name": "Bob"}

	got, err := repo.Update(context.Background(), "u1", set)
	if err != nil {
		t.Fatalf("Update erreur: %v", err)
	}
	if got.DisplayName != "Alice" {
		t.Fatalf("profil = %#v", got)
	}
	update := coll.update.(bson.M)
	if _, ok := update["$set"]; !ok {
		t.Fatalf("update sans $set: %#v", update)
	}
}

func TestProfilRepository_UpdateError(t *testing.T) {
	sentinel := errors.New("update")
	repo := &ProfilRepository{collection: &fakeCollection{err: sentinel}}
	if _, err := repo.Update(context.Background(), "u1", bson.M{"display_name": "Bob"}); !errors.Is(err, sentinel) {
		t.Fatalf("attendu sentinel, obtenu %v", err)
	}
}

func TestProfilRepository_Delete(t *testing.T) {
	repo := &ProfilRepository{collection: &fakeCollection{deleted: 1}}
	if err := repo.Delete(context.Background(), "u1"); err != nil {
		t.Fatalf("Delete erreur: %v", err)
	}
}

func TestProfilRepository_DeleteNoDocuments(t *testing.T) {
	repo := &ProfilRepository{collection: &fakeCollection{deleted: 0}}
	if err := repo.Delete(context.Background(), "u1"); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("attendu ErrNoDocuments, obtenu %v", err)
	}
}

func TestProfilRepository_DeleteError(t *testing.T) {
	sentinel := errors.New("delete")
	repo := &ProfilRepository{collection: &fakeCollection{err: sentinel}}
	if err := repo.Delete(context.Background(), "u1"); !errors.Is(err, sentinel) {
		t.Fatalf("attendu sentinel, obtenu %v", err)
	}
}
