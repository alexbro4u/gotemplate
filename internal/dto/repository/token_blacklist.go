package repository

import "time"

type AddToBlacklistInput struct {
	JTI       string
	ExpiresAt time.Time
}
