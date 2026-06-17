// Package repository : accès aux données Mongo du post-service. Les méthodes
// renvoient les erreurs brutes du driver (notamment mongo.ErrNoDocuments) ; la
// traduction en erreurs métier est faite par la couche service.
package repository

import (
	"context"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/webdad/post-service/internal/models"
)

// PostRepository agrège les collections du domaine post (posts + likes +
// comments). Les likes/comments référencent leur post par l'identifiant
// hexadécimal (`post_id`, string) ; les compteurs dénormalisés vivent sur le
// document `posts` et sont maintenus par `$inc`.
type PostRepository struct {
	posts        *mongo.Collection
	likes        *mongo.Collection
	comments     *mongo.Collection
	commentLikes *mongo.Collection
	reposts      *mongo.Collection
	pollVotes    *mongo.Collection
	// Signets : collections + appartenances + préférences de rafale.
	bookmarkCollections *mongo.Collection
	bookmarks           *mongo.Collection
	bookmarkPrefs       *mongo.Collection
}

func NewPostRepository(db *mongo.Database) *PostRepository {
	return &PostRepository{
		posts:               db.Collection("posts"),
		likes:               db.Collection("likes"),
		comments:            db.Collection("comments"),
		commentLikes:        db.Collection("comment_likes"),
		reposts:             db.Collection("reposts"),
		pollVotes:           db.Collection("poll_votes"),
		bookmarkCollections: db.Collection("bookmark_collections"),
		bookmarks:           db.Collection("bookmarks"),
		bookmarkPrefs:       db.Collection("bookmark_prefs"),
	}
}

// --- Posts -------------------------------------------------------------------

// notHidden renvoie la condition « post non masqué par la modération »
// (is_hidden absent ou false). Fusionnée dans tous les filtres de lecture
// publique pour que les posts retirés en suppression douce sortent des fils
// sans être effacés. Une nouvelle map à chaque appel (pas d'aliasing).
func notHidden() bson.M { return bson.M{"is_hidden": bson.M{"$ne": true}} }

func withHashtag(filter bson.M, hashtag string) bson.M {
	if hashtag != "" {
		filter["hashtags"] = hashtag
	}
	return filter
}

func withAnyHashtag(filter bson.M) bson.M {
	filter["hashtags"] = bson.M{"$exists": true, "$ne": bson.A{}}
	return filter
}

// Create insère le post et renseigne post.ID avec l'identifiant généré.
func (r *PostRepository) Create(ctx context.Context, post *models.Post) error {
	res, err := r.posts.InsertOne(ctx, post)
	if err != nil {
		return err
	}
	if oid, ok := res.InsertedID.(bson.ObjectID); ok {
		post.ID = oid
	}
	return nil
}

// GetAll renvoie le fil global trié du plus récent au plus ancien, paginé.
// Les posts masqués par la modération sont exclus.
func (r *PostRepository) GetAll(ctx context.Context, limit, skip int64) ([]models.Post, error) {
	return r.find(ctx, notHidden(), limit, skip)
}

// GetAllByHashtag renvoie le fil global limité aux posts contenant ce hashtag.
func (r *PostRepository) GetAllByHashtag(ctx context.Context, hashtag, sortMode string, limit, skip int64) ([]models.Post, error) {
	return r.findSorted(ctx, withHashtag(notHidden(), hashtag), sortMode, limit, skip)
}

// GetAllWithHashtags renvoie le fil global limité aux posts contenant au moins
// un hashtag.
func (r *PostRepository) GetAllWithHashtags(ctx context.Context, limit, skip int64) ([]models.Post, error) {
	return r.find(ctx, withAnyHashtag(notHidden()), limit, skip)
}

// GetByProfile renvoie les posts d'un auteur, triés du plus récent au plus ancien.
func (r *PostRepository) GetByProfile(ctx context.Context, authorID string, limit, skip int64) ([]models.Post, error) {
	return r.GetByProfileHashtag(ctx, authorID, "", limit, skip)
}

// GetByProfileHashtag renvoie les posts d'un auteur, éventuellement filtrés par hashtag.
func (r *PostRepository) GetByProfileHashtag(ctx context.Context, authorID, hashtag string, limit, skip int64) ([]models.Post, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "pinned_at", Value: -1}, {Key: "created_at", Value: -1}}).
		SetLimit(limit + skip)

	filter := withHashtag(notHidden(), hashtag)
	filter["author_id"] = authorID
	cursor, err := r.posts.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	posts := []models.Post{}
	if err := cursor.All(ctx, &posts); err != nil {
		return nil, err
	}

	reposts, err := r.ListRepostsByUser(ctx, authorID, limit+skip, 0)
	if err != nil {
		return nil, err
	}
	for _, repost := range reposts {
		oid, err := bson.ObjectIDFromHex(repost.PostID)
		if err != nil {
			continue
		}
		post, err := r.Get(ctx, oid)
		if err != nil {
			continue
		}
		// Un repost pointant un post masqué par la modération ne réapparaît pas
		// par la bande sur le profil de celui qui l'a reposté.
		if post.IsHidden {
			continue
		}
		if hashtag != "" && !postHasHashtag(post, hashtag) {
			continue
		}
		post.RepostedByID = repost.UserID
		post.RepostedAt = &repost.CreatedAt
		posts = append(posts, *post)
	}

	sortProfilePosts(posts)
	if skip >= int64(len(posts)) {
		return []models.Post{}, nil
	}
	end := skip + limit
	if end > int64(len(posts)) {
		end = int64(len(posts))
	}
	return posts[skip:end], nil
}

// GetByAuthors renvoie les posts d'un ensemble d'auteurs (fil « Abonnements »),
// triés du plus récent au plus ancien. Une seule requête indexée (`$in` sur
// author_id) : la sélection est faite côté DB, pas côté client.
func (r *PostRepository) GetByAuthors(ctx context.Context, authorIDs []string, limit, skip int64) ([]models.Post, error) {
	return r.GetByAuthorsHashtag(ctx, authorIDs, "", "", limit, skip)
}

// GetByAuthorsHashtag renvoie les posts d'auteurs donnés, éventuellement filtrés par hashtag.
func (r *PostRepository) GetByAuthorsHashtag(ctx context.Context, authorIDs []string, hashtag, sortMode string, limit, skip int64) ([]models.Post, error) {
	filter := withHashtag(notHidden(), hashtag)
	filter["author_id"] = bson.M{"$in": authorIDs}
	return r.findSorted(ctx, filter, sortMode, limit, skip)
}

// ListTopHashtags agrège les hashtags les plus présents dans les posts non masqués.
func (r *PostRepository) ListTopHashtags(ctx context.Context, limit int64) ([]models.HashtagTrend, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"is_hidden": bson.M{"$ne": true},
			"hashtags":  bson.M{"$exists": true, "$ne": bson.A{}},
		}}},
		{{Key: "$unwind", Value: "$hashtags"}},
		{{Key: "$group", Value: bson.M{"_id": "$hashtags", "count": bson.M{"$sum": 1}}}},
		{{Key: "$sort", Value: bson.D{{Key: "count", Value: -1}, {Key: "_id", Value: 1}}}},
		{{Key: "$limit", Value: limit}},
	}

	cursor, err := r.posts.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var rows []struct {
		Tag   string `bson:"_id"`
		Count int64  `bson:"count"`
	}
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, err
	}
	trends := make([]models.HashtagTrend, 0, len(rows))
	for _, row := range rows {
		trends = append(trends, models.HashtagTrend{Tag: row.Tag, Count: row.Count})
	}
	return trends, nil
}

// StatsByIDs renvoie les posts non masqués dont l'_id figure dans oids, en
// projection LÉGÈRE : auteur (pour la barrière de visibilité côté service) +
// compteurs dénormalisés, sans contenu ni médias. Sert au rafraîchissement
// périodique des compteurs côté front. L'ordre n'est pas garanti (le front
// indexe par id).
func (r *PostRepository) StatsByIDs(ctx context.Context, oids []bson.ObjectID) ([]models.Post, error) {
	if len(oids) == 0 {
		return []models.Post{}, nil
	}
	filter := bson.M{"_id": bson.M{"$in": oids}, "is_hidden": bson.M{"$ne": true}}
	opts := options.Find().SetProjection(bson.M{
		"author_id":      1,
		"likes_count":    1,
		"comments_count": 1,
		"reposts_count":  1,
	})
	cursor, err := r.posts.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	posts := []models.Post{}
	if err := cursor.All(ctx, &posts); err != nil {
		return nil, err
	}
	return posts, nil
}

// find factorise la lecture paginée + triée des posts.
func (r *PostRepository) find(ctx context.Context, filter bson.M, limit, skip int64) ([]models.Post, error) {
	return r.findSorted(ctx, filter, "", limit, skip)
}

func (r *PostRepository) findSorted(ctx context.Context, filter bson.M, sortMode string, limit, skip int64) ([]models.Post, error) {
	opts := options.Find().
		SetSort(postSort(sortMode)).
		SetLimit(limit).
		SetSkip(skip)

	cursor, err := r.posts.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	posts := []models.Post{}
	if err := cursor.All(ctx, &posts); err != nil {
		return nil, err
	}
	return posts, nil
}

func postSort(sortMode string) bson.D {
	if sortMode == "top" {
		return bson.D{
			{Key: "likes_count", Value: -1},
			{Key: "reposts_count", Value: -1},
			{Key: "comments_count", Value: -1},
			{Key: "created_at", Value: -1},
		}
	}
	return bson.D{{Key: "created_at", Value: -1}}
}

// Get renvoie un post par son ObjectID (mongo.ErrNoDocuments si absent).
func (r *PostRepository) Get(ctx context.Context, id bson.ObjectID) (*models.Post, error) {
	var post models.Post
	if err := r.posts.FindOne(ctx, bson.M{"_id": id}).Decode(&post); err != nil {
		return nil, err
	}
	return &post, nil
}

// Delete supprime un post (mongo.ErrNoDocuments si rien n'a été supprimé).
func (r *PostRepository) Delete(ctx context.Context, id bson.ObjectID) error {
	res, err := r.posts.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// Hide masque un post (suppression douce de modération) : il sort des fils
// publics mais reste en base, restaurable. Pose is_hidden/hidden_by/hidden_at
// et renvoie le document à jour (mongo.ErrNoDocuments si absent).
func (r *PostRepository) Hide(ctx context.Context, id bson.ObjectID, byUserID string, at time.Time) (*models.Post, error) {
	update := bson.M{
		"$set": bson.M{
			"is_hidden":  true,
			"hidden_by":  byUserID,
			"hidden_at":  at,
			"updated_at": at,
		},
	}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var post models.Post
	if err := r.posts.FindOneAndUpdate(ctx, bson.M{"_id": id}, update, opts).Decode(&post); err != nil {
		return nil, err
	}
	return &post, nil
}

// RestoreHidden lève le masquage d'un post (retour dans les fils publics) et
// efface les métadonnées de modération. Renvoie le document à jour.
func (r *PostRepository) RestoreHidden(ctx context.Context, id bson.ObjectID) (*models.Post, error) {
	update := bson.M{
		"$set":   bson.M{"is_hidden": false, "updated_at": time.Now()},
		"$unset": bson.M{"hidden_by": "", "hidden_at": ""},
	}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var post models.Post
	if err := r.posts.FindOneAndUpdate(ctx, bson.M{"_id": id}, update, opts).Decode(&post); err != nil {
		return nil, err
	}
	return &post, nil
}

// ListHidden renvoie les posts masqués (corbeille de modération, partagée
// mod/admin), du plus récemment masqué au plus ancien, paginés.
func (r *PostRepository) ListHidden(ctx context.Context, limit, skip int64) ([]models.Post, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "hidden_at", Value: -1}}).
		SetLimit(limit).
		SetSkip(skip)

	cursor, err := r.posts.Find(ctx, bson.M{"is_hidden": true}, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	posts := []models.Post{}
	if err := cursor.All(ctx, &posts); err != nil {
		return nil, err
	}
	return posts, nil
}

// ListPurgeable renvoie les posts masqués dont le masquage date d'avant
// `before` (= now − rétention) → candidats à la purge RGPD définitive.
func (r *PostRepository) ListPurgeable(ctx context.Context, before time.Time, limit int64) ([]models.Post, error) {
	filter := bson.M{"is_hidden": true, "hidden_at": bson.M{"$lt": before}}
	cursor, err := r.posts.Find(ctx, filter, options.Find().SetLimit(limit))
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()
	posts := []models.Post{}
	if err := cursor.All(ctx, &posts); err != nil {
		return nil, err
	}
	return posts, nil
}

// ListPurgeWarnable renvoie les posts masqués entrés dans la fenêtre de préavis
// (`hidden_at < before` = now − (rétention − préavis)) et pas encore prévenus
// (`purge_warned_at` absent) → à notifier une fois.
func (r *PostRepository) ListPurgeWarnable(ctx context.Context, before time.Time, limit int64) ([]models.Post, error) {
	filter := bson.M{
		"is_hidden":       true,
		"hidden_at":       bson.M{"$lt": before},
		"purge_warned_at": bson.M{"$exists": false},
	}
	cursor, err := r.posts.Find(ctx, filter, options.Find().SetLimit(limit))
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()
	posts := []models.Post{}
	if err := cursor.All(ctx, &posts); err != nil {
		return nil, err
	}
	return posts, nil
}

// MarkPurgeWarned pose la date d'envoi du préavis de purge (idempotence du
// balayage : on ne re-notifie pas un post déjà prévenu).
func (r *PostRepository) MarkPurgeWarned(ctx context.Context, id bson.ObjectID, at time.Time) error {
	_, err := r.posts.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"purge_warned_at": at}})
	return err
}

// PurgeByAuthor efface DÉFINITIVEMENT toutes les données d'un utilisateur
// (effacement RGPD) : ses posts, ses commentaires, ses likes, ses reposts et
// ses signets (collections + appartenances + préférences). Best-effort sur
// chaque collection ; renvoie le nombre de posts supprimés. Ne réajuste pas les
// compteurs des posts d'AUTRES utilisateurs (compte effacé → dérive cosmétique
// acceptable).
func (r *PostRepository) PurgeByAuthor(ctx context.Context, userID string) (int64, error) {
	res, err := r.posts.DeleteMany(ctx, bson.M{"author_id": userID})
	var postsDeleted int64
	if res != nil {
		postsDeleted = res.DeletedCount
	}
	_, _ = r.comments.DeleteMany(ctx, bson.M{"author_id": userID})
	_, _ = r.likes.DeleteMany(ctx, bson.M{"user_id": userID})
	_, _ = r.commentLikes.DeleteMany(ctx, bson.M{"user_id": userID})
	_, _ = r.reposts.DeleteMany(ctx, bson.M{"user_id": userID})
	_, _ = r.bookmarks.DeleteMany(ctx, bson.M{"user_id": userID})
	_, _ = r.bookmarkCollections.DeleteMany(ctx, bson.M{"user_id": userID})
	_, _ = r.bookmarkPrefs.DeleteMany(ctx, bson.M{"user_id": userID})
	return postsDeleted, err
}

// Update modifie le contenu et renvoie le document à jour (ReturnDocument
// After). mongo.ErrNoDocuments si le post n'existe pas.
func (r *PostRepository) Update(ctx context.Context, id bson.ObjectID, content string, hashtags []string) (*models.Post, error) {
	update := bson.M{
		"$set": bson.M{
			"content":    content,
			"hashtags":   hashtags,
			"updated_at": time.Now(),
		},
	}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var post models.Post
	if err := r.posts.FindOneAndUpdate(ctx, bson.M{"_id": id}, update, opts).Decode(&post); err != nil {
		return nil, err
	}
	return &post, nil
}

// UnpinByAuthor retire l'épinglage de tous les posts d'un auteur. Appelé avant
// Pin pour garantir un seul post épinglé par profil.
func (r *PostRepository) UnpinByAuthor(ctx context.Context, authorID string) error {
	_, err := r.posts.UpdateMany(
		ctx,
		bson.M{"author_id": authorID, "pinned_at": bson.M{"$exists": true}},
		bson.M{"$unset": bson.M{"pinned_at": ""}, "$set": bson.M{"updated_at": time.Now()}},
	)
	return err
}

// Pin pose pinned_at sur un post et renvoie le document à jour.
func (r *PostRepository) Pin(ctx context.Context, id bson.ObjectID, pinnedAt time.Time) (*models.Post, error) {
	update := bson.M{
		"$set": bson.M{
			"pinned_at":  pinnedAt,
			"updated_at": pinnedAt,
		},
	}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var post models.Post
	if err := r.posts.FindOneAndUpdate(ctx, bson.M{"_id": id}, update, opts).Decode(&post); err != nil {
		return nil, err
	}
	return &post, nil
}

// Unpin retire pinned_at d'un post et renvoie le document à jour.
func (r *PostRepository) Unpin(ctx context.Context, id bson.ObjectID) (*models.Post, error) {
	update := bson.M{
		"$unset": bson.M{"pinned_at": ""},
		"$set":   bson.M{"updated_at": time.Now()},
	}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var post models.Post
	if err := r.posts.FindOneAndUpdate(ctx, bson.M{"_id": id}, update, opts).Decode(&post); err != nil {
		return nil, err
	}
	return &post, nil
}

// IncCounter applique `$inc` sur un compteur dénormalisé (likes_count /
// comments_count) et renvoie le post à jour. Le compteur reste `int32` pour
// rester conforme au validateur $jsonSchema (bsonType "int").
func (r *PostRepository) IncCounter(ctx context.Context, id bson.ObjectID, field string, delta int32) (*models.Post, error) {
	update := bson.M{"$inc": bson.M{field: delta}}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var post models.Post
	if err := r.posts.FindOneAndUpdate(ctx, bson.M{"_id": id}, update, opts).Decode(&post); err != nil {
		return nil, err
	}
	return &post, nil
}

func (r *PostRepository) AddPollVote(ctx context.Context, postID, userID, choiceID string) (bool, error) {
	_, err := r.pollVotes.InsertOne(ctx, &models.PollVote{
		PostID:    postID,
		UserID:    userID,
		ChoiceID:  choiceID,
		CreatedAt: time.Now(),
	})
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *PostRepository) PollVoteChoice(ctx context.Context, postID, userID string) (string, error) {
	var vote models.PollVote
	if err := r.pollVotes.FindOne(ctx, bson.M{"post_id": postID, "user_id": userID}).Decode(&vote); err != nil {
		return "", err
	}
	return vote.ChoiceID, nil
}

func (r *PostRepository) IncPollChoice(ctx context.Context, id bson.ObjectID, choiceID string) (*models.Post, error) {
	update := bson.M{
		"$inc": bson.M{
			"poll.total_votes":           1,
			"poll.choices.$.votes_count": 1,
		},
		"$set": bson.M{"updated_at": time.Now()},
	}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var post models.Post
	if err := r.posts.FindOneAndUpdate(ctx, bson.M{"_id": id, "poll.choices.id": choiceID}, update, opts).Decode(&post); err != nil {
		return nil, err
	}
	return &post, nil
}

func (r *PostRepository) ClosePoll(ctx context.Context, id bson.ObjectID, at time.Time) (*models.Post, error) {
	update := bson.M{
		"$set": bson.M{
			"poll.closed_at": at,
			"updated_at":     at,
		},
	}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var post models.Post
	if err := r.posts.FindOneAndUpdate(ctx, bson.M{"_id": id, "poll": bson.M{"$exists": true}}, update, opts).Decode(&post); err != nil {
		return nil, err
	}
	return &post, nil
}

// --- Likes -------------------------------------------------------------------

// AddLike enregistre un like (idempotent grâce à l'index unique
// post_id+user_id). Renvoie true si le like a été créé, false s'il existait
// déjà (clé dupliquée) → le service n'incrémente le compteur que dans le 1er cas.
func (r *PostRepository) AddLike(ctx context.Context, postID, userID string) (bool, error) {
	_, err := r.likes.InsertOne(ctx, bson.M{
		"post_id":    postID,
		"user_id":    userID,
		"created_at": time.Now(),
	})
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// RemoveLike supprime un like. Renvoie true si un like a effectivement été
// supprimé (sinon le service n'a pas à décrémenter).
func (r *PostRepository) RemoveLike(ctx context.Context, postID, userID string) (bool, error) {
	res, err := r.likes.DeleteOne(ctx, bson.M{"post_id": postID, "user_id": userID})
	if err != nil {
		return false, err
	}
	return res.DeletedCount > 0, nil
}

// LikedPostIDs renvoie les ids des posts likés par un utilisateur (sert à
// initialiser l'état des cœurs côté front, façon getFollowingIds).
func (r *PostRepository) LikedPostIDs(ctx context.Context, userID string) ([]string, error) {
	return r.distinctStrings(ctx, r.likes, bson.M{"user_id": userID}, "post_id")
}

// LikersByPost renvoie les ids des utilisateurs ayant liké un post.
func (r *PostRepository) LikersByPost(ctx context.Context, postID string) ([]string, error) {
	return r.distinctStrings(ctx, r.likes, bson.M{"post_id": postID}, "user_id")
}

// LikedPostsByUser retourne les posts likés par userID, dans l'ordre du like le
// plus récent en premier, paginés (limit/offset). Les posts masqués (is_hidden)
// sont exclus. L'ordre de like est préservé.
func (r *PostRepository) LikedPostsByUser(ctx context.Context, userID string, limit, offset int64) ([]*models.Post, error) {
	type likeDoc struct {
		PostID string `bson:"post_id"`
	}

	cur, err := r.likes.Find(ctx, bson.M{"user_id": userID},
		options.Find().
			SetSort(bson.D{{Key: "created_at", Value: -1}}).
			SetLimit(limit).
			SetSkip(offset),
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cur.Close(ctx) }()

	var docs []likeDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return []*models.Post{}, nil
	}

	oids := make([]bson.ObjectID, 0, len(docs))
	orderByHex := make(map[string]int, len(docs))
	for i, d := range docs {
		oid, err := bson.ObjectIDFromHex(d.PostID)
		if err != nil {
			continue
		}
		oids = append(oids, oid)
		orderByHex[d.PostID] = i
	}
	if len(oids) == 0 {
		return []*models.Post{}, nil
	}

	postCur, err := r.posts.Find(ctx, bson.M{
		"_id":       bson.M{"$in": oids},
		"is_hidden": bson.M{"$ne": true},
	})
	if err != nil {
		return nil, err
	}
	defer func() { _ = postCur.Close(ctx) }()

	var posts []*models.Post
	if err := postCur.All(ctx, &posts); err != nil {
		return nil, err
	}

	sort.Slice(posts, func(i, j int) bool {
		return orderByHex[posts[i].ID.Hex()] < orderByHex[posts[j].ID.Hex()]
	})
	return posts, nil
}

// DeleteLikesByPost purge les likes d'un post (nettoyage à la suppression).
func (r *PostRepository) DeleteLikesByPost(ctx context.Context, postID string) error {
	_, err := r.likes.DeleteMany(ctx, bson.M{"post_id": postID})
	return err
}

// --- Reposts -----------------------------------------------------------------

func (r *PostRepository) AddRepost(ctx context.Context, postID, userID string) (*models.Repost, bool, error) {
	repost := &models.Repost{
		PostID:    postID,
		UserID:    userID,
		CreatedAt: time.Now(),
	}
	res, err := r.reposts.InsertOne(ctx, repost)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			existing, getErr := r.GetRepost(ctx, postID, userID)
			return existing, false, getErr
		}
		return nil, false, err
	}
	if oid, ok := res.InsertedID.(bson.ObjectID); ok {
		repost.ID = oid
	}
	return repost, true, nil
}

func (r *PostRepository) GetRepost(ctx context.Context, postID, userID string) (*models.Repost, error) {
	var repost models.Repost
	if err := r.reposts.FindOne(ctx, bson.M{"post_id": postID, "user_id": userID}).Decode(&repost); err != nil {
		return nil, err
	}
	return &repost, nil
}

func (r *PostRepository) RemoveRepost(ctx context.Context, postID, userID string) (bool, error) {
	res, err := r.reposts.DeleteOne(ctx, bson.M{"post_id": postID, "user_id": userID})
	if err != nil {
		return false, err
	}
	return res.DeletedCount > 0, nil
}

func (r *PostRepository) RepostedPostIDs(ctx context.Context, userID string) ([]string, error) {
	return r.distinctStrings(ctx, r.reposts, bson.M{"user_id": userID}, "post_id")
}

func (r *PostRepository) ListRepostsByUser(ctx context.Context, userID string, limit, skip int64) ([]models.Repost, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(limit).
		SetSkip(skip)
	cursor, err := r.reposts.Find(ctx, bson.M{"user_id": userID}, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	reposts := []models.Repost{}
	if err := cursor.All(ctx, &reposts); err != nil {
		return nil, err
	}
	return reposts, nil
}

func (r *PostRepository) DeleteRepostsByPost(ctx context.Context, postID string) error {
	_, err := r.reposts.DeleteMany(ctx, bson.M{"post_id": postID})
	return err
}

func (r *PostRepository) DeletePollVotesByPost(ctx context.Context, postID string) error {
	_, err := r.pollVotes.DeleteMany(ctx, bson.M{"post_id": postID})
	return err
}

// --- Comments ----------------------------------------------------------------

// AddComment insère le commentaire et renseigne comment.ID.
func (r *PostRepository) AddComment(ctx context.Context, comment *models.Comment) error {
	res, err := r.comments.InsertOne(ctx, comment)
	if err != nil {
		return err
	}
	if oid, ok := res.InsertedID.(bson.ObjectID); ok {
		comment.ID = oid
	}
	return nil
}

// CommentStatsByIDs renvoie les compteurs de likes des commentaires demandés en
// projection légère, sans leur contenu.
func (r *PostRepository) CommentStatsByIDs(ctx context.Context, oids []bson.ObjectID) ([]models.Comment, error) {
	if len(oids) == 0 {
		return []models.Comment{}, nil
	}
	opts := options.Find().SetProjection(bson.M{
		"likes_count": 1,
	})
	cursor, err := r.comments.Find(ctx, bson.M{"_id": bson.M{"$in": oids}}, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	comments := []models.Comment{}
	if err := cursor.All(ctx, &comments); err != nil {
		return nil, err
	}
	return comments, nil
}

// ListComments renvoie les commentaires RACINE d'un post (parent_id absent/null),
// du plus ancien au plus récent, paginés. Les réponses sont chargées à part
// (ListReplies).
func (r *PostRepository) ListComments(ctx context.Context, postID string, limit, skip int64) ([]models.Comment, error) {
	return r.findComments(ctx, bson.M{"post_id": postID, "parent_id": nil}, limit, skip)
}

// ListReplies renvoie les réponses d'un commentaire racine, chronologiques, paginées.
func (r *PostRepository) ListReplies(ctx context.Context, parentID string, limit, skip int64) ([]models.Comment, error) {
	return r.findComments(ctx, bson.M{"parent_id": parentID}, limit, skip)
}

// findComments factorise la lecture paginée + triée (asc) des commentaires.
func (r *PostRepository) findComments(ctx context.Context, filter bson.M, limit, skip int64) ([]models.Comment, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: 1}}).
		SetLimit(limit).
		SetSkip(skip)

	cursor, err := r.comments.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	comments := []models.Comment{}
	if err := cursor.All(ctx, &comments); err != nil {
		return nil, err
	}
	return comments, nil
}

// IncReplyCount applique `$inc` sur le compteur de réponses d'un commentaire racine.
func (r *PostRepository) IncReplyCount(ctx context.Context, id bson.ObjectID, delta int32) error {
	_, err := r.comments.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$inc": bson.M{"reply_count": delta}})
	return err
}

// IncCommentCounter applique `$inc` sur un compteur dénormalisé d'un
// commentaire (likes_count) et renvoie le document à jour.
func (r *PostRepository) IncCommentCounter(ctx context.Context, id bson.ObjectID, field string, delta int32) (*models.Comment, error) {
	update := bson.M{"$inc": bson.M{field: delta}}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var comment models.Comment
	if err := r.comments.FindOneAndUpdate(ctx, bson.M{"_id": id}, update, opts).Decode(&comment); err != nil {
		return nil, err
	}
	return &comment, nil
}

// DeleteRepliesByParent supprime toutes les réponses d'un commentaire racine
// (cascade à la suppression). Renvoie le nombre de réponses supprimées.
func (r *PostRepository) DeleteRepliesByParent(ctx context.Context, parentID string) (int64, error) {
	res, err := r.comments.DeleteMany(ctx, bson.M{"parent_id": parentID})
	if err != nil {
		return 0, err
	}
	return res.DeletedCount, nil
}

// CommentIDsByParent renvoie tous les ids hexadécimaux des réponses d'un
// commentaire racine. Sert au nettoyage en cascade des likes de réponses.
func (r *PostRepository) CommentIDsByParent(ctx context.Context, parentID string) ([]string, error) {
	return r.distinctStrings(ctx, r.comments, bson.M{"parent_id": parentID}, "_id")
}

// ListCommentsByAuthor renvoie tous les commentaires d'un auteur, du plus
// récent au plus ancien, paginés. Utilisé pour l'onglet « Réponses » du profil.
func (r *PostRepository) ListCommentsByAuthor(ctx context.Context, authorID string, limit, skip int64) ([]models.Comment, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(limit).
		SetSkip(skip)

	cursor, err := r.comments.Find(ctx, bson.M{"author_id": authorID}, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	comments := []models.Comment{}
	if err := cursor.All(ctx, &comments); err != nil {
		return nil, err
	}
	return comments, nil
}

// GetComment renvoie un commentaire par son ObjectID (mongo.ErrNoDocuments si absent).
func (r *PostRepository) GetComment(ctx context.Context, id bson.ObjectID) (*models.Comment, error) {
	var comment models.Comment
	if err := r.comments.FindOne(ctx, bson.M{"_id": id}).Decode(&comment); err != nil {
		return nil, err
	}
	return &comment, nil
}

// DeleteComment supprime un commentaire (mongo.ErrNoDocuments si rien supprimé).
func (r *PostRepository) DeleteComment(ctx context.Context, id bson.ObjectID) error {
	res, err := r.comments.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// AddCommentLike enregistre un like de commentaire, idempotent via l'index
// unique comment_id+user_id.
func (r *PostRepository) AddCommentLike(ctx context.Context, commentID, userID string) (bool, error) {
	_, err := r.commentLikes.InsertOne(ctx, bson.M{
		"comment_id": commentID,
		"user_id":    userID,
		"created_at": time.Now(),
	})
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// RemoveCommentLike supprime un like de commentaire.
func (r *PostRepository) RemoveCommentLike(ctx context.Context, commentID, userID string) (bool, error) {
	res, err := r.commentLikes.DeleteOne(ctx, bson.M{"comment_id": commentID, "user_id": userID})
	if err != nil {
		return false, err
	}
	return res.DeletedCount > 0, nil
}

// LikedCommentIDsByUser renvoie l'intersection entre les commentaires demandés
// et les likes détenus par userID. Sert à hydrater l'état initial des cœurs.
func (r *PostRepository) LikedCommentIDsByUser(ctx context.Context, userID string, commentIDs []string) ([]string, error) {
	if userID == "" || len(commentIDs) == 0 {
		return []string{}, nil
	}
	return r.distinctStrings(ctx, r.commentLikes, bson.M{
		"user_id":    userID,
		"comment_id": bson.M{"$in": commentIDs},
	}, "comment_id")
}

// DeleteCommentLikesByComment purge les likes d'un commentaire supprimé.
func (r *PostRepository) DeleteCommentLikesByComment(ctx context.Context, commentID string) error {
	_, err := r.commentLikes.DeleteMany(ctx, bson.M{"comment_id": commentID})
	return err
}

// DeleteCommentLikesByComments purge les likes d'un lot de commentaires.
func (r *PostRepository) DeleteCommentLikesByComments(ctx context.Context, commentIDs []string) error {
	if len(commentIDs) == 0 {
		return nil
	}
	_, err := r.commentLikes.DeleteMany(ctx, bson.M{"comment_id": bson.M{"$in": commentIDs}})
	return err
}

// DeleteCommentsByPost purge les commentaires d'un post (nettoyage à la suppression).
func (r *PostRepository) DeleteCommentsByPost(ctx context.Context, postID string) error {
	_, err := r.comments.DeleteMany(ctx, bson.M{"post_id": postID})
	return err
}

// distinctStrings factorise la projection d'un champ string sur un filtre.
func (r *PostRepository) distinctStrings(ctx context.Context, coll *mongo.Collection, filter bson.M, field string) ([]string, error) {
	cursor, err := coll.Find(ctx, filter, options.Find().SetProjection(bson.M{field: 1}))
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var docs []bson.M
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(docs))
	for _, d := range docs {
		if v, ok := d[field].(string); ok {
			ids = append(ids, v)
			continue
		}
		if oid, ok := d[field].(bson.ObjectID); ok {
			ids = append(ids, oid.Hex())
		}
	}
	return ids, nil
}

func sortProfilePosts(posts []models.Post) {
	sort.SliceStable(posts, func(i, j int) bool {
		if posts[i].PinnedAt != nil && posts[j].PinnedAt == nil {
			return true
		}
		if posts[i].PinnedAt == nil && posts[j].PinnedAt != nil {
			return false
		}
		return profileSortTime(posts[i]).After(profileSortTime(posts[j]))
	})
}

func profileSortTime(post models.Post) time.Time {
	if post.RepostedAt != nil {
		return *post.RepostedAt
	}
	return post.CreatedAt
}

func postHasHashtag(post *models.Post, hashtag string) bool {
	if post == nil {
		return false
	}
	for _, tag := range post.Hashtags {
		if tag == hashtag {
			return true
		}
	}
	return false
}
