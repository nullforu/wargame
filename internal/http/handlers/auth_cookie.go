package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"wargame/internal/config"

	"github.com/gin-gonic/gin"
)

const (
	accessTokenCookieName  = "access_token"
	refreshTokenCookieName = "refresh_token"
	csrfTokenCookieName    = "csrf_token"
	csrfTokenHeaderName    = "X-CSRF-Token"
)

func setAuthCookies(ctx *gin.Context, cfg config.Config, accessToken, refreshToken string) {
	setCookie(ctx, cfg, accessTokenCookieName, accessToken, int(cfg.JWT.AccessTTL.Seconds()), true)
	setCookie(ctx, cfg, refreshTokenCookieName, refreshToken, int(cfg.JWT.RefreshTTL.Seconds()), true)

	csrfToken, ok := currentCSRFToken(ctx)
	if !ok {
		csrfToken = randomTokenHex(32)
	}

	maxAge := max(int(cfg.JWT.RefreshTTL.Seconds()), int(cfg.JWT.AccessTTL.Seconds()))

	setCookie(ctx, cfg, csrfTokenCookieName, csrfToken, maxAge, false)
	ctx.Header(csrfTokenHeaderName, csrfToken)
	ctx.Header("Cache-Control", "no-store")
	ctx.Header("Pragma", "no-cache")
}

func clearAuthCookies(ctx *gin.Context, cfg config.Config) {
	setCookie(ctx, cfg, accessTokenCookieName, "", -1, true)
	setCookie(ctx, cfg, refreshTokenCookieName, "", -1, true)
	setCookie(ctx, cfg, csrfTokenCookieName, "", -1, false)
	ctx.Header("Cache-Control", "no-store")
	ctx.Header("Pragma", "no-cache")
}

func currentRefreshToken(ctx *gin.Context) (string, bool) {
	v, err := ctx.Cookie(refreshTokenCookieName)
	if err != nil || v == "" {
		return "", false
	}

	return v, true
}

func currentCSRFToken(ctx *gin.Context) (string, bool) {
	v, err := ctx.Cookie(csrfTokenCookieName)
	if err != nil || v == "" {
		return "", false
	}

	return v, true
}

func setCookie(ctx *gin.Context, cfg config.Config, name, value string, maxAge int, httpOnly bool) {
	secure := cfg.AppEnv == "production"
	sameSite := http.SameSiteLaxMode
	ctx.SetSameSite(sameSite)
	ctx.SetCookie(name, value, maxAge, "/", "", secure, httpOnly)
}

func randomTokenHex(bytesLen int) string {
	b := make([]byte, bytesLen)
	if _, err := rand.Read(b); err != nil {
		return "fallback-csrf-token"
	}

	return hex.EncodeToString(b)
}
