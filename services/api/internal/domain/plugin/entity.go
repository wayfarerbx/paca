// Package plugindom defines the plugin aggregate and its domain contracts.
// Each installed plugin has a manifest that describes its routes, extension
// points, and event subscriptions.  System-wide extension settings managed
// by the super admin control ordering and visibility for all users.
package plugindom

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Plugin represents one installed plugin in the registry.
type Plugin struct {
	ID          uuid.UUID
	Name        string         // reverse-DNS id, e.g. "com.paca.checklist"
	Version     string         // semver, e.g. "1.0.0"
	Manifest    PluginManifest // parsed plugin.json
	Enabled     bool
	InstalledAt time.Time
	UpdatedAt   time.Time
}

// PluginManifest is the structured content of a plugin's plugin.json file.
// Fields not used by the core runtime are stored as-is in the JSONB column.
type PluginManifest struct {
	// ID is the reverse-DNS plugin identifier (must match Plugin.Name).
	ID string `json:"id"`
	// DisplayName is the human-readable name shown in the UI.
	DisplayName string `json:"displayName"`
	// Description is a short description of the plugin.
	Description string `json:"description,omitempty"`
	// Version is the semver version of the plugin.
	Version string `json:"version"`
	// Backend holds backend-specific manifest settings.
	Backend *BackendManifest `json:"backend,omitempty"`
	// Frontend holds frontend-specific manifest settings.
	Frontend *FrontendManifest `json:"frontend,omitempty"`
	// MCP holds MCP-server-specific manifest settings.
	MCP *MCPManifest `json:"mcp,omitempty"`
	// Permissions lists the host function scopes the plugin requires.
	Permissions []string `json:"permissions,omitempty"`
}

// MCPManifest describes the MCP (Model Context Protocol) side of the plugin.
// When present, the Paca MCP server loads the module at RemoteEntryURL at
// startup and merges the exported tools into the server's tool list.
type MCPManifest struct {
	// RemoteEntryURL is the URL to the plugin's MCP entry module.
	// The module must be a Node.js-compatible ESM bundle that exports a
	// PluginMCPEntry as its default export (see @paca-ai/plugin-sdk-mcp).
	RemoteEntryURL string `json:"remoteEntryUrl"`
}

// BackendManifest describes the backend (WASM) side of the plugin.
type BackendManifest struct {
	// Routes is the list of HTTP routes the plugin registers.
	// Each route is mounted at /api/v1/plugins/{pluginId}/projects/:projectId/{path}.
	Routes []PluginRoute `json:"routes,omitempty"`
	// EventSubscriptions lists the event topics the plugin subscribes to.
	EventSubscriptions []string `json:"eventSubscriptions,omitempty"`
	// AllowedOutboundDomains is the list of hostnames the plugin is permitted to
	// contact via paca.fetch.  Requests to unlisted domains are rejected.
	AllowedOutboundDomains []string `json:"allowedOutboundDomains,omitempty"`
	// AllowedConfigKeys is the list of host config keys the plugin may read via
	// paca.config_get. Keys not listed here are not exposed to the plugin.
	AllowedConfigKeys []string `json:"allowedConfigKeys,omitempty"`
}

// FrontendManifest describes the frontend (Module Federation) side of the plugin.
type FrontendManifest struct {
	// RemoteEntryURL is the URL to the Module Federation remote entry JS file.
	RemoteEntryURL string `json:"remoteEntryUrl,omitempty"`
	// ExtensionPoints is the list of extension points the plugin registers into.
	ExtensionPoints []ExtensionPointRegistration `json:"extensionPoints,omitempty"`
}

// PluginRoute defines a single HTTP route exposed by the plugin backend.
type PluginRoute struct {
	Method string `json:"method"` // GET | POST | PATCH | PUT | DELETE
	Path   string `json:"path"`   // relative path, e.g. "/items" or "/items/:id"
}

// ExtensionPointRegistration describes a frontend component registered into an
// extension point in the host application.
type ExtensionPointRegistration struct {
	// Point is the extension point identifier, e.g. "task.detail.section".
	Point string `json:"point"`
	// Component is the exported React component name from the remote entry.
	Component string `json:"component"`
	// Order is the default display order within the extension point.
	Order int `json:"order,omitempty"`
}

// MarshalManifest serialises the manifest to JSON bytes for JSONB storage.
func (m PluginManifest) MarshalManifest() ([]byte, error) {
	return json.Marshal(m)
}

// UnmarshalManifest parses raw JSONB bytes into a PluginManifest.
func UnmarshalManifest(data []byte) (PluginManifest, error) {
	var m PluginManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return PluginManifest{}, err
	}
	return m, nil
}

// PluginExtensionSetting stores system-wide ordering and visibility settings
// for a single extension point of a specific plugin.  These are configured
// by the super admin and apply to all users.
type PluginExtensionSetting struct {
	ID             uuid.UUID
	PluginID       uuid.UUID
	ExtensionPoint string
	Settings       ExtensionSettingData
	UpdatedAt      time.Time
}

// ExtensionSettingData is the structured content stored in the settings JSONB
// column for a plugin_extension_settings row.
type ExtensionSettingData struct {
	// Hidden controls whether the extension point registration is visible to
	// all users.  Defaults to false (visible).
	Hidden bool `json:"hidden"`
	// Order is the admin-chosen display order for this registration.
	// Lower values appear first.  A value of 0 means "use plugin default".
	Order int `json:"order,omitempty"`
}
