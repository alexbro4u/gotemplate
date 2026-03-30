package auth

import (
	"context"
	"log/slog"

	"github.com/alexbro4u/uowkit/uow"
	"github.com/google/uuid"
	"github.com/alexbro4u/gotemplate/internal/core/jwt"
	"github.com/alexbro4u/gotemplate/internal/dto/repository"
	"github.com/alexbro4u/gotemplate/internal/dto/service"
	apperrors "github.com/alexbro4u/gotemplate/internal/errors"
	"github.com/alexbro4u/gotemplate/internal/layers/repositories"
	"github.com/alexbro4u/gotemplate/pkg/password"
	"github.com/alexbro4u/gotemplate/pkg/sqlxadapter"

	"github.com/go-playground/validator/v10"
)

type Deps struct {
	Logger        *slog.Logger                     `validate:"required"`
	UoW           uow.UnitOfWork                   `validate:"required"`
	UserRepo      repositories.UserRepository      `validate:"required"`
	UserGroupRepo repositories.UserGroupRepository `validate:"required"`
	JWTService    *jwt.Service                     `validate:"required"`
	Validator     *validator.Validate              `validate:"required"`
}

type Service struct {
	logger        *slog.Logger
	uow           uow.UnitOfWork
	userRepo      repositories.UserRepository
	userGroupRepo repositories.UserGroupRepository
	jwtService    *jwt.Service
	validator     *validator.Validate
}

func New(deps Deps) (*Service, error) {
	if err := deps.Validator.Struct(deps); err != nil {
		return nil, apperrors.Wrap(err, "validate deps")
	}

	return &Service{
		logger:        deps.Logger,
		uow:           deps.UoW,
		userRepo:      deps.UserRepo,
		userGroupRepo: deps.UserGroupRepo,
		jwtService:    deps.JWTService,
		validator:     deps.Validator,
	}, nil
}

func (s *Service) Register(ctx context.Context, in service.RegisterInput) (*service.RegisterOutput, error) {
	if err := s.validator.Struct(in); err != nil {
		return nil, apperrors.Wrap(err, "validate input")
	}

	passwordHash, err := password.Hash(in.Password)
	if err != nil {
		return nil, apperrors.Wrap(err, "hash password")
	}

	var createOutput *repository.CreateUserOutput

	if err := s.uow.Do(ctx, func(tx uow.Tx) error {
		sqlxTx, ok := sqlxadapter.Unwrap(tx)
		if !ok {
			return apperrors.New("unexpected tx type")
		}

		userRepoTx := s.userRepo.WithExecutor(sqlxTx)
		userGroupRepoTx := s.userGroupRepo.WithExecutor(sqlxTx)

		var err error
		createOutput, err = userRepoTx.Create(ctx, repository.CreateUserInput{
			Email:        in.Email,
			Name:         in.Name,
			PasswordHash: passwordHash,
			Role:         nil,
		})
		if err != nil {
			return err
		}

		if err := userGroupRepoTx.AddUserToGroup(ctx, createOutput.User.ID, "users"); err != nil {
			return apperrors.Wrap(err, "add user to default group")
		}

		return nil
	}); err != nil {
		if apperrors.CodeIs(err, apperrors.CodeUniqueViolation) {
			return nil, apperrors.ErrUserAlreadyExists
		}
		return nil, apperrors.Wrap(err, "register tx")
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

	if err := s.userRepo.UpdatePassword(ctx, repository.UpdatePasswordInput{
		UUID:         in.UserUUID,
		PasswordHash: newHash,
	}); err != nil {
		return apperrors.Wrap(err, "update password")
	}

	return nil
}

// parseUUID is a helper to parse UUID strings used in auth context.
func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}
