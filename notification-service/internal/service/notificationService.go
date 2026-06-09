// Package service porte la logique métier du notification-service : agrégation
// des événements en notifications « stackées », résolution des mentions, et
// diffusion temps réel via un Publisher (le hub WebSocket).
package service

import (
	"context"
	"errors"
	"log"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/webdad/notification-service/internal/models"
	"github.com/webdad/notification-service/internal/repository"
	"github.com/webdad/notification-service/internal/userdir"
)

// Erreurs métier — traduites en codes HTTP par les handlers.
var (
	// ErrNotFound : notification inexistante (ou pas au demandeur) → 404.
	ErrNotFound = errors.New("notification introuvable")
	// ErrInvalidID : id mal formé (pas un ObjectID hexadécimal) → 400.
	ErrInvalidID = errors.New("identifiant de notification invalide")
)

// Bornes de pagination.
const (
	DefaultLimit = 20
	MaxLimit     = 100
)

// Publisher diffuse un événement aux connexions temps réel des utilisateurs
// visés (le hub WebSocket l'implémente). Interface → service testable sans hub.
type Publisher interface {
	Publish(userIDs []string, event any)
}

// NotificationService orchestre dépôt + résolveur de handles + diffusion.
type NotificationService struct {
	repo      *repository.NotificationRepository
	publisher Publisher
	resolver  userdir.Resolver
}

func NewNotificationService(r *repository.NotificationRepository, p Publisher, res userdir.Resolver) *NotificationService {
	return &NotificationService{repo: r, publisher: p, resolver: res}
}

// HandleEvent applique un événement reçu de post-service : agrégation,
// (dé)comptage, résolution de mentions, puis diffusion temps réel. Best-effort
// par nature (l'émetteur ne réessaie pas) : on log et on continue sur erreur
// partielle plutôt que d'échouer tout l'événement.
func (s *NotificationService) HandleEvent(ctx context.Context, ev models.Event) error {
	switch ev.Type {
	case models.EventPostDeleted:
		if ev.PostID == "" {
			return nil
		}
		recipients, err := s.repo.DeleteByPost(ctx, ev.PostID)
		if err != nil {
			return err
		}
		for _, r := range recipients {
			s.pushRefresh(r)
		}
		return nil

	case models.TypeLike, models.TypeComment, models.TypeReply, models.TypeRepost, models.TypeQuote:
		recipient := ev.RecipientID
		if recipient == "" || recipient == ev.ActorID {
			return nil // pas de notification à soi-même
		}
		gk, ok := groupKeyFor(ev)
		if !ok {
			return nil
		}
		return s.applyToGroup(ctx, recipient, gk, ev)

	case models.TypeMention:
		return s.handleMentions(ctx, ev)

	default:
		return nil
	}
}

// handleMentions résout chaque @handle en id (via user-service) et agrège une
// notification de mention par destinataire distinct (hors auteur). Best-effort :
// un handle introuvable est ignoré.
func (s *NotificationService) handleMentions(ctx context.Context, ev models.Event) error {
	gk, ok := groupKeyFor(ev)
	if !ok {
		return nil
	}
	seen := make(map[string]bool)
	for _, raw := range ev.MentionHandles {
		handle := normalizeHandle(raw)
		if handle == "" {
			continue
		}
		id, found, err := s.resolver.ResolveHandle(ctx, handle)
		if err != nil {
			log.Printf("[notification] résolution du handle @%s : %v", handle, err)
			continue
		}
		if !found || id == ev.ActorID || seen[id] {
			continue
		}
		seen[id] = true
		if err := s.applyToGroup(ctx, id, gk, ev); err != nil {
			log.Printf("[notification] mention pour %s : %v", id, err)
		}
	}
	return nil
}

// applyToGroup agrège (ou défait) un événement dans le groupe (recipient, gk)
// puis pousse la mise à jour temps réel.
func (s *NotificationService) applyToGroup(ctx context.Context, recipient, groupKey string, ev models.Event) error {
	if ev.Retract {
		n, deleted, err := s.repo.Decrement(ctx, recipient, groupKey)
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil // rien à défaire
		}
		if err != nil {
			return err
		}
		if deleted {
			s.pushDeleted(recipient, n.ID.Hex())
		} else {
			s.pushNotification(n)
		}
		return nil
	}

	saved, err := s.repo.Upsert(ctx, &models.Notification{
		RecipientID: recipient,
		GroupKey:    groupKey,
		Type:        ev.Type,
		PostID:      ev.PostID,
		CommentID:   ev.CommentID,
		LastActorID: ev.ActorID,
	})
	if err != nil {
		return err
	}
	s.pushNotification(saved)
	return nil
}

// --- Lecture (API authentifiée du destinataire) -----------------------------

// List renvoie les notifications de l'utilisateur, paginées par curseur.
func (s *NotificationService) List(ctx context.Context, recipientID string, limit int64, before string) ([]models.Notification, error) {
	var cursor *bson.ObjectID
	if before != "" {
		oid, err := bson.ObjectIDFromHex(before)
		if err != nil {
			return nil, ErrInvalidID
		}
		cursor = &oid
	}
	return s.repo.List(ctx, recipientID, clampLimit(limit), cursor)
}

// UnreadCount renvoie le nombre de notifications non lues (badge).
func (s *NotificationService) UnreadCount(ctx context.Context, recipientID string) (int64, error) {
	return s.repo.CountUnread(ctx, recipientID)
}

// MarkAllRead passe toutes les notifications de l'utilisateur en lues.
func (s *NotificationService) MarkAllRead(ctx context.Context, recipientID string) error {
	return s.repo.MarkAllRead(ctx, recipientID)
}

// MarkRead passe une notification de l'utilisateur en lue.
func (s *NotificationService) MarkRead(ctx context.Context, recipientID, id string) error {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return ErrInvalidID
	}
	if err := s.repo.MarkRead(ctx, recipientID, oid); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

// --- Diffusion temps réel ----------------------------------------------------

func (s *NotificationService) pushNotification(n *models.Notification) {
	s.publisher.Publish([]string{n.RecipientID}, map[string]any{"type": "notification", "data": n})
}

func (s *NotificationService) pushDeleted(recipientID, id string) {
	s.publisher.Publish([]string{recipientID}, map[string]any{"type": "notification_deleted", "data": map[string]string{"id": id}})
}

// pushRefresh demande au client de recharger sa liste + son compteur (utilisé
// après une purge en cascade, où l'on ne réémet pas chaque notification).
func (s *NotificationService) pushRefresh(recipientID string) {
	s.publisher.Publish([]string{recipientID}, map[string]any{"type": "notification_refresh"})
}

// --- Helpers PURS (testés) ---------------------------------------------------

// groupKeyFor construit la clé d'agrégation d'un événement. C'est elle qui
// définit le « stacking » : tous les likes d'un post partagent `like:<post_id>`.
func groupKeyFor(ev models.Event) (string, bool) {
	switch ev.Type {
	case models.TypeLike:
		if ev.PostID == "" {
			return "", false
		}
		return "like:" + ev.PostID, true
	case models.TypeComment:
		if ev.PostID == "" {
			return "", false
		}
		return "comment:" + ev.PostID, true
	case models.TypeReply:
		if ev.CommentID == "" {
			return "", false
		}
		return "reply:" + ev.CommentID, true
	case models.TypeRepost:
		// Tous les reposts d'un post s'agrègent (post_id = post original).
		if ev.PostID == "" {
			return "", false
		}
		return "repost:" + ev.PostID, true
	case models.TypeQuote:
		// Une citation ne s'agrège pas : le groupe est le post citant lui-même.
		if ev.PostID == "" {
			return "", false
		}
		return "quote:" + ev.PostID, true
	case models.TypeMention:
		// Une mention ne s'agrège pas entre sources : le groupe est la source
		// (le commentaire si présent, sinon le post).
		src := ev.CommentID
		if src == "" {
			src = ev.PostID
		}
		if src == "" {
			return "", false
		}
		return "mention:" + src, true
	}
	return "", false
}

// normalizeHandle nettoie un handle de mention (retire un éventuel « @ » et les
// espaces). Renvoie "" si vide.
func normalizeHandle(raw string) string {
	return strings.TrimPrefix(strings.TrimSpace(raw), "@")
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
