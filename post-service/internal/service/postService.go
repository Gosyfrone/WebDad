// Package service porte la logique métier du post-service : validation des
// identifiants, contrôle de propriété et traduction des erreurs du dépôt en
// erreurs métier (mappées vers des codes HTTP par les handlers).
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

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

type PostService struct {
	repo *repository.PostRepository
	// notif émet les événements de notification (like, commentaire, mention…).
	// Par défaut un no-op : le post-service reste autonome si le
	// notification-service n'est pas configuré. Câblé via SetNotifier au boot.
	notif notifier.Notifier
	// bookmarkWindow : durée de la fenêtre glissante de rafale. Un clic court qui
	// suit le précédent signet de moins de bookmarkWindow range automatiquement
	// dans la dernière collection ; au-delà, le serveur redemande la collection.
	// <= 0 = jamais d'auto-classement (toujours proposer).
	bookmarkWindow time.Duration
	profilClient   profilVisibilityClient
	followClient   followStatusClient
}

type profilVisibilityClient interface {
	Visibility(ctx context.Context, userID string) (string, error)
}

type followStatusClient interface {
	IsFollowing(ctx context.Context, followerID, followingID string) (bool, error)
}

type Option func(*PostService)

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

func NewPostService(r *repository.PostRepository, opts ...Option) *PostService {
	s := &PostService{repo: r, notif: notifier.Noop{}}
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
func (s *PostService) CreatePost(ctx context.Context, authorID, content, quotePostID string, media []models.MediaRef) (*models.Post, error) {
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
	post := &models.Post{
		AuthorID:      authorID,
		Content:       content,
		Media:         media,
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
	return post, nil
}

// GetPosts renvoie le fil global, du plus récent au plus ancien, paginé.
func (s *PostService) GetPosts(ctx context.Context, viewerID string, limit, offset int64) ([]models.Post, error) {
	return s.visibleFeedPage(ctx, viewerID, limit, offset, s.repo.GetAll)
}

// GetPost renvoie un post par son id si le profil de l'auteur est lisible par
// le visiteur courant.
func (s *PostService) GetPost(ctx context.Context, id, viewerID string) (*models.Post, error) {
	oid, err := parseID(id)
	if err != nil {
		return nil, err
	}
	post, err := s.repo.Get(ctx, oid)
	if err != nil {
		return nil, translateNotFound(err)
	}
	allowed, err := s.canReadAuthor(ctx, viewerID, post.AuthorID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrPrivateProfil
	}
	return post, nil
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
	updated, err := s.repo.Update(ctx, oid, content)
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

// DeletePost supprime un post si l'acteur en a le droit, puis purge ses likes
// et commentaires (best-effort, pour ne pas laisser d'orphelins).
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
	if err := s.repo.Delete(ctx, oid); err != nil {
		return translateNotFound(err)
	}
	_ = s.repo.DeleteLikesByPost(ctx, id)
	_ = s.repo.DeleteCommentsByPost(ctx, id)
	_ = s.repo.DeleteRepostsByPost(ctx, id)
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

// GetByProfile renvoie les posts d'un auteur, du plus récent au plus ancien.
func (s *PostService) GetByProfile(ctx context.Context, authorID, viewerID string, limit, offset int64) ([]models.Post, error) {
	allowed, err := s.canReadAuthor(ctx, viewerID, authorID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return []models.Post{}, nil
	}
	return s.repo.GetByProfile(ctx, authorID, clampLimit(limit), clampOffset(offset))
}

// GetFeed renvoie les posts d'un ensemble d'auteurs (fil « Abonnements »). Le
// front fournit les ids suivis (seul le user-service connaît le graphe) ; la
// sélection + le tri + la pagination sont faits côté DB ($in indexé). Liste
// vide → aucun post (pas de requête inutile).
func (s *PostService) GetFeed(ctx context.Context, authorIDs []string, viewerID string, limit, offset int64) ([]models.Post, error) {
	if len(authorIDs) == 0 {
		return []models.Post{}, nil
	}
	return s.visibleFeedPage(ctx, viewerID, limit, offset, func(ctx context.Context, pageLimit, pageOffset int64) ([]models.Post, error) {
		return s.repo.GetByAuthors(ctx, authorIDs, pageLimit, pageOffset)
	})
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

// PostLikers renvoie les ids des utilisateurs ayant liké un post.
func (s *PostService) PostLikers(ctx context.Context, id string) ([]string, error) {
	if _, err := parseID(id); err != nil {
		return nil, err
	}
	return s.repo.LikersByPost(ctx, id)
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
func (s *PostService) CreateComment(ctx context.Context, postID, authorID, content, parentID string, media []models.MediaRef) (*models.Comment, error) {
	oid, err := parseID(postID)
	if err != nil {
		return nil, err
	}
	post, err := s.repo.Get(ctx, oid)
	if err != nil {
		return nil, translateNotFound(err)
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
		PostID:    postID,
		ParentID:  rootID,
		AuthorID:  authorID,
		Content:   content,
		Media:     media,
		CreatedAt: now,
		UpdatedAt: now,
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
		// Commentaire racine → l'auteur du post.
		s.notif.Emit(notifier.Event{
			Type:        notifier.TypeComment,
			ActorID:     actorID,
			RecipientID: postAuthorID,
			PostID:      postID,
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

// ListComments renvoie les commentaires RACINE d'un post (chronologiques, paginés).
func (s *PostService) ListComments(ctx context.Context, postID string, limit, offset int64) ([]models.Comment, error) {
	if _, err := parseID(postID); err != nil {
		return nil, err
	}
	return s.repo.ListComments(ctx, postID, clampLimit(limit), clampOffset(offset))
}

// ListReplies renvoie les réponses d'un commentaire (chronologiques, paginées).
func (s *PostService) ListReplies(ctx context.Context, commentID string, limit, offset int64) ([]models.Comment, error) {
	if _, err := parseID(commentID); err != nil {
		return nil, err
	}
	return s.repo.ListReplies(ctx, commentID, clampLimit(limit), clampOffset(offset))
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
		n, _ := s.repo.DeleteRepliesByParent(ctx, commentID)
		removed += int32(n)
	} else if rcoid, err := parseID(comment.ParentID); err == nil {
		// Réponse : décrémente le compteur de réponses de la racine.
		_ = s.repo.IncReplyCount(ctx, rcoid, -1)
	}

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
	return authorID == actorID ||
		actorRole == models.RoleModerator ||
		actorRole == models.RoleAdmin
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
