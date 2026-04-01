//go:build integration

package auth_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/alexbro4u/uowkit/uow"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alexbro4u/gotemplate/internal/core/jwt"
	"github.com/alexbro4u/gotemplate/internal/dto/service"
	apperrors "github.com/alexbro4u/gotemplate/internal/errors"
	blacklistrepo "github.com/alexbro4u/gotemplate/internal/layers/repositories/blacklist"
	passwordresetrepo "github.com/alexbro4u/gotemplate/internal/layers/repositories/password_reset"
	userrepo "github.com/alexbro4u/gotemplate/internal/layers/repositories/user"
	usergrouprepo "github.com/alexbro4u/gotemplate/internal/layers/repositories/user_group"
	authservice "github.com/alexbro4u/gotemplate/internal/layers/services/auth"
	"github.com/alexbro4u/gotemplate/internal/testutil"
	"github.com/alexbro4u/gotemplate/pkg/sqlxadapter"
)

func setupIntegrationAuth(t *testing.T) *authservice.Service {
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
	br, err := blacklistrepo.New(blacklistrepo.Deps{DB: db, Validator: v})
	require.NoError(t, err)
	pr, err := passwordresetrepo.New(passwordresetrepo.Deps{DB: db, Validator: v})
	require.NoError(t, err)

	txFactory := sqlxadapter.NewTxFactory(db, nil)
	unitOfWork := uow.New(txFactory)

	svc, err := authservice.New(authservice.Deps{
		Logger:            slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
		UoW:               unitOfWork,
		UserRepo:          ur,
		UserGroupRepo:     ugr,
		BlacklistRepo:     br,
		PasswordResetRepo: pr,
		JWTService:        jwt.New("integration-test-secret-key-minimum-32-chars!!"),
		Validator:         v,
	})
	require.NoError(t, err)
	return svc
}

func TestIntegration_Register_Success(t *testing.T) {
	svc := setupIntegrationAuth(t)
	ctx := context.Background()

	out, err := svc.Register(ctx, service.RegisterInput{
		Email:    "register@example.com",
		Name:     "Register User",
		Password: "password123",
	})
	require.NoError(t, err)
	assert.Equal(t, "register@example.com", out.User.Email)
	assert.Equal(t, "Register User", out.User.Name)
	assert.NotEmpty(t, out.AccessToken)
	assert.NotEmpty(t, out.RefreshToken)
}

func TestIntegration_Register_Duplicate(t *testing.T) {
	svc := setupIntegrationAuth(t)
	ctx := context.Background()

	_, err := svc.Register(ctx, service.RegisterInput{
		Email: "dup@example.com", Name: "User", Password: "password123",
	})
	require.NoError(t, err)

	_, err = svc.Register(ctx, service.RegisterInput{
		Email: "dup@example.com", Name: "User2", Password: "password456",
	})
	require.Error(t, err)
	assert.True(t, apperrors.CodeIs(err, apperrors.CodeUserAlreadyExists))
}

func TestIntegration_LoginAfterRegister(t *testing.T) {
	svc := setupIntegrationAuth(t)
	ctx := context.Background()

	_, err := svc.Register(ctx, service.RegisterInput{
		Email: "login@example.com", Name: "Login User", Password: "password123",
	})
	require.NoError(t, err)

	out, err := svc.Login(ctx, service.LoginInput{
		Email: "login@example.com", Password: "password123",
	})
	require.NoError(t, err)
	assert.Equal(t, "login@example.com", out.User.Email)
	assert.NotEmpty(t, out.AccessToken)
}

func TestIntegration_RefreshAfterRegister(t *testing.T) {
	svc := setupIntegrationAuth(t)
	ctx := context.Background()

	reg, err := svc.Register(ctx, service.RegisterInput{
		Email: "refresh@example.com", Name: "Refresh User", Password: "password123",
	})
	require.NoError(t, err)

	out, err := svc.Refresh(ctx, service.RefreshInput{RefreshToken: reg.RefreshToken})
	require.NoError(t, err)
	assert.NotEmpty(t, out.AccessToken)
	assert.NotEmpty(t, out.RefreshToken)
}

func TestIntegration_ChangePassword(t *testing.T) {
	svc := setupIntegrationAuth(t)
	ctx := context.Background()

	reg, err := svc.Register(ctx, service.RegisterInput{
		Email: "chpwd@example.com", Name: "ChPwd User", Password: "oldpassword",
	})
	require.NoError(t, err)

	// Change password
	claims, err := jwt.New("integration-test-secret-key-minimum-32-chars!!").ValidateToken(reg.AccessToken)
	require.NoError(t, err)

	err = svc.ChangePassword(ctx, service.ChangePasswordInput{
		UserUUID:    claims.UserUUID,
		OldPassword: "oldpassword",
		NewPassword: "newpassword123",
	})
	require.NoError(t, err)

	// Login with new password
	out, err := svc.Login(ctx, service.LoginInput{
		Email: "chpwd@example.com", Password: "newpassword123",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, out.AccessToken)

	// Old password no longer works
	_, err = svc.Login(ctx, service.LoginInput{
		Email: "chpwd@example.com", Password: "oldpassword",
	})
	require.Error(t, err)
}
