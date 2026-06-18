// Package service porte la logique métier du report-service : validation des
// signalements (motif, longueur de texte par catégorie, entité), agrégation en
// tickets parents, cycle de vie, transfert et avertissements.
package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/webdad/report-service/internal/client"
	"github.com/webdad/report-service/internal/models"
	"github.com/webdad/report-service/internal/repository"
)

// Erreurs métier — traduites en codes HTTP par les handlers.
var (
	ErrNotFound   = errors.New("ticket introuvable")
	ErrInvalidID  = errors.New("identifiant invalide")
	ErrValidation = errors.New("requête invalide")
	ErrNotBug     = errors.New("seuls les tickets de bug peuvent être transférés")
	// ErrAlreadyReported : un utilisateur ne peut signaler une entité qu'une fois → 409.
	ErrAlreadyReported = errors.New("vous avez déjà signalé cet élément")
	// ErrReportingLocked : l'entité a été jugée conforme par la modération
	// (ticket `approved`) → tout nouveau signalement est refusé (verrou définitif) → 409.
	ErrReportingLocked = errors.New("cet élément a été validé par la modération et ne peut plus être signalé")
	// ErrAlreadyActioned : une sanction (retrait du contenu ou avertissement) a déjà
	// été posée sur le ticket → on ne peut plus le « valider » comme conforme
	// (décisions contradictoires) → 409.
	ErrAlreadyActioned = errors.New("une sanction a déjà été appliquée : validation impossible")
)

// Bornes de la liste de tickets.
const (
	DefaultLimit = 50
	MaxLimit     = 200
)

// ReopenThreshold : nombre de NOUVEAUX signalements requis pour rouvrir
// automatiquement un ticket clôturé. En dessous, le ticket reste clôturé (le
// signalement est tout de même enregistré). Un seul re-signalement ne défait
// donc pas la décision d'un modérateur ; il faut une récidive (≥ 2).
const ReopenThreshold int32 = 2

// validReasons : motifs acceptés (clés Mongo bornées). Voir models.
var validReasons = map[string]bool{
	models.ReasonInappropriate: true,
	models.ReasonOffensive:     true,
	models.ReasonBug:           true,
	models.ReasonSpam:          true,
	models.ReasonOther:         true,
}

// validEntityTypes : types d'entité signalable (hors `app`, réservé aux bugs).
var validEntityTypes = map[string]bool{
	models.EntityPost:         true,
	models.EntityMessage:      true,
	models.EntityGroupMessage: true,
	models.EntityProfile:      true,
}

var validStatuses = map[string]bool{
	models.StatusOpen:     true,
	models.StatusClosed:   true,
	models.StatusReopened: true,
}

// ReportService orchestre le dépôt et les règles métier.
type ReportService struct {
	repo  *repository.ReportRepository
	posts client.PostModerator
}

// NewReportService construit le service. `posts` pilote l'auto-masquage côté
// post-service (passer client.NoopPostModerator{} si non configuré / en test).
func NewReportService(r *repository.ReportRepository, posts client.PostModerator) *ReportService {
	if posts == nil {
		posts = client.NoopPostModerator{}
	}
	return &ReportService{repo: r, posts: posts}
}

// CreateReportInput : charge utile d'un nouveau signalement (déposé par tout
// utilisateur authentifié).
type CreateReportInput struct {
	Category      string
	EntityType    string
	EntityID      string
	EntityOwnerID string
	Reason        string
	Text          string
	// DisclosedContent : copie en clair divulguée par le signaleur pour un
	// contenu chiffré (message privé/groupe) — cf. models.Report.
	DisclosedContent string
	AttachmentID     string
}

// CreateReport valide puis enregistre un signalement. Modération → agrégé dans
// le ticket parent de l'entité ; bug → ticket autonome.
func (s *ReportService) CreateReport(ctx context.Context, reporterID string, in CreateReportInput) (*models.Ticket, error) {
	in.Reason = strings.TrimSpace(in.Reason)
	in.Text = strings.TrimSpace(in.Text)

	if !validReasons[in.Reason] {
		return nil, ErrValidation
	}

	report := models.Report{
		ReporterID:       reporterID,
		Reason:           in.Reason,
		Text:             in.Text,
		DisclosedContent: strings.TrimSpace(in.DisclosedContent),
		AttachmentID:     strings.TrimSpace(in.AttachmentID),
		CreatedAt:        time.Now(),
	}

	switch in.Category {
	case models.CategoryBug:
		// Limite 500 caractères pour un rapport de bug.
		if len([]rune(in.Text)) > models.MaxBugText {
			return nil, ErrValidation
		}
		// L'entité (post/profil/message) est conservée si fournie → visible côté
		// admin ; sinon bug applicatif générique (entity_type=app).
		entityType := ""
		if validEntityTypes[in.EntityType] {
			entityType = in.EntityType
		}
		return s.repo.InsertBugTicket(ctx, entityType, strings.TrimSpace(in.EntityID), strings.TrimSpace(in.EntityOwnerID), report)

	case models.CategoryModeration:
		// Limite 255 caractères pour un signalement de modération.
		if len([]rune(in.Text)) > models.MaxModerationText {
			return nil, ErrValidation
		}
		entityID := strings.TrimSpace(in.EntityID)
		if !validEntityTypes[in.EntityType] || entityID == "" {
			return nil, ErrValidation
		}
		// Verrou : une entité jugée conforme par la modération (`approved`) ne
		// peut plus être signalée (décision terminale).
		if existing, gerr := s.repo.GetModerationByEntity(ctx, in.EntityType, entityID); gerr == nil && existing.Status == models.StatusApproved {
			return nil, ErrReportingLocked
		}
		t, err := s.repo.UpsertModerationReport(ctx, in.EntityType, entityID, strings.TrimSpace(in.EntityOwnerID), report)
		if errors.Is(err, repository.ErrAlreadyReported) {
			return nil, ErrAlreadyReported
		}
		if err != nil {
			return nil, err
		}
		// Réouverture automatique : un ticket CLÔTURÉ qui atteint le seuil de
		// nouveaux signalements depuis sa clôture se rouvre seul. En deçà, il
		// reste clôturé (le signalement est quand même empilé).
		if t.Status == models.StatusClosed && t.ReportsSinceClosed >= ReopenThreshold {
			if reopened, rerr := s.repo.AutoReopen(ctx, t.ID, report.CreatedAt); rerr == nil {
				t = reopened
			}
			// best-effort : si la réouverture échoue, le signalement reste enregistré.
		}
		// Auto-masquage : un POST de modération qui atteint le seuil configuré est
		// masqué (best-effort) en attendant la décision d'un modérateur. Ne
		// s'applique JAMAIS aux bugs (catégorie admin) ni à un ticket déjà
		// terminal/clôturé. Journalise l'action sur le ticket (une seule fois).
		s.maybeAutoHide(ctx, t)
		return t, nil

	default:
		return nil, ErrValidation
	}
}

// maybeAutoHide masque un POST (best-effort) quand son ticket de modération
// atteint le seuil configuré et qu'il n'a pas DÉJÀ été auto-masqué. Le masquage
// effectif est posé côté post-service (barrière de visibilité serveur) ; on
// journalise l'action sur le ticket pour la traçabilité et l'idempotence.
func (s *ReportService) maybeAutoHide(ctx context.Context, t *models.Ticket) {
	if t == nil || t.EntityType != models.EntityPost || t.EntityID == "" {
		return
	}
	// Pas d'auto-masquage sur une décision déjà terminale (approuvé/clôturé).
	if t.Status == models.StatusApproved || t.Status == models.StatusClosed {
		return
	}
	settings, err := s.repo.GetSettings(ctx)
	if err != nil {
		return // best-effort : pas de réglage lisible → on s'abstient
	}
	if settings.AutoHideThreshold <= 0 || t.ReportCount < settings.AutoHideThreshold {
		return
	}
	// Idempotence : ne masquer (et journaliser) qu'une fois.
	for _, a := range t.Actions {
		if a.Type == models.ActionAutoHidden {
			return
		}
	}
	action := models.Action{Type: models.ActionAutoHidden, CreatedAt: time.Now()}
	if updated, aerr := s.repo.AddAction(ctx, t.ID, action, nil); aerr == nil {
		*t = *updated
	}
	s.posts.AutoHide(t.EntityID)
}

// hasBlockingSanction indique qu'une sanction TERMINALE pour la validation a déjà
// été posée sur le ticket (retrait du contenu ou avertissement de l'auteur) : on
// ne peut alors plus juger l'entité « conforme » (décisions contradictoires).
func hasBlockingSanction(actions []models.Action) bool {
	for _, a := range actions {
		if a.Type == models.ActionContentRemoved || a.Type == models.ActionWarned {
			return true
		}
	}
	return false
}

// Approve marque une entité comme CONFORME (décision terminale du modérateur) :
// statut → approved, action tracée, démasquage du post si auto-masqué, et verrou
// de re-signalement (toute tentative ultérieure est refusée, cf. CreateReport).
func (s *ReportService) Approve(ctx context.Context, id, moderatorID string) (*models.Ticket, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrInvalidID
	}
	// Garde-fou : valider (« conforme ») contredit une sanction déjà posée — on
	// refuse si le contenu a été retiré ou l'auteur averti sur ce ticket.
	current, err := s.repo.GetTicket(ctx, oid)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if hasBlockingSanction(current.Actions) {
		return nil, ErrAlreadyActioned
	}
	action := models.Action{ModeratorID: moderatorID, Type: models.ActionApproved, CreatedAt: time.Now()}
	t, err := s.repo.AddAction(ctx, oid, action, bson.M{"status": models.StatusApproved, "reports_since_closed": int32(0)})
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if t.EntityType == models.EntityPost && t.EntityID != "" {
		s.posts.AutoUnhide(t.EntityID)
	}
	return t, nil
}

// Settings renvoie la configuration runtime de la modération (seuil d'auto-masquage).
func (s *ReportService) Settings(ctx context.Context) (*models.Settings, error) {
	return s.repo.GetSettings(ctx)
}

// UpdateThreshold fixe le seuil d'auto-masquage (≥ 0 ; 0 = désactivé). Réservé
// à l'administrateur (gating au niveau route).
func (s *ReportService) UpdateThreshold(ctx context.Context, threshold int32) (*models.Settings, error) {
	if threshold < 0 {
		return nil, ErrValidation
	}
	return s.repo.UpdateThreshold(ctx, threshold)
}

// ListTickets renvoie les tickets filtrés (tri décroissant par volume côté repo).
func (s *ReportService) ListTickets(ctx context.Context, f repository.TicketFilter, limit int64) ([]models.Ticket, error) {
	if f.Status != "" && !validStatuses[f.Status] {
		return nil, ErrValidation
	}
	if limit <= 0 || limit > MaxLimit {
		limit = DefaultLimit
	}
	return s.repo.ListTickets(ctx, f, limit)
}

// GetTicket renvoie un ticket par id (404 si absent / id mal formé).
func (s *ReportService) GetTicket(ctx context.Context, id string) (*models.Ticket, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrInvalidID
	}
	t, err := s.repo.GetTicket(ctx, oid)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

// Reply ajoute une réponse interne au ticket, en enregistrant le modérateur.
func (s *ReportService) Reply(ctx context.Context, id, moderatorID, text string) (*models.Ticket, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrInvalidID
	}
	text = strings.TrimSpace(text)
	if text == "" || len([]rune(text)) > models.MaxBugText {
		return nil, ErrValidation
	}
	action := models.Action{ModeratorID: moderatorID, Type: models.ActionReply, Text: text, CreatedAt: time.Now()}
	t, err := s.repo.AddAction(ctx, oid, action, nil)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

// ChangeStatus applique un nouvel état (open|closed|reopened) et trace l'action.
func (s *ReportService) ChangeStatus(ctx context.Context, id, moderatorID, status string) (*models.Ticket, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrInvalidID
	}
	if !validStatuses[status] {
		return nil, ErrValidation
	}
	action := models.Action{ModeratorID: moderatorID, Type: models.ActionStatusChange, Status: status, CreatedAt: time.Now()}
	// Tout changement de statut manuel remet le compteur de réouverture à zéro :
	// le seuil ne compte que les signalements reçus DEPUIS la dernière décision.
	t, err := s.repo.AddAction(ctx, oid, action, bson.M{"status": status, "reports_since_closed": int32(0)})
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

// LogRemoval inscrit dans le journal du ticket qu'un modérateur a RETIRÉ le
// contenu signalé (post masqué, message supprimé…). Ne change PAS le statut :
// la clôture reste une décision manuelle du modérateur.
func (s *ReportService) LogRemoval(ctx context.Context, id, moderatorID string) (*models.Ticket, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrInvalidID
	}
	action := models.Action{ModeratorID: moderatorID, Type: models.ActionContentRemoved, CreatedAt: time.Now()}
	t, err := s.repo.AddAction(ctx, oid, action, nil)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

// Transfer déplace un ticket de bug vers la modération (réservé aux admins,
// gating au niveau route). Refuse un ticket qui n'est pas un bug.
func (s *ReportService) Transfer(ctx context.Context, id, adminID string) (*models.Ticket, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrInvalidID
	}
	current, err := s.repo.GetTicket(ctx, oid)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if current.Category != models.CategoryBug {
		return nil, ErrNotBug
	}
	action := models.Action{ModeratorID: adminID, Type: models.ActionTransfer, CreatedAt: time.Now()}
	t, err := s.repo.AddAction(ctx, oid, action, bson.M{"category": models.CategoryModeration})
	if err != nil {
		return nil, err
	}
	return t, nil
}

// IssueWarning crée un avertissement à destination d'un utilisateur.
func (s *ReportService) IssueWarning(ctx context.Context, issuedBy, targetUserID, ticketID, message string) (*models.Warning, error) {
	targetUserID = strings.TrimSpace(targetUserID)
	message = strings.TrimSpace(message)
	if targetUserID == "" || message == "" || len([]rune(message)) > models.MaxWarningMessage {
		return nil, ErrValidation
	}
	ticketID = strings.TrimSpace(ticketID)
	w := &models.Warning{
		TargetUserID: targetUserID,
		TicketID:     ticketID,
		Message:      message,
		IssuedBy:     issuedBy,
		Acknowledged: false,
		CreatedAt:    time.Now(),
	}
	created, err := s.repo.CreateWarning(ctx, w)
	if err != nil {
		return nil, err
	}
	// Avertissement émis DEPUIS un ticket → on le journalise (`warned`) pour que le
	// ticket soit auto-descriptif : trace de la sanction (cf. garde-fou Approve) et
	// affichage dans le journal des actions. Best-effort : un id de ticket invalide
	// ou un ticket disparu n'invalide pas l'avertissement, déjà créé.
	if ticketID != "" {
		if oid, oerr := bson.ObjectIDFromHex(ticketID); oerr == nil {
			action := models.Action{ModeratorID: issuedBy, Type: models.ActionWarned, CreatedAt: time.Now()}
			_, _ = s.repo.AddAction(ctx, oid, action, nil)
		}
	}
	return created, nil
}

// WarningCount renvoie le nombre total d'avertissements reçus par un utilisateur
// (profil de risque côté modération).
func (s *ReportService) WarningCount(ctx context.Context, userID string) (int64, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return 0, ErrValidation
	}
	return s.repo.CountWarnings(ctx, userID)
}

// PendingWarnings renvoie les avertissements non acquittés de l'utilisateur.
func (s *ReportService) PendingWarnings(ctx context.Context, userID string) ([]models.Warning, error) {
	return s.repo.PendingWarnings(ctx, userID)
}

// AckWarning acquitte un avertissement de l'utilisateur courant.
func (s *ReportService) AckWarning(ctx context.Context, id, userID string) error {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return ErrInvalidID
	}
	err = s.repo.AckWarning(ctx, oid, userID)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return ErrNotFound
	}
	return err
}
