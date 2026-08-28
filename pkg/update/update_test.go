package update

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Masterminds/semver"
)

// mustVer parses a semver string and panics on failure; used only in tests.
func mustVer(v string) *semver.Version {
	ver, err := semver.NewVersion(v)
	if err != nil {
		panic(fmt.Sprintf("mustVer(%q): %v", v, err))
	}
	return ver
}

// buildReleaseJSON returns a single GitHub release payload.
func buildReleaseJSON(tag string, withBundled bool) []byte {
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x64"
	} else if arch == "386" {
		arch = "x86"
	}
	prefix := fmt.Sprintf("install-it.%s-%s", runtime.GOOS, arch)

	assets := []map[string]string{
		{"name": prefix + ".zip", "browser_download_url": "http://dl.example.com/" + prefix + ".zip"},
	}
	if withBundled {
		assets = append(assets, map[string]string{
			"name":                 prefix + "-bundled.zip",
			"browser_download_url": "http://dl.example.com/" + prefix + "-bundled.zip",
		})
	}

	payload := map[string]interface{}{
		"tag_name":     tag,
		"body":         "Release notes for " + tag,
		"published_at": "2024-06-01T00:00:00Z",
		"assets":       assets,
	}
	b, _ := json.Marshal(payload)
	return b
}

// TestCheckForUpdates covers four distinct scenarios via table-driven subtests.
func TestCheckForUpdates(t *testing.T) {
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x64"
	} else if arch == "386" {
		arch = "x86"
	}
	prefix := fmt.Sprintf("install-it.%s-%s", runtime.GOOS, arch)
	stdURL := "http://dl.example.com/" + prefix + ".zip"
	bundleURL := "http://dl.example.com/" + prefix + "-bundled.zip"

	tests := []struct {
		name             string
		localVer         string
		remoteTag        string
		preferPreRelease bool
		preferBundled    bool
		withBundled      bool
		wantHasUpdate    bool
		wantDownloadURL  string
	}{
		{
			name:            "stable: remote newer triggers update",
			localVer:        "1.0.0",
			remoteTag:       "v1.1.0",
			wantHasUpdate:   true,
			wantDownloadURL: stdURL,
		},
		{
			name:          "stable: local >= remote means no update",
			localVer:      "1.1.0",
			remoteTag:     "v1.1.0",
			wantHasUpdate: false,
		},
		{
			name:             "pre-release: newest index item selected",
			localVer:         "1.0.0",
			remoteTag:        "v2.0.0-beta.1",
			preferPreRelease: true,
			wantHasUpdate:    true,
			wantDownloadURL:  stdURL,
		},
		{
			name:            "bundled asset preferred when preferBundled=true and available",
			localVer:        "1.0.0",
			remoteTag:       "v1.2.0",
			preferBundled:   true,
			withBundled:     true,
			wantHasUpdate:   true,
			wantDownloadURL: bundleURL,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			releaseJSON := buildReleaseJSON(tc.remoteTag, tc.withBundled)

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if strings.HasSuffix(r.URL.Path, "/releases/latest") {
					w.Write(releaseJSON)
				} else {
					// /releases — return array
					fmt.Fprintf(w, "[%s]", releaseJSON)
				}
			}))
			defer srv.Close()

			u := &Updater{
				Version: mustVer(tc.localVer),
				apiBase: srv.URL,
			}

			result, err := u.CheckForUpdates(tc.preferBundled, tc.preferPreRelease)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.HasUpdate != tc.wantHasUpdate {
				t.Errorf("HasUpdate = %v, want %v", result.HasUpdate, tc.wantHasUpdate)
			}
			if tc.wantDownloadURL != "" && result.DownloadUrl != tc.wantDownloadURL {
				t.Errorf("DownloadUrl = %q, want %q", result.DownloadUrl, tc.wantDownloadURL)
			}
			if tc.withBundled && result.DownloadUrlBundled != bundleURL {
				t.Errorf("DownloadUrlBundled = %q, want %q", result.DownloadUrlBundled, bundleURL)
			}
		})
	}
}

// makeZip creates an in-memory zip archive with the given name→content entries.
func makeZip(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("zip.Create(%q): %v", name, err)
		}
		w.Write([]byte(content))
	}
	zw.Close()
	return buf.Bytes()
}

// TestExtractZipToDir validates normal extraction and zip-slip rejection.
func TestExtractZipToDir(t *testing.T) {
	t.Run("normal: files and subdirectories extracted correctly", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "extract-normal-*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		data := makeZip(t, map[string]string{
			"file.txt":          "hello",
			"sub/nested.txt":    "world",
			"sub/deep/data.bin": "binary",
		})
		zipPath := filepath.Join(tmpDir, "test.zip")
		os.WriteFile(zipPath, data, 0644)

		destDir := filepath.Join(tmpDir, "dest")
		if err := extractZipToDir(zipPath, destDir); err != nil {
			t.Fatalf("extractZipToDir: %v", err)
		}

		checks := map[string]string{
			filepath.Join(destDir, "file.txt"):                "hello",
			filepath.Join(destDir, "sub", "nested.txt"):       "world",
			filepath.Join(destDir, "sub", "deep", "data.bin"): "binary",
		}
		for path, want := range checks {
			got, err := os.ReadFile(path)
			if err != nil {
				t.Errorf("ReadFile(%q): %v", path, err)
				continue
			}
			if string(got) != want {
				t.Errorf("%q = %q, want %q", path, got, want)
			}
		}
	})

	t.Run("zip-slip: traversal path returns error", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "extract-slip-*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		// Craft a zip entry that escapes destDir.
		var buf bytes.Buffer
		zw := zip.NewWriter(&buf)
		w, _ := zw.Create("../evil.txt")
		w.Write([]byte("pwned"))
		zw.Close()

		zipPath := filepath.Join(tmpDir, "slip.zip")
		os.WriteFile(zipPath, buf.Bytes(), 0644)

		destDir := filepath.Join(tmpDir, "dest")
		err = extractZipToDir(zipPath, destDir)
		if err == nil {
			t.Fatal("expected error for zip-slip entry, got nil")
		}
		if !strings.Contains(err.Error(), "errUpdateZipSlip") {
			t.Errorf("error %q should carry errUpdateZipSlip code", err.Error())
		}

		// The traversal target must not have been created.
		if _, statErr := os.Stat(filepath.Join(tmpDir, "evil.txt")); !os.IsNotExist(statErr) {
			t.Error("zip-slip file was created outside destDir")
		}
	})
}
