package sysinfo

import (
	"strings"

	"golang.org/x/sys/windows"
)

// Device setup class GUIDs for classifying PnP devices by type.
const (
	GUID_DEVCLASS_DISPLAY   = "{4d36e968-e325-11ce-bfc1-08002be10318}"
	GUID_DEVCLASS_NET       = "{4d36e972-e325-11ce-bfc1-08002be10318}"
	GUID_DEVCLASS_BLUETOOTH = "{e0cbf06c-cd8b-4647-bb8a-263b43f0f974}"
)

// PnPDevice represents a Plug and Play device enumerated via SetupAPI.
type PnPDevice struct {
	ClassGuid    string
	HardwareID   []string
	CompatibleID []string
	Name         string
	InstallState uint32 // 0 = CM_INSTALL_STATE_INSTALLED (driver fully installed)
	Service      string // function driver service name (empty = no driver bound)
}

// isGpuDevice returns true if the PnP device is a GPU/display adapter.
// Classification uses a 3-tier approach:
//  1. ClassGuid matches GUID_DEVCLASS_DISPLAY
//  2. CompatibleID contains "CC_03" (display class)
//  3. Name contains display-related keywords
func isGpuDevice(e PnPDevice) bool {
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

// isNicDevice returns true if the PnP device is a network adapter.
// Classification uses a 3-tier approach:
//  1. ClassGuid matches GUID_DEVCLASS_NET
//  2. CompatibleID contains "CC_02" (network class)
//  3. Name contains network-related keywords
func isNicDevice(e PnPDevice) bool {
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

// isBluetoothDevice returns true if the PnP device is in the Bluetooth
// device class. Combined with isPhysicalDevice, identifies the actual
// radio (which has a USB hardware ID) versus paired devices and protocol
// services (which are software-emulated and lack one).
func isBluetoothDevice(e PnPDevice) bool {
	return e.ClassGuid == GUID_DEVCLASS_BLUETOOTH
}

var displayFallbackServices = map[string]bool{
	"basicdisplay": true, // display.inf
	"basicrender":  true, // basicrender.inf (WARP)
	"vgasave":      true, // vga.sys (legacy)
}

// isFallbackDriver returns true when no real vendor driver is loaded.
// Empty Service = no driver bound (NICs have no universal fallback).
// Display devices additionally check against inbox fallback services.
func isFallbackDriver(e PnPDevice) bool {
	svc := strings.ToLower(e.Service)
	if svc == "" {
		return true
	}
	if isGpuDevice(e) {
		return displayFallbackServices[svc]
	}
	return false
}

// isPhysicalDevice returns true if the PnP device represents real
// physical hardware — i.e. its first hardware or compatible ID carries a
// PCI\ or USB\ prefix. Software-emulated virtual adapters (Hyper-V
// switches, OpenVPN TAP, virtual Bluetooth transports) lack such IDs and
// are filtered out by this check.
func isPhysicalDevice(e PnPDevice) bool {
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

// getDeviceString retrieves a string registry property from a device.
// Returns "" on error (property not available).
func getDeviceString(devInfo windows.DevInfo, data *windows.DevInfoData, property windows.SPDRP) string {
	val, err := windows.SetupDiGetDeviceRegistryProperty(devInfo, data, property)
	if err != nil {
		return ""
	}
	s, ok := val.(string)
	if !ok {
		return ""
	}
	return s
}

// getDeviceStringSlice retrieves a multi-string registry property from a device.
// Returns nil on error (property not available).
func getDeviceStringSlice(devInfo windows.DevInfo, data *windows.DevInfoData, property windows.SPDRP) []string {
	val, err := windows.SetupDiGetDeviceRegistryProperty(devInfo, data, property)
	if err != nil {
		return nil
	}
	s, ok := val.([]string)
	if !ok {
		return nil
	}
	return s
}

// enumeratePnPDevices returns all present PnP devices using the SetupAPI
// (SetupDiGetClassDevsEx + SetupDiEnumDeviceInfo). This is 10-100x faster
// than the equivalent WMI Win32_PnPEntity query and does not contend on the
// WMI library's package-level mutex, so it can run truly in parallel with
// other WMI queries.
func enumeratePnPDevices() ([]PnPDevice, error) {
	devInfo, err := windows.SetupDiGetClassDevsEx(
		nil, // classGUID — nil + DIGCF_ALLCLASSES = all classes
		"",  // enumerator — no filter
		0,   // hwndParent — no parent window
		windows.DIGCF_ALLCLASSES|windows.DIGCF_PRESENT,
		windows.DevInfo(0), // deviceInfoSet — 0 = create new list
		"",                 // machineName — local machine
	)
	if err != nil {
		return nil, err
	}
	defer windows.SetupDiDestroyDeviceInfoList(devInfo)

	var devices []PnPDevice
	index := 0
	for {
		data, err := windows.SetupDiEnumDeviceInfo(devInfo, index)
		if err != nil {
			if err == windows.ERROR_NO_MORE_ITEMS {
				break
			}
			index++
			continue
		}

		var installState uint32
		if val, err := windows.SetupDiGetDeviceRegistryProperty(devInfo, data, windows.SPDRP_INSTALL_STATE); err == nil {
			if v, ok := val.(uint32); ok {
				installState = v
			}
		}

		devices = append(devices, PnPDevice{
			ClassGuid:    getDeviceString(devInfo, data, windows.SPDRP_CLASSGUID),
			HardwareID:   getDeviceStringSlice(devInfo, data, windows.SPDRP_HARDWAREID),
			CompatibleID: getDeviceStringSlice(devInfo, data, windows.SPDRP_COMPATIBLEIDS),
			Name:         getDeviceString(devInfo, data, windows.SPDRP_DEVICEDESC),
			InstallState: installState,
			Service:      getDeviceString(devInfo, data, windows.SPDRP_SERVICE),
		})
		index++
	}
	return devices, nil
}
