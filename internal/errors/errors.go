package errors

import (
	"errors"
	"net/http"

	"github.com/alexbro4u/errkit"
)

const (
	CodeUserNotFound       = "USER_NOT_FOUND"
	CodeGroupNotFound      = "GROUP_NOT_FOUND"
	CodeInvalidCredentials = "INVALID_CREDENTIALS"
	CodeUserAlreadyExists  = "USER_ALREADY_EXISTS"
	CodeUniqueViolation    = "UNIQUE_VIOLATION"
	CodeValidationFailed   = "VALIDATION_FAILED"
	CodeInternal           = "INTERNAL"
)

var (
	ErrUserNotFound = errkit.New("user not found",
		errkit.Code(CodeUserNotFound),
		errkit.HTTP(http.StatusNotFound),
	)
	ErrGroupNotFound = errkit.New("group not found",
		errkit.Code(CodeGroupNotFound),
		errkit.HTTP(http.StatusNotFound),
	)
	ErrInvalidCredentials = errkit.New("invalid credentials",
		errkit.Code(CodeInvalidCredentials),
		errkit.HTTP(http.StatusUnauthorized),
	)
	ErrUserAlreadyExists = errkit.New("user already exists",
		errkit.Code(CodeUserAlreadyExists),
		errkit.HTTP(http.StatusConflict),
	)
	ErrUniqueViolation = errkit.New("unique constraint violation",
		errkit.Code(CodeUniqueViolation),
		errkit.HTTP(http.StatusConflict),
	)
	ErrRequestCacheNotFound = errkit.New("request cache not found",
		errkit.Code(CodeUserNotFound),
		errkit.HTTP(http.StatusNotFound),
	)
)

func Is(err, target error) bool {
	return errors.Is(err, target)
}

func CodeIs(err error, code string) bool {
	return errkit.CodeIs(err, code)
}

func Wrap(err error, msg string, opts ...errkit.Option) error {
	return errkit.Wrap(err, msg, opts...)
}

func New(msg string, opts ...errkit.Option) error {
	return errkit.New(msg, opts...)
}

func HTTPStatus(err error) int {
	return errkit.HTTPStatus(err)
}
