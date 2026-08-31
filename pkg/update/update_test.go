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
	"strings"
	"testing"
	"time"

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
func buildReleaseJSON(tag string) []byte {
	payload := map[string]interface{}{
		"tag_name":     tag,
		"body":         "Release notes for " + tag,
		"published_at": "2024-06-01T00:00:00Z",
	}
	b, _ := json.Marshal(payload)
	return b
}

// TestCheckForUpdates covers four distinct scenarios via table-driven subtests.
func TestCheckForUpdates(t *testing.T) {
	tests := []struct {
		name             string
		localVer         string
		remoteTag        string
		preferPreRelease bool
		wantHasUpdate    bool
	}{
		{
			name:          "stable: remote newer triggers update",
			localVer:      "1.0.0",
			remoteTag:     "v1.1.0",
			wantHasUpdate: true,
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
		},
		{
			name:          "release metadata without assets still works",
			localVer:      "1.0.0",
			remoteTag:     "v1.2.0",
			wantHasUpdate: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			releaseJSON := buildReleaseJSON(tc.remoteTag)

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

			result, err := u.CheckForUpdates(tc.preferPreRelease)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.HasUpdate != tc.wantHasUpdate {
				t.Errorf("HasUpdate = %v, want %v", result.HasUpdate, tc.wantHasUpdate)
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
			"install-it.exe":              "binary",
			"internals/app.txt":           "hello",
			"internals/sub/nested.txt":    "world",
			"internals/sub/deep/data.bin": "binary",
		})
		zipPath := filepath.Join(tmpDir, "test.zip")
		os.WriteFile(zipPath, data, 0644)

		destDir := filepath.Join(tmpDir, "dest")
		if err := extractZipToDir(zipPath, destDir); err != nil {
			t.Fatalf("extractZipToDir: %v", err)
		}

		checks := map[string]string{
			filepath.Join(destDir, "install-it.exe"):                       "binary",
			filepath.Join(destDir, "internals", "app.txt"):                 "hello",
			filepath.Join(destDir, "internals", "sub", "nested.txt"):       "world",
			filepath.Join(destDir, "internals", "sub", "deep", "data.bin"): "binary",
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

// TestCheckForUpdates_Non200ReturnsErrInfoUnavailable verifies that a non-200
// GitHub API response (403/404/rate-limit) surfaces as errUpdateInfoUnavailable
// instead of being decoded as "no update".
func TestCheckForUpdates_Non200ReturnsErrInfoUnavailable(t *testing.T) {
	for _, prerelease := range []bool{false, true} {
		name := "stable"
		if prerelease {
			name = "pre-release"
		}
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.NotFound(w, r)
			}))
			defer srv.Close()

			u := &Updater{Version: mustVer("1.0.0"), apiBase: srv.URL}
			_, err := u.CheckForUpdates(prerelease)
			if err == nil {
				t.Fatal("expected error for 404 response, got nil")
			}
			if !strings.Contains(err.Error(), "errUpdateInfoUnavailable") {
				t.Errorf("error %q should carry errUpdateInfoUnavailable code", err.Error())
			}
		})
	}
}

// TestCheckForUpdates_TimeoutConfigurable verifies the default HTTP client
// carries the 30s timeout and that an injected client override is honored.
func TestCheckForUpdates_TimeoutConfigurable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	u := &Updater{apiBase: srv.URL}
	if _, err := u.httpGet(u.releasesURL(true)); err != nil {
		t.Fatalf("httpGet: %v", err)
	}
	if got := u.client.Timeout; got != 30*time.Second {
		t.Errorf("default client timeout = %v, want %v", got, 30*time.Second)
	}

	custom := &http.Client{Timeout: 5 * time.Second}
	u.client = custom
	if _, err := u.httpGet(u.releasesURL(true)); err != nil {
		t.Fatalf("httpGet with custom client: %v", err)
	}
	if got := u.client; got != custom {
		t.Errorf("injected client not honored: got %v", got)
	}
}

// TestExtractZipToDir_PassesThroughUnknownRoots verifies that unknown root
// entries are extracted as-is: the apply phase only deploys internals/, and
// the stage dir is scrubbed afterwards, so there is no need to reject them.
func TestExtractZipToDir_PassesThroughUnknownRoots(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "extract-root-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	data := makeZip(t, map[string]string{
		"rogue.txt":         "pwned",
		"internals/app.txt": "hello",
	})
	zipPath := filepath.Join(tmpDir, "rogue.zip")
	os.WriteFile(zipPath, data, 0644)

	destDir := filepath.Join(tmpDir, "dest")
	if err := extractZipToDir(zipPath, destDir); err != nil {
		t.Fatalf("extractZipToDir: %v", err)
	}

	// Unknown root entry is extracted as-is.
	if got, err := os.ReadFile(filepath.Join(destDir, "rogue.txt")); err != nil {
		t.Errorf("rogue.txt not extracted: %v", err)
	} else if string(got) != "pwned" {
		t.Errorf("rogue.txt = %q, want %q", got, "pwned")
	}

	// Legit internals/ is extracted too.
	if _, err := os.Stat(filepath.Join(destDir, "internals", "app.txt")); err != nil {
		t.Errorf("internals/app.txt not extracted: %v", err)
	}
}
