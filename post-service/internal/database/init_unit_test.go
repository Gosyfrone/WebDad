package database

import (
	"context"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// fakeSchemaDB implémente schemaDatabase en mémoire : il permet de tester
// ensureSchema et ses étapes (collections, backfills, index) sans Mongo, en
// pilotant les erreurs et compteurs renvoyés. Même pattern que profil-service.
type fakeSchemaDB struct {
	collections        []string // collections déjà présentes (ListCollectionNames)
	listErr            error
	createErr          error
	runErr             error
	updateManyErr      error
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
	if f.db.updateManyErr != nil {
		return nil, f.db.updateManyErr
	}
	return &mongo.UpdateResult{ModifiedCount: f.db.modified}, nil
}

func (f *fakeSchemaCollection) Indexes() schemaIndexView {
	return fakeSchemaIndexView{db: f.db}
}

type fakeSchemaIndexView struct {
	db *fakeSchemaDB
}

func (f fakeSchemaIndexView) CreateMany(context.Context, []mongo.IndexModel, ...options.Lister[options.CreateIndexesOptions]) ([]string, error) {
	if f.db.createIndexesErr != nil {
		return nil, f.db.createIndexesErr
	}
	return []string{"idx"}, nil
}

// TestEnsureSchema_FakeSuccess : sur une base vierge, ensureSchema crée toutes
// les collections, applique les backfills (ModifiedCount > 0 → branche de log)
// et crée les index, sans erreur.
func TestEnsureSchema_FakeSuccess(t *testing.T) {
	db := &fakeSchemaDB{modified: 3}
	if err := ensureSchema(context.Background(), db); err != nil {
		t.Fatalf("ensureSchema erreur: %v", err)
	}
	if len(db.createdCollections) != len(collectionOrder) {
		t.Fatalf("collections créées = %d, attendu %d", len(db.createdCollections), len(collectionOrder))
	}
}

// TestEnsureSchema_FakeErrors : ensureSchema propage l'erreur de chaque étape
// (court-circuit) — ici l'échec de ListCollectionNames puis d'un backfill.
func TestEnsureSchema_FakeErrors(t *testing.T) {
	if err := ensureSchema(context.Background(), &fakeSchemaDB{listErr: errors.New("list")}); err == nil {
		t.Fatal("erreur ensureCollections attendue")
	}
	// Collections présentes (collMod ok) mais backfill en échec.
	if err := ensureSchema(context.Background(), &fakeSchemaDB{collections: collectionOrder, updateManyErr: errors.New("backfill")}); err == nil {
		t.Fatal("erreur backfill attendue")
	}
}

// TestEnsureSchema_WrapperRealAdapter : EnsureSchema (point d'entrée public)
// câble l'adaptateur réel ; avec un contexte annulé la première étape échoue,
// ce qui couvre le wrapper sans Mongo joignable.
func TestEnsureSchema_WrapperRealAdapter(t *testing.T) {
	client, err := mongo.Connect(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Fatalf("mongo.Connect: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := EnsureSchema(ctx, client.Database("post_wrapper_test")); err == nil {
		t.Fatal("EnsureSchema (ctx annulé) devrait échouer")
	}
}

// TestEnsureCollections_ExistingUpdatesValidator : collection déjà présente →
// pas de recréation, mais un collMod (RunCommand) pour resynchroniser le validateur.
func TestEnsureCollections_ExistingUpdatesValidator(t *testing.T) {
	db := &fakeSchemaDB{collections: collectionOrder}
	if err := ensureCollections(context.Background(), db); err != nil {
		t.Fatalf("ensureCollections erreur: %v", err)
	}
	if len(db.createdCollections) != 0 {
		t.Fatalf("collections existantes ne doivent pas être recréées: %#v", db.createdCollections)
	}
	if len(db.commands) != len(collectionOrder) {
		t.Fatalf("collMod attendu pour chaque collection, got=%d", len(db.commands))
	}
}

// TestEnsureCollections_Errors : chaque étape propage son erreur.
func TestEnsureCollections_Errors(t *testing.T) {
	if err := ensureCollections(context.Background(), &fakeSchemaDB{listErr: errors.New("list")}); err == nil {
		t.Fatal("erreur ListCollectionNames attendue")
	}
	if err := ensureCollections(context.Background(), &fakeSchemaDB{createErr: errors.New("create")}); err == nil {
		t.Fatal("erreur CreateCollection attendue")
	}
	if err := ensureCollections(context.Background(), &fakeSchemaDB{collections: collectionOrder, runErr: errors.New("collmod")}); err == nil {
		t.Fatal("erreur collMod attendue")
	}
}

// TestBackfills_Fake : succès (avec ModifiedCount) et propagation d'erreur des
// deux migrations idempotentes au boot.
func TestBackfills_Fake(t *testing.T) {
	if err := backfillReplyAudience(context.Background(), &fakeSchemaDB{modified: 1}); err != nil {
		t.Fatalf("backfillReplyAudience erreur: %v", err)
	}
	if err := backfillReplyAudience(context.Background(), &fakeSchemaDB{updateManyErr: errors.New("ra")}); err == nil {
		t.Fatal("erreur reply_audience attendue")
	}
	if err := backfillCommentLikesCount(context.Background(), &fakeSchemaDB{modified: 1}); err != nil {
		t.Fatalf("backfillCommentLikesCount erreur: %v", err)
	}
	if err := backfillCommentLikesCount(context.Background(), &fakeSchemaDB{updateManyErr: errors.New("cl")}); err == nil {
		t.Fatal("erreur comment likes_count attendue")
	}
}

// TestEnsureIndexes_Fake : succès et propagation d'erreur de CreateMany.
func TestEnsureIndexes_Fake(t *testing.T) {
	if err := ensureIndexes(context.Background(), &fakeSchemaDB{}); err != nil {
		t.Fatalf("ensureIndexes erreur: %v", err)
	}
	if err := ensureIndexes(context.Background(), &fakeSchemaDB{createIndexesErr: errors.New("index")}); err == nil {
		t.Fatal("erreur CreateMany attendue")
	}
}

// TestSchemaDefinitions : cohérence des maps statiques (validateurs + index pour
// chaque collection déclarée dans collectionOrder).
func TestSchemaDefinitions(t *testing.T) {
	for _, name := range collectionOrder {
		v, ok := validators[name]
		if !ok {
			t.Fatalf("validateur manquant pour %q", name)
		}
		js, ok := v["$jsonSchema"].(bson.M)
		if !ok {
			t.Fatalf("$jsonSchema invalide pour %q: %#v", name, v)
		}
		if req, ok := js["required"].(bson.A); !ok || len(req) == 0 {
			t.Fatalf("required vide pour %q", name)
		}
	}
}

// TestMongoSchemaAdapters : exerce les adaptateurs réels (mongoSchemaDatabase /
// mongoSchemaCollection) avec un contexte annulé → chaque délégation au driver
// renvoie une erreur, ce qui couvre les méthodes d'adaptation sans Mongo joignable.
func TestMongoSchemaAdapters(t *testing.T) {
	client, err := mongo.Connect(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Fatalf("mongo.Connect: %v", err)
	}
	db := mongoSchemaDatabase{db: client.Database("post_adapter_test")}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := db.ListCollectionNames(ctx, bson.M{}); err == nil {
		t.Fatal("ListCollectionNames (ctx annulé) devrait échouer")
	}
	if err := db.CreateCollection(ctx, "posts"); err == nil {
		t.Fatal("CreateCollection (ctx annulé) devrait échouer")
	}
	if _, err := db.RunCommand(ctx, bson.D{{Key: "ping", Value: 1}}).Raw(); err == nil {
		t.Fatal("RunCommand (ctx annulé) devrait échouer")
	}
	coll := db.Collection("posts")
	if _, err := coll.UpdateMany(ctx, bson.M{}, bson.M{"$set": bson.M{"x": 1}}); err == nil {
		t.Fatal("UpdateMany (ctx annulé) devrait échouer")
	}
	if coll.Indexes() == nil {
		t.Fatal("Indexes() nil")
	}
}
