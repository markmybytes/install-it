# Backend Message i18n Unification — Handoff

> Reference document for unifying how Go backend messages/errors reach the Vue frontend for translation. Surfaced by the CPU temperature follow-ups (#48, #49, #50); this is the canonical plan for the unification work.

---

## 1. Problem

Backend→UI error handling is inconsistent across ~21 frontend touchpoints and leaks raw Go error text across the Wails bridge at scale.

**Frontend patterns (all wrong in different ways):**

| Pattern | Where | Problem |
|---|---|---|
| RAW `err.toString()` toast | DriverFormComponent, MatchRuleFormComponent, app-info.vue, UpdateModal.vue, porter Abort | English leaks to UI |
| Localized toast, error dropped (`catch(() => t('toastX'))`) | drivers/index, match-rules/index, App.vue, settings.vue, index.vue matcher | Loses actual failure; one generic key for all causes |
| English substring-matching | ProgressModal.toastErrMsg (16-line `err.includes()` chain), CommandStatusModal, TaskStatus | Breaks when backend text changes; not localizable |

**Backend audit (verified against source):** roughly **75 sites** return error text across the bridge verbatim through bound methods:

| Class | Count | Examples |
|---|---|---|
| Deliberate interpolation | ~25 | `action.go:94` `"cannot open source file %s"`, `update.go:238` `"zip slip detected: %s"`, `porter.go:105` `"unsupported archive format version %d"` |
| Raw inner-error leaks | ~50 | storage CRUD returns raw **gorm SQL text** on every method; updater returns raw http/os/json errors; App dialogs, Matcher WMI, cputemp MSR reads return untouched OS errors |

Nobody designed the leak class — those are accidents crossing the bridge. All of it is English-coupled and unlocalizable.

---

## 2. Decisions

| # | Decision | Rationale |
|---|---|---|
| 1 | **Backend emits a stable code; frontend owns localized text** | i18n lives on the frontend. |
| 2 | **The code IS the vue-i18n key** | Zero mapping tables; adding an error forces both the code and the i18n entry. |
| 3 | **Prefix = severity** | `err*` → red toast, `warn*` → amber, `msg*` → info, `toast*` → general. Matches STYLE.md key prefixes. |
| 4 | **Structured errors cross the bridge natively via Wails `ErrorFormatter`** | Verified in pinned Wails v2.12.0: the formatter's return value is JSON-marshaled and reaches the JS promise rejection as a real object (`v2/internal/frontend/dispatcher/calls.go` → `reject(message.error)`). No string smuggling needed. |
| 5 | **Minimal immutable error type; no error chaining** | The app has **no logging** at this stage — wrapped inner chains would have zero consumers. `.Error()` returns the bare code for tests/devtools; structure travels via `MarshalJSON`. |
| 6 | **Bound methods are conversion points** | Internal helpers may build `*errcode.Error` freely; every bound method must return one (or fall back to a raw string during migration). Never wrap codes with `%w` in bound returns. |

---

## 3. Architecture

### 3.1 Backend: `pkg/errcode`

```go
// pkg/errcode/errcode.go
package errcode

import "encoding/json"

// Error carries a stable code (doubles as vue-i18n key) plus optional i18n
// interpolation params. Immutable: unexported fields, build via New/Newf only.
type Error struct {
	code   string
	params map[string]any
}

func New(code string) *Error { return &Error{code: code} }

func Newf(code string, params map[string]any) *Error {
	p := make(map[string]any, len(params))
	for k, v := range params {
		p[k] = v
	}
	return &Error{code: code, params: p}
}

// Error returns the bare code — readable in tests and devtools.
// Structured data crosses the bridge via MarshalJSON + ErrorFormatter, not here.
func (e *Error) Error() string { return e.code }

// Is enables errors.Is across fmt.Errorf wrapping.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	return ok && t.code == e.code
}

func (e *Error) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Code   string         `json:"code"`
		Params map[string]any `json:"params,omitempty"`
	}{e.code, e.params})
}
```

Deliberately absent: `Err` field, `Unwrap`, `Wrap`. No logger exists to print chains; reintroduce only if logging arrives.

Sentinel codes are **colocated with the emitting package**, as exported immutable values:

```go
// pkg/porter/errors.go
var (
    ErrExportFailed = errcode.New("errExportFailed")
    ErrImportFailed = errcode.New("errImportFailed")
)

// Parametrized sites construct inline:
return errcode.Newf("errImportFileOpen", map[string]any{"path": filePath})
```

Per-package files: `pkg/sysinfo/errors.go`, `pkg/porter/errors.go`, `pkg/execute/errors.go`, `pkg/storage/errors.go`, `pkg/update/errors.go`, etc.

### 3.2 Wails wiring (one global config point)

```go
// main.go — covers ALL bound structs
ErrorFormatter: func(err error) any {
    var ec *errcode.Error
    if errors.As(err, &ec) {
        return ec // marshaled to {"code": "...", "params": {...}}
    }
    return err.Error() // unmigrated/raw errors fall back to plain string
},
```

### 3.3 Contract

```
Go:    errcode.New("warnCPUTempUnavailable")
        ↓ ErrorFormatter
JS rejection: { code: "warnCPUTempUnavailable", params?: {...} }
        ↓
Vue:   t(code, params) → localized message; prefix drives toast color
```

| Prefix | Toast color | UI treatment |
|---|---|---|
| `err*` | `error` (red) | Error toast |
| `warn*` | `warning` (amber) | Warning toast or gray-state badge |
| `msg*` | `info` | Informational |
| `toast*` | `error` or `info` | General toast |

Non-code errors arrive as plain strings during migration; the composable handles both shapes.

### 3.4 Frontend: `useBackendError` composable

```ts
// frontend/src/composables/useBackendError.ts
import { useI18n } from 'vue-i18n'
import { useToast } from '#imports'

export function useBackendError() {
  const { t } = useI18n()
  const toast = useToast()

  function codeOf(err: unknown): { code: string; params?: Record<string, any> } {
    if (err && typeof err === 'object' && 'code' in err) {
      return err as { code: string; params?: Record<string, any> }
    }
    return { code: String(err) } // raw fallback during migration
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

No JSON parsing, no English substring matching, ever.

### 3.5 Version pin caveat + contract test

Object pass-through works on **v2.12.0** (pinned in `go.mod`). Wails master (post-PR #5244) wraps rejections in `new Error(...)`, which would stringify structured payloads into `[object Object]`.

**Required:** one contract test asserting a bound-method rejection arrives as an object with a `code` field, so any future Wails upgrade fails loudly instead of silently corrupting the channel.

---

## 4. Reference adopter: CPU temperature (first site)

Validates end-to-end: backend code → formatter → frontend badge + i18n toast.

### 4.1 Backend

`pkg/sysinfo/errors.go`:
```go
var (
    ErrCPUTempUnavailable = errcode.New("warnCPUTempUnavailable")
    ErrCPUTempNoReadings  = errcode.New("warnCPUTempNoReadings")
    ErrCPUTempReadFailed  = errcode.New("warnCPUTempReadFailed")
)
```

`pkg/sysinfo/sysinfo.go`:
```go
func (i SysInfo) CPUTemperature() (float64, error) {
    if !cputemp.IsAvailable() {
        return 0, ErrCPUTempUnavailable
    }
    temps, err := cputemp.GetCPUTemperatures()
    if err != nil {
        return 0, ErrCPUTempReadFailed // detail intentionally dropped; no logs by design
    }
    if len(temps) == 0 {
        return 0, ErrCPUTempNoReadings
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
"warnCPUTempReadFailed":  "Failed to read CPU temperature."
```

---

## 5. Migration phases

| Phase | Scope | Risk |
|---|---|---|
| **1. Scaffold** | Create `pkg/errcode`; wire `ErrorFormatter` in `main.go`; add contract test (§3.5); create `useBackendError`; add CPU-temp codes + i18n entries; wire CPU temp as reference adopter. | None outside CPU temp |
| **2. Worst offenders** | Migrate `ProgressModal.toastErrMsg` → `.catch(toastError)`; porter bound methods return codes (parametrized `Newf` where a path/status genuinely helps the user); execute abort paths emit proper codes instead of relying on frontend substring matches. | Low |
| **3. Remaining sites** | Storage CRUD (kill gorm SQL leaks), updater, matcher, App dialogs, system-utilities.vue (9×), remaining `.catch` blocks. Each bound method returns codes. | Medium |
| **4. Audit** | `grep -r 'err.includes\|reason.toString' frontend/src` → zero hits. Spot-check that no bound method still returns raw inner errors. | None |

Params policy: default to bare codes. Add `Newf` params only where the dynamic value has real user value (e.g., which file failed during import). Everything else collapses to a generic code — the ProgressModal already streams per-file progress, so the toast rarely needs the detail too.

---

## 6. Anti-patterns to avoid

1. **No English substring-matching anywhere** — composables/components must not hardcode English. User-abort gets a proper `msgAborted`-style code emitted from Go (`porter.go` abort, `execute.go` abort).
2. **No catch-all `.catch(() => t('genericError'))`** — always route through `toastError`/`message` so codes survive.
3. **Boundary discipline** — never `%w`-wrap a code in a bound-method return; internal wrapping is fine since `errors.As` sees through `fmt.Errorf`.
4. **i18n completeness** — every code needs entries in BOTH `en.json` and `zh_Hant_HK.json`. Missing keys fall back to the visible key string (obvious in testing, not an excuse).
5. **Don't centralize codes** — sentinels live beside the package that emits them.
6. **Don't grow the type back** — no `Wrap`/`Unwrap`/`Err` until a logger exists; no always-JSON `.Error()`; the dual fallback (object|string) in `codeOf` is intentional, don't "fix" it.

---

## 7. Out of scope (future)

- **Porter progress messages** (`JobSnapshot.Messages` — `"Packing: X"`, `"Backing up: X"`, …): informational progress crossing via snapshot struct, not the error path. Needs a separate `Message{Code, Params}` pass.
- **Updater HTTP error detail** — covered in Phase 3 with plain codes.
- **Dark-mode message variants** — unnecessary; codes are locale-independent.

---

## 8. References

- CPU temperature feature: PR #48 (initial), #49 (badge colors), #50 (error sentinel + negative temps).
- Wails v2 error transport: `v2/internal/frontend/dispatcher/calls.go` (`.Error()` default, `errfmt` override), `options.ErrorFormatter` docs; master PR #5244 changes rejection wrapping — recheck on upgrade.
- STYLE.md — i18n key prefixes (`err*`, `warn*`, `toast*`, `msg*`) and general conventions.
