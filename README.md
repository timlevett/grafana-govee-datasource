# Govee Datasource for Grafana

A production-ready Grafana backend datasource plugin that connects your Grafana
dashboards to the [Govee](https://www.govee.com/) smart home OpenAPI.

Monitor temperature, humidity, battery level, power state, brightness, and
other sensor readings from your Govee devices in real time — displayed on
Grafana Stat, Gauge, Time Series, and Table panels.

---

## Features

- **Secure API key handling** — your Govee API key is stored encrypted in
  Grafana's secret storage and processed exclusively in the Go backend. It is
  never sent to the browser.
- **Device auto-discovery** — the query editor automatically fetches your
  registered Govee devices and presents them in a dropdown.
- **Capability-aware metric selector** — the plugin filters the metric list to
  only the capabilities your device actually supports.
- **Multiple metrics** — temperature, humidity, battery, power state,
  brightness, colour temperature, online status, and any custom capability
  instance.
- **Rate limit awareness** — in-memory daily request counter with automatic
  midnight UTC reset (10,000 req/day Govee limit).
- **Compatible with Grafana 10+**

---

## Prerequisites

| Requirement | Version |
|-------------|---------|
| Grafana | >= 10.0.0 |
| Node.js | >= 20 |
| Go | >= 1.21 |
| A Govee account | — |
| A Govee API key | — |

---

## Getting a Govee API key

1. Open the **Govee Home** mobile app (iOS or Android).
2. Go to **Profile** (bottom-right) → **Settings** (gear icon, top-right).
3. Tap **Apply for API Key**.
4. Fill in the form; the key is emailed to you within minutes.

The key looks like: `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`

> **Rate limit:** 10,000 API requests per day per account. With a typical
> Grafana dashboard refreshing every 30 seconds and 10 devices, you'll use
> ~28,800 requests/day — consider lengthening refresh intervals or reducing
> panel count if you have many devices.

---

## Installation

### Option A: Build from source

```bash
git clone https://github.com/timlevett/grafana-govee-datasource.git
cd grafana-govee-datasource

# Install Node dependencies and build frontend
npm ci
npm run build

# Build the Go backend binary
go build -o dist/gpx_govee_datasource ./pkg/main.go

# Copy the dist/ folder to Grafana's plugin directory
# (default on Linux: /var/lib/grafana/plugins/)
cp -r dist/ /var/lib/grafana/plugins/timlevett-govee-datasource/
```

For unsigned plugin development, add to your `grafana.ini` or environment:
```ini
[plugins]
allow_loading_unsigned_plugins = timlevett-govee-datasource
```

Restart Grafana, then navigate to **Configuration → Plugins** to enable it.

### Option B: Grafana Plugin Catalog

Once published, search for "Govee" in the Grafana plugin catalog.

---

## Configuration

1. In Grafana, go to **Configuration → Data Sources → Add data source**.
2. Search for "Govee" and click it.
3. Paste your Govee API key in the **Govee API Key** field.
4. Optionally override the **API Base URL** (default: `https://openapi.api.govee.com`).
5. Click **Save & Test** — you should see a success message listing your device count.

---

## Usage

### Creating a panel

1. Create or open a dashboard and add a panel.
2. Select your Govee datasource.
3. In the query editor:
   - **Query Type**: `Current State` (instant value) or `Time Series`.
   - **Device**: select from the dropdown (devices are fetched from your account).
   - **Metric**: choose the capability to monitor (e.g. Temperature, Humidity).
4. Click **Apply**.

### Recommended panel types

| Metric | Panel type |
|--------|-----------|
| Temperature | Stat, Gauge, Time Series |
| Humidity | Stat, Gauge |
| Battery | Gauge, Bar Gauge |
| Power State | Stat |
| Brightness | Gauge |
| Online Status | Stat |

### Custom capability instances

Select **Custom (enter instance)** in the metric dropdown and type the
capability `instance` string from the Govee API response directly (e.g.
`sensorTemperature`, `sensorHumidity`). This is useful for newer devices whose
capabilities are not yet listed in the plugin defaults.

---

## Development

### Setup

```bash
git clone https://github.com/timlevett/grafana-govee-datasource.git
cd grafana-govee-datasource

# Install Node deps
npm ci

# Copy environment template
cp .env.example .env
# Edit .env — set GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS etc.
```

### Frontend development

```bash
# Start webpack in watch mode
npm run dev

# In a separate terminal: start Grafana (with plugin path pointing to dist/)
```

### Backend development

```bash
# Build the backend binary (host OS)
go build -o gpx_govee_datasource ./pkg/main.go

# Cross-compile for Linux amd64
GOOS=linux GOARCH=amd64 go build -o gpx_govee_datasource_linux_amd64 ./pkg/main.go
```

### Running tests

```bash
# Frontend (Jest)
npm test

# Frontend with coverage
npm test -- --coverage

# Backend (Go)
go test ./...

# Backend with race detector
go test -race ./...
```

### Linting

```bash
# Frontend
npm run lint
npm run lint:fix

# Backend
golangci-lint run ./...

# Type-check (no emit)
npm run typecheck
```

### Full build

```bash
make build
```

---

## Project structure

```
grafana-govee-datasource/
├── .github/
│   └── workflows/
│       └── ci.yml          # CI: build, test, lint (frontend + backend)
├── pkg/
│   ├── main.go             # Go entry point
│   ├── models/
│   │   └── models.go       # Shared data models (QueryModel, PluginSettings)
│   └── plugin/
│       ├── datasource.go   # QueryData, CheckHealth, CallResource
│       └── govee.go        # Govee API client + rate limiter
├── src/
│   ├── __mocks__/          # Jest mocks for Grafana packages
│   ├── components/
│   │   ├── ConfigEditor.tsx
│   │   └── QueryEditor.tsx
│   ├── img/
│   │   └── logo.svg
│   ├── datasource.ts       # Frontend DataSource class
│   ├── module.ts           # Plugin registration
│   └── types.ts            # TypeScript type definitions
├── .env.example
├── .eslintrc.js
├── .gitignore
├── CLAUDE.md               # AI agent guide
├── Makefile
├── go.mod
├── jest.config.js
├── package.json
├── plugin.json             # Grafana plugin manifest
├── tsconfig.json
└── webpack.config.ts
```

---

## Architecture: API key security

```
Browser (Grafana UI)
    │
    │  secureJsonData.apiKey  ──►  Grafana DB (encrypted)
    │                                    │
    │  CallResource /devices  ──►  Go backend  ──►  Govee API (with Govee-API-Key header)
    │                                    │
    │  QueryData              ──►  Go backend  ──►  Govee API (with Govee-API-Key header)
    │                                    │
    │◄─────────── Data frames (numbers/strings only, NO key) ─────────────
```

The API key travels: Browser → Grafana server → Go plugin binary → Govee API.
It is **never** returned to the browser in any response.

---

## Govee API reference

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/router/api/v1/user/devices` | GET | List all registered devices |
| `/router/api/v1/device/state` | POST | Query current device state |
| `/router/api/v1/device/control` | POST | Send control commands |

All requests require the `Govee-API-Key` header.

---

## Contributing

Pull requests are welcome. Please:

1. Fork the repo and create a branch from `main`.
2. Run `make test` and `make lint` — both must pass.
3. Add tests for new functionality.
4. Keep the API key security model intact (see CLAUDE.md for details).
5. Open a PR with a clear description.

---

## License

[Apache 2.0](LICENSE)
