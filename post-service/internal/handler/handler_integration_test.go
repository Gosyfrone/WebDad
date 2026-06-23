package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/webdad/post-service/internal/database"
	"github.com/webdad/post-service/internal/middleware"
	"github.com/webdad/post-service/internal/realtime"
	"github.com/webdad/post-service/internal/repository"
	"github.com/webdad/post-service/internal/service"
)

const itSecret = "integration-test-secret"

func mongoURI() string {
	if u := os.Getenv("MONGO_TEST_URI"); u != "" {
		return u
	}
	return "mongodb://localhost:27017"
}

// newMongoRouter monte le routeur complet sur un Mongo réel (DB éphémère). Skip
// si Mongo est indisponible ou refuse les opérations (auth du stack de dev) —
// la CI fournit un Mongo dédié sans auth (cf. ci-go.yml).
func newMongoRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client, err := mongo.Connect(options.Client().ApplyURI(mongoURI()))
	if err != nil {
		t.Skipf("Mongo indisponible (%v)", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		t.Skipf("Mongo injoignable (%v)", err)
	}
	db := client.Database(fmt.Sprintf("webdad_post_it_%d", time.Now().UnixNano()))
	if err := db.Collection("__probe").Drop(ctx); err != nil {
		_ = client.Disconnect(ctx)
		t.Skipf("Mongo refuse les opérations (%v)", err)
	}
	if err := database.EnsureSchema(ctx, db); err != nil {
		_ = client.Disconnect(ctx)
		t.Skipf("EnsureSchema impossible (%v)", err)
	}
	t.Cleanup(func() {
		_ = db.Drop(ctx)
		_ = client.Disconnect(ctx)
	})

	svc := service.NewPostService(repository.NewPostRepository(db), service.WithBookmarkWindow(5*time.Minute))
	r := gin.New()
	RegisterRoutes(r, "post-it", svc, itSecret, realtime.NewHub(), nil, "it-internal-secret")
	return r
}

func itToken(t *testing.T, userID, role string) string {
	t.Helper()
	now := time.Now()
	claims := middleware.Claims{
		UserID:        userID,
		Email:         userID + "@breezy.dev",
		EmailVerified: true,
		Role:          role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(itSecret))
	if err != nil {
		t.Fatalf("itToken: %v", err)
	}
	return tok
}

// okCode : succès HTTP (2xx). Les écritures renvoient 200 ou 204 selon le handler.
func okCode(c int) bool { return c >= 200 && c < 300 }

// do exécute une requête authentifiée et renvoie le recorder.
func do(t *testing.T, r *gin.Engine, method, path, tok, body string) *httptest.ResponseRecorder {
	t.Helper()
	var rdr *bytes.Reader
	if body != "" {
		rdr = bytes.NewReader([]byte(body))
	} else {
		rdr = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, rdr)
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// createPost crée un post via l'API et renvoie son id.
func createPost(t *testing.T, r *gin.Engine, tok, content string) string {
	t.Helper()
	w := do(t, r, http.MethodPost, "/posts", tok, fmt.Sprintf(`{"content":%q}`, content))
	if w.Code != http.StatusCreated {
		t.Fatalf("CreatePost = %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("décodage création: %v", err)
	}
	return resp.Data.ID
}

func TestIT_PostLifecycle(t *testing.T) {
	r := newMongoRouter(t)
	tok := itToken(t, "alice", "user")

	id := createPost(t, r, tok, "mon premier post #breezy @bob")

	if w := do(t, r, http.MethodGet, "/posts/"+id, tok, ""); !okCode(w.Code) {
		t.Fatalf("GetPost = %d", w.Code)
	}
	if w := do(t, r, http.MethodGet, "/posts", "", ""); !okCode(w.Code) {
		t.Fatalf("ListPosts = %d", w.Code)
	}
	if w := do(t, r, http.MethodGet, "/posts?author_id=alice", tok, ""); !okCode(w.Code) {
		t.Fatalf("ListPosts(author) = %d", w.Code)
	}
	if w := do(t, r, http.MethodGet, "/posts/stats?ids="+id, tok, ""); !okCode(w.Code) {
		t.Fatalf("PostStats = %d", w.Code)
	}
	if w := do(t, r, http.MethodGet, "/posts/trends", tok, ""); !okCode(w.Code) {
		t.Fatalf("Trends = %d", w.Code)
	}
	// Édition + épinglage.
	// NB : on garde un hashtag dans le contenu édité — repo.Update persiste
	// hashtags=null si le contenu n'en a aucun, ce que le validateur Mongo rejette
	// (bug latent signalé à part).
	if w := do(t, r, http.MethodPatch, "/posts/"+id, tok, `{"content":"édité #maj"}`); !okCode(w.Code) {
		t.Fatalf("UpdatePost = %d, body=%s", w.Code, w.Body.String())
	}
	if w := do(t, r, http.MethodPatch, "/posts/"+id+"/pin", tok, ""); !okCode(w.Code) {
		t.Fatalf("PinPost = %d", w.Code)
	}
	if w := do(t, r, http.MethodDelete, "/posts/"+id+"/pin", tok, ""); !okCode(w.Code) {
		t.Fatalf("UnpinPost = %d", w.Code)
	}
	// Suppression par l'auteur.
	if w := do(t, r, http.MethodDelete, "/posts/"+id, tok, ""); !okCode(w.Code) {
		t.Fatalf("DeletePost = %d", w.Code)
	}
}

func TestIT_LikesAndReposts(t *testing.T) {
	r := newMongoRouter(t)
	author := itToken(t, "alice", "user")
	liker := itToken(t, "bob", "user")
	id := createPost(t, r, author, "post à aimer")

	if w := do(t, r, http.MethodPost, "/posts/"+id+"/like", liker, ""); !okCode(w.Code) {
		t.Fatalf("LikePost = %d, body=%s", w.Code, w.Body.String())
	}
	if w := do(t, r, http.MethodGet, "/posts/"+id+"/likes", "", ""); !okCode(w.Code) {
		t.Fatalf("ListPostLikes = %d", w.Code)
	}
	if w := do(t, r, http.MethodGet, "/posts/me/liked-ids", liker, ""); !okCode(w.Code) {
		t.Fatalf("LikedByMe = %d", w.Code)
	}
	if w := do(t, r, http.MethodDelete, "/posts/"+id+"/like", liker, ""); !okCode(w.Code) {
		t.Fatalf("UnlikePost = %d", w.Code)
	}
	if w := do(t, r, http.MethodPost, "/posts/"+id+"/repost", liker, ""); !okCode(w.Code) {
		t.Fatalf("RepostPost = %d, body=%s", w.Code, w.Body.String())
	}
	if w := do(t, r, http.MethodGet, "/posts/me/reposted-ids", liker, ""); !okCode(w.Code) {
		t.Fatalf("RepostedByMe = %d", w.Code)
	}
	if w := do(t, r, http.MethodDelete, "/posts/"+id+"/repost", liker, ""); !okCode(w.Code) {
		t.Fatalf("UnrepostPost = %d", w.Code)
	}
}

func TestIT_Comments(t *testing.T) {
	r := newMongoRouter(t)
	author := itToken(t, "alice", "user")
	id := createPost(t, r, author, "post commenté")

	w := do(t, r, http.MethodPost, "/posts/"+id+"/comments", author, `{"content":"joli @bob"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("CreateComment = %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	cid := resp.Data.ID

	if w := do(t, r, http.MethodGet, "/posts/"+id+"/comments", "", ""); !okCode(w.Code) {
		t.Fatalf("ListPostComments = %d", w.Code)
	}
	if w := do(t, r, http.MethodGet, "/posts/comments?author_id=alice", author, ""); !okCode(w.Code) {
		t.Fatalf("ListCommentsByAuthor = %d", w.Code)
	}
	if w := do(t, r, http.MethodGet, "/posts/comments/stats?ids="+cid, "", ""); !okCode(w.Code) {
		t.Fatalf("CommentStats = %d", w.Code)
	}
	if w := do(t, r, http.MethodPost, "/posts/"+id+"/comments/"+cid+"/like", author, ""); !okCode(w.Code) {
		t.Fatalf("LikeComment = %d, body=%s", w.Code, w.Body.String())
	}
	if w := do(t, r, http.MethodDelete, "/posts/"+id+"/comments/"+cid+"/like", author, ""); !okCode(w.Code) {
		t.Fatalf("UnlikeComment = %d", w.Code)
	}
	if w := do(t, r, http.MethodDelete, "/posts/"+id+"/comments/"+cid, author, ""); !okCode(w.Code) {
		t.Fatalf("DeleteComment = %d", w.Code)
	}
}

func TestIT_Bookmarks(t *testing.T) {
	r := newMongoRouter(t)
	tok := itToken(t, "alice", "user")
	id := createPost(t, r, tok, "à ranger")

	// Crée une collection.
	w := do(t, r, http.MethodPost, "/posts/bookmarks/collections", tok, `{"name":"Voyages"}`)
	if w.Code != http.StatusCreated && !okCode(w.Code) {
		t.Fatalf("CreateCollection = %d, body=%s", w.Code, w.Body.String())
	}
	var coll struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &coll)
	cid := coll.Data.ID

	if w := do(t, r, http.MethodGet, "/posts/bookmarks/collections", tok, ""); !okCode(w.Code) {
		t.Fatalf("ListCollections = %d", w.Code)
	}
	if w := do(t, r, http.MethodPatch, "/posts/bookmarks/collections/"+cid, tok, `{"name":"Vacances"}`); !okCode(w.Code) {
		t.Fatalf("RenameCollection = %d, body=%s", w.Code, w.Body.String())
	}
	if w := do(t, r, http.MethodPost, "/posts/"+id+"/bookmark", tok, fmt.Sprintf(`{"collection_id":%q}`, cid)); !okCode(w.Code) {
		t.Fatalf("Bookmark = %d, body=%s", w.Code, w.Body.String())
	}
	if w := do(t, r, http.MethodGet, "/posts/me/bookmarked-ids", tok, ""); !okCode(w.Code) {
		t.Fatalf("BookmarkedByMe = %d", w.Code)
	}
	if w := do(t, r, http.MethodGet, "/posts/bookmarks", tok, ""); !okCode(w.Code) {
		t.Fatalf("ListAll = %d", w.Code)
	}
	if w := do(t, r, http.MethodGet, "/posts/"+id+"/bookmark/collections", tok, ""); !okCode(w.Code) {
		t.Fatalf("PostCollections = %d", w.Code)
	}
	if w := do(t, r, http.MethodGet, "/posts/bookmarks/collections/"+cid+"/posts", tok, ""); !okCode(w.Code) {
		t.Fatalf("ListCollectionPosts = %d", w.Code)
	}
	if w := do(t, r, http.MethodDelete, "/posts/"+id+"/bookmark", tok, ""); !okCode(w.Code) {
		t.Fatalf("Unbookmark = %d", w.Code)
	}
	if w := do(t, r, http.MethodDelete, "/posts/bookmarks/collections/"+cid, tok, ""); !okCode(w.Code) {
		t.Fatalf("DeleteCollection = %d", w.Code)
	}
}

func TestIT_ModerationFlow(t *testing.T) {
	r := newMongoRouter(t)
	author := itToken(t, "alice", "user")
	mod := itToken(t, "mod", "moderator")
	admin := itToken(t, "admin", "admin")
	id := createPost(t, r, author, "post à modérer")

	// Un modérateur retire le post d'autrui → masquage doux.
	if w := do(t, r, http.MethodDelete, "/posts/"+id, mod, ""); !okCode(w.Code) {
		t.Fatalf("DeletePost (mod) = %d", w.Code)
	}
	if w := do(t, r, http.MethodGet, "/posts/moderation/deleted", mod, ""); !okCode(w.Code) {
		t.Fatalf("ListHidden = %d", w.Code)
	}
	if w := do(t, r, http.MethodPost, "/posts/"+id+"/restore", mod, ""); !okCode(w.Code) {
		t.Fatalf("RestorePost = %d", w.Code)
	}
	if w := do(t, r, http.MethodPatch, "/posts/"+id+"/nsfw", mod, `{"nsfw":true}`); !okCode(w.Code) {
		t.Fatalf("SetNsfw = %d, body=%s", w.Code, w.Body.String())
	}
	// Purge RGPD par auteur (admin).
	if w := do(t, r, http.MethodDelete, "/posts/by-author/alice", admin, ""); !okCode(w.Code) {
		t.Fatalf("PurgeUserData = %d, body=%s", w.Code, w.Body.String())
	}
}

func TestIT_Poll(t *testing.T) {
	r := newMongoRouter(t)
	author := itToken(t, "alice", "user")
	voter := itToken(t, "bob", "user")

	body := `{"content":"sondage","poll":{"duration_minutes":60,"audience":"everyone","choices":[{"label":"A"},{"label":"B"}]}}`
	w := do(t, r, http.MethodPost, "/posts", author, body)
	if w.Code != http.StatusCreated {
		t.Fatalf("CreatePost(poll) = %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			ID   string `json:"id"`
			Poll struct {
				Choices []struct {
					ID string `json:"id"`
				} `json:"choices"`
			} `json:"poll"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("décodage poll: %v", err)
	}
	id := resp.Data.ID
	choiceID := resp.Data.Poll.Choices[0].ID

	if w := do(t, r, http.MethodPost, "/posts/"+id+"/poll/vote", voter, fmt.Sprintf(`{"choice_id":%q}`, choiceID)); !okCode(w.Code) {
		t.Fatalf("VotePoll = %d, body=%s", w.Code, w.Body.String())
	}
	if w := do(t, r, http.MethodPost, "/posts/"+id+"/poll/close", author, ""); !okCode(w.Code) {
		t.Fatalf("ClosePoll = %d, body=%s", w.Code, w.Body.String())
	}
}
