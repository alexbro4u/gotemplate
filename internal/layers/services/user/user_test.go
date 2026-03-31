package user_test

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

	"github.com/alexbro4u/gotemplate/internal/dto/repository"
	"github.com/alexbro4u/gotemplate/internal/dto/service"
	"github.com/alexbro4u/gotemplate/internal/entity"
	apperrors "github.com/alexbro4u/gotemplate/internal/errors"
	userservice "github.com/alexbro4u/gotemplate/internal/layers/services/user"
	"github.com/alexbro4u/gotemplate/internal/mocks"
)

type userTestSuite struct {
	svc           *userservice.Service
	userRepo      *mocks.MockUserRepository
	userGroupRepo *mocks.MockUserGroupRepository
	uow           *mocks.MockUnitOfWork
}

func setupUserTest(t *testing.T) *userTestSuite {
	t.Helper()
	s := &userTestSuite{
		userRepo:      mocks.NewMockUserRepository(t),
		userGroupRepo: mocks.NewMockUserGroupRepository(t),
		uow:           mocks.NewMockUnitOfWork(t),
	}
	var err error
	s.svc, err = userservice.New(userservice.Deps{
		Logger:        slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
		UoW:           s.uow,
		UserRepo:      s.userRepo,
		UserGroupRepo: s.userGroupRepo,
		Validator:     validator.New(),
	})
	require.NoError(t, err)
	return s
}

func testUser() *entity.User {
	return &entity.User{
		ID:           1,
		UUID:         uuid.New(),
		Email:        "test@example.com",
		Name:         "Test User",
		PasswordHash: "$2a$10$abcdefghijklmnopqrstuuABCDEFGHIJKLMNOPQRSTUVWXYZ012",
		Role:         entity.RoleUser,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

func TestService_Get_Success(t *testing.T) {
	s := setupUserTest(t)
	ctx := context.Background()
	user := testUser()

	s.userRepo.EXPECT().
		Get(mock.Anything, repository.GetUserInput{UUID: user.UUID}).
		Return(&repository.GetUserOutput{User: user}, nil)

	out, err := s.svc.Get(ctx, service.GetUserInput{UUID: user.UUID})
	require.NoError(t, err)
	assert.Equal(t, user.Email, out.User.Email)
	assert.Equal(t, user.Name, out.User.Name)
}

func TestService_Get_NotFound(t *testing.T) {
	s := setupUserTest(t)
	ctx := context.Background()
	id := uuid.New()

	s.userRepo.EXPECT().
		Get(mock.Anything, repository.GetUserInput{UUID: id}).
		Return(nil, apperrors.ErrUserNotFound)

	_, err := s.svc.Get(ctx, service.GetUserInput{UUID: id})
	require.Error(t, err)
	assert.True(t, apperrors.CodeIs(err, apperrors.CodeUserNotFound))
}

func TestService_Update_Success(t *testing.T) {
	s := setupUserTest(t)
	ctx := context.Background()
	id := uuid.New()
	newEmail := "new@example.com"

	s.userRepo.EXPECT().
		Update(mock.Anything, repository.UpdateUserInput{UUID: id, Email: &newEmail, Name: nil}).
		Return(nil)

	err := s.svc.Update(ctx, service.UpdateUserInput{UUID: id, Email: &newEmail})
	require.NoError(t, err)
}

func TestService_Update_NotFound(t *testing.T) {
	s := setupUserTest(t)
	ctx := context.Background()
	id := uuid.New()
	name := "New Name"

	s.userRepo.EXPECT().
		Update(mock.Anything, repository.UpdateUserInput{UUID: id, Name: &name}).
		Return(apperrors.ErrUserNotFound)

	err := s.svc.Update(ctx, service.UpdateUserInput{UUID: id, Name: &name})
	require.Error(t, err)
	assert.True(t, apperrors.CodeIs(err, apperrors.CodeUserNotFound))
}

func TestService_Update_ValidationError(t *testing.T) {
	s := setupUserTest(t)
	ctx := context.Background()
	empty := ""

	err := s.svc.Update(ctx, service.UpdateUserInput{UUID: uuid.Nil, Email: &empty})
	require.Error(t, err)
}

func TestService_Delete_Success(t *testing.T) {
	s := setupUserTest(t)
	ctx := context.Background()
	id := uuid.New()

	s.userRepo.EXPECT().
		Delete(mock.Anything, repository.DeleteUserInput{UUID: id}).
		Return(nil)

	err := s.svc.Delete(ctx, service.DeleteUserInput{UUID: id})
	require.NoError(t, err)
}

func TestService_Delete_NotFound(t *testing.T) {
	s := setupUserTest(t)
	ctx := context.Background()
	id := uuid.New()

	s.userRepo.EXPECT().
		Delete(mock.Anything, repository.DeleteUserInput{UUID: id}).
		Return(apperrors.ErrUserNotFound)

	err := s.svc.Delete(ctx, service.DeleteUserInput{UUID: id})
	require.Error(t, err)
	assert.True(t, apperrors.CodeIs(err, apperrors.CodeUserNotFound))
}

func TestService_List_Success(t *testing.T) {
	s := setupUserTest(t)
	ctx := context.Background()
	users := []*entity.User{testUser(), testUser()}

	s.userRepo.EXPECT().
		List(mock.Anything, 10, 0).
		Return(&repository.ListUsersOutput{Users: users, Total: 2}, nil)

	out, err := s.svc.List(ctx, service.ListUsersInput{Limit: 10, Offset: 0})
	require.NoError(t, err)
	assert.Len(t, out.Users, 2)
	assert.Equal(t, int64(2), out.Total)
	assert.Equal(t, 10, out.Limit)
	assert.Equal(t, 0, out.Offset)
}

func TestService_List_ValidationError(t *testing.T) {
	s := setupUserTest(t)
	ctx := context.Background()

	_, err := s.svc.List(ctx, service.ListUsersInput{Limit: 0})
	require.Error(t, err)
}
