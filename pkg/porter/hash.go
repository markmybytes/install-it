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
func localDataHashes(dirRoot string) (map[string]string, int64, error) {
	hashes := make(map[string]string)
	var size int64

	sum, n, present, err := localFileHashFn(filepath.Join(dirRoot, "conf", "data.db"))
	if err != nil {
		return nil, 0, err
	}
	if present {
		hashes["conf/data.db"] = sum
		size += n
	}

	driversRoot := filepath.Join(dirRoot, "drivers")
	var walkErr error
	err = filepath.Walk(driversRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
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
		sum, n, present, hashErr := localFileHashFn(path)
		if hashErr != nil {
			walkErr = hashErr
			return nil
		}
		// File vanished mid-walk: skip the frame rather than recording an
		// empty digest, so the fold honestly differs (reported as "changed").
		if !present {
			return nil
		}
		hashes[filepath.ToSlash(rel)] = sum
		size += n
		return nil
	})
	if err != nil {
		return nil, 0, err
	}
	if walkErr != nil {
		return nil, 0, walkErr
	}
	return hashes, size, nil
}
