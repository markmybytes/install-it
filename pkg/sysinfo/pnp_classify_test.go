package sysinfo

import (
	"testing"
)

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
