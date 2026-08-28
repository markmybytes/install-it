package storage

import (
	"errors"

	"gorm.io/gorm"

	"install-it/pkg/errcode"
)

type RuleSource string

const (
	Cpu         RuleSource = "cpu"
	Motherboard RuleSource = "motherboard"
	Gpu         RuleSource = "gpu"
	Memory      RuleSource = "memory"
	Nic         RuleSource = "nic"
	Storage     RuleSource = "storage"
)

type RuleOperator string

const (
	Contain    RuleOperator = "contain"
	NotContain RuleOperator = "notContain"
	Equal      RuleOperator = "equal"
	NotEqual   RuleOperator = "notEqual"
	Regex      RuleOperator = "regex"
)

type Rule struct {
	Source          RuleSource   `json:"source"`
	Operator        RuleOperator `json:"operator"`
	IsCaseSensitive bool         `json:"is_case_sensitive"`
	ShouldHitAll    bool         `json:"should_hit_all"`
	Values          []string     `json:"values"`
}

type RuleSet struct {
	Id             uint            `json:"id" gorm:"primaryKey;autoIncrement"`
	Name           string          `json:"name"`
	Rules          []Rule          `json:"rules" gorm:"serializer:json"`
	ShouldHitAll   bool            `json:"should_hit_all"`
	DriverGroups   []*DriverGroup  `json:"-" gorm:"many2many:rule_set_driver_groups;constraint:OnDelete:CASCADE"`
	DriverGroupIds []uint          `json:"driver_group_ids" gorm:"-"`
}

func populateDriverGroupIds(rs *RuleSet) {
	rs.DriverGroupIds = make([]uint, len(rs.DriverGroups))
	for i, dg := range rs.DriverGroups {
		rs.DriverGroupIds[i] = dg.Id
	}
}

type RuleSetStorage struct {
	db *Database
}

func NewRuleSetStorage(db *Database) *RuleSetStorage {
	return &RuleSetStorage{db: db}
}

func (s *RuleSetStorage) All() ([]RuleSet, error) {
	var ruleSets []*RuleSet
	if err := s.db.DB().Preload("DriverGroups").Find(&ruleSets).Error; err != nil {
		return nil, errcode.New("errStorageReadFailed")
	}
	result := make([]RuleSet, len(ruleSets))
	for i, rs := range ruleSets {
		populateDriverGroupIds(rs)
		result[i] = *rs
	}
	return result, nil
}

func (s *RuleSetStorage) Get(id uint) (RuleSet, error) {
	var rs RuleSet
	if err := s.db.DB().Preload("DriverGroups").First(&rs, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return RuleSet{}, errcode.New("errStorageNotFound")
		}
		return RuleSet{}, errcode.New("errStorageReadFailed")
	}
	populateDriverGroupIds(&rs)
	return rs, nil
}

func (s *RuleSetStorage) Add(ruleSet RuleSet) error {
	return s.db.DB().Transaction(func(tx *gorm.DB) error {
		ruleSet.DriverGroups = idsToDriverGroups(ruleSet.DriverGroupIds)
		if err := tx.Omit("DriverGroups.*").Create(&ruleSet).Error; err != nil {
			return errcode.New("errStorageWriteFailed")
		}
		return nil
	})
}

func (s *RuleSetStorage) Update(ruleSet RuleSet) error {
	return s.db.DB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit("DriverGroups").Save(&ruleSet).Error; err != nil {
			return errcode.New("errStorageWriteFailed")
		}
		if err := tx.Model(&ruleSet).Association("DriverGroups").Replace(idsToDriverGroups(ruleSet.DriverGroupIds)); err != nil {
			return errcode.New("errStorageWriteFailed")
		}
		return nil
	})
}

func (s *RuleSetStorage) Remove(id uint) error {
	return s.db.DB().Transaction(func(tx *gorm.DB) error {
		result := tx.Delete(&RuleSet{}, id)
		if result.Error != nil {
			return errcode.New("errStorageWriteFailed")
		}
		if result.RowsAffected == 0 {
			return errcode.New("errStorageNotFound")
		}
		return nil
	})
}

func (s *RuleSetStorage) Clone(id uint) error {
	return s.db.DB().Transaction(func(tx *gorm.DB) error {
		var original RuleSet
		if err := tx.Preload("DriverGroups").First(&original, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errcode.New("errStorageNotFound")
			}
			return errcode.New("errStorageReadFailed")
		}

		newRS := RuleSet{
			Name:         original.Name + " (copy)",
			Rules:        original.Rules,
			ShouldHitAll: original.ShouldHitAll,
			DriverGroups: original.DriverGroups,
		}
		if err := tx.Omit("DriverGroups.*").Create(&newRS).Error; err != nil {
			return errcode.New("errStorageWriteFailed")
		}
		return nil
	})
}

// idsToDriverGroups converts a slice of IDs to DriverGroup pointer stubs for GORM associations.
func idsToDriverGroups(ids []uint) []*DriverGroup {
	groups := make([]*DriverGroup, len(ids))
	for i, id := range ids {
		groups[i] = &DriverGroup{Id: id}
	}
	return groups
}
