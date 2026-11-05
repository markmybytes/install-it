package porter

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
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

// testLocalFiles writes nested files under base (internal-package helper).
func testLocalFiles(t *testing.T, base string, files map[string][]byte) {
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

func testLocalDataFiles() map[string][]byte {
	return map[string][]byte{
		"conf/data.db":      []byte(strings.Repeat("x", 100)),
		"drivers/a/one.bin": []byte(strings.Repeat("a", 100)),
		"drivers/b/two.bin": []byte(strings.Repeat("b", 100)),
	}
}

func TestLocalDataHashes_TickProgress(t *testing.T) {
	base := t.TempDir()
	testLocalFiles(t, base, testLocalDataFiles())

	var lastDone, lastTotal int64
	calls := 0
	hashes, size, err := localDataHashes(base, func(done, total int64) error {
		lastDone, lastTotal = done, total
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("localDataHashes: %v", err)
	}
	if calls != 3 || lastDone != 300 || lastTotal != 300 {
		t.Errorf("tick: calls=%d lastDone=%d lastTotal=%d, want 3/300/300", calls, lastDone, lastTotal)
	}
	if size != 300 || len(hashes) != 3 {
		t.Errorf("size=%d hashes=%d, want 300/3", size, len(hashes))
	}
}

func TestLocalDataHashes_TickAbort(t *testing.T) {
	base := t.TempDir()
	testLocalFiles(t, base, testLocalDataFiles())

	hashes, size, err := localDataHashes(base, func(done, total int64) error {
		return errors.New("aborted")
	})
	if err == nil {
		t.Fatal("expected abort error")
	}
	if hashes != nil || size != 0 {
		t.Errorf("on abort hashes and size must be nil/0, got %v/%d", hashes, size)
	}
}

func TestValidateZip_RejectsWhenJobRunning(t *testing.T) {
	p := &Porter{}
	p.job = newJob()
	p.job.start()

	if _, err := p.ValidateZip(""); err == nil {
		t.Error("ValidateZip must reject while a job is running")
	}
}

// (d) canceled job mid-compare => buildPreview errors with errImportCanceled.
// (Cancel arrives via the tick so it can't be observed through public ValidateZip.)
func TestBuildPreview_CancelPropagates(t *testing.T) {
	base := t.TempDir()
	testLocalFiles(t, base, testLocalDataFiles())

	zipPath := filepath.Join(t.TempDir(), "x.zip")
	writeZipInternal(t, zipPath, Manifest{
		FormatVersion: CurrentFormatVersion,
		ExportedAt:    time.Now(),
		Hashes:        Hashes{Data: "deadbeef"},
	}, map[string][]byte{"conf/data.db": []byte("db")})

	s, err := scanZip(zipPath)
	if err != nil {
		t.Fatal(err)
	}

	p := &Porter{
		DirRoot: base,
		job:     newJob(),
	}
	p.job.start()
	p.job.cancel()

	_, err = p.buildPreview(s)
	if !errors.Is(err, errImportCanceled) {
		t.Fatalf("want errImportCanceled, got %v", err)
	}
}

// (e) local unreadable => Available=false, never an error — for BOTH rows.
func TestValidateZip_DataReadError_AvailableFalse(t *testing.T) {
	base := t.TempDir()
	testLocalFiles(t, base, testLocalDataFiles())

	zipPath := filepath.Join(t.TempDir(), "x.zip")
	writeZipInternal(t, zipPath, Manifest{
		FormatVersion: CurrentFormatVersion,
		ExportedAt:    time.Now(),
		Hashes:        Hashes{Data: "deadbeef"},
	}, map[string][]byte{"conf/data.db": []byte("db")})

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
	if preview.Data.Available {
		t.Errorf("expected Data.Available=false on local read failure: %+v", preview.Data)
	}
}

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
