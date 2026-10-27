package porter

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeZipInternal writes a manifest (last) plus files.
func writeZipInternal(t *testing.T, path string, m Manifest, files map[string][]byte) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	for name, content := range files {
		entry, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(content); err != nil {
			t.Fatal(err)
		}
	}

	data, _ := json.Marshal(m)
	entry, err := w.Create("manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write(data); err != nil {
		t.Fatal(err)
	}
}

// (e) local unreadable => Available=false, never an error.
func TestValidateZip_LocalReadError_AvailableFalse(t *testing.T) {
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, "conf"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "conf", "setting.json"), []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}

	zipPath := filepath.Join(t.TempDir(), "x.zip")
	writeZipInternal(t, zipPath, Manifest{
		FormatVersion: CurrentFormatVersion,
		ExportedAt:    time.Now(),
		Hashes:        Hashes{Settings: "deadbeef"},
	}, map[string][]byte{"conf/setting.json": []byte(`{}`)})

	orig := localFileHashFn
	defer func() { localFileHashFn = orig }()
	localFileHashFn = func(string) (string, int64, bool, error) {
		return "", 0, false, errors.New("read failure")
	}

	p := &Porter{DirRoot: base}
	preview, err := p.ValidateZip(zipPath)
	if err != nil {
		t.Fatalf("ValidateZip must not error on local read failure: %v", err)
	}
	if preview.Settings.Available {
		t.Errorf("expected Settings.Available=false on local read failure: %+v", preview.Settings)
	}
}

// (f) ImportFromFile must not touch the live filesystem for hashing.
func TestImportFromFile_DoesNotHashLocal(t *testing.T) {
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, "conf"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "conf", "setting.json"), []byte(`{"old":true}`), 0644); err != nil {
		t.Fatal(err)
	}

	zipPath := filepath.Join(base, "import.zip")
	writeZipInternal(t, zipPath, Manifest{
		FormatVersion: CurrentFormatVersion,
		ExportedAt:    time.Now(),
	}, map[string][]byte{"conf/setting.json": []byte(`{"new":true}`)})

	calls := 0
	orig := localFileHashFn
	defer func() { localFileHashFn = orig }()
	localFileHashFn = func(path string) (string, int64, bool, error) {
		calls++
		return localFileHash(path)
	}

	p := &Porter{DirRoot: base}
	if err := p.ImportFromFile(zipPath, ImportOptions{Settings: true}); err != nil {
		t.Fatalf("ImportFromFile: %v", err)
	}
	if calls != 0 {
		t.Errorf("ImportFromFile must not hash local files, got %d calls", calls)
	}
}
