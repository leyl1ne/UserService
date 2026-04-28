package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
)

type Payload struct {
	UserID    string `json:"user_id"`
	UserRole  string `json:"user_role"`
	CompanyID string `json:"company_id"`
}

type Claims struct {
	Payload
	jwt.RegisteredClaims
}

type JWTGenerator struct {
	jwtSecret      []byte
	accessTokenTTL time.Duration
}

func NewJWTGenerator(cfg Config) *JWTGenerator {
	return &JWTGenerator{
		jwtSecret:      []byte(cfg.Secret),
		accessTokenTTL: cfg.AccessTokeTTL,
	}
}

func (t *JWTGenerator) GenerateAccessToken(payload Payload) (string, error) {
	const op = "infrastructure.auth.jwt.Generate"

	now := time.Now()
	claims := Claims{
		Payload: Payload{
			UserID:    payload.UserID,
			UserRole:  payload.UserRole,
			CompanyID: payload.CompanyID,
		},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(t.accessTokenTTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(t.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("%s: signed string: %w", op, err)
	}
	return tokenString, nil
}

func (t *JWTGenerator) Validate(tokenString string) (Payload, error) {
	const op = "infrastructure.auth.jwt.Validate"

	claims := &Claims{}
	var payload Payload

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%s: %w", op, ErrInvalidToken)
		}
		return t.jwtSecret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return payload, fmt.Errorf("%s: %w", op, ErrExpiredToken)
		}
		return payload, fmt.Errorf("%s: parse token: %w", op, err)
	}

	if !token.Valid {
		return payload, fmt.Errorf("%s: %w", op, ErrInvalidToken)
	}

	return Payload{
		UserID:    claims.UserID,
		UserRole:  claims.UserRole,
		CompanyID: claims.CompanyID,
	}, nil
}
