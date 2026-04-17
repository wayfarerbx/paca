// Package middleware provides per-route HTTP middleware for authentication and
// authorization.
package middleware

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/paca/api/internal/apierr"
	domainauth "github.com/paca/api/internal/domain/auth"
	jwttoken "github.com/paca/api/internal/platform/token"
	"github.com/paca/api/internal/transport/http/presenter"
)

const claimsKey = "claims"

// actorContextKey is the unexported key used to store the authenticated user's
// UUID in the Go request context.
type actorContextKey struct{}

// Authn validates the access JWT and stores the parsed claims in the Gin
// context as well as the caller's user UUID in the Go request context so
// service-layer code can access it without depending on Gin.
// It first checks the access_token HttpOnly cookie, then falls back to the
// Authorization: Bearer header for API/CLI clients.
func Authn(tm *jwttoken.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := ""

		// 1. Try the access_token cookie (browser clients).
		if cookie, err := c.Cookie("access_token"); err == nil && cookie != "" {
			tokenStr = cookie
		}

		// 2. Fall back to Authorization: Bearer header (API/CLI clients).
		if tokenStr == "" {
			header := c.GetHeader("Authorization")
			if header != "" {
				parts := strings.SplitN(header, " ", 2)
				if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
					tokenStr = parts[1]
				}
			}
		}

		if tokenStr == "" {
			presenter.Error(c, apierr.New(apierr.CodeMissingToken, "missing authentication"))
			return
		}

		claims, err := tm.Verify(tokenStr)
		if err != nil {
			presenter.Error(c, apierr.New(apierr.CodeTokenInvalid, "invalid or expired token"))
			return
		}

		if claims.Kind != "access" {
			presenter.Error(c, apierr.New(apierr.CodeTokenInvalid, "expected access token"))
			return
		}

		c.Set(claimsKey, claims)

		// Embed the actor UUID in the Go request context so service layers can
		// read it without coupling to Gin.
		if actorID, parseErr := uuid.Parse(claims.Subject); parseErr == nil {
			ctx := context.WithValue(c.Request.Context(), actorContextKey{}, actorID)
			c.Request = c.Request.WithContext(ctx)
		}

		c.Next()
	}
}

// ClaimsFrom retrieves the authenticated claims from the Gin context.
// It returns nil if no claims are present (e.g. on unauthenticated routes).
func ClaimsFrom(c *gin.Context) *domainauth.Claims {
	v, exists := c.Get(claimsKey)
	if !exists {
		return nil
	}
	claims, _ := v.(*domainauth.Claims)
	return claims
}

// ClaimsContextKey returns the context key used to store JWT claims.
// Intended for use in tests that need to inject synthetic claims.
func ClaimsContextKey() string { return claimsKey }

// ActorIDFromContext extracts the authenticated user's UUID from a Go
// context.Context.  Returns (uuid.Nil, false) when no actor is set (e.g. in
// tests that don't go through the Authn middleware).
func ActorIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	v := ctx.Value(actorContextKey{})
	if v == nil {
		return uuid.Nil, false
	}
	id, ok := v.(uuid.UUID)
	return id, ok
}

// WithActorID returns a new context that carries actorID.
// Use in tests to simulate an authenticated caller.
func WithActorID(ctx context.Context, actorID uuid.UUID) context.Context {
	return context.WithValue(ctx, actorContextKey{}, actorID)
}
