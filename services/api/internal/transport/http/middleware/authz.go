package middleware

import (
	"context"
	"errors"

	"github.com/Paca-AI/api/internal/apierr"
	projectdom "github.com/Paca-AI/api/internal/domain/project"
	"github.com/Paca-AI/api/internal/platform/authz"
	"github.com/Paca-AI/api/internal/transport/http/presenter"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ScopeResolver resolves a scope-specific project ID for permission checks.
// nil means global-only authorization.
type ScopeResolver func(c *gin.Context) (*uuid.UUID, error)

// GlobalScope forces global-only permission checks.
func GlobalScope() ScopeResolver {
	return func(*gin.Context) (*uuid.UUID, error) { return nil, nil }
}

// ProjectScopeFromParam resolves a project ID from a route parameter.
func ProjectScopeFromParam(param string) ScopeResolver {
	return func(c *gin.Context) (*uuid.UUID, error) {
		v := c.Param(param)
		if v == "" {
			return nil, apierr.New(apierr.CodeBadRequest, "missing project id")
		}
		id, err := uuid.Parse(v)
		if err != nil {
			return nil, apierr.New(apierr.CodeBadRequest, "invalid project id")
		}
		return &id, nil
	}
}

// RequirePermissions enforces permission-based authorization and supports
// global and project-scoped checks.
func RequirePermissions(authorizer *authz.Authorizer, scope ScopeResolver, permissions ...authz.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !EnforcePermissions(c, authorizer, scope, permissions...) {
			return
		}
		c.Next()
	}
}

// EnforcePermissions checks authorization without advancing the Gin handler chain.
func EnforcePermissions(c *gin.Context, authorizer *authz.Authorizer, scope ScopeResolver, permissions ...authz.Permission) bool {
	claims := ClaimsFrom(c)
	if claims == nil {
		presenter.Error(c, apierr.New(apierr.CodeUnauthenticated, "unauthenticated"))
		return false
	}

	if authorizer == nil {
		presenter.Error(c, apierr.New(apierr.CodeInternalError, "authorization not configured"))
		return false
	}

	resolver := scope
	if resolver == nil {
		resolver = GlobalScope()
	}
	projectID, err := resolver(c)
	if err != nil {
		presenter.Error(c, err)
		return false
	}

	agentID, hasAgentID := AgentIDFromGinContext(c)

	var allowed bool
	if hasAgentID && projectID != nil {
		allowed, err = authorizer.HasPermissionsForAgent(c.Request.Context(), agentID, *projectID, permissions...)
	} else {
		userID, parseErr := uuid.Parse(claims.Subject)
		if parseErr != nil {
			presenter.Error(c, apierr.New(apierr.CodeBadRequest, "invalid subject claim"))
			return false
		}
		allowed, err = authorizer.HasPermissions(c.Request.Context(), userID, projectID, claims.Role, permissions...)
	}

	if err != nil {
		presenter.Error(c, err)
		return false
	}
	if !allowed {
		presenter.Error(c, apierr.New(apierr.CodeForbidden, "insufficient permissions"))
		return false
	}

	return true
}

// Authz keeps backwards-compatible middleware semantics for global scope.
func Authz(authorizer *authz.Authorizer, permissions ...authz.Permission) gin.HandlerFunc {
	return RequirePermissions(authorizer, GlobalScope(), permissions...)
}

// PermissionGroup pairs a scope resolver with the permissions required in that scope.
// Used with RequireAnyPermissions to express OR-style authorization policies.
type PermissionGroup struct {
	Scope       ScopeResolver
	Permissions []authz.Permission
}

// RequireAnyPermissions grants access if the user satisfies at least one of the
// provided PermissionGroups. Groups are evaluated in order; the first satisfied
// group short-circuits the check. If no group is satisfied, 403 is returned.
//
// Typical use: allow access when the caller holds a broad global permission
// (e.g. projects.read) OR a narrower project-scoped one.
// Also supports agent authentication via X-Agent-ID header with the agent API key.
func RequireAnyPermissions(authorizer *authz.Authorizer, groups ...PermissionGroup) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := ClaimsFrom(c)
		if claims == nil {
			presenter.Error(c, apierr.New(apierr.CodeUnauthenticated, "unauthenticated"))
			return
		}

		if authorizer == nil {
			presenter.Error(c, apierr.New(apierr.CodeInternalError, "authorization not configured"))
			return
		}

		agentID, hasAgentID := AgentIDFromGinContext(c)
		var userID uuid.UUID

		if !hasAgentID {
			var parseErr error
			userID, parseErr = uuid.Parse(claims.Subject)
			if parseErr != nil {
				presenter.Error(c, apierr.New(apierr.CodeBadRequest, "invalid subject claim"))
				return
			}
		}

		var firstScopeErr error
		for _, group := range groups {
			resolver := group.Scope
			if resolver == nil {
				resolver = GlobalScope()
			}
			projectID, err := resolver(c)
			if err != nil {
				if firstScopeErr == nil {
					firstScopeErr = err
				}
				continue
			}

			var allowed bool
			if hasAgentID && projectID != nil {
				allowed, err = authorizer.HasPermissionsForAgent(c.Request.Context(), agentID, *projectID, group.Permissions...)
			} else {
				allowed, err = authorizer.HasPermissions(c.Request.Context(), userID, projectID, claims.Role, group.Permissions...)
			}

			if err != nil {
				presenter.Error(c, err)
				return
			}
			if allowed {
				c.Next()
				return
			}
		}

		if firstScopeErr != nil {
			presenter.Error(c, firstScopeErr)
			return
		}
		presenter.Error(c, apierr.New(apierr.CodeForbidden, "insufficient permissions"))
	}
}

// ProjectVisibilityChecker is the minimal interface the public-project
// middleware requires.  It is satisfied by *projectsvc.Service.
type ProjectVisibilityChecker interface {
	IsProjectPublic(ctx context.Context, id uuid.UUID) (bool, error)
}

// RequirePublicProjectOrPermissions grants access when at least one of the
// following conditions is true:
//
//   - The request is authenticated and the caller satisfies any of the
//     provided PermissionGroups (same logic as RequireAnyPermissions).
//   - The project identified by the "projectId" route parameter has
//     is_public = true, regardless of authentication status.
//
// Use this instead of RequireAnyPermissions on read-only project-scoped routes
// that should be accessible to anonymous users when the project is public.
// Also supports agent authentication via X-Agent-ID header with the agent API key.
func RequirePublicProjectOrPermissions(checker ProjectVisibilityChecker, authorizer *authz.Authorizer, groups ...PermissionGroup) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := ClaimsFrom(c)

		agentID, hasAgentID := AgentIDFromGinContext(c)
		var userID uuid.UUID

		if !hasAgentID && claims != nil {
			var parseErr error
			userID, parseErr = uuid.Parse(claims.Subject)
			if parseErr != nil {
				presenter.Error(c, apierr.New(apierr.CodeBadRequest, "invalid subject claim"))
				return
			}
		}

		// Authenticated path: run normal permission check.
		if claims != nil {
			var firstScopeErr error
			for _, group := range groups {
				resolver := group.Scope
				if resolver == nil {
					resolver = GlobalScope()
				}
				projectID, err := resolver(c)
				if err != nil {
					if firstScopeErr == nil {
						firstScopeErr = err
					}
					continue
				}

				var allowed bool
				if hasAgentID && projectID != nil {
					allowed, err = authorizer.HasPermissionsForAgent(c.Request.Context(), agentID, *projectID, group.Permissions...)
				} else {
					allowed, err = authorizer.HasPermissions(c.Request.Context(), userID, projectID, claims.Role, group.Permissions...)
				}

				if err != nil {
					presenter.Error(c, err)
					return
				}
				if allowed {
					c.Next()
					return
				}
			}
			if firstScopeErr != nil {
				presenter.Error(c, firstScopeErr)
				return
			}
			presenter.Error(c, apierr.New(apierr.CodeForbidden, "insufficient permissions"))
			return
		}

		// Unauthenticated path: allow only when the project is public.
		projectIDStr := c.Param("projectId")
		if projectIDStr == "" {
			presenter.Error(c, apierr.New(apierr.CodeUnauthenticated, "unauthenticated"))
			return
		}
		projectID, err := uuid.Parse(projectIDStr)
		if err != nil {
			presenter.Error(c, apierr.New(apierr.CodeBadRequest, "invalid project id"))
			return
		}
		isPublic, err := checker.IsProjectPublic(c.Request.Context(), projectID)
		if err != nil {
			if errors.Is(err, projectdom.ErrNotFound) {
				presenter.Error(c, apierr.New(apierr.CodeUnauthenticated, "unauthenticated"))
				return
			}
			presenter.Error(c, err)
			return
		}
		if !isPublic {
			presenter.Error(c, apierr.New(apierr.CodeUnauthenticated, "unauthenticated"))
			return
		}
		c.Next()
	}
}
