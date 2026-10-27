// Package porter_test provides external black-box tests for the porter package.
package porter_test

import (
	"archive/zip"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"install-it/pkg/porter"
	"install-it/pkg/status"
)

// ==================== helpers ====================

// createTestZip creates a zip archive at path containing the given files (name → content).
func createTestZip(t *testing.T, path string, files map[string][]byte) error {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	for name, content := range files {
		entry, err := w.Create(name)
		if err != nil {
			return err
		}
		if _, err := entry.Write(content); err != nil {
			return err
		}
	}
	return nil
}

// createExportZip creates a valid install-it ZIP with manifest.json and the given files.
func createExportZip(t *testing.T, path string, files map[string][]byte) error {
	t.Helper()
	return createExportZipWithHashes(t, path, porter.Manifest{
		FormatVersion: porter.CurrentFormatVersion,
		ExportedAt:    time.Now(),
	}, files)
}

// createExportZipWithHashes writes a manifest (last) plus the given files.
func createExportZipWithHashes(t *testing.T, path string, manifest porter.Manifest, files map[string][]byte) error {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	for name, content := range files {
		entry, err := w.Create(name)
		if err != nil {
			return err
		}
		if _, err := entry.Write(content); err != nil {
			return err
		}
	}

	data, err := json.Marshal(manifest)
	if err != nil {
		return err
	}
	entry, err := w.Create("manifest.json")
	if err != nil {
		return err
	}
	_, err = entry.Write(data)
	return err
}

// writeLocal writes nested files under base.
func writeLocal(t *testing.T, base string, files map[string][]byte) {
	t.Helper()
	for name, content := range files {
		full := filepath.Join(base, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, content, 0644); err != nil {
			t.Fatal(err)
		}
	}
}

// sha1Hex is a direct, independent digest for test expectations.
func sha1Hex(b []byte) string {
	sum := sha1.Sum(b)
	return hex.EncodeToString(sum[:])
}

// independentDataFold mirrors the documented Data recipe with stdlib only, so a
// drift in the production fold fails the test.
func independentDataFold(files map[string][]byte) string {
	var frames []string
	for name, content := range files {
		if name != "conf/data.db" && !strings.HasPrefix(name, "drivers/") {
			continue
		}
		frames = append(frames, fmt.Sprintf("%d:%s:%s\n", len(name), name, sha1Hex(content)))
	}
	sort.Strings(frames)

	h := sha1.New()
	for _, frame := range frames {
		h.Write([]byte(frame))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// ==================== JobSnapshot / Progress ====================

func TestPorter_StatusIdle(t *testing.T) {
	t.Parallel()

	p := &porter.Porter{}
	if p.Status() != status.Pending {
		t.Errorf("fresh Porter.Status(): got %v, want Pending", p.Status())
	}
}

func TestPorter_AbortNoJob(t *testing.T) {
	t.Parallel()

	p := &porter.Porter{}
	if err := p.Abort(); err == nil {
		t.Error("expected error from Abort with no running job, got nil")
	}
}

func TestPorter_ProgressNoJob(t *testing.T) {
	t.Parallel()

	p := &porter.Porter{}
	_, err := p.Progress()
	if err == nil {
		t.Error("expected error from Progress with no started job, got nil")
	}
}

func TestPorter_ExportCreatesZip(t *testing.T) {
	base := t.TempDir()
	confDir := filepath.Join(base, "conf")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(confDir, "setting.json"), []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}

	exportDest := t.TempDir()
	p := &porter.Porter{
		DirRoot: base,
		Targets: []string{confDir},
	}

	if err := p.Export(exportDest); err != nil {
		t.Fatalf("Export: %v", err)
	}

	zipPath := filepath.Join(exportDest, "install-it.zip")
	if _, err := os.Stat(zipPath); err != nil {
		t.Fatalf("install-it.zip not found after Export: %v", err)
	}

	// Verify manifest exists in zip
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer r.Close()

	hasManifest := false
	hasSetting := false
	for _, f := range r.File {
		if f.Name == "manifest.json" {
			hasManifest = true
		}
		if strings.HasSuffix(f.Name, "setting.json") {
			hasSetting = true
		}
	}
	if !hasManifest {
		t.Error("export zip is missing manifest.json")
	}
	if !hasSetting {
		t.Error("export zip is missing conf/setting.json")
	}
}

// ==================== ValidateZip ====================

func TestValidateZip_Valid(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	zipPath := filepath.Join(base, "test.zip")

	if err := createExportZip(t, zipPath, map[string][]byte{
		filepath.Join("conf", "setting.json"): []byte(`{"key":"val"}`),
		filepath.Join("conf", "data.db"):      []byte("fake-db-content"),
	}); err != nil {
		t.Fatalf("createExportZip: %v", err)
	}

	p := &porter.Porter{}
	preview, err := p.ValidateZip(zipPath)
	if err != nil {
		t.Fatalf("ValidateZip: %v", err)
	}

	if !preview.HasSettings {
		t.Error("expected HasSettings=true")
	}
	if !preview.HasDatabase {
		t.Error("expected HasDatabase=true")
	}
	if !preview.HasData {
		t.Error("expected HasData=true")
	}
}

func TestValidateZip_MissingManifest(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	zipPath := filepath.Join(base, "test.zip")

	if err := createTestZip(t, zipPath, map[string][]byte{
		"random.txt": []byte("hello"),
	}); err != nil {
		t.Fatalf("createTestZip: %v", err)
	}

	p := &porter.Porter{}
	_, err := p.ValidateZip(zipPath)
	if err == nil {
		t.Error("expected error for missing manifest, got nil")
	}
}

func TestValidateZip_NoRecognizedData(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	zipPath := filepath.Join(base, "test.zip")

	// Create a zip with manifest but no recognized data
	if err := createExportZip(t, zipPath, nil); err != nil {
		t.Fatalf("createExportZip: %v", err)
	}

	p := &porter.Porter{}
	_, err := p.ValidateZip(zipPath)
	if err == nil {
		t.Error("expected error for no recognized data, got nil")
	}
}

// ==================== ImportFromFile ====================

func TestPorter_ImportFromFileInvalidPath(t *testing.T) {
	t.Parallel()

	p := &porter.Porter{
		DirRoot: t.TempDir(),
	}

	err := p.ImportFromFile(filepath.Join(t.TempDir(), "nonexistent.zip"), porter.ImportOptions{Settings: true, Data: true})
	if err == nil {
		t.Error("expected error for non-existent zip path, got nil")
	}
}

func TestPorter_ImportFromFile_SettingsOnly(t *testing.T) {
	base := t.TempDir()
	confDir := filepath.Join(base, "conf")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Pre-create a setting.json so it gets backed up
	if err := os.WriteFile(filepath.Join(confDir, "setting.json"), []byte(`{"old":true}`), 0644); err != nil {
		t.Fatal(err)
	}
	// Pre-create a data.db that should remain untouched
	if err := os.WriteFile(filepath.Join(confDir, "data.db"), []byte("original-db"), 0644); err != nil {
		t.Fatal(err)
	}

	zipPath := filepath.Join(base, "import.zip")
	if err := createExportZip(t, zipPath, map[string][]byte{
		"conf/setting.json": []byte(`{"new":true}`),
	}); err != nil {
		t.Fatalf("createExportZip: %v", err)
	}

	p := &porter.Porter{
		DirRoot: base,
		Targets: []string{confDir},
	}

	if err := p.ImportFromFile(zipPath, porter.ImportOptions{Settings: true, Data: false}); err != nil {
		t.Fatalf("ImportFromFile: %v", err)
	}

	// setting.json should be updated
	settingData, err := os.ReadFile(filepath.Join(confDir, "setting.json"))
	if err != nil {
		t.Fatalf("read setting.json: %v", err)
	}
	if string(settingData) != `{"new":true}` {
		t.Errorf("setting.json content mismatch: got %q, want %q", string(settingData), `{"new":true}`)
	}

	// data.db should remain unchanged
	dbData, err := os.ReadFile(filepath.Join(confDir, "data.db"))
	if err != nil {
		t.Fatalf("read data.db: %v", err)
	}
	if string(dbData) != "original-db" {
		t.Errorf("data.db should not have changed: got %q, want %q", string(dbData), "original-db")
	}
}

func TestPorter_ImportFromFile_DataOnly(t *testing.T) {
	base := t.TempDir()
	confDir := filepath.Join(base, "conf")
	driversDir := filepath.Join(base, "drivers")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(driversDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Pre-create setting.json that should remain untouched
	if err := os.WriteFile(filepath.Join(confDir, "setting.json"), []byte(`{"keep":true}`), 0644); err != nil {
		t.Fatal(err)
	}

	zipPath := filepath.Join(base, "import.zip")
	if err := createExportZip(t, zipPath, map[string][]byte{
		"conf/data.db":               []byte("new-db-content"),
		"drivers/network/driver.inf": []byte("driver-inf"),
	}); err != nil {
		t.Fatalf("createExportZip: %v", err)
	}

	p := &porter.Porter{
		DirRoot: base,
		Targets: []string{confDir, driversDir},
	}

	if err := p.ImportFromFile(zipPath, porter.ImportOptions{Settings: false, Data: true}); err != nil {
		t.Fatalf("ImportFromFile: %v", err)
	}

	// data.db should be extracted
	dbData, err := os.ReadFile(filepath.Join(confDir, "data.db"))
	if err != nil {
		t.Fatalf("read data.db: %v", err)
	}
	if string(dbData) != "new-db-content" {
		t.Errorf("data.db mismatch: got %q, want %q", string(dbData), "new-db-content")
	}

	// setting.json should remain unchanged
	settingData, err := os.ReadFile(filepath.Join(confDir, "setting.json"))
	if err != nil {
		t.Fatalf("read setting.json: %v", err)
	}
	if string(settingData) != `{"keep":true}` {
		t.Errorf("setting.json should not have changed")
	}

	// driver should be extracted
	if _, err := os.Stat(filepath.Join(driversDir, "network", "driver.inf")); err != nil {
		t.Errorf("driver should have been extracted: %v", err)
	}
}

func TestPorter_ImportFromFile_Both(t *testing.T) {
	base := t.TempDir()
	confDir := filepath.Join(base, "conf")
	driversDir := filepath.Join(base, "drivers")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(driversDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(confDir, "setting.json"), []byte(`{"old":true}`), 0644); err != nil {
		t.Fatal(err)
	}

	zipPath := filepath.Join(base, "import.zip")
	if err := createExportZip(t, zipPath, map[string][]byte{
		"conf/setting.json":       []byte(`{"new":true}`),
		"conf/data.db":            []byte("new-db"),
		"drivers/display/gpu.inf": []byte("gpu-inf"),
	}); err != nil {
		t.Fatalf("createExportZip: %v", err)
	}

	p := &porter.Porter{
		DirRoot: base,
		Targets: []string{confDir, driversDir},
	}

	if err := p.ImportFromFile(zipPath, porter.ImportOptions{Settings: true, Data: true}); err != nil {
		t.Fatalf("ImportFromFile: %v", err)
	}

	// Verify all files
	settingData, _ := os.ReadFile(filepath.Join(confDir, "setting.json"))
	if string(settingData) != `{"new":true}` {
		t.Error("setting.json not updated")
	}
	dbData, _ := os.ReadFile(filepath.Join(confDir, "data.db"))
	if string(dbData) != "new-db" {
		t.Error("data.db not updated")
	}
	if _, err := os.Stat(filepath.Join(driversDir, "display", "gpu.inf")); err != nil {
		t.Error("gpu.inf not extracted")
	}
}

// ==================== Export → Import roundtrip ====================

func TestPorter_ExportImportRoundtrip(t *testing.T) {
	base := t.TempDir()
	confDir := filepath.Join(base, "conf")
	driversDir := filepath.Join(base, "drivers")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(driversDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write original files
	if err := os.WriteFile(filepath.Join(confDir, "setting.json"), []byte(`{"version":"1.0"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(confDir, "data.db"), []byte("original-db"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(driversDir, "network"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(driversDir, "network", "driver.inf"), []byte("original-driver"), 0644); err != nil {
		t.Fatal(err)
	}

	// Export
	exportDest := t.TempDir()
	exporter := &porter.Porter{
		DirRoot: base,
		Targets: []string{confDir, driversDir},
	}
	if err := exporter.Export(exportDest); err != nil {
		t.Fatalf("Export: %v", err)
	}

	zipPath := filepath.Join(exportDest, "install-it.zip")
	if _, err := os.Stat(zipPath); err != nil {
		t.Fatalf("zip not created: %v", err)
	}

	// Import into a fresh directory
	importBase := t.TempDir()
	importConf := filepath.Join(importBase, "conf")
	importDrivers := filepath.Join(importBase, "drivers")
	if err := os.MkdirAll(importConf, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(importDrivers, 0755); err != nil {
		t.Fatal(err)
	}
	// Pre-create same files so backup logic runs
	if err := os.WriteFile(filepath.Join(importConf, "setting.json"), []byte(`{"old":true}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(importConf, "data.db"), []byte("old-db"), 0644); err != nil {
		t.Fatal(err)
	}

	importer := &porter.Porter{
		DirRoot: importBase,
		Targets: []string{importConf, importDrivers},
	}
	if err := importer.ImportFromFile(zipPath, porter.ImportOptions{Settings: true, Data: true}); err != nil {
		t.Fatalf("ImportFromFile: %v", err)
	}

	// Verify files were imported
	gotSetting, _ := os.ReadFile(filepath.Join(importConf, "setting.json"))
	if string(gotSetting) != `{"version":"1.0"}` {
		t.Errorf("setting.json mismatch: got %q, want %q", string(gotSetting), `{"version":"1.0"}`)
	}
	gotDB, _ := os.ReadFile(filepath.Join(importConf, "data.db"))
	if string(gotDB) != "original-db" {
		t.Errorf("data.db mismatch: got %q, want %q", string(gotDB), "original-db")
	}
	gotDriver, _ := os.ReadFile(filepath.Join(importDrivers, "network", "driver.inf"))
	if string(gotDriver) != "original-driver" {
		t.Errorf("driver.inf mismatch: got %q, want %q", string(gotDriver), "original-driver")
	}
}

// ==================== Rollback ====================

func TestPorter_RollbackOnExtractionFailure(t *testing.T) {
	base := t.TempDir()
	confDir := filepath.Join(base, "conf")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Original file
	originalContent := `{"original":true}`
	if err := os.WriteFile(filepath.Join(confDir, "setting.json"), []byte(originalContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a valid export zip
	zipPath := filepath.Join(base, "import.zip")
	if err := createExportZip(t, zipPath, map[string][]byte{
		"conf/setting.json": []byte(`{"new":true}`),
	}); err != nil {
		t.Fatalf("createExportZip: %v", err)
	}

	// Inject a failure that prevents the backup rename from succeeding:
	//   - On Unix: make the conf directory non-writable (0500). os.Rename of a file
	//     out of the directory requires write permission on the directory.
	//   - On Windows: hold an open file handle. Go opens without FILE_SHARE_DELETE,
	//     so os.Rename of the open file fails with ERROR_SHARING_VIOLATION.
	if runtime.GOOS != "windows" {
		if err := os.Chmod(confDir, 0500); err != nil {
			t.Fatalf("Chmod: %v", err)
		}
		defer os.Chmod(confDir, 0755)
	} else {
		f, err := os.Open(filepath.Join(confDir, "setting.json"))
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		defer f.Close()
	}

	p := &porter.Porter{
		DirRoot: base,
		Targets: []string{confDir},
	}

	err := p.ImportFromFile(zipPath, porter.ImportOptions{Settings: true, Data: false})
	if err == nil {
		t.Fatal("expected error when destination is read-only, got nil")
	}

	// On Unix, restore permissions to read the file
	if runtime.GOOS != "windows" {
		os.Chmod(confDir, 0755)
	}

	// Original should be restored from backup
	settingData, readErr := os.ReadFile(filepath.Join(confDir, "setting.json"))
	if readErr != nil {
		t.Fatalf("read setting.json: %v", readErr)
	}
	if string(settingData) != originalContent {
		t.Errorf("setting.json should be restored to original: got %q, want %q", string(settingData), originalContent)
	}
}

// TestBackupCleanupRoundtrip tests backup, extraction, and cleanupBackups in a complete flow
// through the public API.
func TestPorter_BackupCleanupRoundtrip(t *testing.T) {
	base := t.TempDir()
	confDir := filepath.Join(base, "conf")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		t.Fatal(err)
	}
	originalContent := `{"original":true}`
	if err := os.WriteFile(filepath.Join(confDir, "setting.json"), []byte(originalContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a valid export zip
	zipPath := filepath.Join(base, "import.zip")
	if err := createExportZip(t, zipPath, map[string][]byte{
		"conf/setting.json": []byte(`{"new":true}`),
	}); err != nil {
		t.Fatalf("createExportZip: %v", err)
	}

	p := &porter.Porter{
		DirRoot: base,
		Targets: []string{confDir},
	}

	if err := p.ImportFromFile(zipPath, porter.ImportOptions{Settings: true, Data: false}); err != nil {
		t.Fatalf("ImportFromFile: %v", err)
	}

	// Original should be replaced with new content
	settingData, err := os.ReadFile(filepath.Join(confDir, "setting.json"))
	if err != nil {
		t.Fatalf("read setting.json: %v", err)
	}
	if string(settingData) != `{"new":true}` {
		t.Errorf("setting.json content mismatch: got %q, want %q", string(settingData), `{"new":true}`)
	}

	// Backup files should be cleaned up
	// Walk the whole tree to find any leftover backups
	var leftovers []string
	filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
		if err == nil && strings.Contains(path, ".porter-bak-") {
			leftovers = append(leftovers, path)
		}
		return nil
	})
	if len(leftovers) > 0 {
		t.Errorf("backup files should have been cleaned up, found: %v", leftovers)
	}
}

// ==================== RecoverOrphanedBackups ====================

func TestPorter_RecoverOrphanedBackups(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	confDir := filepath.Join(base, "conf")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Simulate an orphaned backup: the original was moved into .porter-{ts}/conf/setting.json
	originalContent := `{"survived":true}`
	timestamp := "20250101T120000"
	backupDir := filepath.Join(base, ".porter-"+timestamp)
	if err := os.MkdirAll(filepath.Join(backupDir, "conf"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(backupDir, "conf", "setting.json"), []byte(originalContent), 0644); err != nil {
		t.Fatal(err)
	}

	p := &porter.Porter{
		DirRoot: base,
	}
	if err := p.RecoverOrphanedBackups(); err != nil {
		t.Fatalf("RecoverOrphanedBackups: %v", err)
	}

	// The original file should be restored
	data, err := os.ReadFile(filepath.Join(confDir, "setting.json"))
	if err != nil {
		t.Fatalf("read setting.json after recovery: %v", err)
	}
	if string(data) != originalContent {
		t.Errorf("setting.json content mismatch: got %q, want %q", string(data), originalContent)
	}

	// The backup folder should be gone
	if _, err := os.Stat(backupDir); !os.IsNotExist(err) {
		t.Error("backup folder should have been removed after recovery")
	}
}

func TestPorter_RecoverOrphanedBackups_Dir(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	driversDir := filepath.Join(base, "drivers")
	if err := os.MkdirAll(driversDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Simulate orphaned backup of a directory: drivers/ was moved into .porter-{ts}/drivers/
	timestamp := "20250101T120000"
	backupDir := filepath.Join(base, ".porter-"+timestamp)
	backupDrivers := filepath.Join(backupDir, "drivers")
	if err := os.MkdirAll(filepath.Join(backupDrivers, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(backupDrivers, "sub", "driver.exe"), []byte("driver"), 0644); err != nil {
		t.Fatal(err)
	}

	p := &porter.Porter{
		DirRoot: base,
	}
	if err := p.RecoverOrphanedBackups(); err != nil {
		t.Fatalf("RecoverOrphanedBackups: %v", err)
	}

	// The original directory should be restored
	if _, err := os.Stat(filepath.Join(driversDir, "sub", "driver.exe")); os.IsNotExist(err) {
		t.Error("drivers directory should have been restored")
	}

	// The backup folder should be gone
	if _, err := os.Stat(backupDir); !os.IsNotExist(err) {
		t.Error("backup folder should have been removed after recovery")
	}
}

// ==================== OnBeforeBackup/OnAfterImport hooks ====================

func TestPorter_HooksCalledWhenDBInBackup(t *testing.T) {
	base := t.TempDir()
	confDir := filepath.Join(base, "conf")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Pre-create data.db to trigger hook
	if err := os.WriteFile(filepath.Join(confDir, "data.db"), []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(confDir, "setting.json"), []byte(`{"old":true}`), 0644); err != nil {
		t.Fatal(err)
	}

	zipPath := filepath.Join(base, "import.zip")
	if err := createExportZip(t, zipPath, map[string][]byte{
		"conf/data.db":      []byte("new-db"),
		"conf/setting.json": []byte(`{"new":true}`),
	}); err != nil {
		t.Fatalf("createExportZip: %v", err)
	}

	beforeCalled := false
	afterCalled := false

	p := &porter.Porter{
		DirRoot: base,
		Targets: []string{confDir},
		OnBeforeBackup: func() error {
			beforeCalled = true
			return nil
		},
		OnAfterImport: func() error {
			afterCalled = true
			return nil
		},
	}

	if err := p.ImportFromFile(zipPath, porter.ImportOptions{Settings: true, Data: true}); err != nil {
		t.Fatalf("ImportFromFile: %v", err)
	}

	if !beforeCalled {
		t.Error("OnBeforeBackup was not called")
	}
	if !afterCalled {
		t.Error("OnAfterImport was not called")
	}
}

func TestPorter_HooksNotCalledWhenDBNotInBackup(t *testing.T) {
	base := t.TempDir()
	confDir := filepath.Join(base, "conf")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Only setting.json exists, no data.db
	if err := os.WriteFile(filepath.Join(confDir, "setting.json"), []byte(`{"old":true}`), 0644); err != nil {
		t.Fatal(err)
	}

	zipPath := filepath.Join(base, "import.zip")
	if err := createExportZip(t, zipPath, map[string][]byte{
		"conf/setting.json": []byte(`{"new":true}`),
	}); err != nil {
		t.Fatalf("createExportZip: %v", err)
	}

	beforeCalled := false
	afterCalled := false

	p := &porter.Porter{
		DirRoot: base,
		Targets: []string{confDir},
		OnBeforeBackup: func() error {
			beforeCalled = true
			return nil
		},
		OnAfterImport: func() error {
			afterCalled = true
			return nil
		},
	}

	if err := p.ImportFromFile(zipPath, porter.ImportOptions{Settings: true, Data: false}); err != nil {
		t.Fatalf("ImportFromFile: %v", err)
	}

	if beforeCalled {
		t.Error("OnBeforeBackup should not have been called (data.db not in backup set)")
	}
	if afterCalled {
		t.Error("OnAfterImport should not have been called")
	}
}

// ==================== Manifest hashes / preview diff ====================

func TestExport_ManifestLastWithHashes(t *testing.T) {
	base := t.TempDir()
	files := map[string][]byte{
		"conf/setting.json":      []byte(`{"a":1}`),
		"conf/data.db":           []byte("db-bytes"),
		"drivers/net/driver.inf": []byte("inf-bytes"),
	}
	writeLocal(t, base, files)

	dest := t.TempDir()
	p := &porter.Porter{
		DirRoot: base,
		Targets: []string{filepath.Join(base, "conf"), filepath.Join(base, "drivers")},
	}
	if err := p.Export(dest); err != nil {
		t.Fatalf("Export: %v", err)
	}

	r, err := zip.OpenReader(filepath.Join(dest, "install-it.zip"))
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer r.Close()

	last := r.File[len(r.File)-1]
	if last.Name != "manifest.json" {
		t.Fatalf("manifest must be the last entry, got %q", last.Name)
	}

	rc, err := last.Open()
	if err != nil {
		t.Fatalf("open manifest: %v", err)
	}
	defer rc.Close()

	var m porter.Manifest
	if err := json.NewDecoder(rc).Decode(&m); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}

	if got, want := m.Hashes.Settings, sha1Hex(files["conf/setting.json"]); got != want {
		t.Errorf("settings hash: got %q want %q", got, want)
	}
	if got, want := m.Hashes.Data, independentDataFold(files); got != want {
		t.Errorf("data hash: got %q want %q", got, want)
	}
}

// TestExport_SettingsOnly_NoDataKey: a settings-only export (no conf/data.db,
// no drivers/**) folds to an empty Data digest, so the manifest must omit the
// "data" key entirely rather than emitting a non-empty sha1("") digest.
func TestExport_SettingsOnly_NoDataKey(t *testing.T) {
	base := t.TempDir()
	writeLocal(t, base, map[string][]byte{"conf/setting.json": []byte(`{"a":1}`)})

	dest := t.TempDir()
	p := &porter.Porter{
		DirRoot: base,
		Targets: []string{filepath.Join(base, "conf")},
	}
	if err := p.Export(dest); err != nil {
		t.Fatalf("Export: %v", err)
	}

	r, err := zip.OpenReader(filepath.Join(dest, "install-it.zip"))
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer r.Close()

	found := false
	for _, f := range r.File {
		if f.Name != "manifest.json" {
			continue
		}
		found = true
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open manifest: %v", err)
		}
		defer rc.Close()

		var m struct {
			Hashes map[string]json.RawMessage `json:"hashes"`
		}
		if err := json.NewDecoder(rc).Decode(&m); err != nil {
			t.Fatalf("decode manifest: %v", err)
		}
		if data, ok := m.Hashes["data"]; ok {
			t.Errorf("settings-only export must omit hashes.data, got %s", data)
		}
		if _, ok := m.Hashes["settings"]; !ok {
			t.Errorf("settings hash should be present: %+v", m.Hashes)
		}
	}
	if !found {
		t.Fatal("manifest.json not found in export")
	}
}

func TestValidateZip_PreviewCompare(t *testing.T) {
	base := t.TempDir()
	files := map[string][]byte{
		"conf/setting.json":      []byte(`{"a":1}`),
		"conf/data.db":           []byte("db-bytes"),
		"drivers/net/driver.inf": []byte("inf-bytes"),
	}
	writeLocal(t, base, files)

	dest := t.TempDir()
	p := &porter.Porter{
		DirRoot: base,
		Targets: []string{filepath.Join(base, "conf"), filepath.Join(base, "drivers")},
	}
	if err := p.Export(dest); err != nil {
		t.Fatalf("Export: %v", err)
	}
	zipPath := filepath.Join(dest, "install-it.zip")

	preview, err := p.ValidateZip(zipPath)
	if err != nil {
		t.Fatalf("ValidateZip: %v", err)
	}
	if !preview.Settings.Available || !preview.Settings.Identical {
		t.Errorf("settings should be available+identical: %+v", preview.Settings)
	}
	if !preview.Data.Available || !preview.Data.Identical {
		t.Errorf("data should be available+identical: %+v", preview.Data)
	}
	if !preview.Settings.LocalPresent || !preview.Data.LocalPresent {
		t.Errorf("local should be present: %+v %+v", preview.Settings, preview.Data)
	}
	if preview.Data.LocalCount != 2 {
		t.Errorf("data local count: got %d want 2", preview.Data.LocalCount)
	}

	// Modify a local driver file: data differs, settings untouched.
	writeLocal(t, base, map[string][]byte{"drivers/net/driver.inf": []byte("changed")})

	preview2, err := p.ValidateZip(zipPath)
	if err != nil {
		t.Fatalf("ValidateZip (modified): %v", err)
	}
	if !preview2.Data.Available {
		t.Errorf("data should remain available after local edit: %+v", preview2.Data)
	}
	if preview2.Data.Identical {
		t.Error("data should not be identical after local modification")
	}
	if !preview2.Settings.Identical {
		t.Error("settings should remain identical")
	}
}

func TestValidateZip_LegacyManifestNoHashes(t *testing.T) {
	base := t.TempDir()
	zipPath := filepath.Join(base, "legacy.zip")
	if err := createExportZip(t, zipPath, map[string][]byte{
		"conf/setting.json": []byte(`{}`),
		"conf/data.db":      []byte("x"),
	}); err != nil {
		t.Fatalf("createExportZip: %v", err)
	}

	p := &porter.Porter{DirRoot: t.TempDir()}
	preview, err := p.ValidateZip(zipPath)
	if err != nil {
		t.Fatalf("ValidateZip: %v", err)
	}
	if preview.Settings.Available || preview.Data.Available {
		t.Errorf("legacy manifest must yield Available=false: %+v %+v", preview.Settings, preview.Data)
	}
}

func TestValidateZip_HashPresentEntryMissing(t *testing.T) {
	base := t.TempDir()
	zipPath := filepath.Join(base, "x.zip")
	if err := createExportZipWithHashes(t, zipPath, porter.Manifest{
		FormatVersion: porter.CurrentFormatVersion,
		ExportedAt:    time.Now(),
		Hashes:        porter.Hashes{Settings: "aaaa", Data: "bbbb"},
	}, map[string][]byte{
		"conf/setting.json": []byte(`{}`),
	}); err != nil {
		t.Fatalf("createExportZipWithHashes: %v", err)
	}

	p := &porter.Porter{DirRoot: t.TempDir()}
	preview, err := p.ValidateZip(zipPath)
	if err != nil {
		t.Fatalf("ValidateZip: %v", err)
	}
	if preview.Data.Available {
		t.Error("data hash without data entries must be Available=false")
	}
	if preview.Data.IncomingPresent {
		t.Error("data IncomingPresent must be false when entries missing")
	}
	if !preview.Settings.Available {
		t.Errorf("settings should still be available: %+v", preview.Settings)
	}
	if preview.Settings.LocalPresent || preview.Settings.Identical {
		t.Errorf("missing local settings: expect LocalPresent=false, Identical=false: %+v", preview.Settings)
	}
}
