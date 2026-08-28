package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/wailsapp/go-webview2/webviewloader"
	wails_runtime "github.com/wailsapp/wails/v2/pkg/runtime"

	"install-it/pkg/errcode"
)

type App struct {
	ctx context.Context
}

func NewApp() *App {
	return &App{}
}

func (m *App) SetContext(ctx context.Context) {
	m.ctx = ctx
}

func (a *App) Cwd() (string, error) {
	if exePath, err := os.Executable(); err != nil {
		return "", errcode.New("errAppExecLookup")
	} else {
		return filepath.Dir(exePath), nil
	}
}

func (a *App) SelectFolder(relative bool) (string, error) {
	if path, err := wails_runtime.OpenDirectoryDialog(a.ctx, wails_runtime.OpenDialogOptions{}); err != nil || path == "" {
		return "", errcode.New("errAppDialogOpen")
	} else if relative {
		exePath, err := os.Executable()
		if err != nil {
			return "", errcode.New("errAppExecLookup")
		}
		rel, err := filepath.Rel(filepath.Dir(exePath), path)
		if err != nil {
			return "", errcode.New("errAppExecLookup")
		}
		return rel, nil
	} else {
		return path, nil
	}
}

func (a *App) SelectFile(relative bool) (string, error) {
	if path, err := wails_runtime.OpenFileDialog(a.ctx, wails_runtime.OpenDialogOptions{}); err != nil || path == "" {
		return "", errcode.New("errAppDialogOpen")
	} else if relative {
		exePath, err := os.Executable()
		if err != nil {
			return "", errcode.New("errAppExecLookup")
		}
		rel, err := filepath.Rel(filepath.Dir(exePath), path)
		if err != nil {
			return "", errcode.New("errAppExecLookup")
		}
		return rel, nil
	} else {
		return path, nil
	}
}

func (a App) PathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (a App) ExecutableExists(path string) bool {
	_, err := exec.LookPath(path)
	return err == nil
}

func (a App) WebView2Version() (string, error) {
	v, err := webviewloader.GetAvailableCoreWebView2BrowserVersionString(pathWV2)
	if err != nil {
		return "", errcode.New("errAppWebviewVersion")
	}
	return v, nil
}

func (a App) WebView2Path() string {
	return pathWV2
}

func (a App) AppConfigPath() string {
	return dirConf
}

func (a App) AppDriverPath() string {
	return dirDir
}

func (a App) AppVersion() string {
	return version.String()
}

func (a App) AppBinaryType() string {
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x64"
	} else if arch == "386" {
		arch = "x86"
	}
	return fmt.Sprintf("%s-%s", runtime.GOOS, arch)
}
