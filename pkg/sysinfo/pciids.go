package sysinfo

// To download the PCI ID database for local development:
//   go generate ./pkg/sysinfo/
//
//go:generate curl -sL -o data/pci.ids.gz https://pci-ids.ucw.cz/v2.2/pci.ids.gz

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// pciDB holds parsed PCI vendor and device name mappings.
type pciDB struct {
	vendors  map[string]string // "10DE" → "NVIDIA Corporation"
	products map[string]string // "10DE:2208" → "GeForce RTX 3080"
}

// getPCIDB returns the parsed PCI ID database, loaded once on first call.
// Returns an empty database if pci.ids.gz cannot be found or read.
var getPCIDB = sync.OnceValue(func() *pciDB {
	return loadPCIDB()
})

// Search order:
//  1. {exe_dir}/internals/data/pci.ids.gz — release
//  2. pkg/sysinfo/data/pci.ids.gz — dev (relative to working directory)
func loadPCIDB() *pciDB {
	empty := &pciDB{vendors: map[string]string{}, products: map[string]string{}}

	var paths []string
	if exe, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(exe), "internals", "data", "pci.ids.gz"))
	}
	paths = append(paths, filepath.Join("pkg", "sysinfo", "data", "pci.ids.gz"))

	var r io.ReadCloser
	for _, p := range paths {
		f, err := os.Open(p)
		if err == nil {
			r = f
			break
		}
	}
	if r == nil {
		fmt.Fprintln(os.Stderr, "install-it: pci.ids.gz not found — GPU/NIC names will be unavailable")
		return empty
	}
	defer r.Close()

	gz, err := gzip.NewReader(r)
	if err != nil {
		return empty
	}
	defer gz.Close()

	return parsePCIDB(gz)
}

// Format:
//
//	XXXX  Vendor Name          (no leading whitespace)
//	\tXXXX  Device Name        (single tab)
//	\t\tXXXX XXXX  Subsystem   (two tabs — ignored)
func parsePCIDB(r io.Reader) *pciDB {
	db := &pciDB{
		vendors:  make(map[string]string),
		products: make(map[string]string),
	}

	scanner := bufio.NewScanner(r)
	var currentVendor string

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || line[0] == '#' {
			continue
		}

		switch {
		case line[0] != '\t':
			if id, name, ok := strings.Cut(line, "  "); ok {
				currentVendor = strings.TrimSpace(id)
				db.vendors[currentVendor] = strings.TrimSpace(name)
			}
		case strings.HasPrefix(line, "\t\t"):
			// subsystem — skip
		default:
			if currentVendor == "" {
				continue
			}
			if devID, devName, ok := strings.Cut(strings.TrimLeft(line, "\t"), "  "); ok {
				db.products[currentVendor+":"+strings.TrimSpace(devID)] = strings.TrimSpace(devName)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "install-it: pci.ids parse error — using partial database:", err)
	}

	return db
}

// hwidRe matches VEN_XXXX and DEV_XXXX in a Windows HardwareID string.
var hwidRe = regexp.MustCompile(`(?i)VEN_([0-9A-F]{4}).*?DEV_([0-9A-F]{4})`)

// ResolvePciName converts a Windows HardwareID string to a human-readable
// PCI device name using the pci.ids database.
//
// Returns "VendorName DeviceName" (e.g. "NVIDIA Corporation GeForce RTX 3080").
// Returns empty string if the HardwareID cannot be parsed or the IDs are not found.
func ResolvePciName(hardwareID string) string {
	matches := hwidRe.FindStringSubmatch(hardwareID)
	if len(matches) != 3 {
		return ""
	}

	ven := strings.ToLower(matches[1])
	dev := strings.ToLower(matches[2])

	db := getPCIDB()
	venName, venOk := db.vendors[ven]
	devName, devOk := db.products[ven+":"+dev]

	if !venOk && !devOk {
		return ""
	}
	return strings.TrimSpace(venName + " " + devName)
}
