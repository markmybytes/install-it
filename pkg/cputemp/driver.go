package cputemp

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	pawnIOUninstallKey = `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\PawnIO`
	pawnIODevice       = `\\.\GLOBALROOT\Device\PawnIO`

	// errorSuccessRebootRequired — PawnIO_setup exits with this after a
	// successful install that needs a reboot; treat it as success.
	errorSuccessRebootRequired = 3010
)

// Init installs PawnIO if absent and opens the driver device, loading the
// module for the detected CPU vendor. Never uninstalls. No RunOnce, no
// marker file — the registry uninstall key is the only install check.
//
// On any failure the error is logged to stderr and returned; ready stays
// false and IsAvailable() reports false, so the app degrades silently.
func Init(exeDir string) error {
	installer := filepath.Join(exeDir, "internals", "data", "PawnIO_setup.exe")
	intelMSR := filepath.Join(exeDir, "internals", "data", "IntelMSR.bin")
	ryzenSMU := filepath.Join(exeDir, "internals", "data", "RyzenSMU.bin")

	if err := ensureInstalled(installer); err != nil {
		return fail(err)
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
		return fail(fmt.Errorf("open PawnIO device: %w", err))
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
		// without setting ready: IsAvailable() stays false, CPUTemperature()
		// returns -1, badge hidden. No speculative vendor code.
		windows.CloseHandle(handle)
		return nil
	}
	if err != nil {
		windows.CloseHandle(handle)
		return fail(err)
	}

	drvHandle.Store(uintptr(handle))
	ready.Store(true) // LAST — gates all readers
	return nil
}

// fail logs an Init error to stderr and returns it unchanged.
func fail(err error) error {
	fmt.Fprintln(os.Stderr, "install-it: CPU temp unavailable:", err)
	return err
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
		return fmt.Errorf("run %s: %w (%s)", filepath.Base(installer), err, strings.TrimSpace(string(out)))
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
		return fmt.Errorf("read module %s: %w", filepath.Base(path), err)
	}
	if len(blob) == 0 {
		return fmt.Errorf("read module %s: empty file", filepath.Base(path))
	}
	var returned uint32
	if err := windows.DeviceIoControl(handle, IOCTL_PIO_LOAD_BINARY,
		&blob[0], uint32(len(blob)),
		nil, 0,
		&returned, nil); err != nil {
		return fmt.Errorf("load module %s: %w", filepath.Base(path), err)
	}
	return nil
}
