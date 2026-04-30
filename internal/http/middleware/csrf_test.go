package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupCSRFRouter(handler gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CSRF())
	r.POST("/test", handler)
	r.GET("/test", handler)
	r.POST("/api/auth/login", handler)

	return r
}

func TestRequiresCSRF(t *testing.T) {
	if !requiresCSRF(http.MethodPost) || !requiresCSRF(http.MethodPut) || !requiresCSRF(http.MethodPatch) || !requiresCSRF(http.MethodDelete) {
		t.Fatal("expected modifying methods to require csrf")
	}

	if requiresCSRF(http.MethodGet) || requiresCSRF(http.MethodHead) || requiresCSRF(http.MethodOptions) {
		t.Fatal("expected safe methods to skip csrf")
	}
}

func TestCSRFIgnoredPath(t *testing.T) {
	if !isCSRFIgnoredPath("/api/auth/login") {
		t.Fatal("expected login path ignored")
	}

	if !isCSRFIgnoredPath("/api/auth/register") {
		t.Fatal("expected register path ignored")
	}

	if isCSRFIgnoredPath("/api/challenges/1/submit") {
		t.Fatal("expected non-auth path not ignored")
	}
}

func TestCSRFMiddleware_SafeMethodPasses(t *testing.T) {
	called := false
	r := setupCSRFRouter(func(ctx *gin.Context) {
		called = true
		ctx.Status(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || !called {
		t.Fatalf("expected pass-through for safe method, status=%d called=%v", rec.Code, called)
	}
}

func TestCSRFMiddleware_IgnoredAuthPathPasses(t *testing.T) {
	called := false
	r := setupCSRFRouter(func(ctx *gin.Context) {
		called = true
		ctx.Status(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || !called {
		t.Fatalf("expected pass-through for ignored path, status=%d called=%v", rec.Code, called)
	}
}

func TestCSRFMiddleware_NoAuthCookiesPasses(t *testing.T) {
	called := false
	r := setupCSRFRouter(func(ctx *gin.Context) {
		called = true
		ctx.Status(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || !called {
		t.Fatalf("expected pass-through without auth cookies, status=%d called=%v", rec.Code, called)
	}
}

func TestCSRFMiddleware_WithAuthCookieMissingCSRFCookieForbidden(t *testing.T) {
	called := false
	r := setupCSRFRouter(func(ctx *gin.Context) {
		called = true
		ctx.Status(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "token"})
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}

	if called {
		t.Fatal("handler should not be called")
	}
}

func TestCSRFMiddleware_WithAuthCookieMissingHeaderForbidden(t *testing.T) {
	called := false
	r := setupCSRFRouter(func(ctx *gin.Context) {
		called = true
		ctx.Status(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "token"})
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "csrf-1"})
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}

	if called {
		t.Fatal("handler should not be called")
	}
}

func TestCSRFMiddleware_WithAuthCookieMismatchForbidden(t *testing.T) {
	called := false
	r := setupCSRFRouter(func(ctx *gin.Context) {
		called = true
		ctx.Status(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "token"})
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "csrf-1"})
	req.Header.Set(csrfHeaderName, "csrf-2")
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}

	if called {
		t.Fatal("handler should not be called")
	}
}

func TestCSRFMiddleware_WithAuthCookieAndMatchPasses(t *testing.T) {
	called := false
	r := setupCSRFRouter(func(ctx *gin.Context) {
		called = true
		ctx.Status(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "token"})
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "csrf-ok"})
	req.Header.Set(csrfHeaderName, " csrf-ok ")
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if !called {
		t.Fatal("handler should be called")
	}
}
