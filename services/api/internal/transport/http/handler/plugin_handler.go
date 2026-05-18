package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/Paca-AI/api/internal/apierr"
	plugindom "github.com/Paca-AI/api/internal/domain/plugin"
	projectdom "github.com/Paca-AI/api/internal/domain/project"
	"github.com/Paca-AI/api/internal/platform/authz"
	pluginrt "github.com/Paca-AI/api/internal/platform/plugin"
	jwttoken "github.com/Paca-AI/api/internal/platform/token"
	"github.com/Paca-AI/api/internal/transport/http/dto"
	"github.com/Paca-AI/api/internal/transport/http/middleware"
	"github.com/Paca-AI/api/internal/transport/http/presenter"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// PluginHandler handles plugin management endpoints.
type PluginHandler struct {
	svc             plugindom.Service
	runtime         *pluginrt.Runtime
	memberRepo      projectdom.MemberRepository
	tokenManager    *jwttoken.Manager
	apiKeyAuth      middleware.APIKeyAuthenticator
	authorizer      *authz.Authorizer
	marketplace     *pluginrt.MarketplaceClient
	installer       *pluginrt.Installer
	migrationRunner *pluginrt.MigrationRunner
}

// NewPluginHandler creates a PluginHandler.
func NewPluginHandler(svc plugindom.Service, runtime *pluginrt.Runtime, memberRepo projectdom.MemberRepository) *PluginHandler {
	return &PluginHandler{svc: svc, runtime: runtime, memberRepo: memberRepo}
}

// WithRouteAuth wires authentication/authorization dependencies used for
// per-route plugin middleware policy enforcement.
func (h *PluginHandler) WithRouteAuth(
	tm *jwttoken.Manager,
	apiKeyAuth middleware.APIKeyAuthenticator,
	authorizer *authz.Authorizer,
) *PluginHandler {
	h.tokenManager = tm
	h.apiKeyAuth = apiKeyAuth
	h.authorizer = authorizer
	return h
}

// WithMarketplace wires marketplace dependencies onto the existing handler.
func (h *PluginHandler) WithMarketplace(
	marketplace *pluginrt.MarketplaceClient,
	installer *pluginrt.Installer,
	migrationRunner *pluginrt.MigrationRunner,
) *PluginHandler {
	h.marketplace = marketplace
	h.installer = installer
	h.migrationRunner = migrationRunner
	return h
}

// -------------------------------------------------------------------------
// PLUG-BE-10: Plugin management API
// -------------------------------------------------------------------------

// ListPlugins handles GET /api/v1/plugins.
func (h *PluginHandler) ListPlugins(c *gin.Context) {
	plugins, err := h.svc.ListPlugins(c.Request.Context())
	if err != nil {
		presenter.Error(c, err)
		return
	}
	pluginIDs := make([]uuid.UUID, 0, len(plugins))
	for _, p := range plugins {
		pluginIDs = append(pluginIDs, p.ID)
	}

	settingsByPlugin, err := h.svc.ListExtensionSettingsForPlugins(c.Request.Context(), pluginIDs)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	items := make([]dto.PluginResponse, 0, len(plugins))
	for _, p := range plugins {
		items = append(items, dto.PluginResponseFromEntityWithSettings(p, settingsByPlugin[p.ID]))
	}
	presenter.OK(c, dto.PluginListResponse{Plugins: items})
}

// InstallPlugin handles POST /api/v1/admin/plugins.
func (h *PluginHandler) InstallPlugin(c *gin.Context) {
	var req dto.InstallPluginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		presenter.Error(c, apierr.New(apierr.CodeBadRequest, err.Error()))
		return
	}
	plugin, err := h.svc.InstallPlugin(c.Request.Context(), plugindom.InstallInput{
		Name:     req.Name,
		Version:  req.Version,
		Manifest: req.Manifest,
		Enabled:  req.Enabled,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Created(c, dto.PluginResponseFromEntity(plugin))
}

// ListMarketplacePlugins handles GET /api/v1/admin/plugins/marketplace.
func (h *PluginHandler) ListMarketplacePlugins(c *gin.Context) {
	if h.marketplace == nil {
		presenter.Error(c, apierr.New(apierr.CodeInternalError, "marketplace not configured"))
		return
	}

	catalog, err := h.marketplace.List(c.Request.Context())
	if err != nil {
		presenter.Error(c, apierr.New(apierr.CodeInternalError, "failed to fetch marketplace catalog"))
		return
	}

	presenter.OK(c, dto.MarketplacePluginListResponseFromCatalog(catalog))
}

// InstallMarketplacePlugin handles POST /api/v1/admin/plugins/marketplace/install.
func (h *PluginHandler) InstallMarketplacePlugin(c *gin.Context) {
	if h.marketplace == nil || h.installer == nil {
		presenter.Error(c, apierr.New(apierr.CodeInternalError, "marketplace installer not configured"))
		return
	}

	var req dto.InstallMarketplacePluginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		presenter.Error(c, apierr.New(apierr.CodeBadRequest, err.Error()))
		return
	}

	entry, err := h.marketplace.FindPlugin(c.Request.Context(), req.Name)
	if err != nil {
		if err == pluginrt.ErrMarketplacePluginNotFound {
			presenter.Error(c, apierr.New(apierr.CodePluginNotFound, "plugin not found in marketplace"))
			return
		}
		presenter.Error(c, apierr.New(apierr.CodeBadRequest, "failed to resolve marketplace plugin: "+err.Error()))
		return
	}

	manifest, err := h.installer.Install(c.Request.Context(), *entry)
	if err != nil {
		presenter.Error(c, apierr.New(apierr.CodeBadRequest, "failed to install plugin artifacts: "+err.Error()))
		return
	}

	// Ensure downloaded artifacts are cleaned up if any subsequent step fails.
	pluginName := entry.Name
	success := false
	defer func() {
		if !success {
			if uninstallErr := h.installer.Uninstall(pluginName); uninstallErr != nil {
				slog.Error("failed to clean up plugin artifacts after install failure", "name", pluginName, "error", uninstallErr)
			}
		}
	}()

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	pl, err := h.svc.InstallPlugin(c.Request.Context(), plugindom.InstallInput{
		Name:     entry.Name,
		Version:  entry.Version,
		Manifest: manifest,
		Enabled:  enabled,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}

	if h.migrationRunner != nil {
		if err := h.migrationRunner.Run(c.Request.Context(), pl.Name); err != nil {
			_ = h.svc.DeletePlugin(c.Request.Context(), pl.ID)
			presenter.Error(c, apierr.New(apierr.CodeInternalError, "failed to run plugin migrations: "+err.Error()))
			return
		}
	}

	if pl.Enabled && h.runtime != nil {
		if err := h.runtime.Load(c.Request.Context(), *pl); err != nil {
			_ = h.svc.DeletePlugin(c.Request.Context(), pl.ID)
			presenter.Error(c, apierr.New(apierr.CodeInternalError, "failed to load plugin runtime: "+err.Error()))
			return
		}
	}

	success = true
	presenter.Created(c, dto.PluginResponseFromEntity(pl))
}

// UpdatePlugin handles PATCH /api/v1/admin/plugins/:pluginId.
func (h *PluginHandler) UpdatePlugin(c *gin.Context) {
	id, err := parsePluginID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	var req dto.UpdatePluginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		presenter.Error(c, apierr.New(apierr.CodeBadRequest, err.Error()))
		return
	}
	plugin, err := h.svc.UpdatePlugin(c.Request.Context(), id, plugindom.UpdateInput{
		Version:  req.Version,
		Manifest: req.Manifest,
		Enabled:  req.Enabled,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.OK(c, dto.PluginResponseFromEntity(plugin))
}

// UpgradeMarketplacePlugin handles POST /api/v1/admin/plugins/:pluginId/upgrade.
// It fetches the latest version from the marketplace catalog, validates that the
// marketplace version is strictly newer than the installed version, downloads and
// replaces all artifacts, runs any new migrations, updates the DB record, and
// reloads the plugin in the WASM runtime.
func (h *PluginHandler) UpgradeMarketplacePlugin(c *gin.Context) {
	if h.marketplace == nil || h.installer == nil {
		presenter.Error(c, apierr.New(apierr.CodeInternalError, "marketplace installer not configured"))
		return
	}

	id, err := parsePluginID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	// Resolve the installed plugin record.
	plugins, err := h.svc.ListPlugins(c.Request.Context())
	if err != nil {
		presenter.Error(c, err)
		return
	}
	var installed *plugindom.Plugin
	for _, p := range plugins {
		if p.ID == id {
			installed = p
			break
		}
	}
	if installed == nil {
		presenter.Error(c, apierr.New(apierr.CodePluginNotFound, "plugin not found"))
		return
	}

	// Look up the marketplace entry by the plugin's reverse-DNS name.
	entry, err := h.marketplace.FindPlugin(c.Request.Context(), installed.Name)
	if err != nil {
		if err == pluginrt.ErrMarketplacePluginNotFound {
			presenter.Error(c, apierr.New(apierr.CodePluginNotFound, "plugin not found in marketplace"))
			return
		}
		presenter.Error(c, apierr.New(apierr.CodeInternalError, "failed to resolve marketplace plugin: "+err.Error()))
		return
	}

	// Enforce semver ordering: refuse no-ops and downgrades.
	cmp, err := compareSemver(entry.Version, installed.Version)
	if err != nil {
		presenter.Error(c, apierr.New(apierr.CodeBadRequest, "version comparison failed: "+err.Error()))
		return
	}
	switch {
	case cmp == 0:
		presenter.Error(c, apierr.New(apierr.CodePluginAlreadyUpToDate, "plugin is already up to date"))
		return
	case cmp < 0:
		presenter.Error(c, apierr.New(apierr.CodePluginDowngradeNotAllowed, "marketplace version is older than installed version"))
		return
	}

	// Download and overwrite existing artifacts.
	manifest, err := h.installer.Install(c.Request.Context(), *entry)
	if err != nil {
		presenter.Error(c, apierr.New(apierr.CodeBadRequest, "failed to install plugin artifacts: "+err.Error()))
		return
	}
	if manifest.Version != entry.Version {
		presenter.Error(c, apierr.New(
			apierr.CodeBadRequest,
			fmt.Sprintf(
				"downloaded plugin manifest version %q does not match marketplace version %q",
				manifest.Version,
				entry.Version,
			),
		))
		return
	}

	// Capture the currently persisted plugin state so we can delay DB writes
	// until the upgraded runtime is active and best-effort roll back on failure.
	plugins, err = h.svc.ListPlugins(c.Request.Context())
	if err != nil {
		if h.installer != nil {
			if cleanupErr := h.installer.Uninstall(installed.Name); cleanupErr != nil {
				slog.Error("plugin upgrade: failed to clean up installed artifacts after state lookup failure", "name", installed.Name, "error", cleanupErr)
			}
		}
		presenter.Error(c, err)
		return
	}

	var current *plugindom.Plugin
	for i := range plugins {
		if plugins[i].ID == id {
			current = plugins[i]
			break
		}
	}
	if current == nil {
		if h.installer != nil {
			if cleanupErr := h.installer.Uninstall(installed.Name); cleanupErr != nil {
				slog.Error("plugin upgrade: failed to clean up installed artifacts after missing plugin state", "name", installed.Name, "error", cleanupErr)
			}
		}
		presenter.Error(c, apierr.New(apierr.CodePluginNotFound, "plugin not found"))
		return
	}

	cleanupArtifacts := func(reason string) {
		if h.installer == nil {
			return
		}
		if cleanupErr := h.installer.Uninstall(installed.Name); cleanupErr != nil {
			slog.Error("plugin upgrade: failed to clean up installed artifacts", "name", installed.Name, "reason", reason, "error", cleanupErr)
		}
	}

	// Run any new migrations introduced by the upgraded version (idempotent).
	if h.migrationRunner != nil {
		if err := h.migrationRunner.Run(c.Request.Context(), installed.Name); err != nil {
			cleanupArtifacts("migration failure")
			presenter.Error(c, apierr.New(apierr.CodeInternalError, "failed to run plugin migrations: "+err.Error()))
			return
		}
	}

	newVersion := entry.Version

	// Reload the WASM runtime with the new binary before persisting the new
	// version so the DB only advances once the upgraded module is actually live.
	runtimePlugin := *current
	runtimePlugin.Version = newVersion
	runtimePlugin.Manifest = manifest
	if runtimePlugin.Enabled && h.runtime != nil {
		if err := h.runtime.Load(c.Request.Context(), runtimePlugin); err != nil {
			cleanupArtifacts("runtime reload failure")
			slog.Error("plugin upgrade: failed to reload runtime", "name", runtimePlugin.Name, "error", err)
			presenter.Error(c, apierr.New(apierr.CodeInternalError, "artifacts upgraded but runtime reload failed: "+err.Error()))
			return
		}
	}

	// Persist the new version and manifest only after migrations and runtime
	// reload succeed.
	updated, err := h.svc.UpdatePlugin(c.Request.Context(), id, plugindom.UpdateInput{
		Version:  &newVersion,
		Manifest: &manifest,
	})
	if err != nil {
		if current.Enabled && h.runtime != nil {
			if rollbackErr := h.runtime.Load(c.Request.Context(), *current); rollbackErr != nil {
				slog.Error("plugin upgrade: failed to roll back runtime after DB update failure", "name", current.Name, "error", rollbackErr)
			}
		}
		cleanupArtifacts("db update failure")
		presenter.Error(c, err)
		return
	}

	presenter.OK(c, dto.PluginResponseFromEntity(updated))
}

// DeletePlugin handles DELETE /api/v1/admin/plugins/:pluginId.
func (h *PluginHandler) DeletePlugin(c *gin.Context) {
	id, err := parsePluginID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	var pluginName string
	if h.runtime != nil || h.installer != nil {
		plugins, err := h.svc.ListPlugins(c.Request.Context())
		if err != nil {
			presenter.Error(c, err)
			return
		}
		for _, p := range plugins {
			if p.ID == id {
				pluginName = p.Name
				break
			}
		}
	}

	if err := h.svc.DeletePlugin(c.Request.Context(), id); err != nil {
		presenter.Error(c, err)
		return
	}
	if h.runtime != nil && pluginName != "" {
		h.runtime.Unload(c.Request.Context(), pluginName)
	}
	if h.installer != nil && pluginName != "" {
		if err := h.installer.Uninstall(pluginName); err != nil {
			// Log but don't fail the request — the DB record is already gone.
			slog.Error("failed to uninstall plugin artifacts", "name", pluginName, "error", err)
		}
	}
	presenter.NoContent(c)
}

// -------------------------------------------------------------------------
// PLUG-BE-11: Plugin extension setting endpoint (admin-only)
// -------------------------------------------------------------------------

// UpdateExtensionSetting handles PATCH /api/v1/admin/plugin-extension-settings.
// Only the super admin may call this endpoint.
func (h *PluginHandler) UpdateExtensionSetting(c *gin.Context) {
	var req dto.UpdatePluginExtensionSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		presenter.Error(c, apierr.New(apierr.CodeBadRequest, err.Error()))
		return
	}

	setting, err := h.svc.UpdateExtensionSetting(c.Request.Context(), plugindom.UpdateExtensionSettingInput{
		PluginID:       req.PluginID,
		ExtensionPoint: req.ExtensionPoint,
		Settings:       req.Settings,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.OK(c, dto.PluginExtensionSettingFromEntity(setting))
}

// -------------------------------------------------------------------------
// PLUG-BE-08: Plugin route proxy
// -------------------------------------------------------------------------

// ProxyRequest handles any request under
// /api/v1/plugins/:pluginId/* and dispatches it to the
// matching plugin's HandleRequest WASM export.
func (h *PluginHandler) ProxyRequest(c *gin.Context) {
	if h.runtime == nil {
		presenter.Error(c, apierr.New(apierr.CodeInternalError, "plugin runtime not available"))
		return
	}

	pluginID := c.Param("pluginId")

	// Validate that the plugin exists and is enabled.
	plugin, err := h.svc.ListPlugins(c.Request.Context())
	if err != nil {
		presenter.Error(c, err)
		return
	}
	var found *plugindom.Plugin
	for _, p := range plugin {
		if p.Name == pluginID && p.Enabled {
			found = p
			break
		}
	}
	if found == nil {
		presenter.Error(c, apierr.New(apierr.CodePluginNotFound, "plugin not found or disabled"))
		return
	}

	// The full sub-path after /plugins/:pluginId/ is captured by the wildcard param.
	// Plugins may embed /projects/:projectId/ within this path at their own discretion.
	subPath := c.Param("path")
	if subPath == "" {
		subPath = "/"
	}

	route, pathParams := matchPluginRoute(found.Manifest.Backend.Routes, c.Request.Method, subPath)
	if !h.applyPluginRouteMiddlewares(c, route, pathParams) {
		return
	}

	// Determine whether the route requires project-member resolution and which
	// path-param holds the project UUID (configurable via manifest; defaults to
	// "projectId" when not specified).
	resolveProjectMember, projectParamName := projectMemberParam(route)

	// Build caller identity from JWT claims.
	claims := middleware.ClaimsFrom(c)
	callerID := ""
	userIDStr := ""
	callerRole := ""
	if claims != nil {
		callerRole = claims.Role
		userIDStr = claims.Subject

		// Resolve project member only when the route's manifest explicitly declares
		// project-scoped permission enforcement.  Plugin routes that don't carry a
		// requirePermissions(scope=project) middleware won't incur a DB lookup, and
		// plugins that choose a non-standard path structure are not forced to embed
		// :projectId at any particular position.
		if resolveProjectMember {
			if projectIDStr := pathParams[projectParamName]; projectIDStr != "" {
				if h.memberRepo == nil {
					presenter.Error(c, apierr.New(apierr.CodeInternalError, "plugin member resolver not available"))
					return
				}
				projectID, err := uuid.Parse(projectIDStr)
				if err != nil {
					presenter.Error(c, apierr.New(apierr.CodeBadRequest, "invalid projectId in path"))
					return
				}
				userID, err := uuid.Parse(claims.Subject)
				if err != nil {
					presenter.Error(c, apierr.New(apierr.CodeBadRequest, "invalid subject claim"))
					return
				}
				member, err := h.memberRepo.FindMemberByUserProject(c.Request.Context(), userID, projectID)
				if err != nil {
					presenter.Error(c, err)
					return
				}
				callerID = member.ID.String()
			}
		}
	}

	// Read request body.
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		presenter.Error(c, apierr.New(apierr.CodeBadRequest, "failed to read request body"))
		return
	}

	// Build flattened headers map (first value per header name).
	headers := make(map[string]string, len(c.Request.Header))
	for k, vs := range c.Request.Header {
		if len(vs) > 0 {
			headers[k] = vs[0]
		}
	}

	req := &pluginrt.HTTPRequest{
		Method:     c.Request.Method,
		Path:       subPath,
		ProjectID:  pathParams[projectParamName],
		CallerID:   callerID,
		UserID:     userIDStr,
		CallerRole: callerRole,
		Headers:    headers,
		Body:       bodyBytes,
	}

	// Attach request to context for HTTP host functions.
	reqCtx := pluginrt.WithPluginRequest(c.Request.Context(), req)

	reqBytes, err := json.Marshal(req)
	if err != nil {
		presenter.Error(c, apierr.New(apierr.CodeInternalError, "failed to serialise request"))
		return
	}

	respBytes, err := h.runtime.HandleRequest(reqCtx, pluginID, reqBytes)
	if err != nil {
		presenter.Error(c, apierr.New(apierr.CodeInternalError, "plugin execution error: "+err.Error()))
		return
	}

	// Parse the plugin response envelope. The current SDK returns:
	// {"status": number, "headers": object, "body": base64-bytes}
	var pluginResp struct {
		Status  int               `json:"status"`
		Headers map[string]string `json:"headers"`
		Body    []byte            `json:"body"`
	}
	if err := json.Unmarshal(respBytes, &pluginResp); err != nil {
		// Fallback: send raw bytes as JSON.
		c.Data(http.StatusOK, "application/json", respBytes)
		return
	}

	statusCode := pluginResp.Status
	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	contentType := ""
	if pluginResp.Headers != nil {
		contentType = pluginResp.Headers["Content-Type"]
		if contentType == "" {
			contentType = pluginResp.Headers["content-type"]
		}
	}
	if contentType == "" {
		contentType = "application/json"
	}

	for k, v := range pluginResp.Headers {
		if !strings.EqualFold(k, "Content-Type") {
			c.Header(k, v)
		}
	}

	c.Data(statusCode, contentType, pluginResp.Body)
}

func (h *PluginHandler) applyPluginRouteMiddlewares(c *gin.Context, route *plugindom.PluginRoute, pathParams map[string]string) bool {
	for _, mw := range h.routeMiddlewares(route) {
		name := strings.ToLower(strings.TrimSpace(mw.Name))
		switch name {
		case "authn":
			if h.tokenManager == nil {
				presenter.Error(c, apierr.New(apierr.CodeInternalError, "plugin route auth middleware is not configured"))
				return false
			}
			if !middleware.EnforceAuthn(c, h.tokenManager, h.apiKeyAuth) {
				return false
			}
		case "optionalauthn":
			if h.tokenManager == nil {
				presenter.Error(c, apierr.New(apierr.CodeInternalError, "plugin route auth middleware is not configured"))
				return false
			}
			if !middleware.EnforceOptionalAuthn(c, h.tokenManager, h.apiKeyAuth) {
				return false
			}
		case "requirefreshpassword":
			if !middleware.EnforceFreshPassword(c) {
				return false
			}
		case "requirejwtauth":
			if !middleware.EnforceJWTAuth(c) {
				return false
			}
		case "requirepermissions":
			if h.authorizer == nil {
				presenter.Error(c, apierr.New(apierr.CodeInternalError, "plugin route authorization middleware is not configured"))
				return false
			}
			if len(mw.Permissions) == 0 {
				presenter.Error(c, apierr.New(apierr.CodeInternalError, "plugin route requirePermissions requires at least one permission"))
				return false
			}
			scopeResolver := middleware.GlobalScope()
			scope := strings.ToLower(strings.TrimSpace(mw.Scope))
			switch scope {
			case "", "global":
				// keep global scope
			case "project":
				projectParam := strings.TrimSpace(mw.ProjectParam)
				if projectParam == "" {
					projectParam = "projectId"
				}
				pParam := projectParam
				scopeResolver = func(_ *gin.Context) (*uuid.UUID, error) {
					v := pathParams[pParam]
					if v == "" {
						return nil, nil
					}
					id, err := uuid.Parse(v)
					if err != nil {
						return nil, err
					}
					return &id, nil
				}
			default:
				presenter.Error(c, apierr.New(apierr.CodeInternalError, "plugin route requirePermissions has invalid scope"))
				return false
			}

			perms := make([]authz.Permission, 0, len(mw.Permissions))
			for _, p := range mw.Permissions {
				perms = append(perms, authz.Permission(p))
			}
			if !middleware.EnforcePermissions(c, h.authorizer, scopeResolver, perms...) {
				return false
			}
		default:
			presenter.Error(c, apierr.New(apierr.CodeInternalError, "plugin route uses unsupported middleware: "+mw.Name))
			return false
		}
	}

	return true
}

func (h *PluginHandler) routeMiddlewares(route *plugindom.PluginRoute) []plugindom.PluginRouteMiddleware {
	if route != nil {
		// Distinguish omitted (nil) vs explicit empty list:
		// nil -> apply default policy, [] -> no middlewares.
		if route.Middlewares != nil {
			return route.Middlewares
		}
		if route.Public {
			return nil
		}
	}

	// Default policy for plugin routes: optional authn + fresh-password check.
	// Project-scoped permission checks must be declared explicitly in the
	// plugin route's middlewares list.
	return []plugindom.PluginRouteMiddleware{
		{Name: "optionalAuthn"},
		{Name: "requireFreshPassword"},
	}
}

func matchPluginRoute(routes []plugindom.PluginRoute, method, path string) (*plugindom.PluginRoute, map[string]string) {
	method = strings.ToUpper(strings.TrimSpace(method))
	var matched *plugindom.PluginRoute
	var matchedParams map[string]string
	var matchedScore routePatternScore
	for i := range routes {
		r := &routes[i]
		if strings.ToUpper(strings.TrimSpace(r.Method)) != method {
			continue
		}
		if params, ok := matchPathPattern(r.Path, path); ok {
			score := scoreRoutePattern(r.Path)
			// Prefer more specific patterns (fewer wildcards, then more static segments).
			// Ties keep manifest order for deterministic fallback behavior.
			if matched == nil || score.moreSpecificThan(matchedScore) {
				matched = r
				matchedScore = score
				matchedParams = params
			}
		}
	}
	if matchedParams == nil {
		matchedParams = make(map[string]string)
	}
	return matched, matchedParams
}

type routePatternScore struct {
	staticSegments   int
	paramSegments    int
	wildcardSegments int
}

// scoreRoutePattern computes path-pattern specificity by counting static,
// param, and wildcard segments.
func scoreRoutePattern(pattern string) routePatternScore {
	var score routePatternScore
	for _, segment := range splitPathSegments(pattern) {
		switch {
		case strings.HasPrefix(segment, "*"):
			score.wildcardSegments++
		case strings.HasPrefix(segment, ":"):
			score.paramSegments++
		default:
			score.staticSegments++
		}
	}
	return score
}

// moreSpecificThan orders routes by specificity:
// fewer wildcards, then more static segments, then fewer param segments.
func (score routePatternScore) moreSpecificThan(other routePatternScore) bool {
	if score.wildcardSegments != other.wildcardSegments {
		return score.wildcardSegments < other.wildcardSegments
	}
	if score.staticSegments != other.staticSegments {
		return score.staticSegments > other.staticSegments
	}
	if score.paramSegments != other.paramSegments {
		return score.paramSegments < other.paramSegments
	}
	return false
}

func matchPathPattern(pattern, path string) (map[string]string, bool) {
	patternSegments := splitPathSegments(pattern)
	pathSegments := splitPathSegments(path)
	params := make(map[string]string)
	pathIdx := 0
	for i := range patternSegments {
		patternSegment := patternSegments[i]

		if strings.HasPrefix(patternSegment, "*") {
			name := strings.TrimPrefix(patternSegment, "*")
			// Catch-all wildcards are only supported as the last path segment.
			if name == "" || i != len(patternSegments)-1 {
				// Empty wildcard names are invalid ("/items/*" has no param key).
				return nil, false
			}
			if pathIdx >= len(pathSegments) {
				return nil, false
			}
			params[name] = strings.Join(pathSegments[pathIdx:], "/")
			return params, true
		}

		if pathIdx >= len(pathSegments) {
			return nil, false
		}
		pathSegment := pathSegments[pathIdx]

		if strings.HasPrefix(patternSegment, ":") {
			name := strings.TrimPrefix(patternSegment, ":")
			if name == "" {
				return nil, false
			}
			params[name] = pathSegment
			pathIdx++
			continue
		}

		if patternSegment != pathSegment {
			return nil, false
		}
		pathIdx++
	}

	if pathIdx != len(pathSegments) {
		return nil, false
	}

	return params, true
}

// projectMemberParam returns whether the route manifest requires project-member
// resolution and, if so, which path-param name holds the project UUID.
// The param name is taken from the first requirePermissions(scope=project)
// middleware's ProjectParam field, defaulting to "projectId" when not set.
func projectMemberParam(route *plugindom.PluginRoute) (bool, string) {
	if route == nil {
		return false, ""
	}
	for _, mw := range route.Middlewares {
		if strings.ToLower(strings.TrimSpace(mw.Name)) == "requirepermissions" &&
			strings.ToLower(strings.TrimSpace(mw.Scope)) == "project" {
			param := strings.TrimSpace(mw.ProjectParam)
			if param == "" {
				param = "projectId"
			}
			return true, param
		}
	}
	return false, ""
}

func splitPathSegments(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

// -------------------------------------------------------------------------
// Helpers
// -------------------------------------------------------------------------

func parsePluginID(c *gin.Context) (uuid.UUID, error) {
	raw := c.Param("pluginId")
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, apierr.New(apierr.CodeBadRequest, "invalid pluginId: "+raw)
	}
	return id, nil
}

// compareSemver returns a positive integer when a > b, 0 when equal, and a
// negative integer when a < b. Only strict "X.Y.Z" (or "vX.Y.Z") versions
// are accepted; pre-release identifiers and build metadata cause an error.
func compareSemver(a, b string) (int, error) {
	pa, err := parseSemver(a)
	if err != nil {
		return 0, fmt.Errorf("invalid version %q: %w", a, err)
	}
	pb, err := parseSemver(b)
	if err != nil {
		return 0, fmt.Errorf("invalid version %q: %w", b, err)
	}
	for i := range pa {
		if pa[i] != pb[i] {
			return pa[i] - pb[i], nil
		}
	}
	return 0, nil
}

// parseSemver parses a strict "X.Y.Z" (or "vX.Y.Z") version string into its
// major, minor, and patch integer components. Pre-release identifiers (e.g.
// "1.0.0-beta.1") and build metadata (e.g. "1.0.0+001") are rejected with an
// error so that callers never silently treat different precedence levels as
// equal.
func parseSemver(v string) ([3]int, error) {
	v = strings.TrimPrefix(v, "v")
	// Reject build metadata.
	if strings.ContainsRune(v, '+') {
		return [3]int{}, fmt.Errorf("version %q must not contain build metadata", v)
	}
	// Reject pre-release identifiers.
	if strings.ContainsRune(v, '-') {
		return [3]int{}, fmt.Errorf("version %q must not contain pre-release identifiers; only strict X.Y.Z versions are supported", v)
	}
	parts := strings.SplitN(v, ".", 3)
	if len(parts) != 3 {
		return [3]int{}, fmt.Errorf("expected major.minor.patch, got %q", v)
	}
	var result [3]int
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return [3]int{}, fmt.Errorf("non-numeric version component %q", p)
		}
		result[i] = n
	}
	return result, nil
}
