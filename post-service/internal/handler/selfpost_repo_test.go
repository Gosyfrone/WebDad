package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/gin-gonic/gin"
	"github.com/webdad/post-service/internal/models"
	"github.com/webdad/post-service/internal/realtime"
	"github.com/webdad/post-service/internal/service"
)

// callerUserID est l'UserID généré par makePostToken.
const callerUserID = "11111111-1111-1111-1111-111111111111"

// selfPost est un post factice dont l'auteur est le callerUserID (= token de test).
func selfPost() *models.Post {
	return &models.Post{
		ID:            bson.NewObjectID(),
		AuthorID:      callerUserID,
		Content:       "post de test",
		ReplyAudience: models.ReplyAudienceEveryone,
		LikesCount:    0,
		CommentsCount: 0,
		RepostsCount:  0,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

// pollPost est un post factice avec un sondage ouvert (choice validOID).
func pollPost() *models.Post {
	p := selfPost()
	expire := time.Now().Add(24 * time.Hour)
	p.Poll = &models.Poll{
		Choices: []models.PollChoice{
			{ID: validOID, Label: "Option A", VotesCount: 0},
		},
		EndsAt:     expire,
		Audience:   models.PollAudienceEveryone,
		TotalVotes: 0,
	}
	return p
}

// ─── selfPostRepo ─────────────────────────────────────────────────────────────
//
// Embed emptyRepo et override Get + les méthodes qui consomment le post retourné.
// Permet de couvrir les chemins de succès (c.JSON/c.Status 2xx) des handlers
// qui ont besoin d'un *models.Post valide sans Mongo.

type selfPostRepo struct{ emptyRepo }

func (r *selfPostRepo) Get(_ context.Context, _ bson.ObjectID) (*models.Post, error) {
	return selfPost(), nil
}
func (r *selfPostRepo) Pin(_ context.Context, _ bson.ObjectID, _ time.Time) (*models.Post, error) {
	return selfPost(), nil
}
func (r *selfPostRepo) Unpin(_ context.Context, _ bson.ObjectID) (*models.Post, error) {
	return selfPost(), nil
}
func (r *selfPostRepo) Update(_ context.Context, _ bson.ObjectID, content string, _ []string) (*models.Post, error) {
	p := selfPost()
	p.Content = content
	return p, nil
}
func (r *selfPostRepo) SetNsfw(_ context.Context, _ bson.ObjectID, _ bool, _ string, _ time.Time) (*models.Post, error) {
	return selfPost(), nil
}
func (r *selfPostRepo) AddRepost(_ context.Context, _, _ string) (*models.Repost, bool, error) {
	return &models.Repost{PostID: validOID, UserID: callerUserID, CreatedAt: time.Now()}, true, nil
}
func (r *selfPostRepo) IncCounter(_ context.Context, _ bson.ObjectID, _ string, _ int32) (*models.Post, error) {
	return selfPost(), nil
}
func (r *selfPostRepo) ClosePoll(_ context.Context, _ bson.ObjectID, _ time.Time) (*models.Post, error) {
	return selfPost(), nil
}
func (r *selfPostRepo) GetComment(_ context.Context, _ bson.ObjectID) (*models.Comment, error) {
	return &models.Comment{
		ID:         bson.NewObjectID(),
		PostID:     validOID,
		AuthorID:   callerUserID,
		Content:    "commentaire de test",
		LikesCount: 0,
	}, nil
}

// pollPostRepo : selfPost with poll (needed for VotePoll / ClosePoll success).
type pollPostRepo struct{ selfPostRepo }

func (r *pollPostRepo) Get(_ context.Context, _ bson.ObjectID) (*models.Post, error) {
	return pollPost(), nil
}
func (r *pollPostRepo) AddPollVote(_ context.Context, _, _, _ string) (bool, error) {
	return true, nil // created = true, not a duplicate vote
}
func (r *pollPostRepo) IncPollChoice(_ context.Context, _ bson.ObjectID, _ string) (*models.Post, error) {
	return pollPost(), nil
}

// forbiddenPostRepo : Get retourne un post appartenant à "other-user" (pas le caller).
// Permet de tester ErrForbidden → respondPostError → 403.
type forbiddenPostRepo struct{ emptyRepo }

func (r *forbiddenPostRepo) Get(_ context.Context, _ bson.ObjectID) (*models.Post, error) {
	p := selfPost()
	p.AuthorID = "other-user-id"
	return p, nil
}

// alreadyVotedRepo : Get retourne un pollPost valide, AddPollVote retourne false
// (already voted) → service → ErrPollAlreadyVoted → 403.
type alreadyVotedRepo struct{ emptyRepo }

func (r *alreadyVotedRepo) Get(_ context.Context, _ bson.ObjectID) (*models.Post, error) {
	return pollPost(), nil
}
func (r *alreadyVotedRepo) AddPollVote(_ context.Context, _, _, _ string) (bool, error) {
	return false, nil // not created = already voted
}
func (r *alreadyVotedRepo) IncCommentCounter(_ context.Context, _ bson.ObjectID, _ string, _ int32) (*models.Comment, error) {
	return &models.Comment{}, nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func newSelfPostRouter(t interface{ Helper() }) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	svc := service.NewPostService(&selfPostRepo{})
	RegisterRoutes(r, "post-test", svc, nilTestSecret, realtime.NewHub(), nil, nilInternalSec)
	return r
}

func newPollPostRouter(t interface{ Helper() }) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	svc := service.NewPostService(&pollPostRepo{})
	RegisterRoutes(r, "post-test", svc, nilTestSecret, realtime.NewHub(), nil, nilInternalSec)
	return r
}

func newForbiddenPostRouter(t interface{ Helper() }) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	svc := service.NewPostService(&forbiddenPostRepo{})
	RegisterRoutes(r, "post-test", svc, nilTestSecret, realtime.NewHub(), nil, nilInternalSec)
	return r
}

func newAlreadyVotedRouter(t interface{ Helper() }) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	svc := service.NewPostService(&alreadyVotedRepo{})
	RegisterRoutes(r, "post-test", svc, nilTestSecret, realtime.NewHub(), nil, nilInternalSec)
	return r
}

// ─── Tests des chemins de succès avec selfPostRepo ───────────────────────────

func TestGetPost_SelfRepo_200(t *testing.T) {
	r := newSelfPostRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/posts/"+validOID, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("GetPost(selfPostRepo) = %d, attendu 200", w.Code)
	}
}

func TestUpdatePost_SelfRepo_200(t *testing.T) {
	r := newSelfPostRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPatch, "/posts/"+validOID,
		strings.NewReader(`{"content":"contenu mis à jour"}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("UpdatePost(selfPostRepo) = %d, attendu 200", w.Code)
	}
}

func TestPinPost_SelfRepo_200(t *testing.T) {
	r := newSelfPostRouter(t)
	tok := makePostToken(t, "user")
	// PinPost route: PATCH /posts/:id/pin
	req := httptest.NewRequest(http.MethodPatch, "/posts/"+validOID+"/pin", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("PinPost(selfPostRepo) = %d, attendu 200", w.Code)
	}
}

func TestUnpinPost_SelfRepo_200(t *testing.T) {
	r := newSelfPostRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodDelete, "/posts/"+validOID+"/pin", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("UnpinPost(selfPostRepo) = %d, attendu 200", w.Code)
	}
}

func TestRepostPost_SelfRepo_200(t *testing.T) {
	r := newSelfPostRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPost, "/posts/"+validOID+"/repost", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("RepostPost(selfPostRepo) = %d, attendu 200", w.Code)
	}
}

func TestDeletePost_SelfRepo_204(t *testing.T) {
	r := newSelfPostRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodDelete, "/posts/"+validOID, nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("DeletePost(selfPostRepo) = %d, attendu 204", w.Code)
	}
}

func TestRestorePost_SelfRepo_200(t *testing.T) {
	r := newEmptyRepoRouter(t) // emptyRepo.RestoreHidden retourne nil, nil → 200
	tok := makePostToken(t, "moderator")
	req := httptest.NewRequest(http.MethodPost, "/posts/"+validOID+"/restore", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("RestorePost(emptyRepo, mod) = %d, attendu 200", w.Code)
	}
}

func TestVotePoll_SelfRepo_200(t *testing.T) {
	r := newPollPostRouter(t)
	tok := makePostToken(t, "user")
	// Route: POST /posts/:id/poll/vote, body: {"choice_id": "..."}
	req := httptest.NewRequest(http.MethodPost,
		"/posts/"+validOID+"/poll/vote",
		strings.NewReader(`{"choice_id":"`+validOID+`"}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("VotePoll(pollPostRepo) = %d, attendu 200", w.Code)
	}
}

func TestClosePoll_SelfRepo_200(t *testing.T) {
	r := newSelfPostRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPost, "/posts/"+validOID+"/poll/close", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// selfPost n'a pas de poll → ErrInvalidPoll → 400
	// (couverture du chemin "no poll" dans ClosePoll service).
	if w.Code == http.StatusUnauthorized {
		t.Errorf("ClosePoll(selfPostRepo) = %d, ne doit pas être 401", w.Code)
	}
}

// ─── Tests ErrForbidden (respondPostError case 4) ────────────────────────────

func TestDeletePost_Forbidden_403(t *testing.T) {
	r := newForbiddenPostRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodDelete, "/posts/"+validOID, nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// Get → post owned by "other-user" → canModify → false → ErrForbidden → 403.
	if w.Code != http.StatusForbidden {
		t.Errorf("DeletePost(forbiddenPostRepo) = %d, attendu 403", w.Code)
	}
}

func TestPinPost_Forbidden_403(t *testing.T) {
	r := newForbiddenPostRouter(t)
	tok := makePostToken(t, "user")
	// PinPost route: PATCH /posts/:id/pin
	req := httptest.NewRequest(http.MethodPatch, "/posts/"+validOID+"/pin", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// canPin(otherPost, caller) → false → ErrForbidden → 403.
	if w.Code != http.StatusForbidden {
		t.Errorf("PinPost(forbiddenPostRepo) = %d, attendu 403", w.Code)
	}
}

// ─── Tests ErrPollAlreadyVoted (respondPostError case 6) ─────────────────────

func TestVotePoll_AlreadyVoted_403(t *testing.T) {
	r := newAlreadyVotedRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPost,
		"/posts/"+validOID+"/poll/vote",
		strings.NewReader(`{"choice_id":"`+validOID+`"}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// AddPollVote → false (déjà voté) → service → ErrPollAlreadyVoted → 403.
	if w.Code != http.StatusForbidden {
		t.Errorf("VotePoll(alreadyVotedRepo) = %d, attendu 403", w.Code)
	}
}

// ─── Tests SetNsfw succès (moderator token, selfPostRepo) ────────────────────

// ─── Chemins de succès supplémentaires ───────────────────────────────────────

func TestUnrepostPost_SelfRepo_200(t *testing.T) {
	r := newSelfPostRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodDelete, "/posts/"+validOID+"/repost", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// Get → selfPost, RemoveRepost → false (not removed) → return count → 200.
	if w.Code != http.StatusOK {
		t.Errorf("UnrepostPost(selfPostRepo) = %d, attendu 200", w.Code)
	}
}

func TestLikePost_SelfRepo_200(t *testing.T) {
	r := newSelfPostRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPost, "/posts/"+validOID+"/like", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// Get → selfPost, AddLike → false (not created) → return post.LikesCount → 200.
	if w.Code != http.StatusOK {
		t.Errorf("LikePost(selfPostRepo) = %d, attendu 200", w.Code)
	}
}

func TestUnlikePost_SelfRepo_200(t *testing.T) {
	r := newSelfPostRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodDelete, "/posts/"+validOID+"/like", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// Get → selfPost, RemoveLike → false (not removed) → return count → 200.
	if w.Code != http.StatusOK {
		t.Errorf("UnlikePost(selfPostRepo) = %d, attendu 200", w.Code)
	}
}

func TestClosePoll_PollRepo_200(t *testing.T) {
	r := newPollPostRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPost, "/posts/"+validOID+"/poll/close", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// Get → pollPost (poll owned by caller), poll not closed, ClosePoll → selfPost → 200.
	if w.Code != http.StatusOK {
		t.Errorf("ClosePoll(pollPostRepo) = %d, attendu 200", w.Code)
	}
}

func TestLikeComment_SelfRepo_200(t *testing.T) {
	r := newSelfPostRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPost,
		"/posts/"+validOID+"/comments/"+validOID+"/like", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// GetComment → selfComment, AddCommentLike → false → return count → 200.
	if w.Code != http.StatusOK {
		t.Errorf("LikeComment(selfPostRepo) = %d, attendu 200", w.Code)
	}
}

func TestUnlikeComment_SelfRepo_200(t *testing.T) {
	r := newSelfPostRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodDelete,
		"/posts/"+validOID+"/comments/"+validOID+"/like", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// GetComment → selfComment, RemoveCommentLike → false → return count → 200.
	if w.Code != http.StatusOK {
		t.Errorf("UnlikeComment(selfPostRepo) = %d, attendu 200", w.Code)
	}
}

func TestDeletePostComment_SelfRepo_204(t *testing.T) {
	r := newSelfPostRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodDelete,
		"/posts/"+validOID+"/comments/"+validOID, nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// GetComment → selfComment (caller is author) → canAct → true → delete → 204.
	if w.Code != http.StatusNoContent {
		t.Errorf("DeletePostComment(selfPostRepo) = %d, attendu 204", w.Code)
	}
}

func TestSetNsfw_SelfRepo_200(t *testing.T) {
	r := newSelfPostRouter(t)
	tok := makePostToken(t, "moderator")
	req := httptest.NewRequest(http.MethodPatch, "/posts/"+validOID+"/nsfw",
		strings.NewReader(`{"nsfw":true}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("SetNsfw(selfPostRepo, mod) = %d, attendu 200", w.Code)
	}
}
