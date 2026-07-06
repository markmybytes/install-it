package sysinfo

import (
	"fmt"
	"math"
	"strings"
	"sync"

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
// Pass "" for an unfiltered query, or a full clause like
// "WHERE ClassGuid = '...'" for server-side filtering.
func queryWMI[T any](where string) ([]T, error) {
	var cls []T
	q := wmi.CreateQuery(&cls, where)
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

// resolvedGpuNames returns human-readable GPU names. The name comes from
// Win32_PnPEntity (driver-independent). It runs a single PnP query that
// covers both ClassGuid-filtered display devices and ClassGuid IS NULL
// entities (handled client-side by isGpuPnPEntity's 3-tier fallback).
func (i SysInfo) resolvedGpuNames() ([]string, error) {
	entities, _ := queryWMI[Win32_PnPEntity](
		"WHERE ClassGuid = '" + GUID_DEVCLASS_DISPLAY + "' OR ClassGuid IS NULL",
	)

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

// resolvedNicNames returns the names of physical network adapters and
// Bluetooth radios as a flat list. Virtual adapters (Hyper-V switches,
// OpenVPN TAP, Bluetooth transports, BT PAN, etc.) are filtered out by
// requiring a PCI\ or USB\ hardware ID. Names are formatted as
// "<PnP name> (<PCI vendor>)" — the vendor suffix is only present when
// the PCI vendor lookup succeeds (i.e. for PCI devices, not USB radios
// like Bluetooth where the VEN_XXXX regex doesn't match).
// It runs a single PnP query that covers both ClassGuid-filtered net +
// Bluetooth devices and ClassGuid IS NULL entities (handled client-side
// by isNicPnPEntity / isBluetoothPnPEntity / isPhysicalPnPEntity).
func (i SysInfo) resolvedNicNames() ([]string, error) {
	entities, _ := queryWMI[Win32_PnPEntity](
		"WHERE ClassGuid = '" + GUID_DEVCLASS_NET +
			"' OR ClassGuid = '" + GUID_DEVCLASS_BLUETOOTH +
			"' OR ClassGuid IS NULL",
	)

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
	cpus, err := queryWMI[Win32_Processor]("")
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
	mems, err := queryWMI[Win32_PhysicalMemory]("")
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
	boards, err := queryWMI[Win32_BaseBoard]("")
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
	disks, err := queryWMI[Win32_DiskDrive]("")
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

	// Run the independent WMI queries concurrently. Each is a separate
	// DCOM/RPC call to the WMI service, so they parallelize well. Errors
	// are silently ignored per the existing contract — only the result is
	// dropped.
	//
	// GPU and NIC resolvers are fully self-contained (they each query
	// Win32_PnPEntity internally),
	// so they are not part of the fan-out here.
	var wg sync.WaitGroup

	run := func(fn func()) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fn()
		}()
	}

	var (
		cpuRes  []string
		memRes  []string
		moboRes []string
		diskRes []string
	)

	run(func() { cpuRes, _ = i.resolvedCpuNames() })
	run(func() { memRes, _ = i.resolvedMemoryNames() })
	run(func() { moboRes, _ = i.resolvedMotherboardNames() })
	run(func() { diskRes, _ = i.resolvedDiskNames() })

	wg.Wait()

	// GPU and NIC resolvers are fully self-contained and run their own
	// internal PnP queries after the parallel
	// batch — the wmi library serializes all queries via a package-level
	// mutex, so there is no additional wall-clock cost.
	if names, err := i.resolvedGpuNames(); err == nil && len(names) > 0 {
		hw.Gpu = names
	}
	if names, err := i.resolvedNicNames(); err == nil && len(names) > 0 {
		hw.Nic = names
	}

	if len(cpuRes) > 0 {
		hw.Cpu = cpuRes
	}
	if len(memRes) > 0 {
		hw.Memory = memRes
	}
	if len(moboRes) > 0 {
		hw.Motherboard = moboRes
	}
	if len(diskRes) > 0 {
		hw.Storage = diskRes
	}

	return hw, nil
}
