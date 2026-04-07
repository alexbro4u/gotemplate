package auth

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/alexbro4u/gotemplate/internal/core/jwt"
	"github.com/alexbro4u/gotemplate/internal/dto/repository"
	"github.com/alexbro4u/gotemplate/internal/dto/service"
	apperrors "github.com/alexbro4u/gotemplate/internal/errors"
	"github.com/alexbro4u/gotemplate/internal/layers/repositories"
	"github.com/alexbro4u/gotemplate/pkg/cache"
	"github.com/alexbro4u/gotemplate/pkg/password"
	"github.com/alexbro4u/gotemplate/pkg/sqlxadapter"
	"github.com/google/uuid"

	"github.com/alexbro4u/uowkit/uow"
	"github.com/go-playground/validator/v10"
)

type Deps struct {
	Logger            *slog.Logger                         `validate:"required"`
	UoW               uow.UnitOfWork                       `validate:"required"`
	UserRepo          repositories.UserRepository          `validate:"required"`
	UserGroupRepo     repositories.UserGroupRepository     `validate:"required"`
	JWTService        *jwt.Service                         `validate:"required"`
	Validator         *validator.Validate                  `validate:"required"`
	BlacklistRepo     repositories.BlacklistRepository     `validate:"required"`
	BlacklistCache    cache.Adder                          `validate:"required"`
	PasswordResetRepo repositories.PasswordResetRepository `validate:"required"`
}

type Service struct {
	logger            *slog.Logger
	uow               uow.UnitOfWork
	userRepo          repositories.UserRepository
	userGroupRepo     repositories.UserGroupRepository
	jwtService        *jwt.Service
	validator         *validator.Validate
	blacklistRepo     repositories.BlacklistRepository
	blacklistCache    cache.Adder
	passwordResetRepo repositories.PasswordResetRepository
}

func New(deps Deps) (*Service, error) {
	if err := deps.Validator.Struct(deps); err != nil {
		return nil, apperrors.Wrap(err, "validate deps")
	}

	return &Service{
		logger:            deps.Logger,
		uow:               deps.UoW,
		userRepo:          deps.UserRepo,
		userGroupRepo:     deps.UserGroupRepo,
		jwtService:        deps.JWTService,
		validator:         deps.Validator,
		blacklistRepo:     deps.BlacklistRepo,
		blacklistCache:    deps.BlacklistCache,
		passwordResetRepo: deps.PasswordResetRepo,
	}, nil
}

func (s *Service) Register(ctx context.Context, in service.RegisterInput) (*service.RegisterOutput, error) {
	if err := s.validator.Struct(in); err != nil {
		return nil, apperrors.Wrap(err, "validate input")
	}

	in.Email = strings.ToLower(strings.TrimSpace(in.Email))

	passwordHash, err := password.Hash(in.Password)
	if err != nil {
		return nil, apperrors.Wrap(err, "hash password")
	}

	var createOutput *repository.CreateUserOutput

	if txErr := s.uow.Do(ctx, func(tx uow.Tx) error {
		sqlxTx, ok := sqlxadapter.Unwrap(tx)
		if !ok {
			return apperrors.New("unexpected tx type")
		}

		userRepoTx := s.userRepo.WithExecutor(sqlxTx)
		userGroupRepoTx := s.userGroupRepo.WithExecutor(sqlxTx)

		var createErr error
		createOutput, createErr = userRepoTx.Create(ctx, repository.CreateUserInput{
			Email:        in.Email,
			Name:         in.Name,
			PasswordHash: passwordHash,
			Role:         nil,
		})
		if createErr != nil {
			return createErr
		}

		if groupErr := userGroupRepoTx.AddUserToGroup(ctx, createOutput.User.ID, "users"); groupErr != nil {
			return apperrors.Wrap(groupErr, "add user to default group")
		}

		return nil
	}); txErr != nil {
		if apperrors.CodeIs(txErr, apperrors.CodeUniqueViolation) {
			return nil, apperrors.ErrUserAlreadyExists
		}
		return nil, apperrors.Wrap(txErr, "register tx")
	}

	groups, err := s.userGroupRepo.GetGroupNamesByUserID(ctx, createOutput.User.ID)
	if err != nil {
		return nil, apperrors.Wrap(err, "get user groups")
	}
	pair, err := s.jwtService.GenerateTokenPair(
		createOutput.User.UUID,
		createOutput.User.Email,
		string(createOutput.User.Role),
		groups,
	)
	if err != nil {
		return nil, apperrors.Wrap(err, "generate token pair")
	}

	return &service.RegisterOutput{
		User:         service.UserToDTO(createOutput.User),
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
	}, nil
}

func (s *Service) Login(ctx context.Context, in service.LoginInput) (*service.LoginOutput, error) {
	if err := s.validator.Struct(in); err != nil {
		return nil, apperrors.Wrap(err, "validate input")
	}

	in.Email = strings.ToLower(strings.TrimSpace(in.Email))

	userOutput, err := s.userRepo.GetByEmail(ctx, in.Email)
	if err != nil {
		if apperrors.CodeIs(err, apperrors.CodeUserNotFound) {
			return nil, apperrors.ErrInvalidCredentials
		}
		return nil, apperrors.Wrap(err, "get user by email")
	}

	if !password.Compare(in.Password, userOutput.User.PasswordHash) {
		return nil, apperrors.ErrInvalidCredentials
	}

	groups, err := s.userGroupRepo.GetGroupNamesByUserID(ctx, userOutput.User.ID)
	if err != nil {
		return nil, apperrors.Wrap(err, "get user groups")
	}
	pair, err := s.jwtService.GenerateTokenPair(
		userOutput.User.UUID,
		userOutput.User.Email,
		string(userOutput.User.Role),
		groups,
	)
	if err != nil {
		return nil, apperrors.Wrap(err, "generate token pair")
	}

	return &service.LoginOutput{
		User:         service.UserToDTO(userOutput.User),
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
	}, nil
}

func (s *Service) Refresh(ctx context.Context, in service.RefreshInput) (*service.RefreshOutput, error) {
	if err := s.validator.Struct(in); err != nil {
		return nil, apperrors.Wrap(err, "validate input")
	}

	claims, err := s.jwtService.ValidateRefreshToken(in.RefreshToken)
	if err != nil {
		return nil, apperrors.ErrInvalidCredentials
	}

	userOutput, err := s.userRepo.Get(ctx, repository.GetUserInput{UUID: claims.UserUUID})
	if err != nil {
		return nil, apperrors.Wrap(err, "get user for refresh")
	}

	groups, err := s.userGroupRepo.GetGroupNamesByUserID(ctx, userOutput.User.ID)
	if err != nil {
		return nil, apperrors.Wrap(err, "get user groups")
	}

	pair, err := s.jwtService.GenerateTokenPair(
		userOutput.User.UUID,
		userOutput.User.Email,
		string(userOutput.User.Role),
		groups,
	)
	if err != nil {
		return nil, apperrors.Wrap(err, "generate token pair")
	}

	return &service.RefreshOutput{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
	}, nil
}

func (s *Service) GetMe(ctx context.Context, in service.GetMeInput) (*service.GetMeOutput, error) {
	output, err := s.userRepo.Get(ctx, repository.GetUserInput{
		UUID: in.UserUUID,
	})
	if err != nil {
		if apperrors.CodeIs(err, apperrors.CodeUserNotFound) {
			return nil, apperrors.ErrUserNotFound
		}
		return nil, apperrors.Wrap(err, "get me")
	}

	return &service.GetMeOutput{
		User: service.UserToDTO(output.User),
	}, nil
}

func (s *Service) UpdateMe(ctx context.Context, in service.UpdateMeInput) error {
	if err := s.validator.Struct(in); err != nil {
		return apperrors.Wrap(err, "validate input")
	}

	if in.Email != nil {
		normalized := strings.ToLower(strings.TrimSpace(*in.Email))
		in.Email = &normalized
	}

	err := s.userRepo.Update(ctx, repository.UpdateUserInput{
		UUID:  in.UserUUID,
		Email: in.Email,
		Name:  in.Name,
	})
	if err != nil {
		if apperrors.CodeIs(err, apperrors.CodeUserNotFound) {
			return apperrors.ErrUserNotFound
		}
		return apperrors.Wrap(err, "update me")
	}

	return nil
}

func (s *Service) Logout(ctx context.Context, in service.LogoutInput) error {
	if err := s.blacklistRepo.Add(ctx, repository.AddToBlacklistInput{
		JTI:       in.JTI,
		ExpiresAt: in.ExpiresAt,
	}); err != nil {
		return err
	}
	s.blacklistCache.Add(in.JTI, in.ExpiresAt)
	return nil
}

func (s *Service) ChangePassword(ctx context.Context, in service.ChangePasswordInput) error {
	if err := s.validator.Struct(in); err != nil {
		return apperrors.Wrap(err, "validate input")
	}

	userOutput, err := s.userRepo.Get(ctx, repository.GetUserInput{
		UUID: in.UserUUID,
	})
	if err != nil {
		if apperrors.CodeIs(err, apperrors.CodeUserNotFound) {
			return apperrors.ErrUserNotFound
		}
		return apperrors.Wrap(err, "get user for password change")
	}

	if !password.Compare(in.OldPassword, userOutput.User.PasswordHash) {
		return apperrors.ErrInvalidCredentials
	}

	newHash, err := password.Hash(in.NewPassword)
	if err != nil {
		return apperrors.Wrap(err, "hash new password")
	}

	if updErr := s.userRepo.UpdatePassword(ctx, repository.UpdatePasswordInput{
		UUID:         in.UserUUID,
		PasswordHash: newHash,
	}); updErr != nil {
		return apperrors.Wrap(updErr, "update password")
	}

	if in.JTI != "" {
		if blErr := s.blacklistRepo.Add(ctx, repository.AddToBlacklistInput{
			JTI:       in.JTI,
			ExpiresAt: in.ExpiresAt,
		}); blErr != nil {
			s.logger.WarnContext(ctx, "failed to blacklist token after password change", "error", blErr)
		} else {
			s.blacklistCache.Add(in.JTI, in.ExpiresAt)
		}
	}

	return nil
}

const passwordResetTTL = 1 * time.Hour

func (s *Service) RequestPasswordReset(ctx context.Context, in service.RequestPasswordResetInput) (*service.RequestPasswordResetOutput, error) {
	if err := s.validator.Struct(in); err != nil {
		return nil, apperrors.Wrap(err, "validate input")
	}

	in.Email = strings.ToLower(strings.TrimSpace(in.Email))

	userOutput, userErr := s.userRepo.GetByEmail(ctx, in.Email)
	if userErr != nil {
		// Do not reveal whether email exists
		return &service.RequestPasswordResetOutput{Token: ""}, nil //nolint:nilerr // intentional: security — do not reveal if email exists
	}

	token := uuid.New().String()
	if createErr := s.passwordResetRepo.Create(ctx, repository.CreatePasswordResetInput{
		Token:     token,
		UserID:    userOutput.User.ID,
		ExpiresAt: time.Now().Add(passwordResetTTL),
	}); createErr != nil {
		return nil, apperrors.Wrap(createErr, "create password reset token")
	}

	return &service.RequestPasswordResetOutput{Token: token}, nil
}

func (s *Service) ConfirmPasswordReset(ctx context.Context, in service.ConfirmPasswordResetInput) error {
	if err := s.validator.Struct(in); err != nil {
		return apperrors.Wrap(err, "validate input")
	}

	tokenData, tokenErr := s.passwordResetRepo.Get(ctx, repository.GetPasswordResetInput{Token: in.Token})
	if tokenErr != nil {
		return apperrors.ErrTokenNotFound
	}
	if tokenData.Used {
		return apperrors.ErrTokenAlreadyUsed
	}
	if time.Now().After(tokenData.ExpiresAt) {
		return apperrors.ErrTokenExpired
	}

	newHash, err := password.Hash(in.NewPassword)
	if err != nil {
		return apperrors.Wrap(err, "hash new password")
	}

	if updateErr := s.userRepo.UpdatePasswordByID(ctx, repository.UpdatePasswordByIDInput{
		UserID:       tokenData.UserID,
		PasswordHash: newHash,
	}); updateErr != nil {
		return apperrors.Wrap(updateErr, "update password")
	}

	return s.passwordResetRepo.MarkUsed(ctx, repository.MarkPasswordResetUsedInput{Token: in.Token})
}
