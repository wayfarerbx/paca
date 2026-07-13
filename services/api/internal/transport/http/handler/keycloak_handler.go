package handler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Paca-AI/api/internal/apierr"
	userdom "github.com/Paca-AI/api/internal/domain/user"
	"github.com/Paca-AI/api/internal/transport/http/presenter"
)

// keycloakStateCookie carries the CSRF state value between InitiateKeycloak
// and KeycloakCallback. It is short-lived and scoped to the callback path.
const (
	keycloakStateCookie = "keycloak_state"
	keycloakStatePath   = "/api/v1/auth/keycloak"
	keycloakStateTTL    = 5 * time.Minute
)

// keycloakTokenResponse is the subset of the OIDC token endpoint response we
// need. Fields not listed here (id_token, expires_in, ...) are ignored.
type keycloakTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Error       string `json:"error"`
	ErrorDesc   string `json:"error_description"`
}

// keycloakAccessClaims is the subset of access-token JWT claims we read.
// The token is decoded without signature verification: it was received
// directly from Keycloak over TLS via the confidential-client token
// exchange (never from the browser/end user), so the transport itself is
// the trust boundary.
type keycloakAccessClaims struct {
	Subject           string `json:"sub"`
	PreferredUsername string `json:"preferred_username"`
	Name              string `json:"name"`
	RealmAccess       struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
}

func (h *AuthHandler) keycloakEnabled() bool {
	return h.keycloak.Enabled() && h.users != nil
}

func (h *AuthHandler) keycloakAuthURL() string {
	return strings.TrimRight(h.keycloak.Host, "/") + "/realms/" + h.keycloak.Realm + "/protocol/openid-connect/auth"
}

func (h *AuthHandler) keycloakTokenURL() string {
	return strings.TrimRight(h.keycloak.Host, "/") + "/realms/" + h.keycloak.Realm + "/protocol/openid-connect/token"
}

func (h *AuthHandler) keycloakRedirectURI() string {
	return strings.TrimRight(h.publicURL, "/") + "/api/v1/auth/keycloak/callback"
}

// KeycloakInitiate handles GET /auth/keycloak. It redirects the browser to
// Keycloak's authorization endpoint, starting the authorization-code flow.
func (h *AuthHandler) KeycloakInitiate(w http.ResponseWriter, r *http.Request) {
	if !h.keycloakEnabled() {
		presenter.Error(w, r, apierr.New(apierr.CodeBadRequest, "keycloak login is not configured"))
		return
	}

	state, err := randomState()
	if err != nil {
		presenter.Error(w, r, apierr.New(apierr.CodeInternalError, "failed to start keycloak login"))
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     keycloakStateCookie,
		Value:    state,
		Path:     keycloakStatePath,
		HttpOnly: true,
		Secure:   h.cookie.Secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(keycloakStateTTL.Seconds()),
	})

	q := url.Values{}
	q.Set("client_id", h.keycloak.ClientID)
	q.Set("redirect_uri", h.keycloakRedirectURI())
	q.Set("response_type", "code")
	q.Set("scope", "openid profile roles")
	q.Set("state", state)

	http.Redirect(w, r, h.keycloakAuthURL()+"?"+q.Encode(), http.StatusFound)
}

// KeycloakCallback handles GET /auth/keycloak/callback. It exchanges the
// authorization code for tokens, finds-or-creates the local user account,
// syncs their role from Keycloak realm roles, and issues Paca session
// cookies exactly as Login does.
func (h *AuthHandler) KeycloakCallback(w http.ResponseWriter, r *http.Request) {
	if !h.keycloakEnabled() {
		presenter.Error(w, r, apierr.New(apierr.CodeBadRequest, "keycloak login is not configured"))
		return
	}

	stateCookie, err := r.Cookie(keycloakStateCookie)
	if err != nil || stateCookie.Value == "" || stateCookie.Value != r.URL.Query().Get("state") {
		presenter.Error(w, r, apierr.New(apierr.CodeBadRequest, "invalid or expired keycloak login state"))
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     keycloakStateCookie,
		Value:    "",
		Path:     keycloakStatePath,
		HttpOnly: true,
		Secure:   h.cookie.Secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	code := r.URL.Query().Get("code")
	if code == "" {
		presenter.Error(w, r, apierr.New(apierr.CodeBadRequest, "missing authorization code"))
		return
	}

	claims, err := h.exchangeKeycloakCode(r.Context(), code)
	if err != nil {
		presenter.Error(w, r, apierr.New(apierr.CodeUnauthenticated, "keycloak login failed"))
		return
	}
	if claims.PreferredUsername == "" {
		presenter.Error(w, r, apierr.New(apierr.CodeUnauthenticated, "keycloak identity missing preferred_username"))
		return
	}

	role := userdom.RoleUser
	for _, rn := range claims.RealmAccess.Roles {
		if rn == h.keycloak.AdminRole {
			role = userdom.RoleAdmin
			break
		}
	}

	u, err := h.findOrSyncKeycloakUser(r.Context(), claims, role)
	if err != nil {
		presenter.Error(w, r, apierr.New(apierr.CodeInternalError, "failed to provision keycloak user"))
		return
	}

	pair, err := h.svc.LoginAsUser(r.Context(), u.ID.String(), u.Username, u.Role, true)
	if err != nil {
		presenter.Error(w, r, err)
		return
	}

	h.setTokenCookies(w, pair, pair.RefreshTTL)
	http.Redirect(w, r, strings.TrimRight(h.publicURL, "/")+"/", http.StatusFound)
}

// exchangeKeycloakCode exchanges an authorization code for an access token
// and decodes its claims (without signature verification — see
// keycloakAccessClaims doc comment for why that is safe here).
func (h *AuthHandler) exchangeKeycloakCode(ctx context.Context, code string) (*keycloakAccessClaims, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", h.keycloak.ClientID)
	form.Set("client_secret", h.keycloak.ClientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", h.keycloakRedirectURI())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.keycloakTokenURL(), strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("keycloak: build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := h.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("keycloak: token request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var tok keycloakTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return nil, fmt.Errorf("keycloak: decode token response: %w", err)
	}
	if resp.StatusCode != http.StatusOK || tok.AccessToken == "" {
		return nil, fmt.Errorf("keycloak: token exchange failed: status=%d error=%s desc=%s", resp.StatusCode, tok.Error, tok.ErrorDesc)
	}

	return decodeJWTClaims(tok.AccessToken)
}

// decodeJWTClaims decodes (without verifying) the payload segment of a
// compact JWT into keycloakAccessClaims.
func decodeJWTClaims(token string) (*keycloakAccessClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("keycloak: malformed access token")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("keycloak: decode token payload: %w", err)
	}
	var claims keycloakAccessClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("keycloak: unmarshal token claims: %w", err)
	}
	return &claims, nil
}

// findOrSyncKeycloakUser finds the local user matching the Keycloak
// preferred_username, creating it on first login. On every login the role is
// re-synced to match the caller's current Keycloak realm roles.
func (h *AuthHandler) findOrSyncKeycloakUser(ctx context.Context, claims *keycloakAccessClaims, role string) (*userdom.User, error) {
	u, err := h.users.FindByUsername(ctx, claims.PreferredUsername)
	if err != nil {
		if !errors.Is(err, userdom.ErrNotFound) {
			return nil, err
		}

		fullName := claims.Name
		if fullName == "" {
			fullName = claims.PreferredUsername
		}
		pw, perr := randomState()
		if perr != nil {
			return nil, perr
		}
		return h.users.Create(ctx, userdom.CreateInput{
			Username: claims.PreferredUsername,
			// The password is never used for login (Keycloak owns the
			// credential) — a random value just satisfies the NOT NULL
			// password_hash column.
			Password: pw,
			FullName: fullName,
			Role:     role,
		})
	}

	if u.Role != role {
		return h.users.AdminUpdate(ctx, u.ID, userdom.AdminUpdateInput{Role: role})
	}
	return u, nil
}

// randomState returns a URL-safe random string suitable for OAuth2 state
// parameters and one-off placeholder passwords.
func randomState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
