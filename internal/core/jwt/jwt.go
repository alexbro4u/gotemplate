package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	DefaultAccessExpiration  = 15 * time.Minute
	DefaultRefreshExpiration = 7 * 24 * time.Hour
)

type Claims struct {
	UserUUID  uuid.UUID `json:"user_uuid"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Groups    []string  `json:"groups"`
	TokenType string    `json:"token_type"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

type Config struct {
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

type Service struct {
	secretKey  []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func New(secretKey string, opts ...Config) *Service {
	accessTTL := DefaultAccessExpiration
	refreshTTL := DefaultRefreshExpiration
	if len(opts) > 0 {
		if opts[0].AccessTTL > 0 {
			accessTTL = opts[0].AccessTTL
		}
		if opts[0].RefreshTTL > 0 {
			refreshTTL = opts[0].RefreshTTL
		}
	}
	return &Service{
		secretKey:  []byte(secretKey),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

func (s *Service) GenerateToken(userUUID uuid.UUID, email, role string, groups []string) (string, error) {
	return s.generateToken(userUUID, email, role, groups, "access", s.accessTTL)
}

func (s *Service) GenerateRefreshToken(userUUID uuid.UUID, email, role string, groups []string) (string, error) {
	return s.generateToken(userUUID, email, role, groups, "refresh", s.refreshTTL)
}

func (s *Service) GenerateTokenPair(userUUID uuid.UUID, email, role string, groups []string) (*TokenPair, error) {
	access, err := s.GenerateToken(userUUID, email, role, groups)
	if err != nil {
		return nil, err
	}
	refresh, err := s.GenerateRefreshToken(userUUID, email, role, groups)
	if err != nil {
		return nil, err
	}
	return &TokenPair{AccessToken: access, RefreshToken: refresh}, nil
}

func (s *Service) generateToken(userUUID uuid.UUID, email, role string, groups []string, tokenType string, ttl time.Duration) (string, error) {
	now := time.Now()
	if groups == nil {
		groups = []string{}
	}
	claims := Claims{
		UserUUID:  userUUID,
		Email:     email,
		Role:      role,
		Groups:    groups,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.secretKey)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return tokenString, nil
}

func (s *Service) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}
	if !token.Valid {
		return nil, errors.New("token invalid")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errors.New("invalid token claims type")
	}

	now := time.Now()
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(now) {
		return nil, errors.New("token expired")
	}
	if claims.NotBefore != nil && claims.NotBefore.Time.After(now) {
		return nil, errors.New("token not yet valid")
	}
	if claims.IssuedAt != nil && claims.IssuedAt.Time.After(now) {
		return nil, errors.New("token issued in the future")
	}

	return claims, nil
}

func (s *Service) ValidateRefreshToken(tokenString string) (*Claims, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != "refresh" {
		return nil, errors.New("not a refresh token")
	}
	return claims, nil
}
