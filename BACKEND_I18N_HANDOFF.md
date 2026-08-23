# Backend Message i18n Unification — Handoff

> Handoff document capturing the full context, decisions, architecture, and migration plan for unifying how Go backend messages/errors reach the Vue frontend for translation. The CPU temperature feature follow-ups (#48, #49, #50) surfaced the problem; this doc is the reference for the unification work the author plans to do later.

---

## 1. Problem

The app has **21 backend→UI message touchpoints** across the Go↔Vue boundary, handled with three inconsistent ad-hoc patterns:

| Pattern | Where | Problem |
|---|---|---|
| RAW `err.toString()` / `reason.toString()` toast | DriverFormComponent, MatchRuleFormComponent, app-info.vue, UpdateModal.vue, porter Abort | English leaks to UI; no localization |
| Localized toast, Go error dropped (`catch(() => t('toastX'))`) | drivers/index, match-rules/index, App.vue, settings.vue, index.vue (matcher) | Loses the actual error; one generic key for all failures |
| **English substring-matching** (fragile, English-coupled) | ProgressModal.toastErrMsg (16-line `err.includes()` chain), CommandStatusModal, TaskStatus | Breaks when backend text changes; not localizable |

There is no global helper. The worst offender is `ProgressModal.vue:146-162` (`toastErrMsg`) which substring-matches Go error strings against English phrases (`"context canceled"`, `"cannot find the path"`, `"zip: not a valid zip file"`, …) to pick an i18n key, falling back to the raw error.

**Trigger:** CPU temperature follow-ups. `CPUTemperature()` returned `-1` as a failure sentinel (#50 fixed this to return an `error`), and the question of how to surface that error to the user (gray `-- °C` badge + toast) revealed the broader, app-wide inconsistency. The author plans to unify.

---

## 2. Decisions

| # | Decision | Rationale |
|---|---|---|
| 1 | **Backend emits a stable code; frontend owns localized text** | i18n lives on the frontend; backend must not format user-facing strings. |
| 2 | **The code IS the vue-i18n key** | Zero mapping tables; adding a new error forces both the code and the i18n entry (completeness). |
| 3 | **Prefix = severity** | `err*` → error toast (red), `warn*` → warning (amber), `msg*` → info, `toast*` → general. Drives `useBackendError` toast color and matches STYLE.md prefixes. |
| 4 | **Dynamic values via JSON-encoded params** in `.Error()` when present | Wails serializes errors as strings (`.Error()`); embed `{c,p}` JSON for parameterized messages; static errors stay as a plain code (no JSON overhead). |
| 5 | **Bound methods convert internal errors to code errors at the boundary** | Internal `fmt.Errorf("ctx: %w", inner)` is fine for dev logs; the Wails-bound return strips to the code so only the code crosses the bridge. |

Confirmed by the author that returning i18n keys (not English text) from the backend is the intended direction; this doc captures the canonical shape.

---

## 3. Architecture

### 3.1 Backend: `pkg/errcode`

New package:

```go
// pkg/errcode/errcode.go
package errcode

import "encoding/json"

// Error is a machine-readable error whose .Error() returns a stable code
// string that doubles as a frontend i18n key.
//
// The code prefix (err / warn / msg / toast) signals severity to the frontend.
// Params carries vue-i18n interpolation values when present. The inner Err
// is wrapped for Go-side dev logs only — it never crosses the Wails bridge
// (only .Error() does).
type Error struct {
    Code   string
    Params map[string]any
    Err    error
}

func (e *Error) Error() string {
    if len(e.Params) == 0 {
        return e.Code
    }
    b, _ := json.Marshal(struct {
        C string         `json:"c"`
        P map[string]any `json:"p"`
    }{e.Code, e.Params})
    return string(b)
}

func (e *Error) Unwrap() error { return e.Err }

func New(code string) *Error                      { return &Error{Code: code} }
func Newf(code string, params map[string]any) *Error { return &Error{Code: code, Params: params} }
func Wrap(code string, inner error) *Error       { return &Error{Code: code, Err: inner} }
```

**Boundary discipline (must document in package comment):** Wails-bound methods MUST return `*errcode.Error` as the top-level error. Internal helpers may wrap freely for logging, but the bound return converts to a clean code:

```go
// Internal — fine to wrap
func openPawnIO() error {
    return fmt.Errorf("open PawnIO device: %w", windowsErr)
}

// Bound — strips to code
func (i SysInfo) CPUTemperature() (float64, error) {
    if err := openPawnIO(); err != nil {
        log.Printf("cpu temp init: %v", err)   // dev log keeps the chain
        return 0, errCPUTempUnavailable          // bridge gets just the code
    }
}
```

**Sentinel codes are colocated with the emitting package** (not centralized in one file):

- `pkg/sysinfo/errors.go` — CPU temp codes
- `pkg/porter/errors.go` — import/export codes
- `pkg/execute/errors.go` — command executor codes
- `pkg/storage/errors.go` — CRUD codes
- `pkg/update/errors.go` — updater codes
- etc.

### 3.2 Contract

```
Go:  errcode.New("warnCPUTempUnavailable")
        ↓ .Error() = "warnCPUTempUnavailable"
Wails bridge: string "warnCPUTempUnavailable"
        ↓
Vue: t("warnCPUTempUnavailable") → localized message
```

No mapping tables. The code string is the key. Prefix drives severity:

| Prefix | Toast color | UI treatment |
|---|---|---|
| `err*` | `error` (red) | Error toast |
| `warn*` | `warning` (amber) | Warning toast or gray-state badge |
| `msg*` | `info` | Informational |
| `toast*` | `error` or `info` | General toast |

i18n keys follow STYLE.md (flat camelCase, `err*`/`warn*`/`toast*` prefixes).

### 3.3 Frontend: `useBackendError` composable

```ts
// frontend/src/composables/useBackendError.ts
import { useI18n } from 'vue-i18n'
import { useToast } from '#imports'

export function useBackendError() {
  const { t } = useI18n()
  const toast = useToast()

  function codeOf(err: unknown): { code: string; params?: Record<string, any> } {
    const s = String(err)
    if (s.startsWith('{')) {
      try {
        const { c, p } = JSON.parse(s)
        return { code: c, params: p }
      } catch {
        /* fallthrough */
      }
    }
    return { code: s }
  }

  function message(err: unknown): string {
    const { code, params } = codeOf(err)
    return params ? t(code, params) : t(code)
  }

  function toastError(err: unknown) {
    const code = codeOf(err).code
    const color = code.startsWith('warn') ? 'warning'
                : code.startsWith('msg')  ? 'info'
                : 'error'
    toast.add({ title: message(err), color })
  }

  return { message, toastError, codeOf }
}
```

---

## 4. Reference adopter: CPU temperature (first site)

The CPU temp failure path is the first real consumer of the new pattern. It validates the end-to-end shape: backend code → bridge string → frontend badge + i18n toast.

### 4.1 Backend

`pkg/sysinfo/errors.go`:
```go
var (
    errCPUTempUnavailable = errcode.New("warnCPUTempUnavailable")
    errCPUTempNoReadings  = errcode.New("warnCPUTempNoReadings")
    errCPUTempReadFailed  = errcode.New("warnCPUTempReadFailed")
)
```

`pkg/sysinfo/sysinfo.go` — `CPUTemperature()` returns the appropriate code error instead of the current generic `fmt.Errorf("cpu temperature unavailable")` from #50:

```go
func (i SysInfo) CPUTemperature() (float64, error) {
    if !cputemp.IsAvailable() {
        return 0, errCPUTempUnavailable
    }
    temps, err := cputemp.GetCPUTemperatures()
    if err != nil {
        return 0, errcode.Wrap("warnCPUTempReadFailed", err)
    }
    if len(temps) == 0 {
        return 0, errCPUTempNoReadings
    }
    // ... existing aggregation ...
}
```

### 4.2 Frontend

`frontend/src/pages/index.vue`:

```ts
const { message, toastError } = useBackendError()
// refs: cpuTemp: number | null, cpuTempError: string | null, tempToasted: boolean

// .then(t => { cpuTempError.value = null; cpuTemp.value = Math.round(t) })
// .catch(err => {
//   cpuTempError.value = message(err)
//   if (!tempToasted.value) { toastError(err); tempToasted.value = true }
//   // stop polling on permanent failure: timer = null
// })
```

Badge (kept as `UBadge` per #49):
```vue
<UBadge v-if="part === 'cpu' && (cpuTemp !== null || cpuTempError !== null)"
  size="sm" class="ms-1 rounded px-1.5 py-0 text-xs font-medium"
  :class="cpuTempError !== null
    ? 'bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-400'
    : (cpuTemp < 45 ? 'bg-apple-green-100 ...' : ...)">
  {{ cpuTempError !== null ? '--' : cpuTemp }}°C
</UBadge>
```

### 4.3 i18n

Add to `frontend/src/i18n/en.json` and `zh_Hant_HK.json`:

```json
"warnCPUTempUnavailable": "CPU temperature detection is not available.",
"warnCPUTempNoReadings":  "No CPU temperature readings.",
"warnCPUTempReadFailed":  "Failed to read CPU temperature: {detail}"
```

(Add `{detail}` interpolation only if backend params are wired for it.)

---

## 5. Inventory of current touchpoints

Full inventory produced by recon; abbreviated here. Categories:

| Current pattern | Sites | Migration |
|---|---|---|
| RAW `reason.toString()` toast | DriverFormComponent (×2), MatchRuleFormComponent (×2), app-info.vue (×2), UpdateModal.vue, ProgressModal Abort | Phase 3 |
| Localized toast, error dropped | drivers/index.vue (×3), match-rules/index.vue (×2), App.vue (×3), settings.vue, index.vue matcher | Phase 3 |
| English substring-match | ProgressModal.toastErrMsg (16 lines), CommandStatusModal abort, TaskStatus run | Phase 2 |
| Silent (state = null) | index.vue CPUTemperature | Reference adopter |
| Title localized + body RAW | system-utilities.vue (9×) | Phase 3 |

The substring-match sites are the priority — they are the worst offenders (English-coupled, break on backend text changes, and `ProgressModal.toastErrMsg` is 16 lines of `err.includes()`).

---

## 6. Migration phases

| Phase | Scope | Risk |
|---|---|---|
| **1. Scaffold** | Create `pkg/errcode` package + `useBackendError` composable. Add CPU-temp codes + i18n entries. Wire CPU temp as reference adopter. | None — zero behavior change outside CPU temp |
| **2. Worst offender** | Migrate `ProgressModal.toastErrMsg` → `.catch(toastError)`. Porter bound methods return `errcode.Error`. | Low |
| **3. Remaining sites** | Migrate CRUD/matcher/updater/execute/abort `.catch` blocks. Each bound method returns `errcode.Error`. | Medium |
| **4. Audit** | `grep -r 'err.includes\|reason.toString' frontend/src` → expect zero hits. | None |

---

## 7. Refinements / anti-patterns to avoid

1. **No English substring-matching anywhere** — not in the composable, not in components, not in fallback logic. An early draft of `useBackendError` had `if (code.includes('context canceled')) return` to swallow user-abort — replace with a proper `errAborted` / `msgAborted` code emitted from the Go side (`porter.go` abort, `execute.go` abort). Composables must not hardcode English.
2. **No catch-all `.catch(() => t('genericError'))`** — that swallows the actual code and defeats the purpose. Always pass the error through `toastError` (or `message` for non-toast display).
3. **Boundary discipline** — every Wails-bound method is a conversion point. Existing methods that return `fmt.Errorf("pkg: %w", inner)` (porter, execute, storage, update) must be wrapped at the boundary. Worth a package-level comment in `pkg/errcode` and a code-review checklist item.
4. **i18n completeness** — every new code requires entries in BOTH `en.json` and `zh_Hant_HK.json`. Safety net: `vue-i18n` falls back to the key string when missing (visible in UI, obvious in testing). Acceptable; not an excuse to skip locales.
5. **Don't centralize all codes in one file** — colocate sentinels with the package that emits them (`pkg/porter/errors.go`, etc.). Easier to discover and maintain.

---

## 8. Out of scope (future)

- **Porter progress messages** (`JobSnapshot.Messages` — `"Packing: X"`, `"Backing up: X"`, `"Downloading..."`, `"Warning: cleanup issue: %v"`). These are **informational progress, not errors**. They cross via a snapshot struct, not the `error` return path. Unifying them needs a separate `Message{Code, Params}` type emitted in the snapshot — a future pass, not part of `errcode`.
- **Updater `no releases found`** and http errors — covered by Phase 3 when the updater migrates.
- **Dark-mode-specific message variants** — not needed; codes and messages are locale-independent.

---

## 9. Open decisions

1. **Starter scope:** Phase 1 + CPU-temp adopter as proof (recommended), then Phase 2.
2. **Composable vs plain helper module:** `useBackendError` as a composable (uses `useI18n` + `useToast`) — confirm preferred.
3. **Toast-once behavior:** CPU-temp shows the toast once per session (flag `tempToasted`) to avoid spam if the state oscillates. Confirm for other long-running pollers (none currently).
4. **Whether to also return a dev-facing English message alongside the code** for richer logs (the `Err` field already wraps it; currently not surfaced anywhere). Optional.

---

## 10. References

- CPU temperature feature: PR #48 (initial), #49 (badge colors), #50 (error sentinel + negative temps).
- `PAWNIO_TEMP_PLAN.md` — sibling planning doc at repo root (precedent for `.md` handoffs).
- STYLE.md — i18n key prefixes (`err*`, `warn*`, `toast*`, `msg*`) and general conventions.
