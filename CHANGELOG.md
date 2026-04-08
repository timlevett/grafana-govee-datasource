# Changelog

All notable changes to this project will be documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
This project does not yet use semantic versioning — version numbers will be
introduced when the plugin is submitted to the Grafana plugin catalog.

---

## [Unreleased]

### Added
- Docker Compose dev stack (`docker-compose.yml`) for zero-config local development.
- `applyTemplateVariables` implementation — dashboard variables (`$device`, `$metric`) now
  work in query editor fields.
- `CHANGELOG.md` (this file) required by the Grafana plugin catalog.

### Changed
- Removed misleading "Time Series" query-type option from the query editor.
  The Govee API only returns point-in-time snapshots; existing saved queries
  using `timeseries` continue to work without changes.
- Error messages from the Govee API are now classified and sanitised before
  being surfaced in the Grafana UI (HTTP 401 → API key hint, HTTP 429 → rate
  limit guidance, network errors → connectivity hint).

---

## [0.1.0] — 2026-04-08

### Added
- Initial production-ready release.
- Backend Go plugin: `QueryData`, `CheckHealth`, `CallResource` handlers.
- Frontend TypeScript/React: config editor (API key), query editor (device + metric selectors).
- Govee OpenAPI client with in-memory rate limiter (10,000 req/day) and 60-second TTL cache.
- Support for temperature, humidity, battery, power state, brightness, colour temperature,
  online status, and custom capability instances.
- Unit tests for Go backend and Jest tests for frontend components.
- CI workflow (GitHub Actions) for build, test, and lint.
