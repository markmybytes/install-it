# PawnIO CPU Temperature Implementation Plan

## Problem

`pkg/sysinfo/sysinfo.go:110` reads CPU temperature via `gopsutil/v3/host.SensorsTemperatures()`, which queries WMI class `MSAcpi_ThermalZoneTemperature` in namespace `root/wmi`. This reports an ACPI **thermal zone** sensor (on the motherboard, near the CPU socket) — not CPU die temperature.

Result: 17°C (room temperature) instead of the real 38-45°C reported by HWiNFO. HWiNFO reads CPU MSR registers (`IA32_THERM_STATUS`) directly via a kernel driver.

## Chosen Approach: PawnIO, install-if-absent, no cleanup

PawnIO is a modern, signed kernel driver by namazso. It's what LibreHardwareMonitor (LHM) / FanControl use internally.

Requirements:
- App setting `enable_cpu_temp` (default `false`) gates the feature — opt-in.
- Setting takes effect on next app start (driver loads at `OnStartup`).
- On startup with the setting on: check if PawnIO is installed; if not, install it. **No uninstall, no RunOnce, no marker file.** PawnIO stays once installed — matches FanControl/LHM behavior, eliminates all cleanup edge cases.
- Support Intel (IntelMSR) and AMD (RyzenSMU). Other vendors → no temp, silent.
- Optional refresh interval setting (default 5s; 1/5/10/30/60 options).
- Temp is a separate Wails method, not embedded in `ResolvedHardware()`. Polling at 1s would otherwise re-run every WMI class query (CPU, GPU, memory, motherboard, NIC, storage) each tick — heavy and pointless. `SysInfo.CPUTemperature()` does only the IOCTL read.

Elevation is a non-issue: install-it runs `requireAdministrator` (`build/windows/wails.exe.manifest:20`), so PawnIO install (self-elevates anyway), device open (ACL is SY+BA), and IOCTLs (FILE_ANY_ACCESS) all work with no extra handling.

### Architecture

```
App startup (main.go OnStartup):
  ├─ Read AppSetting.enable_cpu_temp
  ├─ If false → skip everything (no driver, no temp)
  └─ If true:
       └─ go cputemp.Init(dirRoot)  // NON-BLOCKING — see "Async Init" below
            ├─ Check HKLM\...\Uninstall\PawnIO registry key
            ├─ If absent: run PawnIO_setup.exe -install -silent (~2-3s)
            │    (idempotent; if it fails, log + return error, IsAvailable()=false)
            ├─ Open \\.\GLOBALROOT\Device\PawnIO
            ├─ Detect CPU vendor (WMI Win32_Processor.Manufacturer)
            └─ Load IntelMSR.bin or RyzenSMU.bin via IOCTL
            On any error: log to stderr, IsAvailable()=false, no temp
       Window appears immediately; temp badge appears once Init completes.

Frontend (index.vue):
  ├─ onBeforeMount: utils.getHardware() once → hwinfos
  ├─ If settings.enable_cpu_temp:
  │    ├─ setTimeout-recursive poll of SysInfo.CPUTemperature()
  │    │    (interval read live from settings each tick — no restart for interval change)
  │    └─ Update cpuTemp ref; render as badge next to "CPU" heading
  └─ onBeforeUnmount: clear pending timeout

SysInfo.ResolvedHardware():
  └─ No longer reads temp. Returns CPU/GPU/mem/mobo/NIC/storage names only.

SysInfo.CPUTemperature() (new):
  └─ If cputemp.IsAvailable(): read max temp via IOCTL, return float64
     Else: return -1 — frontend hides badge
```

### Key design decisions

| Decision | Rationale |
|---|---|
| **No cleanup, no RunOnce** | FanControl/LHM leave PawnIO installed; we do too. Eliminates all RunOnce edge cases. User opted in, so leaving a signed driver is acceptable. |
| **Install-if-absent only** | Single registry check. We don't track ownership — never uninstall, so ownership is irrelevant. |
| **Separate `CPUTemperature()` method** | Polling must not re-run all WMI queries every tick. IOCTL read is microseconds; WMI is 50-200ms × 5 classes. |
| **Setting defaults to `false`** | Opt-in. Kernel driver install is a side effect the user should explicitly allow. |
| **Setting effective on next start** | Driver load happens in `OnStartup`. Live enable/disable would need load/unload logic — YAGNI. UI shows restart hint. |
| **Refresh interval live** | Interval read from settings each tick via setTimeout recursion (ref lookup, free). No restart needed to change interval — only enable/disable requires restart. |
| **Async Init** | `Init()` runs in a goroutine from `OnStartup`; window appears immediately. `IsAvailable()` flips true when ready; frontend polls `CPUTemperature()` which returns -1 until then. Avoids 2-3s frozen launch for an opt-in feature. |
| **Error badge hides** | On read error mid-session: `cpuTemp = null`, badge vanishes, retry next tick. No stale freeze at last good value. |
| **No `//go:embed`** | Follow `internals/data/` pattern (like `pci.ids.gz`). |
| **Direct IOCTL, no DLL** | Pure Go via `golang.org/x/sys/windows`, no CGo, no .NET. |
| **Files are `.bin` not `.amx`** | Renamed in latest PawnIO.Modules release. |
| **No `Close()`** | YAGNI — process exit releases the handle. No Wails `OnShutdown` needed. |
| **Intel + AMD only; others silent** | PawnIO.Modules ships IntelMSR.bin + RyzenSMU.bin. Unknown vendor (ARM, etc.) → `IsAvailable()` true but no module loaded → `CPUTemperature()` returns -1. No speculative vendor code. |

## File Changes

### New files

| File | Purpose |
|---|---|
| `pkg/cputemp/cputemp.go` | Public API: `Init()`, `GetCPUTemperatures()`, `IsAvailable()` |
| `pkg/cputemp/driver.go` | PawnIO lifecycle: registry check, install, open device |
| `pkg/cputemp/ioctl.go` | IOCTL constants, buffer structs, `executeFn()` helper |
| `pkg/cputemp/msr.go` | Module loading, MSR/SMU reading, Intel/AMD temp decode |
| `pkg/cputemp/vendor.go` | CPU vendor detection via WMI (own minimal query) |

### Modified files

| File | Change |
|---|---|
| `pkg/storage/app_setting.go` | Add `EnableCPUTemp bool` + `CPUTempRefreshInterval int` |
| `main.go` | Store `appSettings` in package var; call `cputemp.Init(dirRoot)` in goroutine from `OnStartup` if enabled |
| `pkg/sysinfo/sysinfo.go` | Remove `gopsutil/v3/host` temp block; add `CPUTemperature()` method |
| `frontend/src/pages/index.vue` | Poll `CPUTemperature()`; render temp badge; clear timeout on unmount |
| `frontend/src/i18n/en.json` | Add `settingCPUTemp`, `settingEnableCPUTemp`, `settingCPUTempRestartHint`, `settingCPUTempRefreshInterval` |
| `frontend/src/i18n/zh_Hant_HK.json` | Add Chinese equivalents |
| `frontend/src/pages/settings.vue` | Add checkbox + refresh interval select in Display/Hardware Info section |
| `.github/workflows/build_and_release.yml` | Add PawnIO download steps |

### Runtime files (not committed)

```
internals/data/
├── PawnIO_setup.exe    # ~3.4 MB, downloaded by CI
├── IntelMSR.bin        # ~5.3 KB, downloaded by CI
└── RyzenSMU.bin        # ~39.7 KB, downloaded by CI
```

## Package API: `pkg/cputemp/`

```go
type CPUTemp struct {
    Name  string  // "CPU Package", "CPU Tctl"
    Value float64 // °C
}

// Init installs PawnIO if absent and opens the driver device.
// If PawnIO is already installed (by anyone), skips install.
// Never uninstalls. No RunOnce, no marker file.
func Init(exeDir string) error

// GetCPUTemperatures reads current CPU temperature(s).
func GetCPUTemperatures() ([]CPUTemp, error)

// IsAvailable returns true if driver is open and a module is loaded.
func IsAvailable() bool
```

No `Close()` — process exit releases the handle.

## IOCTL Constants

```go
const (
    IOCTL_PIO_LOAD_BINARY = 0x9C402084  // CTL_CODE(41394, 0x821, METHOD_BUFFERED, FILE_ANY_ACCESS)
    IOCTL_PIO_EXECUTE_FN  = 0x9C402104  // CTL_CODE(41394, 0x841, METHOD_BUFFERED, FILE_ANY_ACCESS)
)

// Execute input: [32 bytes function name] + [8 bytes params...]
func executeFn(handle windows.Handle, fnName string, params ...uint64) (uint64, error) {
    bufSize := 32 + len(params)*8
    buf := make([]byte, bufSize)
    copy(buf[:32], fnName)
    for i, p := range params {
        binary.LittleEndian.PutUint64(buf[32+i*8:], p)
    }
    out := make([]byte, 8)
    var returned uint32
    err := windows.DeviceIoControl(handle, IOCTL_PIO_EXECUTE_FN,
        &buf[0], uint32(bufSize),
        &out[0], 8,
        &returned, nil)
    return binary.LittleEndian.Uint64(out), err
}
```

## Driver Lifecycle: `Init(exeDir)`

```
1. Resolve paths:
     installer = exeDir + "\internals\data\PawnIO_setup.exe"
     intelMSR  = exeDir + "\internals\data\IntelMSR.bin"
     ryzenSMU  = exeDir + "\internals\data\RyzenSMU.bin"

2. Check if PawnIO installed:
     key = HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\PawnIO
     if key exists → skip install
     if absent → run: PawnIO_setup.exe -install -silent
       (if exitCode != 0 && != ERROR_SUCCESS_REBOOT_REQUIRED → return error)

3. Open device:
     handle = CreateFile(`\\.\GLOBALROOT\Device\PawnIO`, GENERIC_READ|WRITE, ...)
     if fails → return error

4. Detect CPU vendor:
     cputemp writes its own minimal WMI query (cannot use sysinfo.queryWMI — unexported).
     Use github.com/yusufpapurcu/wmi directly with a minimal struct:
       type cpuVendor struct{ Manufacturer string }
       var cpus []cpuVendor
       wmi.Query("SELECT Manufacturer FROM Win32_Processor", &cpus)
       // cpus[0].Manufacturer == "GenuineIntel" or "AuthenticAMD"

5. Load module:
     read IntelMSR.bin or RyzenSMU.bin
     IOCTL_PIO_LOAD_BINARY(handle, moduleBlob)
     Unknown vendor → no module, IsAvailable()=false for temp purposes.

6. Store handle + vendor in package-level vars
```

No RunOnce. No marker file. Registry key presence is the only "is it installed?" check, used solely to skip a redundant install.

## Temperature Reading

### Intel path

```
1. Read TjMax: executeFn("ioctl_read_msr", 0x1A2)
   tjMax = float64((value >> 16) & 0xFF)
   if tjMax == 0 → tjMax = 100

2. Read package temp: executeFn("ioctl_read_msr", 0x1B1)
   if (value & 0x80000000) == 0 → invalid, skip
   digitalReadout = float64((value >> 16) & 0x7F)
   temp = tjMax - digitalReadout

3. Return [{Name: "CPU Package", Value: temp}]
```

### AMD path (RyzenSMU.bin)

```
1. executeFn("ioctl_read_smu_register", 0x59800)
   rawTemp = float64((value >> 21)) * 0.125
   if (value & 0x80000) != 0 || (value & 0x30000) == 0x30000:
       rawTemp -= 49.0
   tctl = rawTemp  // this is Tctl, not Tdie

2. Return [{Name: "CPU Tctl", Value: tctl}]
```

**Known limitation:** SMN 0x59800 returns Tctl, not Tdie. Tdie = Tctl - offset where offset varies by model (Zen 2: ~10-27°C, Zen 3: ~20°C, Zen 4: ~25°C). Without per-model detection, we report Tctl. Labeling as "Tctl" is honest about what we're showing.

## AppSetting Changes

### `pkg/storage/app_setting.go`

```go
type AppSetting struct {
    // ... existing fields ...
    EnableCPUTemp          bool `json:"enable_cpu_temp"`
    CPUTempRefreshInterval int  `json:"cpu_temp_refresh_interval"` // seconds
}
```

Default in `All()` when file doesn't exist:
```go
s.setting = AppSetting{
    // ... existing defaults ...
    EnableCPUTemp:          false,
    CPUTempRefreshInterval: 5,
}
```

### `main.go` changes

Add package-level var:
```go
var appSettings *storage.AppSettingStorage
```

In `main()`:
```go
appSettings = &storage.AppSettingStorage{Path: filepath.Join(dirConf, "setting.json")}
// ...
Bind: []interface{}{
    // ...
    appSettings,  // use the var, not a new instance
    // ...
},
```

In `OnStartup`:
```go
if settings, err := appSettings.All(); err == nil && settings.EnableCPUTemp {
    go func() {
        if err := cputemp.Init(dirRoot); err != nil {
            fmt.Fprintln(os.Stderr, "install-it: CPU temp unavailable:", err)
        }
    }()
}
```

## sysinfo.go Integration

### Remove (lines 108-123)

The `host.SensorsTemperatures()` block and the `"github.com/shirou/gopsutil/v3/host"` import. `ResolvedHardware()` no longer embeds temp in CPU names.

### Add new method

```go
// CPUTemperature returns the current CPU package temperature in °C via PawnIO,
// or -1 if unavailable. Lightweight: IOCTL read only, no WMI queries.
func (i SysInfo) CPUTemperature() (float64, error) {
    if !cputemp.IsAvailable() {
        return -1, nil
    }
    temps, err := cputemp.GetCPUTemperatures()
    if err != nil || len(temps) == 0 {
        return -1, err
    }
    var max float64
    for _, t := range temps {
        if t.Value > max {
            max = t.Value
        }
    }
    return max, nil
}
```

`gopsutil/v3/host` import removed. `gopsutil/v3` stays (used by `pkg/execute/command.go`).

## Frontend Changes

### `frontend/src/pages/index.vue`

Add polling + temp badge:

```vue
<script setup lang="ts">
import * as sysinfoApi from '@/wailsjs/go/sysinfo/SysInfo'
// ... existing imports ...

const cpuTemp = ref<number | null>(null)
let stopPolling: (() => void) | null = null

onBeforeMount(() => {
  utils.getHardware().then(v => (hwinfos.value = v))

  if (settingStore.settings.enable_cpu_temp) {
    // setTimeout recursion: each tick reads the current interval setting,
    // so interval changes apply live without teardown/rebuild.
    let timer: ReturnType<typeof setTimeout>
    const tick = () =>
      sysinfoApi
        .CPUTemperature()
        .then(t => {
          // -1 = unavailable (Init not done, or driver absent). Hide badge.
          // On read error mid-session: set null (badge vanishes), retry next tick.
          cpuTemp.value = t >= 0 ? Math.round(t) : null
        })
        .catch(() => {
          cpuTemp.value = null
        })
        .finally(() => {
          const secs = Math.max(1, Number(settingStore.settings.cpu_temp_refresh_interval) || 5)
          timer = setTimeout(tick, secs * 1000)
        })
    tick()
    stopPolling = () => clearTimeout(timer)
  }
})

onBeforeUnmount(() => {
  stopPolling?.()
})
</script>
```

Template — temp badge next to CPU heading:

```vue
<h2 class="text-sm font-bold">
  {{ $t(hwKey(part)) }}
  <span
    v-if="part === 'cpu' && cpuTemp !== null"
    class="ms-1 rounded bg-apple-green-100 px-1.5 text-xs font-medium text-apple-green-700"
  >
    {{ cpuTemp }}°C
  </span>
</h2>
```

After command completion (existing `@completed` handler), keep `utils.getHardware()` refresh — temp polling continues independently.

### `frontend/src/i18n/en.json`

Add:
```json
"settingCPUTemp": "CPU Temperature",
"settingEnableCPUTemp": "Enable CPU Temperature Detection",
"settingCPUTempRestartHint": "Takes effect on next app start",
"settingCPUTempRefreshInterval": "Refresh Interval"
```

### `frontend/src/i18n/zh_Hant_HK.json`

Add Chinese equivalents, e.g.:
```json
"settingCPUTemp": "CPU 溫度",
"settingEnableCPUTemp": "啟用 CPU 溫度偵測",
"settingCPUTempRestartHint": "下次啟動應用時生效",
"settingCPUTempRefreshInterval": "更新間隔"
```

### `frontend/src/pages/settings.vue`

Add in the **Display → Hardware Info** section (after the NIC filters, ~line 287), since this is a hardware-display setting:

```vue
<hr class="border-gray-100 dark:border-gray-800" />

<section>
  <h3 class="mb-3 text-lg font-bold">{{ $t('settingCPUTemp') }}</h3>

  <div class="flex flex-col gap-y-3">
    <label class="flex w-fit cursor-pointer items-center select-none">
      <UCheckbox
        v-model="settings.enable_cpu_temp"
        name="enable_cpu_temp"
        color="primary"
        class="me-2"
      />
      <span>{{ $t('settingEnableCPUTemp') }}</span>
    </label>

    <p class="text-xs text-gray-500">{{ $t('settingCPUTempRestartHint') }}</p>

    <div
      class="flex flex-col gap-y-1 transition-opacity duration-200"
      :class="{ 'pointer-events-none opacity-40': !settings.enable_cpu_temp }"
    >
      <label>{{ $t('settingCPUTempRefreshInterval') }}</label>

      <div class="flex items-center gap-x-2">
        <USelect
          v-model="settings.cpu_temp_refresh_interval"
          name="cpu_temp_refresh_interval"
          color="primary"
          class="w-24"
          :items="[1, 5, 10, 30, 60].map(v => ({ label: String(v), value: v }))"
          :disabled="!settings.enable_cpu_temp"
        />
        <span class="text-sm text-gray-500">{{ $t('labelSeconds') }}</span>
      </div>
    </div>
  </div>
</section>
```

The `settings` ref is bound to `AppSetting` via the store. Wails auto-generates `models.ts` on `wails dev`/`wails build`.

## CI/CD Changes

### `.github/workflows/build_and_release.yml`

Add after "Download PCI ID database" step:

```yaml
- name: Download PawnIO driver and modules
  shell: pwsh
  run: |
    $expected = "1f519a22e47187f70a1379a48ca604981c4fcf694f4e65b734aaa74a9fba3032"
    Invoke-WebRequest -Uri "https://github.com/namazso/PawnIO.Setup/releases/download/2.2.0/PawnIO_setup.exe" -OutFile "build\bin\internals\data\PawnIO_setup.exe" -UseBasicParsing
    $actual = (Get-FileHash "build\bin\internals\data\PawnIO_setup.exe" -Algorithm SHA256).Hash.ToLower()
    if ($actual -ne $expected) { throw "PawnIO_setup.exe SHA256 mismatch: expected $expected, got $actual" }
    Invoke-WebRequest -Uri "https://github.com/namazso/PawnIO.Modules/releases/download/0.2.9/release_0_2_9.zip" -OutFile "build\bin\internals\data\pawnio_modules.zip" -UseBasicParsing
    Expand-Archive -Path "build\bin\internals\data\pawnio_modules.zip" -DestinationPath "build\bin\internals\data\pawnio_modules" -Force
    Move-Item "build\bin\internals\data\pawnio_modules\IntelMSR.bin" "build\bin\internals\data\IntelMSR.bin"
    Move-Item "build\bin\internals\data\pawnio_modules\RyzenSMU.bin" "build\bin\internals\data\RyzenSMU.bin"
    Remove-Item "build\bin\internals\data\pawnio_modules.zip"
    Remove-Item -Recurse "build\bin\internals\data\pawnio_modules"
```

ZIP step already includes `internals\data` — no change needed.

**Verify versions/SHAs at implementation time** — PawnIO.Setup and PawnIO.Modules may have released newer versions since this plan was written.

**32-bit build:** PawnIO_setup.exe includes both x86 and x64 drivers in the self-extracting archive and installs the correct one for the OS architecture. Verified — PawnIO supports both.

## Error Handling

| Scenario | Behavior |
|---|---|
| `enable_cpu_temp = false` | Init never called. No temp. Badge hidden. |
| PawnIO installer fails | Log to stderr. `IsAvailable()` false. `CPUTemperature()` returns -1. Badge hidden. |
| Device open fails (reboot needed) | Log to stderr. `IsAvailable()` false. Badge hidden. |
| Module load fails | Log to stderr. `GetCPUTemperatures()` errors. Badge hidden. |
| MSR read fails (invalid bit) | Skip that reading. |
| AMD SMU read fails | Return error. No temp that tick. |
| Unknown CPU vendor | No module loaded. `IsAvailable()` false. Badge hidden. |
| Read error mid-session | Badge hides (`cpuTemp = null`), next tick retries. |
| All error paths | Silent degradation — badge hidden, no temp shown. |

## Verification

1. `go build ./...` compiles (all packages)
2. `go vet ./pkg/cputemp/` passes
3. `go vet ./pkg/sysinfo/` passes (after import removal + new method)
4. `go vet ./pkg/storage/` passes (after AppSetting change)
5. `cd frontend && npm run type-check` passes (after Wails regenerates models)
6. Manual: with `enable_cpu_temp=true` + restart, CPU heading shows temp badge updating at the configured interval; interval changes apply live without restart
7. Manual: with `enable_cpu_temp=false`, no badge, no PawnIO install attempt
8. Review with @oracle for architecture sanity

## Execution Order

| Step | Depends on | Parallelizable |
|---|---|---|
| 1. Add `EnableCPUTemp` + `CPUTempRefreshInterval` to `AppSetting` | None | — |
| 2. Create `pkg/cputemp/` package | None | ✅ with step 1 |
| 3. Integrate into `main.go` + `sysinfo.go` (remove old temp block, add `CPUTemperature()`) | Steps 1, 2 | — |
| 4. Add i18n keys + settings UI (checkbox + interval select) | Step 1 | ✅ with step 3 |
| 5. Frontend polling + temp badge in `index.vue` | Step 3 (needs `CPUTemperature` binding) | ✅ with step 4 |
| 6. Update CI/CD workflow | None | ✅ with steps 2-5 |
| 7. Verify build + review | Steps 2-6 | — |

## Asset URLs

- PawnIO_setup.exe: `https://github.com/namazso/PawnIO.Setup/releases/download/2.2.0/PawnIO_setup.exe`
  - SHA256: `1f519a22e47187f70a1379a48ca604981c4fcf694f4e65b734aaa74a9fba3032`
  - Size: 3,410,960 bytes
- PawnIO Modules: `https://github.com/namazso/PawnIO.Modules/releases/download/0.2.9/release_0_2_9.zip`
  - Contains: `IntelMSR.bin` (5,292 bytes), `RyzenSMU.bin` (39,652 bytes), and other modules

## Known Limitations

1. **PawnIO persists after first enable.** Once install-it installs it, it stays until manually removed (via PawnIO_setup.exe -uninstall, or another tool). Acceptable for opt-in feature; matches FanControl/LHM behavior. Document in user-facing text if needed.

2. **Enable/disable requires restart.** Driver loads at `OnStartup`; UI shows restart hint. Interval changes apply live (no restart). Live enable would need load/unload logic — YAGNI.

3. **AMD Tdie vs Tctl.** SMN 0x59800 returns Tctl, not Tdie. Tdie = Tctl - offset where offset varies by CPU model. Without per-model detection, we report Tctl. Noticeably higher than HWiNFO "CPU Die" on Zen 2+. Labeling as "Tctl" is honest. Badge just shows the number; no label distinction in the compact UI.

4. **No per-core temps (Intel).** Only package temp from MSR 0x1B1. Per-core requires MSR 0x19C per logical processor with `SetThreadAffinityMask`. Package temp is more useful for overall CPU health and fits the compact badge UI.

5. **Other CPU vendors.** Only Intel + AMD have PawnIO modules. ARM/unknown → no temp, silent. No speculative support for hypothetical vendors.