package password

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidPassword = errors.New("invalid password")

type Hasher struct {
	cost int
}

func NewHasher(cfg Config) (*Hasher, error) {
	const op = "infrastructure.auth.password.NewHasher"

	if cfg.Cost < bcrypt.MinCost || cfg.Cost > bcrypt.MaxCost {
		return nil, fmt.Errorf("%s: invalid bcrypt cost: %d", op, cfg.Cost)
	}

	return &Hasher{
		cost: cfg.Cost,
	}, nil
}

func (h *Hasher) Hash(password string) (string, error) {
	const op = "infrastructure.auth.password.Hash"

	if password == "" {
		return "", fmt.Errorf("%s: %w", op, ErrInvalidPassword)
	}

	hashBytes, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("%s: generate hash: %w", op, err)
	}

	return string(hashBytes), nil
}

func (h *Hasher) Compare(passwordHash, password string) error {
	const op = "infrastructure.auth.password.compare"

	if password == "" {
		return fmt.Errorf("%s: %w", op, ErrInvalidPassword)
	}

	err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return fmt.Errorf("%s: %w", op, ErrInvalidPassword)
		}

		return fmt.Errorf("%s: compare password: %w", op, err)
	}

	return nil
}
