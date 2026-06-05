// Package repository encapsule l'accès aux données PostgreSQL du service user.
// Aucune logique métier ici : uniquement des requêtes SQL paramétrées.
package repository

import (
	"database/sql"
	"strings"

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

const userColumns = `id, username, is_active, created_at, updated_at, username_changed_at`

// userColumnsU : mêmes colonnes préfixées par l'alias `u` (jointures follows).
const userColumnsU = `u.id, u.username, u.is_active, u.created_at, u.updated_at, u.username_changed_at`

// detailColumns : userColumns + compteurs du graphe social (sous-requêtes
// corrélées). Réservé aux vues « profil » (un seul utilisateur).
const detailColumns = userColumns + `,
	(SELECT COUNT(*) FROM follows WHERE following_id = users.id) AS follower_count,
	(SELECT COUNT(*) FROM follows WHERE follower_id  = users.id) AS following_count`

type scanner interface{ Scan(...any) error }

// scanUser projette une ligne vers un *User.
func scanUser(row scanner) (*models.User, error) {
	u := &models.User{}
	if err := row.Scan(&u.ID, &u.Username, &u.IsActive, &u.CreatedAt, &u.UpdatedAt, &u.UsernameChangedAt); err != nil {
		return nil, err
	}
	return u, nil
}

// scanDetails projette une ligne userColumns + compteurs vers *UserDetails.
func scanDetails(row scanner) (*models.UserDetails, error) {
	d := &models.UserDetails{}
	if err := row.Scan(
		&d.ID, &d.Username, &d.IsActive, &d.CreatedAt, &d.UpdatedAt, &d.UsernameChangedAt,
		&d.FollowerCount, &d.FollowingCount,
	); err != nil {
		return nil, err
	}
	return d, nil
}

// Create insère un nouvel utilisateur avec un id imposé (= credentials.id).
func (r *UserRepository) Create(id, username string) (*models.User, error) {
	const q = `
		INSERT INTO users (id, username)
		VALUES ($1, $2)
		RETURNING ` + userColumns
	return scanUser(r.db.QueryRow(q, id, username))
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

// Update modifie le username (COALESCE : nil = inchangé). Le nom affiché et
// les autres champs décoratifs se modifient via profil-service.
//
// username_changed_at est posé à NOW() UNIQUEMENT si le username change
// réellement (le CASE compare $2 à l'ancienne valeur — Postgres évalue les
// expressions du SET sur la ligne d'origine), pour capturer la baseline du
// cooldown sans la réinitialiser sur un PATCH sans-op.
func (r *UserRepository) Update(id string, username *string) (*models.User, error) {
	const q = `
		UPDATE users
		SET username = COALESCE($2, username),
		    username_changed_at = CASE
		        WHEN $2 IS NOT NULL AND $2 <> username THEN NOW()
		        ELSE username_changed_at
		    END
		WHERE id = $1
		RETURNING ` + userColumns
	return scanUser(r.db.QueryRow(q, id, username))
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

// Search retourne les utilisateurs actifs dont le username contient `term`
// (insensible à la casse), triés par handle. Les wildcards LIKE de `term` sont
// échappés (recherche littérale, pas d'injection de motif).
func (r *UserRepository) Search(term string, limit, offset int) ([]models.User, error) {
	const q = `
		SELECT ` + userColumns + `
		FROM users
		WHERE is_active = true AND username ILIKE $1
		ORDER BY username ASC
		LIMIT $2 OFFSET $3`
	return r.queryUsers(q, "%"+escapeLike(term)+"%", limit, offset)
}

// ListByFollowers retourne les utilisateurs actifs triés par nombre d'abonnés
// décroissant (« Qui suivre »). Le tri réutilise la sous-requête corrélée des
// compteurs (non sélectionnée ici : seules les colonnes user suffisent).
func (r *UserRepository) ListByFollowers(limit, offset int) ([]models.User, error) {
	const q = `
		SELECT ` + userColumns + `
		FROM users
		WHERE is_active = true
		ORDER BY (SELECT COUNT(*) FROM follows WHERE following_id = users.id) DESC,
		         created_at DESC
		LIMIT $1 OFFSET $2`
	return r.queryUsers(q, limit, offset)
}

// escapeLike neutralise les métacaractères LIKE (`%`, `_`, `\`) pour que la
// saisie utilisateur soit traitée littéralement dans un motif ILIKE.
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
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
