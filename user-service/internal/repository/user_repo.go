// Package repository encapsule l'accès aux données PostgreSQL du service user.
// Aucune logique métier ici : uniquement des requêtes SQL paramétrées.
package repository

import (
	"database/sql"

	"github.com/webdad/user-service/internal/models"
)

// UserRepository porte le pool de connexions.
type UserRepository struct {
	db *sql.DB
}

// New construit le repository.
func New(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

const userColumns = `id, username, display_name, is_active, created_at, updated_at`

// userColumnsU : mêmes colonnes préfixées par l'alias `u` (jointures follows).
const userColumnsU = `u.id, u.username, u.display_name, u.is_active, u.created_at, u.updated_at`

// detailColumns : userColumns + compteurs du graphe social (sous-requêtes
// corrélées). Réservé aux vues « profil » (un seul utilisateur).
const detailColumns = userColumns + `,
	(SELECT COUNT(*) FROM follows WHERE following_id = users.id) AS follower_count,
	(SELECT COUNT(*) FROM follows WHERE follower_id  = users.id) AS following_count`

type scanner interface{ Scan(...any) error }

// scanUser projette une ligne (display_name nullable) vers un *User.
func scanUser(row scanner) (*models.User, error) {
	u := &models.User{}
	var displayName sql.NullString
	if err := row.Scan(&u.ID, &u.Username, &displayName, &u.IsActive, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return nil, err
	}
	u.DisplayName = displayName.String
	return u, nil
}

// scanDetails projette une ligne userColumns + compteurs vers *UserDetails.
func scanDetails(row scanner) (*models.UserDetails, error) {
	d := &models.UserDetails{}
	var displayName sql.NullString
	if err := row.Scan(
		&d.ID, &d.Username, &displayName, &d.IsActive, &d.CreatedAt, &d.UpdatedAt,
		&d.FollowerCount, &d.FollowingCount,
	); err != nil {
		return nil, err
	}
	d.DisplayName = displayName.String
	return d, nil
}

// Create insère un nouvel utilisateur avec un id imposé (= credentials.id).
func (r *UserRepository) Create(id, username, displayName string) (*models.User, error) {
	const q = `
		INSERT INTO users (id, username, display_name)
		VALUES ($1, $2, NULLIF($3, ''))
		RETURNING ` + userColumns
	return scanUser(r.db.QueryRow(q, id, username, displayName))
}

// ExistsByID indique si un utilisateur existe (sans charger la ligne).
func (r *UserRepository) ExistsByID(id string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`, id).Scan(&exists)
	return exists, err
}

// GetByID retourne un utilisateur par son id (sql.ErrNoRows si absent).
func (r *UserRepository) GetByID(id string) (*models.User, error) {
	const q = `SELECT ` + userColumns + ` FROM users WHERE id = $1`
	return scanUser(r.db.QueryRow(q, id))
}

// GetDetailsByID retourne un utilisateur + compteurs (sql.ErrNoRows si absent).
func (r *UserRepository) GetDetailsByID(id string) (*models.UserDetails, error) {
	const q = `SELECT ` + detailColumns + ` FROM users WHERE id = $1`
	return scanDetails(r.db.QueryRow(q, id))
}

// GetDetailsByUsername retourne un utilisateur + compteurs par son handle.
func (r *UserRepository) GetDetailsByUsername(username string) (*models.UserDetails, error) {
	const q = `SELECT ` + detailColumns + ` FROM users WHERE username = $1`
	return scanDetails(r.db.QueryRow(q, username))
}

// List retourne une page d'utilisateurs actifs (plus récents d'abord).
// Les comptes désactivés (is_active=false) sont masqués.
func (r *UserRepository) List(limit, offset int) ([]models.User, error) {
	const q = `
		SELECT ` + userColumns + `
		FROM users
		WHERE is_active = true
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`
	return r.queryUsers(q, limit, offset)
}

// Update modifie username et/ou display_name (COALESCE : nil = inchangé ;
// chaîne vide sur display_name = vidage volontaire).
func (r *UserRepository) Update(id string, username, displayName *string) (*models.User, error) {
	const q = `
		UPDATE users
		SET username     = COALESCE($2, username),
		    display_name = COALESCE($3, display_name)
		WHERE id = $1
		RETURNING ` + userColumns
	return scanUser(r.db.QueryRow(q, id, username, displayName))
}

// SoftDelete désactive un compte (is_active=false) sans le supprimer.
// Retourne sql.ErrNoRows si l'utilisateur n'existe pas.
func (r *UserRepository) SoftDelete(id string) error {
	const q = `UPDATE users SET is_active = false WHERE id = $1`
	res, err := r.db.Exec(q, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ─── Graphe social (follows) ──────────────────────────────────────────────

// Follow crée la relation follower→following (idempotent : ON CONFLICT).
// Les deux ids doivent exister (FK). La contrainte no_self_follow empêche
// l'auto-follow au niveau base.
func (r *UserRepository) Follow(followerID, followingID string) error {
	const q = `
		INSERT INTO follows (follower_id, following_id)
		VALUES ($1, $2)
		ON CONFLICT (follower_id, following_id) DO NOTHING`
	_, err := r.db.Exec(q, followerID, followingID)
	return err
}

// Unfollow supprime la relation (idempotent : aucune erreur si absente).
func (r *UserRepository) Unfollow(followerID, followingID string) error {
	const q = `DELETE FROM follows WHERE follower_id = $1 AND following_id = $2`
	_, err := r.db.Exec(q, followerID, followingID)
	return err
}

// ListFollowers retourne les utilisateurs qui suivent `id` (paginé).
func (r *UserRepository) ListFollowers(id string, limit, offset int) ([]models.User, error) {
	const q = `
		SELECT ` + userColumnsU + `
		FROM follows f
		JOIN users u ON u.id = f.follower_id
		WHERE f.following_id = $1 AND u.is_active = true
		ORDER BY f.created_at DESC
		LIMIT $2 OFFSET $3`
	return r.queryUsers(q, id, limit, offset)
}

// ListFollowing retourne les utilisateurs suivis par `id` (paginé).
func (r *UserRepository) ListFollowing(id string, limit, offset int) ([]models.User, error) {
	const q = `
		SELECT ` + userColumnsU + `
		FROM follows f
		JOIN users u ON u.id = f.following_id
		WHERE f.follower_id = $1 AND u.is_active = true
		ORDER BY f.created_at DESC
		LIMIT $2 OFFSET $3`
	return r.queryUsers(q, id, limit, offset)
}

// queryUsers exécute une requête renvoyant des lignes userColumns.
func (r *UserRepository) queryUsers(q string, args ...any) ([]models.User, error) {
	rows, err := r.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	users := make([]models.User, 0)
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, *u)
	}
	return users, rows.Err()
}
