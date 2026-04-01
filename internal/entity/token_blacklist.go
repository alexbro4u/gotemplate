package entity

import "time"

type TokenBlacklist struct {
	JTI       string    `db:"jti"`
	ExpiresAt time.Time `db:"expires_at"`
}
