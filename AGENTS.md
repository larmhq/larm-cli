# Agents

Instructions for Claude Code agents working on the larm-cli project.

## What is this?

`larm` is the open source CLI for the Larm uptime monitoring platform. It talks to the Larm REST API at `https://app.larm.dev/api/v1`.

## Tech stack

- **Go 1.26.1** (pinned in `mise.toml`)
- **Cobra** for command structure
- **oapi-codegen** for typed API client (generated from `api/openapi.yaml`)
- **Viper** for config management (XDG: `~/.config/larm/config.yml`)
- **gojq** for JQ filtering
- **go-isatty** for TTY detection

## Project structure

```
larm-cli/
  main.go                          # Entry point
  Makefile                         # build, test, lint, generate, check-generate, snapshot
  api/
    openapi.yaml                   # OpenAPI 3.1 spec (source for client generation)
  cmd/
    root.go                        # Root command + persistent flags
    helpers.go                     # Shared: newTypedClient, getOutputFlags, printOutput, resolveAuth, handleDelete, confirmAction
    view.go                        # View rendering: viewRow, writeView, field/colorField/staticField/separator helpers
    auth.go                        # auth login (device flow + --with-token), logout, status, token
    config.go                      # config get/set/list
    api.go                         # Raw API escape hatch
    describe.go                    # JSON schema for LLM agent discovery
    monitors.go                    # monitors list/show/create/update/delete/state/uptime/response-times/cert + view functions
    alerts.go                      # alerts list/show/create/update/delete + viewAlertChannel
    disruptions.go                 # disruptions list/show/create/update + viewDisruption (with timeline)
    status_pages.go                # status-pages list/show/create/update/delete + viewStatusPage
    components.go                  # components list/show/create/update/delete (--status-page-id) + viewComponent
    webhooks.go                    # webhooks list/show/create/update/delete + viewWebhook
  internal/
    api/
      client.go                    # Raw HTTP client for identity checks (context-aware)
      retry.go                     # RetryTransport: exponential backoff, Retry-After, body buffering
    auth/
      auth.go                      # Auth resolution: flag > env > config > OAuth token (auto-refresh)
    client/
      client.gen.go                # Generated oapi-codegen client (DO NOT EDIT)
      oapi_codegen_config.yml      # Generation config
    config/
      config.go                    # XDG config, key constants, key validation, 0600 permissions
    output/
      output.go                    # Table/JSON formatting, JQ, field selection, ResolveField, color, ViewFunc
      error.go                     # APIError with exit codes, structured JSON errors, suggestions, errors.As
      envelope.go                  # UnwrapData for {"data": ...} envelopes
  tools.go                         # oapi-codegen tool dependency
```

## Building and running

| Command | What it does |
|---------|-------------|
| `make build` | Build binary to `bin/larm` |
| `make run ARGS="monitors list"` | Build and run with args |
| `make test` | Run all tests |
| `make lint` | golangci-lint |
| `make fmt` | Format + goimports |
| `make install` | Install to GOPATH/bin |
| `make generate` | Regenerate typed client from OpenAPI spec |
| `make check-generate` | CI check: generated code matches committed |
| `make clean` | Remove build artifacts |
| `make snapshot` | Test goreleaser locally |

Always run `make build && make test && make lint` before considering work done. Always run `goimports -local github.com/larmhq/larm-cli -w cmd/ internal/` after editing Go files.

## How to add a new resource command

1. Check that the endpoints exist in `api/openapi.yaml` and the generated client has the methods.
2. Create `cmd/<resource>.go` following the pattern in `cmd/monitors.go`:
   - Define cobra commands with `Use`, `Short`, `Example`, `Args`, `RunE`
   - Use `newTypedClient(cmd)` to get the API client
   - Mark required flags with `MarkFlagRequired`
   - **List commands**: use `handleAndPrintWithDefaults` with default fields string and `ColorHints`
   - **Show/create/update commands**: use `handleAndPrintWithDefaults` with `output.PrintOpts{ViewFunc: viewMyResource}` -- write a view function using the `viewRow`/`writeView` pattern from `cmd/view.go`
   - **Delete commands**: use `confirmAction` before deleting, then `handleDelete`
3. Register commands in `init()` with `rootCmd.AddCommand()`

### View functions

Show commands and single-object responses (create, update) use custom view functions instead of the generic table renderer. This follows `gh`'s pattern -- view commands control field order, flatten nested data, and skip empty fields.

A view function has signature `func(io.Writer, json.RawMessage) error` and uses the declarative `viewRow` system from `cmd/view.go`:

```go
func viewMyResource(w io.Writer, data json.RawMessage) error {
    var m map[string]any
    if err := json.Unmarshal(data, &m); err != nil {
        return err
    }
    return writeView(w, m, []viewRow{
        field("id", "id"),
        field("name", "name"),
        colorField("status", "status", output.StatusColor),
        colorField("enabled", "enabled", output.BoolColor),
        staticField("items", joinStrings(m["items"], "(none)")),
        field("inserted_at", "inserted_at"),
    })
}
```

- `field(label, key)` -- resolves a dot-path from the JSON, humanizes timestamps
- `colorField(label, key, colorFunc)` -- same but applies color
- `staticField(label, value)` -- pre-computed value (for flattened/formatted data)
- `separator()` -- blank line between sections
- `joinStrings(v, fallback)` -- joins arrays into comma-separated strings
- Empty/nil values are automatically skipped
- JSON mode bypasses ViewFunc entirely -- raw API response

## How to regenerate the client

1. Update `api/openapi.yaml` (copy from the backend: `apps/backend/priv/openapi.yaml`)
2. Run `make generate`
3. Verify with `make build`

Note: the CLI spec has one intentional divergence from the backend spec -- `MonitorStateEnum` uses `nullable: true` (OpenAPI 3.0 style) instead of `type: [string, "null"]` (JSON Schema style) for oapi-codegen compatibility.

## Authentication

Three commands:
- **Device flow OAuth** (default `larm auth login`): opens browser, user approves, CLI gets token
- **API key** (`larm auth login --with-token`): paste key from stdin
- **Logout** (`larm auth logout`): clears all stored credentials

Switching auth methods clears the other -- `--with-token` clears OAuth tokens, device flow clears API keys.

Resolution order: `--api-key` flag > `LARM_API_KEY` env > config `api_key` > config `access_token` (auto-refresh)

OAuth tokens auto-refresh transparently when within 5 minutes of expiry.

## Output

Two rendering paths:
- **List commands**: generic table renderer with default columns and color hints
- **Show/detail commands**: custom `ViewFunc` per resource with controlled field order, flattened nested data, color

Both paths:
- TTY: human-readable (table or key-value)
- Piped: JSON (auto-detected)
- `--output table|json` to override
- `--fields name,id` for column selection (list commands)
- `--jq '.name'` for JSON filtering
- `--quiet` suppresses output on success

## Code conventions

- Every command should have an `Example` field
- Use `resolveAuth(cmd)` not `auth.Resolve` directly
- Use `handleAndPrintWithDefaults` for API responses (list + show)
- Use `handleDelete` + `confirmAction` for delete commands
- Show commands use `ViewFunc` with `writeView` -- never the generic table for single objects
- HTTP clients must have 30s timeout
- All HTTP requests must use `http.NewRequestWithContext` for cancellation
- JSON payloads must use `json.Marshal` -- never `fmt.Sprintf` with interpolated values
- Config keys must use constants from `config.Key*` -- never raw strings
- Config keys must be in `ValidKeys` whitelist
- Never edit `client.gen.go` manually
- Run `goimports -local github.com/larmhq/larm-cli -w cmd/ internal/` after editing
