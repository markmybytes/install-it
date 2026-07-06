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

// queryWMI executes a WMI query returning all instances of the given type.
func queryWMI[T any]() ([]T, error) {
	var cls []T
	q := wmi.CreateQuery(&cls, "")
	if err := wmi.Query(q, &cls); err != nil {
		return cls, err
	}
	return cls, nil
}

// isGpuPnPEntity returns true if the PnP entity is a GPU/display adapter.
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

// isNicPnPEntity returns true if the PnP entity is a network adapter.
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

// isBluetoothPnPEntity returns true if the PnP entity is in the Bluetooth
// device class. Combined with isPhysicalPnPEntity, identifies the actual
// radio (which has a USB hardware ID) versus paired devices and protocol
// services (which are software-emulated and lack one).
func isBluetoothPnPEntity(e Win32_PnPEntity) bool {
	return e.ClassGuid == GUID_DEVCLASS_BLUETOOTH
}

// isPhysicalPnPEntity returns true if the PnP entity represents real
// physical hardware — i.e. its first hardware or compatible ID carries a
// PCI\ or USB\ prefix. Software-emulated virtual adapters (Hyper-V
// switches, OpenVPN TAP, virtual Bluetooth transports) lack such IDs and
// are filtered out by this check.
func isPhysicalPnPEntity(e Win32_PnPEntity) bool {
	for _, hwid := range e.HardwareID {
		if strings.HasPrefix(hwid, "PCI\\") || strings.HasPrefix(hwid, "USB\\") {
			return true
		}
	}
	for _, cid := range e.CompatibleID {
		if strings.HasPrefix(cid, "PCI\\") || strings.HasPrefix(cid, "USB\\") {
			return true
		}
	}
	return false
}

// ResolvedGpuNames returns human-readable GPU names. The name comes from
// Win32_PnPEntity (driver-independent); VRAM comes from
// Win32_VideoController (driver-dependent, omitted when no display driver).
func (i SysInfo) resolvedGpuNames() ([]string, error) {
	entities, err := queryWMI[Win32_PnPEntity]()
	if err != nil {
		return nil, err
	}

	// Best-effort: query for VRAM. May fail or return empty if no display
	// driver is installed.
	controllers, _ := queryWMI[Win32_VideoController]()
	vramByName := make(map[string]uint64)
	for _, c := range controllers {
		vramByName[strings.TrimSpace(c.Name)] = c.AdapterRAM
	}

	var names []string
	seen := make(map[string]bool)
	for _, e := range entities {
		if !isGpuPnPEntity(e) {
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
		if vram := vramByName[pnpName]; vram > 0 {
			name = name + " (" + formatBytes(vram) + ")"
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

// formatBytes returns a decimal GB string, or "" for zero (e.g. when no
// display driver is installed and AdapterRAM is 0).
func formatBytes(bytes uint64) string {
	if bytes == 0 {
		return ""
	}
	return fmt.Sprintf("%.1f GB", float64(bytes)/1e9)
}

// ResolvedNicNames returns the names of physical network adapters and
// Bluetooth radios as a flat list. Virtual adapters (Hyper-V switches,
// OpenVPN TAP, Bluetooth transports, BT PAN, etc.) are filtered out by
// requiring a PCI\ or USB\ hardware ID. Names are formatted as
// "<PnP name> (<PCI vendor>)" — the vendor suffix is only present when
// the PCI vendor lookup succeeds (i.e. for PCI devices, not USB radios
// like Bluetooth where the VEN_XXXX regex doesn't match).
func (i SysInfo) resolvedNicNames() ([]string, error) {
	// Win32_PnPEntity is used so HardwareID and Name are available even
	// when no NIC or Bluetooth driver is installed.
	entities, err := queryWMI[Win32_PnPEntity]()
	if err != nil {
		return nil, err
	}

	var names []string
	seen := make(map[string]bool)
	for _, e := range entities {
		if !isPhysicalPnPEntity(e) {
			continue
		}
		if !isNicPnPEntity(e) && !isBluetoothPnPEntity(e) {
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
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		if vendor != "" {
			name = name + " (" + vendor + ")"
		}
		names = append(names, name)
	}

	return names, nil
}

func (i SysInfo) resolvedCpuNames() ([]string, error) {
	cpus, err := queryWMI[Win32_Processor]()
	if err != nil {
		return nil, err
	}
	var names []string
	for _, v := range cpus {
		names = append(names, v.Name)
	}
	return names, nil
}

func (i SysInfo) resolvedMemoryNames() ([]string, error) {
	mems, err := queryWMI[Win32_PhysicalMemory]()
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

func (i SysInfo) resolvedMotherboardNames() ([]string, error) {
	boards, err := queryWMI[Win32_BaseBoard]()
	if err != nil {
		return nil, err
	}
	var names []string
	for _, v := range boards {
		names = append(names, fmt.Sprintf("%s %s", v.Manufacturer, v.Product))
	}
	return names, nil
}

func (i SysInfo) resolvedDiskNames() ([]string, error) {
	disks, err := queryWMI[Win32_DiskDrive]()
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
	if info, err := i.resolvedNicNames(); err == nil {
		hw.Nic = info
	}
	if names, err := i.resolvedDiskNames(); err == nil && names != nil {
		hw.Storage = names
	}

	return hw, nil
}
