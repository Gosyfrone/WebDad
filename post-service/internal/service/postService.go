// Package service porte la logique métier du post-service : validation des
// identifiants, contrôle de propriété et traduction des erreurs du dépôt en
// erreurs métier (mappées vers des codes HTTP par les handlers).
package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/webdad/post-service/internal/client"
	"github.com/webdad/post-service/internal/models"
	"github.com/webdad/post-service/internal/notifier"
	"github.com/webdad/post-service/internal/repository"
)

// Erreurs métier — traduites en codes HTTP par les handlers.
var (
	// ErrPostNotFound : aucun post pour cet id → 404.
	ErrPostNotFound = errors.New("post introuvable")
	// ErrInvalidID : id de post mal formé (pas un ObjectID hexadécimal) → 400.
	ErrInvalidID = errors.New("identifiant de post invalide")
	// ErrForbidden : l'utilisateur n'est ni l'auteur ni un modérateur/admin → 403.
	ErrForbidden = errors.New("action non autorisée sur ce post")
	// ErrPrivateProfil : profil privé inaccessible au visiteur courant → 403.
	ErrPrivateProfil = errors.New("profil privé")
	// ErrDependencyUnavailable : profil-service / user-service indisponible → 503.
	ErrDependencyUnavailable = errors.New("service dépendant indisponible")
	ErrInvalidPoll           = errors.New("sondage invalide")
	ErrPollClosed            = errors.New("sondage terminé")
	ErrPollAlreadyVoted      = errors.New("vote déjà enregistré")
	// ErrInvalidReplyAudience : valeur d'audience des réponses hors enum → 400.
	ErrInvalidReplyAudience = errors.New("audience des réponses invalide")
	// ErrReplyNotAllowed : le post restreint les réponses aux abonnés et le lecteur
	// n'est ni l'auteur, ni un abonné, ni un modérateur/admin → 403.
	ErrReplyNotAllowed = errors.New("réponses réservées aux abonnés")
	// ErrCollectionNotFound : collection de signets absente ou n'appartenant pas à
	// l'utilisateur (on ne distingue pas pour ne pas divulguer l'existence) → 404.
	ErrCollectionNotFound = errors.New("collection de signets introuvable")
	// ErrDefaultCollection : action interdite sur la collection par défaut
	// (suppression / renommage) → 403.
	ErrDefaultCollection = errors.New("collection par défaut non modifiable")
)

// Statuts renvoyés par un clic court sur le bouton signet.
const (
	// BookmarkStatusFiled : le post a été rangé automatiquement (fenêtre active).
	BookmarkStatusFiled = "filed"
	// BookmarkStatusNeedsChoice : ouverture de rafale — le front doit proposer la
	// collection (1er signet ou fenêtre expirée) ; rien n'est rangé.
	BookmarkStatusNeedsChoice = "needs_choice"
)

// Bornes de pagination des listes.
const (
	DefaultLimit = 20
	MaxLimit     = 100
)

// MaxStatsIDs borne le nombre d'ids acceptés par une requête de compteurs
// (GET /posts/stats) — au-delà, on n'honore que les premiers. Couvre largement
// une page de fil affichée côté front.
const MaxStatsIDs = 100

var hashtagPattern = regexp.MustCompile(`(^|[^\p{L}\p{N}_])#([\p{L}\p{N}_]{1,64})`)

type PostService struct {
	repo *repository.PostRepository
	// notif émet les événements de notification (like, commentaire, mention…).
	// Par défaut un no-op : le post-service reste autonome si le
	// notification-service n'est pas configuré. Câblé via SetNotifier au boot.
	notif notifier.Notifier
	// feed diffuse en temps réel les nouveaux posts (ping WebSocket). Par défaut
	// un no-op : le post-service fonctionne sans le hub temps réel.
	feed feedBroadcaster
	// bookmarkWindow : durée de la fenêtre glissante de rafale. Un clic court qui
	// suit le précédent signet de moins de bookmarkWindow range automatiquement
	// dans la dernière collection ; au-delà, le serveur redemande la collection.
	// <= 0 = jamais d'auto-classement (toujours proposer).
	bookmarkWindow time.Duration
	profilClient   profilVisibilityClient
	followClient   followStatusClient
	// purgeAfter : rétention d'un tweet masqué avant purge RGPD ; purgeWarnBefore :
	// préavis avant purge. <= 0 sur purgeAfter → balayage désactivé.
	purgeAfter      time.Duration
	purgeWarnBefore time.Duration
}

type profilVisibilityClient interface {
	Visibility(ctx context.Context, userID string) (string, error)
	LikesVisibility(ctx context.Context, userID string) (string, error)
}

type followStatusClient interface {
	IsFollowing(ctx context.Context, followerID, followingID string) (bool, error)
}

// feedBroadcaster diffuse en temps réel la création d'un post (ping WebSocket).
// Implémenté par realtime.Hub ; no-op par défaut (service testable sans hub).
type feedBroadcaster interface {
	PostCreated(postID, authorID string)
}

// noopBroadcaster : diffusion désactivée (pas de hub temps réel câblé).
type noopBroadcaster struct{}

func (noopBroadcaster) PostCreated(string, string) {}

type Option func(*PostService)

// WithFeedBroadcaster branche le hub temps réel du fil (best-effort).
func WithFeedBroadcaster(b feedBroadcaster) Option {
	return func(s *PostService) {
		if b != nil {
			s.feed = b
		}
	}
}

func WithProfilClient(c profilVisibilityClient) Option {
	return func(s *PostService) {
		s.profilClient = c
	}
}

func WithFollowClient(c followStatusClient) Option {
	return func(s *PostService) {
		s.followClient = c
	}
}

func WithNotifier(n notifier.Notifier) Option {
	return func(s *PostService) {
		if n != nil {
			s.notif = n
		}
	}
}

func WithBookmarkWindow(window time.Duration) Option {
	return func(s *PostService) {
		s.bookmarkWindow = window
	}
}

// WithPurgeRetention configure la rétention RGPD des tweets masqués (durée avant
// purge définitive) et le préavis avant purge.
func WithPurgeRetention(after, warnBefore time.Duration) Option {
	return func(s *PostService) {
		s.purgeAfter = after
		s.purgeWarnBefore = warnBefore
	}
}

func NewPostService(r *repository.PostRepository, opts ...Option) *PostService {
	s := &PostService{repo: r, notif: notifier.Noop{}, feed: noopBroadcaster{}}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// SetNotifier branche l'émetteur d'événements de notification (best-effort).
func (s *PostService) SetNotifier(n notifier.Notifier) {
	if n != nil {
		s.notif = n
	}
}

// CreatePost crée un post pour authorID (dérivé du JWT) et renvoie le document
// créé (avec son id généré). Les compteurs sont posés à 0 explicitement.
func (s *PostService) CreatePost(ctx context.Context, authorID, content, quotePostID string, media []models.MediaRef, pollReq *models.CreatePollRequest, replyAudience string) (*models.Post, error) {
	quotedAuthorID := "" // auteur du post cité (destinataire de la notif « citation »)
	if quotePostID != "" {
		quoteOID, err := parseID(quotePostID)
		if err != nil {
			return nil, err
		}
		quoted, err := s.repo.Get(ctx, quoteOID)
		if err != nil {
			return nil, translateNotFound(err)
		}
		quotedAuthorID = quoted.AuthorID
	}
	now := time.Now()
	poll, err := buildPoll(pollReq, now)
	if err != nil {
		return nil, err
	}
	audience := replyAudience
	if audience == "" {
		audience = models.ReplyAudienceEveryone
	}
	if audience != models.ReplyAudienceEveryone && audience != models.ReplyAudienceFollowers {
		return nil, ErrInvalidReplyAudience
	}
	post := &models.Post{
		AuthorID:      authorID,
		Content:       content,
		Hashtags:      ExtractHashtags(content),
		Media:         media,
		Poll:          poll,
		ReplyAudience: audience,
		QuotePostID:   quotePostID,
		LikesCount:    0,
		CommentsCount: 0,
		RepostsCount:  0,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := s.repo.Create(ctx, post); err != nil {
		return nil, err
	}
	hydratePoll(post, authorID, "")
	post.CanReply = true // l'auteur peut toujours répondre à son propre post
	// Citation = tag implicite de l'auteur cité → notif (navigation vers le post
	// citant). Une notif par citation (façon mention, pas d'agrégation). La
	// suppression du post citant la purge via la cascade post_deleted.
	if quotePostID != "" {
		s.notif.Emit(notifier.Event{
			Type:        notifier.TypeQuote,
			ActorID:     authorID,
			RecipientID: quotedAuthorID,
			PostID:      post.ID.Hex(),
		})
	}
	// Notifie les utilisateurs mentionnés (@handle) dans le post.
	if handles := notifier.ParseMentions(content); len(handles) > 0 {
		s.notif.Emit(notifier.Event{
			Type:           notifier.TypeMention,
			ActorID:        authorID,
			PostID:         post.ID.Hex(),
			MentionHandles: handles,
		})
	}
	// Diffuse le nouveau post en temps réel (ping WebSocket) — best-effort,
	// uniquement pour les comptes publics (voir broadcastNewPost). N'impacte
	// jamais la création.
	s.broadcastNewPost(authorID, post.ID.Hex())
	return post, nil
}

// broadcastNewPost notifie en temps réel (best-effort, fire-and-forget) que
// authorID vient de publier postID. On ne diffuse qu'aux comptes PUBLICS : un
// post de compte privé ne concerne que les abonnés approuvés et ne doit pas
// révéler l'activité de l'auteur aux autres (la sécurité ne dépend jamais du
// front). Le ping ne porte que des ids — le contenu reste protégé par la
// barrière de visibilité du fil normal côté lecture.
func (s *PostService) broadcastNewPost(authorID, postID string) {
	go func() {
		if s.profilClient != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			vis, err := s.profilClient.Visibility(ctx, authorID)
			if err != nil || vis != client.VisibilityPublic {
				return
			}
		}
		s.feed.PostCreated(postID, authorID)
	}()
}

// GetPosts renvoie le fil global, du plus récent au plus ancien, paginé.
func (s *PostService) GetPosts(ctx context.Context, viewerID, hashtag, sortMode string, hashtagAny bool, limit, offset int64) ([]models.Post, error) {
	hashtag = normalizeHashtag(hashtag)
	sortMode = normalizePostSort(sortMode)
	fetch := s.repo.GetAll
	if hashtag != "" {
		fetch = func(ctx context.Context, pageLimit, pageOffset int64) ([]models.Post, error) {
			return s.repo.GetAllByHashtag(ctx, hashtag, sortMode, pageLimit, pageOffset)
		}
	} else if hashtagAny {
		fetch = s.repo.GetAllWithHashtags
	}
	posts, err := s.visibleFeedPage(ctx, viewerID, limit, offset, fetch)
	if err != nil {
		return nil, err
	}
	s.hydratePolls(ctx, posts, viewerID)
	s.hydrateReplyPermissions(ctx, posts, viewerID)
	return posts, nil
}

// GetPost renvoie un post par son id si le profil de l'auteur est lisible par
// le visiteur courant. Un post masqué par la modération est invisible (404)
// sauf pour un modérateur/admin (qui le consulte depuis la corbeille).
func (s *PostService) GetPost(ctx context.Context, id, viewerID, viewerRole string) (*models.Post, error) {
	oid, err := parseID(id)
	if err != nil {
		return nil, err
	}
	post, err := s.repo.Get(ctx, oid)
	if err != nil {
		return nil, translateNotFound(err)
	}
	// Masqué (retrait manuel) OU auto-masqué (seuil de signalements) : invisible
	// au public, mais consultable par un modérateur (pour juger sur pièce dans le
	// détail du ticket).
	if (post.IsHidden || post.AutoHidden) && !isModerator(viewerRole) {
		return nil, ErrPostNotFound
	}
	allowed, err := s.canReadAuthor(ctx, viewerID, post.AuthorID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrPrivateProfil
	}
	s.hydratePoll(ctx, post, viewerID)
	s.hydrateReplyPermission(ctx, post, viewerID, viewerRole)
	return post, nil
}

// PostStats renvoie les compteurs (likes/commentaires/reposts) des posts
// demandés que le visiteur courant a le droit de voir. Léger (projection sur les
// compteurs, une seule requête `$in`) : sert au rafraîchissement périodique des
// compteurs côté front, façon X. Les posts invisibles (compte privé non suivi,
// masqués par la modération, introuvables, id invalide) sont simplement absents
// de la réponse — pas d'erreur (le front ne fait que patcher ce qu'il connaît).
func (s *PostService) PostStats(ctx context.Context, ids []string, viewerID string) ([]models.PostStat, error) {
	if len(ids) > MaxStatsIDs {
		ids = ids[:MaxStatsIDs]
	}
	oids := make([]bson.ObjectID, 0, len(ids))
	for _, id := range ids {
		if oid, err := parseID(id); err == nil {
			oids = append(oids, oid)
		}
	}
	posts, err := s.repo.StatsByIDs(ctx, oids)
	if err != nil {
		return nil, err
	}
	// Même barrière de visibilité que le fil (canReadAuthor), mémoïsée par auteur
	// pour ne pas multiplier les appels profil/follow quand une page contient
	// plusieurs posts du même auteur.
	allowedByAuthor := make(map[string]bool)
	stats := make([]models.PostStat, 0, len(posts))
	for _, post := range posts {
		allowed, ok := allowedByAuthor[post.AuthorID]
		if !ok {
			allowed, err = s.canReadAuthor(ctx, viewerID, post.AuthorID)
			if err != nil {
				return nil, err
			}
			allowedByAuthor[post.AuthorID] = allowed
		}
		if !allowed {
			continue
		}
		stats = append(stats, models.PostStat{
			ID:            post.ID.Hex(),
			LikesCount:    post.LikesCount,
			CommentsCount: post.CommentsCount,
			RepostsCount:  post.RepostsCount,
		})
	}
	return stats, nil
}

// UpdatePost modifie le contenu d'un post si l'acteur en a le droit (auteur,
// modérateur ou admin).
func (s *PostService) UpdatePost(ctx context.Context, id, content, actorID, actorRole string) (*models.Post, error) {
	oid, err := parseID(id)
	if err != nil {
		return nil, err
	}
	post, err := s.repo.Get(ctx, oid)
	if err != nil {
		return nil, translateNotFound(err)
	}
	if !canModify(post, actorID, actorRole) {
		return nil, ErrForbidden
	}
	updated, err := s.repo.Update(ctx, oid, content, ExtractHashtags(content))
	return updated, translateNotFound(err)
}

// PinPost épingle un post sur le profil de son auteur. Contrairement à la
// suppression/édition, les modérateurs/admins ne peuvent pas épingler à la
// place de l'auteur : c'est un choix de profil personnel.
func (s *PostService) PinPost(ctx context.Context, id, actorID string) (*models.Post, error) {
	oid, err := parseID(id)
	if err != nil {
		return nil, err
	}
	post, err := s.repo.Get(ctx, oid)
	if err != nil {
		return nil, translateNotFound(err)
	}
	if !canPin(post, actorID) {
		return nil, ErrForbidden
	}
	if err := s.repo.UnpinByAuthor(ctx, post.AuthorID); err != nil {
		return nil, err
	}
	pinned, err := s.repo.Pin(ctx, oid, time.Now())
	return pinned, translateNotFound(err)
}

// UnpinPost retire l'épinglage d'un post. Seul l'auteur peut le faire.
func (s *PostService) UnpinPost(ctx context.Context, id, actorID string) (*models.Post, error) {
	oid, err := parseID(id)
	if err != nil {
		return nil, err
	}
	post, err := s.repo.Get(ctx, oid)
	if err != nil {
		return nil, translateNotFound(err)
	}
	if !canPin(post, actorID) {
		return nil, ErrForbidden
	}
	unpinned, err := s.repo.Unpin(ctx, oid)
	return unpinned, translateNotFound(err)
}

// DeletePost retire un post. Deux comportements selon l'acteur :
//   - l'auteur supprime SON post → suppression définitive (hard) + purge des
//     likes/commentaires/reposts/signets et des notifications associées ;
//   - un modérateur/admin retire le post d'un AUTRE utilisateur → suppression
//     douce (masquage) : le post sort des fils mais file dans la corbeille de
//     modération (partagée mod/admin), restaurable. Rien n'est purgé tant qu'il
//     n'est pas effacé définitivement (PurgePost).
func (s *PostService) DeletePost(ctx context.Context, id, actorID, actorRole string) error {
	oid, err := parseID(id)
	if err != nil {
		return err
	}
	post, err := s.repo.Get(ctx, oid)
	if err != nil {
		return translateNotFound(err)
	}
	if !canModify(post, actorID, actorRole) {
		return ErrForbidden
	}
	// Auto-suppression (par l'auteur lui-même) = définitive. Retrait par la
	// modération (acteur ≠ auteur) = masquage réversible vers la corbeille.
	if post.AuthorID == actorID {
		return s.hardDeletePost(ctx, oid, id, actorID)
	}
	_, err = s.repo.Hide(ctx, oid, actorID, time.Now())
	return translateNotFound(err)
}

// hardDeletePost efface DÉFINITIVEMENT un post et purge en cascade ses
// dépendances (likes/commentaires/reposts/signets) + les notifications qui le
// pointent. Mutualisé entre l'auto-suppression et la purge de modération.
func (s *PostService) hardDeletePost(ctx context.Context, oid bson.ObjectID, id, actorID string) error {
	if err := s.repo.Delete(ctx, oid); err != nil {
		return translateNotFound(err)
	}
	_ = s.repo.DeleteLikesByPost(ctx, id)
	_ = s.repo.DeleteCommentsByPost(ctx, id)
	_ = s.repo.DeleteRepostsByPost(ctx, id)
	_ = s.repo.DeletePollVotesByPost(ctx, id)
	_ = s.repo.DeleteBookmarksByPost(ctx, id)
	// Purge en cascade les notifications pointant vers ce post (likes,
	// commentaires, mentions) — plus de notification orpheline vers un post mort.
	s.notif.Emit(notifier.Event{
		Type:    notifier.EventPostDeleted,
		ActorID: actorID,
		PostID:  id,
	})
	return nil
}

// ListHiddenPosts renvoie la corbeille de modération (posts masqués par
// suppression douce), réservée aux modérateurs/admins. Du plus récemment
// masqué au plus ancien.
func (s *PostService) ListHiddenPosts(ctx context.Context, actorRole string, limit, offset int64) ([]models.Post, error) {
	if !isModerator(actorRole) {
		return nil, ErrForbidden
	}
	posts, err := s.repo.ListHidden(ctx, clampLimit(limit), clampOffset(offset))
	if err != nil {
		return nil, err
	}
	// Calcule la date de purge prévue (transient) pour l'affichage « bientôt
	// purgé » côté modération. Sans rétention configurée, pas de date.
	if s.purgeAfter > 0 {
		for i := range posts {
			if posts[i].HiddenAt != nil {
				at := posts[i].HiddenAt.Add(s.purgeAfter)
				posts[i].PurgeAt = &at
			}
		}
	}
	return posts, nil
}

// SweepPurge effectue un passage de purge RGPD : notifie les auteurs des tweets
// entrant dans la fenêtre de préavis, puis purge définitivement ceux dont la
// rétention est dépassée. Renvoie (prévenus, purgés). No-op si rétention <= 0.
func (s *PostService) SweepPurge(ctx context.Context) (warned, purged int, err error) {
	if s.purgeAfter <= 0 {
		return 0, 0, nil
	}
	now := time.Now()
	purgeBefore, warnBefore := purgeCutoffs(now, s.purgeAfter, s.purgeWarnBefore)

	// Préavis (avant la purge, pour ne pas notifier un post qu'on efface dans le
	// même passage).
	warnable, err := s.repo.ListPurgeWarnable(ctx, warnBefore, MaxLimit)
	if err != nil {
		return 0, 0, err
	}
	for i := range warnable {
		p := &warnable[i]
		s.notif.Emit(notifier.Event{
			Type:        notifier.EventPostPurgeWarning,
			RecipientID: p.AuthorID,
			PostID:      p.ID.Hex(),
		})
		if e := s.repo.MarkPurgeWarned(ctx, p.ID, now); e == nil {
			warned++
		}
	}

	// Purge définitive.
	purgeable, err := s.repo.ListPurgeable(ctx, purgeBefore, MaxLimit)
	if err != nil {
		return warned, 0, err
	}
	for i := range purgeable {
		p := &purgeable[i]
		if e := s.hardDeletePost(ctx, p.ID, p.ID.Hex(), ""); e == nil {
			purged++
		}
	}
	return warned, purged, nil
}

// RunPurgeSweeper lance le balayage périodique jusqu'à annulation du contexte.
// À appeler dans une goroutine au démarrage. Ne fait rien si rétention <= 0.
func (s *PostService) RunPurgeSweeper(ctx context.Context, interval time.Duration) {
	if s.purgeAfter <= 0 || interval <= 0 {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		// Un premier passage immédiat, puis à chaque tick.
		if warned, purged, err := s.SweepPurge(ctx); err != nil {
			log.Printf("[purge] balayage : %v", err)
		} else if warned > 0 || purged > 0 {
			log.Printf("[purge] %d préavis, %d tweets purgés", warned, purged)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// RestorePost lève le masquage d'un post depuis la corbeille (modérateur/admin).
// Pas de confirmation : action réversible et non destructive.
func (s *PostService) RestorePost(ctx context.Context, id, actorRole string) (*models.Post, error) {
	if !isModerator(actorRole) {
		return nil, ErrForbidden
	}
	oid, err := parseID(id)
	if err != nil {
		return nil, err
	}
	post, err := s.repo.RestoreHidden(ctx, oid)
	return post, translateNotFound(err)
}

// AutoHide / AutoUnhide pilotent l'auto-masquage d'un post déclenché par le
// report-service (seuil de signalements). Appelés en serveur-à-serveur via les
// endpoints internes (authentifiés par secret partagé) — PAS de contrôle de rôle
// ici : la garde est le secret interne. mongo.ErrNoDocuments → 404 plus haut.
func (s *PostService) AutoHide(ctx context.Context, id string) (*models.Post, error) {
	oid, err := parseID(id)
	if err != nil {
		return nil, err
	}
	post, err := s.repo.SetAutoHidden(ctx, oid, true)
	return post, translateNotFound(err)
}

func (s *PostService) AutoUnhide(ctx context.Context, id string) (*models.Post, error) {
	oid, err := parseID(id)
	if err != nil {
		return nil, err
	}
	post, err := s.repo.SetAutoHidden(ctx, oid, false)
	return post, translateNotFound(err)
}

// PurgeUserData efface DÉFINITIVEMENT toutes les données d'un utilisateur
// (effacement RGPD, réservé admin). Posts, commentaires, likes, reposts et
// signets. Irréversible.
func (s *PostService) PurgeUserData(ctx context.Context, userID, actorRole string) (int64, error) {
	if actorRole != models.RoleAdmin {
		return 0, ErrForbidden
	}
	return s.repo.PurgeByAuthor(ctx, userID)
}

// PurgePost efface DÉFINITIVEMENT un post depuis la corbeille de modération
// (modérateur/admin) + cascade. Irréversible.
func (s *PostService) PurgePost(ctx context.Context, id, actorID, actorRole string) error {
	if !isModerator(actorRole) {
		return ErrForbidden
	}
	oid, err := parseID(id)
	if err != nil {
		return err
	}
	if _, err := s.repo.Get(ctx, oid); err != nil {
		return translateNotFound(err)
	}
	return s.hardDeletePost(ctx, oid, id, actorID)
}

// GetByProfile renvoie les posts d'un auteur, du plus récent au plus ancien.
func (s *PostService) GetByProfile(ctx context.Context, authorID, viewerID, hashtag string, limit, offset int64) ([]models.Post, error) {
	allowed, err := s.canReadAuthor(ctx, viewerID, authorID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return []models.Post{}, nil
	}
	posts, err := s.repo.GetByProfileHashtag(ctx, authorID, normalizeHashtag(hashtag), clampLimit(limit), clampOffset(offset))
	if err != nil {
		return nil, err
	}
	s.hydratePolls(ctx, posts, viewerID)
	s.hydrateReplyPermissions(ctx, posts, viewerID)
	return posts, nil
}

// GetFeed renvoie les posts d'un ensemble d'auteurs (fil « Abonnements »). Le
// front fournit les ids suivis (seul le user-service connaît le graphe) ; la
// sélection + le tri + la pagination sont faits côté DB ($in indexé). Liste
// vide → aucun post (pas de requête inutile).
func (s *PostService) GetFeed(ctx context.Context, authorIDs []string, viewerID, hashtag, sortMode string, limit, offset int64) ([]models.Post, error) {
	if len(authorIDs) == 0 {
		return []models.Post{}, nil
	}
	hashtag = normalizeHashtag(hashtag)
	sortMode = normalizePostSort(sortMode)
	posts, err := s.visibleFeedPage(ctx, viewerID, limit, offset, func(ctx context.Context, pageLimit, pageOffset int64) ([]models.Post, error) {
		return s.repo.GetByAuthorsHashtag(ctx, authorIDs, hashtag, sortMode, pageLimit, pageOffset)
	})
	if err != nil {
		return nil, err
	}
	s.hydratePolls(ctx, posts, viewerID)
	s.hydrateReplyPermissions(ctx, posts, viewerID)
	return posts, nil
}

func (s *PostService) VotePoll(ctx context.Context, postID, actorID, choiceID string) (*models.Post, error) {
	oid, err := parseID(postID)
	if err != nil {
		return nil, err
	}
	post, err := s.repo.Get(ctx, oid)
	if err != nil {
		return nil, translateNotFound(err)
	}
	if post.Poll == nil {
		return nil, ErrInvalidPoll
	}
	if pollIsClosed(post.Poll, time.Now()) {
		return nil, ErrPollClosed
	}
	allowed, err := s.canReadAuthor(ctx, actorID, post.AuthorID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrPrivateProfil
	}
	if post.Poll.Audience == models.PollAudienceFollowers && actorID != post.AuthorID {
		if s.followClient == nil {
			return nil, ErrForbidden
		}
		follows, err := s.followClient.IsFollowing(ctx, actorID, post.AuthorID)
		if err != nil {
			return nil, fmt.Errorf("%w: vérification abonnement: %v", ErrDependencyUnavailable, err)
		}
		if !follows {
			return nil, ErrForbidden
		}
	}
	if !pollHasChoice(post.Poll, choiceID) {
		return nil, ErrInvalidPoll
	}
	created, err := s.repo.AddPollVote(ctx, postID, actorID, choiceID)
	if err != nil {
		return nil, err
	}
	if !created {
		s.hydratePoll(ctx, post, actorID)
		s.hydrateReplyPermission(ctx, post, actorID, "")
		return post, ErrPollAlreadyVoted
	}
	updated, err := s.repo.IncPollChoice(ctx, oid, choiceID)
	if err != nil {
		return nil, translateNotFound(err)
	}
	hydratePoll(updated, actorID, choiceID)
	s.hydrateReplyPermission(ctx, updated, actorID, "")
	return updated, nil
}

func (s *PostService) ClosePoll(ctx context.Context, postID, actorID string) (*models.Post, error) {
	oid, err := parseID(postID)
	if err != nil {
		return nil, err
	}
	post, err := s.repo.Get(ctx, oid)
	if err != nil {
		return nil, translateNotFound(err)
	}
	if post.Poll == nil {
		return nil, ErrInvalidPoll
	}
	if post.AuthorID != actorID {
		return nil, ErrForbidden
	}
	if pollIsClosed(post.Poll, time.Now()) {
		s.hydratePoll(ctx, post, actorID)
		s.hydrateReplyPermission(ctx, post, actorID, "")
		return post, nil
	}
	closed, err := s.repo.ClosePoll(ctx, oid, time.Now())
	if err != nil {
		return nil, translateNotFound(err)
	}
	hydratePoll(closed, actorID, "")
	s.hydrateReplyPermission(ctx, closed, actorID, "")
	return closed, nil
}

// TrendingHashtags compte les hashtags des posts lisibles par le visiteur courant.
// query filtre optionnellement sur un préfixe normalisé, utile pour les suggestions.
func (s *PostService) TrendingHashtags(ctx context.Context, viewerID, query string, limit int64) ([]models.HashtagTrend, error) {
	limit = clampLimit(limit)
	query = normalizeTrendQuery(query)
	counts := make(map[string]int64)
	allowedByAuthor := make(map[string]bool)
	sourceOffset := int64(0)

	for {
		batch, err := s.repo.GetAll(ctx, MaxLimit, sourceOffset)
		if err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			break
		}
		for _, post := range batch {
			allowed, ok := allowedByAuthor[post.AuthorID]
			if !ok {
				allowed, err = s.canReadAuthor(ctx, viewerID, post.AuthorID)
				if err != nil {
					return nil, err
				}
				allowedByAuthor[post.AuthorID] = allowed
			}
			if !allowed {
				continue
			}
			for _, tag := range post.Hashtags {
				if !matchesTrendQuery(tag, query) {
					continue
				}
				counts[tag]++
			}
		}
		if int64(len(batch)) < MaxLimit {
			break
		}
		sourceOffset += MaxLimit
	}

	trends := make([]models.HashtagTrend, 0, len(counts))
	for tag, count := range counts {
		trends = append(trends, models.HashtagTrend{Tag: tag, Count: count})
	}
	sort.Slice(trends, func(i, j int) bool {
		if trends[i].Count == trends[j].Count {
			return trends[i].Tag < trends[j].Tag
		}
		return trends[i].Count > trends[j].Count
	})
	if int64(len(trends)) > limit {
		trends = trends[:limit]
	}
	return trends, nil
}

type postFetcher func(ctx context.Context, limit, offset int64) ([]models.Post, error)

func (s *PostService) visibleFeedPage(ctx context.Context, viewerID string, limit, offset int64, fetch postFetcher) ([]models.Post, error) {
	posts, err := s.visiblePage(ctx, viewerID, limit, offset, fetch)
	if err != nil {
		return nil, err
	}
	return withoutProfilePins(posts), nil
}

func (s *PostService) visiblePage(ctx context.Context, viewerID string, limit, offset int64, fetch postFetcher) ([]models.Post, error) {
	limit = clampLimit(limit)
	offset = clampOffset(offset)

	visible := make([]models.Post, 0, limit)
	seenVisible := int64(0)
	sourceOffset := int64(0)
	allowedByAuthor := make(map[string]bool)

	for int64(len(visible)) < limit {
		batch, err := fetch(ctx, MaxLimit, sourceOffset)
		if err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			break
		}

		for _, post := range batch {
			allowed, ok := allowedByAuthor[post.AuthorID]
			if !ok {
				allowed, err = s.canReadAuthor(ctx, viewerID, post.AuthorID)
				if err != nil {
					return nil, err
				}
				allowedByAuthor[post.AuthorID] = allowed
			}
			if !allowed {
				continue
			}
			if seenVisible < offset {
				seenVisible++
				continue
			}
			visible = append(visible, post)
			if int64(len(visible)) == limit {
				break
			}
		}

		if int64(len(batch)) < MaxLimit {
			break
		}
		sourceOffset += MaxLimit
	}

	return visible, nil
}

func (s *PostService) canReadAuthor(ctx context.Context, viewerID, authorID string) (bool, error) {
	if viewerID != "" && viewerID == authorID {
		return true, nil
	}
	if s.profilClient == nil {
		return true, nil
	}

	visibility, err := s.profilClient.Visibility(ctx, authorID)
	if err != nil {
		return false, fmt.Errorf("%w: vérification visibilité: %v", ErrDependencyUnavailable, err)
	}
	if visibility != client.VisibilityPrivate {
		return true, nil
	}
	if viewerID == "" || s.followClient == nil {
		return false, nil
	}

	isFollowing, err := s.followClient.IsFollowing(ctx, viewerID, authorID)
	if err != nil {
		return false, fmt.Errorf("%w: vérification abonnement: %v", ErrDependencyUnavailable, err)
	}
	return isFollowing, nil
}

// LikePost enregistre un like de actorID sur un post et renvoie le nombre de
// likes à jour. Idempotent : reliker ne double pas le compteur.
func (s *PostService) LikePost(ctx context.Context, id, actorID string) (int32, error) {
	oid, err := parseID(id)
	if err != nil {
		return 0, err
	}
	post, err := s.repo.Get(ctx, oid)
	if err != nil {
		return 0, translateNotFound(err)
	}
	created, err := s.repo.AddLike(ctx, id, actorID)
	if err != nil {
		return 0, err
	}
	if !created {
		return post.LikesCount, nil
	}
	updated, err := s.repo.IncCounter(ctx, oid, "likes_count", 1)
	if err != nil {
		return 0, err
	}
	// Notifie l'auteur du post (agrégé : 300 likes = une seule notification).
	s.notif.Emit(notifier.Event{
		Type:        notifier.TypeLike,
		ActorID:     actorID,
		RecipientID: post.AuthorID,
		PostID:      id,
	})
	return updated.LikesCount, nil
}

// UnlikePost retire le like de actorID et renvoie le nombre de likes à jour.
// Idempotent : déliker un post non liké ne décrémente pas.
func (s *PostService) UnlikePost(ctx context.Context, id, actorID string) (int32, error) {
	oid, err := parseID(id)
	if err != nil {
		return 0, err
	}
	post, err := s.repo.Get(ctx, oid)
	if err != nil {
		return 0, translateNotFound(err)
	}
	removed, err := s.repo.RemoveLike(ctx, id, actorID)
	if err != nil {
		return 0, err
	}
	if !removed {
		return post.LikesCount, nil
	}
	updated, err := s.repo.IncCounter(ctx, oid, "likes_count", -1)
	if err != nil {
		return 0, err
	}
	// Défait la notification de like correspondante (décrément / suppression).
	s.notif.Emit(notifier.Event{
		Type:        notifier.TypeLike,
		ActorID:     actorID,
		RecipientID: post.AuthorID,
		PostID:      id,
		Retract:     true,
	})
	return updated.LikesCount, nil
}

// LikedPostIDs renvoie les ids des posts likés par actorID.
func (s *PostService) LikedPostIDs(ctx context.Context, actorID string) ([]string, error) {
	return s.repo.LikedPostIDs(ctx, actorID)
}

// LikeComment enregistre un like de actorID sur un commentaire et renvoie le
// nombre de likes à jour. Idempotent : reliker ne double pas le compteur.
func (s *PostService) LikeComment(ctx context.Context, postID, commentID, actorID string) (int32, error) {
	if _, err := parseID(postID); err != nil {
		return 0, err
	}
	coid, err := parseID(commentID)
	if err != nil {
		return 0, err
	}
	comment, err := s.repo.GetComment(ctx, coid)
	if err != nil {
		return 0, translateNotFound(err)
	}
	if comment.PostID != postID {
		return 0, ErrPostNotFound
	}
	created, err := s.repo.AddCommentLike(ctx, commentID, actorID)
	if err != nil {
		return 0, err
	}
	if !created {
		return comment.LikesCount, nil
	}
	updated, err := s.repo.IncCommentCounter(ctx, coid, "likes_count", 1)
	if err != nil {
		return 0, err
	}
	// Notifie l'auteur du commentaire (agrégé par commentaire ; auto-like filtré
	// côté notification-service via recipient == actor).
	s.notif.Emit(notifier.Event{
		Type:        notifier.TypeCommentLike,
		ActorID:     actorID,
		RecipientID: comment.AuthorID,
		PostID:      postID,
		CommentID:   commentID,
	})
	return updated.LikesCount, nil
}

// UnlikeComment retire le like de actorID sur un commentaire. Idempotent :
// déliker un commentaire non liké ne décrémente pas.
func (s *PostService) UnlikeComment(ctx context.Context, postID, commentID, actorID string) (int32, error) {
	if _, err := parseID(postID); err != nil {
		return 0, err
	}
	coid, err := parseID(commentID)
	if err != nil {
		return 0, err
	}
	comment, err := s.repo.GetComment(ctx, coid)
	if err != nil {
		return 0, translateNotFound(err)
	}
	if comment.PostID != postID {
		return 0, ErrPostNotFound
	}
	removed, err := s.repo.RemoveCommentLike(ctx, commentID, actorID)
	if err != nil {
		return 0, err
	}
	if !removed {
		return comment.LikesCount, nil
	}
	updated, err := s.repo.IncCommentCounter(ctx, coid, "likes_count", -1)
	if err != nil {
		return 0, err
	}
	// Défait la notification de like de commentaire correspondante.
	s.notif.Emit(notifier.Event{
		Type:        notifier.TypeCommentLike,
		ActorID:     actorID,
		RecipientID: comment.AuthorID,
		PostID:      postID,
		CommentID:   commentID,
		Retract:     true,
	})
	return updated.LikesCount, nil
}

// CommentStats renvoie les compteurs de likes des commentaires demandés. Les
// ids invalides/introuvables sont simplement ignorés.
func (s *PostService) CommentStats(ctx context.Context, ids []string) ([]models.CommentStat, error) {
	if len(ids) > MaxStatsIDs {
		ids = ids[:MaxStatsIDs]
	}
	oids := make([]bson.ObjectID, 0, len(ids))
	for _, id := range ids {
		if oid, err := parseID(id); err == nil {
			oids = append(oids, oid)
		}
	}
	comments, err := s.repo.CommentStatsByIDs(ctx, oids)
	if err != nil {
		return nil, err
	}
	stats := make([]models.CommentStat, 0, len(comments))
	for _, comment := range comments {
		stats = append(stats, models.CommentStat{
			ID:         comment.ID.Hex(),
			LikesCount: comment.LikesCount,
		})
	}
	return stats, nil
}

// PostLikers renvoie les ids des utilisateurs ayant liké un post.
func (s *PostService) PostLikers(ctx context.Context, id string) ([]string, error) {
	if _, err := parseID(id); err != nil {
		return nil, err
	}
	return s.repo.LikersByPost(ctx, id)
}

// ListLikedByUser retourne les posts likés par authorID, triés du like le plus
// récent au plus ancien, paginés. callerID peut être vide (visiteur).
// Retourne ErrForbidden si les likes de authorID sont privés et que callerID
// n'est pas authorID.
func (s *PostService) ListLikedByUser(ctx context.Context, authorID, callerID string, limit, offset int64) ([]*models.Post, error) {
	if s.profilClient != nil && callerID != authorID {
		lv, err := s.profilClient.LikesVisibility(ctx, authorID)
		if err != nil {
			return nil, fmt.Errorf("%w: vérification likes-visibility: %v", ErrDependencyUnavailable, err)
		}
		if lv == client.VisibilityPrivate {
			return nil, ErrForbidden
		}
	}
	return s.repo.LikedPostsByUser(ctx, authorID, clampLimit(limit), clampOffset(offset))
}

// RepostPost enregistre un repost simple de actorID sur un post et renvoie le
// post original annoté (`reposted_by_id`, `reposted_at`) pour l'affichage profil.
func (s *PostService) RepostPost(ctx context.Context, id, actorID string) (*models.Post, error) {
	oid, err := parseID(id)
	if err != nil {
		return nil, err
	}
	post, err := s.repo.Get(ctx, oid)
	if err != nil {
		return nil, translateNotFound(err)
	}
	repost, created, err := s.repo.AddRepost(ctx, id, actorID)
	if err != nil {
		return nil, err
	}
	if created {
		post, err = s.repo.IncCounter(ctx, oid, "reposts_count", 1)
		if err != nil {
			return nil, err
		}
		// Repost = tag implicite de l'auteur → notif agrégée (« X et N autres
		// ont reposté votre publication »), navigation vers le post original.
		s.notif.Emit(notifier.Event{
			Type:        notifier.TypeRepost,
			ActorID:     actorID,
			RecipientID: post.AuthorID,
			PostID:      id,
		})
	}
	post.RepostedByID = actorID
	post.RepostedAt = &repost.CreatedAt
	return post, nil
}

// UnrepostPost retire un repost simple. Idempotent.
func (s *PostService) UnrepostPost(ctx context.Context, id, actorID string) (int32, error) {
	oid, err := parseID(id)
	if err != nil {
		return 0, err
	}
	post, err := s.repo.Get(ctx, oid)
	if err != nil {
		return 0, translateNotFound(err)
	}
	removed, err := s.repo.RemoveRepost(ctx, id, actorID)
	if err != nil {
		return 0, err
	}
	if !removed {
		return post.RepostsCount, nil
	}
	updated, err := s.repo.IncCounter(ctx, oid, "reposts_count", -1)
	if err != nil {
		return 0, err
	}
	// Défait la notification de repost correspondante (décrément / suppression).
	s.notif.Emit(notifier.Event{
		Type:        notifier.TypeRepost,
		ActorID:     actorID,
		RecipientID: post.AuthorID,
		PostID:      id,
		Retract:     true,
	})
	return updated.RepostsCount, nil
}

func (s *PostService) RepostedPostIDs(ctx context.Context, actorID string) ([]string, error) {
	return s.repo.RepostedPostIDs(ctx, actorID)
}

// CreateComment ajoute un commentaire (auteur dérivé du JWT) sur un post
// existant et incrémente son compteur. Si `parentID` est fourni, c'est une
// réponse : elle est rattachée à plat au commentaire RACINE (cf. resolveParentID,
// threading à 2 niveaux) et incrémente le `reply_count` de cette racine.
func (s *PostService) CreateComment(ctx context.Context, postID, authorID, authorRole, content, parentID string, media []models.MediaRef) (*models.Comment, error) {
	oid, err := parseID(postID)
	if err != nil {
		return nil, err
	}
	post, err := s.repo.Get(ctx, oid)
	if err != nil {
		return nil, translateNotFound(err)
	}

	// Barrière « qui peut répondre » : si le post réserve les réponses aux abonnés,
	// seuls l'auteur, ses abonnés et les modérateurs/admins peuvent commenter.
	allowed, err := s.canReplyTo(ctx, post, authorID, authorRole)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrReplyNotAllowed
	}

	rootID := ""
	rootAuthorID := "" // auteur du commentaire racine du fil (destinataire de la notif « réponse »)
	if parentID != "" {
		pcoid, err := parseID(parentID)
		if err != nil {
			return nil, err
		}
		parent, err := s.repo.GetComment(ctx, pcoid)
		if err != nil {
			return nil, translateNotFound(err)
		}
		if parent.PostID != postID {
			return nil, ErrPostNotFound // parent rattaché à un autre post
		}
		rootID = resolveParentID(parent, parentID)
		// Le threading est à plat sous la racine : la notif de réponse va à
		// l'auteur de la RACINE du fil (clé d'agrégation = la racine), ce qui
		// reste cohérent à la suppression (où seul `parent_id`=racine est stocké).
		if parent.ParentID == "" {
			rootAuthorID = parent.AuthorID // parent est déjà la racine
		} else {
			rootAuthorID = s.commentAuthor(ctx, rootID)
		}
	}

	now := time.Now()
	comment := &models.Comment{
		PostID:     postID,
		ParentID:   rootID,
		AuthorID:   authorID,
		Content:    content,
		Media:      media,
		LikesCount: 0,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.repo.AddComment(ctx, comment); err != nil {
		return nil, err
	}
	if _, err := s.repo.IncCounter(ctx, oid, "comments_count", 1); err != nil {
		return nil, err
	}
	if rootID != "" {
		if rcoid, err := parseID(rootID); err == nil {
			_ = s.repo.IncReplyCount(ctx, rcoid, 1)
		}
	}
	s.emitCommentEvents(post.AuthorID, rootID, rootAuthorID, authorID, postID, comment.ID.Hex(), content, false)
	return comment, nil
}

// emitCommentEvents émet les notifications liées à un commentaire (à sa création
// comme à sa suppression, via `retract`). Règles :
//   - commentaire RACINE (`rootID` vide) → notifie l'auteur du POST ;
//   - RÉPONSE (`rootID` non vide) → notifie l'auteur du commentaire RACINE du fil
//     (« réponse à ton commentaire »), agrégée par cette racine ;
//   - mentions (@handle) dans le contenu → notifie les mentionnés.
//
// `commentID` est l'id du commentaire lui-même (source d'agrégation des mentions,
// stable entre création et suppression).
func (s *PostService) emitCommentEvents(postAuthorID, rootID, rootAuthorID, actorID, postID, commentID, content string, retract bool) {
	if rootID == "" {
		// Commentaire racine → l'auteur du post. `CommentID` = ce commentaire,
		// pour permettre le deep-link de la notification vers le commentaire
		// (l'agrégation reste par post, cf. groupKeyFor → `comment:<post_id>`).
		s.notif.Emit(notifier.Event{
			Type:        notifier.TypeComment,
			ActorID:     actorID,
			RecipientID: postAuthorID,
			PostID:      postID,
			CommentID:   commentID,
			Retract:     retract,
		})
	} else {
		// Réponse → l'auteur de la racine du fil (agrégée par cette racine).
		s.notif.Emit(notifier.Event{
			Type:        notifier.TypeReply,
			ActorID:     actorID,
			RecipientID: rootAuthorID,
			PostID:      postID,
			CommentID:   rootID,
			Retract:     retract,
		})
	}
	if handles := notifier.ParseMentions(content); len(handles) > 0 {
		s.notif.Emit(notifier.Event{
			Type:           notifier.TypeMention,
			ActorID:        actorID,
			PostID:         postID,
			CommentID:      commentID,
			MentionHandles: handles,
			Retract:        retract,
		})
	}
}

// commentAuthor renvoie l'auteur d'un commentaire (best-effort, "" si absent).
func (s *PostService) commentAuthor(ctx context.Context, id string) string {
	oid, err := parseID(id)
	if err != nil {
		return ""
	}
	c, err := s.repo.GetComment(ctx, oid)
	if err != nil {
		return ""
	}
	return c.AuthorID
}

// postAuthor renvoie l'auteur d'un post (best-effort, "" si absent).
func (s *PostService) postAuthor(ctx context.Context, id string) string {
	oid, err := parseID(id)
	if err != nil {
		return ""
	}
	p, err := s.repo.Get(ctx, oid)
	if err != nil {
		return ""
	}
	return p.AuthorID
}

// ListCommentsByAuthor renvoie les commentaires écrits par authorID, enrichis
// de leur post parent, filtrés par la barrière de visibilité : les réponses
// dont le post parent est masqué ou n'est pas lisible par viewerID sont exclues.
func (s *PostService) ListCommentsByAuthor(ctx context.Context, authorID, viewerID string, limit, offset int64) ([]models.CommentWithPost, error) {
	limit = clampLimit(limit)
	offset = clampOffset(offset)

	visible := make([]models.CommentWithPost, 0, limit)
	seenVisible := int64(0)
	sourceOffset := int64(0)
	allowedByAuthor := make(map[string]bool)
	postCache := make(map[string]*models.Post)       // nil = masqué ou introuvable
	commentCache := make(map[string]*models.Comment) // commentaire parent (réponses), nil = introuvable

	for int64(len(visible)) < limit {
		batch, err := s.repo.ListCommentsByAuthor(ctx, authorID, MaxLimit, sourceOffset)
		if err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			break
		}

		for _, comment := range batch {
			parent, fetched := postCache[comment.PostID]
			if !fetched {
				oid, parseErr := parseID(comment.PostID)
				if parseErr == nil {
					p, getErr := s.repo.Get(ctx, oid)
					if getErr == nil && !p.IsHidden {
						postCache[comment.PostID] = p
						parent = p
					} else {
						postCache[comment.PostID] = nil
					}
				} else {
					postCache[comment.PostID] = nil
				}
			}
			if parent == nil {
				continue
			}

			allowed, ok := allowedByAuthor[parent.AuthorID]
			if !ok {
				var chkErr error
				allowed, chkErr = s.canReadAuthor(ctx, viewerID, parent.AuthorID)
				if chkErr != nil {
					return nil, chkErr
				}
				allowedByAuthor[parent.AuthorID] = allowed
			}
			if !allowed {
				continue
			}

			if seenVisible < offset {
				seenVisible++
				continue
			}

			// Réponse à un autre commentaire → hydrate le commentaire parent
			// (post → commentaire parent → réponse). Best-effort : si introuvable
			// (supprimé), on laisse `ParentComment` nil.
			var parentComment *models.Comment
			if comment.ParentID != "" {
				cached, ok := commentCache[comment.ParentID]
				if !ok {
					cached = nil
					if cid, parseErr := parseID(comment.ParentID); parseErr == nil {
						if pc, getErr := s.repo.GetComment(ctx, cid); getErr == nil {
							cached = pc
						}
					}
					commentCache[comment.ParentID] = cached
				}
				parentComment = cached
			}

			visible = append(visible, models.CommentWithPost{Comment: comment, ParentPost: parent, ParentComment: parentComment})
			if int64(len(visible)) == limit {
				break
			}
		}

		if int64(len(batch)) < MaxLimit {
			break
		}
		sourceOffset += MaxLimit
	}

	s.hydrateCommentWithPostLikes(ctx, visible, viewerID)
	return visible, nil
}

// ListComments renvoie les commentaires RACINE d'un post (chronologiques, paginés).
func (s *PostService) ListComments(ctx context.Context, postID, viewerID string, limit, offset int64) ([]models.Comment, error) {
	if _, err := parseID(postID); err != nil {
		return nil, err
	}
	comments, err := s.repo.ListComments(ctx, postID, clampLimit(limit), clampOffset(offset))
	if err != nil {
		return nil, err
	}
	s.hydrateCommentLikes(ctx, comments, viewerID)
	return comments, nil
}

// ListReplies renvoie les réponses d'un commentaire (chronologiques, paginées).
func (s *PostService) ListReplies(ctx context.Context, commentID, viewerID string, limit, offset int64) ([]models.Comment, error) {
	if _, err := parseID(commentID); err != nil {
		return nil, err
	}
	comments, err := s.repo.ListReplies(ctx, commentID, clampLimit(limit), clampOffset(offset))
	if err != nil {
		return nil, err
	}
	s.hydrateCommentLikes(ctx, comments, viewerID)
	return comments, nil
}

// DeleteComment supprime un commentaire si l'acteur en a le droit (auteur du
// commentaire ou modérateur/admin), ajuste les compteurs et, pour un
// commentaire racine, supprime ses réponses en cascade.
func (s *PostService) DeleteComment(ctx context.Context, commentID, actorID, actorRole string) error {
	coid, err := parseID(commentID)
	if err != nil {
		return err
	}
	comment, err := s.repo.GetComment(ctx, coid)
	if err != nil {
		return translateNotFound(err)
	}
	if !canAct(comment.AuthorID, actorID, actorRole) {
		return ErrForbidden
	}
	if err := s.repo.DeleteComment(ctx, coid); err != nil {
		return translateNotFound(err)
	}

	removed := int32(1)
	if comment.ParentID == "" {
		// Commentaire racine : cascade des réponses.
		replyIDs, _ := s.repo.CommentIDsByParent(ctx, commentID)
		n, _ := s.repo.DeleteRepliesByParent(ctx, commentID)
		_ = s.repo.DeleteCommentLikesByComments(ctx, replyIDs)
		removed += int32(n)
	} else if rcoid, err := parseID(comment.ParentID); err == nil {
		// Réponse : décrémente le compteur de réponses de la racine.
		_ = s.repo.IncReplyCount(ctx, rcoid, -1)
	}
	_ = s.repo.DeleteCommentLikesByComment(ctx, commentID)

	// Décrémente le compteur du post (id pris sur le commentaire, source de
	// vérité). Best-effort : un post déjà supprimé n'a plus de compteur.
	if poid, err := parseID(comment.PostID); err == nil {
		_, _ = s.repo.IncCounter(ctx, poid, "comments_count", -removed)
	}

	// Défait les notifications du commentaire (symétrique de la création).
	postAuthorID := ""
	rootAuthorID := ""
	if comment.ParentID == "" {
		postAuthorID = s.postAuthor(ctx, comment.PostID)
	} else {
		rootAuthorID = s.commentAuthor(ctx, comment.ParentID)
	}
	s.emitCommentEvents(postAuthorID, comment.ParentID, rootAuthorID, comment.AuthorID, comment.PostID, commentID, comment.Content, true)
	return nil
}

func (s *PostService) hydrateCommentLikes(ctx context.Context, comments []models.Comment, viewerID string) {
	if viewerID == "" || len(comments) == 0 {
		return
	}
	ids := make([]string, 0, len(comments))
	for _, comment := range comments {
		ids = append(ids, comment.ID.Hex())
	}
	likedIDs, err := s.repo.LikedCommentIDsByUser(ctx, viewerID, ids)
	if err != nil {
		return
	}
	likedSet := make(map[string]bool, len(likedIDs))
	for _, id := range likedIDs {
		likedSet[id] = true
	}
	for i := range comments {
		comments[i].Liked = likedSet[comments[i].ID.Hex()]
	}
}

func (s *PostService) hydrateCommentWithPostLikes(ctx context.Context, items []models.CommentWithPost, viewerID string) {
	if viewerID == "" || len(items) == 0 {
		return
	}
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID.Hex())
	}
	likedIDs, err := s.repo.LikedCommentIDsByUser(ctx, viewerID, ids)
	if err != nil {
		return
	}
	likedSet := make(map[string]bool, len(likedIDs))
	for _, id := range likedIDs {
		likedSet[id] = true
	}
	for i := range items {
		items[i].Liked = likedSet[items[i].ID.Hex()]
	}
}

// resolveParentID : threading à 2 niveaux — répondre à une réponse rattache la
// nouvelle réponse à la RACINE (parent.ParentID), pas à la réponse elle-même.
// Répondre à un commentaire racine garde son id. Fonction PURE (testée).
func resolveParentID(parent *models.Comment, requestedID string) string {
	if parent.ParentID != "" {
		return parent.ParentID
	}
	return requestedID
}

// canModify : un post n'est modifiable/supprimable que par son auteur ou par un
// modérateur / administrateur. Fonction PURE (testée unitairement).
func canModify(post *models.Post, actorID, actorRole string) bool {
	if post == nil {
		return false
	}
	return canAct(post.AuthorID, actorID, actorRole)
}

// canPin : l'épinglage est une action de personnalisation du profil, réservée
// à l'auteur du post.
func canPin(post *models.Post, actorID string) bool {
	return post != nil && post.AuthorID == actorID
}

// canAct : règle d'autorisation commune (posts ET commentaires) — l'auteur, un
// modérateur ou un admin peut agir. Fonction PURE.
func canAct(authorID, actorID, actorRole string) bool {
	return authorID == actorID || isModerator(actorRole)
}

// isModerator : un modérateur OU un administrateur (l'admin est un sur-ensemble
// du modérateur). Sert aux actions de modération (corbeille, restauration,
// purge). Fonction PURE.
func isModerator(role string) bool {
	return role == models.RoleModerator || role == models.RoleAdmin
}

// purgeCutoffs calcule les bornes de balayage RGPD à partir de `now` :
//   - purge : un tweet masqué avant cette date est purgé (hidden_at < now-after) ;
//   - warn  : avant cette date, on entre dans la fenêtre de préavis
//     (hidden_at < now-(after-warnBefore)).
//
// warn est forcément >= purge (la fenêtre de préavis précède la purge). PURE.
func purgeCutoffs(now time.Time, after, warnBefore time.Duration) (purge, warn time.Time) {
	return now.Add(-after), now.Add(-(after - warnBefore))
}

// withoutProfilePins masque l'état d'épinglage dans les feeds publics. Le pin
// reste une information de profil, exposée uniquement par GetByProfile.
func withoutProfilePins(posts []models.Post) []models.Post {
	if len(posts) == 0 {
		return posts
	}
	cleaned := make([]models.Post, len(posts))
	copy(cleaned, posts)
	for i := range cleaned {
		cleaned[i].PinnedAt = nil
	}
	return cleaned
}

func buildPoll(req *models.CreatePollRequest, now time.Time) (*models.Poll, error) {
	if req == nil {
		return nil, nil
	}
	if req.DurationMinutes < 1 || req.DurationMinutes > 7*24*60 {
		return nil, ErrInvalidPoll
	}
	audience := req.Audience
	if audience == "" {
		audience = models.PollAudienceEveryone
	}
	if audience != models.PollAudienceEveryone && audience != models.PollAudienceFollowers {
		return nil, ErrInvalidPoll
	}
	seen := make(map[string]bool, len(req.Choices))
	choices := make([]models.PollChoice, 0, len(req.Choices))
	for _, raw := range req.Choices {
		label := strings.TrimSpace(raw.Label)
		if label == "" || len([]rune(label)) > 80 {
			return nil, ErrInvalidPoll
		}
		key := strings.ToLower(label)
		if seen[key] {
			return nil, ErrInvalidPoll
		}
		seen[key] = true
		image := strings.TrimSpace(raw.ImageURL)
		if len(image) > 512 {
			return nil, ErrInvalidPoll
		}
		choices = append(choices, models.PollChoice{
			ID:         bson.NewObjectID().Hex(),
			Label:      label,
			VotesCount: 0,
			ImageURL:   image,
		})
	}
	if len(choices) < 2 || len(choices) > 4 {
		return nil, ErrInvalidPoll
	}
	return &models.Poll{
		Choices:    choices,
		EndsAt:     now.Add(time.Duration(req.DurationMinutes) * time.Minute),
		Audience:   audience,
		TotalVotes: 0,
	}, nil
}

func pollHasChoice(poll *models.Poll, choiceID string) bool {
	if poll == nil {
		return false
	}
	for _, choice := range poll.Choices {
		if choice.ID == choiceID {
			return true
		}
	}
	return false
}

// canReplyTo applique la barrière « qui peut répondre » d'un post. Renvoie `true`
// si l'audience est `everyone`, ou si le lecteur est l'auteur, un modérateur/admin,
// ou un abonné de l'auteur. Le check d'abonnement (appel user-service) n'est fait
// que pour les posts effectivement restreints aux abonnés — coût nul sinon.
func (s *PostService) canReplyTo(ctx context.Context, post *models.Post, actorID, actorRole string) (bool, error) {
	if models.ReplyAudienceOf(post) != models.ReplyAudienceFollowers {
		return true, nil
	}
	if actorID != "" && actorID == post.AuthorID {
		return true, nil
	}
	if actorRole == models.RoleModerator || actorRole == models.RoleAdmin {
		return true, nil
	}
	if actorID == "" || s.followClient == nil {
		return false, nil
	}
	follows, err := s.followClient.IsFollowing(ctx, actorID, post.AuthorID)
	if err != nil {
		return false, fmt.Errorf("%w: vérification abonnement: %v", ErrDependencyUnavailable, err)
	}
	return follows, nil
}

// hydrateReplyPermission renseigne le champ transient CanReply d'un post pour le
// lecteur courant (rôle inconnu ici → on n'accorde pas le bypass mod/admin, mais
// le serveur reste autoritaire à l'écriture via canReplyTo). Pour une restriction,
// on échoue FERMÉ : en cas d'erreur de dépendance (vérification d'abonnement
// indisponible) on masque le composer (CanReply=false) plutôt que de l'offrir à
// tort — l'écriture re-tranchera de toute façon.
func (s *PostService) hydrateReplyPermissions(ctx context.Context, posts []models.Post, viewerID string) {
	for i := range posts {
		s.hydrateReplyPermission(ctx, &posts[i], viewerID, "")
	}
}

func (s *PostService) hydrateReplyPermission(ctx context.Context, post *models.Post, viewerID, viewerRole string) {
	if post == nil {
		return
	}
	allowed, err := s.canReplyTo(ctx, post, viewerID, viewerRole)
	if err != nil {
		post.CanReply = false
		return
	}
	post.CanReply = allowed
}

func (s *PostService) hydratePolls(ctx context.Context, posts []models.Post, viewerID string) {
	for i := range posts {
		s.hydratePoll(ctx, &posts[i], viewerID)
	}
}

func (s *PostService) hydratePoll(ctx context.Context, post *models.Post, viewerID string) {
	if post == nil || post.Poll == nil {
		return
	}
	voted := ""
	if viewerID != "" {
		if choiceID, err := s.repo.PollVoteChoice(ctx, post.ID.Hex(), viewerID); err == nil {
			voted = choiceID
		}
	}
	hydratePoll(post, viewerID, voted)
}

func hydratePoll(post *models.Post, viewerID, votedChoiceID string) {
	if post == nil || post.Poll == nil {
		return
	}
	now := time.Now()
	closed := pollIsClosed(post.Poll, now)
	isAuthor := viewerID != "" && viewerID == post.AuthorID
	canViewResults := closed || isAuthor || votedChoiceID != ""
	post.Poll.VotedChoiceID = votedChoiceID
	post.Poll.CanViewResults = canViewResults
	post.Poll.CanClose = isAuthor && !closed
	maxVotes := int32(-1)
	winners := make([]string, 0, len(post.Poll.Choices))
	for _, choice := range post.Poll.Choices {
		if choice.VotesCount > maxVotes {
			maxVotes = choice.VotesCount
			winners = []string{choice.ID}
			continue
		}
		if choice.VotesCount == maxVotes {
			winners = append(winners, choice.ID)
		}
	}
	if canViewResults && closed && maxVotes > 0 {
		post.Poll.WinnerChoiceIDs = winners
	}
	if !canViewResults {
		post.Poll.TotalVotes = 0
		post.Poll.WinnerChoiceIDs = nil
		for i := range post.Poll.Choices {
			post.Poll.Choices[i].VotesCount = 0
		}
	}
}

func pollIsClosed(poll *models.Poll, now time.Time) bool {
	return poll != nil && (poll.ClosedAt != nil || !now.Before(poll.EndsAt))
}

// ExtractHashtags normalise les hashtags d'un texte : minuscules, sans #, uniques.
func ExtractHashtags(content string) []string {
	matches := hashtagPattern.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(matches))
	tags := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		tag := normalizeHashtag(match[2])
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		tags = append(tags, tag)
	}
	return tags
}

func normalizeHashtag(raw string) string {
	tag := strings.TrimSpace(strings.TrimPrefix(raw, "#"))
	tag = strings.ToLower(tag)
	if tag == "" || !hasLetter(tag) {
		return ""
	}
	return tag
}

func normalizeTrendQuery(raw string) string {
	return strings.ToLower(strings.TrimPrefix(strings.TrimSpace(raw), "#"))
}

func matchesTrendQuery(tag, query string) bool {
	return query == "" || strings.HasPrefix(tag, query)
}

func normalizePostSort(raw string) string {
	if strings.ToLower(strings.TrimSpace(raw)) == "top" {
		return "top"
	}
	return "recent"
}

func hasLetter(value string) bool {
	for _, r := range value {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

// parseID valide qu'un id est bien un ObjectID hexadécimal.
func parseID(id string) (bson.ObjectID, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return bson.ObjectID{}, ErrInvalidID
	}
	return oid, nil
}

// translateNotFound mappe l'absence de document Mongo vers ErrPostNotFound ;
// laisse les autres erreurs (et nil) intactes.
func translateNotFound(err error) error {
	if errors.Is(err, mongo.ErrNoDocuments) {
		return ErrPostNotFound
	}
	return err
}

// clampLimit borne la taille de page (défaut 20, max 100).
func clampLimit(limit int64) int64 {
	if limit <= 0 {
		return DefaultLimit
	}
	if limit > MaxLimit {
		return MaxLimit
	}
	return limit
}

// clampOffset interdit un décalage négatif.
func clampOffset(offset int64) int64 {
	if offset < 0 {
		return 0
	}
	return offset
}
