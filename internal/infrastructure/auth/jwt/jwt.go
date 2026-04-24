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

type Claims struct {
	UserID    string `json:"user_id"`
	UserRole  string `json:"user_role"`
	CompanyID string `json:"company_id"`
	jwt.RegisteredClaims
}

type JWTGenerator struct {
	jwtSecret      []byte
	accessTokenTTL time.Duration
}

func NewJWTGenerator(jwtSecret string, accessTokenTTL time.Duration) *JWTGenerator {
	return &JWTGenerator{
		jwtSecret:      []byte(jwtSecret),
		accessTokenTTL: accessTokenTTL,
	}
}

func (t *JWTGenerator) Generate(userID, userRole, companyID string) (string, error) {
	const op = "infrastructure.auth.jwt.Generate"

	now := time.Now()
	claims := Claims{
		UserID:    userID,
		UserRole:  userRole,
		CompanyID: companyID,
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

func (t *JWTGenerator) Validate(tokenString string) (userID, userRole, companyID string, err error) {
	const op = "infrastructure.auth.jwt.ValidateToken"

	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%s: %w", op, ErrInvalidToken)
		}
		return t.jwtSecret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return "", "", "", fmt.Errorf("%s: %w", op, ErrExpiredToken)
		}
		return "", "", "", fmt.Errorf("%s: parse token: %w", op, err)
	}

	if !token.Valid {
		return "", "", "", fmt.Errorf("%s: %w", op, ErrInvalidToken)
	}

	return claims.UserID, claims.UserRole, claims.CompanyID, nil
}
