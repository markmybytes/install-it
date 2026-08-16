// Package cputemp reads CPU die temperature via the PawnIO kernel driver
// (namazso, the driver LibreHardwareMonitor/FanControl use internally).
//
// Windows-only. Init() installs PawnIO if absent, opens the device and loads
// the module for the detected CPU vendor (IntelMSR.bin / RyzenSMU.bin).
// GetCPUTemperatures() then reads MSR/SMU registers via IOCTLs.
package cputemp

import (
	"encoding/binary"
	"strings"
	"sync/atomic"

	"github.com/yusufpapurcu/wmi"
	"golang.org/x/sys/windows"
)

// IOCTL codes for PawnIO 2.2.0 (pawnio_um.h):
// CTL_CODE(41394, 0x821, METHOD_BUFFERED, FILE_ANY_ACCESS) / 0x841.
const (
	IOCTL_PIO_LOAD_BINARY = 0xA1B22084
	IOCTL_PIO_EXECUTE_FN  = 0xA1B22104
)

// CPU vendor IDs.
const (
	vendorUnknown int32 = iota
	vendorIntel
	vendorAMD
)

// CPUTemp is a single CPU temperature reading.
type CPUTemp struct {
	Name  string  // "CPU Package" or "CPU Tctl"
	Value float64 // °C
}

// Package-level driver state. Init runs in a goroutine while
// GetCPUTemperatures is polled, so every field is atomic. ready is set
// LAST in Init and gates all readers.
var (
	drvHandle atomic.Uintptr // PawnIO device handle; 0 = none
	cpuVendor atomic.Int32   // vendorUnknown / vendorIntel / vendorAMD
	ready     atomic.Bool    // true only after Init completed successfully
)

// executeFn invokes a PawnIO module function via IOCTL_PIO_EXECUTE_FN.
// Input layout: [32 bytes null-terminated function name] + [8 bytes per param].
func executeFn(handle windows.Handle, fnName string, params ...uint64) (uint64, error) {
	bufSize := 32 + len(params)*8
	buf := make([]byte, bufSize)
	copy(buf[:32], fnName) // lstrcpynA caps at 31 chars + null; copy is fine
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

// cpuVendorInfo is the minimal Win32_Processor projection used by
// detectVendor. A dedicated struct keeps this package independent of
// pkg/sysinfo's (unexported) query helper.
type cpuVendorInfo struct {
	Manufacturer string
}

// detectVendor identifies the CPU vendor via WMI. Returns vendorUnknown
// (no module loaded, temps unavailable) when the query fails or the vendor
// has no PawnIO module.
func detectVendor() int32 {
	var cpus []cpuVendorInfo
	if err := wmi.Query("SELECT Manufacturer FROM Win32_Processor", &cpus); err != nil || len(cpus) == 0 {
		return vendorUnknown
	}
	switch strings.TrimSpace(cpus[0].Manufacturer) {
	case "GenuineIntel":
		return vendorIntel
	case "AuthenticAMD":
		return vendorAMD
	}
	return vendorUnknown
}

// IsAvailable returns true if the driver is open, a module is loaded, and
// Init completed successfully.
func IsAvailable() bool {
	return ready.Load()
}
