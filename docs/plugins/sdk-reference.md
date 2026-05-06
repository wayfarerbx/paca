# Plugin SDK Reference

The Paca Plugin SDK consists of two packages:

- **`@paca-ai/plugin-sdk-react`** — TypeScript/React SDK for frontend plugin components.
- **`github.com/Paca-AI/plugin-sdk`** — Go SDK for backend WASM plugins.

Both packages are maintained in `plugins/sdk/` within the monorepo.

See [paca-plugin-example](https://github.com/Paca-AI/paca-plugin-example) for a working reference implementation that exercises every API documented here.

---

## TypeScript SDK (`@paca-ai/plugin-sdk-react`)

### Installation

```sh
bun add @paca-ai/plugin-sdk-react
```

The package is also declared as a `shared` dependency in Module Federation config so the host's singleton instance is used (see [Frontend Plugin System](frontend-plugin-system.md)).

---

### `PluginSDK`

The main SDK object passed to every plugin component via its context prop.

```ts
interface PluginSDK {
  /** HTTP client scoped to /api/v1/plugins/{pluginId}/projects/{projectId}/ */
  api: PluginApiClient;

  /** UI utilities */
  ui: PluginUI;

  /** Metadata about this plugin and the host */
  meta: PluginMeta;
}
```

---

### `PluginApiClient`

The API client provides typed wrappers for common Paca API calls and plugin route helpers. The host creates and injects the instance — plugins must not construct their own.

```ts
class PluginApiClient {
  constructor(opts: PluginApiClientOptions)

  // Core read-only helpers (scoped to the current project)
  listTasks(filters?: TaskFilters): Promise<TaskSummary[]>
  getTask(taskId: string): Promise<Task>
  getProject(): Promise<ProjectSummary>
  listMembers(): Promise<ProjectMember[]>

  // Plugin route helpers (prefixed with /plugins/{pluginId}/projects/{projectId}/)
  pluginGet<T>(pluginId: string, path: string): Promise<T>
  pluginPost<T>(pluginId: string, path: string, body: unknown): Promise<T>
  pluginPatch<T>(pluginId: string, path: string, body: unknown): Promise<T>
  pluginDelete(pluginId: string, path: string): Promise<void>
}

interface PluginApiClientOptions {
  baseUrl: string;    // e.g. "https://app.paca.dev/api/v1"
  projectId: string;  // current project ID, injected by the host
  fetch: (url: string, init?: RequestInit) => Promise<Response>;
}
```

**Example:**
```ts
// Call a backend route registered by this plugin
const result = await api.pluginGet<{ data: MyItem[] }>(meta.pluginId, `tasks/${taskId}/items`);

// Fetch core platform data
const members = await api.listMembers();
const tasks = await api.listTasks({ status_ids: ["done"] });
```

---

### `PluginUI`

Utilities for showing UI feedback without depending on host internals.

```ts
interface PluginUI {
  /** Show a toast notification */
  toast(opts: ToastOptions): void;

  /** Show a confirmation dialog; resolves true if confirmed */
  confirm(opts: ConfirmOptions): Promise<boolean>;

  /** Navigate within the host application using its router */
  navigate(path: string): void;
}

interface ToastOptions {
  title: string;
  description?: string;
  variant?: "default" | "success" | "destructive";
  duration?: number;
}

interface ConfirmOptions {
  title: string;         // required — shown as the dialog heading
  description?: string;
  confirmLabel?: string;
  cancelLabel?: string;
  variant?: "default" | "destructive";
}
```

---

### `PluginMeta`

```ts
interface PluginMeta {
  pluginId: string;     // e.g. "com.paca.example"
  displayName: string;  // human-readable plugin name
  version: string;      // semver version string
}
```

---

### Extension Point Component Contracts

Each extension point has a typed React component interface exported from `@paca-ai/plugin-sdk-react`. All interfaces extend `BaseExtensionProps`, which injects `api`, `ui`, and `meta` as top-level props.

```ts
interface BaseExtensionProps {
  api: PluginApiClient;
  ui: PluginUI;
  meta: PluginMeta;
}
```

Your exported component **must** match the prop signature for its extension point.

#### `task.detail.section`

```ts
import type { TaskDetailSectionProps } from "@paca-ai/plugin-sdk-react";

export default function MyTaskDetailSection(props: TaskDetailSectionProps) { ... }

interface TaskDetailSectionProps extends BaseExtensionProps {
  taskId: string;
  projectId: string;
}
```

#### `sidebar.general.section`

```ts
import type { SidebarGeneralSectionProps } from "@paca-ai/plugin-sdk-react";

export default function MyGeneralSection(props: SidebarGeneralSectionProps) { ... }

interface SidebarGeneralSectionProps extends BaseExtensionProps {
  isCollapsed: boolean;
}
```

#### `sidebar.project.section`

```ts
import type { SidebarProjectSectionProps } from "@paca-ai/plugin-sdk-react";

export default function MyProjectSection(props: SidebarProjectSectionProps) { ... }

interface SidebarProjectSectionProps extends BaseExtensionProps {
  projectId: string;
  isCollapsed: boolean;
}
```

#### `project.settings.tab`

```ts
import type { ProjectSettingsTabProps } from "@paca-ai/plugin-sdk-react";

export default function MySettingsTab(props: ProjectSettingsTabProps) { ... }

interface ProjectSettingsTabProps extends BaseExtensionProps {
  projectId: string;
}
```

#### `view`

```ts
import type { ViewExtensionProps } from "@paca-ai/plugin-sdk-react";

export default function MyView(props: ViewExtensionProps) { ... }

interface ViewExtensionProps extends BaseExtensionProps {
  projectId: string;
  viewConfig?: Record<string, unknown>;
}
```

---

### Shared Types

```ts
// Task
interface TaskSummary {
  id: string;
  title: string;
  task_number: number;      // snake_case matching the API JSON
  status_id: string | null;
  assignee_id: string | null;
}

interface Task extends TaskSummary {
  project_id: string;
}

interface TaskFilters {
  status_ids?: string[];
  assignee_ids?: string[];
  sprint_id?: string;
  parent_task_id?: string;
  page?: number;
  page_size?: number;
}

// Project
interface ProjectSummary {
  id: string;
  name: string;
  description: string;
  task_id_prefix: string;
}

// Members
interface ProjectMember {
  id: string;
  username: string;
  full_name: string;
  role_name: string;
}

interface ProjectPermissions {
  canManageProject: boolean;
  canManageMembers: boolean;
  canWriteTasks: boolean;
  canReadTasks: boolean;
}
```

---

### React Query Integration

Plugins may use TanStack Query. The SDK exports `PluginQueryClientProvider` and `usePluginQuery` to namespace cache entries under the plugin ID so they cannot collide with the host or sibling plugins.

```ts
import {
  PluginQueryClientProvider,
  usePluginQuery,
  usePluginQueryClient,
} from "@paca-ai/plugin-sdk-react";

// Wrap your root component
export default function Root(props: TaskDetailSectionProps) {
  return (
    <PluginQueryClientProvider>
      <MyComponent {...props} />
    </PluginQueryClientProvider>
  );
}

// Use inside the provider — query key is prefixed with ["plugin", pluginId, ...]
function MyComponent({ api, meta, taskId }: TaskDetailSectionProps) {
  const { data, isLoading } = usePluginQuery(
    meta.pluginId,
    ["my-items", taskId],
    () => api.pluginGet<MyItem[]>(meta.pluginId, `tasks/${taskId}/items`),
  );
}

// Manual cache invalidation
function afterMutation(pluginId: string) {
  const qc = usePluginQueryClient();
  qc.invalidateQueries({ queryKey: ["plugin", pluginId, "my-items"] });
}
```

`PluginQueryClientProvider` accepts an optional `queryClient` prop to reuse the host's `QueryClient` instance. When running inside the host's Module Federation shell the host injects its client automatically.

---

## Go SDK (`github.com/Paca-AI/plugin-sdk`)

### Installation

```sh
go get github.com/Paca-AI/plugin-sdk
```

Build target must be `GOARCH=wasm GOOS=wasip1`. Standard Go 1.21+ WASI preview 1 is supported; TinyGo produces smaller binaries.

---

### Entry Point

Every plugin has a single Go entry file. The SDK exports all required WASM symbols internally — you only need to implement the `Plugin` interface and call `plugin.Run` from `init()`.

```go
//go:build wasip1

package main

import plugin "github.com/Paca-AI/plugin-sdk"

type myPlugin struct {
    db  *plugin.DB
    kv  *plugin.KV
    log *plugin.Logger
    cfg *plugin.Config
}

func (p *myPlugin) Init(ctx *plugin.Context) error {
    // Store handles for use in route/event handlers
    p.db  = ctx.DB()
    p.kv  = ctx.KV()
    p.log = ctx.Log()
    p.cfg = ctx.Config()

    ctx.Route("GET",    "/tasks/:taskId/items",     p.listItems)
    ctx.Route("POST",   "/tasks/:taskId/items",     p.createItem)
    ctx.Route("DELETE", "/tasks/:taskId/items/:id", p.deleteItem)
    ctx.On("task.deleted", p.onTaskDeleted)
    return nil
}

func (p *myPlugin) Shutdown() {}

func init() { plugin.Run(&myPlugin{}) }
func main() {}
```

---

### `plugin.Plugin` Interface

```go
type Plugin interface {
    // Init is called once when the plugin is loaded.
    // Register all routes and event handlers here.
    Init(ctx *Context) error

    // Shutdown is called before the plugin is unloaded.
    Shutdown()
}
```

---

### `plugin.Context`

Passed to `Init`. Provides route/event registration and access to host services.

```go
// Route registers an HTTP handler for method + path.
// method must be one of: GET, POST, PUT, PATCH, DELETE
// path may contain named segments, e.g. /tasks/:taskId/items/:id
func (c *Context) Route(method, path string, handler RouteHandler)

// On subscribes to a platform event topic.
// topic must be declared in plugin.json under backend.eventSubscriptions.
func (c *Context) On(topic string, handler EventHandler)

// Host service accessors — store these on your plugin struct during Init.
func (c *Context) DB()     *DB
func (c *Context) KV()     *KV
func (c *Context) Log()    *Logger
func (c *Context) Config() *Config
```

---

### `plugin.RouteHandler`

```go
type RouteHandler func(req *Request, resp *Response)

type Request struct {
    Method  string
    Path    string
    Headers map[string]string
    Body    []byte
    Caller  CallerIdentity
}

type CallerIdentity struct {
    CallerID   string
    CallerRole string
    ProjectID  string
}

// PathParam returns the named path parameter from the route pattern.
func (r *Request) PathParam(name string) string

// QueryParam returns the named URL query parameter.
func (r *Request) QueryParam(name string) string

func (r *Response) JSON(status int, body any)
func (r *Response) Text(status int, text string)
func (r *Response) NoContent()
func (r *Response) Error(status int, message string)
```

#### `plugin.JSONBody`

Decodes the JSON request body into a typed value:

```go
func JSONBody[T any](req *Request) (T, error)
```

**Example:**

```go
body, err := plugin.JSONBody[struct {
    Title string `json:"title"`
}](req)
if err != nil || body.Title == "" {
    resp.Error(400, "title is required")
    return
}
```

---

### `plugin.EventHandler`

```go
type EventHandler func(event *Event)

type Event struct {
    Topic   string // e.g. "task.deleted"
    Payload []byte // raw JSON bytes
}
```

#### `plugin.JSONPayload`

Decodes the event payload into a typed value:

```go
func JSONPayload[T any](evt *Event) (T, error)
```

**Example:**

```go
func (p *myPlugin) onTaskDeleted(evt *plugin.Event) {
    payload, err := plugin.JSONPayload[struct {
        TaskID string `json:"task_id"`
    }](evt)
    if err != nil {
        p.log.Error("bad payload")
        return
    }
    p.db.Exec(`DELETE FROM my_items WHERE task_id = $1`, payload.TaskID)
}
```

#### `plugin.EmitEvent`

Emits a custom event that other plugins or the platform can subscribe to:

```go
func EmitEvent(topic string, payload any)
```

---

### `plugin.DB`

Provides raw parameterised SQL access to the plugin's own tables. Plugins may also read core platform tables.

```go
// Query executes a SELECT (or INSERT … RETURNING) and returns the result rows.
func (d *DB) Query(sql string, params ...any) (*DBQueryResult, error)

// Exec executes an INSERT, UPDATE, or DELETE and returns the affected row count.
func (d *DB) Exec(sql string, params ...any) (int64, error)

type DBQueryResult struct {
    Columns []string
    Rows    [][]any
}
```

**Example:**

```go
// SELECT with a WHERE clause
result, err := p.db.Query(
    `SELECT id, title FROM my_items WHERE task_id = $1`,
    taskID,
)

// INSERT with RETURNING
result, err := p.db.Query(
    `INSERT INTO my_items (task_id, title) VALUES ($1, $2) RETURNING id, title`,
    taskID, title,
)

// DELETE
_, err = p.db.Exec(
    `DELETE FROM my_items WHERE id = $1`,
    itemID,
)
```

---

### `plugin.KV`

Per-plugin persistent key-value store backed by the platform database. Values are plain strings.

```go
func (kv *KV) Get(key string) (value string, ok bool)
func (kv *KV) Set(key, value string)
func (kv *KV) Delete(key string)
```

**Example:**

```go
// Increment a counter
count := 0
if v, ok := p.kv.Get("item.count"); ok {
    fmt.Sscanf(v, "%d", &count)
}
count++
p.kv.Set("item.count", fmt.Sprintf("%d", count))
```

---

### `plugin.Logger`

```go
func (l *Logger) Debug(msg string)
func (l *Logger) Info(msg string)
func (l *Logger) Warn(msg string)
func (l *Logger) Error(msg string)
```

---

### `plugin.Config`

Read-only access to operator-supplied configuration values for this plugin.

```go
func (c *Config) Get(key string) (value string, ok bool)
```

**Example:**

```go
prefix, _ := p.cfg.Get("greeting.prefix")
if prefix == "" {
    prefix = "Hello"
}
```

---

### Full Plugin Example (Go)

```go
//go:build wasip1

package main

import (
    "fmt"
    plugin "github.com/Paca-AI/plugin-sdk"
)

type examplePlugin struct {
    db  *plugin.DB
    kv  *plugin.KV
    log *plugin.Logger
    cfg *plugin.Config
}

func (p *examplePlugin) Init(ctx *plugin.Context) error {
    p.db  = ctx.DB()
    p.kv  = ctx.KV()
    p.log = ctx.Log()
    p.cfg = ctx.Config()

    ctx.Route("GET",    "/tasks/:taskId/messages",     p.listMessages)
    ctx.Route("POST",   "/tasks/:taskId/messages",     p.createMessage)
    ctx.Route("DELETE", "/tasks/:taskId/messages/:id", p.deleteMessage)
    ctx.On("task.deleted", p.onTaskDeleted)
    return nil
}

func (p *examplePlugin) Shutdown() {}

func (p *examplePlugin) listMessages(req *plugin.Request, resp *plugin.Response) {
    taskID := req.PathParam("taskId")
    result, err := p.db.Query(
        `SELECT id, name, message FROM hello_messages WHERE task_id = $1`,
        taskID,
    )
    if err != nil {
        p.log.Error("listMessages query failed")
        resp.Error(500, "query failed")
        return
    }
    resp.JSON(200, result)
}

func (p *examplePlugin) createMessage(req *plugin.Request, resp *plugin.Response) {
    body, err := plugin.JSONBody[struct {
        Name    string `json:"name"`
        Message string `json:"message"`
    }](req)
    if err != nil || body.Name == "" {
        resp.Error(400, "name is required")
        return
    }

    prefix, _ := p.cfg.Get("greeting.prefix")
    if prefix == "" {
        prefix = "Hello"
    }
    greeting := fmt.Sprintf("%s, %s! %s", prefix, body.Name, body.Message)

    result, err := p.db.Query(
        `INSERT INTO hello_messages (task_id, name, message)
         VALUES ($1, $2, $3) RETURNING id, name, message`,
        req.PathParam("taskId"), body.Name, greeting,
    )
    if err != nil {
        resp.Error(500, "insert failed")
        return
    }
    resp.JSON(201, result)
}

func (p *examplePlugin) deleteMessage(req *plugin.Request, resp *plugin.Response) {
    _, err := p.db.Exec(
        `DELETE FROM hello_messages WHERE id = $1`,
        req.PathParam("id"),
    )
    if err != nil {
        resp.Error(500, "delete failed")
        return
    }
    resp.NoContent()
}

func (p *examplePlugin) onTaskDeleted(evt *plugin.Event) {
    payload, err := plugin.JSONPayload[struct {
        TaskID string `json:"task_id"`
    }](evt)
    if err != nil {
        p.log.Error("bad task.deleted payload")
        return
    }
    p.db.Exec(`DELETE FROM hello_messages WHERE task_id = $1`, payload.TaskID)
}

func init() { plugin.Run(&examplePlugin{}) }
func main() {}
```

---

### Unit Testing with `plugintest`

The `plugintest` package (part of the SDK) provides in-memory backends so you can test route and event handlers without a live database or WASM runtime.

```go
package main_test

import (
    "encoding/json"
    "testing"

    plugin "github.com/Paca-AI/plugin-sdk"
    "github.com/Paca-AI/plugin-sdk/plugintest"
)

func TestListMessages(t *testing.T) {
    tc := plugintest.NewContext(t)

    // Seed initial data
    tc.DB.SeedRows("hello_messages",
        []string{"id", "task_id", "name", "message"},
        [][]any{
            {"id-1", "task-a", "Alice", "Hello, Alice!"},
        },
    )

    // Set config values the plugin reads during Init
    tc.Config.Set("greeting.prefix", "Hi")

    // Init the plugin
    var p examplePlugin
    if err := p.Init(tc.PluginContext()); err != nil {
        t.Fatal(err)
    }

    // Call a route
    res := tc.Call("GET", "/tasks/:taskId/messages", plugintest.Request{
        PathParams: map[string]string{"taskId": "task-a"},
        Caller:     plugin.CallerIdentity{ProjectID: "proj-1"},
    })
    if res.StatusCode != 200 {
        t.Fatalf("expected 200, got %d: %s", res.StatusCode, res.BodyString())
    }

    // Dispatch a platform event
    payload, _ := json.Marshal(map[string]string{"task_id": "task-a"})
    plugin.DispatchEvent(tc.PluginContext(), "task.deleted", payload)

    if rows := tc.DB.AllRows("hello_messages"); len(rows) != 0 {
        t.Fatalf("expected rows deleted, got %d", len(rows))
    }
}
```

Key `plugintest` API:

| Symbol | Description |
|---|---|
| `plugintest.NewContext(t)` | Create a fresh test harness; cleanup registered automatically. |
| `tc.DB.SeedRows(table, cols, rows)` | Pre-populate an in-memory table. |
| `tc.DB.AllRows(table)` | Read all rows after mutations. |
| `tc.KV.Set(key, value)` | Pre-seed KV entries. |
| `tc.Config.Set(key, value)` | Pre-seed config values. |
| `tc.PluginContext()` | Return `*plugin.Context` to pass to `Plugin.Init`. |
| `tc.Call(method, path, req)` | Dispatch a test request; returns `*plugin.Response`. |
| `plugin.DispatchEvent(ctx, topic, payload)` | Fire a platform event directly to the plugin's handler. |
