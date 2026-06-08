// Package service porte la logique métier du message-service : contrôle
// d'appartenance/rôle, dédup des DM, traduction des erreurs du dépôt. Le service
// reste AVEUGLE au contenu (chiffré côté client) : il n'orchestre que des
// ciphertexts et des enveloppes de clé.
package service

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/webdad/message-service/internal/models"
	"github.com/webdad/message-service/internal/repository"
)

// Erreurs métier — traduites en codes HTTP par les handlers.
var (
	ErrConversationNotFound = errors.New("conversation introuvable")
	ErrInvalidID            = errors.New("identifiant invalide")
	ErrNotMember            = errors.New("vous n'êtes pas membre de cette conversation")
	ErrCannotWrite          = errors.New("écriture non autorisée (lecture seule)")
	ErrKeyNotFound          = errors.New("clé publique introuvable pour cet utilisateur")
	ErrSelfConversation     = errors.New("impossible de démarrer une conversation avec soi-même")
	ErrMissingEnvelope      = errors.New("enveloppe de clé manquante pour un membre")
	ErrInvalidGroup         = errors.New("groupe invalide (créateur absent des enveloppes ou aucun membre)")
	ErrNotGroup             = errors.New("opération réservée aux groupes")
	ErrAlreadyMember        = errors.New("cet utilisateur est déjà membre")
	ErrGroupFull            = errors.New("groupe complet (32 membres maximum)")
	ErrOwnerOnly            = errors.New("action réservée au créateur du groupe")
	ErrOwnerCannotLeave     = errors.New("le créateur ne peut pas quitter le groupe (le supprimer à la place)")
	ErrTargetNotMember      = errors.New("cet utilisateur n'est pas membre du groupe")
)

// Bornes de pagination des messages.
const (
	DefaultLimit = 30
	MaxLimit     = 100
)

// MaxGroupMembers : nombre maximum de membres (« talkers ») d'un groupe.
const MaxGroupMembers = 32

type MessageService struct {
	repo *repository.MessageRepository
}

func NewMessageService(r *repository.MessageRepository) *MessageService {
	return &MessageService{repo: r}
}

// --- Clés publiques ----------------------------------------------------------

// PublishKey publie/met à jour la clé publique d'identité de l'utilisateur.
func (s *MessageService) PublishKey(ctx context.Context, userID, publicKey string) error {
	return s.repo.UpsertKey(ctx, userID, publicKey)
}

// GetKey renvoie la clé publique d'un utilisateur (404 si absente).
func (s *MessageService) GetKey(ctx context.Context, userID string) (*models.UserKey, error) {
	k, err := s.repo.GetKey(ctx, userID)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrKeyNotFound
	}
	return k, err
}

// --- Conversations -----------------------------------------------------------

// CreateDM crée (ou retrouve) le DM entre creatorID et peerID. Le client a déjà
// généré la clé de contenu et l'a emballée pour les deux membres : `envelopes`
// mappe user_id -> enveloppe. Idempotent : si le DM existe, on le renvoie tel
// quel (on n'écrase pas les enveloppes existantes).
func (s *MessageService) CreateDM(ctx context.Context, creatorID, peerID string, envelopes map[string]string) (*models.ConversationView, error) {
	if peerID == creatorID {
		return nil, ErrSelfConversation
	}

	pair := sortedPair(creatorID, peerID)
	key := dmKey(creatorID, peerID)

	// DM déjà existant → on le renvoie (vue du créateur).
	if existing, err := s.repo.FindDMByKey(ctx, key); err == nil {
		return s.viewFor(ctx, existing, creatorID)
	} else if !errors.Is(err, mongo.ErrNoDocuments) {
		return nil, err
	}

	// Les deux enveloppes sont requises (sinon un membre ne pourrait pas lire).
	for _, uid := range pair {
		if strings.TrimSpace(envelopes[uid]) == "" {
			return nil, ErrMissingEnvelope
		}
	}

	now := time.Now()
	conv := &models.Conversation{
		Type:      models.TypeDM,
		MemberIDs: pair,
		DMKey:     key,
		CreatedBy: creatorID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.CreateConversation(ctx, conv); err != nil {
		// Course possible (deux POST simultanés) : l'index unique dm_key rejette
		// le second → on retombe sur l'existant.
		if mongo.IsDuplicateKeyError(err) {
			if existing, ferr := s.repo.FindDMByKey(ctx, key); ferr == nil {
				return s.viewFor(ctx, existing, creatorID)
			}
		}
		return nil, err
	}

	convID := conv.ID.Hex()
	for _, uid := range pair {
		m := &models.Member{
			ConversationID: convID,
			UserID:         uid,
			Role:           models.MemberTalker, // DM : les deux peuvent écrire
			KeyEnvelope:    envelopes[uid],
			CreatedAt:      now,
		}
		if err := s.repo.AddMember(ctx, m); err != nil {
			return nil, err
		}
	}

	return s.viewFor(ctx, conv, creatorID)
}

// CreateGroup crée un groupe. Le créateur (owner) a généré la clé de contenu,
// chiffré le nom (title/titleNonce) avec elle, et l'a emballée pour CHAQUE
// membre initial : `envelopes` mappe user_id -> enveloppe. Les clés de la map
// définissent le set de membres (créateur inclus). Cap : 32 membres.
func (s *MessageService) CreateGroup(ctx context.Context, creatorID, title, titleNonce string, envelopes map[string]string) (*models.ConversationView, error) {
	if len(envelopes) == 0 || strings.TrimSpace(envelopes[creatorID]) == "" {
		return nil, ErrInvalidGroup
	}
	if len(envelopes) > MaxGroupMembers {
		return nil, ErrGroupFull
	}
	for _, env := range envelopes {
		if strings.TrimSpace(env) == "" {
			return nil, ErrMissingEnvelope
		}
	}

	memberIDs := make([]string, 0, len(envelopes))
	for uid := range envelopes {
		memberIDs = append(memberIDs, uid)
	}

	now := time.Now()
	conv := &models.Conversation{
		Type:       models.TypeGroup,
		MemberIDs:  memberIDs,
		Title:      title,
		TitleNonce: titleNonce,
		CreatedBy:  creatorID,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.repo.CreateConversation(ctx, conv); err != nil {
		return nil, err
	}

	convID := conv.ID.Hex()
	for uid, env := range envelopes {
		role := models.MemberTalker
		if uid == creatorID {
			role = models.MemberOwner
		}
		m := &models.Member{
			ConversationID: convID,
			UserID:         uid,
			Role:           role,
			KeyEnvelope:    env,
			CreatedAt:      now,
		}
		if err := s.repo.AddMember(ctx, m); err != nil {
			return nil, err
		}
	}

	return s.viewFor(ctx, conv, creatorID)
}

// AddMember ajoute (invite) un membre à un groupe. N'importe quel membre peut
// inviter (il détient la clé de contenu → il l'emballe pour l'invité). Refus si :
// l'acteur n'est pas membre, ce n'est pas un groupe, la cible est déjà membre,
// ou le cap est atteint. Renvoie les ids des membres (après ajout) pour diffusion.
func (s *MessageService) AddMember(ctx context.Context, conversationID, actorID, targetID, envelope string) ([]string, error) {
	if _, err := s.requireMember(ctx, conversationID, actorID); err != nil {
		return nil, err
	}
	conv, err := s.getConversationByID(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if conv.Type != models.TypeGroup {
		return nil, ErrNotGroup
	}

	if _, err := s.repo.GetMember(ctx, conversationID, targetID); err == nil {
		return nil, ErrAlreadyMember
	} else if !errors.Is(err, mongo.ErrNoDocuments) {
		return nil, err
	}

	count, err := s.repo.CountMembers(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if count >= MaxGroupMembers {
		return nil, ErrGroupFull
	}

	now := time.Now()
	m := &models.Member{
		ConversationID: conversationID,
		UserID:         targetID,
		Role:           models.MemberTalker,
		KeyEnvelope:    envelope,
		CreatedAt:      now,
	}
	if err := s.repo.AddMember(ctx, m); err != nil {
		return nil, err
	}
	if err := s.repo.PushMemberID(ctx, conv.ID, targetID); err != nil {
		return nil, err
	}
	return s.repo.MemberIDs(ctx, conversationID)
}

// RemoveMember retire un membre. Deux cas :
//   - targetID == actorID : QUITTER (interdit à l'owner → ErrOwnerCannotLeave) ;
//   - targetID != actorID : EXCLURE, réservé à l'owner (ErrOwnerOnly sinon).
//
// Renvoie l'ensemble des ids concernés (membres restants + la cible) pour
// notifier en temps réel y compris la personne retirée.
func (s *MessageService) RemoveMember(ctx context.Context, conversationID, actorID, targetID string) ([]string, error) {
	actor, err := s.requireMember(ctx, conversationID, actorID)
	if err != nil {
		return nil, err
	}
	conv, err := s.getConversationByID(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if conv.Type != models.TypeGroup {
		return nil, ErrNotGroup
	}
	if err := checkRemoval(actor.Role, targetID == actorID); err != nil {
		return nil, err
	}
	if targetID != actorID {
		// La cible doit être membre (et n'est pas l'owner — on ne retire pas le créateur).
		target, err := s.repo.GetMember(ctx, conversationID, targetID)
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrTargetNotMember
		}
		if err != nil {
			return nil, err
		}
		if target.Role == models.MemberOwner {
			return nil, ErrOwnerOnly
		}
	}

	// Ids notifiés = membres actuels (avant retrait) — inclut la cible.
	notify, _ := s.repo.MemberIDs(ctx, conversationID)

	removed, err := s.repo.RemoveMember(ctx, conversationID, targetID)
	if err != nil {
		return nil, err
	}
	if !removed {
		return nil, ErrTargetNotMember
	}
	if err := s.repo.PullMemberID(ctx, conv.ID, targetID); err != nil {
		return nil, err
	}
	return notify, nil
}

// DeleteGroup supprime un groupe (owner uniquement) et purge membres + messages.
// Renvoie les ids des membres (avant suppression) pour notification.
func (s *MessageService) DeleteGroup(ctx context.Context, conversationID, actorID string) ([]string, error) {
	actor, err := s.requireMember(ctx, conversationID, actorID)
	if err != nil {
		return nil, err
	}
	conv, err := s.getConversationByID(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if conv.Type != models.TypeGroup {
		return nil, ErrNotGroup
	}
	if actor.Role != models.MemberOwner {
		return nil, ErrOwnerOnly
	}

	notify, _ := s.repo.MemberIDs(ctx, conversationID)

	if err := s.repo.DeleteConversation(ctx, conv.ID); err != nil {
		return nil, translateNotFound(err)
	}
	_ = s.repo.DeleteMembersByConversation(ctx, conversationID)
	_ = s.repo.DeleteMessagesByConversation(ctx, conversationID)
	return notify, nil
}

// UpdateGroup renomme un groupe (nom re-chiffré côté client) — owner uniquement.
func (s *MessageService) UpdateGroup(ctx context.Context, conversationID, actorID, title, titleNonce string) (*models.ConversationView, error) {
	actor, err := s.requireMember(ctx, conversationID, actorID)
	if err != nil {
		return nil, err
	}
	conv, err := s.getConversationByID(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if conv.Type != models.TypeGroup {
		return nil, ErrNotGroup
	}
	if actor.Role != models.MemberOwner {
		return nil, ErrOwnerOnly
	}

	updated, err := s.repo.UpdateTitle(ctx, conv.ID, title, titleNonce)
	if err != nil {
		return nil, translateNotFound(err)
	}
	return s.viewFor(ctx, updated, actorID)
}

// ListMembers renvoie la liste des membres (id + rôle, SANS les enveloppes des
// autres) — réservée aux membres de la conversation.
func (s *MessageService) ListMembers(ctx context.Context, conversationID, actorID string) ([]models.MemberView, error) {
	if _, err := s.requireMember(ctx, conversationID, actorID); err != nil {
		return nil, err
	}
	members, err := s.repo.ListMembers(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	views := make([]models.MemberView, 0, len(members))
	for _, m := range members {
		views = append(views, models.MemberView{UserID: m.UserID, Role: m.Role})
	}
	return views, nil
}

// ListConversations renvoie les conversations de l'utilisateur (avec SON
// enveloppe et SON rôle), triées par activité décroissante.
func (s *MessageService) ListConversations(ctx context.Context, userID string) ([]models.ConversationView, error) {
	memberships, err := s.repo.ListMembersByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	views := make([]models.ConversationView, 0, len(memberships))
	for _, m := range memberships {
		oid, err := parseID(m.ConversationID)
		if err != nil {
			continue
		}
		conv, err := s.repo.GetConversation(ctx, oid)
		if err != nil {
			continue // conversation supprimée → on ignore l'appartenance orpheline
		}
		views = append(views, buildView(conv, &m))
	}

	sort.Slice(views, func(i, j int) bool {
		return views[i].UpdatedAt.After(views[j].UpdatedAt)
	})
	return views, nil
}

// GetConversation renvoie la vue d'une conversation pour un membre (403 sinon).
func (s *MessageService) GetConversation(ctx context.Context, conversationID, userID string) (*models.ConversationView, error) {
	oid, err := parseID(conversationID)
	if err != nil {
		return nil, err
	}
	conv, err := s.repo.GetConversation(ctx, oid)
	if err != nil {
		return nil, translateNotFound(err)
	}
	return s.viewFor(ctx, conv, userID)
}

// --- Messages ----------------------------------------------------------------

// ListMessages renvoie une page de messages (réservée aux membres).
func (s *MessageService) ListMessages(ctx context.Context, conversationID, userID string, limit int64, beforeID string) ([]models.Message, error) {
	if _, err := s.requireMember(ctx, conversationID, userID); err != nil {
		return nil, err
	}

	var before *bson.ObjectID
	if beforeID != "" {
		oid, err := bson.ObjectIDFromHex(beforeID)
		if err != nil {
			return nil, ErrInvalidID
		}
		before = &oid
	}
	return s.repo.ListMessages(ctx, conversationID, clampLimit(limit), before)
}

// SendMessage persiste un message chiffré (membre + droit d'écriture requis) et
// renvoie le message créé ainsi que les ids des membres (pour la diffusion WS).
func (s *MessageService) SendMessage(ctx context.Context, conversationID, senderID, ciphertext, nonce string) (*models.Message, []string, error) {
	member, err := s.requireMember(ctx, conversationID, senderID)
	if err != nil {
		return nil, nil, err
	}
	if !canWrite(member.Role) {
		return nil, nil, ErrCannotWrite
	}

	now := time.Now()
	msg := &models.Message{
		ConversationID: conversationID,
		SenderID:       senderID,
		Ciphertext:     ciphertext,
		Nonce:          nonce,
		CreatedAt:      now,
	}
	if err := s.repo.InsertMessage(ctx, msg); err != nil {
		return nil, nil, err
	}

	// Remonte la conversation en tête de liste (best-effort).
	if oid, perr := parseID(conversationID); perr == nil {
		_ = s.repo.TouchConversation(ctx, oid, now)
	}

	memberIDs, err := s.repo.MemberIDs(ctx, conversationID)
	if err != nil {
		// Le message est persisté ; l'échec de diffusion n'est pas fatal.
		memberIDs = nil
	}
	return msg, memberIDs, nil
}

// IsMember indique si un utilisateur est membre (utilisé par le handler WS pour
// autoriser l'abonnement temps réel à une conversation).
func (s *MessageService) IsMember(ctx context.Context, conversationID, userID string) (bool, error) {
	_, err := s.repo.GetMember(ctx, conversationID, userID)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// --- Helpers internes --------------------------------------------------------

// getConversationByID parse l'id et charge la conversation (404 si absente).
func (s *MessageService) getConversationByID(ctx context.Context, conversationID string) (*models.Conversation, error) {
	oid, err := parseID(conversationID)
	if err != nil {
		return nil, err
	}
	conv, err := s.repo.GetConversation(ctx, oid)
	if err != nil {
		return nil, translateNotFound(err)
	}
	return conv, nil
}

// requireMember renvoie l'appartenance ou ErrNotMember (403).
func (s *MessageService) requireMember(ctx context.Context, conversationID, userID string) (*models.Member, error) {
	if _, err := parseID(conversationID); err != nil {
		return nil, err
	}
	m, err := s.repo.GetMember(ctx, conversationID, userID)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotMember
	}
	if err != nil {
		return nil, err
	}
	return m, nil
}

// viewFor construit la vue d'une conversation du point de vue d'un membre
// (récupère son appartenance pour l'enveloppe + le rôle). ErrNotMember si absent.
func (s *MessageService) viewFor(ctx context.Context, conv *models.Conversation, userID string) (*models.ConversationView, error) {
	m, err := s.repo.GetMember(ctx, conv.ID.Hex(), userID)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotMember
	}
	if err != nil {
		return nil, err
	}
	v := buildView(conv, m)
	return &v, nil
}

// --- Fonctions pures (testées unitairement) ----------------------------------

// sortedPair renvoie la paire d'ids triée (ordre déterministe pour le dm_key).
func sortedPair(a, b string) []string {
	if a <= b {
		return []string{a, b}
	}
	return []string{b, a}
}

// dmKey : clé de dédup d'un DM, indépendante de l'ordre des participants.
func dmKey(a, b string) string {
	p := sortedPair(a, b)
	return p[0] + ":" + p[1]
}

// checkRemoval : règle d'autorisation du retrait d'un membre (fonction PURE,
// testée). `isSelf` = l'acteur se retire lui-même (quitter) vs retirer autrui
// (exclure). L'owner ne peut pas quitter ; seul l'owner peut exclure autrui.
func checkRemoval(actorRole string, isSelf bool) error {
	if isSelf {
		if actorRole == models.MemberOwner {
			return ErrOwnerCannotLeave
		}
		return nil
	}
	if actorRole != models.MemberOwner {
		return ErrOwnerOnly
	}
	return nil
}

// canWrite : un membre peut écrire s'il est owner, admin ou talker. viewer =
// lecture seule (communautés).
func canWrite(role string) bool {
	switch role {
	case models.MemberOwner, models.MemberAdmin, models.MemberTalker:
		return true
	default:
		return false
	}
}

// buildView assemble la vue renvoyée au client (conversation + données du membre).
func buildView(conv *models.Conversation, m *models.Member) models.ConversationView {
	return models.ConversationView{
		ID:         conv.ID.Hex(),
		Type:       conv.Type,
		MemberIDs:  conv.MemberIDs,
		Title:      conv.Title,
		TitleNonce: conv.TitleNonce,
		MyRole:     m.Role,
		MyEnvelope: m.KeyEnvelope,
		CreatedBy:  conv.CreatedBy,
		CreatedAt:  conv.CreatedAt,
		UpdatedAt:  conv.UpdatedAt,
	}
}

// parseID valide qu'un id est bien un ObjectID hexadécimal.
func parseID(id string) (bson.ObjectID, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return bson.ObjectID{}, ErrInvalidID
	}
	return oid, nil
}

// translateNotFound mappe l'absence de document Mongo vers ErrConversationNotFound.
func translateNotFound(err error) error {
	if errors.Is(err, mongo.ErrNoDocuments) {
		return ErrConversationNotFound
	}
	return err
}

// clampLimit borne la taille de page (défaut 30, max 100).
func clampLimit(limit int64) int64 {
	if limit <= 0 {
		return DefaultLimit
	}
	if limit > MaxLimit {
		return MaxLimit
	}
	return limit
}
