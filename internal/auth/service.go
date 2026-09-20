package auth

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo      *Repository
	jwtSecret []byte
}

func NewService(repo *Repository, jwtSecretString string) *Service {
	jwtSecret := []byte(jwtSecretString)
	return &Service{repo: repo, jwtSecret: jwtSecret}
}

func (s *Service) Register(ctx context.Context, username, password string) (User, error) {
	user, err := NewUser(username, password)
	if err != nil {
		return User{}, err
	}
	err = s.repo.Register(ctx, user)
	if err != nil {
		return User{}, err
	}
	return user, nil
}

func (s *Service) Login(ctx context.Context, username, password string) (string, error) {
	existingUser, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return "", ErrInvalidCredentials
		}
		return "", ErrInternalServer
	}

	err = bcrypt.CompareHashAndPassword([]byte(existingUser.PasswordHash), []byte(password))
	if err != nil {
		return "", ErrInvalidCredentials
	}
	token, err := s.generateJWT(existingUser.ID)
	if err != nil {
		return "", ErrInternalServer
	}

	return token, nil
}

func (s *Service) GetKeys(ctx context.Context, userID string) ([]Key, error) {
	k, err := s.repo.GetKeysList(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrKeyNotFound) {
			return nil, ErrKeyNotFound
		}
		return nil, ErrInternalServer
	}
	return k, nil
}

func (s *Service) GenerateKey(ctx context.Context, userID string, role KeyRole, eventTypes []string) (GeneratedKey, error) {
	k, err := NewKey(userID, role, eventTypes, time.Now().Add(7*time.Hour*24))
	if err != nil {
		if errors.Is(err, ErrRoleNotFound) {
			return GeneratedKey{}, ErrRoleNotFound
		}
		return GeneratedKey{}, ErrInternalServer
	}

	err = s.repo.SaveKey(ctx, k.Key)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return GeneratedKey{}, ErrUserNotFound
		}
		return GeneratedKey{}, ErrInternalServer
	}

	return k, nil
}

func (s *Service) RemoveKey(ctx context.Context, userID string, keyID string) error {
	err := s.repo.RemoveKey(ctx, keyID, userID)
	if err != nil {
		if errors.Is(err, ErrKeyNotFound) {
			return ErrKeyNotFound
		}
		return ErrInternalServer
	}
	return nil
}

func (s *Service) ValidateKey(ctx context.Context, keyString string) (Key, error) {
	hash := HashKey(keyString)
	return s.repo.GetActiveKeyByHash(ctx, hash)
}

func (s *Service) ParseJWT(tokenString string) (string, error) {
	var claims jwt.RegisteredClaims
	_, err := jwt.ParseWithClaims(tokenString, &claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.jwtSecret, nil
	},
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return "", ErrInvalidToken
	}

	userID := claims.Subject
	return userID, nil
}

func (s *Service) generateJWT(userID string) (string, error) {
	if len(s.jwtSecret) == 0 {
		return "", ErrInternalServer
	}
	t := jwt.New(jwt.SigningMethodHS256)
	t.Claims = &jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(60 * time.Minute)),
		Issuer:    "relay-hook",
		ID:        uuid.New().String(),
	}
	return t.SignedString(s.jwtSecret)
}
