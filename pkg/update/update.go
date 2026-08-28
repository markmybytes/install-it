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
	"time"

	"github.com/Masterminds/semver"
	"install-it/pkg/errcode"
)

type Updater struct {
	DirRoot string
	Version *semver.Version
	// apiBase overrides the GitHub API root; set in tests via httptest.
	apiBase string
	// client overrides the HTTP client; set in tests.
	client *http.Client
}

func (u *Updater) httpGet(url string) (*http.Response, error) {
	if u.client != nil {
		return u.client.Get(url)
	}
	return http.Get(url)
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
	HasUpdate          bool   `json:"hasUpdate"`
	LatestVersion      string `json:"latestVersion"`
	DownloadUrl        string `json:"downloadUrl"`
	DownloadUrlBundled string `json:"downloadUrlBundled"`
	ReleaseNotes       string `json:"releaseNotes"`
	ReleaseAt          string `json:"releaseAt"`
}

func (u *Updater) CheckForUpdates(preferBundled, preferPreRelease bool) (*UpdateCheckResult, error) {
	type releasePayload struct {
		TagName     string `json:"tag_name"`
		Body        string `json:"body"`
		PublishedAt string `json:"published_at"`
		Assets      []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}

	var body releasePayload
	if preferPreRelease {
		resp, err := u.httpGet(u.releasesURL(false))
		if err != nil {
			return nil, errcode.New("errUpdateCheckFailed")
		}
		defer resp.Body.Close()

		var releases []releasePayload
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

		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			return nil, errcode.New("errUpdateCheckFailed")
		}
	}

	latestTag := strings.TrimPrefix(body.TagName, "v")
	latestVer, _ := semver.NewVersion(latestTag)

	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x64"
	} else if arch == "386" {
		arch = "x86"
	}
	binPrefix := fmt.Sprintf("install-it.%s-%s", runtime.GOOS, arch)

	var url, urlBundled string
	for _, a := range body.Assets {
		if a.Name == binPrefix+".zip" {
			url = a.BrowserDownloadURL
		} else if a.Name == binPrefix+"-bundled.zip" {
			urlBundled = a.BrowserDownloadURL
		}
	}

	primaryUrl := url
	if preferBundled && urlBundled != "" {
		primaryUrl = urlBundled
	}

	return &UpdateCheckResult{
		HasUpdate:          latestVer != nil && latestVer.GreaterThan(u.Version),
		LatestVersion:      latestTag,
		DownloadUrl:        primaryUrl,
		DownloadUrlBundled: urlBundled,
		ReleaseNotes:       body.Body,
		ReleaseAt:          body.PublishedAt,
	}, nil
}

func (u *Updater) TriggerNativeUpdate(downloadUrl string) error {
	resp, err := http.Get(downloadUrl)
	if err != nil {
		return errcode.New("errUpdateDownloadFailed")
	}
	defer resp.Body.Close()

	tmpZip, err := os.CreateTemp(u.DirRoot, "update-*.zip")
	if err != nil {
		return errcode.New("errUpdateWriteFailed")
	}
	defer os.Remove(tmpZip.Name())

	if _, err := io.Copy(tmpZip, resp.Body); err != nil {
		tmpZip.Close()
		return errcode.New("errUpdateWriteFailed")
	}
	tmpZip.Close()

	stageDir := filepath.Join(u.DirRoot, ".update_stage")
	if err := extractZipToDir(tmpZip.Name(), stageDir); err != nil {
		os.RemoveAll(stageDir)
		return err
	}

	exe := filepath.Join(u.DirRoot, "install-it.exe")
	old := filepath.Join(u.DirRoot, "install-it.exe.old")
	newExe := filepath.Join(stageDir, "install-it.exe")

	if err := os.Rename(exe, old); err != nil {
		os.RemoveAll(stageDir)
		return errcode.New("errUpdateRenameFailed")
	}
	if err := os.Rename(newExe, exe); err != nil {
		os.Rename(old, exe) // rollback
		os.RemoveAll(stageDir)
		return errcode.New("errUpdateRenameFailed")
	}

	cmd := exec.Command(exe)
	cmd.Dir = u.DirRoot
	cmd.SysProcAttr = detachedProc()
	if err := cmd.Start(); err != nil {
		return errcode.New("errUpdateSpawnFailed")
	}

	os.Remove(tmpZip.Name())
	os.Exit(0)
	return nil
}

func (u *Updater) CheckAndApplyUpdates() {
	oldExe := filepath.Join(u.DirRoot, "install-it.exe.old")

	// Wait up to 10s for the ghost WebView2 processes to die
	if _, err := os.Stat(oldExe); err == nil {
		deleted := false
		for i := 0; i < 100; i++ {
			if os.Remove(oldExe) == nil {
				deleted = true
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if !deleted {
			return // Process locked. Abort file deployment to protect integrity.
		}
	}

	stageDir := filepath.Join(u.DirRoot, ".update_stage")
	entries, err := os.ReadDir(stageDir)
	if err != nil {
		return
	}

	for _, e := range entries {
		name := e.Name()
		if strings.EqualFold(name, "install-it.exe") {
			continue
		}

		dest := filepath.Join(u.DirRoot, name)

		if e.IsDir() {
			if !strings.EqualFold(name, "internals") {
				continue
			}
			os.RemoveAll(dest)
		} else {
			os.Remove(dest)
		}
		os.Rename(filepath.Join(stageDir, name), dest)
	}

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
			os.MkdirAll(target, f.Mode())
			continue
		}

		os.MkdirAll(filepath.Dir(target), os.ModePerm)
		rc, err := f.Open()
		if err != nil {
			return errcode.New("errUpdateExtractFailed")
		}

		if out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode()); err == nil {
			io.Copy(out, rc)
			out.Close()
		}
		rc.Close()
	}
	return nil
}
