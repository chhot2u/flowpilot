# Code Deletion Log

## [2026-03-25] Refactor Session — Frontend Dead Export Cleanup

### Unused Exports Removed

- `frontend/src/lib/step-actions.ts` — removed `loadSupportedStepActions()`.
  - Reason: The helper was defined but never imported or called anywhere; all live code paths use `ensureStepActionStateLoaded()` instead.

- `frontend/src/lib/step-actions.ts` — `SupportedStepAction` and `StepActionState` are no longer exported.
  - Reason: Both types were internal implementation details of the module with no external imports.

- `frontend/src/lib/types.ts` — removed unused interfaces: `PaginatedTasks`, `DOMSnapshot`, `DiffRequest`.
  - Reason: Exact reference checks plus `svelte-check` validation confirmed these interfaces had no live frontend imports or usages.

### Unused Dependencies Removed

- None.
  - Reason: `depcheck` flagged `@testing-library/svelte`, `jsdom`, `svelte-check`, and `typescript`, but they are used indirectly via scripts/tooling (`vitest`, `npm run check`, TypeScript/Svelte config) so they were retained.

### Duplicate Code Analysis

- None removed in this pass.
  - Reason: The investigation focused on low-risk dead exports; no fully duplicate frontend implementation was identified that could be consolidated safely without a broader refactor.

### Impact

- Files modified: 3
  - `frontend/src/lib/step-actions.ts`
  - `frontend/src/lib/types.ts`
  - `docs/DELETION_LOG.md`
- Dead exported symbols removed or internalized: 6
- Runtime behavior changes: none intended

### Testing

- Targeted frontend tests passing: `cd frontend && npm test -- --run src/lib/step-actions.test.ts src/lib/store.test.ts` — 36 tests passed.
- Frontend type and Svelte checks passing: `cd frontend && npm run check` — 0 errors, 0 warnings.

## [2026-03-25] Refactor Session — Internal API Export Cleanup

### Unused Exports Removed

- `frontend/src/main.ts` — removed the default export of the Svelte app instance.
  - Reason: The entry module is executed for side effects only; no module imported the default export.

### Unexported Internal-Only Symbols

- `app_proxy.go` — retained `App.AddProxyWithRateLimit()` as an exported API.
  - Reason: Follow-up audit found `frontend/src/components/ProxyPanel.svelte` still calls this runtime-exposed Wails method to preserve per-proxy request-rate limits. The earlier internalization was reverted to avoid a functional regression.

- `app.go` — `App.WatchConfig()` → `App.watchConfig()`
  - Reason: Only called internally during startup; not part of the Wails API surface.

- `app.go` — `App.GetTaskMetrics()` → `App.getTaskMetrics()`
  - Reason: Used only by internal Prometheus metrics generation, not by frontend bindings.

- `app_metrics_server.go` — `App.MetricsAddress()` → `App.metricsAddress()`
  - Reason: Used only by same-package tests; not exposed through Wails.

### Impact

- Files modified: 7
  - `frontend/src/main.ts`
  - `app_proxy.go`
  - `app_proxy_test.go`
  - `app.go`
  - `app_metrics.go`
  - `app_metrics_server.go`
  - `app_metrics_server_test.go`
- Dead exports removed or internalized: 4
- Runtime behavior changes: none intended

### Testing

- Go validation passing: `gofmt -w app.go app_metrics.go app_metrics_server.go app_proxy.go app_proxy_test.go app_metrics_server_test.go && go test -tags=dev ./...` — all packages OK.
- Targeted frontend tests passing: `cd frontend && npm test -- --run src/lib/step-actions.test.ts src/lib/store.test.ts` — 36 tests passed.
- Frontend type and Svelte checks passing: `cd frontend && npm run check` — 0 errors, 0 warnings.

## [2026-03-25] Refactor Session — Batch Helper Internalization

### Unexported Internal-Only Symbols

- `internal/batch/naming.go` — `DefaultNameTemplate()` → `defaultNameTemplate()`, `ValidateTemplate()` → `validateTemplate()`
  - Reason: Only used within the `batch` package and same-package tests.

- `internal/batch/variables.go` — `TemplateVars` → `templateVars`, `ApplyTemplate()` → `applyTemplate()`, `ExtractDomain()` → `extractDomain()`
  - Reason: Only used within the `batch` package and same-package tests.

### Impact

- Files modified: 4
  - `internal/batch/naming.go`
  - `internal/batch/variables.go`
  - `internal/batch/batch.go`
  - `internal/batch/batch_test.go`
- Dead exports removed or internalized: 5
- Runtime behavior changes: none intended

### Testing

- Go validation passing: `gofmt -w internal/batch/naming.go internal/batch/variables.go internal/batch/batch.go internal/batch/batch_test.go && go test -tags=dev ./internal/batch ./...` — `internal/batch` and all packages OK.

## [2026-03-25] Refactor Session — Validation Leaf Internalization

### Unexported Internal-Only Symbols

- `internal/validation/validate.go` — internalized package-private proxy leaf validators: `ValidateProxyServer()` → `validateProxyServer()`, `ValidateProxyProtocol()` → `validateProxyProtocol()`, `ValidateProxyFallback()` → `validateProxyFallback()`.
  - Reason: Exact reference checks plus Go validation confirmed these helpers are only called by higher-level validation functions inside the `validation` package and by same-package tests. `ValidatePriority()`, `ValidateTags()`, and `ValidateStatus()` were retained as exported because the app layer calls them directly.

### Impact

- Files modified: 2
  - `internal/validation/validate.go`
  - `internal/validation/validate_test.go`
- Dead exports removed or internalized: 3
- Runtime behavior changes: none intended

### Testing

- Go validation passing: `gofmt -w internal/validation/validate.go internal/validation/validate_test.go && go test -tags=dev ./internal/validation ./...` — `internal/validation` and all packages OK.

## [2026-03-24] Refactor Session — Dead Code & Unused Export Cleanup


### Unused Exports Removed

- `internal/models/task.go` — `TaskContext` struct and `NewTaskContext()` function
  - Reason: Defined but never referenced in any production code path. No callers outside the models package itself.

- `internal/models/batch.go` — `TemplateVariable` struct and `SupportedVariables()` function
  - Reason: Only consumed by tests (`models_test.go`). Not used in any production code path. The corresponding test was also removed.

- `internal/logs/export.go` — `Exporter.ExportTaskLogs()` method (non-zip, returning `(string, string, error)`)
  - Reason: `app_export.go` only calls `ExportTaskLogsZip`. The non-zip variant was dead code never reachable from the application layer. Its two tests (`TestExportTaskLogs`, `TestExportTaskLogsNoData`) were also removed.

### Unexported Internal-Only Symbols

- `internal/browser/browser.go` — `ErrEvalScriptTooLarge` → `errEvalScriptTooLarge`
  - Reason: Only used internally within `browser.go` (`validateEvalScript`). No callers in any other package. Unexported to match Go convention for package-private sentinels.

- `internal/browser/browser.go` — `ErrEvalScriptEmpty` → `errEvalScriptEmpty`
  - Reason: Same as above — only used internally within `validateEvalScript`. Unexported.

- `internal/browser/pool.go` — `PoolStats` → `poolStats`, `BrowserPool.Stats()` → `BrowserPool.stats()`
  - Reason: `Stats()` was only called from `pool_test.go` (same package). No external callers in application code. Unexported to reflect package-private use. Test updated accordingly.

### Duplicate Code Analysis

- `ErrEvalNotAllowed` is defined in both `internal/browser/browser.go` and `internal/validation/validate.go` with slightly different messages. Each is used exclusively within its own package (browser enforces it at execution time; validation enforces it at input validation time). These serve distinct layers and are intentionally separate — **not consolidated** as removing either would change semantics.

### Unused Dependencies Removed

- None. `go mod tidy` confirmed all module dependencies are transitively required.

### Unused Imports Removed

- None additional. `go vet` and `go build` confirmed all imports are in use.

### Impact

- Files modified: 7
  - `internal/models/task.go`
  - `internal/models/batch.go`
  - `internal/models/models_test.go`
  - `internal/logs/export.go`
  - `internal/logs/export_test.go`
  - `internal/browser/browser.go`
  - `internal/browser/pool.go`
  - `internal/browser/pool_test.go`
- Lines of production code removed: ~60
- Lines of test code removed: ~100

### Testing

- All tests passing: `go test -tags=dev ./...` — all 17 packages OK
- `go vet` clean: `go vet -tags=dev ./...` — no issues
- `go build` clean: `go build -tags=dev ./...` — no errors
