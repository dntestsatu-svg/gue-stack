package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Manager struct {
	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
	issuer        string
	audience      string
}

type Claims struct {
	UserID  uint64 `json:"user_id"`
	Email   string `json:"email"`
	TokenID string `json:"token_id,omitempty"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	RefreshID    string `json:"-"`
	ExpiresIn    int64  `json:"expires_in"`
}

func NewManager(accessSecret, refreshSecret string, accessTTL, refreshTTL time.Duration, issuer, audience string) *Manager {
	return &Manager{
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
		issuer:        issuer,
		audience:      audience,
	}
}

func (m *Manager) GenerateTokenPair(userID uint64, email string, now time.Time) (*TokenPair, error) {
	refreshID := uuid.NewString()

	accessClaims := &Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Audience:  []string{m.audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTTL)),
			Subject:   fmt.Sprintf("%d", userID),
		},
	}

	refreshClaims := &Claims{
		UserID:  userID,
		Email:   email,
		TokenID: refreshID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Audience:  []string{m.audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.refreshTTL)),
			Subject:   fmt.Sprintf("%d", userID),
			ID:        refreshID,
		},
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString(m.accessSecret)
	if err != nil {
		return nil, err
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString(m.refreshSecret)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		RefreshID:    refreshID,
		ExpiresIn:    int64(m.accessTTL.Seconds()),
	}, nil
}

func (m *Manager) ParseAccessToken(token string) (*Claims, error) {
	return m.parse(token, m.accessSecret)
}

func (m *Manager) ParseRefreshToken(token string) (*Claims, error) {
	return m.parse(token, m.refreshSecret)
}

func (m *Manager) RefreshTTL() time.Duration {
	return m.refreshTTL
}

func (m *Manager) parse(token string, secret []byte) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	return claims, nil
}
