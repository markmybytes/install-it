package sysinfo

import (
	"golang.org/x/sys/windows"
)

// PnPDevice represents a Plug and Play device enumerated via SetupAPI.
type PnPDevice struct {
	ClassGuid    string
	HardwareID   []string
	CompatibleID []string
	Name         string
	InstallState uint32 // 0 = CM_INSTALL_STATE_INSTALLED (driver fully installed)
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
		nil,                        // classGUID — nil + DIGCF_ALLCLASSES = all classes
		"",                         // enumerator — no filter
		0,                          // hwndParent — no parent window
		windows.DIGCF_ALLCLASSES|windows.DIGCF_PRESENT,
		windows.DevInfo(0), // deviceInfoSet — 0 = create new list
		"", // machineName — local machine
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
		})
		index++
	}
	return devices, nil
}
