package storage

import (
	"encoding/json"
	"errors"
	"os"
)

type AppSetting struct {
	CreatePartition    bool          `json:"create_partition"`
	SetPassword        bool          `json:"set_password"`
	Password           string        `json:"password"`
	ParallelInstall    bool          `json:"parallel_install"`
	SuccessAction      SuccessAction `json:"success_action"`
	SuccessActionDelay int           `json:"success_action_delay"`
	Language           string        `json:"language"`
	DriverDownloadUrl  string        `json:"driver_download_url"`
	AutoCheckUpdate    bool          `json:"auto_check_update"`
	HideNotFound       bool          `json:"hide_not_found"`
	AllowPreRelease    bool          `json:"allow_pre_release"`
}

type SuccessAction string

const (
	Nothing  SuccessAction = "nothing"
	Shutdown SuccessAction = "shutdown"
	Reboot   SuccessAction = "reboot"
	Firmware SuccessAction = "firmware"
)

type AppSettingStorage struct {
	Path    string
	setting AppSetting
}

func (s *AppSettingStorage) All() (AppSetting, error) {
	bytes, err := os.ReadFile(s.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.setting = AppSetting{
				AutoCheckUpdate:    true,
				Language:           "en",
				ParallelInstall:    true,
				SuccessAction:      Nothing,
				SuccessActionDelay: 5,
			}
			return s.setting, s.write()
		}
		return AppSetting{}, err
	}

	if err := json.Unmarshal(bytes, &s.setting); err != nil {
		return AppSetting{}, err
	}
	return s.setting, nil
}

func (s *AppSettingStorage) Update(v AppSetting) (AppSetting, error) {
	s.setting = v
	return s.setting, s.write()
}

func (s *AppSettingStorage) write() error {
	bytes, err := json.Marshal(s.setting)
	if err != nil {
		return err
	}
	return os.WriteFile(s.Path, bytes, 0644)
}
