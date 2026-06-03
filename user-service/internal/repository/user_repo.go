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

// scanUser projette une ligne (display_name nullable) vers un *User.
func scanUser(row interface{ Scan(...any) error }) (*models.User, error) {
	u := &models.User{}
	var displayName sql.NullString
	if err := row.Scan(&u.ID, &u.Username, &displayName, &u.IsActive, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return nil, err
	}
	u.DisplayName = displayName.String
	return u, nil
}

// Create insère un nouvel utilisateur avec un id imposé (= credentials.id).
func (r *UserRepository) Create(id, username, displayName string) (*models.User, error) {
	const q = `
		INSERT INTO users (id, username, display_name)
		VALUES ($1, $2, NULLIF($3, ''))
		RETURNING ` + userColumns
	return scanUser(r.db.QueryRow(q, id, username, displayName))
}

// Upsert crée l'utilisateur s'il n'existe pas (par id), sinon le retourne
// inchangé. Sert au provisioning paresseux depuis les claims JWT.
func (r *UserRepository) Upsert(id, username, displayName string) (*models.User, error) {
	const q = `
		INSERT INTO users (id, username, display_name)
		VALUES ($1, $2, NULLIF($3, ''))
		ON CONFLICT (id) DO UPDATE SET id = users.id
		RETURNING ` + userColumns
	return scanUser(r.db.QueryRow(q, id, username, displayName))
}

// GetByID retourne un utilisateur par son id (sql.ErrNoRows si absent).
func (r *UserRepository) GetByID(id string) (*models.User, error) {
	const q = `SELECT ` + userColumns + ` FROM users WHERE id = $1`
	return scanUser(r.db.QueryRow(q, id))
}

// List retourne une page d'utilisateurs (ordre : plus récents d'abord).
func (r *UserRepository) List(limit, offset int) ([]models.User, error) {
	const q = `
		SELECT ` + userColumns + `
		FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.db.Query(q, limit, offset)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	users := make([]models.User, 0, limit)
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, *u)
	}
	return users, rows.Err()
}

// Update modifie username et/ou display_name (COALESCE : nil = inchangé).
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
