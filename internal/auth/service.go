package auth

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"golang.org/x/crypto/bcrypt"
)

var (
	tracer         = otel.Tracer("auth")
	meter          = otel.Meter("auth")
	loginAttempts  metric.Int64Counter
	keyGenerations metric.Int64Counter
)

func init() {
	var err error
	loginAttempts, err = meter.Int64Counter("auth.login_attempts",
		metric.WithDescription("Number of login attempts"))
	if err != nil {
		panic(err)
	}
	keyGenerations, err = meter.Int64Counter("auth.key_generations",
		metric.WithDescription("Number of key generations"))
	if err != nil {
		panic(err)
	}
}

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
	ctx, span := tracer.Start(ctx, "auth.Login")
	defer span.End()

	existingUser, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			loginAttempts.Add(ctx, 1, metric.WithAttributes(attribute.String("result", "invalid_credentials")))
			return "", ErrInvalidCredentials
		}
		span.RecordError(err)
		loginAttempts.Add(ctx, 1, metric.WithAttributes(attribute.String("result", "internal_error")))
		return "", ErrInternalServer
	}

	err = bcrypt.CompareHashAndPassword([]byte(existingUser.PasswordHash), []byte(password))
	if err != nil {
		loginAttempts.Add(ctx, 1, metric.WithAttributes(attribute.String("result", "invalid_credentials")))
		return "", ErrInvalidCredentials
	}
	token, err := s.generateJWT(existingUser.ID)
	if err != nil {
		loginAttempts.Add(ctx, 1, metric.WithAttributes(attribute.String("result", "internal_error")))
		span.RecordError(err)
		return "", ErrInternalServer
	}

	loginAttempts.Add(ctx, 1, metric.WithAttributes(attribute.String("result", "success")))
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
	ctx, span := tracer.Start(ctx, "auth.GenerateKey")
	defer span.End()

	span.SetAttributes(attribute.String("user_id", userID))

	k, err := NewKey(userID, role, eventTypes, time.Now().Add(7*time.Hour*24))
	if err != nil {
		if errors.Is(err, ErrRoleNotFound) {
			keyGenerations.Add(ctx, 1, metric.WithAttributes(attribute.String("result", "invalid_role")))
			span.RecordError(err)
			return GeneratedKey{}, ErrRoleNotFound
		}
		keyGenerations.Add(ctx, 1, metric.WithAttributes(attribute.String("result", "internal_error")))
		span.RecordError(err)
		return GeneratedKey{}, ErrInternalServer
	}

	err = s.repo.SaveKey(ctx, k.Key)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			keyGenerations.Add(ctx, 1, metric.WithAttributes(attribute.String("result", "user_not_found")))
			span.RecordError(err)
			return GeneratedKey{}, ErrUserNotFound
		}
		keyGenerations.Add(ctx, 1, metric.WithAttributes(attribute.String("result", "internal_error")))
		span.RecordError(err)
		return GeneratedKey{}, ErrInternalServer
	}

	keyGenerations.Add(ctx, 1, metric.WithAttributes(attribute.String("result", "success")))
	return k, nil
}

func (s *Service) RemoveKey(ctx context.Context, userID string, keyID string) error {
	ctx, span := tracer.Start(ctx, "auth.RemoveKey")
	defer span.End()

	err := s.repo.RemoveKey(ctx, keyID, userID)
	if err != nil {
		if errors.Is(err, ErrKeyNotFound) {
			span.RecordError(err)
			return ErrKeyNotFound
		}
		span.RecordError(err)
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
