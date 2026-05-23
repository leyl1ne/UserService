package company

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	companymodel "github.com/leyl1ne/UserService/internal/model/company"
	"github.com/leyl1ne/UserService/internal/service"
	"github.com/leyl1ne/UserService/internal/service/company/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_Service_CreateCompany(t *testing.T) {
	ctx := context.Background()
	desc := "A test company"

	type createInput struct {
		name        string
		description *string
	}

	type repoReturns struct {
		createErr error
	}

	cases := []struct {
		name        string
		input       createInput
		repoReturns repoReturns
		expectErr   error
	}{
		{
			name: "success with description",
			input: createInput{
				name:        "Acme Corp",
				description: &desc,
			},
			repoReturns: repoReturns{
				createErr: nil,
			},
			expectErr: nil,
		},
		{
			name: "success without description",
			input: createInput{
				name:        "Acme Corp",
				description: nil,
			},
			repoReturns: repoReturns{
				createErr: nil,
			},
			expectErr: nil,
		},
		{
			name: "empty name — validation error",
			input: createInput{
				name:        "",
				description: &desc,
			},
			repoReturns: repoReturns{},
			expectErr:   service.NewValidationError("name", "field is required"),
		},
		{
			name: "repository error on create",
			input: createInput{
				name:        "Acme Corp",
				description: nil,
			},
			repoReturns: repoReturns{
				createErr: errors.New("db connection lost"),
			},
			expectErr: nil, // wrapped, not a sentinel
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := mocks.NewRepository(t)

			if tc.input.name != "" {
				mockRepo.
					On("CreateCompany", mock.Anything, mock.AnythingOfType("*company.Company")).
					Return(tc.repoReturns.createErr).
					Once()
			}

			svc := NewService(mockRepo)

			result, err := svc.CreateCompany(ctx, CreateCompanyInput{
				Name:        tc.input.name,
				Description: tc.input.description,
			})

			switch {
			case tc.input.name == "":
				require.Error(t, err)
				require.Nil(t, result)
				var valErr service.ValidationError
				require.True(t, errors.As(err, &valErr), "expected ValidationError")
				require.Equal(t, map[string]string{
					"name": "field is required",
				}, valErr.Fields)
			case tc.repoReturns.createErr != nil:
				require.Error(t, err)
				require.Nil(t, result)
			default:
				require.NoError(t, err)
				require.NotNil(t, result)
				require.Equal(t, tc.input.name, result.Name)
				require.Equal(t, tc.input.description, result.Description)
				require.NotEqual(t, uuid.Nil, result.ID)
				require.False(t, result.CreatedAt.IsZero())
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func Test_Service_GetCompanyByID(t *testing.T) {
	ctx := context.Background()
	companyID := uuid.New()
	now := time.Now()

	type getCompanyInput struct {
		companyID uuid.UUID
	}

	type repoReturns struct {
		company *companymodel.Company
		error   error
	}

	cases := []struct {
		name        string
		input       getCompanyInput
		repoReturns repoReturns
		expectErr   error
	}{
		{
			name: "success",
			input: getCompanyInput{
				companyID: companyID,
			},
			repoReturns: repoReturns{
				company: &companymodel.Company{
					ID:          companyID,
					Name:        "Test Corp",
					Description: nil,
					CreatedAt:   now,
				},
				error: nil,
			},
			expectErr: nil,
		},
		{
			name: "success with description",
			input: getCompanyInput{
				companyID: companyID,
			},
			repoReturns: repoReturns{
				company: func() *companymodel.Company {
					d := "Some description"
					return &companymodel.Company{
						ID:          companyID,
						Name:        "Test Corp",
						Description: &d,
						CreatedAt:   now,
					}
				}(),
				error: nil,
			},
			expectErr: nil,
		},
		{
			name: "company not found",
			input: getCompanyInput{
				companyID: uuid.New(),
			},
			repoReturns: repoReturns{
				company: nil,
				error:   companymodel.ErrCompanyNotFound,
			},
			expectErr: service.ErrCompanyNotFound,
		},
		{
			name: "repository unexpected error",
			input: getCompanyInput{
				companyID: uuid.New(),
			},
			repoReturns: repoReturns{
				company: nil,
				error:   errors.New("db connection lost"),
			},
			expectErr: nil, // wrapped, not a sentinel
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := mocks.NewRepository(t)
			mockRepo.
				On("GetCompanyByID", mock.Anything, tc.input.companyID).
				Return(tc.repoReturns.company, tc.repoReturns.error).
				Once()

			svc := NewService(mockRepo)

			result, err := svc.GetCompanyByID(ctx, tc.input.companyID)
			if tc.expectErr != nil {
				require.ErrorIs(t, err, tc.expectErr)
				require.Nil(t, result)
			} else if tc.repoReturns.error != nil {
				require.Error(t, err)
				require.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				require.Equal(t, tc.repoReturns.company.ID, result.ID)
				require.Equal(t, tc.repoReturns.company.Name, result.Name)
				require.Equal(t, tc.repoReturns.company.Description, result.Description)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}
