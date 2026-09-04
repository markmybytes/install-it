package cputemp

import "testing"

func TestStatus(t *testing.T) {
	defer lifecycle.Store(lifecycleNotStarted)

	tests := []struct {
		state int32
		want  string
	}{
		{lifecycleNotStarted, "not_started"},
		{lifecycleInitializing, "initializing"},
		{lifecycleUnavailable, "unavailable"},
		{lifecycleAvailable, "available"},
	}
	for _, tt := range tests {
		lifecycle.Store(tt.state)
		if got := Status(); got != tt.want {
			t.Errorf("Status() = %q, want %q", got, tt.want)
		}
	}
}

func TestInitMarksFailedInitializationComplete(t *testing.T) {
	if Status() == "initializing" {
		t.Fatal("CPU temperature initialization unexpectedly in progress")
	}

	if err := initDriver(t.TempDir()); err == nil {
		t.Fatal("Init unexpectedly succeeded without CPU temperature assets")
	}

	if Status() == "initializing" {
		t.Fatal("CPU temperature initialization still in progress after Init returned")
	}
	if Status() == "available" {
		t.Fatal("CPU temperature reported available after failed Init")
	}
}
