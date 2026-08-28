package cputemp

import (
	"fmt"
	"os"
	"sync"

	"golang.org/x/sys/windows"

	"install-it/pkg/errcode"
)

const (
	msrTjMax        = 0x1A2 // IA32_TEMPERATURE_TARGET
	msrPackageTemp  = 0x1B1 // IA32_PACKAGE_THERM_STATUS
	smuTempRegister = 0x59800
)

// accessPCMutexName is the SMU register access lock the RyzenSMU module doc
// requires holding while calling ioctl_read_smu_register. The doc names the
// object "\BaseNamedObjects\Access_PCI" (its kernel path); Win32 OpenMutex
// takes the bare name and resolves it into that same directory.
var (
	accessPCMutexName, _ = windows.UTF16PtrFromString("Access_PCI")
	accessPCLogOnce      sync.Once
)

// GetCPUTemperatures reads the current CPU temperature(s) via the loaded
// PawnIO module. Returns a nil slice and nil error when unavailable
// (Init not done, unknown vendor, or no handle).
func GetCPUTemperatures() ([]CPUTemp, error) {
	if !ready.Load() {
		return nil, nil
	}
	handle := windows.Handle(drvHandle.Load())
	if handle == 0 {
		return nil, nil
	}
	switch cpuVendor.Load() {
	case vendorIntel:
		return readIntel(handle)
	case vendorAMD:
		return readAMD(handle)
	default:
		return nil, nil // unknown vendor
	}
}

// readIntel returns the CPU package temperature from MSR 0x1B1.
func readIntel(handle windows.Handle) ([]CPUTemp, error) {
	tjMaxVal, err := executeFn(handle, "ioctl_read_msr", msrTjMax)
	if err != nil {
		return nil, errcode.New("errCPUTempMSRRead")
	}
	tjMax := float64((tjMaxVal >> 16) & 0xFF)
	if tjMax == 0 {
		tjMax = 100
	}

	val, err := executeFn(handle, "ioctl_read_msr", msrPackageTemp)
	if err != nil {
		return nil, errcode.New("errCPUTempMSRRead")
	}
	if val&0x80000000 == 0 {
		// VALID bit clear — no reading yet. Skip this tick.
		return nil, errcode.New("errCPUTempMSRRead")
	}
	digitalReadout := float64((val >> 16) & 0x7F)
	return []CPUTemp{{Name: "CPU Package", Value: tjMax - digitalReadout}}, nil
}

// readAMD returns the CPU Tctl temperature from SMN 0x59800.
func readAMD(handle windows.Handle) ([]CPUTemp, error) {
	release := acquireAccessPCMutex()
	if release != nil {
		defer release()
	}

	val, err := executeFn(handle, "ioctl_read_smu_register", smuTempRegister)
	if err != nil {
		return nil, errcode.New("errCPUTempSMURead")
	}

	rawTemp := float64(val>>21) * 0.125
	if val&0x80000 != 0 || val&0x30000 == 0x30000 {
		rawTemp -= 49.0
	}
	return []CPUTemp{{Name: "CPU Tctl", Value: rawTemp}}, nil
}

// acquireAccessPCMutex acquires the Access_PCI mutant as a courtesy protocol
// (matches LibreHardwareMonitor): readers of the SMU register serialize on
// it. Nothing blocks on it — if it can't be opened or isn't granted within
// 1s, the read proceeds anyway. Returns a release func, or nil if the
// mutant was never acquired.
func acquireAccessPCMutex() func() {
	mutex, err := windows.OpenMutex(windows.SYNCHRONIZE, false, accessPCMutexName)
	if err != nil {
		accessPCLogOnce.Do(func() {
			fmt.Fprintln(os.Stderr, "install-it: Access_PCI mutant unavailable, reading SMU without it:", err)
		})
		return nil
	}
	event, err := windows.WaitForSingleObject(mutex, 1000) // 1s timeout, not INFINITE
	// WAIT_ABANDONED: previous owner died without releasing; OS still grants
	// ownership, so we must release it. Treat same as WAIT_OBJECT_0.
	if err == nil && (event == windows.WAIT_OBJECT_0 || event == windows.WAIT_ABANDONED) {
		return func() {
			windows.ReleaseMutex(mutex)
			windows.CloseHandle(mutex)
		}
	}
	// Timed out or failed — proceed without the lock (courtesy protocol).
	windows.CloseHandle(mutex)
	return nil
}
