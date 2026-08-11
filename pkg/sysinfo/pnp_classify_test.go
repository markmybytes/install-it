package sysinfo

import (
	"testing"
)

// TestResolveDeviceNames tests the resolveDeviceNames function.
func TestResolveDeviceNames(t *testing.T) {
	// An always-true filter for most test cases.
	includeAll := func(PnPDevice) bool { return true }
	// A filter that always returns false.
	includeNone := func(PnPDevice) bool { return false }

	tests := []struct {
		name     string
		devices  []PnPDevice
		include  func(PnPDevice) bool
		expected []string
	}{
		{
			name: "Installed device with resolvable PCI → PCI name used (always preferred over PnP)",
			devices: []PnPDevice{
				{
					Name:         "NVIDIA GeForce RTX 3080",
					HardwareID:   []string{"PCI\\VEN_10DE&DEV_2208&SUBSYS_220810DE&REV_A1"},
					InstallState: 0,
				},
			},
			include:  includeAll,
			expected: []string{"NVIDIA Corporation GA102 [GeForce RTX 3080 Ti]"},
		},
		{
			name: "Empty PnP name and unresolvable PCI ID → skipped",
			devices: []PnPDevice{
				{
					Name:         "",
					HardwareID:   []string{"PCI\\VEN_C0FF&DEV_EEEE"},
					InstallState: 0,
				},
			},
			include:  includeAll,
			expected: nil,
		},
		{
			name: "Two identical devices → both appear (no dedup)",
			devices: []PnPDevice{
				{
					Name:         "Realtek PCIe GbE Family Controller",
					HardwareID:   []string{"PCI\\VEN_10EC&DEV_8168"},
					InstallState: 0,
				},
				{
					Name:         "Realtek PCIe GbE Family Controller",
					HardwareID:   []string{"PCI\\VEN_10EC&DEV_8168"},
					InstallState: 0,
				},
			},
			include:  includeAll,
			expected: []string{"Realtek Semiconductor Co., Ltd. RTL8111/8168/8211/8411 PCI Express Gigabit Ethernet Controller", "Realtek Semiconductor Co., Ltd. RTL8111/8168/8211/8411 PCI Express Gigabit Ethernet Controller"},
		},
		{
			name: "MS Basic Display Adapter with InstallState==0 (fallback driver) → PCI name used",
			devices: []PnPDevice{
				{
					Name:         "Microsoft Basic Display Adapter",
					HardwareID:   []string{"PCI\\VEN_10DE&DEV_2208&SUBSYS_220810DE&REV_A1"},
					InstallState: 0,
				},
			},
			include:  includeAll,
			expected: []string{"NVIDIA Corporation GA102 [GeForce RTX 3080 Ti]"},
		},
		{
			name: "Uninstalled generic PnP name (InstallState!=0) → PCI name used",
			devices: []PnPDevice{
				{
					Name:         "Microsoft Basic Display Adapter",
					HardwareID:   []string{"PCI\\VEN_10DE&DEV_2208&SUBSYS_220810DE&REV_A1"},
					InstallState: 1,
				},
			},
			include:  includeAll,
			expected: []string{"NVIDIA Corporation GA102 [GeForce RTX 3080 Ti]"},
		},
		{
			name: "Known vendor but unknown device → ResolvePciName returns vendor name",
			devices: []PnPDevice{
				{
					Name:         "Microsoft Basic Display Adapter",
					HardwareID:   []string{"PCI\\VEN_10DE&DEV_EEEE"},
					InstallState: 1,
				},
			},
			include:  includeAll,
			expected: []string{"NVIDIA Corporation"},
		},
		{
			name: "Generic PnP, unresolvable PCI, InstallState==0 → PnP name (no suffix)",
			devices: []PnPDevice{
				{
					Name:         "Unknown Display Device",
					HardwareID:   []string{"PCI\\VEN_C0FF&DEV_EEEE"},
					InstallState: 0,
				},
			},
			include:  includeAll,
			expected: []string{"Unknown Display Device"},
		},
		{
			name: "Device excluded by filter → not in output",
			devices: []PnPDevice{
				{
					Name: "NVIDIA GeForce RTX 3080",
				},
			},
			include:  includeNone,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveDeviceNames(tt.devices, tt.include)
			if len(got) != len(tt.expected) {
				t.Fatalf("resolveDeviceNames() = %v (len=%d), want %v (len=%d)",
					got, len(got), tt.expected, len(tt.expected))
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("resolveDeviceNames()[%d] = %q, want %q", i, got[i], tt.expected[i])
				}
			}
		})
	}
}

// TestIsGpuDevice tests the classification of PnP devices as GPUs.
func TestIsGpuDevice(t *testing.T) {
	tests := []struct {
		name   string
		device PnPDevice
		want   bool
	}{
		{
			name:   "ClassGuid = GUID_DEVCLASS_DISPLAY → true",
			device: PnPDevice{ClassGuid: GUID_DEVCLASS_DISPLAY},
			want:   true,
		},
		{
			name:   "ClassGuid = GUID_DEVCLASS_NET → false",
			device: PnPDevice{ClassGuid: GUID_DEVCLASS_NET},
			want:   false,
		},
		{
			name: "ClassGuid empty, CompatibleID contains CC_03 → true",
			device: PnPDevice{
				CompatibleID: []string{"PCI\\VEN_10DE&DEV_2208&REV_A1", "CC_03"},
			},
			want: true,
		},
		{
			name: "ClassGuid empty, CompatibleID contains CC_02 → false",
			device: PnPDevice{
				CompatibleID: []string{"CC_02"},
			},
			want: false,
		},
		{
			name: "ClassGuid empty, no CompatibleID, Name contains Display → true",
			device: PnPDevice{
				Name: "Microsoft Basic Display Adapter",
			},
			want: true,
		},
		{
			name: "ClassGuid empty, no CompatibleID, Name contains Ethernet → false",
			device: PnPDevice{
				Name: "Realtek Ethernet Controller",
			},
			want: false,
		},
		{
			name: "ClassGuid empty, no CompatibleID, Name is PCI Device → false",
			device: PnPDevice{
				Name: "PCI Device",
			},
			want: false,
		},
		{
			name:   "Empty struct → false",
			device: PnPDevice{},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isGpuDevice(tt.device)
			if got != tt.want {
				t.Errorf("isGpuDevice() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsNicDevice tests the classification of PnP devices as NICs.
func TestIsNicDevice(t *testing.T) {
	tests := []struct {
		name   string
		device PnPDevice
		want   bool
	}{
		{
			name:   "ClassGuid = GUID_DEVCLASS_NET → true",
			device: PnPDevice{ClassGuid: GUID_DEVCLASS_NET},
			want:   true,
		},
		{
			name:   "ClassGuid = GUID_DEVCLASS_DISPLAY → false",
			device: PnPDevice{ClassGuid: GUID_DEVCLASS_DISPLAY},
			want:   false,
		},
		{
			name: "ClassGuid empty, CompatibleID contains CC_02 → true",
			device: PnPDevice{
				CompatibleID: []string{"CC_02"},
			},
			want: true,
		},
		{
			name: "ClassGuid empty, CompatibleID contains CC_03 → false",
			device: PnPDevice{
				CompatibleID: []string{"CC_03"},
			},
			want: false,
		},
		{
			name: "ClassGuid empty, no CompatibleID, Name contains Ethernet → true",
			device: PnPDevice{
				Name: "Realtek PCIe Ethernet Controller",
			},
			want: true,
		},
		{
			name: "ClassGuid empty, no CompatibleID, Name contains Network → true",
			device: PnPDevice{
				Name: "Bluetooth Device (Personal Area Network)",
			},
			want: true,
		},
		{
			name: "ClassGuid empty, no CompatibleID, Name contains Display → false",
			device: PnPDevice{
				Name: "Microsoft Basic Display Adapter",
			},
			want: false,
		},
		{
			name: "ClassGuid empty, no CompatibleID, Name is PCI Device → false",
			device: PnPDevice{
				Name: "PCI Device",
			},
			want: false,
		},
		{
			name:   "Empty struct → false",
			device: PnPDevice{},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isNicDevice(tt.device)
			if got != tt.want {
				t.Errorf("isNicDevice() = %v, want %v", got, tt.want)
			}
		})
	}
}
