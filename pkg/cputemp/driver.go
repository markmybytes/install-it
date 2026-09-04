// To download PawnIO assets for local development:
//   go generate ./pkg/cputemp/
//
//go:generate curl -fsSL -o data/PawnIO_setup.exe https://github.com/namazso/PawnIO.Setup/releases/download/2.2.0/PawnIO_setup.exe
//go:generate curl -fsSL -o data/pawnio_modules.zip https://github.com/namazso/PawnIO.Modules/releases/download/0.2.10/release_0_2_10.zip
//go:generate powershell -NoProfile -Command "Expand-Archive data/pawnio_modules.zip data/.pawnio -Force; Move-Item -Force data/.pawnio/IntelMSR.bin,data/.pawnio/RyzenSMU.bin data/; Remove-Item -Recurse data/.pawnio,data/pawnio_modules.zip"

package cputemp

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	"install-it/pkg/errcode"
)

const (
	pawnIOUninstallKey = `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\PawnIO`
	pawnIODevice       = `\\.\GLOBALROOT\Device\PawnIO`

	// errorSuccessRebootRequired — PawnIO_setup exits with this after a
	// successful install that needs a reboot; treat it as success.
	errorSuccessRebootRequired = 3010
)

// initDriver installs PawnIO if absent and opens the driver device, loading the
// module for the detected CPU vendor. Never uninstalls. No RunOnce, no
// marker file — the registry uninstall key is the only install check.
//
// On any failure the error is logged to stderr and lifecycle becomes
// unavailable, so the app degrades silently.
func initDriver(exeDir string) error {
	succeeded := false
	defer func() {
		if !succeeded {
			lifecycle.Store(lifecycleUnavailable)
		}
	}()

	installer, intelMSR, ryzenSMU, err := resolveAssets(exeDir)
	if err != nil {
		return fail(errcode.New("errCPUTempInitFailed"))
	}

	if err := ensureInstalled(installer); err != nil {
		return fail(errcode.New("errCPUTempInitFailed"))
	}

	handle, err := windows.CreateFile(
		windows.StringToUTF16Ptr(pawnIODevice),
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil,
		windows.OPEN_EXISTING,
		0, 0,
	)
	if err != nil {
		return fail(errcode.New("errCPUTempInitFailed"))
	}

	vendor := detectVendor()
	cpuVendor.Store(vendor)

	switch vendor {
	case vendorIntel:
		err = loadModule(handle, intelMSR)
	case vendorAMD:
		err = loadModule(handle, ryzenSMU)
	default:
		// Unknown vendor — no module to load. Close the handle and return
		// without setting lifecycle available: CPUTemperature() reports unavailable.
		// No speculative vendor code.
		windows.CloseHandle(handle)
		return nil
	}
	if err != nil {
		windows.CloseHandle(handle)
		return fail(errcode.New("errCPUTempInitFailed"))
	}

	drvHandle.Store(uintptr(handle))
	succeeded = true
	lifecycle.Store(lifecycleAvailable) // LAST — gates all readers
	return nil
}

// fail logs an Init error to stderr and returns it unchanged.
func fail(err error) error {
	fmt.Fprintln(os.Stderr, "install-it: CPU temp unavailable:", err)
	return err
}

// resolveAssets locates PawnIO_setup.exe, IntelMSR.bin, RyzenSMU.bin.
// Probes exeDir/internals/data first (CI / wails build), then the source
// directory's data/ subdir (go generate / wails dev). Mirrors the dual-path
// lookup in pkg/sysinfo/pciids.go.
func resolveAssets(exeDir string) (installer, intelMSR, ryzenSMU string, err error) {
	var dirs []string
	dirs = append(dirs, filepath.Join(exeDir, "internals", "data"))
	if _, src, _, ok := runtime.Caller(0); ok {
		dirs = append(dirs, filepath.Join(filepath.Dir(src), "data"))
	}

	for _, dir := range dirs {
		installer = filepath.Join(dir, "PawnIO_setup.exe")
		intelMSR = filepath.Join(dir, "IntelMSR.bin")
		ryzenSMU = filepath.Join(dir, "RyzenSMU.bin")
		if _, err = os.Stat(installer); err == nil {
			return installer, intelMSR, ryzenSMU, nil
		}
	}
	return "", "", "", errcode.Newf("errCPUTempAssetsMissing", map[string]any{"dirs": dirs})
}

// ensureInstalled runs the PawnIO installer unless the registry uninstall
// key is already present. Returns nil on success or
// ERROR_SUCCESS_REBOOT_REQUIRED (install succeeded, reboot pending).
func ensureInstalled(installer string) error {
	if pawnIOInstalled() {
		return nil
	}

	cmd := exec.Command(installer, "-install", "-silent")
	// ponytail: -install/-silent flags are unverifiable from public source
	// (PawnIO.Setup repo is README-only); kept as-is per plan. Runtime-verify.
	if out, err := cmd.CombinedOutput(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == errorSuccessRebootRequired {
			return nil
		}
		return errcode.Newf("errCPUTempInstallFailed", map[string]any{
			"installer": filepath.Base(installer),
			"output":    strings.TrimSpace(string(out)),
		})
	}
	return nil
}

// pawnIOInstalled reports whether PawnIO is registered in the uninstall key.
func pawnIOInstalled() bool {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, pawnIOUninstallKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	k.Close()
	return true
}

// loadModule loads a PawnIO module binary via IOCTL_PIO_LOAD_BINARY.
func loadModule(handle windows.Handle, path string) error {
	blob, err := os.ReadFile(path)
	if err != nil {
		return errcode.Newf("errCPUTempModuleRead", map[string]any{"module": filepath.Base(path)})
	}
	if len(blob) == 0 {
		return errcode.Newf("errCPUTempModuleEmpty", map[string]any{"module": filepath.Base(path)})
	}
	var returned uint32
	if err := windows.DeviceIoControl(handle, IOCTL_PIO_LOAD_BINARY,
		&blob[0], uint32(len(blob)),
		nil, 0,
		&returned, nil); err != nil {
		return errcode.Newf("errCPUTempModuleLoad", map[string]any{"module": filepath.Base(path)})
	}
	return nil
}
