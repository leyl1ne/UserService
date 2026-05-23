package jwt

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func Test_JWTGenerator_GenerateAccessToken(t *testing.T) {
	cfg := Config{
		Secret:        "test-secret-key-for-testing",
		AccessTokeTTL: time.Hour,
	}
	gen := NewJWTGenerator(cfg)

	type generateInput struct {
		payload Payload
	}

	cases := []struct {
		name      string
		input     generateInput
		expectErr error
	}{
		{
			name: "success",
			input: generateInput{
				payload: Payload{
					UserID:    "user-123",
					UserRole:  "SELLER",
					CompanyID: "company-456",
				},
			},
			expectErr: nil,
		},
		{
			name: "success with empty company id",
			input: generateInput{
				payload: Payload{
					UserID:    "user-789",
					UserRole:  "BUYER",
					CompanyID: "",
				},
			},
			expectErr: nil,
		},
		{
			name: "success with empty fields",
			input: generateInput{
				payload: Payload{
					UserID:    "",
					UserRole:  "",
					CompanyID: "",
				},
			},
			expectErr: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			token, err := gen.GenerateAccessToken(tc.input.payload)
			if tc.expectErr != nil {
				require.ErrorIs(t, err, tc.expectErr)
				require.Empty(t, token)
			} else {
				require.NoError(t, err)
				require.NotEmpty(t, token)
			}
		})
	}
}

func Test_JWTGenerator_Validate(t *testing.T) {
	secret := "test-secret-key-for-testing"
	ttl := time.Hour

	cfg := Config{
		Secret:        secret,
		AccessTokeTTL: ttl,
	}
	gen := NewJWTGenerator(cfg)

	type validateInput struct {
		payload     Payload
		tokenString string
	}

	cases := []struct {
		name         string
		setup        func() validateInput
		expectErr    error
		checkPayload func(t *testing.T, got Payload)
	}{
		{
			name: "success",
			setup: func() validateInput {
				payload := Payload{
					UserID:    "user-123",
					UserRole:  "SELLER",
					CompanyID: "company-456",
				}
				token, err := gen.GenerateAccessToken(payload)
				require.NoError(t, err)
				return validateInput{
					payload:     payload,
					tokenString: token,
				}
			},
			expectErr: nil,
			checkPayload: func(t *testing.T, got Payload) {
				require.Equal(t, "user-123", got.UserID)
				require.Equal(t, "SELLER", got.UserRole)
				require.Equal(t, "company-456", got.CompanyID)
			},
		},
		{
			name: "success with empty company id",
			setup: func() validateInput {
				payload := Payload{
					UserID:    "user-789",
					UserRole:  "BUYER",
					CompanyID: "",
				}
				token, err := gen.GenerateAccessToken(payload)
				require.NoError(t, err)
				return validateInput{
					payload:     payload,
					tokenString: token,
				}
			},
			expectErr: nil,
			checkPayload: func(t *testing.T, got Payload) {
				require.Equal(t, "user-789", got.UserID)
				require.Equal(t, "BUYER", got.UserRole)
				require.Equal(t, "", got.CompanyID)
			},
		},
		{
			name: "expired token",
			setup: func() validateInput {
				expiredCfg := Config{
					Secret:        secret,
					AccessTokeTTL: -time.Second,
				}
				expiredGen := NewJWTGenerator(expiredCfg)
				payload := Payload{
					UserID:    "user-expired",
					UserRole:  "WAREHOUSE",
					CompanyID: "company-expired",
				}
				token, err := expiredGen.GenerateAccessToken(payload)
				require.NoError(t, err)
				return validateInput{
					payload:     payload,
					tokenString: token,
				}
			},
			expectErr:    ErrExpiredToken,
			checkPayload: nil,
		},
		{
			name: "invalid token string",
			setup: func() validateInput {
				return validateInput{
					tokenString: "this.is.not.a.valid.token",
				}
			},
			expectErr:    jwt.ErrTokenMalformed,
			checkPayload: nil,
		},
		{
			name: "token signed with different secret",
			setup: func() validateInput {
				wrongCfg := Config{
					Secret:        "wrong-secret-key",
					AccessTokeTTL: ttl,
				}
				wrongGen := NewJWTGenerator(wrongCfg)
				payload := Payload{
					UserID:    "user-wrong",
					UserRole:  "LOGISTICS",
					CompanyID: "company-wrong",
				}
				token, err := wrongGen.GenerateAccessToken(payload)
				require.NoError(t, err)
				return validateInput{
					tokenString: token,
				}
			},
			expectErr:    jwt.ErrSignatureInvalid,
			checkPayload: nil,
		},
		{
			name: "empty token string",
			setup: func() validateInput {
				return validateInput{
					tokenString: "",
				}
			},
			expectErr:    jwt.ErrTokenMalformed,
			checkPayload: nil,
		},
		{
			name: "tampered token",
			setup: func() validateInput {
				payload := Payload{
					UserID:    "user-tamper",
					UserRole:  "SELLER",
					CompanyID: "company-tamper",
				}
				token, err := gen.GenerateAccessToken(payload)
				require.NoError(t, err)
				tampered := token[:len(token)-5] + "XXXXX"
				return validateInput{
					tokenString: tampered,
				}
			},
			expectErr:    jwt.ErrSignatureInvalid,
			checkPayload: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := tc.setup()
			payload, err := gen.Validate(input.tokenString)
			if tc.expectErr != nil {
				require.ErrorIs(t, err, tc.expectErr)
			} else {
				require.NoError(t, err)
				if tc.checkPayload != nil {
					tc.checkPayload(t, payload)
				}
			}
		})
	}
}
