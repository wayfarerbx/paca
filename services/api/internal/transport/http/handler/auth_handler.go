package handler

import (
	"net/http"
	"time"

	"github.com/Paca-AI/api/internal/apierr"
	"github.com/Paca-AI/api/internal/config"
	domainauth "github.com/Paca-AI/api/internal/domain/auth"
	projectdom "github.com/Paca-AI/api/internal/domain/project"
	userdom "github.com/Paca-AI/api/internal/domain/user"
	"github.com/Paca-AI/api/internal/transport/http/dto"
	"github.com/Paca-AI/api/internal/transport/http/middleware"
	"github.com/Paca-AI/api/internal/transport/http/presenter"
	"github.com/google/uuid"
)

const (
	accessCookieName  = "access_token"
	refreshCookieName = "refresh_token"
	// refreshCookiePath restricts the refresh cookie to the rotation endpoint
	// so browsers never send it on regular API requests.
	refreshCookiePath = "/api/v1/auth/refresh"
)

// CookieConfig carries compile-time-safe settings for auth cookies.
type CookieConfig struct {
	Secure            bool
	AccessTTL         time.Duration
	RefreshTTL        time.Duration // persistent session (remember me = true)
	RefreshSessionTTL time.Duration // ephemeral session (remember me = false)
}

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	svc    domainauth.Service
	cookie CookieConfig

	// users and keycloak are only set when WithKeycloak has been called;
	// KeycloakInitiate/KeycloakCallback respond with apierr.CodeNotFound
	// when keycloak.Enabled() is false.
	users     userdom.Service
	keycloak  config.KeycloakConfig
	publicURL string
	http      *http.Client

	// projects and defaultProjectID are only set when WithDefaultProject has
	// been called; when set, every Keycloak login ensures the user is a
	// member of this project (see keycloak_handler.go).
	projects         projectdom.Service
	defaultProjectID uuid.UUID
}

// NewAuthHandler returns an AuthHandler wired to the provided auth service.
func NewAuthHandler(svc domainauth.Service, cookie CookieConfig) *AuthHandler {
	return &AuthHandler{svc: svc, cookie: cookie}
}

// WithKeycloak enables the Keycloak/OIDC login routes. users is used to
// find-or-create the local account for a Keycloak-authenticated identity.
// publicURL is the externally reachable base URL of this API (used to build
// the OAuth2 redirect_uri) — it must match a valid redirect URI configured on
// the Keycloak client.
func (h *AuthHandler) WithKeycloak(users userdom.Service, kc config.KeycloakConfig, publicURL string) *AuthHandler {
	h.users = users
	h.keycloak = kc
	h.publicURL = publicURL
	h.http = &http.Client{Timeout: 10 * time.Second}
	return h
}

// WithDefaultProject makes every Keycloak login ensure the authenticated user
// is a member of projectID, in addition to find-or-create/role-sync. This is
// a no-op (parsed once, checked at call time) when projectID is empty.
func (h *AuthHandler) WithDefaultProject(projects projectdom.Service, projectID string) *AuthHandler {
	h.projects = projects
	h.defaultProjectID, _ = uuid.Parse(projectID)
	return h
}

// Login handles POST /auth/login.
// On success, access and refresh tokens are set as HttpOnly cookies; no token
// values appear in the response body.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if !middleware.BindJSON(w, r, &req) {
		return
	}

	if req.Username == "" {
		presenter.Error(w, r, apierr.New(apierr.CodeBadRequest, "username is required"))
		return
	}
	if len(req.Password) < 8 {
		presenter.Error(w, r, apierr.New(apierr.CodeBadRequest, "password must be at least 8 characters"))
		return
	}

	pair, err := h.svc.Login(r.Context(), req.Username, req.Password, req.RememberMe)
	if err != nil {
		presenter.Error(w, r, err)
		return
	}

	h.setTokenCookies(w, pair, pair.RefreshTTL)
	presenter.OK(w, r, map[string]any{"message": "logged in"})
}

// Refresh handles POST /auth/refresh.
// The refresh token is read from the HttpOnly refresh_token cookie and, on
// success, a rotated token pair is written back as cookies.
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	refreshCookie, err := r.Cookie(refreshCookieName)
	if err != nil || refreshCookie.Value == "" {
		presenter.Error(w, r, apierr.New(apierr.CodeMissingToken, "missing refresh token"))
		return
	}

	pair, err := h.svc.Refresh(r.Context(), refreshCookie.Value)
	if err != nil {
		presenter.Error(w, r, err)
		return
	}

	h.setTokenCookies(w, pair, pair.RefreshTTL)
	presenter.OK(w, r, map[string]any{"message": "token refreshed"})
}

// Logout handles POST /auth/logout.  Requires an authenticated access token.
// Revokes the session family and clears both auth cookies.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFrom(r)
	if claims == nil {
		presenter.Error(w, r, apierr.New(apierr.CodeUnauthenticated, "unauthenticated"))
		return
	}

	if err := h.svc.Logout(r.Context(), claims.FamilyID); err != nil {
		presenter.Error(w, r, err)
		return
	}

	h.clearCookies(w, r)
	presenter.OK(w, r, map[string]any{"message": "logged out"})
}

// setTokenCookies writes both tokens into HttpOnly Set-Cookie headers.
// refreshTTL controls the MaxAge of the refresh cookie and should match the
// TTL embedded in the refresh JWT (see TokenPair.RefreshTTL).
func (h *AuthHandler) setTokenCookies(w http.ResponseWriter, pair *domainauth.TokenPair, refreshTTL time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     accessCookieName,
		Value:    pair.AccessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookie.Secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(h.cookie.AccessTTL.Seconds()),
	})
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    pair.RefreshToken,
		Path:     refreshCookiePath,
		HttpOnly: true,
		Secure:   h.cookie.Secure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(refreshTTL.Seconds()),
	})
}

// clearCookies expires both auth cookies immediately.
func (h *AuthHandler) clearCookies(w http.ResponseWriter, _ *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     accessCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookie.Secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     refreshCookiePath,
		HttpOnly: true,
		Secure:   h.cookie.Secure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
}
