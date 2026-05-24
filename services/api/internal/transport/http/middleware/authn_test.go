package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	apikeydom "github.com/Paca-AI/api/internal/domain/apikey"
	jwttoken "github.com/Paca-AI/api/internal/platform/token"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// stubAPIKeyAuth is a minimal APIKeyAuthenticator for unit tests.
type stubAPIKeyAuth struct {
	key        *apikeydom.APIKey
	err        error
	isAgentKey bool
}

func (s *stubAPIKeyAuth) Authenticate(_ context.Context, _ string) (*apikeydom.APIKey, error) {
	return s.key, s.err
}

func (s *stubAPIKeyAuth) IsAgentKey(_ context.Context, _ string) bool {
	return s.isAgentKey
}

func newTestTokenManager() *jwttoken.Manager {
	return jwttoken.New("test-secret", 15*time.Minute, 24*time.Hour)
}

func TestAuthn_MissingToken(t *testing.T) {
	r := gin.New()
	r.GET("/protected", Authn(newTestTokenManager()), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}

	var env struct {
		ErrorCode string `json:"error_code"`
	}
	if err := json.NewDecoder(w.Body).Decode(&env); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if env.ErrorCode != "AUTH_MISSING_TOKEN" {
		t.Fatalf("expected AUTH_MISSING_TOKEN, got %q", env.ErrorCode)
	}
}

func TestAuthn_InvalidToken(t *testing.T) {
	r := gin.New()
	r.GET("/protected", Authn(newTestTokenManager()), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer not-a-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthn_ValidAccessTokenInHeader(t *testing.T) {
	tm := newTestTokenManager()
	at, err := tm.IssueAccess("user-id", "alice", "USER", "fam", false)
	if err != nil {
		t.Fatalf("issue access token: %v", err)
	}

	r := gin.New()
	r.GET("/protected", Authn(tm), func(c *gin.Context) {
		claims := ClaimsFrom(c)
		if claims == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "claims missing"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"username": claims.Username})
	})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+at)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}
}

func TestAuthn_RefreshTokenRejected(t *testing.T) {
	tm := newTestTokenManager()
	rt, err := tm.IssueRefresh("user-id", "alice", "USER", "fam")
	if err != nil {
		t.Fatalf("issue refresh token: %v", err)
	}

	r := gin.New()
	r.GET("/protected", Authn(tm), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+rt)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestClaimsFrom_Missing(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	if claims := ClaimsFrom(c); claims != nil {
		t.Fatal("expected nil claims when absent")
	}
}

func TestAuthn_APIKey_AuthorizationHeader(t *testing.T) {
	userID := uuid.New()
	stub := &stubAPIKeyAuth{key: &apikeydom.APIKey{ID: uuid.New(), UserID: userID}}

	r := gin.New()
	r.GET("/protected", Authn(newTestTokenManager(), stub), func(c *gin.Context) {
		if !IsAPIKeyAuth(c) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "expected API key auth"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "ApiKey test-api-key")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}
}

func TestAuthn_APIKey_XAPIKeyHeader(t *testing.T) {
	userID := uuid.New()
	stub := &stubAPIKeyAuth{key: &apikeydom.APIKey{ID: uuid.New(), UserID: userID}}

	r := gin.New()
	r.GET("/protected", Authn(newTestTokenManager(), stub), func(c *gin.Context) {
		if !IsAPIKeyAuth(c) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "expected API key auth"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/protected", nil)
	req.Header.Set("X-API-Key", "test-api-key")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}
}

func TestAuthn_APIKey_InvalidKey(t *testing.T) {
	stub := &stubAPIKeyAuth{err: errors.New("bad key")}

	r := gin.New()
	r.GET("/protected", Authn(newTestTokenManager(), stub), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/protected", nil)
	req.Header.Set("X-API-Key", "invalid-key")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthn_APIKey_RevokedKey(t *testing.T) {
	stub := &stubAPIKeyAuth{err: apikeydom.ErrRevoked}

	r := gin.New()
	r.GET("/protected", Authn(newTestTokenManager(), stub), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/protected", nil)
	req.Header.Set("X-API-Key", "revoked-key")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	var env struct {
		ErrorCode string `json:"error_code"`
	}
	if err := json.NewDecoder(w.Body).Decode(&env); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if env.ErrorCode != "API_KEY_REVOKED" {
		t.Fatalf("expected API_KEY_REVOKED, got %q", env.ErrorCode)
	}
}

func TestAuthn_APIKey_ExpiredKey(t *testing.T) {
	stub := &stubAPIKeyAuth{err: apikeydom.ErrExpired}

	r := gin.New()
	r.GET("/protected", Authn(newTestTokenManager(), stub), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/protected", nil)
	req.Header.Set("X-API-Key", "expired-key")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	var env struct {
		ErrorCode string `json:"error_code"`
	}
	if err := json.NewDecoder(w.Body).Decode(&env); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if env.ErrorCode != "API_KEY_EXPIRED" {
		t.Fatalf("expected API_KEY_EXPIRED, got %q", env.ErrorCode)
	}
}

func TestAuthn_APIKey_NotConfigured(t *testing.T) {
	// No API key authenticator passed — should reject ApiKey header.
	r := gin.New()
	r.GET("/protected", Authn(newTestTokenManager()), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "ApiKey some-key")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRequireJWTAuth_BlocksAPIKey(t *testing.T) {
	userID := uuid.New()
	stub := &stubAPIKeyAuth{key: &apikeydom.APIKey{ID: uuid.New(), UserID: userID}}

	r := gin.New()
	r.GET("/sensitive", Authn(newTestTokenManager(), stub), RequireJWTAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/sensitive", nil)
	req.Header.Set("X-API-Key", "some-key")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
	var env struct {
		ErrorCode string `json:"error_code"`
	}
	if err := json.NewDecoder(w.Body).Decode(&env); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if env.ErrorCode != "FORBIDDEN" {
		t.Fatalf("expected FORBIDDEN error code, got %q", env.ErrorCode)
	}
}

func TestRequireJWTAuth_AllowsJWT(t *testing.T) {
	tm := newTestTokenManager()
	at, err := tm.IssueAccess("user-id", "alice", "USER", "fam", false)
	if err != nil {
		t.Fatalf("issue access token: %v", err)
	}

	r := gin.New()
	r.GET("/sensitive", Authn(tm), RequireJWTAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/sensitive", nil)
	req.Header.Set("Authorization", "Bearer "+at)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}
}

func TestAuthn_APIKey_WithValidAgentID(t *testing.T) {
	userID := uuid.New()
	agentID := uuid.New()
	stub := &stubAPIKeyAuth{key: &apikeydom.APIKey{ID: uuid.New(), UserID: userID}, isAgentKey: true}

	r := gin.New()
	r.GET("/protected", Authn(newTestTokenManager(), stub), func(c *gin.Context) {
		if !IsAPIKeyAuth(c) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "expected API key auth"})
			return
		}

		retrievedAgentID, ok := AgentIDFromGinContext(c)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "agent ID not found in context"})
			return
		}

		if retrievedAgentID != agentID {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "agent ID mismatch"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/protected", nil)
	req.Header.Set("X-API-Key", "test-api-key")
	req.Header.Set("X-Agent-ID", agentID.String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}
}

func TestAuthn_APIKey_UserKeyCannotFakeAgentID(t *testing.T) {
	userID := uuid.New()
	agentID := uuid.New()
	stub := &stubAPIKeyAuth{key: &apikeydom.APIKey{ID: uuid.New(), UserID: userID}, isAgentKey: false}

	r := gin.New()
	r.GET("/protected", Authn(newTestTokenManager(), stub), func(c *gin.Context) {
		if !IsAPIKeyAuth(c) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "expected API key auth"})
			return
		}

		retrievedAgentID, ok := AgentIDFromGinContext(c)
		if ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "agent ID should not be set for user API key"})
			return
		}

		if retrievedAgentID != uuid.Nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "agent ID should be Nil"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/protected", nil)
	req.Header.Set("X-API-Key", "test-api-key")
	req.Header.Set("X-Agent-ID", agentID.String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}
}

func TestAuthn_APIKey_WithInvalidAgentID(t *testing.T) {
	userID := uuid.New()
	stub := &stubAPIKeyAuth{key: &apikeydom.APIKey{ID: uuid.New(), UserID: userID}, isAgentKey: true}

	r := gin.New()
	r.GET("/protected", Authn(newTestTokenManager(), stub), func(c *gin.Context) {
		if !IsAPIKeyAuth(c) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "expected API key auth"})
			return
		}

		retrievedAgentID, ok := AgentIDFromGinContext(c)
		if ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "agent ID should not be found with invalid UUID"})
			return
		}

		if retrievedAgentID != uuid.Nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "agent ID should be Nil"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/protected", nil)
	req.Header.Set("X-API-Key", "test-api-key")
	req.Header.Set("X-Agent-ID", "not-a-valid-uuid")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}
}

func TestAuthn_APIKey_WithoutAgentID(t *testing.T) {
	userID := uuid.New()
	stub := &stubAPIKeyAuth{key: &apikeydom.APIKey{ID: uuid.New(), UserID: userID}, isAgentKey: true}

	r := gin.New()
	r.GET("/protected", Authn(newTestTokenManager(), stub), func(c *gin.Context) {
		if !IsAPIKeyAuth(c) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "expected API key auth"})
			return
		}

		retrievedAgentID, ok := AgentIDFromGinContext(c)
		if ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "agent ID should not be found when header is absent"})
			return
		}

		if retrievedAgentID != uuid.Nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "agent ID should be Nil"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/protected", nil)
	req.Header.Set("X-API-Key", "test-api-key")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}
}

func TestAgentIDFromContext(t *testing.T) {
	ctx := context.Background()

	agentID, ok := AgentIDFromContext(ctx)
	if ok || agentID != uuid.Nil {
		t.Fatalf("expected no agent ID in empty context")
	}

	testAgentID := uuid.New()
	ctx = WithAgentID(ctx, testAgentID)

	retrievedAgentID, ok := AgentIDFromContext(ctx)
	if !ok {
		t.Fatalf("expected agent ID to be found")
	}
	if retrievedAgentID != testAgentID {
		t.Fatalf("expected agent ID %v, got %v", testAgentID, retrievedAgentID)
	}
}

func TestActorIDFromContext(t *testing.T) {
	ctx := context.Background()

	actorID, ok := ActorIDFromContext(ctx)
	if ok || actorID != uuid.Nil {
		t.Fatalf("expected no actor ID in empty context")
	}

	testActorID := uuid.New()
	ctx = WithActorID(ctx, testActorID)

	retrievedActorID, ok := ActorIDFromContext(ctx)
	if !ok {
		t.Fatalf("expected actor ID to be found")
	}
	if retrievedActorID != testActorID {
		t.Fatalf("expected actor ID %v, got %v", testActorID, retrievedActorID)
	}
}
