package porter

import (
	"archive/zip"
	"errors"
	"fmt"
	"install-it/pkg/errcode"
	"install-it/pkg/status"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// errImportCanceled is the sentinel cancel error; local read failures are
// NEVER this value — dataDiff swallows the latter, propagates only the former.
var errImportCanceled = errcode.New("errImportCanceled")

// TargetDiff describes how one import category compares to the local data.
// Available is false whenever the comparison can't be rendered (legacy manifest,
// missing incoming hash, or a local read failure) — the frontend hides the row.
type TargetDiff struct {
	Available       bool  `json:"available"`
	IncomingPresent bool  `json:"incomingPresent"`
	LocalPresent    bool  `json:"localPresent"`
	Identical       bool  `json:"identical"`
	IncomingCount   int64 `json:"incomingCount"`
	IncomingSize    int64 `json:"incomingSize"`
	LocalCount      int64 `json:"localCount"`
	LocalSize       int64 `json:"localSize"`
}

// ImportPreview describes what's in a ZIP, returned by ValidateZip and DownloadAndValidate.
type ImportPreview struct {
	// RFC3339 string, not time.Time — Wails binding generator has no model
	// for time.Time. Manifest keeps time.Time (internal, never bound).
	ExportedAt  string `json:"exportedAt"`
	HasSettings bool   `json:"hasSettings"`
	HasData     bool   `json:"hasData"`
	HasDatabase bool   `json:"hasDatabase"`
	HasDrivers  bool   `json:"hasDrivers"`
	DriverCount int    `json:"driverCount"`
	DriverSize  int64  `json:"driverSize"`

	Settings TargetDiff `json:"settings"`
	Data     TargetDiff `json:"data"`
}

// ImportOptions is the user's category selection for import.
type ImportOptions struct {
	Settings bool `json:"settings"`
	Data     bool `json:"data"`
}

// Porter handles export/import of program data. Only one job runs at a time —
// calling any job-starting method while a job is running will be rejected.
type Porter struct {
	DirRoot string   // Root directory for import/export operations
	Targets []string // Target directories to be backed up or compressed

	OnBeforeBackup func() error // Called before backup to close DB
	OnAfterImport  func() error // Called after import to reopen DB

	job      *job   // Current job, nil when idle
	tempPath string // Pending download path
}

func (p *Porter) Status() status.Status {
	if p.job == nil {
		return status.Pending
	}
	p.job.mu.Lock()
	defer p.job.mu.Unlock()
	return p.job.status
}

func (p *Porter) Abort() error {
	if p.job == nil {
		return errcode.New("errImportNoRunningJob")
	}
	p.job.mu.Lock()
	defer p.job.mu.Unlock()
	if p.job.status != status.Running {
		return errcode.New("errImportNoRunningJob")
	}
	p.job.cancel()
	return nil
}

func (p *Porter) Progress() (JobSnapshot, error) {
	if p.job == nil {
		return JobSnapshot{}, errcode.New("errImportNoStartedJob")
	}
	return p.job.snapshot(), nil
}

func (p *Porter) Export(dest string) (err error) {
	if p.job != nil && p.job.status == status.Running {
		return errcode.New("errImportAlreadyRunning")
	}

	p.job = newJob()
	p.job.start()
	defer func() {
		if err != nil {
			p.job.fail(err)
		} else {
			p.job.complete()
		}
	}()

	return toZip(p.job, dest, p.DirRoot, p.Targets)
}

// zipScan is the result of scanning a ZIP + manifest. It never touches the live
// filesystem, so it is cheap enough to reuse on the import path.
type zipScan struct {
	manifest     Manifest
	hasSettings  bool
	hasDatabase  bool
	hasDrivers   bool
	driverCount  int64
	driverSize   int64
	settingsSize int64
	databaseSize int64
}

func (s zipScan) hasData() bool { return s.hasDatabase || s.hasDrivers }

func (s zipScan) preview() ImportPreview {
	return ImportPreview{
		ExportedAt:  s.manifest.ExportedAt.Format(time.RFC3339),
		HasSettings: s.hasSettings,
		HasData:     s.hasData(),
		HasDatabase: s.hasDatabase,
		HasDrivers:  s.hasDrivers,
		DriverCount: int(s.driverCount),
		DriverSize:  s.driverSize,
	}
}

func boolCount(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// scanZip opens path, reads the manifest and counts recognized entries. It is
// the shared validation core: no live-FS hashing, so imports don't pay for it.
func scanZip(path string) (zipScan, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return zipScan{}, errcode.New("errImportFileOpen")
	}
	defer zr.Close()

	m, err := readManifest(zr)
	if err != nil {
		return zipScan{}, err
	}

	if m.FormatVersion != CurrentFormatVersion {
		return zipScan{}, errcode.Newf("errImportVersionUnsupported", map[string]any{"version": m.FormatVersion, "expected": CurrentFormatVersion})
	}

	s := zipScan{manifest: m}
	for _, f := range zr.File {
		name := filepath.ToSlash(f.Name)
		size := int64(f.FileInfo().Size())

		switch {
		case name == "conf/setting.json":
			s.hasSettings = true
			s.settingsSize = size
		case name == "conf/data.db":
			s.hasDatabase = true
			s.databaseSize = size
		case strings.HasPrefix(name, "drivers/"):
			s.hasDrivers = true
			if !f.FileInfo().IsDir() {
				s.driverCount++
				s.driverSize += size
			}
		}
	}

	if !s.hasSettings && !s.hasData() {
		return zipScan{}, errcode.New("errImportNoData")
	}

	return s, nil
}

// ValidateZip reads a ZIP's manifest and computes a preview-diff against DirRoot:
// per-category digests (sha1), counts and sizes. Runs as a job so the frontend
// can poll progress (local hashing can take minutes on large driver sets).
func (p *Porter) ValidateZip(path string) (preview ImportPreview, err error) {
	if p.job != nil && p.job.status == status.Running {
		return ImportPreview{}, errcode.New("errImportAlreadyRunning")
	}
	p.job = newJob()
	p.job.start()
	defer func() {
		if err != nil {
			p.job.fail(err)
		} else {
			p.job.complete()
		}
	}()

	s, err := scanZip(path)
	if err != nil {
		return ImportPreview{}, err
	}

	return p.buildPreview(s)
}

// buildPreview derives the settings/data diff rows from a scan. With a running
// job it sets the compare step (progress is driven by localDataHashes' tick).
// Local-read failures are swallowed (row hidden); only cancellation errors out.
func (p *Porter) buildPreview(s zipScan) (ImportPreview, error) {
	if p.job != nil {
		// Live-FS hashing runs synchronously; surface progress so the UI isn't frozen.
		p.job.setStep("compare")
		p.job.msg("Comparing with local data...")
	}
	preview := s.preview()
	preview.Settings = p.settingsDiff(s)
	data, err := p.dataDiff(s)
	if err != nil {
		return ImportPreview{}, err
	}
	preview.Data = data
	return preview, nil
}

func (p *Porter) settingsDiff(s zipScan) TargetDiff {
	d := TargetDiff{
		IncomingPresent: s.hasSettings,
		IncomingCount:   boolCount(s.hasSettings),
		IncomingSize:    s.settingsSize,
	}

	// Legacy / missing incoming hash, or hash disagrees with the entry scan.
	if !s.hasSettings || s.manifest.Hashes.Settings == "" {
		return d
	}

	sum, size, present, err := localFileHashFn(filepath.Join(p.DirRoot, "conf", "setting.json"))
	if err != nil {
		return d // local read failed: not renderable
	}

	d.LocalPresent = present
	if present {
		d.LocalCount = 1
		d.LocalSize = size
		d.Identical = sum == s.manifest.Hashes.Settings
	}
	d.Available = true
	return d
}

// dataDiffTick, if p.job is set, reports byte progress and honors Abort by
// returning the cancel error — the hashing walk stops at the next file.
// Returns nil with no job (e.g. direct test calls).
func (p *Porter) dataDiffTick() func(done, total int64) error {
	if p.job == nil {
		return nil
	}
	return func(done, total int64) error {
		p.job.mu.Lock()
		canceled := p.job.ctx.Err() != nil
		p.job.mu.Unlock()
		if canceled {
			return errImportCanceled
		}
		if total > 0 {
			p.job.setProgress(float64(done) / float64(total))
		}
		return nil
	}
}

func (p *Porter) dataDiff(s zipScan) (TargetDiff, error) {
	d := TargetDiff{
		IncomingPresent: s.hasData(),
		IncomingCount:   boolCount(s.hasDatabase) + s.driverCount,
		IncomingSize:    s.databaseSize + s.driverSize,
	}

	if !d.IncomingPresent || s.manifest.Hashes.Data == "" {
		return d, nil
	}

	hashes, size, err := localDataHashes(p.DirRoot, p.dataDiffTick())
	if err != nil {
		// Read failures hide the row (same as settingsDiff); only the cancel
		// sentinel propagates so the clean path surfaces Aborted, no preview.
		if !errors.Is(err, errImportCanceled) {
			return d, nil
		}
		return d, err
	}

	d.LocalPresent = len(hashes) > 0
	d.LocalCount = int64(len(hashes))
	d.LocalSize = size
	if d.LocalPresent {
		d.Identical = foldDataHash(hashes) == s.manifest.Hashes.Data
	}
	d.Available = true
	return d, nil
}

// DownloadAndValidate fetches a ZIP from url, validates it,
// and stores the path internally for ImportFromURL to consume.
//
// Call this first, then ImportFromURL.
// Calling DownloadAndValidate again before ImportFromURL replaces the stored path
// (previous temp file is removed).
func (p *Porter) DownloadAndValidate(url string) (preview ImportPreview, err error) {
	if p.job != nil && p.job.status == status.Running {
		return ImportPreview{}, errcode.New("errImportAlreadyRunning")
	}
	p.job = newJob()
	p.job.start()
	defer func() {
		if err != nil {
			// Any failure after download cancels the pending import: drop the temp
			// ZIP so DownloadAndValidate retries stay clean.
			if p.tempPath != "" {
				os.Remove(p.tempPath)
			}
			p.tempPath = ""
			p.job.fail(err)
		} else {
			p.job.complete()
		}
	}()

	path, err := download(p.job, url)
	if err != nil {
		return ImportPreview{}, errcode.New("errImportDownloadFailed")
	}

	p.tempPath = path

	s, err := scanZip(path)
	if err != nil {
		return ImportPreview{}, err
	}

	return p.buildPreview(s)
}

// ImportFromFile extracts selected categories from a local ZIP file into DirRoot.
// Backs up existing files first; rolls back on extraction failure.
func (p *Porter) ImportFromFile(path string, opts ImportOptions) (err error) {
	if p.job != nil && p.job.status == status.Running {
		return errcode.New("errImportAlreadyRunning")
	}
	p.job = newJob()
	p.job.start()

	var (
		dbClosed  bool
		timestamp string
	)

	defer func() {
		if dbClosed && p.OnAfterImport != nil {
			if reopenErr := p.OnAfterImport(); reopenErr != nil {
				if err == nil {
					err = errcode.New("errImportDbReopenFailed")
				}
			}
		}
	}()

	scan, err := scanZip(path)
	if err != nil {
		p.job.fail(err)
		return err
	}
	preview := scan.preview()

	// Compute backup set: files and dirs that are in ZIP ∩ selected by opts ∩ exist on disk
	var backupFiles, backupDirs []string

	if opts.Settings {
		if preview.HasSettings {
			if _, statErr := os.Stat(filepath.Join(p.DirRoot, "conf", "setting.json")); statErr == nil {
				backupFiles = append(backupFiles, filepath.Join("conf", "setting.json"))
			}
		}
	}

	if opts.Data {
		if !preview.HasData {
			err := errcode.New("errImportNoCategories")
			p.job.fail(err)
			return err
		}
		if preview.HasDatabase {
			if _, statErr := os.Stat(filepath.Join(p.DirRoot, "conf", "data.db")); statErr == nil {
				backupFiles = append(backupFiles, filepath.Join("conf", "data.db"))
			}
		}
		if preview.HasDrivers {
			if _, statErr := os.Stat(filepath.Join(p.DirRoot, "drivers")); statErr == nil {
				backupDirs = append(backupDirs, "drivers")
			}
		}
	} else {
		if !opts.Settings || !preview.HasSettings {
			err := errcode.New("errNoCategoriesSelected")
			p.job.fail(err)
			return err
		}
	}

	if len(backupFiles) == 0 && len(backupDirs) == 0 {
		err := errcode.New("errImportNothingToImport")
		p.job.fail(err)
		return err
	}

	for _, f := range backupFiles {
		if strings.HasSuffix(f, "conf/data.db") || strings.HasSuffix(f, "conf\\data.db") {
			dbClosed = true
			break
		}
	}

	if dbClosed && p.OnBeforeBackup != nil {
		p.job.msg("Closing database for backup...")
		if err := p.OnBeforeBackup(); err != nil {
			p.job.fail(err)
			return errcode.New("errImportDbCloseFailed")
		}
	}

	p.job.msg("Backing up existing files...")
	timestamp, err = backup(p.job, p.DirRoot, backupFiles, backupDirs)
	if err != nil {
		p.job.fail(err)
		return err
	}

	p.job.msg("Extracting archive...")
	err = fromZip(p.job, path, p.DirRoot, opts)
	if err != nil {
		p.job.msg("Extraction failed, rolling back...")
		rollbackErr := rollback(p.job, p.DirRoot, timestamp, backupFiles, backupDirs)
		p.job.fail(err)
		if rollbackErr != nil {
			return errcode.Newf("errImportRollbackFailed", map[string]any{"error": rollbackErr})
		}
		return err
	}

	p.job.msg("Cleaning up backups...")
	if err := cleanupBackups(p.job, p.DirRoot, timestamp); err != nil {
		p.job.msg(fmt.Sprintf("Warning: cleanup issue: %v", err))
	}

	p.job.complete()
	return nil
}

// ImportFromURL imports the ZIP previously downloaded by DownloadAndValidate.
// Calls ImportFromFile with the stored path. The stored path is cleared after
// import (success or failure).
func (p *Porter) ImportFromURL(opts ImportOptions) error {
	if p.tempPath == "" {
		return errcode.New("errImportNoDownloadedFile")
	}
	path := p.tempPath

	defer func() {
		os.Remove(path)
		p.tempPath = ""
	}()

	return p.ImportFromFile(path, opts)
}

// RecoverOrphanedBackups restores files from .porter-* backup directories left
// behind by interrupted imports. Safe to call at startup.
func (p *Porter) RecoverOrphanedBackups() error {
	matches, _ := filepath.Glob(filepath.Join(p.DirRoot, ".porter-*"))
	for _, backupDir := range matches {
		walkErr := filepath.Walk(backupDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || path == backupDir || info.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(backupDir, path)
			if err != nil {
				return errcode.New("errImportRecoverFailed")
			}
			original := filepath.Join(p.DirRoot, rel)
			if _, statErr := os.Stat(original); statErr == nil {
				os.Remove(original)
			}
			os.MkdirAll(filepath.Dir(original), os.ModePerm)
			return os.Rename(path, original)
		})

		if walkErr != nil {
			// Leave backup dir for manual recovery or next startup retry
			continue
		}
		os.RemoveAll(backupDir)
	}
	return nil
}
