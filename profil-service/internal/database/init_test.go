package database

import (
	"context"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type fakeSchemaDB struct {
	collections        []string
	listErr            error
	createErr          error
	runErr             error
	updateManyErr      error
	updateManyErrAt    int
	updateManyCalls    int
	updateOneErr       error
	dropErr            error
	createIndexesErr   error
	modified           int64
	createdCollections []string
	commands           []any
	requested          []string
}

func (f *fakeSchemaDB) ListCollectionNames(context.Context, any, ...options.Lister[options.ListCollectionsOptions]) ([]string, error) {
	return f.collections, f.listErr
}

func (f *fakeSchemaDB) CreateCollection(_ context.Context, name string, _ ...options.Lister[options.CreateCollectionOptions]) error {
	f.createdCollections = append(f.createdCollections, name)
	return f.createErr
}

func (f *fakeSchemaDB) RunCommand(_ context.Context, command any, _ ...options.Lister[options.RunCmdOptions]) rawResult {
	f.commands = append(f.commands, command)
	return fakeRawResult{err: f.runErr}
}

func (f *fakeSchemaDB) Collection(name string, _ ...options.Lister[options.CollectionOptions]) schemaCollection {
	f.requested = append(f.requested, name)
	return &fakeSchemaCollection{db: f}
}

type fakeRawResult struct {
	err error
}

func (f fakeRawResult) Raw() (bson.Raw, error) {
	return nil, f.err
}

type fakeSchemaCollection struct {
	db *fakeSchemaDB
}

func (f *fakeSchemaCollection) UpdateMany(context.Context, any, any, ...options.Lister[options.UpdateManyOptions]) (*mongo.UpdateResult, error) {
	f.db.updateManyCalls++
	if f.db.updateManyErr != nil && (f.db.updateManyErrAt == 0 || f.db.updateManyErrAt == f.db.updateManyCalls) {
		return nil, f.db.updateManyErr
	}
	return &mongo.UpdateResult{ModifiedCount: f.db.modified}, nil
}

func (f *fakeSchemaCollection) UpdateOne(context.Context, any, any, ...options.Lister[options.UpdateOneOptions]) (*mongo.UpdateResult, error) {
	if f.db.updateOneErr != nil {
		return nil, f.db.updateOneErr
	}
	return &mongo.UpdateResult{ModifiedCount: f.db.modified}, nil
}

func (f *fakeSchemaCollection) Indexes() schemaIndexView {
	return fakeSchemaIndexView{db: f.db}
}

type fakeSchemaIndexView struct {
	db *fakeSchemaDB
}

func (f fakeSchemaIndexView) DropOne(context.Context, string, ...options.Lister[options.DropIndexesOptions]) error {
	return f.db.dropErr
}

func (f fakeSchemaIndexView) CreateMany(context.Context, []mongo.IndexModel, ...options.Lister[options.CreateIndexesOptions]) ([]string, error) {
	if f.db.createIndexesErr != nil {
		return nil, f.db.createIndexesErr
	}
	return []string{"user_id_1"}, nil
}

func TestIsIndexNotFound(t *testing.T) {
	if !isIndexNotFound(mongo.CommandError{Code: 27}) {
		t.Fatal("code Mongo 27 doit etre reconnu comme index not found")
	}
	if isIndexNotFound(mongo.CommandError{Code: 26}) {
		t.Fatal("un autre code Mongo ne doit pas etre reconnu")
	}
	if isIndexNotFound(errors.New("boom")) {
		t.Fatal("une erreur generique ne doit pas etre reconnue")
	}
}

func TestSchemaDefinitions(t *testing.T) {
	profileValidator, ok := validators["profiles"]
	if !ok {
		t.Fatal("validateur profiles manquant")
	}
	jsonSchema, ok := profileValidator["$jsonSchema"].(bson.M)
	if !ok {
		t.Fatalf("schema profiles invalide: %#v", profileValidator)
	}
	required, ok := jsonSchema["required"].(bson.A)
	if !ok || len(required) == 0 {
		t.Fatalf("required profiles invalide: %#v", jsonSchema["required"])
	}
	if len(indexes["profiles"]) != 1 {
		t.Fatalf("indexes profiles = %d, attendu 1", len(indexes["profiles"]))
	}
	if len(collectionOrder) != 1 || collectionOrder[0] != "profiles" {
		t.Fatalf("collectionOrder inattendu: %#v", collectionOrder)
	}
}

func TestMongoSchemaAdapters(t *testing.T) {
	client, err := mongo.Connect(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Fatalf("mongo.Connect: %v", err)
	}
	db := mongoSchemaDatabase{db: client.Database("profil_adapter_test")}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := db.ListCollectionNames(ctx, bson.M{}); err == nil {
		t.Fatal("ListCollectionNames avec contexte annule devrait echouer")
	}
	if err := db.CreateCollection(ctx, "profiles"); err == nil {
		t.Fatal("CreateCollection avec contexte annule devrait echouer")
	}
	if _, err := db.RunCommand(ctx, bson.D{{Key: "ping", Value: 1}}).Raw(); err == nil {
		t.Fatal("RunCommand avec contexte annule devrait echouer")
	}

	coll := db.Collection("profiles")
	if _, err := coll.UpdateMany(ctx, bson.M{}, bson.M{"$set": bson.M{"x": 1}}); err == nil {
		t.Fatal("UpdateMany avec contexte annule devrait echouer")
	}
	if _, err := coll.UpdateOne(ctx, bson.M{}, bson.M{"$set": bson.M{"x": 1}}); err == nil {
		t.Fatal("UpdateOne avec contexte annule devrait echouer")
	}
	if coll.Indexes() == nil {
		t.Fatal("Indexes nil")
	}
}

func TestEnsureSchemaSuccessCreatesMissingCollection(t *testing.T) {
	db := &fakeSchemaDB{modified: 2}
	if err := ensureSchema(context.Background(), db); err != nil {
		t.Fatalf("ensureSchema erreur: %v", err)
	}
	if len(db.createdCollections) != 1 || db.createdCollections[0] != "profiles" {
		t.Fatalf("collections creees inattendues: %#v", db.createdCollections)
	}
	if len(db.requested) == 0 {
		t.Fatal("aucune collection demandee")
	}
}

func TestEnsureSchema_StopsAtEachFailedStage(t *testing.T) {
	tests := []struct {
		name string
		db   *fakeSchemaDB
	}{
		{"collections", &fakeSchemaDB{listErr: errors.New("list")}},
		{"legacy indexes", &fakeSchemaDB{dropErr: errors.New("drop")}},
		{"indexes", &fakeSchemaDB{createIndexesErr: errors.New("indexes")}},
		{"visibility", &fakeSchemaDB{updateManyErr: errors.New("visibility"), updateManyErrAt: 1}},
		{"certification", &fakeSchemaDB{updateManyErr: errors.New("certification"), updateManyErrAt: 4}},
		{"birth date", &fakeSchemaDB{updateManyErr: errors.New("birth"), updateManyErrAt: 5}},
		{"seed", &fakeSchemaDB{updateOneErr: errors.New("seed")}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := ensureSchema(context.Background(), tc.db); err == nil {
				t.Fatal("une erreur était attendue")
			}
		})
	}
}

func TestEnsureCollectionsExistingUpdatesValidator(t *testing.T) {
	db := &fakeSchemaDB{collections: []string{"profiles"}}
	if err := ensureCollections(context.Background(), db); err != nil {
		t.Fatalf("ensureCollections erreur: %v", err)
	}
	if len(db.createdCollections) != 0 {
		t.Fatalf("collection existante ne doit pas etre recreee: %#v", db.createdCollections)
	}
	if len(db.commands) != 1 {
		t.Fatalf("collMod attendu, commandes=%#v", db.commands)
	}
}

func TestEnsureCollectionsErrors(t *testing.T) {
	if err := ensureCollections(context.Background(), &fakeSchemaDB{listErr: errors.New("list")}); err == nil {
		t.Fatal("erreur ListCollectionNames attendue")
	}
	if err := ensureCollections(context.Background(), &fakeSchemaDB{createErr: errors.New("create")}); err == nil {
		t.Fatal("erreur CreateCollection attendue")
	}
	if err := ensureCollections(context.Background(), &fakeSchemaDB{collections: []string{"profiles"}, runErr: errors.New("collmod")}); err == nil {
		t.Fatal("erreur collMod attendue")
	}
}

func TestBackfills(t *testing.T) {
	if err := backfillBirthDate(context.Background(), &fakeSchemaDB{modified: 1}); err != nil {
		t.Fatalf("backfillBirthDate erreur: %v", err)
	}
	if err := backfillBirthDate(context.Background(), &fakeSchemaDB{updateManyErr: errors.New("birth")}); err == nil {
		t.Fatal("erreur birth_date attendue")
	}

	if err := backfillVisibility(context.Background(), &fakeSchemaDB{modified: 1}); err != nil {
		t.Fatalf("backfillVisibility erreur: %v", err)
	}
	if err := backfillVisibility(context.Background(), &fakeSchemaDB{updateManyErr: errors.New("visibility")}); err == nil {
		t.Fatal("erreur visibility attendue")
	}
}

func TestDropLegacyIndexes(t *testing.T) {
	if err := dropLegacyIndexes(context.Background(), &fakeSchemaDB{}); err != nil {
		t.Fatalf("dropLegacyIndexes erreur: %v", err)
	}
	if err := dropLegacyIndexes(context.Background(), &fakeSchemaDB{dropErr: mongo.CommandError{Code: 27}}); err != nil {
		t.Fatalf("index absent doit etre ignore: %v", err)
	}
	if err := dropLegacyIndexes(context.Background(), &fakeSchemaDB{dropErr: errors.New("drop")}); err == nil {
		t.Fatal("erreur drop attendue")
	}
}

func TestEnsureIndexes(t *testing.T) {
	if err := ensureIndexes(context.Background(), &fakeSchemaDB{}); err != nil {
		t.Fatalf("ensureIndexes erreur: %v", err)
	}
	if err := ensureIndexes(context.Background(), &fakeSchemaDB{createIndexesErr: errors.New("index")}); err == nil {
		t.Fatal("erreur index attendue")
	}
}

func TestEnsureSeed(t *testing.T) {
	if err := ensureSeed(context.Background(), &fakeSchemaDB{}); err != nil {
		t.Fatalf("ensureSeed erreur: %v", err)
	}
	if err := ensureSeed(context.Background(), &fakeSchemaDB{updateOneErr: errors.New("seed")}); err == nil {
		t.Fatal("erreur seed attendue")
	}
}
