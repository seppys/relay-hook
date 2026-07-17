package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrKeyNotFound        = errors.New("key not found")
	ErrRoleNotFound       = errors.New("role not found")
	ErrInvalidID          = errors.New("invalid or missing id")
	ErrInvalidPayload     = errors.New("invalid or missing payload")
	ErrExistingUsername   = errors.New("username already exists")
	ErrInternalServer     = errors.New("server error")
	ErrNotFound           = errors.New("not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrInvalidToken       = errors.New("invalid token")
)

type KeyRole string

const (
	KeyRoleEmitter    KeyRole = "emitter"
	KeyRoleSubscriber KeyRole = "subscriber"
)

func (k KeyRole) Validate() bool {
	switch k {
	case KeyRoleEmitter, KeyRoleSubscriber:
		return true
	default:
		return false
	}
}

type User struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"`
}

type Key struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	KeyHash    string    `json:"-"`
	Role       KeyRole   `json:"role"`
	EventTypes []string  `json:"event_types"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

func (k Key) Permits(eventType string) bool {
	return len(k.EventTypes) == 0 || slices.Contains(k.EventTypes, eventType)
}

type GeneratedKey struct {
	Key          Key    `json:"key"`
	PlaintextKey string `json:"plaintext_key"`
}

func NewUser(username string, password string) (User, error) {
	id := uuid.New().String()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, err
	}
	return User{ID: id, Username: username, PasswordHash: string(hash)}, nil
}

func NewKey(userID string, role KeyRole, eventTypes []string, expiresAt time.Time) (GeneratedKey, error) {
	plainKey, err := newPlaintextKey(role)
	if err != nil {
		return GeneratedKey{}, err
	}
	key := Key{
		ID:         uuid.New().String(),
		UserID:     userID,
		KeyHash:    HashKey(plainKey),
		Role:       role,
		EventTypes: eventTypes,
		CreatedAt:  time.Now(),
		ExpiresAt:  expiresAt,
	}
	return GeneratedKey{Key: key, PlaintextKey: plainKey}, nil
}

func newPlaintextKey(role KeyRole) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating key: %w", err)
	}

	var prefix string
	switch role {
	case KeyRoleEmitter:
		prefix = "rh_emit"
	case KeyRoleSubscriber:
		prefix = "rh_sub"
	default:
		return "", ErrRoleNotFound
	}

	return prefix + "_" + base64.RawURLEncoding.EncodeToString(b), nil
}

func HashKey(plainKey string) string {
	bytes := sha256.Sum256([]byte(plainKey))
	return hex.EncodeToString(bytes[:])
}
