// Package repository : accès aux données Mongo du report-service. Les méthodes
// renvoient les erreurs brutes du driver (notamment mongo.ErrNoDocuments) ; la
// traduction en erreurs métier est faite par la couche service.
package repository

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/webdad/report-service/internal/models"
)

// ErrAlreadyReported : ce rapporteur a déjà signalé cette entité (un seul
// signalement par (utilisateur, entité) en modération). Traduit en 409 plus haut.
var ErrAlreadyReported = errors.New("entité déjà signalée par cet utilisateur")

// ReportRepository encapsule les collections `tickets`, `warnings` et `settings`.
type ReportRepository struct {
	tickets  *mongo.Collection
	warnings *mongo.Collection
	settings *mongo.Collection
}

func NewReportRepository(db *mongo.Database) *ReportRepository {
	return &ReportRepository{
		tickets:  db.Collection("tickets"),
		warnings: db.Collection("warnings"),
		settings: db.Collection("settings"),
	}
}

// TicketFilter regroupe les filtres cumulables de la liste des tickets.
type TicketFilter struct {
	Category   string     // "moderation" | "bug" (obligatoire en pratique)
	Status     string     // "" = tous
	MinReports int        // 0 = pas de seuil
	Since      *time.Time // borne basse du dernier signalement
	Until      *time.Time // borne haute du dernier signalement
}

// UpsertModerationReport agrège un signalement de MODÉRATION dans le ticket
// parent de l'entité (clé (entity_type, entity_id)) : crée le ticket s'il
// n'existe pas, sinon empile le signalement, incrémente le compteur global et le
// compteur du motif, et rafraîchit `last_reported_at`. C'est le cœur de
// l'agrégation « un seul ticket parent par entité ». Renvoie le ticket à jour.
func (r *ReportRepository) UpsertModerationReport(ctx context.Context, entityType, entityID, entityOwnerID string, report models.Report) (*models.Ticket, error) {
	now := report.CreatedAt
	update := bson.M{
		"$push": bson.M{"reports": report},
		"$inc": bson.M{
			"report_count":                 int32(1),
			"reports_since_closed":         int32(1), // remis à 0 à chaque changement de statut
			"reason_tags." + report.Reason: int32(1),
		},
		"$set": bson.M{"last_reported_at": now, "updated_at": now},
		"$setOnInsert": bson.M{
			"category":        models.CategoryModeration,
			"entity_type":     entityType,
			"entity_id":       entityID,
			"entity_owner_id": entityOwnerID,
			"status":          models.StatusOpen,
			"created_at":      now,
		},
	}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)

	// Le filtre EXCLUT un ticket où ce rapporteur figure déjà → un même
	// utilisateur ne peut signaler une entité qu'UNE fois. Conséquence :
	//   - ticket inexistant → upsert insère (1er signalement) ;
	//   - ticket existant, rapporteur nouveau → match → empile ;
	//   - ticket existant, rapporteur déjà présent → pas de match → tentative
	//     d'insert → violation de l'index unique partiel (E11000).
	// Une E11000 peut aussi venir d'une COURSE (deux 1ers signalements simultanés
	// créent le même ticket) : on réessaie alors UNE fois (le ticket existe
	// désormais, on empilera si le rapporteur est nouveau, sinon doublon confirmé).
	for attempt := 0; attempt < 2; attempt++ {
		filter := bson.M{
			"category":            models.CategoryModeration,
			"entity_type":         entityType,
			"entity_id":           entityID,
			"reports.reporter_id": bson.M{"$ne": report.ReporterID},
		}
		var out models.Ticket
		err := r.tickets.FindOneAndUpdate(ctx, filter, update, opts).Decode(&out)
		if err == nil {
			return &out, nil
		}
		if mongo.IsDuplicateKeyError(err) {
			if attempt == 0 {
				continue // course probable : on retente une fois
			}
			return nil, ErrAlreadyReported // doublon confirmé après retry
		}
		return nil, err
	}
	return nil, ErrAlreadyReported
}

// InsertBugTicket crée un ticket de BUG autonome (pas d'agrégation par entité).
// L'entité signalée est CONSERVÉE quand elle existe (un bug signalé SUR un post
// ou un profil garde sa référence → la modération/admin peut voir l'élément) ;
// faute d'entité, on retombe sur `app` (bug applicatif générique).
func (r *ReportRepository) InsertBugTicket(ctx context.Context, entityType, entityID, entityOwnerID string, report models.Report) (*models.Ticket, error) {
	now := report.CreatedAt
	if entityType == "" {
		entityType = models.EntityApp
	}
	t := models.Ticket{
		Category:       models.CategoryBug,
		EntityType:     entityType,
		EntityID:       entityID,
		EntityOwnerID:  entityOwnerID,
		Status:         models.StatusOpen,
		ReportCount:    1,
		ReasonTags:     map[string]int32{report.Reason: 1},
		Reports:        []models.Report{report},
		LastReportedAt: now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	res, err := r.tickets.InsertOne(ctx, t)
	if err != nil {
		return nil, err
	}
	if oid, ok := res.InsertedID.(bson.ObjectID); ok {
		t.ID = oid
	}
	return &t, nil
}

// ListTickets renvoie les tickets filtrés et triés en DEUX paliers :
//  1. les tickets ACTIFS (ouverts/rouverts) d'abord, puis les CLÔTURÉS ;
//  2. à l'intérieur de chaque palier, par nombre de signalements décroissant
//     (puis dernier signalement décroissant à volume égal).
//
// Le tri sur « actif vs clôturé » nécessite une clé calculée (`_closed`), d'où
// un pipeline d'agrégation plutôt qu'un simple Find/Sort.
func (r *ReportRepository) ListTickets(ctx context.Context, f TicketFilter, limit int64) ([]models.Ticket, error) {
	query := bson.M{}
	if f.Category != "" {
		query["category"] = f.Category
	}
	if f.Status != "" {
		query["status"] = f.Status
	}
	if f.MinReports > 0 {
		query["report_count"] = bson.M{"$gte": int32(f.MinReports)}
	}
	if f.Since != nil || f.Until != nil {
		rng := bson.M{}
		if f.Since != nil {
			rng["$gte"] = *f.Since
		}
		if f.Until != nil {
			rng["$lte"] = *f.Until
		}
		query["last_reported_at"] = rng
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: query}},
		// _closed = 1 pour un ticket clôturé, 0 sinon (ouvert/rouvert = actif).
		{{Key: "$addFields", Value: bson.D{{Key: "_closed", Value: bson.D{
			{Key: "$cond", Value: bson.A{bson.D{{Key: "$eq", Value: bson.A{"$status", models.StatusClosed}}}, 1, 0}},
		}}}}},
		{{Key: "$sort", Value: bson.D{
			{Key: "_closed", Value: 1},           // actifs avant clôturés
			{Key: "report_count", Value: -1},     // puis volume décroissant
			{Key: "last_reported_at", Value: -1}, // puis le plus récent
		}}},
		{{Key: "$limit", Value: limit}},
		{{Key: "$project", Value: bson.D{{Key: "_closed", Value: 0}}}},
	}

	cursor, err := r.tickets.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	out := []models.Ticket{}
	if err := cursor.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetTicket renvoie un ticket par id. mongo.ErrNoDocuments si absent.
func (r *ReportRepository) GetTicket(ctx context.Context, id bson.ObjectID) (*models.Ticket, error) {
	var out models.Ticket
	if err := r.tickets.FindOne(ctx, bson.M{"_id": id}).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetModerationByEntity renvoie le ticket parent de modération d'une entité
// (clé (entity_type, entity_id)), s'il existe. Sert au verrou de re-signalement :
// un ticket `approved` interdit tout nouveau signalement de l'entité.
// mongo.ErrNoDocuments si aucun ticket n'existe encore pour l'entité.
func (r *ReportRepository) GetModerationByEntity(ctx context.Context, entityType, entityID string) (*models.Ticket, error) {
	var out models.Ticket
	filter := bson.M{"category": models.CategoryModeration, "entity_type": entityType, "entity_id": entityID}
	if err := r.tickets.FindOne(ctx, filter).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

// AddAction empile une action de modération et rafraîchit `updated_at`. Les
// mutations optionnelles (statut, catégorie) sont fusionnées dans `$set`.
// Renvoie le ticket à jour. mongo.ErrNoDocuments si le ticket est absent.
func (r *ReportRepository) AddAction(ctx context.Context, id bson.ObjectID, action models.Action, set bson.M) (*models.Ticket, error) {
	merged := bson.M{"updated_at": action.CreatedAt}
	for k, v := range set {
		merged[k] = v
	}
	update := bson.M{"$push": bson.M{"actions": action}, "$set": merged}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var out models.Ticket
	if err := r.tickets.FindOneAndUpdate(ctx, bson.M{"_id": id}, update, opts).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

// AutoReopen rouvre un ticket clôturé que le seuil de re-signalements a atteint :
// statut → reopened, compteur remis à zéro, action `auto_reopen` tracée (sans
// modérateur : c'est le système). Renvoie le ticket à jour.
func (r *ReportRepository) AutoReopen(ctx context.Context, id bson.ObjectID, now time.Time) (*models.Ticket, error) {
	action := models.Action{Type: models.ActionAutoReopen, CreatedAt: now}
	return r.AddAction(ctx, id, action, bson.M{
		"status":               models.StatusReopened,
		"reports_since_closed": int32(0),
	})
}

// GetSettings renvoie le document singleton de configuration (`_id="global"`).
// Le seed au boot (EnsureSchema) garantit sa présence ; en repli défensif, un
// document absent renvoie le défaut plutôt qu'une erreur.
func (r *ReportRepository) GetSettings(ctx context.Context) (*models.Settings, error) {
	var out models.Settings
	err := r.settings.FindOne(ctx, bson.M{"_id": models.SettingsSingletonID}).Decode(&out)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return &models.Settings{ID: models.SettingsSingletonID, AutoHideThreshold: models.DefaultAutoHideThreshold}, nil
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateThreshold fixe le seuil d'auto-masquage (upsert idempotent). Renvoie la
// configuration à jour.
func (r *ReportRepository) UpdateThreshold(ctx context.Context, threshold int32) (*models.Settings, error) {
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	var out models.Settings
	err := r.settings.FindOneAndUpdate(ctx,
		bson.M{"_id": models.SettingsSingletonID},
		bson.M{"$set": bson.M{"auto_hide_threshold": threshold}},
		opts,
	).Decode(&out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateWarning insère un avertissement (non acquitté).
func (r *ReportRepository) CreateWarning(ctx context.Context, w *models.Warning) (*models.Warning, error) {
	res, err := r.warnings.InsertOne(ctx, w)
	if err != nil {
		return nil, err
	}
	if oid, ok := res.InsertedID.(bson.ObjectID); ok {
		w.ID = oid
	}
	return w, nil
}

// CountWarnings renvoie le nombre TOTAL d'avertissements reçus par un utilisateur
// (acquittés ou non) — sert au « profil de risque » affiché côté modération.
func (r *ReportRepository) CountWarnings(ctx context.Context, userID string) (int64, error) {
	return r.warnings.CountDocuments(ctx, bson.M{"target_user_id": userID})
}

// PendingWarnings renvoie les avertissements non acquittés d'un utilisateur,
// du plus ancien au plus récent (on affiche d'abord le plus vieux).
func (r *ReportRepository) PendingWarnings(ctx context.Context, userID string) ([]models.Warning, error) {
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}})
	cursor, err := r.warnings.Find(ctx, bson.M{"target_user_id": userID, "acknowledged": false}, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	out := []models.Warning{}
	if err := cursor.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// AckWarning acquitte un avertissement (réservé à sa cible via le filtre).
// mongo.ErrNoDocuments si absent / pas destiné à l'utilisateur.
func (r *ReportRepository) AckWarning(ctx context.Context, id bson.ObjectID, userID string) error {
	now := time.Now()
	res, err := r.warnings.UpdateOne(ctx,
		bson.M{"_id": id, "target_user_id": userID},
		bson.M{"$set": bson.M{"acknowledged": true, "acknowledged_at": now}},
	)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}
