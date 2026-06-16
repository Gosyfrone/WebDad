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
	"github.com/webdad/message-service/internal/notifier"
	"github.com/webdad/message-service/internal/repository"
)

// Erreurs métier — traduites en codes HTTP par les handlers.
var (
	ErrConversationNotFound = errors.New("conversation introuvable")
	ErrInvalidID            = errors.New("identifiant invalide")
	ErrNotMember            = errors.New("vous n'êtes pas membre de cette conversation")
	ErrCannotWrite          = errors.New("écriture non autorisée (lecture seule)")
	ErrKeyNotFound          = errors.New("clé publique introuvable pour cet utilisateur")
	ErrBackupNotFound       = errors.New("aucune sauvegarde chiffrée pour cet utilisateur")
	ErrSelfConversation     = errors.New("impossible de démarrer une conversation avec soi-même")
	ErrMissingEnvelope      = errors.New("enveloppe de clé manquante pour un membre")
	ErrInvalidGroup         = errors.New("groupe invalide (créateur absent des enveloppes ou aucun membre)")
	ErrInvalidCommunity     = errors.New("communauté invalide (nom ou clé de contenu manquant)")
	ErrNotGroup             = errors.New("opération réservée aux groupes")
	ErrNotCommunity         = errors.New("opération réservée aux communautés")
	ErrNotManageable        = errors.New("opération réservée aux groupes et communautés")
	ErrAlreadyMember        = errors.New("cet utilisateur est déjà membre")
	ErrTalkersFull          = errors.New("nombre maximum de participants pouvant écrire atteint (32)")
	ErrInvalidRole          = errors.New("rôle invalide (talker ou viewer attendu)")
	ErrOwnerOnly            = errors.New("action réservée au créateur")
	ErrOwnerCannotLeave     = errors.New("le créateur ne peut pas quitter (le supprimer à la place)")
	ErrTargetNotMember      = errors.New("cet utilisateur n'est pas membre")
	ErrMessageNotFound      = errors.New("message introuvable")
	ErrNotMessageOwner      = errors.New("seul l'expéditeur peut modifier ce message")
	ErrCannotDelete         = errors.New("suppression non autorisée (auteur, owner ou admin requis)")
)

// Bornes de pagination des messages.
const (
	DefaultLimit = 30
	MaxLimit     = 100
)

// MaxTalkers : nombre maximum de participants pouvant écrire (owner + talkers).
// Vaut pour les groupes (tous talkers) ET les communautés (les viewers, en
// lecture seule, ne sont PAS comptés et sont illimités).
const MaxTalkers = 32

type MessageService struct {
	repo     *repository.MessageRepository
	notifier notifier.Notifier
}

func NewMessageService(r *repository.MessageRepository) *MessageService {
	return &MessageService{repo: r, notifier: notifier.Noop{}}
}

// SetNotifier branche l'émission d'événements vers le notification-service
// (best-effort). Sans appel, le service reste autonome (Noop).
func (s *MessageService) SetNotifier(n notifier.Notifier) {
	if n != nil {
		s.notifier = n
	}
}

// --- Clés publiques ----------------------------------------------------------

// PublishKey publie/met à jour la clé publique d'identité de l'utilisateur.
func (s *MessageService) PublishKey(ctx context.Context, userID, publicKey string) error {
	return s.repo.UpsertKey(ctx, userID, publicKey)
}

// PurgeUser efface DÉFINITIVEMENT la participation d'un utilisateur à la
// messagerie (effacement RGPD).
func (s *MessageService) PurgeUser(ctx context.Context, userID string) error {
	return s.repo.PurgeUser(ctx, userID)
}

// GetKey renvoie la clé publique d'un utilisateur (404 si absente).
func (s *MessageService) GetKey(ctx context.Context, userID string) (*models.UserKey, error) {
	k, err := s.repo.GetKey(ctx, userID)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrKeyNotFound
	}
	return k, err
}

// --- Sauvegarde chiffrée de la clé privée -----------------------------------

// PutBackup enregistre/remplace la sauvegarde chiffrée de la clé privée de
// l'utilisateur (blobs opaques produits côté client, cf. modèle zero-knowledge).
func (s *MessageService) PutBackup(ctx context.Context, userID string, b models.PutBackupRequest) error {
	return s.repo.UpsertBackup(ctx, userID, b)
}

// GetBackup renvoie la sauvegarde chiffrée de l'utilisateur (404 si absente).
func (s *MessageService) GetBackup(ctx context.Context, userID string) (*models.KeyBackup, error) {
	b, err := s.repo.GetBackup(ctx, userID)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrBackupNotFound
	}
	return b, err
}

// BackupStatus indique si une sauvegarde existe (pilote l'UI définir/débloquer).
func (s *MessageService) BackupStatus(ctx context.Context, userID string) (bool, error) {
	return s.repo.BackupExists(ctx, userID)
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
	if len(envelopes) > MaxTalkers {
		return nil, ErrTalkersFull
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

// CreateCommunity crée une communauté. Modèle HYBRIDE : le client génère la clé
// de contenu et la CONFIE au serveur (`contentKey`), qui la remettra à chaque
// nouvel arrivant (auto-join illimité). Le nom (`title`) est en CLAIR (semi-
// public, découvrable). Le créateur est l'owner (et l'unique membre au départ).
// ⚠️ Conséquence assumée : le serveur peut lire les communautés (pas DM/groupes).
func (s *MessageService) CreateCommunity(ctx context.Context, creatorID, title, contentKey string) (*models.ConversationView, error) {
	if strings.TrimSpace(title) == "" || strings.TrimSpace(contentKey) == "" {
		return nil, ErrInvalidCommunity
	}

	now := time.Now()
	conv := &models.Conversation{
		Type:       models.TypeCommunity,
		MemberIDs:  []string{creatorID},
		Title:      title,
		ContentKey: contentKey,
		CreatedBy:  creatorID,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.repo.CreateConversation(ctx, conv); err != nil {
		return nil, err
	}
	if err := s.repo.AddMember(ctx, &models.Member{
		ConversationID: conv.ID.Hex(),
		UserID:         creatorID,
		Role:           models.MemberOwner,
		CreatedAt:      now,
	}); err != nil {
		return nil, err
	}
	return s.viewFor(ctx, conv, creatorID)
}

// JoinCommunity : auto-join d'une communauté en VIEWER (lecture seule). Renvoie
// la vue AVEC la clé de contenu (le serveur la remet). Idempotent : déjà membre
// → renvoie sa vue inchangée. Renvoie aussi les ids à notifier (membres + soi).
func (s *MessageService) JoinCommunity(ctx context.Context, conversationID, userID string) (*models.ConversationView, []string, error) {
	conv, err := s.getConversationByID(ctx, conversationID)
	if err != nil {
		return nil, nil, err
	}
	if conv.Type != models.TypeCommunity {
		return nil, nil, ErrNotCommunity
	}

	// Déjà membre → idempotent.
	if _, err := s.repo.GetMember(ctx, conversationID, userID); err == nil {
		view, verr := s.viewFor(ctx, conv, userID)
		return view, nil, verr
	} else if !errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil, err
	}

	now := time.Now()
	if err := s.repo.AddMember(ctx, &models.Member{
		ConversationID: conversationID,
		UserID:         userID,
		Role:           models.MemberViewer, // rejoint en lecture seule
		CreatedAt:      now,
	}); err != nil {
		return nil, nil, err
	}
	if err := s.repo.PushMemberID(ctx, conv.ID, userID); err != nil {
		return nil, nil, err
	}

	view, err := s.viewFor(ctx, conv, userID)
	if err != nil {
		return nil, nil, err
	}
	notify, _ := s.repo.MemberIDs(ctx, conversationID)
	return view, notify, nil
}

// SetMemberRole promeut/rétrograde un membre d'une communauté entre talker
// (peut écrire) et viewer (lecture seule). Owner uniquement. Le cap de 32 ne
// porte que sur les talkers (la promotion le vérifie). On ne touche pas l'owner.
func (s *MessageService) SetMemberRole(ctx context.Context, conversationID, actorID, targetID, role string) ([]string, error) {
	if role != models.MemberTalker && role != models.MemberViewer {
		return nil, ErrInvalidRole
	}
	actor, err := s.requireMember(ctx, conversationID, actorID)
	if err != nil {
		return nil, err
	}
	conv, err := s.getConversationByID(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if conv.Type != models.TypeCommunity {
		return nil, ErrNotCommunity
	}
	if actor.Role != models.MemberOwner {
		return nil, ErrOwnerOnly
	}

	target, err := s.repo.GetMember(ctx, conversationID, targetID)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrTargetNotMember
	}
	if err != nil {
		return nil, err
	}
	if target.Role == models.MemberOwner {
		return nil, ErrOwnerOnly // le rôle de l'owner n'est pas modifiable
	}

	// Promotion viewer → talker : vérifier le cap des talkers (no-op si déjà talker).
	if role == models.MemberTalker && !canWrite(target.Role) {
		count, cerr := s.repo.CountWritableMembers(ctx, conversationID)
		if cerr != nil {
			return nil, cerr
		}
		if count >= MaxTalkers {
			return nil, ErrTalkersFull
		}
	}

	if err := s.repo.SetMemberRole(ctx, conversationID, targetID, role); err != nil {
		return nil, translateNotFound(err)
	}
	notify, _ := s.repo.MemberIDs(ctx, conversationID)
	return notify, nil
}

// ListCommunities renvoie l'annuaire public des communautés (nom en clair, nb de
// membres, déjà-membre), paginé, recherche par nom optionnelle. Jamais la clé.
func (s *MessageService) ListCommunities(ctx context.Context, userID string, limit, offset int64, q string) ([]models.CommunityListItem, error) {
	convs, err := s.repo.ListCommunities(ctx, clampLimit(limit), clampOffset(offset), q)
	if err != nil {
		return nil, err
	}
	items := make([]models.CommunityListItem, 0, len(convs))
	for _, conv := range convs {
		id := conv.ID.Hex()
		count, _ := s.repo.CountMembers(ctx, id)
		_, mErr := s.repo.GetMember(ctx, id, userID)
		items = append(items, models.CommunityListItem{
			ID:          id,
			Title:       conv.Title,
			MemberCount: count,
			IsMember:    mErr == nil,
			CreatedBy:   conv.CreatedBy,
			CreatedAt:   conv.CreatedAt,
			UpdatedAt:   conv.UpdatedAt,
		})
	}
	return items, nil
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

	count, err := s.repo.CountWritableMembers(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if count >= MaxTalkers {
		return nil, ErrTalkersFull
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
	if !isManageable(conv.Type) {
		return nil, ErrNotManageable
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
	if !isManageable(conv.Type) {
		return nil, ErrNotManageable
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
	if !isManageable(conv.Type) {
		return nil, ErrNotManageable
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
// enveloppe, SON rôle, SON épinglage), triées « épinglées d'abord » puis par
// activité décroissante. Les conversations « supprimées côté user » (`cleared_at`)
// sans message plus récent sont masquées (elles réapparaissent au prochain message).
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
		// Supprimée côté user : masquée tant qu'aucun message n'est postérieur.
		if m.ClearedAt != nil {
			hasNewer, herr := s.repo.HasMessagesAfter(ctx, m.ConversationID, *m.ClearedAt)
			if herr == nil && !hasNewer {
				continue
			}
		}
		view := buildView(conv, &m)
		s.attachReceipts(ctx, &view, userID)
		views = append(views, view)
	}

	sort.SliceStable(views, func(i, j int) bool {
		return convLess(views[i], views[j])
	})
	return views, nil
}

// PinConversation (dés)épingle une conversation pour l'utilisateur (membre requis)
// et renvoie la vue à jour.
func (s *MessageService) PinConversation(ctx context.Context, conversationID, userID string, pinned bool) (*models.ConversationView, error) {
	if _, err := s.requireMember(ctx, conversationID, userID); err != nil {
		return nil, err
	}
	var at *time.Time
	if pinned {
		now := time.Now()
		at = &now
	}
	if err := s.repo.SetMemberPinned(ctx, conversationID, userID, at); err != nil {
		return nil, translateNotFound(err)
	}
	conv, err := s.getConversationByID(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	return s.viewFor(ctx, conv, userID)
}

// ClearConversation « supprime » la conversation côté user : masque + coupe
// l'historique (membre requis). N'affecte pas les autres membres.
func (s *MessageService) ClearConversation(ctx context.Context, conversationID, userID string) error {
	if _, err := s.requireMember(ctx, conversationID, userID); err != nil {
		return err
	}
	if err := s.repo.SetMemberCleared(ctx, conversationID, userID, time.Now()); err != nil {
		return translateNotFound(err)
	}
	return nil
}

// MuteConversation met en sourdine (`mute=true`) ou réactive (`mute=false`) une
// conversation pour le membre courant, et renvoie sa vue à jour. Membre requis.
func (s *MessageService) MuteConversation(ctx context.Context, conversationID, userID string, mute bool) (*models.ConversationView, error) {
	if _, err := s.requireMember(ctx, conversationID, userID); err != nil {
		return nil, err
	}
	var at *time.Time
	if mute {
		now := time.Now()
		at = &now
	}
	if err := s.repo.SetMemberMuted(ctx, conversationID, userID, at); err != nil {
		return nil, translateNotFound(err)
	}
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

// MarkRead avance le curseur de lecture du membre courant à maintenant (la
// conversation est désormais « lue jusqu'ici »). Membre requis. Renvoie l'instant
// retenu + les ids des AUTRES membres (à qui diffuser l'accusé « ouvert ») pour
// que les expéditeurs voient les coches se mettre à jour en temps réel.
func (s *MessageService) MarkRead(ctx context.Context, conversationID, userID string) (time.Time, []string, error) {
	if _, err := s.requireMember(ctx, conversationID, userID); err != nil {
		return time.Time{}, nil, err
	}
	now := time.Now()
	if err := s.repo.SetMemberRead(ctx, conversationID, userID, now); err != nil {
		return time.Time{}, nil, translateNotFound(err)
	}
	return now, s.otherMembers(ctx, conversationID, userID), nil
}

// MarkDelivered avance le curseur de LIVRAISON (« remis ») de plusieurs membres à
// `at` (best-effort, par membre). Sert au marquage des destinataires EN LIGNE au
// moment de l'envoi (livraison instantanée par WebSocket).
func (s *MessageService) MarkDelivered(ctx context.Context, conversationID string, userIDs []string, at time.Time) {
	for _, uid := range userIDs {
		_ = s.repo.SetMemberDelivered(ctx, conversationID, uid, at)
	}
}

// TouchDelivered marque le membre courant « remis » jusqu'à maintenant (il vient
// de récupérer l'historique) et renvoie l'instant + les autres membres à notifier.
// Membre requis. Best-effort : utilisé en marge de la lecture de l'historique.
func (s *MessageService) TouchDelivered(ctx context.Context, conversationID, userID string) (time.Time, []string, error) {
	now := time.Now()
	if err := s.repo.SetMemberDelivered(ctx, conversationID, userID, now); err != nil {
		return time.Time{}, nil, translateNotFound(err)
	}
	return now, s.otherMembers(ctx, conversationID, userID), nil
}

// TypingTargets valide l'appartenance et renvoie les autres membres (cibles du
// signal éphémère « en train d'écrire »). Aucune persistance.
func (s *MessageService) TypingTargets(ctx context.Context, conversationID, userID string) ([]string, error) {
	if _, err := s.requireMember(ctx, conversationID, userID); err != nil {
		return nil, err
	}
	return s.otherMembers(ctx, conversationID, userID), nil
}

// otherMembers renvoie les ids des membres d'une conversation distincts de
// `userID` (cibles d'un accusé de réception). Best-effort (nil si échec).
func (s *MessageService) otherMembers(ctx context.Context, conversationID, userID string) []string {
	ids, err := s.repo.MemberIDs(ctx, conversationID)
	if err != nil {
		return nil
	}
	others := make([]string, 0, len(ids))
	for _, id := range ids {
		if id != userID {
			others = append(others, id)
		}
	}
	return others
}

// UnreadCount renvoie le nombre de conversations de l'utilisateur ayant au moins
// un message non lu (calcul serveur, sans lire le contenu chiffré).
func (s *MessageService) UnreadCount(ctx context.Context, userID string) (int, error) {
	return s.repo.CountUnreadConversations(ctx, userID)
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

// ListMessages renvoie une page de messages (réservée aux membres). Si le membre
// a « supprimé côté user » la conversation (`cleared_at`), l'historique est coupé
// à cette date (on ne renvoie que les messages postérieurs).
func (s *MessageService) ListMessages(ctx context.Context, conversationID, userID string, limit int64, beforeID string) ([]models.Message, error) {
	member, err := s.requireMember(ctx, conversationID, userID)
	if err != nil {
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
	return s.repo.ListMessages(ctx, conversationID, clampLimit(limit), before, member.ClearedAt)
}

// SendMessage persiste un message chiffré (membre + droit d'écriture requis) et
// renvoie le message créé ainsi que les ids des membres (pour la diffusion WS).
//
// Un événement `message` est envoyé aux autres membres des DM/groupes privés
// (pas aux communautés), agrégé côté notification-service par expéditeur unique.
// `mentionedIDs` (fournis par le client : ids des membres mentionnés @handle —
// le serveur ne lit pas le contenu chiffré) déclenchent aussi une notification
// `message_mention` par membre réellement présent (hors l'expéditeur).
func (s *MessageService) SendMessage(ctx context.Context, conversationID, senderID, ciphertext, nonce string, mentionedIDs []string) (*models.Message, []string, error) {
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

	if oid, perr := parseID(conversationID); perr == nil {
		if conv, cerr := s.repo.GetConversation(ctx, oid); cerr == nil && conv.Type != models.TypeCommunity {
			for _, rid := range messageTargets(memberIDs, senderID) {
				s.notifier.Emit(notifier.Event{
					Type:           notifier.TypeMessage,
					ActorID:        senderID,
					RecipientID:    rid,
					ConversationID: conversationID,
				})
			}
		}
	}

	// Notifie les membres mentionnés (best-effort, fire-and-forget). On ne
	// notifie QUE des membres réels (≠ l'expéditeur) : un id non-membre fourni
	// par le client est ignoré (pas de fuite, pas de notif parasite).
	for _, rid := range mentionedTargets(mentionedIDs, memberIDs, senderID) {
		s.notifier.Emit(notifier.Event{
			Type:           notifier.TypeMessageMention,
			ActorID:        senderID,
			RecipientID:    rid,
			ConversationID: conversationID,
		})
	}

	return msg, memberIDs, nil
}

// EditMessage remplace la version chiffrée d'un message. Le serveur ne lit
// jamais le clair : il conserve la version originale chiffrée et diffuse la
// nouvelle version chiffrée aux membres.
func (s *MessageService) EditMessage(ctx context.Context, conversationID, messageID, actorID, ciphertext, nonce string, mentionedIDs []string) (*models.Message, []string, error) {
	member, err := s.requireMember(ctx, conversationID, actorID)
	if err != nil {
		return nil, nil, err
	}
	if !canWrite(member.Role) {
		return nil, nil, ErrCannotWrite
	}
	oid, err := parseID(messageID)
	if err != nil {
		return nil, nil, err
	}
	msg, err := s.repo.GetMessage(ctx, conversationID, oid)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil, ErrMessageNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	if msg.SenderID != actorID {
		return nil, nil, ErrNotMessageOwner
	}

	updated, err := s.repo.UpdateMessageCiphertext(ctx, msg, ciphertext, nonce, time.Now())
	if err != nil {
		return nil, nil, translateNotFound(err)
	}

	memberIDs, err := s.repo.MemberIDs(ctx, conversationID)
	if err != nil {
		memberIDs = nil
	}
	for _, rid := range mentionedTargets(mentionedIDs, memberIDs, actorID) {
		s.notifier.Emit(notifier.Event{
			Type:           notifier.TypeMessageMention,
			ActorID:        actorID,
			RecipientID:    rid,
			ConversationID: conversationID,
		})
	}

	return updated, memberIDs, nil
}

// DeleteMessage supprime « pour tout le monde » (tombstone) : vide le contenu
// chiffré et pose `deleted_at`. Autorisé à l'AUTEUR du message, ou à l'owner /
// admin d'un groupe ou d'une communauté (modération). Renvoie le message
// tombstoné + les ids des membres (diffusion WS).
func (s *MessageService) DeleteMessage(ctx context.Context, conversationID, messageID, actorID string) (*models.Message, []string, error) {
	member, err := s.requireMember(ctx, conversationID, actorID)
	if err != nil {
		return nil, nil, err
	}
	oid, err := parseID(messageID)
	if err != nil {
		return nil, nil, err
	}
	msg, err := s.repo.GetMessage(ctx, conversationID, oid)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil, ErrMessageNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	conv, err := s.getConversationByID(ctx, conversationID)
	if err != nil {
		return nil, nil, err
	}
	if !canDeleteMessage(member.Role, conv.Type, msg.SenderID, actorID) {
		return nil, nil, ErrCannotDelete
	}

	updated, err := s.repo.SoftDeleteMessage(ctx, conversationID, oid, time.Now())
	if err != nil {
		return nil, nil, translateNotFound(err)
	}

	memberIDs, err := s.repo.MemberIDs(ctx, conversationID)
	if err != nil {
		memberIDs = nil
	}
	return updated, memberIDs, nil
}

// canDeleteMessage : règle d'autorisation de la suppression d'un message
// (fonction PURE, testée). L'auteur peut toujours supprimer le sien ; dans un
// groupe ou une communauté, l'owner et l'admin peuvent supprimer ceux des autres
// (modération). Aucun droit de modération en DM (pas de hiérarchie).
func canDeleteMessage(actorRole, convType, senderID, actorID string) bool {
	if senderID == actorID {
		return true
	}
	if convType == models.TypeGroup || convType == models.TypeCommunity {
		return actorRole == models.MemberOwner || actorRole == models.MemberAdmin
	}
	return false
}

// mentionedTargets filtre les ids mentionnés pour ne garder que des membres
// réels, distincts, et différents de l'expéditeur. Fonction PURE (testée).
func mentionedTargets(mentioned, memberIDs []string, senderID string) []string {
	if len(mentioned) == 0 || len(memberIDs) == 0 {
		return nil
	}
	members := make(map[string]bool, len(memberIDs))
	for _, m := range memberIDs {
		members[m] = true
	}
	seen := make(map[string]bool, len(mentioned))
	out := make([]string, 0, len(mentioned))
	for _, id := range mentioned {
		if id == "" || id == senderID || seen[id] || !members[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

// messageTargets renvoie tous les membres à notifier pour un nouveau message,
// hors expéditeur. Les doublons ne sont pas attendus dans `memberIDs`, mais on
// déduplique par prudence.
func messageTargets(memberIDs []string, senderID string) []string {
	seen := make(map[string]bool, len(memberIDs))
	out := make([]string, 0, len(memberIDs))
	for _, id := range memberIDs {
		if id == "" || id == senderID || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
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
	s.attachReceipts(ctx, &v, userID)
	return &v, nil
}

// attachReceipts renseigne `MemberReceipts` (curseurs remis/ouvert des AUTRES
// membres) pour les accusés de réception côté expéditeur. DM et groupes
// uniquement (pas les communautés : trop de membres, accusés non pertinents).
// Best-effort : un échec laisse simplement les accusés vides.
func (s *MessageService) attachReceipts(ctx context.Context, view *models.ConversationView, requesterID string) {
	if view.Type != models.TypeDM && view.Type != models.TypeGroup {
		return
	}
	members, err := s.repo.ListMembers(ctx, view.ID)
	if err != nil {
		return
	}
	receipts := make([]models.MemberReceipt, 0, len(members))
	for _, m := range members {
		if m.UserID == requesterID {
			continue
		}
		receipts = append(receipts, models.MemberReceipt{
			UserID:      m.UserID,
			DeliveredAt: m.LastDeliveredAt,
			ReadAt:      m.LastReadAt,
		})
	}
	view.MemberReceipts = receipts
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

// isManageable : une conversation « administrable » (membres/suppression/renommage)
// est un groupe ou une communauté — pas un DM. Fonction PURE (testée).
func isManageable(convType string) bool {
	return convType == models.TypeGroup || convType == models.TypeCommunity
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

// convLess ordonne deux conversations pour la liste : épinglées d'abord (par
// date d'épinglage décroissante), puis par activité décroissante. Fonction PURE
// (testée). `a` passe avant `b` si la fonction renvoie true.
func convLess(a, b models.ConversationView) bool {
	ap, bp := a.PinnedAt != nil, b.PinnedAt != nil
	if ap != bp {
		return ap // l'épinglée passe devant la non-épinglée
	}
	if ap && bp && !a.PinnedAt.Equal(*b.PinnedAt) {
		return a.PinnedAt.After(*b.PinnedAt) // épinglée la plus récente d'abord
	}
	return a.UpdatedAt.After(b.UpdatedAt)
}

// buildView assemble la vue renvoyée au client (conversation + données du membre).
func buildView(conv *models.Conversation, m *models.Member) models.ConversationView {
	v := models.ConversationView{
		ID:         conv.ID.Hex(),
		Type:       conv.Type,
		MemberIDs:  conv.MemberIDs,
		Title:      conv.Title,
		TitleNonce: conv.TitleNonce,
		MyRole:     m.Role,
		MyEnvelope: m.KeyEnvelope,
		PinnedAt:   m.PinnedAt,
		LastReadAt: m.LastReadAt,
		Muted:      m.MutedAt != nil,
		CreatedBy:  conv.CreatedBy,
		CreatedAt:  conv.CreatedAt,
		UpdatedAt:  conv.UpdatedAt,
	}
	// Communauté : on remet la clé de contenu (détenue par le serveur) au membre.
	// buildView n'est appelé qu'après vérification d'appartenance → jamais à un
	// non-membre.
	if conv.Type == models.TypeCommunity {
		v.ContentKey = conv.ContentKey
	}
	return v
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

// clampOffset interdit un décalage négatif.
func clampOffset(offset int64) int64 {
	if offset < 0 {
		return 0
	}
	return offset
}
