# Wails v2 → v3 Migration Estimate

> Effort estimate and work breakdown for migrating install-it from Wails v2.12.0 to Wails v3. Produced from a full audit of the app's Wails surface area plus verification of v3 API facts against the `wailsapp/wails` master branch. Companion to [BACKEND_I18N_HANDOFF.md](BACKEND_I18N_HANDOFF.md).

---

## 1. Current Wails surface (audited)

The business logic (`pkg/storage`, `matching`, `porter`, `update`, `sysinfo`, `cputemp`) is Wails-free. Everything the framework touches:

| Surface | Where | Notes |
|---|---|---|
| Bootstrap | `main.go` | `wails.Run`: 9 bound structs, `EnumBind` ×5 enum arrays, `OnStartup` hook, `Windows.WebviewBrowserPath`, asset embed |
| Go runtime calls | `app.go` | `OpenDirectoryDialog`, `OpenFileDialog`; direct `webviewloader.GetAvailableCoreWebView2BrowserVersionString` for WebView2 version display |
| Go→JS events | `pkg/execute/execute.go` | `EventsEmit("execute:exited", id, CommandResult)` — **the only event in the app**; porter progress is polling via bound `Progress()`, not events |
| Frontend bindings | ~20 files | imports from `@/wailsjs/go/...` and `@/wailsjs/go/models` |
| Frontend runtime | 4 components | `EventsOn` (CommandStatusModal), `BrowserOpenURL` + `Environment` (app-info), `Quit` (UpdateModal), `WindowReloadApp` (ProgressModal) |
| Test mocks | `frontend/src/__tests__/setup.ts` | mocks 8 wailsjs modules + runtime |
| Build | `wails.json`, `.github/workflows/build_and_release.yml` | pinned CLI from go.mod, `-upx`, **386 + amd64 matrix**, fixed WebView2 runtime downloaded into `internals/bin/WebView2` (consumed via `WebviewBrowserPath`) |

## 2. Verified v3 facts

Checked against `wailsapp/wails@master` source and CI:

- **Services replace Bind**: `application.New(application.Options{Services: []application.Service{application.NewService(...)}})`; procedural window creation.
- **Bindings**: `wails3 generate bindings` → `frontend/bindings/<go-import-path>/`, static-analyzer based, calls via numeric IDs (`$Call.ByID(...)`), runtime imported from `/wails/runtime.js`.
- **`WebviewBrowserPath` exists in v3** (`v3/pkg/application/application_options.go`) → fixed WebView2 runtime bundling carries over unchanged.
- **SingleInstance** available (`SingleInstanceOptions`) — app doesn't use it today; note it interacts with the self-updater's detached relaunch if ever adopted.
- **`EnumBind` is v2-only** — no v3 equivalent found. Replacement: hand-written TS constants module or small codegen script.
- **ErrorFormatter → MarshalError**: per-service `MarshalError func(err error) []byte` in ServiceOptions. The errcode design from BACKEND_I18N_HANDOFF.md survives as-is; frontend reads structured payload from rejection `.cause`.
- **⚠️ windows/386 absent from v3's cross-compile test matrix** (amd64/arm64 only, `.github/workflows/cross-compile-test-v3.yml`). Moot: x86 is being dropped at migration time (§3).

## 3. Gate decisions (before any code)

1. ~~x86 fate~~ **RESOLVED:** x86 support drops *together with* the v3 migration, not before. v2 keeps shipping x86 until then; removing the `386` leg from `.github/workflows/build_and_release.yml` (build + updater jobs) is part of task 10. The separate `install-it-updater` repo should drop its x86 target in the same release.
2. **v3 stability freeze.** v3 is pre-stable with API churn. Don't start until a stable tag exists that you're willing to pin; starting earlier means paying churn twice.

## 4. Work breakdown

| # | Task | Est |
|---|---|---|
| 1 | Spike on branch: bootstrap rewrite — `application.New`, services registration, window creation, startup hooks replacing `OnStartup` | 0.5–1d |
| 2 | Context injection refactor: `App.SetContext` / `CommandExecutor.SetContext` pattern → v3 service lifecycle | 2–4h |
| 3 | Dialogs: `SelectFolder`/`SelectFile` → v3 dialog API | 1–2h |
| 4 | Event rewire: Go emit (`execute:exited`) + JS subscribe via `@wailsio/runtime` | 1–2h |
| 5 | Regenerate bindings; rewrite ~20 frontend import sites; verify call signatures | 0.5–1d |
| 6 | Enums: TS constants module replacing EnumBind (5 enums, 27 values) | 2–4h |
| 7 | Error seam: `MarshalError` wiring + `codeOf` reads `.cause` | ~2h |
| 8 | Runtime calls: `BrowserOpenURL`, `Quit`, `WindowReloadApp`, `Environment` equivalents | 1–2h |
| 9 | Vitest setup mocks rewrite to new binding modules | 2–4h |
| 10 | Build system: Taskfile (wails3 init generates), `wails3 dev/build`, CI workflow updates; remove 386 leg from release matrix; UPX + WebView2 bundling steps mostly carry over | 0.5–1d |
| 11 | Regression pass: full driver install / porter import-export / updater flows on real hardware | 0.5–1d |

**Total: ~4–7 focused days (~1.5–2 calendar weeks with hardware-testing friction).**

## 5. Risk ranking

1. **v3 maturity / API churn** — rework risk scales with migration duration; freeze scope before starting.
2. Everything else is mechanical mapping — no architectural risk. Storage/porter/update layers and the errcode error-i18n system are version-agnostic by design.

## 6. What carries over unchanged

- All `pkg/` business logic except one `EventsEmit` call in `execute.go`
- SQLite schema/migrations, settings store, self-update rename trick (OS-level, not Wails)
- `pkg/errcode` + i18n keys + composable structure (only transport seam changes)
- WebView2 fixed-runtime bundling strategy (`WebviewBrowserPath` exists in v3)
