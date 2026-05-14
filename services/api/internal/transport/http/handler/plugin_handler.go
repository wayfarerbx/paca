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

	// Run any new migrations introduced by the upgraded version (idempotent).
	if h.migrationRunner != nil {
		if err := h.migrationRunner.Run(c.Request.Context(), installed.Name); err != nil {
			presenter.Error(c, apierr.New(apierr.CodeInternalError, "failed to run plugin migrations: "+err.Error()))
			return
		}
	}

	// Persist the new version and manifest.
	newVersion := entry.Version
	updated, err := h.svc.UpdatePlugin(c.Request.Context(), id, plugindom.UpdateInput{
		Version:  &newVersion,
		Manifest: &manifest,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}

	// Reload the WASM runtime with the new binary so traffic is served by the
	// upgraded module immediately. runtime.Load unloads any existing instance first.
	if updated.Enabled && h.runtime != nil {
		if err := h.runtime.Load(c.Request.Context(), *updated); err != nil {
			slog.Error("plugin upgrade: failed to reload runtime", "name", updated.Name, "error", err)
			presenter.Error(c, apierr.New(apierr.CodeInternalError, "artifacts upgraded but runtime reload failed: "+err.Error()))
			return
		}
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
// /api/v1/plugins/:pluginId/projects/:projectId/* and dispatches it to the
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

	// The sub-path after /projects/:projectId/ is available as the wildcard param.
	subPath := c.Param("path")
	if subPath == "" {
		subPath = "/"
	}

	route := matchPluginRoute(found.Manifest.Backend.Routes, c.Request.Method, subPath)
	if !h.applyPluginRouteMiddlewares(c, route) {
		return
	}

	// Build caller identity from JWT claims.
	claims := middleware.ClaimsFrom(c)
	callerID := ""
	userIDStr := ""
	callerRole := ""
	if claims != nil {
		callerRole = claims.Role
		userIDStr = claims.Subject

		if h.memberRepo == nil {
			presenter.Error(c, apierr.New(apierr.CodeInternalError, "plugin member resolver not available"))
			return
		}

		projectID, err := uuid.Parse(c.Param("projectId"))
		if err != nil {
			presenter.Error(c, apierr.New(apierr.CodeBadRequest, "invalid projectId"))
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

	projectScopedPath := "/projects/" + c.Param("projectId")
	if subPath != "/" {
		projectScopedPath += subPath
	}

	req := &pluginrt.HTTPRequest{
		Method:     c.Request.Method,
		Path:       projectScopedPath,
		ProjectID:  c.Param("projectId"),
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

func (h *PluginHandler) applyPluginRouteMiddlewares(c *gin.Context, route *plugindom.PluginRoute) bool {
	for _, mw := range h.routeMiddlewares(route) {
		name := strings.ToLower(strings.TrimSpace(mw.Name))
		switch name {
		case "authn":
			if h.tokenManager == nil {
				presenter.Error(c, apierr.New(apierr.CodeInternalError, "plugin route auth middleware is not configured"))
				return false
			}
			middleware.Authn(h.tokenManager, h.apiKeyAuth)(c)
		case "optionalauthn":
			if h.tokenManager == nil {
				presenter.Error(c, apierr.New(apierr.CodeInternalError, "plugin route auth middleware is not configured"))
				return false
			}
			middleware.OptionalAuthn(h.tokenManager, h.apiKeyAuth)(c)
		case "requirefreshpassword":
			middleware.RequireFreshPassword()(c)
		case "requirejwtauth":
			middleware.RequireJWTAuth()(c)
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
				scopeResolver = middleware.ProjectScopeFromParam(projectParam)
			default:
				presenter.Error(c, apierr.New(apierr.CodeInternalError, "plugin route requirePermissions has invalid scope"))
				return false
			}

			perms := make([]authz.Permission, 0, len(mw.Permissions))
			for _, p := range mw.Permissions {
				perms = append(perms, authz.Permission(p))
			}
			middleware.RequirePermissions(h.authorizer, scopeResolver, perms...)(c)
		default:
			presenter.Error(c, apierr.New(apierr.CodeInternalError, "plugin route uses unsupported middleware: "+mw.Name))
			return false
		}

		if c.IsAborted() {
			return false
		}
	}

	return true
}

func (h *PluginHandler) routeMiddlewares(route *plugindom.PluginRoute) []plugindom.PluginRouteMiddleware {
	if route != nil {
		if len(route.Middlewares) > 0 {
			return route.Middlewares
		}
		if route.Public {
			return nil
		}
	}

	// Backward-compatible default policy for plugin routes:
	// optional authn + fresh-password check + project read permission.
	return []plugindom.PluginRouteMiddleware{
		{Name: "optionalAuthn"},
		{Name: "requireFreshPassword"},
		{
			Name:         "requirePermissions",
			Scope:        "project",
			ProjectParam: "projectId",
			Permissions:  []string{string(authz.PermissionProjectsRead)},
		},
	}
}

func matchPluginRoute(routes []plugindom.PluginRoute, method, path string) *plugindom.PluginRoute {
	method = strings.ToUpper(strings.TrimSpace(method))
	for i := range routes {
		r := &routes[i]
		if strings.ToUpper(strings.TrimSpace(r.Method)) != method {
			continue
		}
		if _, ok := matchPathPattern(r.Path, path); ok {
			return r
		}
	}
	return nil
}

func matchPathPattern(pattern, path string) (map[string]string, bool) {
	patternSegments := splitPathSegments(pattern)
	pathSegments := splitPathSegments(path)
	if len(patternSegments) != len(pathSegments) {
		return nil, false
	}

	params := make(map[string]string)
	for i := range patternSegments {
		patternSegment := patternSegments[i]
		pathSegment := pathSegments[i]

		if strings.HasPrefix(patternSegment, ":") {
			name := strings.TrimPrefix(patternSegment, ":")
			if name == "" {
				return nil, false
			}
			params[name] = pathSegment
			continue
		}

		if patternSegment != pathSegment {
			return nil, false
		}
	}

	return params, true
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
// negative integer when a < b. It handles versions in "X.Y.Z" or "vX.Y.Z" form.
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

// parseSemver parses a "X.Y.Z" (or "vX.Y.Z") version string into its
// major, minor, and patch integer components.
func parseSemver(v string) ([3]int, error) {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	if len(parts) != 3 {
		return [3]int{}, fmt.Errorf("expected major.minor.patch, got %q", v)
	}
	var result [3]int
	for i, p := range parts {
		// Strip any pre-release suffix (e.g. "1-beta").
		if dash := strings.IndexByte(p, '-'); dash >= 0 {
			p = p[:dash]
		}
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return [3]int{}, fmt.Errorf("non-numeric version component %q", p)
		}
		result[i] = n
	}
	return result, nil
}
