package main

import (
	"context"
	"embed"
	"errors"
	"install-it/pkg/cputemp"
	"install-it/pkg/errcode"
	"install-it/pkg/execute"
	"install-it/pkg/matching"
	"install-it/pkg/porter"
	"install-it/pkg/status"
	"install-it/pkg/storage"
	"install-it/pkg/sysinfo"
	"install-it/pkg/update"
	"os"
	"path/filepath"

	"github.com/Masterminds/semver"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

var (
	dirRoot string
	// Path to the configuration directory
	dirConf string
	// Path to the driver directory
	dirDir string
	// Path to the WebView2 executable
	pathWV2      string
	buildVersion string
	// Version struct, parsed from [buildVersion]
	version *semver.Version
	updater *update.Updater

	db             *storage.Database
	groupStorage   *storage.DriverGroupStorage
	ruleSetStorage *storage.RuleSetStorage
	matcher        *matching.Matcher
	appSettings    *storage.AppSettingStorage
)

func init() {
	if v, err := semver.NewVersion(buildVersion); err == nil {
		version = v
	} else {
		version, _ = semver.NewVersion("0.0.0")
	}

	pathExe, err := os.Executable()
	if err != nil {
		panic(err)
	}
	dirRoot = filepath.Dir(pathExe)

	// process Updates
	updater = &update.Updater{DirRoot: dirRoot, Version: version}
	updater.CheckAndApplyUpdates()

	dirConf = filepath.Join(dirRoot, "conf")
	os.MkdirAll(dirConf, os.ModePerm)

	dirDir = filepath.Join(dirRoot, "drivers")
	for _, sub := range []string{"network", "display", "miscellaneous"} {
		os.MkdirAll(filepath.Join(dirDir, sub), os.ModePerm)
	}

	pathWV2 = filepath.Join(dirRoot, "internals", "bin", "WebView2")
	if _, err := os.Stat(pathWV2); err != nil {
		pathWV2 = ""
	}
}

func main() {
	app := &App{}
	mgt := &execute.CommandExecutor{}

	var err error
	db, err = storage.Open(filepath.Join(dirConf, "data.db"))
	if err != nil {
		panic(err)
	}
	if err := db.Migrate(); err != nil {
		panic(err)
	}

	groupStorage = storage.NewDriverGroupStorage(db)
	ruleSetStorage = storage.NewRuleSetStorage(db)
	matcher = matching.NewMatcher(ruleSetStorage, matching.WMIHardwareQuerier{})
	appSettings = &storage.AppSettingStorage{Path: filepath.Join(dirConf, "setting.json")}

	// Porter instance shared between Bind and OnStartup
	porterInstance := &porter.Porter{
		DirRoot: dirRoot,
		Targets: []string{dirConf, dirDir},
		OnBeforeBackup: func() error {
			return db.Close()
		},
		OnAfterImport: func() error {
			if err := db.Reopen(); err != nil {
				return err
			}
			return db.Migrate()
		},
	}

	err = wails.Run(&options.App{
		Title:     "install-it",
		Width:     768,
		Height:    576,
		MinWidth:  640,
		MinHeight: 480,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		ErrorFormatter: func(err error) any {
			var ec *errcode.Error
			if errors.As(err, &ec) {
				return ec // marshaled to {"code": "...", "params": {...}}
			}
			return err.Error() // unmigrated/raw errors fall back to plain string
		},
		OnStartup: func(ctx context.Context) {
			// Fail-safe cleanup
			oldBin := filepath.Join(dirRoot, "install-it.exe.old")
			if _, err := os.Stat(oldBin); err == nil {
				os.Remove(oldBin)
			}

			// Working directory correction
			if cwd, err := os.Getwd(); err == nil {
				if pathExe, err := os.Executable(); err == nil && cwd != filepath.Dir(pathExe) {
					os.Chdir(filepath.Dir(pathExe))
				}
			}

			// Recover orphaned backups from interrupted imports
			porterInstance.RecoverOrphanedBackups()

			app.SetContext(ctx)
			mgt.SetContext(ctx)

			if s, err := appSettings.All(); err == nil && s.EnableCPUTemp {
				go cputemp.Init(dirRoot)
			}
		},
		Bind: []interface{}{
			app,
			mgt,
			updater,
			appSettings,
			groupStorage,
			ruleSetStorage,
			matcher,
			porterInstance,
			&sysinfo.SysInfo{},
		},
		EnumBind: []interface{}{
			[]struct {
				Value  storage.DriverType
				TSName string
			}{
				{storage.Network, "NETWORK"},
				{storage.Display, "DISPLAY"},
				{storage.Miscellaneous, "MISCELLANEOUS"},
			},
			[]struct {
				Value  storage.SuccessAction
				TSName string
			}{
				{storage.Nothing, "NOTHING"},
				{storage.Reboot, "REBOOT"},
				{storage.Shutdown, "SHUTDOWN"},
				{storage.Firmware, "FIRMWARE"},
			},
			[]struct {
				Value  status.Status
				TSName string
			}{
				{status.Pending, "PENDING"},
				{status.Running, "RUNNING"},
				{status.Completed, "COMPLETED"},
				{status.Failed, "FAILED"},
				{status.Aborting, "ABORTING"},
				{status.Aborted, "ABORTED"},
				{status.Skiped, "SKIPED"},
				{status.Speeded, "SPEEDED"},
				{status.Errored, "ERRORED"},
			},
			[]struct {
				Value  storage.RuleSource
				TSName string
			}{
				{storage.Cpu, "CPU"},
				{storage.Motherboard, "MOTHERBOARD"},
				{storage.Gpu, "GPU"},
				{storage.Memory, "MEMORY"},
				{storage.Nic, "NIC"},
				{storage.Storage, "DISK"},
			},
			[]struct {
				Value  storage.RuleOperator
				TSName string
			}{
				{storage.Contain, "CONTAIN"},
				{storage.NotContain, "NOT_CONTAIN"},
				{storage.Equal, "EQUAL"},
				{storage.NotEqual, "NOT_EQUAL"},
				{storage.Regex, "REGEX"},
			},
		},
		Windows: &windows.Options{
			WebviewBrowserPath: pathWV2,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
