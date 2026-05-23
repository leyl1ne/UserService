package password

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func Test_NewHasher(t *testing.T) {
	type hasherInput struct {
		cost int
	}

	cases := []struct {
		name      string
		input     hasherInput
		expectErr error
	}{
		{
			name: "success with min cost",
			input: hasherInput{
				cost: bcrypt.MinCost,
			},
			expectErr: nil,
		},
		{
			name: "success with default cost",
			input: hasherInput{
				cost: bcrypt.DefaultCost,
			},
			expectErr: nil,
		},
		{
			name: "success with max cost",
			input: hasherInput{
				cost: bcrypt.MaxCost,
			},
			expectErr: nil,
		},
		{
			name: "cost below min",
			input: hasherInput{
				cost: bcrypt.MinCost - 1,
			},
			expectErr: nil, // returns fmt.Errorf, not a sentinel
		},
		{
			name: "cost above max",
			input: hasherInput{
				cost: bcrypt.MaxCost + 1,
			},
			expectErr: nil, // returns fmt.Errorf, not a sentinel
		},
		{
			name: "zero cost",
			input: hasherInput{
				cost: 0,
			},
			expectErr: nil, // returns fmt.Errorf, not a sentinel
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h, err := NewHasher(Config{Cost: tc.input.cost})
			if tc.input.cost < bcrypt.MinCost || tc.input.cost > bcrypt.MaxCost {
				require.Error(t, err)
				require.Nil(t, h)
			} else {
				require.NoError(t, err)
				require.NotNil(t, h)
			}
		})
	}
}

func Test_Hasher_Hash(t *testing.T) {
	hasher, err := NewHasher(Config{Cost: bcrypt.MinCost})
	require.NoError(t, err)

	type hashInput struct {
		password string
	}

	cases := []struct {
		name      string
		input     hashInput
		expectErr error
	}{
		{
			name: "success",
			input: hashInput{
				password: "my_secure_password",
			},
			expectErr: nil,
		},
		{
			name: "success with simple password",
			input: hashInput{
				password: "abc123",
			},
			expectErr: nil,
		},
		{
			name: "empty password",
			input: hashInput{
				password: "",
			},
			expectErr: ErrInvalidPassword,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hash, err := hasher.Hash(tc.input.password)
			if tc.expectErr != nil {
				require.ErrorIs(t, err, tc.expectErr)
				require.Empty(t, hash)
			} else {
				require.NoError(t, err)
				require.NotEmpty(t, hash)
				require.NotEqual(t, tc.input.password, hash)
			}
		})
	}
}

func Test_Hasher_Compare(t *testing.T) {
	hasher, err := NewHasher(Config{Cost: bcrypt.MinCost})
	require.NoError(t, err)

	validPassword := "my_secure_password"
	validHash, err := hasher.Hash(validPassword)
	require.NoError(t, err)

	type compareInput struct {
		passwordHash string
		password     string
	}

	cases := []struct {
		name      string
		input     compareInput
		expectErr error
	}{
		{
			name: "success matching password",
			input: compareInput{
				passwordHash: validHash,
				password:     validPassword,
			},
			expectErr: nil,
		},
		{
			name: "mismatched password",
			input: compareInput{
				passwordHash: validHash,
				password:     "wrong_password",
			},
			expectErr: ErrInvalidPassword,
		},
		{
			name: "empty password",
			input: compareInput{
				passwordHash: validHash,
				password:     "",
			},
			expectErr: ErrInvalidPassword,
		},
		{
			name: "invalid hash with valid password",
			input: compareInput{
				passwordHash: "not-a-bcrypt-hash",
				password:     validPassword,
			},
			expectErr: bcrypt.ErrHashTooShort,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := hasher.Compare(tc.input.passwordHash, tc.input.password)
			if tc.expectErr != nil {
				require.ErrorIs(t, err, tc.expectErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
