package sysinfo

import (
	"testing"
)

// TestIsGpuPnPEntity tests the classification of PnP entities as GPUs.
func TestIsGpuPnPEntity(t *testing.T) {
	tests := []struct {
		name   string
		entity Win32_PnPEntity
		want   bool
	}{
		{
			name:   "ClassGuid = GUID_DEVCLASS_DISPLAY → true",
			entity: Win32_PnPEntity{ClassGuid: GUID_DEVCLASS_DISPLAY},
			want:   true,
		},
		{
			name:   "ClassGuid = GUID_DEVCLASS_NET → false",
			entity: Win32_PnPEntity{ClassGuid: GUID_DEVCLASS_NET},
			want:   false,
		},
		{
			name: "ClassGuid empty, CompatibleID contains CC_03 → true",
			entity: Win32_PnPEntity{
				CompatibleID: []string{"PCI\\VEN_10DE&DEV_2208&REV_A1", "CC_03"},
			},
			want: true,
		},
		{
			name: "ClassGuid empty, CompatibleID contains CC_02 → false",
			entity: Win32_PnPEntity{
				CompatibleID: []string{"CC_02"},
			},
			want: false,
		},
		{
			name: "ClassGuid empty, no CompatibleID, Name contains Display → true",
			entity: Win32_PnPEntity{
				Name: "Microsoft Basic Display Adapter",
			},
			want: true,
		},
		{
			name: "ClassGuid empty, no CompatibleID, Name contains Ethernet → false",
			entity: Win32_PnPEntity{
				Name: "Realtek Ethernet Controller",
			},
			want: false,
		},
		{
			name: "ClassGuid empty, no CompatibleID, Name is PCI Device → false",
			entity: Win32_PnPEntity{
				Name: "PCI Device",
			},
			want: false,
		},
		{
			name:   "Empty struct → false",
			entity: Win32_PnPEntity{},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isGpuPnPEntity(tt.entity)
			if got != tt.want {
				t.Errorf("isGpuPnPEntity() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsNicPnPEntity tests the classification of PnP entities as NICs.
func TestIsNicPnPEntity(t *testing.T) {
	tests := []struct {
		name   string
		entity Win32_PnPEntity
		want   bool
	}{
		{
			name:   "ClassGuid = GUID_DEVCLASS_NET → true",
			entity: Win32_PnPEntity{ClassGuid: GUID_DEVCLASS_NET},
			want:   true,
		},
		{
			name:   "ClassGuid = GUID_DEVCLASS_DISPLAY → false",
			entity: Win32_PnPEntity{ClassGuid: GUID_DEVCLASS_DISPLAY},
			want:   false,
		},
		{
			name: "ClassGuid empty, CompatibleID contains CC_02 → true",
			entity: Win32_PnPEntity{
				CompatibleID: []string{"CC_02"},
			},
			want: true,
		},
		{
			name: "ClassGuid empty, CompatibleID contains CC_03 → false",
			entity: Win32_PnPEntity{
				CompatibleID: []string{"CC_03"},
			},
			want: false,
		},
		{
			name: "ClassGuid empty, no CompatibleID, Name contains Ethernet → true",
			entity: Win32_PnPEntity{
				Name: "Realtek PCIe Ethernet Controller",
			},
			want: true,
		},
		{
			name: "ClassGuid empty, no CompatibleID, Name contains Network → true",
			entity: Win32_PnPEntity{
				Name: "Bluetooth Device (Personal Area Network)",
			},
			want: true,
		},
		{
			name: "ClassGuid empty, no CompatibleID, Name contains Display → false",
			entity: Win32_PnPEntity{
				Name: "Microsoft Basic Display Adapter",
			},
			want: false,
		},
		{
			name: "ClassGuid empty, no CompatibleID, Name is PCI Device → false",
			entity: Win32_PnPEntity{
				Name: "PCI Device",
			},
			want: false,
		},
		{
			name:   "Empty struct → false",
			entity: Win32_PnPEntity{},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isNicPnPEntity(tt.entity)
			if got != tt.want {
				t.Errorf("isNicPnPEntity() = %v, want %v", got, tt.want)
			}
		})
	}
}
