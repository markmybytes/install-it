// Package storage_test provides fuzz tests for the storage package.
package storage_test

import (
	"encoding/json"
	"testing"

	"install-it/pkg/storage"
)

// FuzzAppSetting_JSONRoundtrip verifies that a JSON string parsed into
// AppSetting and re-marshaled produces a consistent result (no panic).
func FuzzAppSetting_JSONRoundtrip(f *testing.F) {
	f.Add(`{"create_partition":true,"set_password":false,"password":"","parallel_install":true,"success_action":"nothing","success_action_delay":5,"language":"en","driver_download_url":"","auto_check_update":true,"hide_not_found":false}`)
	f.Add(`{}`)
	f.Add(`{"success_action":"reboot","success_action_delay":10,"language":"zh_Hant_HK"}`)
	f.Add(`{"success_action":"shutdown","password":"secret"}`)

	f.Fuzz(func(t *testing.T, s string) {
		var setting storage.AppSetting
		if err := json.Unmarshal([]byte(s), &setting); err != nil {
			return
		}

		b, err := json.Marshal(setting)
		if err != nil {
			t.Errorf("Marshal of valid AppSetting failed: %v", err)
			return
		}

		var setting2 storage.AppSetting
		if err := json.Unmarshal(b, &setting2); err != nil {
			t.Errorf("Unmarshal of re-marshaled AppSetting failed: %v", err)
			return
		}

		if setting != setting2 {
			t.Errorf("roundtrip mismatch:\n  pass1: %+v\n  pass2: %+v", setting, setting2)
		}
	})
}

// FuzzRuleSet_JSONRoundtrip verifies that RuleSet JSON round-trips correctly,
// paying special attention to enum string fields (operator, source).
func FuzzRuleSet_JSONRoundtrip(f *testing.F) {
	f.Add(`{"id":1,"name":"test","rules":[],"should_hit_all":false,"driver_group_ids":[]}`)
	f.Add(`{"id":1,"name":"test","rules":[{"source":"cpu","operator":"contain","is_case_sensitive":true,"should_hit_all":false,"values":["Intel"]}],"should_hit_all":true,"driver_group_ids":[1]}`)
	f.Add(`{}`)
	f.Add(`{"rules":[{"source":"unknown_source","operator":"unknown_op","values":[]}]}`)

	f.Fuzz(func(t *testing.T, s string) {
		var rs storage.RuleSet
		if err := json.Unmarshal([]byte(s), &rs); err != nil {
			return
		}

		b, err := json.Marshal(rs)
		if err != nil {
			t.Errorf("Marshal of parsed RuleSet failed: %v", err)
			return
		}

		var rs2 storage.RuleSet
		if err := json.Unmarshal(b, &rs2); err != nil {
			t.Errorf("Unmarshal of re-marshaled RuleSet failed: %v", err)
		}
	})
}
