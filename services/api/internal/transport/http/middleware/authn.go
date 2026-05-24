// Package middleware provides per-route HTTP middleware for authentication and
// authorization.
package middleware

import (
	"context"
	"errors"
	"strings"

	"github.com/Paca-AI/api/internal/apierr"
	apikeydom "github.com/Paca-AI/api/internal/domain/apikey"
	domainauth "github.com/Paca-AI/api/internal/domain/auth"
	jwttoken "github.com/Paca-AI/api/internal/platform/token"
	"github.com/Paca-AI/api/internal/transport/http/presenter"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const claimsKey = "claims"
const authMethodKey = "auth_method"

// actorContextKey is the unexported key used to store the authenticated user's
// UUID in the Go request context.
type actorContextKey struct{}

// agentContextKey is the unexported key used to store the agent ID when
// authenticating with an agent API key.
type agentContextKey struct{}

// APIKeyAuthenticator validates a raw API key string and returns the key record.
// It is satisfied by apikeysvc.Service.
type APIKeyAuthenticator interface {
	Authenticate(ctx context.Context, rawKey string) (*apikeydom.APIKey, error)
}

// AgentAPIKeyAuthenticator extends APIKeyAuthenticator with the ability to check
// if a key is the static agent API key.
type AgentAPIKeyAuthenticator interface {
	APIKeyAuthenticator
	IsAgentKey(ctx context.Context, rawKey string) bool
}

// Authn validates the access JWT and stores the parsed claims in the Gin
// context as well as the caller's user UUID in the Go request context so
// service-layer code can access it without depending on Gin.
// It first checks the access_token HttpOnly cookie, then falls back to the
// Authorization: Bearer header for API/CLI clients, and finally accepts
// Authorization: ApiKey or X-API-Key headers for API key authentication.
func Authn(tm *jwttoken.Manager, apiKeyAuth ...APIKeyAuthenticator) gin.HandlerFunc {
	var apiKeyAuthenticator APIKeyAuthenticator
	if len(apiKeyAuth) > 0 {
		apiKeyAuthenticator = apiKeyAuth[0]
	}
	return func(c *gin.Context) {
		if !EnforceAuthn(c, tm, apiKeyAuthenticator) {
			return
		}
		c.Next()
	}
}

// OptionalAuthn tries to authenticate the request using the same credential
// sources as Authn (cookie → Bearer → API key), but does NOT abort if no
// credentials are present.  Downstream handlers must check ClaimsFrom for nil
// to determine whether the caller is authenticated.
func OptionalAuthn(tm *jwttoken.Manager, apiKeyAuth ...APIKeyAuthenticator) gin.HandlerFunc {
	var apiKeyAuthenticator APIKeyAuthenticator
	if len(apiKeyAuth) > 0 {
		apiKeyAuthenticator = apiKeyAuth[0]
	}
	return func(c *gin.Context) {
		if !EnforceOptionalAuthn(c, tm, apiKeyAuthenticator) {
			return
		}
		c.Next()
	}
}

// EnforceAuthn validates credentials and sets auth context without advancing the Gin handler chain.
func EnforceAuthn(c *gin.Context, tm *jwttoken.Manager, apiKeyAuth ...APIKeyAuthenticator) bool {
	var apiKeyAuthenticator APIKeyAuthenticator
	if len(apiKeyAuth) > 0 {
		apiKeyAuthenticator = apiKeyAuth[0]
	}
	return applyAuthn(c, tm, apiKeyAuthenticator, false)
}

// EnforceOptionalAuthn validates optional credentials and sets auth context without advancing the Gin handler chain.
func EnforceOptionalAuthn(c *gin.Context, tm *jwttoken.Manager, apiKeyAuth ...APIKeyAuthenticator) bool {
	var apiKeyAuthenticator APIKeyAuthenticator
	if len(apiKeyAuth) > 0 {
		apiKeyAuthenticator = apiKeyAuth[0]
	}
	return applyAuthn(c, tm, apiKeyAuthenticator, true)
}

func applyAuthn(c *gin.Context, tm *jwttoken.Manager, apiKeyAuthenticator APIKeyAuthenticator, optional bool) bool {
	tokenStr := ""
	isAPIKey := false

	if cookie, err := c.Cookie("access_token"); err == nil && cookie != "" {
		tokenStr = cookie
	}
	if tokenStr == "" {
		header := c.GetHeader("Authorization")
		if header != "" {
			parts := strings.SplitN(header, " ", 2)
			if len(parts) == 2 {
				switch strings.ToLower(parts[0]) {
				case "bearer":
					tokenStr = parts[1]
				case "apikey":
					tokenStr = parts[1]
					isAPIKey = true
				}
			}
		}
	}
	if tokenStr == "" {
		if v := c.GetHeader("X-API-Key"); v != "" {
			tokenStr = v
			isAPIKey = true
		}
	}

	if tokenStr == "" {
		if optional {
			return true
		}
		presenter.Error(c, apierr.New(apierr.CodeMissingToken, "missing authentication"))
		return false
	}

	if isAPIKey {
		if optional {
			if apiKeyAuthenticator != nil {
				key, err := apiKeyAuthenticator.Authenticate(c.Request.Context(), tokenStr)
				if err == nil {
					var agentID uuid.UUID
					// Only read X-Agent-ID header when using the agent API key
					if agentKeyAuth, ok := apiKeyAuthenticator.(AgentAPIKeyAuthenticator); ok && agentKeyAuth.IsAgentKey(c.Request.Context(), tokenStr) {
						agentIDHeader := c.GetHeader("X-Agent-ID")
						if agentIDHeader != "" {
							if parsedID, parseErr := uuid.Parse(agentIDHeader); parseErr == nil {
								agentID = parsedID
							}
						}
					}
					setAPIKeyAuthContext(c, key.UserID, agentID)
				}
			}
			return true
		}
		if apiKeyAuthenticator == nil {
			presenter.Error(c, apierr.New(apierr.CodeUnauthenticated, "API key authentication not configured"))
			return false
		}
		key, err := apiKeyAuthenticator.Authenticate(c.Request.Context(), tokenStr)
		if err != nil {
			switch {
			case errors.Is(err, apikeydom.ErrRevoked):
				presenter.Error(c, apierr.New(apierr.CodeAPIKeyRevoked, "API key has been revoked"))
			case errors.Is(err, apikeydom.ErrExpired):
				presenter.Error(c, apierr.New(apierr.CodeAPIKeyExpired, "API key has expired"))
			default:
				presenter.Error(c, apierr.New(apierr.CodeTokenInvalid, "invalid or expired API key"))
			}
			return false
		}

		var agentID uuid.UUID
		// Only read X-Agent-ID header when using the agent API key
		if agentKeyAuth, ok := apiKeyAuthenticator.(AgentAPIKeyAuthenticator); ok && agentKeyAuth.IsAgentKey(c.Request.Context(), tokenStr) {
			agentIDHeader := c.GetHeader("X-Agent-ID")
			if agentIDHeader != "" {
				if parsedID, parseErr := uuid.Parse(agentIDHeader); parseErr == nil {
					agentID = parsedID
				}
			}
		}

		setAPIKeyAuthContext(c, key.UserID, agentID)
		return true
	}

	claims, err := tm.Verify(tokenStr)
	if err != nil {
		presenter.Error(c, apierr.New(apierr.CodeTokenInvalid, "invalid or expired token"))
		return false
	}
	if claims.Kind != "access" {
		presenter.Error(c, apierr.New(apierr.CodeTokenInvalid, "expected access token"))
		return false
	}

	c.Set(claimsKey, claims)
	if actorID, parseErr := uuid.Parse(claims.Subject); parseErr == nil {
		ctx := context.WithValue(c.Request.Context(), actorContextKey{}, actorID)
		c.Request = c.Request.WithContext(ctx)
	}

	return true
}

func setAPIKeyAuthContext(c *gin.Context, userID uuid.UUID, agentID uuid.UUID) {
	syntheticClaims := &domainauth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: userID.String(),
		},
		Kind: "access",
	}
	if agentID != uuid.Nil {
		agentIDStr := agentID.String()
		syntheticClaims.AgentID = &agentIDStr
	}
	c.Set(claimsKey, syntheticClaims)
	c.Set(authMethodKey, "apikey")
	ctx := context.WithValue(c.Request.Context(), actorContextKey{}, userID)
	if agentID != uuid.Nil {
		ctx = context.WithValue(ctx, agentContextKey{}, agentID)
	}
	c.Request = c.Request.WithContext(ctx)
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

// AgentIDFromContext extracts the agent ID from a Go context.Context when
// authenticated via an agent API key with X-Agent-ID header.
// Returns (uuid.Nil, false) when no agent ID is set.
func AgentIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	v := ctx.Value(agentContextKey{})
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

// WithAgentID returns a new context that carries agentID.
// Use in tests to simulate an agent-authenticated caller.
func WithAgentID(ctx context.Context, agentID uuid.UUID) context.Context {
	return context.WithValue(ctx, agentContextKey{}, agentID)
}

// IsAPIKeyAuth reports whether the current request was authenticated via an API
// key rather than a JWT/cookie session.
func IsAPIKeyAuth(c *gin.Context) bool {
	v, exists := c.Get(authMethodKey)
	if !exists {
		return false
	}
	return v == "apikey"
}

// AgentIDFromGinContext extracts the agent ID from the Gin context when
// authenticated via an agent API key with X-Agent-ID header.
// Returns (uuid.Nil, false) when no agent ID is set.
func AgentIDFromGinContext(c *gin.Context) (uuid.UUID, bool) {
	return AgentIDFromContext(c.Request.Context())
}

// RequireJWTAuth rejects requests that were authenticated via an API key.
// Apply this middleware to sensitive routes (e.g. API key management) that must
// only be reachable through a JWT/cookie session to prevent privilege escalation
// via a leaked API key.
func RequireJWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !EnforceJWTAuth(c) {
			return
		}
		c.Next()
	}
}

// EnforceJWTAuth rejects API key-authenticated requests without advancing the Gin handler chain.
func EnforceJWTAuth(c *gin.Context) bool {
	if IsAPIKeyAuth(c) {
		presenter.Error(c, apierr.New(apierr.CodeForbidden, "this endpoint requires session authentication and does not accept API key credentials"))
		return false
	}
	return true
}
