package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAppSettingStorage_AllWithoutExistingFile(t *testing.T) {
	s := &AppSettingStorage{Path: filepath.Join(t.TempDir(), "setting.json")}

	result, err := s.All()

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	// Verify default values
	if result.AutoCheckUpdate != true {
		t.Errorf("expected AutoCheckUpdate to be true, got %v", result.AutoCheckUpdate)
	}
	if result.Language != "en" {
		t.Errorf("expected Language to be 'en', got %s", result.Language)
	}
	if result.ParallelInstall != true {
		t.Errorf("expected ParallelInstall to be true, got %v", result.ParallelInstall)
	}
	if result.SuccessAction != Nothing {
		t.Errorf("expected SuccessAction to be 'nothing', got %s", result.SuccessAction)
	}
	if result.SuccessActionDelay != 5 {
		t.Errorf("expected SuccessActionDelay to be 5, got %d", result.SuccessActionDelay)
	}
}

func TestAppSettingStorage_AllWithExistingFile(t *testing.T) {
	existingSetting := AppSetting{
		CreatePartition:    true,
		SetPassword:        true,
		Password:           "test123",
		ParallelInstall:    false,
		SuccessAction:      Reboot,
		SuccessActionDelay: 10,
		Language:           "zh_Hant_HK",
		DriverDownloadUrl:  "https://example.com",
		AutoCheckUpdate:    false,
		HideNotFound:       true,
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "setting.json")
	bytes := jsonMarshal(existingSetting)
	os.WriteFile(path, bytes, 0644)

	s := &AppSettingStorage{Path: path}

	result, err := s.All()

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	// Verify loaded values
	if result.CreatePartition != true {
		t.Errorf("expected CreatePartition to be true, got %v", result.CreatePartition)
	}
	if result.SetPassword != true {
		t.Errorf("expected SetPassword to be true, got %v", result.SetPassword)
	}
	if result.Password != "test123" {
		t.Errorf("expected Password to be 'test123', got %s", result.Password)
	}
	if result.ParallelInstall != false {
		t.Errorf("expected ParallelInstall to be false, got %v", result.ParallelInstall)
	}
	if result.SuccessAction != Reboot {
		t.Errorf("expected SuccessAction to be 'reboot', got %s", result.SuccessAction)
	}
	if result.SuccessActionDelay != 10 {
		t.Errorf("expected SuccessActionDelay to be 10, got %d", result.SuccessActionDelay)
	}
	if result.Language != "zh_Hant_HK" {
		t.Errorf("expected Language to be 'zh_Hant_HK', got %s", result.Language)
	}
	if result.DriverDownloadUrl != "https://example.com" {
		t.Errorf("expected DriverDownloadUrl to be 'https://example.com', got %s", result.DriverDownloadUrl)
	}
	if result.AutoCheckUpdate != false {
		t.Errorf("expected AutoCheckUpdate to be false, got %v", result.AutoCheckUpdate)
	}
	if result.HideNotFound != true {
		t.Errorf("expected HideNotFound to be true, got %v", result.HideNotFound)
	}
}

func TestAppSettingStorage_Update(t *testing.T) {
	s := &AppSettingStorage{Path: filepath.Join(t.TempDir(), "setting.json")}

	newSetting := AppSetting{
		CreatePartition:    true,
		SetPassword:        true,
		Password:           "newpass",
		ParallelInstall:    false,
		SuccessAction:      Shutdown,
		SuccessActionDelay: 30,
		Language:           "zh_Hant_HK",
		DriverDownloadUrl:  "https://drivers.example.com",
		AutoCheckUpdate:    false,
		HideNotFound:       true,
	}

	result, err := s.Update(newSetting)

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	if result.Password != "newpass" {
		t.Errorf("expected Password to be 'newpass', got %s", result.Password)
	}
	if result.SuccessAction != Shutdown {
		t.Errorf("expected SuccessAction to be 'shutdown', got %s", result.SuccessAction)
	}
	if result.SuccessActionDelay != 30 {
		t.Errorf("expected SuccessActionDelay to be 30, got %d", result.SuccessActionDelay)
	}
}

func TestAppSettingStorage_UpdateMultipleTimes(t *testing.T) {
	s := &AppSettingStorage{Path: filepath.Join(t.TempDir(), "setting.json")}

	setting1 := AppSetting{
		Language:        "en",
		AutoCheckUpdate: true,
		SuccessAction:   Nothing,
	}
	s.Update(setting1)

	setting2 := AppSetting{
		Language:        "zh_Hant_HK",
		AutoCheckUpdate: false,
		SuccessAction:   Reboot,
	}
	result, err := s.Update(setting2)

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	if result.Language != "zh_Hant_HK" {
		t.Errorf("expected Language to be 'zh_Hant_HK', got %s", result.Language)
	}
	if result.AutoCheckUpdate != false {
		t.Errorf("expected AutoCheckUpdate to be false, got %v", result.AutoCheckUpdate)
	}
	if result.SuccessAction != Reboot {
		t.Errorf("expected SuccessAction to be 'reboot', got %s", result.SuccessAction)
	}
}

func TestAppSettingStorage_UpdateCachesResult(t *testing.T) {
	s := &AppSettingStorage{Path: filepath.Join(t.TempDir(), "setting.json")}

	setting := AppSetting{
		Language:        "en",
		AutoCheckUpdate: true,
		SuccessAction:   Nothing,
	}
	s.Update(setting)

	result, _ := s.All()

	if result.Language != "en" {
		t.Errorf("expected cached Language to be 'en', got %s", result.Language)
	}
}

// TestLegacySettingMigration simulates loading a settings file that predates
// the AllowPreRelease field. The field must default to false with no error.
func TestLegacySettingMigration(t *testing.T) {
	// JSON from an older version — AllowPreRelease / allow_pre_release key absent.
	legacyJSON := `{
		"create_partition": true,
		"set_password": false,
		"password": "",
		"parallel_install": true,
		"success_action": "nothing",
		"success_action_delay": 5,
		"language": "en",
		"driver_download_url": "",
		"auto_check_update": true,
		"hide_not_found": false
	}`

	dir := t.TempDir()
	path := filepath.Join(dir, "setting.json")
	os.WriteFile(path, []byte(legacyJSON), 0644)

	s := &AppSettingStorage{Path: path}

	result, err := s.All()
	if err != nil {
		t.Fatalf("All() returned unexpected error: %v", err)
	}
	if result.AllowPreRelease != false {
		t.Errorf("AllowPreRelease = %v, want false (zero default for missing key)", result.AllowPreRelease)
	}
	// Sanity-check that other fields decoded correctly.
	if result.Language != "en" {
		t.Errorf("Language = %q, want %q", result.Language, "en")
	}
	if result.AutoCheckUpdate != true {
		t.Errorf("AutoCheckUpdate = %v, want true", result.AutoCheckUpdate)
	}
}

// jsonMarshal is a test helper that panics on marshal error.
func jsonMarshal(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
