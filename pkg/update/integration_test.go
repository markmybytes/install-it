package update

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"testing"
	"time"
)

// writeFile creates parent directories then writes content to path.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}

// TestCheckAndApplyUpdates_StagingFlow verifies the staged update apply logic:
//   - install-it.exe in stage is skipped (new binary is already live after TriggerNativeUpdate)
//   - internals/ is atomically replaced
//   - conf/ and drivers/ are protected (not overwritten by staged entries)
//   - .update_stage/ is removed on completion
func TestCheckAndApplyUpdates_StagingFlow(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "staging-flow-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	stageDir := filepath.Join(tmpDir, ".update_stage")

	// Staged entries
	writeFile(t, filepath.Join(stageDir, "install-it.exe"), "new-exe-binary")
	writeFile(t, filepath.Join(stageDir, "internals", "bin", "test.txt"), "webview2-asset")
	writeFile(t, filepath.Join(stageDir, "conf", "data.db"), "staged-user-data")
	writeFile(t, filepath.Join(stageDir, "drivers", "network", "mock.sys"), "staged-driver")

	ApplyStagedUpdates(tmpDir)

	// internals/ must be deployed to root.
	internalsFile := filepath.Join(tmpDir, "internals", "bin", "test.txt")
	if got, err := os.ReadFile(internalsFile); err != nil {
		t.Errorf("internals/bin/test.txt not deployed: %v", err)
	} else if string(got) != "webview2-asset" {
		t.Errorf("internals/bin/test.txt = %q, want %q", got, "webview2-asset")
	}

	// conf/ must NOT have been deployed (user-data protection).
	if _, err := os.Stat(filepath.Join(tmpDir, "conf", "data.db")); err == nil {
		t.Error("conf/data.db should not have been deployed from stage (user-data protection)")
	}

	// drivers/ must NOT have been deployed (user-data protection).
	if _, err := os.Stat(filepath.Join(tmpDir, "drivers", "network", "mock.sys")); err == nil {
		t.Error("drivers/network/mock.sys should not have been deployed from stage (user-data protection)")
	}

	// .update_stage/ must be fully scrubbed.
	if _, err := os.Stat(stageDir); !os.IsNotExist(err) {
		t.Error(".update_stage/ should have been removed after apply")
	}
}

// TestCheckAndApplyUpdates_LockGateRetry tests the Iron Gate retry loop:
// it creates an install-it.exe.old file, optionally locks it (Windows only),
// launches applyStagedUpdates in a goroutine, releases the lock, and asserts
// that the gate clears and the .old file is removed.
func TestCheckAndApplyUpdates_LockGateRetry(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lock-gate-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	oldExe := filepath.Join(tmpDir, "install-it.exe.old")
	if err := os.WriteFile(oldExe, []byte("old-exe-bytes"), 0644); err != nil {
		t.Fatal(err)
	}

	// Populate a minimal stage so applyStagedUpdates proceeds past the gate.
	stageDir := filepath.Join(tmpDir, ".update_stage")
	writeFile(t, filepath.Join(stageDir, "internals", "version.txt"), "v2-deployed")

	if runtime.GOOS == "windows" {
		// On Windows, syscall.Open uses FILE_SHARE_READ|FILE_SHARE_WRITE (no
		// FILE_SHARE_DELETE), so os.Remove will fail with a sharing violation
		// while the handle is open — exercising the retry loop.
		fd, err := syscall.Open(oldExe, syscall.O_RDONLY, 0)
		if err != nil {
			t.Fatalf("syscall.Open: %v", err)
		}

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			ApplyStagedUpdates(tmpDir)
		}()

		// Let the retry loop spin a few iterations before releasing.
		time.Sleep(350 * time.Millisecond)
		syscall.Close(fd)
		wg.Wait()
	} else {
		// On Unix, os.Remove on an open fd unlinks immediately; just verify
		// the gate detects and removes the file on first try.
		ApplyStagedUpdates(tmpDir)
	}

	// Gate must have deleted the .old file.
	if _, err := os.Stat(oldExe); !os.IsNotExist(err) {
		t.Error("install-it.exe.old should have been deleted by the Iron Gate")
	}

	// Stage must have been applied and cleaned up.
	if _, err := os.Stat(stageDir); !os.IsNotExist(err) {
		t.Error(".update_stage/ should have been removed after apply")
	}
	if _, err := os.ReadFile(filepath.Join(tmpDir, "internals", "version.txt")); err != nil {
		t.Errorf("internals/version.txt should have been deployed: %v", err)
	}
}

func assertNotExist(t *testing.T, path, msg string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("%s: %s should not exist", msg, path)
	}
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("ReadFile(%s): %v", path, err)
		return
	}
	if string(got) != want {
		t.Errorf("%s = %q, want %q", path, got, want)
	}
}

// TestApplyStagedUpdates_RecoveryMatrix exercises the idempotent recovery
// state machine: each partial-deploy state is set up in a fresh tmp dir and
// applyStagedUpdates must converge to a consistent layout. install-it.exe.old
// is the pending-deploy discriminator (swap runs only after extract finishes).
func TestApplyStagedUpdates_RecoveryMatrix(t *testing.T) {
	t.Run("crash mid-swap: staged+old, no live -> finish phase 2", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, ".update_stage", "internals", "v.txt"), "new")
		writeFile(t, filepath.Join(root, "internals.old", "v.txt"), "old")

		ApplyStagedUpdates(root)

		assertNotExist(t, filepath.Join(root, ".update_stage"), ".update_stage after apply")
		assertNotExist(t, filepath.Join(root, "internals.old"), "internals.old after apply")
		assertFileContent(t, filepath.Join(root, "internals", "v.txt"), "new")
	})

	t.Run("interrupted extract: staged+live, no exe.old -> discard staged", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, ".update_stage", "internals", "v.txt"), "new")
		writeFile(t, filepath.Join(root, "internals", "v.txt"), "old")

		ApplyStagedUpdates(root)

		assertNotExist(t, filepath.Join(root, ".update_stage"), ".update_stage after apply")
		assertNotExist(t, filepath.Join(root, "internals.old"), "internals.old after apply")
		assertFileContent(t, filepath.Join(root, "internals", "v.txt"), "old")
	})

	t.Run("staged gone: old only -> restore internals", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "internals.old", "v.txt"), "old")

		ApplyStagedUpdates(root)

		assertNotExist(t, filepath.Join(root, ".update_stage"), ".update_stage after apply")
		assertNotExist(t, filepath.Join(root, "internals.old"), "internals.old after apply")
		assertFileContent(t, filepath.Join(root, "internals", "v.txt"), "old")
	})

	t.Run("first deploy: staged only -> normal apply", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, ".update_stage", "internals", "v.txt"), "new")

		ApplyStagedUpdates(root)

		assertNotExist(t, filepath.Join(root, ".update_stage"), ".update_stage after apply")
		assertNotExist(t, filepath.Join(root, "internals.old"), "internals.old after apply")
		assertFileContent(t, filepath.Join(root, "internals", "v.txt"), "new")
	})

	t.Run("cleanup interrupted: staged gone, old+live -> drop backup", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "internals", "v.txt"), "new")
		writeFile(t, filepath.Join(root, "internals.old", "v.txt"), "old")

		ApplyStagedUpdates(root)

		assertNotExist(t, filepath.Join(root, ".update_stage"), ".update_stage after apply")
		assertNotExist(t, filepath.Join(root, "internals.old"), "internals.old after apply")
		assertFileContent(t, filepath.Join(root, "internals", "v.txt"), "new")
	})

	t.Run("pending replace: staged+live with exe.old marker -> two-phase replace", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, ".update_stage", "internals", "v.txt"), "new")
		writeFile(t, filepath.Join(root, "internals", "v.txt"), "old")
		writeFile(t, filepath.Join(root, "install-it.exe.old"), "old-exe")

		ApplyStagedUpdates(root)

		assertNotExist(t, filepath.Join(root, ".update_stage"), ".update_stage after apply")
		assertNotExist(t, filepath.Join(root, "internals.old"), "internals.old after apply")
		assertNotExist(t, filepath.Join(root, "install-it.exe.old"), "install-it.exe.old after Iron Gate")
		assertFileContent(t, filepath.Join(root, "internals", "v.txt"), "new")
	})
}
