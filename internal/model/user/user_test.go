package user

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func Test_ParseUserRole(t *testing.T) {
	cases := []struct {
		name       string
		role       string
		expectRole UserRole
		expectErr  error
	}{
		{
			name:       "seller uppercase",
			role:       "SELLER",
			expectRole: Seller,
			expectErr:  nil,
		},
		{
			name:       "buyer uppercase",
			role:       "BUYER",
			expectRole: Buyer,
			expectErr:  nil,
		},
		{
			name:       "warehouse uppercase",
			role:       "WAREHOUSE",
			expectRole: Warehouse,
			expectErr:  nil,
		},
		{
			name:       "logistics uppercase",
			role:       "LOGISTICS",
			expectRole: Logistics,
			expectErr:  nil,
		},
		{
			name:       "seller lowercase",
			role:       "seller",
			expectRole: Seller,
			expectErr:  nil,
		},
		{
			name:       "buyer mixed case",
			role:       "Buyer",
			expectRole: Buyer,
			expectErr:  nil,
		},
		{
			name:       "invalid role",
			role:       "ISJDKK",
			expectRole: "",
			expectErr:  ErrInvalidUserRole,
		},
		{
			name:       "empty role",
			role:       "",
			expectRole: "",
			expectErr:  ErrInvalidUserRole,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			role, err := ParseUserRole(tc.role)
			if tc.expectErr != nil {
				require.ErrorIs(t, err, tc.expectErr)
				require.Equal(t, UserRole(""), role)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectRole, role)
			}
		})
	}
}

func Test_NewUser(t *testing.T) {
	type userInput struct {
		email        string
		passwordHash string
		role         string
	}

	cases := []struct {
		name      string
		user      userInput
		expectErr error
	}{
		{
			name: "success seller",
			user: userInput{
				email:        "seller@test.com",
				passwordHash: "hashed_pw_1",
				role:         "SELLER",
			},
			expectErr: nil,
		},
		{
			name: "success buyer",
			user: userInput{
				email:        "buyer@test.com",
				passwordHash: "hashed_pw_2",
				role:         "BUYER",
			},
			expectErr: nil,
		},
		{
			name: "success warehouse",
			user: userInput{
				email:        "warehouse@test.com",
				passwordHash: "hashed_pw_3",
				role:         "WAREHOUSE",
			},
			expectErr: nil,
		},
		{
			name: "success logistics",
			user: userInput{
				email:        "logistics@test.com",
				passwordHash: "hashed_pw_4",
				role:         "LOGISTICS",
			},
			expectErr: nil,
		},
		{
			name: "success role case insensitive",
			user: userInput{
				email:        "lower@test.com",
				passwordHash: "hashed_pw_5",
				role:         "seller",
			},
			expectErr: nil,
		},
		{
			name: "invalid role",
			user: userInput{
				email:        "admin@test.com",
				passwordHash: "hashed_pw_6",
				role:         "KFJKDJF",
			},
			expectErr: ErrInvalidUserRole,
		},
		{
			name: "empty role",
			user: userInput{
				email:        "empty@test.com",
				passwordHash: "hashed_pw_7",
				role:         "",
			},
			expectErr: ErrInvalidUserRole,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			u, err := NewUser(tc.user.email, tc.user.passwordHash, tc.user.role)
			if tc.expectErr != nil {
				require.ErrorIs(t, err, tc.expectErr)
				require.Nil(t, u)
			} else {
				require.NoError(t, err)
				require.NotNil(t, u)
				require.NotEqual(t, uuid.Nil, u.ID)
				require.Equal(t, tc.user.email, u.Email)
				require.Equal(t, tc.user.passwordHash, u.PasswordHash)
				require.Equal(t, UserRole(strings.ToUpper(tc.user.role)), u.Role)
				require.False(t, u.CreatedAt.IsZero())
			}
		})
	}
}
