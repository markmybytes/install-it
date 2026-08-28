package matching

import (
	"regexp"
	"slices"
	"strings"

	"install-it/pkg/errcode"
	"install-it/pkg/storage"
)

// RuleSetReader provides read access to rule sets for matching evaluation.
// It is satisfied by *storage.RuleSetStorage.
type RuleSetReader interface {
	All() ([]storage.RuleSet, error)
}

// HardwareQuerier provides formatted hardware strings per rule source.
// It is satisfied by WMIHardwareQuerier (and fakes in tests).
type HardwareQuerier interface {
	HardwareMap() (map[storage.RuleSource][]string, error)
}

// Matcher evaluates rule sets against live hardware to find matched driver groups.
type Matcher struct {
	rules    RuleSetReader
	hardware HardwareQuerier
}

// NewMatcher creates a Matcher with the given rule reader and hardware querier.
func NewMatcher(rules RuleSetReader, hw HardwareQuerier) *Matcher {
	return &Matcher{rules: rules, hardware: hw}
}

// MatchedGroupIds evaluates all rule sets against live hardware and returns
// the IDs of driver groups that should be selected.
func (m *Matcher) MatchedGroupIds() ([]uint, error) {
	hw, err := m.hardware.HardwareMap()
	if err != nil {
		return nil, errcode.New("errHardwareQueryFailed")
	}

	ruleSets, err := m.rules.All()
	if err != nil {
		return nil, errcode.New("errRulesLoadFailed")
	}

	seen := make(map[uint]bool)
	matched := []uint{}

	for _, rs := range ruleSets {
		results := make([]bool, len(rs.Rules))
		for i, rule := range rs.Rules {
			inputs := hw[rule.Source]
			results[i] = anyMatchesRule(rule, inputs)
		}

		if rs.ShouldHitAll && !slices.Contains(results, false) || !rs.ShouldHitAll && slices.Contains(results, true) {
			for _, dg := range rs.DriverGroups {
				if !seen[dg.Id] {
					matched = append(matched, dg.Id)
					seen[dg.Id] = true
				}
			}
		}
	}

	return matched, nil
}

// anyMatchesRule returns true if any hardware input matches the rule.
func anyMatchesRule(rule storage.Rule, inputs []string) bool {
	for _, input := range inputs {
		if testRule(rule, input) {
			return true
		}
	}
	return false
}

// testRule tests whether a single input string satisfies the rule,
// matching the logic of testMatchRule() in frontend/src/utils/index.ts.
func testRule(rule storage.Rule, input string) bool {
	cmpInput := input
	values := make([]string, len(rule.Values))
	copy(values, rule.Values)

	if !rule.IsCaseSensitive {
		cmpInput = strings.ToLower(cmpInput)
		for i, v := range values {
			values[i] = strings.ToLower(v)
		}
	}

	hits := make([]bool, len(values))
	for i, v := range values {
		switch rule.Operator {
		case storage.Contain:
			hits[i] = strings.Contains(cmpInput, v)
		case storage.NotContain:
			hits[i] = !strings.Contains(cmpInput, v)
		case storage.Equal:
			hits[i] = cmpInput == v
		case storage.NotEqual:
			hits[i] = cmpInput != v
		case storage.Regex:
			pattern := v
			if !rule.IsCaseSensitive {
				pattern = "(?i)" + pattern
			}
			re, err := regexp.Compile(pattern)
			if err == nil {
				hits[i] = re.MatchString(cmpInput)
			}
		}
	}

	if rule.ShouldHitAll {
		return !slices.Contains(hits, false)
	}
	return slices.Contains(hits, true)
}
