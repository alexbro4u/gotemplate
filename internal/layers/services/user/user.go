package user

import (
	"context"
	"log/slog"
	"strings"

	"github.com/alexbro4u/gotemplate/internal/dto/repository"
	"github.com/alexbro4u/gotemplate/internal/dto/service"
	"github.com/alexbro4u/gotemplate/internal/entity"
	apperrors "github.com/alexbro4u/gotemplate/internal/errors"
	"github.com/alexbro4u/gotemplate/internal/layers/repositories"
	"github.com/alexbro4u/gotemplate/pkg/password"
	"github.com/alexbro4u/gotemplate/pkg/sqlxadapter"
	"github.com/alexbro4u/uowkit/uow"

	"github.com/go-playground/validator/v10"
)

type Deps struct {
	Logger        *slog.Logger                     `validate:"required"`
	UoW           uow.UnitOfWork                   `validate:"required"`
	UserRepo      repositories.UserRepository      `validate:"required"`
	UserGroupRepo repositories.UserGroupRepository `validate:"required"`
	Validator     *validator.Validate              `validate:"required"`
	AuditRepo     repositories.AuditRepository
}

type Service struct {
	logger        *slog.Logger
	uow           uow.UnitOfWork
	userRepo      repositories.UserRepository
	userGroupRepo repositories.UserGroupRepository
	validator     *validator.Validate
	auditRepo     repositories.AuditRepository
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
		validator:     deps.Validator,
		auditRepo:     deps.AuditRepo,
	}, nil
}

func (s *Service) Create(ctx context.Context, in service.CreateUserInput) (*service.CreateUserOutput, error) {
	if err := s.validator.Struct(in); err != nil {
		return nil, apperrors.Wrap(err, "validate input")
	}

	in.Email = strings.ToLower(strings.TrimSpace(in.Email))

	passwordHash, err := password.Hash(in.Password)
	if err != nil {
		return nil, apperrors.Wrap(err, "hash password")
	}

	var output *repository.CreateUserOutput

	if txErr := s.uow.Do(ctx, func(tx uow.Tx) error {
		sqlxTx, ok := sqlxadapter.Unwrap(tx)
		if !ok {
			return apperrors.New("unexpected tx type")
		}

		userRepoTx := s.userRepo.WithExecutor(sqlxTx)
		userGroupRepoTx := s.userGroupRepo.WithExecutor(sqlxTx)

		var createErr error
		output, createErr = userRepoTx.Create(ctx, repository.CreateUserInput{
			Email:        in.Email,
			Name:         in.Name,
			PasswordHash: passwordHash,
			Role:         nil,
		})
		if createErr != nil {
			return createErr
		}

		if groupErr := userGroupRepoTx.AddUserToGroup(ctx, output.User.ID, "users"); groupErr != nil {
			return apperrors.Wrap(groupErr, "add user to default group")
		}

		return nil
	}); txErr != nil {
		if apperrors.CodeIs(txErr, apperrors.CodeUniqueViolation) {
			return nil, apperrors.ErrUserAlreadyExists
		}
		return nil, apperrors.Wrap(txErr, "create user tx")
	}

	result := &service.CreateUserOutput{
		User: service.UserToDTO(output.User),
	}

	if s.auditRepo != nil {
		_ = s.auditRepo.Log(ctx, repository.LogAuditInput{
			EntityType: "user",
			EntityID:   output.User.UUID.String(),
			ActorUUID:  output.User.UUID,
			Action:     entity.AuditActionCreate,
		})
	}

	return result, nil
}

func (s *Service) Get(ctx context.Context, in service.GetUserInput) (*service.GetUserOutput, error) {
	output, err := s.userRepo.Get(ctx, repository.GetUserInput{
		UUID: in.UUID,
	})
	if err != nil {
		if apperrors.CodeIs(err, apperrors.CodeUserNotFound) {
			return nil, apperrors.ErrUserNotFound
		}
		return nil, apperrors.Wrap(err, "get user")
	}

	return &service.GetUserOutput{
		User: service.UserToDTO(output.User),
	}, nil
}

func (s *Service) Update(ctx context.Context, in service.UpdateUserInput) error {
	if err := s.validator.Struct(in); err != nil {
		return apperrors.Wrap(err, "validate input")
	}

	err := s.userRepo.Update(ctx, repository.UpdateUserInput{
		UUID:  in.UUID,
		Email: in.Email,
		Name:  in.Name,
	})
	if err != nil {
		if apperrors.CodeIs(err, apperrors.CodeUserNotFound) {
			return apperrors.ErrUserNotFound
		}
		return apperrors.Wrap(err, "update user")
	}

	if s.auditRepo != nil {
		_ = s.auditRepo.Log(ctx, repository.LogAuditInput{
			EntityType: "user",
			EntityID:   in.UUID.String(),
			ActorUUID:  in.UUID,
			Action:     entity.AuditActionUpdate,
		})
	}

	return nil
}

func (s *Service) Delete(ctx context.Context, in service.DeleteUserInput) error {
	err := s.userRepo.Delete(ctx, repository.DeleteUserInput{
		UUID: in.UUID,
	})
	if err != nil {
		if apperrors.CodeIs(err, apperrors.CodeUserNotFound) {
			return apperrors.ErrUserNotFound
		}
		return apperrors.Wrap(err, "delete user")
	}

	if s.auditRepo != nil {
		_ = s.auditRepo.Log(ctx, repository.LogAuditInput{
			EntityType: "user",
			EntityID:   in.UUID.String(),
			ActorUUID:  in.UUID,
			Action:     entity.AuditActionDelete,
		})
	}

	return nil
}

func (s *Service) List(ctx context.Context, in service.ListUsersInput) (*service.ListUsersOutput, error) {
	if err := s.validator.Struct(in); err != nil {
		return nil, apperrors.Wrap(err, "validate input")
	}

	output, err := s.userRepo.List(ctx, in.Limit, in.Offset)
	if err != nil {
		return nil, apperrors.Wrap(err, "list users")
	}

	userDTOs := make([]service.UserDTO, len(output.Users))
	for i, user := range output.Users {
		userDTOs[i] = service.UserToDTO(user)
	}

	return &service.ListUsersOutput{
		Users:  userDTOs,
		Total:  output.Total,
		Limit:  in.Limit,
		Offset: in.Offset,
	}, nil
}
