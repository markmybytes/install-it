package sysinfo

import (
	"fmt"
	"math"
	"strings"

	"github.com/yusufpapurcu/wmi"
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

func (i SysInfo) cpuInfo() ([]Win32_Processor, error) {
	var cls []Win32_Processor
	q := wmi.CreateQuery(&cls, "")
	if err := wmi.Query(q, &cls); err != nil {
		return cls, err
	}
	return cls, nil
}

func (i SysInfo) motherboardInfo() ([]Win32_BaseBoard, error) {
	var cls []Win32_BaseBoard
	q := wmi.CreateQuery(&cls, "")
	if err := wmi.Query(q, &cls); err != nil {
		return cls, err
	}
	return cls, nil
}

func (i SysInfo) memoryInfo() ([]Win32_PhysicalMemory, error) {
	var cls []Win32_PhysicalMemory
	q := wmi.CreateQuery(&cls, "")
	if err := wmi.Query(q, &cls); err != nil {
		return cls, err
	}
	return cls, nil
}

func (i SysInfo) diskInfo() ([]Win32_DiskDrive, error) {
	var cls []Win32_DiskDrive
	q := wmi.CreateQuery(&cls, "")
	if err := wmi.Query(q, &cls); err != nil {
		return cls, err
	}
	return cls, nil
}

// pnPEntityInfo queries all Win32_PnPEntity instances on the system.
// These provide HardwareID and CompatibleID even when no driver is installed.
func (i SysInfo) pnPEntityInfo() ([]Win32_PnPEntity, error) {
	var cls []Win32_PnPEntity
	q := wmi.CreateQuery(&cls, "")
	if err := wmi.Query(q, &cls); err != nil {
		return cls, err
	}
	return cls, nil
}

// IsGpuPnPEntity returns true if the PnP entity is a GPU/display adapter.
// Classification uses a 3-tier approach:
//  1. ClassGuid matches GUID_DEVCLASS_DISPLAY
//  2. CompatibleID contains "CC_03" (display class)
//  3. Name contains display-related keywords
func isGpuPnPEntity(e Win32_PnPEntity) bool {
	if e.ClassGuid != "" {
		return e.ClassGuid == GUID_DEVCLASS_DISPLAY
	}
	for _, cid := range e.CompatibleID {
		if strings.Contains(cid, "CC_03") {
			return true
		}
		if strings.Contains(cid, "CC_02") {
			return false
		}
	}
	nameUpper := strings.ToUpper(e.Name)
	return strings.Contains(nameUpper, "DISPLAY") ||
		strings.Contains(nameUpper, "VIDEO") ||
		strings.Contains(nameUpper, "VGA") ||
		strings.Contains(nameUpper, "3D")
}

// IsNicPnPEntity returns true if the PnP entity is a network adapter.
// Classification uses a 3-tier approach:
//  1. ClassGuid matches GUID_DEVCLASS_NET
//  2. CompatibleID contains "CC_02" (network class)
//  3. Name contains network-related keywords
func isNicPnPEntity(e Win32_PnPEntity) bool {
	if e.ClassGuid != "" {
		return e.ClassGuid == GUID_DEVCLASS_NET
	}
	for _, cid := range e.CompatibleID {
		if strings.Contains(cid, "CC_02") {
			return true
		}
		if strings.Contains(cid, "CC_03") {
			return false
		}
	}
	nameUpper := strings.ToUpper(e.Name)
	return strings.Contains(nameUpper, "ETHERNET") ||
		strings.Contains(nameUpper, "NETWORK") ||
		strings.Contains(nameUpper, "WI-FI") ||
		strings.Contains(nameUpper, "WIRELESS") ||
		strings.Contains(nameUpper, "WLAN")
}

// ResolvedGpuNames returns human-readable GPU names resolved via the pci.ids
// database.
func (i SysInfo) resolvedGpuNames() ([]string, error) {
	entities, err := i.pnPEntityInfo()
	if err != nil {
		return nil, err
	}

	var names []string
	seen := make(map[string]bool)
	for _, e := range entities {
		if isGpuPnPEntity(e) {
			for _, hwid := range e.HardwareID {
				if name := ResolvePciName(hwid); name != "" && !seen[name] {
					seen[name] = true
					names = append(names, name)
				}
			}
		}
	}

	return names, nil
}

// ResolvedNicNames returns human-readable NIC names. Uses the PnP device
// name (driver-supplied or BIOS-supplied) when non-empty, with the PCI
// vendor appended in parentheses. Falls back to the pci.ids-resolved name
// (which already contains the vendor) only when the PnP name is empty.
func (i SysInfo) resolvedNicNames() ([]string, error) {
	entities, err := i.pnPEntityInfo()
	if err != nil {
		return nil, err
	}

	var names []string
	seen := make(map[string]bool)
	for _, e := range entities {
		if !isNicPnPEntity(e) {
			continue
		}
		pnpName := strings.TrimSpace(e.Name)
		var name, vendor string
		if pnpName != "" {
			name = pnpName
			for _, hwid := range e.HardwareID {
				if v := ResolvePciVendor(hwid); v != "" {
					vendor = v
					break
				}
			}
		} else {
			for _, hwid := range e.HardwareID {
				if n := ResolvePciName(hwid); n != "" {
					name = n
					break
				}
			}
		}
		if name == "" {
			continue
		}
		if vendor != "" {
			name = name + " (" + vendor + ")"
		}
		if seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}

	return names, nil
}

// ResolvedCpuNames returns human-readable CPU names from Win32_Processor.
func (i SysInfo) resolvedCpuNames() ([]string, error) {
	cpus, err := i.cpuInfo()
	if err != nil {
		return nil, err
	}
	var names []string
	for _, v := range cpus {
		names = append(names, v.Name)
	}
	return names, nil
}

// ResolvedMemoryNames returns formatted memory strings:
// "Manufacturer PartNumber XGB YMHz".
func (i SysInfo) resolvedMemoryNames() ([]string, error) {
	mems, err := i.memoryInfo()
	if err != nil {
		return nil, err
	}
	var names []string
	for _, v := range mems {
		gb := float64(v.Capacity) / math.Pow(1024, 3)
		gbStr := strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.10g", gb), "0"), ".")
		names = append(names, fmt.Sprintf("%s %s %sGB %dMHz",
			v.Manufacturer, strings.TrimSpace(v.PartNumber), gbStr, v.Speed))
	}
	return names, nil
}

// ResolvedMotherboardNames returns formatted motherboard strings:
// "Manufacturer Product".
func (i SysInfo) resolvedMotherboardNames() ([]string, error) {
	boards, err := i.motherboardInfo()
	if err != nil {
		return nil, err
	}
	var names []string
	for _, v := range boards {
		names = append(names, fmt.Sprintf("%s %s", v.Manufacturer, v.Product))
	}
	return names, nil
}

// ResolvedDiskNames returns formatted disk strings:
// "Model (XGB)" with GB rounded to int.
func (i SysInfo) resolvedDiskNames() ([]string, error) {
	disks, err := i.diskInfo()
	if err != nil {
		return nil, err
	}
	var names []string
	for _, v := range disks {
		gb := int(math.Round(float64(v.Size) / math.Pow(1024, 3)))
		names = append(names, fmt.Sprintf("%s (%dGB)", v.Model, gb))
	}
	return names, nil
}

// ResolvedHardware returns all resolved hardware names in a single call.
// Individual query errors are silently ignored — this is intentional
// degradation for systems missing WMI providers.
func (i SysInfo) ResolvedHardware() (ResolvedHardware, error) {
	hw := ResolvedHardware{
		Cpu:         []string{},
		Gpu:         []string{},
		Memory:      []string{},
		Motherboard: []string{},
		Nic:         []string{},
		Storage:     []string{},
	}

	if names, err := i.resolvedCpuNames(); err == nil && names != nil {
		hw.Cpu = names
	}
	if names, err := i.resolvedGpuNames(); err == nil && names != nil {
		hw.Gpu = names
	}
	if names, err := i.resolvedMemoryNames(); err == nil && names != nil {
		hw.Memory = names
	}
	if names, err := i.resolvedMotherboardNames(); err == nil && names != nil {
		hw.Motherboard = names
	}
	if names, err := i.resolvedNicNames(); err == nil && names != nil {
		hw.Nic = names
	}
	if names, err := i.resolvedDiskNames(); err == nil && names != nil {
		hw.Storage = names
	}

	return hw, nil
}
