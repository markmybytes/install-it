package sysinfo

import (
	"fmt"
	"math"
	"strings"
)

// ResolvedHardware holds all resolved hardware names in a single struct
// for efficient frontend retrieval via one Wails bridge call.
type ResolvedHardware struct {
	Cpu         []string `json:"cpu"`
	Gpu         []string `json:"gpu"`
	Memory      []string `json:"memory"`
	Motherboard []string `json:"motherboard"`
	Nic         []string `json:"nic"`
	Storage     []string `json:"storage"`
}

type SysInfo struct{}

// resolveDeviceNames returns human-readable names for PnP devices matching
// the include predicate.
func resolveDeviceNames(devices []PnPDevice, include func(PnPDevice) bool) []string {
	var names []string
	for _, e := range devices {
		if !include(e) {
			continue
		}
		pnpName := strings.TrimSpace(e.Name)
		var name, vendor string

		// Always try the PCI ID database first. The PnP name is unreliable
		// when a generic fallback driver is loaded (e.g. Microsoft Basic
		// Display Adapter from display.inf), because InstallState is 0
		// (CM_INSTALL_STATE_INSTALLED) even though the real vendor driver
		// is not present.
		for _, hwid := range e.HardwareID {
			if n := ResolvePciName(hwid); n != "" {
				name = n
				break
			}
		}
		if name == "" {
			// PCI resolution failed — use PnP name as fallback.
			// Append vendor suffix for context when the driver is not
			// properly installed.
			name = pnpName
			if e.InstallState != 0 {
				for _, hwid := range e.HardwareID {
					if v := ResolvePciVendor(hwid); v != "" {
						vendor = v
						break
					}
				}
			}
		}

		if name == "" {
			continue
		}
		if vendor != "" {
			name = name + " (" + vendor + ")"
		}
		names = append(names, name)
	}
	return names
}

func (i SysInfo) ResolvedHardware() (ResolvedHardware, error) {
	// The WMI library serializes all queries via a package-level mutex,
	// so there's no parallelism benefit to goroutines. Errors are silently
	// ignored — only the result is dropped.
	var cpuRes, memRes, moboRes, diskRes []string

	if cpus, err := queryWMI[Win32_Processor](""); err == nil {
		for _, v := range cpus {
			cpuRes = append(cpuRes, v.Name)
		}
	}

	if mems, err := queryWMI[Win32_PhysicalMemory](""); err == nil {
		for _, v := range mems {
			gb := float64(v.Capacity) / math.Pow(1024, 3)
			gbStr := strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.10g", gb), "0"), ".")
			memRes = append(memRes, fmt.Sprintf("%s %s %sGB %dMHz",
				v.Manufacturer, strings.TrimSpace(v.PartNumber), gbStr, v.Speed))
		}
	}

	if boards, err := queryWMI[Win32_BaseBoard](""); err == nil {
		for _, v := range boards {
			moboRes = append(moboRes, fmt.Sprintf("%s %s", v.Manufacturer, v.Product))
		}
	}

	if disks, err := queryWMI[Win32_DiskDrive](""); err == nil {
		for _, v := range disks {
			gb := int(math.Round(float64(v.Size) / math.Pow(1024, 3)))
			diskRes = append(diskRes, fmt.Sprintf("%s (%dGB)", v.Model, gb))
		}
	}

	pnpDevices, _ := enumeratePnPDevices()

	return ResolvedHardware{
		Cpu:         cpuRes,
		Gpu:         resolveDeviceNames(pnpDevices, isGpuDevice),
		Memory:      memRes,
		Motherboard: moboRes,
		Nic: resolveDeviceNames(pnpDevices, func(e PnPDevice) bool {
			return isPhysicalDevice(e) && (isNicDevice(e) || isBluetoothDevice(e))
		}),
		Storage: diskRes,
	}, nil
}
