package security

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/example/gue/backend/config"
	"github.com/gin-gonic/gin"
)

type CookieManager struct {
	cfg        config.CookieConfig
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewCookieManager(cfg config.CookieConfig, accessTTL time.Duration, refreshTTL time.Duration) *CookieManager {
	return &CookieManager{
		cfg:        cfg,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

func (m *CookieManager) SetAuthCookies(c *gin.Context, accessToken string, refreshToken string) {
	m.setCookie(c, m.cfg.AccessTokenName, accessToken, int(m.accessTTL.Seconds()), m.cfg.HTTPOnly)
	m.setCookie(c, m.cfg.RefreshTokenName, refreshToken, int(m.refreshTTL.Seconds()), m.cfg.HTTPOnly)
}

func (m *CookieManager) ClearAuthCookies(c *gin.Context) {
	m.setCookie(c, m.cfg.AccessTokenName, "", -1, m.cfg.HTTPOnly)
	m.setCookie(c, m.cfg.RefreshTokenName, "", -1, m.cfg.HTTPOnly)
}

func (m *CookieManager) EnsureCSRFCookie(c *gin.Context) (string, error) {
	if token, err := c.Cookie(m.cfg.CSRFCookieName); err == nil && strings.TrimSpace(token) != "" {
		return token, nil
	}

	token, err := generateToken(32)
	if err != nil {
		return "", fmt.Errorf("generate csrf token: %w", err)
	}
	m.setCookie(c, m.cfg.CSRFCookieName, token, int((24 * time.Hour).Seconds()), false)
	return token, nil
}

func (m *CookieManager) ClearCSRFCookie(c *gin.Context) {
	m.setCookie(c, m.cfg.CSRFCookieName, "", -1, false)
}

func (m *CookieManager) ReadAccessToken(c *gin.Context) (string, error) {
	return c.Cookie(m.cfg.AccessTokenName)
}

func (m *CookieManager) ReadRefreshToken(c *gin.Context) (string, error) {
	return c.Cookie(m.cfg.RefreshTokenName)
}

func (m *CookieManager) CSRFTokenFromCookie(c *gin.Context) (string, error) {
	return c.Cookie(m.cfg.CSRFCookieName)
}

func (m *CookieManager) AccessCookieName() string {
	return m.cfg.AccessTokenName
}

func (m *CookieManager) CSRFCookieName() string {
	return m.cfg.CSRFCookieName
}

func (m *CookieManager) setCookie(c *gin.Context, name string, value string, maxAge int, httpOnly bool) {
	c.SetSameSite(parseSameSite(m.cfg.SameSite))
	c.SetCookie(name, value, maxAge, m.cfg.Path, m.cfg.Domain, m.cfg.Secure, httpOnly)
}

func parseSameSite(value string) http.SameSite {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "none":
		return http.SameSiteNoneMode
	case "lax":
		return http.SameSiteLaxMode
	default:
		return http.SameSiteStrictMode
	}
}

func generateToken(bytesLen int) (string, error) {
	raw := make([]byte, bytesLen)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
