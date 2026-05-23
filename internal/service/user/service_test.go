package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	companymodel "github.com/leyl1ne/UserService/internal/model/company"
	usermodel "github.com/leyl1ne/UserService/internal/model/user"
	"github.com/leyl1ne/UserService/internal/service"
	"github.com/leyl1ne/UserService/internal/service/user/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_Service_GetUserByID(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	companyID := uuid.New()
	now := time.Now()

	type getUserInput struct {
		userID uuid.UUID
	}

	type repoReturns struct {
		user  *usermodel.User
		error error
	}

	cases := []struct {
		name        string
		input       getUserInput
		repoReturns repoReturns
		expectErr   error
	}{
		{
			name: "success",
			input: getUserInput{
				userID: userID,
			},
			repoReturns: repoReturns{
				user: &usermodel.User{
					ID:        userID,
					Email:     "test@mail.com",
					Role:      usermodel.Seller,
					CompanyID: nil,
					CreatedAt: now,
				},
				error: nil,
			},
			expectErr: nil,
		},
		{
			name: "success with company id",
			input: getUserInput{
				userID: userID,
			},
			repoReturns: repoReturns{
				user: &usermodel.User{
					ID:        userID,
					Email:     "test@mail.com",
					Role:      usermodel.Seller,
					CompanyID: &companyID,
					CreatedAt: now,
				},
				error: nil,
			},
			expectErr: nil,
		},
		{
			name: "user not found",
			input: getUserInput{
				userID: uuid.New(),
			},
			repoReturns: repoReturns{
				user:  nil,
				error: usermodel.ErrUserNotFound,
			},
			expectErr: service.ErrUserNotFound,
		},
		{
			name: "repository unexpected error",
			input: getUserInput{
				userID: uuid.New(),
			},
			repoReturns: repoReturns{
				user:  nil,
				error: errors.New("db connection lost"),
			},
			expectErr: nil, // wrapped, not a sentinel
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := mocks.NewRepository(t)
			mockRepo.
				On("GetUserByID", mock.Anything, tc.input.userID).
				Return(tc.repoReturns.user, tc.repoReturns.error).
				Once()

			svc := NewService(mockRepo)

			result, err := svc.GetUserByID(ctx, tc.input.userID)
			if tc.expectErr != nil {
				require.ErrorIs(t, err, tc.expectErr)
				require.Nil(t, result)
			} else if tc.repoReturns.error != nil {
				require.Error(t, err)
				require.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				require.Equal(t, tc.repoReturns.user.ID, result.ID)
				require.Equal(t, tc.repoReturns.user.Email, result.Email)
				require.Equal(t, string(tc.repoReturns.user.Role), result.Role)
				require.Equal(t, tc.repoReturns.user.CompanyID, result.CompanyID)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func Test_Service_UpdateUser(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	now := time.Now()
	newEmail := "updated@mail.com"

	type updateInput struct {
		userID uuid.UUID
		email  string
	}

	type repoReturns struct {
		updateErr error
		user      *usermodel.User
		getErr    error
	}

	cases := []struct {
		name        string
		input       updateInput
		repoReturns repoReturns
		expectErr   error
	}{
		{
			name: "success",
			input: updateInput{
				userID: userID,
				email:  newEmail,
			},
			repoReturns: repoReturns{
				updateErr: nil,
				user: &usermodel.User{
					ID:        userID,
					Email:     newEmail,
					Role:      usermodel.Seller,
					CompanyID: nil,
					CreatedAt: now,
				},
				getErr: nil,
			},
			expectErr: nil,
		},
		{
			name: "duplicate email",
			input: updateInput{
				userID: userID,
				email:  "duplicate@mail.com",
			},
			repoReturns: repoReturns{
				updateErr: usermodel.ErrEmailAlreadyExists,
			},
			expectErr: service.ErrDuplicateEmail,
		},
		{
			name: "user not found on update",
			input: updateInput{
				userID: uuid.New(),
				email:  newEmail,
			},
			repoReturns: repoReturns{
				updateErr: usermodel.ErrUserNotFound,
			},
			expectErr: service.ErrUserNotFound,
		},
		{
			name: "update unexpected error",
			input: updateInput{
				userID: userID,
				email:  newEmail,
			},
			repoReturns: repoReturns{
				updateErr: errors.New("db error"),
			},
			expectErr: nil, // wrapped, not a sentinel
		},
		{
			name: "update succeeds but get user fails",
			input: updateInput{
				userID: userID,
				email:  newEmail,
			},
			repoReturns: repoReturns{
				updateErr: nil,
				user:      nil,
				getErr:    errors.New("db connection lost"),
			},
			expectErr: nil, // wrapped, not a sentinel
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := mocks.NewRepository(t)

			mockRepo.
				On("UpdateUser", mock.Anything, tc.input.userID, tc.input.email).
				Return(tc.repoReturns.updateErr).
				Once()

			if tc.repoReturns.updateErr == nil {
				mockRepo.
					On("GetUserByID", mock.Anything, tc.input.userID).
					Return(tc.repoReturns.user, tc.repoReturns.getErr).
					Once()
			}

			svc := NewService(mockRepo)

			result, err := svc.UpdateUser(ctx, UpdateUserInput{
				UserID: tc.input.userID,
				Email:  tc.input.email,
			})

			if tc.expectErr != nil {
				require.ErrorIs(t, err, tc.expectErr)
				require.Nil(t, result)
			} else if tc.repoReturns.updateErr != nil || tc.repoReturns.getErr != nil {
				require.Error(t, err)
				require.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				require.Equal(t, tc.repoReturns.user.ID, result.ID)
				require.Equal(t, tc.repoReturns.user.Email, result.Email)
				require.Equal(t, string(tc.repoReturns.user.Role), result.Role)
				require.Equal(t, tc.repoReturns.user.CompanyID, result.CompanyID)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func Test_Service_ListUsersByCompany(t *testing.T) {
	ctx := context.Background()
	companyID := uuid.New()
	now := time.Now()

	type listInput struct {
		companyID uuid.UUID
	}

	type repoReturns struct {
		company    *companymodel.Company
		companyErr error
		users      []usermodel.User
		usersErr   error
	}

	cases := []struct {
		name        string
		input       listInput
		repoReturns repoReturns
		expectErr   error
	}{
		{
			name: "success with users",
			input: listInput{
				companyID: companyID,
			},
			repoReturns: repoReturns{
				company: &companymodel.Company{
					ID:        companyID,
					Name:      "Test Corp",
					CreatedAt: now,
				},
				companyErr: nil,
				users: []usermodel.User{
					{
						ID:        uuid.New(),
						Email:     "user1@mail.com",
						Role:      usermodel.Seller,
						CompanyID: &companyID,
						CreatedAt: now,
					},
					{
						ID:        uuid.New(),
						Email:     "user2@mail.com",
						Role:      usermodel.Buyer,
						CompanyID: &companyID,
						CreatedAt: now,
					},
				},
				usersErr: nil,
			},
			expectErr: nil,
		},
		{
			name: "success with empty list",
			input: listInput{
				companyID: companyID,
			},
			repoReturns: repoReturns{
				company: &companymodel.Company{
					ID:        companyID,
					Name:      "Empty Corp",
					CreatedAt: now,
				},
				companyErr: nil,
				users:      []usermodel.User{},
				usersErr:   nil,
			},
			expectErr: nil,
		},
		{
			name: "company not found",
			input: listInput{
				companyID: uuid.New(),
			},
			repoReturns: repoReturns{
				company:    nil,
				companyErr: companymodel.ErrCompanyNotFound,
			},
			expectErr: service.ErrCompanyNotFound,
		},
		{
			name: "check company exists unexpected error",
			input: listInput{
				companyID: companyID,
			},
			repoReturns: repoReturns{
				company:    nil,
				companyErr: errors.New("db connection lost"),
			},
			expectErr: nil, // wrapped, not a sentinel
		},
		{
			name: "list users unexpected error",
			input: listInput{
				companyID: companyID,
			},
			repoReturns: repoReturns{
				company: &companymodel.Company{
					ID:        companyID,
					Name:      "Test Corp",
					CreatedAt: now,
				},
				companyErr: nil,
				users:      nil,
				usersErr:   errors.New("db query failed"),
			},
			expectErr: nil, // wrapped, not a sentinel
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := mocks.NewRepository(t)

			mockRepo.
				On("GetCompanyByID", mock.Anything, tc.input.companyID).
				Return(tc.repoReturns.company, tc.repoReturns.companyErr).
				Once()

			if tc.repoReturns.companyErr == nil {
				mockRepo.
					On("ListUsersByCompanyID", mock.Anything, tc.input.companyID).
					Return(tc.repoReturns.users, tc.repoReturns.usersErr).
					Once()
			}

			svc := NewService(mockRepo)

			result, err := svc.ListUsersByCompany(ctx, tc.input.companyID)
			if tc.expectErr != nil {
				require.ErrorIs(t, err, tc.expectErr)
				require.Nil(t, result)
			} else if tc.repoReturns.companyErr != nil {
				require.Error(t, err)
				require.Nil(t, result)
			} else if tc.repoReturns.usersErr != nil {
				require.Error(t, err)
				require.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.Len(t, result, len(tc.repoReturns.users))
				for i, u := range tc.repoReturns.users {
					require.Equal(t, u.ID, result[i].ID)
					require.Equal(t, u.Email, result[i].Email)
					require.Equal(t, string(u.Role), result[i].Role)
					require.Equal(t, u.CompanyID, result[i].CompanyID)
				}
			}

			mockRepo.AssertExpectations(t)
		})
	}
}
