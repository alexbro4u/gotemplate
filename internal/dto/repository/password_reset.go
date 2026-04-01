package repository

import "time"

type CreatePasswordResetInput struct {
	Token     string
	UserID    int64
	ExpiresAt time.Time
}

type GetPasswordResetInput struct {
	Token string
}

type GetPasswordResetOutput struct {
	UserID    int64
	ExpiresAt time.Time
	Used      bool
}

type MarkPasswordResetUsedInput struct {
	Token string
}
