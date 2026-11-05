package porter

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// dataEntry reports whether arcname participates in the folded Data digest.
func dataEntry(arcname string) bool {
	return arcname == "conf/data.db" || strings.HasPrefix(arcname, "drivers/")
}

// localDataItem is one file candidate for the Data fold, discovered cheaply
// (stat only) before any hashing starts.
type localDataItem struct {
	arcname string
	path    string
	size    int64
}

// foldDataHash builds the Data digest from a map of arcname -> per-file sha1 hex.
// Dedupe is inherent (map key); callers that can see duplicate arcnames must
// keep the LAST occurrence. Frames are sorted as full strings, byte-wise.
//
// With no Data entries the fold is empty: it returns "" (not sha1 of the empty
// input), so the manifest's `data,omitempty` key stays absent. Mirrors it-claws
// tests/test_hashes.py behavior for a settings-only export.
func foldDataHash(fileHashes map[string]string) string {
	frames := make([]string, 0, len(fileHashes))
	for name, sum := range fileHashes {
		if !dataEntry(name) {
			continue
		}
		// length prefix is the byte length of the UTF-8 encoded arcname.
		frames = append(frames, fmt.Sprintf("%d:%s:%s\n", len(name), name, sum))
	}
	if len(frames) == 0 {
		return ""
	}
	sort.Strings(frames)

	h := sha1.New()
	for _, f := range frames {
		io.WriteString(h, f)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// localFileHash hashes a local file. Missing file => present=false, err=nil.
// Any other failure (permission, I/O) => err non-nil.
func localFileHash(path string) (sum string, size int64, present bool, err error) {
	info, statErr := os.Stat(path)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			return "", 0, false, nil
		}
		return "", 0, false, statErr
	}
	if info.IsDir() {
		return "", 0, false, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return "", 0, false, err
	}
	defer f.Close()

	h := sha1.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, false, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, true, nil
}

// localFileHashFn is a test seam so read failures can be simulated portably.
var localFileHashFn = localFileHash

// localDataHashes walks DirRoot/conf/data.db and DirRoot/drivers/** (files only)
// and returns arcname -> sha1 hex plus total size. Missing entries are skipped;
// any read error aborts with a non-nil error.
// tick, if non-nil, fires after every file with (done, total) bytes hashed; its
// non-nil error aborts and is returned — use it for progress and cancellation.
func localDataHashes(dirRoot string, tick func(done, total int64) error) (map[string]string, int64, error) {
	// Phase 1: collect files (stat only, no reads) so Phase 2 can report
	// byte progress without needing a second hashing pass.
	var items []localDataItem

	if it, ok, err := statLocalDataItem(dirRoot, "conf/data.db"); err != nil {
		return nil, 0, err
	} else if ok {
		items = append(items, it)
	}

	driversRoot := filepath.Join(dirRoot, "drivers")
	var walkErr error
	err := filepath.Walk(driversRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil // missing drivers dir: no driver entries
			}
			walkErr = err
			return nil
		}
		if info.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(dirRoot, path)
		if relErr != nil {
			walkErr = relErr
			return nil
		}
		items = append(items, localDataItem{
			arcname: filepath.ToSlash(rel),
			path:    path,
			size:    info.Size(),
		})
		return nil
	})
	if err != nil {
		return nil, 0, err
	}
	if walkErr != nil {
		return nil, 0, walkErr
	}

	// Phase 2: hash the list, reporting byte progress per file.
	hashes := make(map[string]string, len(items))
	var total int64
	for _, it := range items {
		total += it.size
	}

	var hashed int64
	for _, it := range items {
		sum, n, present, err := localFileHashFn(it.path)
		if err != nil {
			return nil, 0, err
		}
		// File vanished mid-walk: skip the frame rather than recording an
		// empty digest, so the fold honestly differs (reported as "changed").
		if !present {
			continue
		}
		hashes[it.arcname] = sum
		hashed += n
		if tick != nil {
			if err := tick(hashed, total); err != nil {
				return nil, 0, err
			}
		}
	}
	return hashes, hashed, nil
}

// statLocalDataItem stats one archive-relative item under dirRoot. ok=false when
// the file is absent (or a directory); err non-nil only on real stat failures.
func statLocalDataItem(dirRoot, arcname string) (localDataItem, bool, error) {
	path := filepath.Join(dirRoot, filepath.FromSlash(arcname))
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return localDataItem{}, false, nil
		}
		return localDataItem{}, false, err
	}
	if info.IsDir() {
		return localDataItem{}, false, nil
	}
	return localDataItem{arcname: arcname, path: path, size: info.Size()}, true, nil
}
