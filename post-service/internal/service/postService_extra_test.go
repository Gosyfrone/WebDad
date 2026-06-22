package service

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/webdad/post-service/internal/models"
	"github.com/webdad/post-service/internal/notifier"
)

// DeleteComment d'une RÉPONSE : décrémente reply_count de la racine et notifie
// via commentAuthor (branche parent.ParentID != "").
func TestDeleteComment_Reply(t *testing.T) {
	postID := newOID()
	rootID := newOID()
	reply := &models.Comment{ID: newOID(), PostID: postID.Hex(), ParentID: rootID.Hex(), AuthorID: "replier"}
	root := &models.Comment{ID: rootID, PostID: postID.Hex(), AuthorID: "root-author"}
	repo := &fakeRepo{
		fnGetComment: func(_ context.Context, id bson.ObjectID) (*models.Comment, error) {
			if id == rootID {
				return root, nil
			}
			return reply, nil
		},
		fnDeleteComment:               func(context.Context, bson.ObjectID) error { return nil },
		fnIncReplyCount:               func(context.Context, bson.ObjectID, int32) error { return nil },
		fnDeleteCommentLikesByComment: func(context.Context, string) error { return nil },
		fnIncCounter:                  func(context.Context, bson.ObjectID, string, int32) (*models.Post, error) { return nil, nil },
	}
	notif := &capNotifier{}
	s := NewPostService(repo, WithNotifier(notif))
	if err := s.DeleteComment(bg(), reply.ID.Hex(), "replier", models.RoleUser); err != nil {
		t.Fatalf("DeleteComment reply: %v", err)
	}
	if notif.typeCount("reply") != 1 {
		t.Fatalf("rétractation reply attendue, events=%v", notif.events)
	}
}

// SetNotifier / WithFeedBroadcaster : câblage optionnel.
func TestSetterOptions(t *testing.T) {
	s := NewPostService(&fakeRepo{})
	s.SetNotifier(notifier.Noop{})
	s.SetNotifier(nil) // nil ignoré, pas de panic

	hub := &recordingFeed{}
	s2 := NewPostService(&fakeRepo{}, WithFeedBroadcaster(hub))
	s2.feed.PostCreated("p", "a")
	if hub.calls != 1 {
		t.Fatalf("le broadcaster câblé doit recevoir PostCreated, calls=%d", hub.calls)
	}
}

type recordingFeed struct{ calls int }

func (r *recordingFeed) PostCreated(string, string) { r.calls++ }

// chanFeed signale chaque PostCreated sur un canal (synchronisation des tests
// de diffusion temps réel, qui s'exécute dans une goroutine).
type chanFeed struct{ ping chan struct{} }

func (c *chanFeed) PostCreated(string, string) { c.ping <- struct{}{} }

func TestNoopBroadcaster(t *testing.T) {
	noopBroadcaster{}.PostCreated("p", "a") // no-op, ne doit pas paniquer.
}

// broadcastNewPost : compte public → diffusion temps réel.
func TestBroadcastNewPost_Public(t *testing.T) {
	feed := &chanFeed{ping: make(chan struct{}, 1)}
	s := NewPostService(&fakeRepo{}, WithFeedBroadcaster(feed), WithProfilClient(&stubProfil{visibility: "public"}))
	s.broadcastNewPost("a1", "p1")
	select {
	case <-feed.ping:
	case <-time.After(2 * time.Second):
		t.Fatal("un post public doit être diffusé")
	}
}

// broadcastNewPost : compte privé → pas de diffusion.
func TestBroadcastNewPost_PrivateSkips(t *testing.T) {
	feed := &chanFeed{ping: make(chan struct{}, 1)}
	s := NewPostService(&fakeRepo{}, WithFeedBroadcaster(feed), WithProfilClient(&stubProfil{visibility: "private"}))
	s.broadcastNewPost("a1", "p1")
	select {
	case <-feed.ping:
		t.Fatal("un post de compte privé ne doit pas être diffusé")
	case <-time.After(150 * time.Millisecond):
	}
}

// followingErrClient : pas de blocage, mais la vérification d'abonnement échoue.
type followingErrClient struct{}

func (followingErrClient) IsFollowing(context.Context, string, string) (bool, error) {
	return false, errBoom
}
func (followingErrClient) HasBlocked(context.Context, string, string) (bool, error) {
	return false, nil
}

// hydrateReplyPermission : une erreur de dépendance échoue FERMÉ (CanReply=false).
func TestHydrateReplyPermission_FailsClosed(t *testing.T) {
	p := postWith("a1")
	p.ReplyAudience = models.ReplyAudienceFollowers
	repo := &fakeRepo{fnGet: func(context.Context, bson.ObjectID) (*models.Post, error) { return p, nil }}
	s := NewPostService(repo, WithFollowClient(followingErrClient{}))
	got, err := s.GetPost(bg(), p.ID.Hex(), "viewer", models.RoleUser)
	if err != nil {
		t.Fatalf("GetPost: %v", err)
	}
	if got.CanReply {
		t.Fatal("en cas d'erreur de vérification d'abonnement, CanReply doit être false (fail-closed)")
	}
}

// RunPurgeSweeper : un passage immédiat puis sortie sur ctx annulé.
func TestRunPurgeSweeper_StopsOnCancel(t *testing.T) {
	repo := &fakeRepo{
		fnListPurgeWarnable: func(context.Context, time.Time, int64) ([]models.Post, error) { return nil, nil },
		fnListPurgeable:     func(context.Context, time.Time, int64) ([]models.Post, error) { return nil, nil },
	}
	s := NewPostService(repo, WithPurgeRetention(24*time.Hour, time.Hour))
	ctx, cancel := context.WithCancel(bg())
	cancel() // annulé d'emblée : un passage immédiat, puis return.
	done := make(chan struct{})
	go func() { s.RunPurgeSweeper(ctx, time.Hour); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("RunPurgeSweeper ne s'est pas arrêté sur ctx annulé")
	}
}

// RunPurgeSweeper sans rétention : no-op immédiat.
func TestRunPurgeSweeper_NoRetention(t *testing.T) {
	s := NewPostService(&fakeRepo{})
	s.RunPurgeSweeper(bg(), time.Hour) // retour immédiat, pas de blocage.
}
