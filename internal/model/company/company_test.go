package company

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func Test_NewCompany(t *testing.T) {
	desc := "Test company description"

	type companyInput struct {
		name        string
		description *string
	}

	cases := []struct {
		name      string
		company   companyInput
		expectErr error
	}{
		{
			name: "success with description",
			company: companyInput{
				name:        "Acme Corp",
				description: &desc,
			},
			expectErr: nil,
		},
		{
			name: "success without description",
			company: companyInput{
				name:        "Acme Corp",
				description: nil,
			},
			expectErr: nil,
		},
		{
			name: "empty name",
			company: companyInput{
				name:        "",
				description: &desc,
			},
			expectErr: ErrCompanyNameRequired,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, err := NewCompany(tc.company.name, tc.company.description)
			if tc.expectErr != nil {
				require.ErrorIs(t, err, tc.expectErr)
				require.Nil(t, c)
			} else {
				require.NoError(t, err)
				require.NotNil(t, c)
				require.NotEqual(t, uuid.Nil, c.ID)
				require.Equal(t, tc.company.name, c.Name)
				require.Equal(t, tc.company.description, c.Description)
				require.False(t, c.CreatedAt.IsZero())
			}
		})
	}
}
