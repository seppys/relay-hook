package auth

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const PgErrUniqueConstraintViolationCode = "23505"
const PgErrForeignKeyViolationCode = "23503"

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetList(ctx context.Context) ([]User, error) {
	var users []User
	rows, err := r.db.Query(ctx, `SELECT id, username, password_hash FROM users`)
	if err != nil {
		return nil, ErrInternalServer
	}

	defer rows.Close()
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Username, &user.PasswordHash); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func (r *Repository) GetById(ctx context.Context, id string) (User, error) {
	var user User
	row := r.db.QueryRow(ctx, `SELECT id, username, password_hash FROM users WHERE id = $1`, id)
	if err := row.Scan(&user.ID, &user.Username, &user.PasswordHash); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return user, ErrNotFound
		}
		return user, ErrInternalServer
	}
	return user, nil
}

func (r *Repository) GetByUsername(ctx context.Context, username string) (User, error) {
	var user User
	row := r.db.QueryRow(ctx, `SELECT id, username, password_hash FROM users WHERE username = $1`, username)
	if err := row.Scan(&user.ID, &user.Username, &user.PasswordHash); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return user, ErrNotFound
		}
		return user, ErrInternalServer
	}
	return user, nil
}

func (r *Repository) Register(ctx context.Context, user User) error {
	_, err := r.db.Exec(ctx, `INSERT INTO users
		(id, username, password_hash) VALUES ($1, $2, $3)`,
		user.ID, user.Username, user.PasswordHash)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == PgErrUniqueConstraintViolationCode {
				return ErrExistingUsername
			}
		}
		return ErrInternalServer
	}
	return nil
}

func (r *Repository) GetActiveKeyByHash(ctx context.Context, keyHash string) (Key, error) {
	var k Key
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, role, created_at, expires_at
		FROM keys
		WHERE key_hash = $1
		  AND (expires_at IS NULL OR expires_at > now())`, keyHash,
	).Scan(&k.ID, &k.UserID, &k.Role, &k.ExpiresAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return Key{}, ErrKeyNotFound
	}
	if err != nil {
		return Key{}, ErrInternalServer
	}
	return k, nil
}

func (r *Repository) SaveKey(ctx context.Context, key Key) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return ErrInternalServer
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`INSERT INTO keys (id, user_id, key_hash, role, created_at, expires_at)
			VALUES ($1, $2, $3, $4, $5, $6)`,
		key.ID, key.UserID, key.KeyHash, key.Role, key.CreatedAt, key.ExpiresAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == PgErrForeignKeyViolationCode {
				return ErrUserNotFound
			} else if pgErr.Code == PgErrUniqueConstraintViolationCode {
				return ErrInvalidID
			}
			return ErrInternalServer
		}
	}

	for _, eventType := range key.EventTypes {
		_, err = tx.Exec(ctx,
			`INSERT INTO key_permissions (key_id, event_type) values ($1, $2)`,
			key.ID, eventType)
		if err != nil {
			return ErrInternalServer
		}
	}
	return tx.Commit(ctx)
}

func (r *Repository) RemoveKey(ctx context.Context, key Key) error {
	pgResult, err := r.db.Exec(ctx, "DELETE FROM keys WHERE id = $1", key.ID)
	if err != nil {
		return ErrInternalServer
	}
	if pgResult.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
