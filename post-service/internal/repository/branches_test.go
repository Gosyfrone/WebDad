package repository

import (
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Branches LOGIQUES (non-erreur) restées non couvertes par les parcours nominaux
// des autres tests d'intégration : agrégation reposts du profil, pagination,
// filtres optionnels.

// TestRepo_ProfileHashtagReposts couvre toutes les branches de la boucle reposts
// de GetByProfileHashtag : id invalide, post introuvable, post masqué, hashtag
// non concordant, et les bornes de pagination (skip au-delà, clamp de fin).
func TestRepo_ProfileHashtagReposts(t *testing.T) {
	r, ctx := newRepo(t)

	// Post propre de l'auteur, taggé.
	mkPost(t, r, ctx, "auth", "mine #go", "go")

	// Post d'un autre, taggé, reposté par auth → doit remonter sur son profil.
	other := mkPost(t, r, ctx, "bob", "bob #go", "go")
	if _, _, err := r.AddRepost(ctx, other.ID.Hex(), "auth"); err != nil {
		t.Fatalf("AddRepost other: %v", err)
	}

	// Post masqué reposté → ignoré (branche IsHidden/AutoHidden).
	hidden := mkPost(t, r, ctx, "carol", "secret #go", "go")
	if _, err := r.Hide(ctx, hidden.ID, "mod", time.Now()); err != nil {
		t.Fatalf("Hide: %v", err)
	}
	if _, _, err := r.AddRepost(ctx, hidden.ID.Hex(), "auth"); err != nil {
		t.Fatalf("AddRepost hidden: %v", err)
	}

	// Post sans le hashtag, reposté → ignoré sous filtre "go" (branche mismatch).
	noTag := mkPost(t, r, ctx, "dave", "sans tag")
	if _, _, err := r.AddRepost(ctx, noTag.ID.Hex(), "auth"); err != nil {
		t.Fatalf("AddRepost noTag: %v", err)
	}

	// Repost d'un id NON hexadécimal → branche bson.ObjectIDFromHex en erreur.
	if _, _, err := r.AddRepost(ctx, "pas-un-objectid", "auth"); err != nil {
		t.Fatalf("AddRepost id invalide: %v", err)
	}
	// Repost d'un id valide mais inexistant → branche Get en erreur (continue).
	if _, _, err := r.AddRepost(ctx, bson.NewObjectID().Hex(), "auth"); err != nil {
		t.Fatalf("AddRepost id fantôme: %v", err)
	}

	// Sous filtre "go" : post propre + repost de "bob" = 2 ; tout le reste filtré.
	got, err := r.GetByProfileHashtag(ctx, "auth", "go", 10, 0)
	if err != nil || len(got) != 2 {
		t.Fatalf("GetByProfileHashtag filtré: %v / %d (attendu 2)", err, len(got))
	}

	// Pagination : skip au-delà du total → tranche vide (branche skip >= len).
	if page, err := r.GetByProfileHashtag(ctx, "auth", "go", 10, 100); err != nil || len(page) != 0 {
		t.Fatalf("skip au-delà → vide attendu, got %v / %d", err, len(page))
	}

	// Clamp de fin : limit énorme depuis skip=1 → reste 1 élément (branche end > len).
	if page, err := r.GetByProfileHashtag(ctx, "auth", "go", 1000, 1); err != nil || len(page) != 1 {
		t.Fatalf("clamp fin: %v / %d (attendu 1)", err, len(page))
	}
}

// TestRepo_LikedPostsEdges couvre les retours anticipés de LikedPostsByUser :
// aucun like (docs vides) et likes ne pointant que des ids non hexadécimaux.
func TestRepo_LikedPostsEdges(t *testing.T) {
	r, ctx := newRepo(t)

	if posts, err := r.LikedPostsByUser(ctx, "fantome", 10, 0); err != nil || len(posts) != 0 {
		t.Fatalf("aucun like → vide attendu, got %v / %d", err, len(posts))
	}

	// Like sur un id non hexadécimal → tous les oids échouent à parser → vide.
	if _, err := r.AddLike(ctx, "pas-un-objectid", "u9"); err != nil {
		t.Fatalf("AddLike id invalide: %v", err)
	}
	if posts, err := r.LikedPostsByUser(ctx, "u9", 10, 0); err != nil || len(posts) != 0 {
		t.Fatalf("likes tous invalides → vide attendu, got %v / %d", err, len(posts))
	}
}

// TestRepo_EmptyArgShortcuts couvre les retours anticipés sur argument vide
// (court-circuit avant tout accès Mongo) des méthodes qui en ont un.
func TestRepo_EmptyArgShortcuts(t *testing.T) {
	r, ctx := newRepo(t)

	if got, err := r.StatsByIDs(ctx, nil); err != nil || len(got) != 0 {
		t.Fatalf("StatsByIDs([]) → vide attendu, got %v / %d", err, len(got))
	}
	if got, err := r.CommentStatsByIDs(ctx, nil); err != nil || len(got) != 0 {
		t.Fatalf("CommentStatsByIDs([]) → vide attendu, got %v / %d", err, len(got))
	}
	if got, err := r.LikedCommentIDsByUser(ctx, "", nil); err != nil || len(got) != 0 {
		t.Fatalf("LikedCommentIDsByUser(\"\",[]) → vide attendu, got %v / %d", err, len(got))
	}
	if err := r.DeleteCommentLikesByComments(ctx, nil); err != nil {
		t.Fatalf("DeleteCommentLikesByComments([]) → nil attendu, got %v", err)
	}
}

// TestRepo_DeleteNotFound couvre la branche « rien supprimé → ErrNoDocuments »
// de DeleteCollection et DeleteComment (id inexistant).
func TestRepo_DeleteNotFound(t *testing.T) {
	r, ctx := newRepo(t)

	if err := r.DeleteCollection(ctx, bson.NewObjectID()); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("DeleteCollection inexistante → ErrNoDocuments, got %v", err)
	}
	if err := r.DeleteComment(ctx, bson.NewObjectID()); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("DeleteComment inexistant → ErrNoDocuments, got %v", err)
	}
}

// TestRepo_ModerationBranches couvre les branches non prises par le cycle de vie
// nominal : SetNsfw(false) (effacement), et les bornes Since/Until de ListHidden.
func TestRepo_ModerationBranches(t *testing.T) {
	r, ctx := newRepo(t)
	p := mkPost(t, r, ctx, "a1", "modéré")
	now := time.Now()

	if _, err := r.SetNsfw(ctx, p.ID, true, "mod", now); err != nil {
		t.Fatalf("SetNsfw(true): %v", err)
	}
	if up, err := r.SetNsfw(ctx, p.ID, false, "mod", now); err != nil || up.Nsfw {
		t.Fatalf("SetNsfw(false): %v / %+v", err, up)
	}

	if _, err := r.Hide(ctx, p.ID, "mod", now); err != nil {
		t.Fatalf("Hide: %v", err)
	}
	since := now.Add(-time.Hour)
	until := now.Add(time.Hour)
	if hidden, err := r.ListHidden(ctx, HiddenFilter{Since: &since, Until: &until}, 10, 0); err != nil || len(hidden) != 1 {
		t.Fatalf("ListHidden borné Since/Until: %v / %d", err, len(hidden))
	}
}
