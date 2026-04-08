# CLAUDE.md — AI Agent Guide for grafana-govee-datasource

This file is specifically for AI coding agents. It documents project structure,
build commands, conventions, and gotchas to make agentic work reliable.

---

## Project overview

A Grafana backend datasource plugin that connects to the Govee smart home
OpenAPI. The plugin has two main components:

1. **TypeScript/React frontend** (`src/`) — query editor, config editor,
   datasource class. Runs in the browser.
2. **Go backend** (`pkg/`) — proxies all Govee API calls server-side so the
   API key is never exposed to the browser.

---

## Key commands

| Task | Command |
|------|---------|
| Install Node deps | `npm ci` |
| Build frontend | `npm run build` |
| Dev (watch mode) | `npm run dev` |
| Type-check | `npm run typecheck` |
| Lint frontend | `npm run lint` |
| Fix lint issues | `npm run lint:fix` |
| Run Jest tests | `npm test` |
| Build Go backend | `go build ./...` |
| Run Go tests | `go test ./...` |
| Run Go linter | `golangci-lint run ./...` |
| Full build | `make build` |
| Full test | `make test` |
| Clean | `make clean` |

---

## Key files

| File | Purpose |
|------|---------|
| `plugin.json` | Plugin manifest — ID, executable name, backend flag |
| `src/types.ts` | All TypeScript types: query model, device, capability, datasource options |
| `src/datasource.ts` | Frontend DataSource class (extends DataSourceWithBackend) |
| `src/module.ts` | Plugin entry point — registers DataSource, ConfigEditor, QueryEditor |
| `src/components/ConfigEditor.tsx` | Datasource settings UI (API key input) |
| `src/components/QueryEditor.tsx` | Panel query editor (device + metric selectors) |
| `pkg/main.go` | Go entry point — calls `datasource.Manage` |
| `pkg/plugin/datasource.go` | QueryData, CheckHealth, CallResource handlers |
| `pkg/plugin/govee.go` | Govee API client (ListDevices, QueryDeviceState, rate limiter) |
| `pkg/models/models.go` | Shared models (PluginSettings, QueryModel) |

---

## Security — API key handling

**CRITICAL**: The Govee API key must NEVER appear in:
- Frontend TypeScript/JavaScript code
- Browser network responses
- Log output
- Git history

How the plugin enforces this:
1. The API key is entered in `secureJsonData.apiKey` in the config editor
   (SecretInput component — write-only, never read back by the browser).
2. Grafana encrypts `secureJsonData` at rest and only provides it to backend
   plugins via `DecryptedSecureJSONData` in `backend.DataSourceInstanceSettings`.
3. The Go backend reads the key via `models.APIKey(settings)` and passes it
   directly to HTTP headers — never stores it in struct fields or logs.
4. `CallResource` (the `/devices` endpoint) fetches devices server-side and
   returns only the device list — the API key is not in the response.

---

## Govee API notes

- Base URL: `https://openapi.api.govee.com`
- Auth header: `Govee-API-Key: <your-key>`
- Rate limit: **10,000 requests/day** per account (tracked in-memory in the Go backend)
- Key endpoints:
  - `GET /router/api/v1/user/devices` — list devices + capabilities
  - `POST /router/api/v1/device/state` — query current device state
  - `POST /router/api/v1/device/control` — send commands to devices
- The device state API returns a `capabilities` array; each entry has `instance`
  (e.g. `"temperature"`) and `state` (a number, string, bool, or nested object).
- The plugin surfaces capability instances as "metrics" in the query editor.

### Rate limit gotcha

The rate limiter resets its counter at midnight UTC. It is **in-memory only**,
so Grafana restarts reset the counter. This means:
- Multiple Grafana instances sharing one API key will each have their own counter.
- Do not set very short dashboard refresh intervals (e.g. 1s) if you have many
  panels — you'll burn through the daily quota quickly.

---

## Grafana plugin SDK patterns

### Instance management

`plugin.NewGoveeDatasource` is registered with `datasource.Manage`. Grafana
calls it once per datasource instance (and again when settings change).
The returned `GoveeDatasource` struct must satisfy:
- `backend.QueryDataHandler`
- `backend.CheckHealthHandler`
- `backend.CallResourceHandler`
- `instancemgmt.InstanceDisposer`

### Data frames

The Govee state API returns point-in-time values (not historical time series).
The plugin emits a single data point at `time.Now()` per query. This is correct
for Stat, Gauge, and Table panels. For Time Series panels, users should set
the dashboard refresh interval to collect historical data in the Grafana time
series database (e.g. using Grafana Alloy or a recording rule).

### CallResource vs query

- `CallResource` (`/devices`) is used by the frontend to populate the device
  dropdown without running a full Grafana query. It is called via
  `getBackendSrv().fetch(...)` pointing at `<datasourceUrl>/resources/devices`.
- `QueryData` is called by Grafana when a panel renders or refreshes.

---

## Common development gotchas

1. **Backend binary name must match `plugin.json` `executable`**: The executable
   is `gpx_govee_datasource`. Grafana will not start the backend if this doesn't
   match.

2. **Unsigned plugin**: For local development, add
   `GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS=timlevett-govee-datasource`
   to your Grafana config or `.env`. See `.env.example`.

3. **Plugin in the right place**: Grafana looks for plugins in its plugin
   directory. Either copy `dist/` there or symlink it. The directory name must
   match the plugin ID: `timlevett-govee-datasource/`.

4. **Go module path**: The module is `github.com/timlevett/grafana-govee-datasource`.
   If you fork and rename, update `go.mod` and all import paths.

5. **`go.sum` must be committed**: Run `go mod tidy` after adding or removing
   dependencies. Both `go.mod` and `go.sum` should be in version control.

6. **Webpack externals**: Grafana's frontend externals (`@grafana/ui`,
   `@grafana/data`, `@grafana/runtime`, `react`, etc.) are loaded by Grafana
   at runtime — they must NOT be bundled. The webpack config lists them under
   `externals`. If you add a new Grafana package dependency, add it there too.

7. **`secureJsonFields` vs `secureJsonData`**: After saving the datasource,
   Grafana sets `secureJsonFields.apiKey = true` but clears `secureJsonData.apiKey`
   from the browser response. The `SecretInput` component shows a "configured"
   state and a reset button when `secureJsonFields.apiKey` is true.

---

## Adding a new metric / capability

1. Add the instance name to the `MetricInstance` type in `src/types.ts`.
2. Add a `SelectableValue` entry to `METRIC_OPTIONS` in `src/types.ts`.
3. If the value requires special parsing (e.g. a nested object), add a case
   to `toFloat64()` in `pkg/plugin/datasource.go`.

## Adding a new Govee API endpoint

1. Add the client method to `pkg/plugin/govee.go` (following the pattern of
   `ListDevices` / `QueryDeviceState`).
2. If it needs a new resource path, add a case to `CallResource` in
   `pkg/plugin/datasource.go`.
3. If it needs a new query type, add it to `QueryType` in
   `pkg/models/models.go` and `src/types.ts`, and handle it in `runQuery`.
