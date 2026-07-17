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

func (r *Repository) GetKeysList(ctx context.Context, userID string) ([]Key, error) {
	var keys []Key
	rows, err := r.db.Query(ctx, `
		SELECT k.id, k.user_id, k.role, k.created_at, k.expires_at,
			   COALESCE(array_agg(p.event_type) FILTER (WHERE p.event_type IS NOT NULL), '{}') AS event_types
		FROM keys k
		LEFT JOIN key_permissions p ON p.key_id = k.id
		WHERE k.user_id = $1
		GROUP BY k.id, k.user_id, k.role, k.created_at, k.expires_at`,
		userID)
	if err != nil {
		return nil, ErrInternalServer
	}

	defer rows.Close()
	for rows.Next() {
		var key Key
		if err := rows.Scan(&key.ID, &key.UserID, &key.Role, &key.CreatedAt, &key.ExpiresAt, &key.EventTypes); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, nil
}

func (r *Repository) GetActiveKeyByID(ctx context.Context, keyID string) (Key, error) {
	var k Key
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, key_hash, role, created_at, expires_at 
		FROM keys 
		WHERE id = $1`,
		keyID,
	).Scan(&k.ID, &k.UserID, &k.Role, &k.CreatedAt, &k.ExpiresAt, &k.EventTypes)

	if errors.Is(err, pgx.ErrNoRows) {
		return Key{}, ErrKeyNotFound
	}
	if err != nil {
		return Key{}, ErrInternalServer
	}
	return k, nil
}

func (r *Repository) GetActiveKeyByHash(ctx context.Context, keyHash string) (Key, error) {
	var k Key
	err := r.db.QueryRow(ctx, `
		SELECT k.id, k.user_id, k.role, k.created_at, k.expires_at,
		       COALESCE(array_agg(p.event_type) FILTER (WHERE p.event_type IS NOT NULL), '{}') AS event_types
		FROM keys k
		LEFT JOIN key_permissions p ON p.key_id = k.id
		WHERE k.key_hash = $1
		  AND (k.expires_at > now())
		GROUP BY k.id, k.user_id, k.role, k.created_at, k.expires_at`, keyHash,
	).Scan(&k.ID, &k.UserID, &k.Role, &k.CreatedAt, &k.ExpiresAt, &k.EventTypes)

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

func (r *Repository) RemoveKey(ctx context.Context, keyID, userID string) error {
	pgResult, err := r.db.Exec(ctx, "DELETE FROM keys WHERE id = $1 AND user_id = $2", keyID, userID)
	if err != nil {
		return ErrInternalServer
	}
	if pgResult.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) RemoveExpiredKeys(ctx context.Context) error {
	_, err := r.db.Exec(ctx, "DELETE FROM keys WHERE expires_at < now()")
	if err != nil {
		return ErrInternalServer
	}
	return nil
}
