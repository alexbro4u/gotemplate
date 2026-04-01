package entity

import "time"

type PasswordResetToken struct {
	Token     string    `db:"token"`
	UserID    int64     `db:"user_id"`
	ExpiresAt time.Time `db:"expires_at"`
	Used      bool      `db:"used"`
	CreatedAt time.Time `db:"created_at"`
}
