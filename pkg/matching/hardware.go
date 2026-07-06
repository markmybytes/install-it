package matching

import (
	"install-it/pkg/storage"
	"install-it/pkg/sysinfo"
)

// WMIHardwareQuerier queries hardware via WMI and formats results
// into strings matching the frontend's getHardware() formatting.
type WMIHardwareQuerier struct{}

// HardwareMap queries WMI and returns formatted strings per rule source.
// Errors from individual hardware queries are silently ignored —
// this is intentional degradation for systems missing WMI providers.
func (WMIHardwareQuerier) HardwareMap() (map[storage.RuleSource][]string, error) {
	si := sysinfo.SysInfo{}
	hw, err := si.ResolvedHardware()
	if err != nil {
		return nil, err
	}
	return map[storage.RuleSource][]string{
		storage.Cpu:         hw.Cpu,
		storage.Gpu:         hw.Gpu,
		storage.Memory:      hw.Memory,
		storage.Motherboard: hw.Motherboard,
		storage.Nic:         hw.Nic,
		storage.Storage:     hw.Storage,
	}, nil
}