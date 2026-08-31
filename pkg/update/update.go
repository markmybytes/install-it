package update

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"install-it/pkg/errcode"
	"install-it/pkg/utils"

	"github.com/Masterminds/semver"
)

const httpTimeout = 30 * time.Second

type Updater struct {
	DirRoot string
	Version *semver.Version
	// apiBase overrides the GitHub API root; set in tests via httptest.
	apiBase string
	// client overrides the HTTP client; set in tests.
	client *http.Client
}

func (u *Updater) httpGet(url string) (*http.Response, error) {
	if u.client == nil {
		u.client = &http.Client{Timeout: httpTimeout}
	}
	return u.client.Get(url)
}

func (u *Updater) releasesURL(latest bool) string {
	base := "https://api.github.com/repos/markmybytes/install-it"
	if u.apiBase != "" {
		base = u.apiBase
	}
	if latest {
		return base + "/releases/latest"
	}
	return base + "/releases"
}

type UpdateCheckResult struct {
	HasUpdate     bool   `json:"hasUpdate"`
	LatestVersion string `json:"latestVersion"`
	ReleaseNotes  string `json:"releaseNotes"`
	ReleaseAt     string `json:"releaseAt"`
}

func (u *Updater) CheckForUpdates(preferPreRelease bool) (*UpdateCheckResult, error) {
	var body struct {
		TagName     string `json:"tag_name"`
		Body        string `json:"body"`
		PublishedAt string `json:"published_at"`
	}

	if preferPreRelease {
		resp, err := u.httpGet(u.releasesURL(false))
		if err != nil {
			return nil, errcode.New("errUpdateCheckFailed")
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, errcode.New("errUpdateInfoUnavailable")
		}

		var releases []struct {
			TagName     string `json:"tag_name"`
			Body        string `json:"body"`
			PublishedAt string `json:"published_at"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
			return nil, errcode.New("errUpdateCheckFailed")
		}
		if len(releases) == 0 {
			return nil, errcode.New("errUpdateNoReleases")
		}
		body = releases[0]
	} else {
		resp, err := u.httpGet(u.releasesURL(true))
		if err != nil {
			return nil, errcode.New("errUpdateCheckFailed")
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, errcode.New("errUpdateInfoUnavailable")
		}

		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			return nil, errcode.New("errUpdateCheckFailed")
		}
	}

	latestVer, _ := semver.NewVersion(body.TagName)
	return &UpdateCheckResult{
		HasUpdate:     latestVer != nil && latestVer.GreaterThan(u.Version),
		LatestVersion: body.TagName,
		ReleaseNotes:  body.Body,
		ReleaseAt:     body.PublishedAt,
	}, nil
}

func (u *Updater) TriggerNativeUpdate(tag string, preferBundled bool) error {
	if tag == "" {
		return errcode.New("errUpdateInfoUnavailable")
	}

	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x64"
	} else if arch == "386" {
		arch = "x86"
	}

	assetName := fmt.Sprintf("install-it.%s-%s", runtime.GOOS, arch)
	if preferBundled {
		assetName += "-bundled"
	}

	downloadURL := fmt.Sprintf(
		"https://github.com/markmybytes/install-it/releases/download/%s/%s.zip",
		tag, assetName,
	)

	resp, err := u.httpGet(strings.TrimSuffix(u.releasesURL(true), "/releases/latest") + "/releases/tags/" + tag)
	if err != nil {
		return errcode.New("errUpdateCheckFailed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errcode.New("errUpdateInfoUnavailable")
	}

	var payload struct {
		Assets []struct {
			Name   string `json:"name"`
			Digest string `json:"digest"`
		} `json:"assets"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return errcode.New("errUpdateCheckFailed")
	}

	var digest string
	for _, asset := range payload.Assets {
		if asset.Name == assetName+".zip" {
			digest = asset.Digest
			break
		}
	}
	if digest == "" {
		return errcode.New("errUpdateInfoUnavailable")
	}

	tmpZip, err := os.CreateTemp(u.DirRoot, "update-*.zip")
	if err != nil {
		return errcode.New("errUpdateWriteFailed")
	}
	defer os.Remove(tmpZip.Name())

	resp, err = u.httpGet(downloadURL)
	if err != nil {
		return errcode.New("errUpdateDownloadFailed")
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return errcode.New("errUpdateInfoUnavailable")
	}

	if _, err := io.Copy(tmpZip, resp.Body); err != nil {
		tmpZip.Close()
		resp.Body.Close()
		return errcode.New("errUpdateWriteFailed")
	}

	tmpZip.Close()
	resp.Body.Close()

	if !utils.VerifySHA256(strings.TrimPrefix(digest, "sha256:"), tmpZip.Name()) {
		return errcode.New("errUpdateChecksumMismatch")
	}

	stageDir := filepath.Join(u.DirRoot, ".update_stage")
	os.RemoveAll(stageDir)
	if err := extractZipToDir(tmpZip.Name(), stageDir); err != nil {
		os.RemoveAll(stageDir)
		return err
	}

	exe := filepath.Join(u.DirRoot, "install-it.exe")
	old := filepath.Join(u.DirRoot, "install-it.exe.old")

	if err := os.Rename(exe, old); err != nil {
		os.RemoveAll(stageDir)
		return errcode.New("errUpdateRenameFailed")
	}
	if err := os.Rename(filepath.Join(stageDir, "install-it.exe"), exe); err != nil {
		os.Rename(old, exe) // rollback
		os.RemoveAll(stageDir)
		return errcode.New("errUpdateRenameFailed")
	}

	cmd := exec.Command(exe)
	cmd.Dir = u.DirRoot
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000008}
	if err := cmd.Start(); err != nil {
		return errcode.New("errUpdateSpawnFailed")
	}

	os.Remove(tmpZip.Name())
	os.Exit(0)
	return nil
}

// ApplyStagedUpdates deploys a staged update's internals. Idempotent and safe
// to re-run after a crash: a recovery state machine first completes or rolls
// back any half-finished deployment, then the normal path replaces internals
// with an explicit two-phase commit. Designed to run pre-Wails (before
// WebView2 loads internals) — internals failures never abort the app: the old
// internals simply stay live, which is not a brick. Free function over dirRoot
// so main() can call it without going through the Wails bridge.
func ApplyStagedUpdates(dirRoot string) {
	stageDir := filepath.Join(dirRoot, ".update_stage")
	staged := filepath.Join(stageDir, "internals")
	oldLive := filepath.Join(dirRoot, "internals.old")
	live := filepath.Join(dirRoot, "internals")
	exeOld := filepath.Join(dirRoot, "install-it.exe.old")

	exists := func(p string) bool {
		_, err := os.Stat(p)
		return err == nil
	}

	// --- Idempotent recovery state machine ---
	//
	// Two-phase commit states:
	//   1. live -> internals.old          (phase 1)
	//   2. staged -> live                 (phase 2)
	//   3. remove internals.old           (cleanup)
	// A crash between any two steps leaves a detectable partial state that is
	// completed or rolled back here. install-it.exe.old discriminates a
	// completed exe swap (staged internals is a pending deploy) from an
	// interrupted extract (staged internals is untrusted garbage): the exe
	// swap runs only after extraction completes, so its presence proves
	// extraction finished.
	switch {
	case exists(staged) && !exists(live) && exists(oldLive):
		// Crash mid-swap (phase 1 done, phase 2 pending): finish the swap.
		if err := os.Rename(staged, live); err == nil {
			os.RemoveAll(oldLive)
		}
	case exists(staged) && exists(live) && !exists(exeOld):
		// Staged leftover from an interrupted extract: live internals are
		// intact, discard the untrusted staged copy.
		os.RemoveAll(staged)
	case !exists(staged) && !exists(live) && exists(oldLive):
		// Staged payload vanished; restore the backed-up internals.
		os.Rename(oldLive, live)
	case !exists(staged) && exists(live) && exists(oldLive):
		// Phase 2 completed but cleanup was interrupted; drop the backup.
		os.RemoveAll(oldLive)
	}
	// Remaining states fall through to the normal path below: staged+live+exeOld
	// (pending replace) and staged+!live+!oldLive (first deploy).

	// --- Two-phase commit for internals ---
	if exists(staged) {
		if exists(live) {
			// Phase 1: back up the live internals.
			if err := os.Rename(live, oldLive); err != nil {
				os.RemoveAll(stageDir)
				return
			}
		}
		// Phase 2: staged internals become live.
		if err := os.Rename(staged, live); err != nil {
			if _, err := os.Stat(oldLive); err == nil {
				// Rollback (best effort).
				os.Rename(oldLive, live)
			}
			os.RemoveAll(stageDir)
			return
		}
		// Cleanup: drop the backup only after both phases succeeded.
		os.RemoveAll(oldLive)
	}

	// Iron Gate runs LAST so exeOld stays intact as the pending-deploy
	// discriminator (case 2's `!exists(exeOld)` gate) across any crash
	// between recovery and commit. Deletion has no dependency on internals.
	// 30s budget — 300 × 100ms keeps the 100ms granularity of the original 10s loop.
	if _, err := os.Stat(exeOld); err == nil {
		deleted := false
		for i := 0; i < 300; i++ {
			if os.Remove(exeOld) == nil {
				deleted = true
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if !deleted {
			// Old exe still locked; abort deploy. Old internals stay live and
			// the next launch retries.
			return
		}
	}

	// Stage is fully consumed; scrub it regardless of outcome.
	os.RemoveAll(stageDir)
}

func extractZipToDir(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return errcode.New("errUpdateExtractFailed")
	}
	defer r.Close()

	cleanDest := filepath.Clean(destDir) + string(os.PathSeparator)

	for _, f := range r.File {
		target := filepath.Join(destDir, filepath.FromSlash(f.Name))
		if !strings.HasPrefix(target, cleanDest) {
			return errcode.Newf("errUpdateZipSlip", map[string]any{"entry": f.Name})
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, f.Mode()); err != nil {
				return errcode.New("errUpdateExtractFailed")
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), os.ModePerm); err != nil {
			return errcode.New("errUpdateExtractFailed")
		}
		rc, err := f.Open()
		if err != nil {
			return errcode.New("errUpdateExtractFailed")
		}

		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return errcode.New("errUpdateExtractFailed")
		}
		if _, err := io.Copy(out, rc); err != nil {
			out.Close()
			rc.Close()
			return errcode.New("errUpdateExtractFailed")
		}
		if err := out.Close(); err != nil {
			rc.Close()
			return errcode.New("errUpdateExtractFailed")
		}
		if err := rc.Close(); err != nil {
			return errcode.New("errUpdateExtractFailed")
		}
	}
	return nil
}
