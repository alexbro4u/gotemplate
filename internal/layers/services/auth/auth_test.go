package auth_test

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/alexbro4u/gotemplate/internal/core/jwt"
	"github.com/alexbro4u/gotemplate/internal/dto/repository"
	"github.com/alexbro4u/gotemplate/internal/dto/service"
	"github.com/alexbro4u/gotemplate/internal/entity"
	apperrors "github.com/alexbro4u/gotemplate/internal/errors"
	authservice "github.com/alexbro4u/gotemplate/internal/layers/services/auth"
	"github.com/alexbro4u/gotemplate/internal/mocks"
	"github.com/alexbro4u/gotemplate/pkg/password"
)

const testSecretKey = "test-secret-key-for-unit-tests-minimum-32-chars!!"

type authTestSuite struct {
	svc           *authservice.Service
	userRepo      *mocks.MockUserRepository
	userGroupRepo *mocks.MockUserGroupRepository
	uow           *mocks.MockUnitOfWork
	jwtSvc        *jwt.Service
}

func setupAuthTest(t *testing.T) *authTestSuite {
	t.Helper()
	s := &authTestSuite{
		userRepo:      mocks.NewMockUserRepository(t),
		userGroupRepo: mocks.NewMockUserGroupRepository(t),
		uow:           mocks.NewMockUnitOfWork(t),
		jwtSvc:        jwt.New(testSecretKey),
	}
	var err error
	s.svc, err = authservice.New(authservice.Deps{
		Logger:        slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
		UoW:           s.uow,
		UserRepo:      s.userRepo,
		UserGroupRepo: s.userGroupRepo,
		JWTService:    s.jwtSvc,
		Validator:     validator.New(),
	})
	require.NoError(t, err)
	return s
}

func testUserWithPassword(t *testing.T, pwd string) *entity.User {
	t.Helper()
	hash, err := password.Hash(pwd)
	require.NoError(t, err)
	return &entity.User{
		ID:           1,
		UUID:         uuid.New(),
		Email:        "user@example.com",
		Name:         "Test User",
		PasswordHash: hash,
		Role:         entity.RoleUser,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

// --- Login ---

func TestService_Login_Success(t *testing.T) {
	s := setupAuthTest(t)
	ctx := context.Background()
	pwd := "password123"
	user := testUserWithPassword(t, pwd)

	s.userRepo.EXPECT().
		GetByEmail(mock.Anything, user.Email).
		Return(&repository.GetUserOutput{User: user}, nil)

	s.userGroupRepo.EXPECT().
		GetGroupNamesByUserID(mock.Anything, user.ID).
		Return([]string{"users"}, nil)

	out, err := s.svc.Login(ctx, service.LoginInput{Email: user.Email, Password: pwd})
	require.NoError(t, err)
	assert.Equal(t, user.Email, out.User.Email)
	assert.NotEmpty(t, out.AccessToken)
	assert.NotEmpty(t, out.RefreshToken)
}

func TestService_Login_WrongPassword(t *testing.T) {
	s := setupAuthTest(t)
	ctx := context.Background()
	user := testUserWithPassword(t, "correct")

	s.userRepo.EXPECT().
		GetByEmail(mock.Anything, user.Email).
		Return(&repository.GetUserOutput{User: user}, nil)

	_, err := s.svc.Login(ctx, service.LoginInput{Email: user.Email, Password: "wrong"})
	require.Error(t, err)
	assert.True(t, apperrors.CodeIs(err, apperrors.CodeInvalidCredentials))
}

func TestService_Login_UserNotFound(t *testing.T) {
	s := setupAuthTest(t)
	ctx := context.Background()

	s.userRepo.EXPECT().
		GetByEmail(mock.Anything, "noone@example.com").
		Return(nil, apperrors.ErrUserNotFound)

	_, err := s.svc.Login(ctx, service.LoginInput{Email: "noone@example.com", Password: "any"})
	require.Error(t, err)
	assert.True(t, apperrors.CodeIs(err, apperrors.CodeInvalidCredentials))
}

func TestService_Login_ValidationError(t *testing.T) {
	s := setupAuthTest(t)
	ctx := context.Background()

	_, err := s.svc.Login(ctx, service.LoginInput{Email: "not-an-email", Password: ""})
	require.Error(t, err)
}

// --- Refresh ---

func TestService_Refresh_Success(t *testing.T) {
	s := setupAuthTest(t)
	ctx := context.Background()
	user := testUserWithPassword(t, "password123")

	pair, err := s.jwtSvc.GenerateTokenPair(user.UUID, user.Email, string(user.Role), []string{"users"})
	require.NoError(t, err)

	s.userRepo.EXPECT().
		Get(mock.Anything, repository.GetUserInput{UUID: user.UUID}).
		Return(&repository.GetUserOutput{User: user}, nil)

	s.userGroupRepo.EXPECT().
		GetGroupNamesByUserID(mock.Anything, user.ID).
		Return([]string{"users"}, nil)

	out, err := s.svc.Refresh(ctx, service.RefreshInput{RefreshToken: pair.RefreshToken})
	require.NoError(t, err)
	assert.NotEmpty(t, out.AccessToken)
	assert.NotEmpty(t, out.RefreshToken)
}

func TestService_Refresh_InvalidToken(t *testing.T) {
	s := setupAuthTest(t)
	ctx := context.Background()

	_, err := s.svc.Refresh(ctx, service.RefreshInput{RefreshToken: "invalid-token"})
	require.Error(t, err)
}

func TestService_Refresh_AccessTokenUsedAsRefresh(t *testing.T) {
	s := setupAuthTest(t)
	ctx := context.Background()
	user := testUserWithPassword(t, "password123")

	pair, err := s.jwtSvc.GenerateTokenPair(user.UUID, user.Email, string(user.Role), []string{"users"})
	require.NoError(t, err)

	_, err = s.svc.Refresh(ctx, service.RefreshInput{RefreshToken: pair.AccessToken})
	require.Error(t, err)
}

// --- GetMe ---

func TestService_GetMe_Success(t *testing.T) {
	s := setupAuthTest(t)
	ctx := context.Background()
	user := testUserWithPassword(t, "password123")

	s.userRepo.EXPECT().
		Get(mock.Anything, repository.GetUserInput{UUID: user.UUID}).
		Return(&repository.GetUserOutput{User: user}, nil)

	out, err := s.svc.GetMe(ctx, service.GetMeInput{UserUUID: user.UUID})
	require.NoError(t, err)
	assert.Equal(t, user.Email, out.User.Email)
}

func TestService_GetMe_NotFound(t *testing.T) {
	s := setupAuthTest(t)
	ctx := context.Background()
	id := uuid.New()

	s.userRepo.EXPECT().
		Get(mock.Anything, repository.GetUserInput{UUID: id}).
		Return(nil, apperrors.ErrUserNotFound)

	_, err := s.svc.GetMe(ctx, service.GetMeInput{UserUUID: id})
	require.Error(t, err)
	assert.True(t, apperrors.CodeIs(err, apperrors.CodeUserNotFound))
}

// --- UpdateMe ---

func TestService_UpdateMe_Success(t *testing.T) {
	s := setupAuthTest(t)
	ctx := context.Background()
	id := uuid.New()
	name := "New Name"

	s.userRepo.EXPECT().
		Update(mock.Anything, repository.UpdateUserInput{UUID: id, Name: &name}).
		Return(nil)

	err := s.svc.UpdateMe(ctx, service.UpdateMeInput{UserUUID: id, Name: &name})
	require.NoError(t, err)
}

func TestService_UpdateMe_NotFound(t *testing.T) {
	s := setupAuthTest(t)
	ctx := context.Background()
	id := uuid.New()
	email := "new@example.com"

	s.userRepo.EXPECT().
		Update(mock.Anything, repository.UpdateUserInput{UUID: id, Email: &email}).
		Return(apperrors.ErrUserNotFound)

	err := s.svc.UpdateMe(ctx, service.UpdateMeInput{UserUUID: id, Email: &email})
	require.Error(t, err)
	assert.True(t, apperrors.CodeIs(err, apperrors.CodeUserNotFound))
}

// --- ChangePassword ---

func TestService_ChangePassword_Success(t *testing.T) {
	s := setupAuthTest(t)
	ctx := context.Background()
	oldPwd := "oldpassword"
	user := testUserWithPassword(t, oldPwd)

	s.userRepo.EXPECT().
		Get(mock.Anything, repository.GetUserInput{UUID: user.UUID}).
		Return(&repository.GetUserOutput{User: user}, nil)

	s.userRepo.EXPECT().
		UpdatePassword(mock.Anything, mock.MatchedBy(func(in repository.UpdatePasswordInput) bool {
			return in.UUID == user.UUID && in.PasswordHash != ""
		})).
		Return(nil)

	err := s.svc.ChangePassword(ctx, service.ChangePasswordInput{
		UserUUID:    user.UUID,
		OldPassword: oldPwd,
		NewPassword: "newpassword123",
	})
	require.NoError(t, err)
}

func TestService_ChangePassword_WrongOldPassword(t *testing.T) {
	s := setupAuthTest(t)
	ctx := context.Background()
	user := testUserWithPassword(t, "correct")

	s.userRepo.EXPECT().
		Get(mock.Anything, repository.GetUserInput{UUID: user.UUID}).
		Return(&repository.GetUserOutput{User: user}, nil)

	err := s.svc.ChangePassword(ctx, service.ChangePasswordInput{
		UserUUID:    user.UUID,
		OldPassword: "wrong",
		NewPassword: "newpassword123",
	})
	require.Error(t, err)
	assert.True(t, apperrors.CodeIs(err, apperrors.CodeInvalidCredentials))
}

func TestService_ChangePassword_ValidationError(t *testing.T) {
	s := setupAuthTest(t)
	ctx := context.Background()

	err := s.svc.ChangePassword(ctx, service.ChangePasswordInput{
		UserUUID:    uuid.Nil,
		OldPassword: "",
		NewPassword: "short",
	})
	require.Error(t, err)
}
