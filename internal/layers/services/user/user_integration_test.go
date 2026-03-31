//go:build integration

package user_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/alexbro4u/uowkit/uow"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alexbro4u/gotemplate/internal/dto/service"
	apperrors "github.com/alexbro4u/gotemplate/internal/errors"
	userrepo "github.com/alexbro4u/gotemplate/internal/layers/repositories/user"
	usergrouprepo "github.com/alexbro4u/gotemplate/internal/layers/repositories/user_group"
	userservice "github.com/alexbro4u/gotemplate/internal/layers/services/user"
	"github.com/alexbro4u/gotemplate/internal/testutil"
	"github.com/alexbro4u/gotemplate/pkg/sqlxadapter"
)

func setupIntegrationUser(t *testing.T) *userservice.Service {
	t.Helper()
	db := testutil.NewTestDB(t)
	testutil.RunMigrations(t, db)
	testutil.TruncateAll(t, db)
	testutil.SeedGroups(t, db)

	v := validator.New()
	ur, err := userrepo.New(userrepo.Deps{DB: db, Validator: v})
	require.NoError(t, err)
	ugr, err := usergrouprepo.New(usergrouprepo.Deps{DB: db, Validator: v})
	require.NoError(t, err)

	txFactory := sqlxadapter.NewTxFactory(db, nil)
	unitOfWork := uow.New(txFactory)

	svc, err := userservice.New(userservice.Deps{
		Logger:        slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
		UoW:           unitOfWork,
		UserRepo:      ur,
		UserGroupRepo: ugr,
		Validator:     v,
	})
	require.NoError(t, err)
	return svc
}

func TestIntegration_CreateUser_Success(t *testing.T) {
	svc := setupIntegrationUser(t)
	ctx := context.Background()

	out, err := svc.Create(ctx, service.CreateUserInput{
		Email:    "create@example.com",
		Name:     "Created User",
		Password: "password123",
	})
	require.NoError(t, err)
	assert.Equal(t, "create@example.com", out.User.Email)
	assert.Equal(t, "Created User", out.User.Name)
	assert.NotEmpty(t, out.User.UUID)
}

func TestIntegration_CreateUser_Duplicate(t *testing.T) {
	svc := setupIntegrationUser(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, service.CreateUserInput{
		Email: "dup@example.com", Name: "User", Password: "password123",
	})
	require.NoError(t, err)

	_, err = svc.Create(ctx, service.CreateUserInput{
		Email: "dup@example.com", Name: "User2", Password: "password456",
	})
	require.Error(t, err)
	assert.True(t, apperrors.CodeIs(err, apperrors.CodeUserAlreadyExists))
}

func TestIntegration_CRUD_Flow(t *testing.T) {
	svc := setupIntegrationUser(t)
	ctx := context.Background()

	// Create
	created, err := svc.Create(ctx, service.CreateUserInput{
		Email: "crud@example.com", Name: "CRUD User", Password: "password123",
	})
	require.NoError(t, err)

	userUUID, err := uuid.Parse(created.User.UUID)
	require.NoError(t, err)

	// Get
	got, err := svc.Get(ctx, service.GetUserInput{UUID: userUUID})
	require.NoError(t, err)
	assert.Equal(t, "crud@example.com", got.User.Email)

	// Update
	newName := "Updated Name"
	err = svc.Update(ctx, service.UpdateUserInput{UUID: userUUID, Name: &newName})
	require.NoError(t, err)

	got, err = svc.Get(ctx, service.GetUserInput{UUID: userUUID})
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", got.User.Name)

	// Delete
	err = svc.Delete(ctx, service.DeleteUserInput{UUID: userUUID})
	require.NoError(t, err)

	// Get after delete → not found
	_, err = svc.Get(ctx, service.GetUserInput{UUID: userUUID})
	require.Error(t, err)
	assert.True(t, apperrors.CodeIs(err, apperrors.CodeUserNotFound))
}

func TestIntegration_List(t *testing.T) {
	svc := setupIntegrationUser(t)
	ctx := context.Background()

	for i := range 5 {
		_, err := svc.Create(ctx, service.CreateUserInput{
			Email:    fmt.Sprintf("list%d@example.com", i),
			Name:     fmt.Sprintf("User %d", i),
			Password: "password123",
		})
		require.NoError(t, err)
	}

	out, err := svc.List(ctx, service.ListUsersInput{Limit: 3, Offset: 0})
	require.NoError(t, err)
	assert.Len(t, out.Users, 3)
	assert.Equal(t, int64(5), out.Total)

	out2, err := svc.List(ctx, service.ListUsersInput{Limit: 10, Offset: 3})
	require.NoError(t, err)
	assert.Len(t, out2.Users, 2)
}
