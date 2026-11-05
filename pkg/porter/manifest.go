package porter

import (
	"archive/zip"
	"encoding/json"
	"time"

	"install-it/pkg/errcode"
)

type Manifest struct {
	FormatVersion int       `json:"format_version"`
	ExportedAt    time.Time `json:"exported_at"`
	// Hashes is always emitted: encoding/json ignores omitempty on struct values.
	// Individual keys self-omit via their own omitempty when empty.
	Hashes Hashes `json:"hashes,omitempty"`
}

// Hashes carries content digests (sha1, lowercase hex) for preview-diff.
type Hashes struct {
	Settings string `json:"settings,omitempty"` // sha1 of conf/setting.json entry bytes
	Data     string `json:"data,omitempty"`     // folded digest of conf/data.db + drivers/**
}

const CurrentFormatVersion = 1

func newManifest() Manifest {
	return Manifest{
		FormatVersion: 1,
		ExportedAt:    time.Now(),
	}
}

func readManifest(zr *zip.ReadCloser) (Manifest, error) {
	for _, f := range zr.File {
		if f.Name == "manifest.json" {
			rc, err := f.Open()
			if err != nil {
				return Manifest{}, errcode.New("errImportManifestOpen")
			}
			defer rc.Close()

			var m Manifest
			if err := json.NewDecoder(rc).Decode(&m); err != nil {
				return Manifest{}, errcode.New("errImportManifestInvalid")
			}
			return m, nil
		}
	}
	return Manifest{}, errcode.New("errImportManifestMissing")
}

func writeManifest(zw *zip.Writer, m Manifest) error {
	entry, err := zw.Create("manifest.json")
	if err != nil {
		return errcode.New("errExportManifestCreate")
	}
	if err := json.NewEncoder(entry).Encode(m); err != nil {
		return errcode.New("errExportManifestEncode")
	}
	return nil
}
