package sysinfo

import "testing"

func TestCPUTemperatureBeforeInitialization(t *testing.T) {
	got, err := (SysInfo{}).CPUTemperature()
	if err != nil {
		t.Fatalf("CPUTemperature() error = %v", err)
	}
	if got != (CPUTemperatureResult{Status: "not_started"}) {
		t.Errorf("CPUTemperature() = %+v, want not_started result", got)
	}
}
