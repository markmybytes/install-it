package porter

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"install-it/pkg/errcode"
)

// dirSize calculates the total size of files in a directory and its subdirectories.
func dirSize(target string) (int64, error) {
	var size int64
	err := filepath.Walk(target, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return errcode.New("errExportSizeCalcFailed")
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

// toZip compresses target directories into install-it.zip at dest, writing
// manifest.json as the first entry. Progress is reported via the job.
func toZip(j *job, dest string, dirRoot string, targets []string) (err error) {
	j.setStep("compression")
	j.msg("Calculating total size...")

	var totalSize int64
	for _, dir := range targets {
		size, err := dirSize(dir)
		if err != nil {
			return errcode.Newf("errExportSizeCalcFailed", map[string]any{"dir": dir})
		}
		totalSize += size
	}

	zipPath := filepath.Join(dest, "install-it.zip")
	j.msg(fmt.Sprintf("Creating archive: %s", zipPath))

	file, err := os.Create(zipPath)
	if err != nil {
		return errcode.New("errExportZipCreateFailed")
	}
	defer func() {
		file.Close()

		if err != nil {
			os.Remove(zipPath)
		}
	}()

	zw := zip.NewWriter(file)
	defer zw.Close()

	if err := writeManifest(zw, newManifest()); err != nil {
		return err
	}

	var written int64
	for _, dir := range targets {
		err = filepath.Walk(dir, func(filePath string, info os.FileInfo, err error) error {
			if j.ctx.Err() != nil {
				return errcode.New("errImportCanceled")
			}
			if err != nil {
				return err
			}

			j.msg(fmt.Sprintf("Packing: %s", filePath))

			if info.IsDir() {
				return nil
			}

			// Compute relative path from dirRoot
			rel, err := filepath.Rel(dirRoot, filePath)
			if err != nil {
				return errcode.New("errExportRelPathFailed")
			}
			// Normalize to forward slashes for ZIP
			entryName := filepath.ToSlash(rel)

			srcFile, err := os.Open(filePath)
			if err != nil {
				return errcode.Newf("errExportFileOpen", map[string]any{"file": filePath})
			}

			zipEntry, err := zw.Create(entryName)
			if err != nil {
				srcFile.Close()
				return errcode.Newf("errExportEntryCreateFailed", map[string]any{"entry": entryName})
			}

			if _, err = io.Copy(zipEntry, srcFile); err != nil {
				srcFile.Close()
				return errcode.Newf("errExportEntryWriteFailed", map[string]any{"entry": entryName})
			}
			srcFile.Close()

			written += info.Size()
			if totalSize > 0 {
				j.setProgress(float64(written) / float64(totalSize))
			}
			return nil
		})

		if err != nil {
			return err
		}
	}

	j.msg(fmt.Sprintf("All files packed into: %s", zipPath))
	return nil
}

// fromZip extracts entries from a ZIP archive to dest, filtered by ImportOptions.
// It skips manifest.json and only extracts entries matching the opts selection.
func fromZip(j *job, orig string, dest string, opts ImportOptions) (err error) {
	j.setStep("extract")
	j.msg("Opening archive...")

	zr, err := zip.OpenReader(orig)
	if err != nil {
		return errcode.New("errImportFileOpen")
	}
	defer zr.Close()

	if err := os.MkdirAll(dest, os.ModePerm); err != nil {
		return errcode.New("errImportMkdirFailed")
	}

	// Count total extractable bytes for progress tracking
	type entryInfo struct {
		file *zip.File
		size int64
	}
	var entries []entryInfo
	var totalBytes int64

	for _, zf := range zr.File {
		name := filepath.ToSlash(zf.Name)
		if name == "manifest.json" {
			continue
		}

		shouldExtract := false
		switch {
		case opts.Settings && name == "conf/setting.json":
			shouldExtract = true
		case opts.Data && name == "conf/data.db":
			shouldExtract = true
		case opts.Data && strings.HasPrefix(name, "drivers/"):
			shouldExtract = true
		}

		if !shouldExtract {
			continue
		}

		entries = append(entries, entryInfo{file: zf, size: int64(zf.FileInfo().Size())})
		totalBytes += int64(zf.FileInfo().Size())
	}

	if totalBytes == 0 {
		return errcode.New("errImportNoMatchingEntries")
	}

	var extracted int64
	extractAndWriteFile := func(zf *zip.File) error {
		if j.ctx.Err() != nil {
			return errcode.New("errImportCanceled")
		}

		zfreader, err := zf.Open()
		if err != nil {
			return errcode.Newf("errImportEntryOpenFailed", map[string]any{"entry": zf.Name})
		}
		defer zfreader.Close()

		// Use forward-slash name consistently
		name := filepath.ToSlash(zf.Name)
		extractPath := filepath.Join(dest, name)

		// ZipSlip protection
		cleanDest := filepath.Clean(dest)
		if !strings.HasPrefix(filepath.Clean(extractPath), cleanDest+string(os.PathSeparator)) && filepath.Clean(extractPath) != cleanDest {
			return errcode.Newf("errImportIllegalPath", map[string]any{"path": extractPath})
		}

		j.msg(fmt.Sprintf("Extracting: %s", name))

		if zf.FileInfo().IsDir() {
			return os.MkdirAll(extractPath, zf.Mode())
		}

		if err := os.MkdirAll(filepath.Dir(extractPath), os.ModePerm); err != nil {
			return errcode.Newf("errImportMkdirFailed", map[string]any{"dir": filepath.Dir(extractPath)})
		}

		outFile, err := os.OpenFile(extractPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, zf.Mode())
		if err != nil {
			return errcode.Newf("errImportFileCreateFailed", map[string]any{"file": extractPath})
		}
		defer outFile.Close()

		_, err = io.Copy(outFile, zfreader)
		if err != nil {
			return errcode.Newf("errImportFileWriteFailed", map[string]any{"file": extractPath})
		}

		extracted += zf.FileInfo().Size()
		if totalBytes > 0 {
			j.setProgress(float64(extracted) / float64(totalBytes))
		}
		return nil
	}

	for _, entry := range entries {
		if err := extractAndWriteFile(entry.file); err != nil {
			return err
		}
	}

	j.msg("Extraction complete")
	return nil
}

// backup moves the specified files and directories into a single .porter-{timestamp}/ folder.
// The timestamp is returned for cleanup/rollback. If any move fails, already-moved items are restored.
func backup(j *job, dirRoot string, files []string, dirs []string) (timestamp string, err error) {
	j.setStep("backup")
	j.msg("Creating backups...")

	timestamp = time.Now().Format("20060102T150405")
	backupDir := filepath.Join(dirRoot, ".porter-"+timestamp)

	var moved []string
	defer func() {
		if err != nil && len(moved) > 0 {
			var rollbackErr error
			for _, item := range moved {
				if rErr := os.Rename(filepath.Join(backupDir, item), filepath.Join(dirRoot, item)); rErr != nil {
					rollbackErr = rErr
					break
				}
			}
			if rollbackErr != nil {
				// Leave backup dir in place for manual recovery
				err = errcode.Newf("errImportBackupRollbackFailed", map[string]any{"error": rollbackErr, "backupDir": backupDir})
				return
			}
			os.RemoveAll(backupDir)
		}
	}()

	for _, item := range slices.Concat(files, dirs) {
		original := filepath.Join(dirRoot, item)
		if _, statErr := os.Stat(original); os.IsNotExist(statErr) {
			continue
		}
		backupPath := filepath.Join(backupDir, item)
		if err := os.MkdirAll(filepath.Dir(backupPath), os.ModePerm); err != nil {
			return timestamp, errcode.New("errImportBackupMkdirFailed")
		}
		j.msg(fmt.Sprintf("Backing up: %s", original))
		if err := os.Rename(original, backupPath); err != nil {
			return timestamp, errcode.Newf("errImportBackupFailed", map[string]any{"from": original, "to": backupPath})
		}
		moved = append(moved, item)
	}

	if len(moved) == 0 {
		j.msg("No existing files to backup")
	} else {
		j.msg(fmt.Sprintf("Backup complete (timestamp: %s)", timestamp))
	}
	return timestamp, nil
}

func cleanupBackups(j *job, dirRoot string, timestamp string) error {
	j.setStep("cleanup")
	j.msg("Cleaning up backups...")
	return os.RemoveAll(filepath.Join(dirRoot, ".porter-"+timestamp))
}

// rollback restores backed-up files and directories, then removes the backup folder.
// Returns a summary of all errors encountered. If any step fails, the backup folder is preserved.
func rollback(j *job, dirRoot string, timestamp string, files []string, dirs []string) error {
	j.msg("Rolling back...")

	backupDir := filepath.Join(dirRoot, ".porter-"+timestamp)

	var errs []error

	for _, d := range dirs {
		original := filepath.Join(dirRoot, d)
		if _, statErr := os.Stat(original); statErr == nil {
			if err := os.RemoveAll(original); err != nil {
				errs = append(errs, errcode.Newf("errImportRollbackRemoveFailed", map[string]any{"path": original}))
			}
		}
		backupPath := filepath.Join(backupDir, d)
		if _, statErr := os.Stat(backupPath); statErr == nil {
			if err := os.Rename(backupPath, original); err != nil {
				errs = append(errs, errcode.Newf("errImportRollbackRenameFailed", map[string]any{"from": backupPath, "to": original}))
			}
		}
	}

	for _, f := range files {
		original := filepath.Join(dirRoot, f)
		if _, statErr := os.Stat(original); statErr == nil {
			if err := os.Remove(original); err != nil {
				errs = append(errs, errcode.Newf("errImportRollbackRemoveFailed", map[string]any{"path": original}))
			}
		}
		backupPath := filepath.Join(backupDir, f)
		if _, statErr := os.Stat(backupPath); statErr == nil {
			if err := os.MkdirAll(filepath.Dir(original), os.ModePerm); err != nil {
				errs = append(errs, errcode.Newf("errImportRollbackMkdirFailed", map[string]any{"path": original}))
			}
			if err := os.Rename(backupPath, original); err != nil {
				errs = append(errs, errcode.Newf("errImportRollbackRenameFailed", map[string]any{"from": backupPath, "to": original}))
			}
		}
	}

	if len(errs) > 0 {
		return errcode.New("errImportRollbackFailed")
	}

	if err := os.RemoveAll(backupDir); err != nil {
		return errcode.New("errImportRollbackFailed")
	}
	return nil
}

// download fetches a ZIP from a URL to a temp file, reporting progress via the job.
func download(j *job, url string) (path string, err error) {
	j.setStep("download")
	j.msg(fmt.Sprintf("Downloading: %s", url))

	req, err := http.NewRequestWithContext(j.ctx, "GET", url, nil)
	if err != nil {
		return "", errcode.New("errImportDownloadRequestFailed")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", errcode.New("errImportDownloadFailed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errcode.Newf("errImportDownloadStatus", map[string]any{"status": resp.StatusCode})
	}

	tmpFile, err := os.CreateTemp("", "*.zip")
	if err != nil {
		return "", errcode.New("errImportTempFileFailed")
	}

	var total int64
	if resp.ContentLength > 0 {
		total = resp.ContentLength
	}

	j.msg("Downloading...")

	written, err := io.Copy(tmpFile, &downloadReader{
		reader: resp.Body,
		job:    j,
		total:  total,
	})
	if err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", errcode.New("errImportDownloadInterrupted")
	}
	tmpFile.Close()

	if total > 0 {
		j.setProgress(1.0)
	}

	absPath, err := filepath.Abs(tmpFile.Name())
	if err != nil {
		return "", errcode.New("errImportTempPathFailed")
	}

	j.msg(fmt.Sprintf("Downloaded %d bytes to %s", written, absPath))
	return absPath, nil
}

// downloadReader wraps an io.Reader and updates job progress on each Read.
type downloadReader struct {
	reader io.Reader
	job    *job
	read   int64
	total  int64
}

func (r *downloadReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.read += int64(n)
	if r.total > 0 {
		r.job.setProgress(float64(r.read) / float64(r.total))
	}
	return n, err
}
