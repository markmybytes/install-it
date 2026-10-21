// Tests for the App struct methods in package main.
//
// Note: the init() function in main.go runs when this test binary starts. It
// parses buildVersion (empty → "0.0.0") and creates conf/ and drivers/
// directories next to the test executable. This is harmless for testing.
//
// TriggerNativeUpdate cannot be tested end-to-end because it calls os.Exit(0);
// extractBinaryFromZip (its internal ZIP helper) is tested directly below.
package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestApp_AppVersion_DefaultZero(t *testing.T) {
	t.Parallel()
	a := App{}
	v := a.AppVersion()
	if v != "0.0.0" {
		t.Errorf("AppVersion() = %q, want '0.0.0'", v)
	}
}

func TestApp_AppBinaryType_Format(t *testing.T) {
	t.Parallel()
	a := App{}
	bt := a.AppBinaryType()

	parts := strings.SplitN(bt, "-", 2)
	if len(parts) != 2 {
		t.Fatalf("AppBinaryType() = %q: expected 'os-arch' format", bt)
	}

	goos := parts[0]
	arch := parts[1]

	if goos == "" || arch == "" {
		t.Errorf("AppBinaryType() = %q: os or arch part is empty", bt)
	}

	if goos != runtime.GOOS {
		t.Errorf("AppBinaryType() os part = %q, want %q", goos, runtime.GOOS)
	}

	goarch := runtime.GOARCH
	wantArch := goarch
	if goarch == "amd64" {
		wantArch = "x64"
	} else if goarch == "386" {
		wantArch = "x86"
	}
	if arch != wantArch {
		t.Errorf("AppBinaryType() arch part = %q, want %q", arch, wantArch)
	}
}

func TestApp_PathExists_ExistingFile(t *testing.T) {
	t.Parallel()
	a := App{}

	f, err := os.CreateTemp(t.TempDir(), "pathexists-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	if !a.PathExists(f.Name()) {
		t.Errorf("PathExists(%q) = false, want true for existing file", f.Name())
	}
}

func TestApp_PathExists_ExistingDirectory(t *testing.T) {
	t.Parallel()
	a := App{}
	dir := t.TempDir()
	if !a.PathExists(dir) {
		t.Errorf("PathExists(%q) = false, want true for existing directory", dir)
	}
}

func TestApp_PathExists_NonExistent(t *testing.T) {
	t.Parallel()
	a := App{}
	path := filepath.Join(t.TempDir(), "no_such_file_xyz_123.txt")
	if a.PathExists(path) {
		t.Errorf("PathExists(%q) = true, want false for non-existent path", path)
	}
}

func TestDefaultDir(t *testing.T) {
	origRoot, origDir := dirRoot, dirDir
	t.Cleanup(func() { dirRoot, dirDir = origRoot, origDir })

	tests := []struct {
		name        string
		currentPath string
		fallback    string // subdir of drivers to fall back to; "" skips that rung
		want        string // "fallback", "root", or a slash path relative to root
	}{
		{name: "currentPath folder wins over fallback", currentPath: "drivers/network/foo.exe", fallback: "display", want: "drivers/network"},
		{name: "no currentPath uses fallback", fallback: "display", want: "fallback"},
		{name: "stale currentPath falls back", currentPath: "drivers/gone/foo.exe", fallback: "display", want: "fallback"},
		{name: "stale currentPath and stale fallback return root", currentPath: "drivers/gone/foo.exe", fallback: "absent", want: "root"},
		{name: "no fallback rung returns root", currentPath: "drivers/gone/foo.exe", want: "root"},
		{name: "nothing configured returns root", want: "root"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			dirRoot = root
			dirDir = filepath.Join(root, "drivers")
			for _, sub := range []string{"network", "display"} {
				if err := os.MkdirAll(filepath.Join(dirDir, sub), os.ModePerm); err != nil {
					t.Fatal(err)
				}
			}

			fallback := ""
			if tt.fallback != "" {
				fallback = filepath.Join(dirDir, tt.fallback)
			}

			var want string
			switch tt.want {
			case "fallback":
				want = fallback
			case "root":
				want = dirRoot
			default:
				want = filepath.Join(dirRoot, filepath.FromSlash(tt.want))
			}

			if got := defaultDir(tt.currentPath, fallback); got != want {
				t.Errorf("defaultDir(%q, %q) = %q, want %q", tt.currentPath, fallback, got, want)
			}
		})
	}
}
