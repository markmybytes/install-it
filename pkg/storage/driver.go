package storage

import (
	"errors"

	"gorm.io/gorm"

	"install-it/pkg/errcode"
)

type DriverType string

const (
	Network       DriverType = "network"
	Display       DriverType = "display"
	Miscellaneous DriverType = "miscellaneous"
)

type DriverGroup struct {
	Id                uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	Name              string     `json:"name"`
	Type              DriverType `json:"type"`
	MutuallyExclusive bool       `json:"mutuallyExclusive"`
	Position          int        `json:"-" gorm:"index"`
	Drivers           []*Driver  `json:"drivers" gorm:"foreignKey:GroupId;constraint:OnDelete:CASCADE"`
}

type Driver struct {
	Id              uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	GroupId         uint       `json:"-" gorm:"index"`
	Name            string     `json:"name"`
	Type            DriverType `json:"type"`
	Path            string     `json:"path"`
	Flags           []string   `json:"flags" gorm:"serializer:json"`
	MinExeTime      float32    `json:"minExeTime"`
	AllowRtCodes    []int32    `json:"allowRtCodes" gorm:"serializer:json"`
	Incompatibles   []*Driver  `json:"-" gorm:"many2many:driver_incompatibles;joinForeignKey:DriverID;joinReferences:IncompatibleDriverID;constraint:OnDelete:CASCADE"`
	IncompatibleIds []uint     `json:"incompatibles" gorm:"-"`
}

func populateIncompatibleIds(d *Driver) {
	d.IncompatibleIds = make([]uint, len(d.Incompatibles))
	for i, inc := range d.Incompatibles {
		d.IncompatibleIds[i] = inc.Id
	}
}

type DriverGroupStorage struct {
	db *Database
}

func NewDriverGroupStorage(db *Database) *DriverGroupStorage {
	return &DriverGroupStorage{db: db}
}

func (s *DriverGroupStorage) All() ([]DriverGroup, error) {
	var groups []*DriverGroup
	if err := s.db.DB().Preload("Drivers.Incompatibles").Order("position").Find(&groups).Error; err != nil {
		return nil, errcode.New("errStorageReadFailed")
	}
	result := make([]DriverGroup, len(groups))
	for i, g := range groups {
		for _, d := range g.Drivers {
			populateIncompatibleIds(d)
		}
		result[i] = *g
	}
	return result, nil
}

func (s *DriverGroupStorage) Get(id uint) (DriverGroup, error) {
	var group DriverGroup
	if err := s.db.DB().Preload("Drivers.Incompatibles").First(&group, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return DriverGroup{}, errcode.New("errStorageNotFound")
		}
		return DriverGroup{}, errcode.New("errStorageReadFailed")
	}
	for _, d := range group.Drivers {
		populateIncompatibleIds(d)
	}
	return group, nil
}

func (s *DriverGroupStorage) Add(group DriverGroup) error {
	return s.db.DB().Transaction(func(tx *gorm.DB) error {
		var maxPos int
		tx.Model(&DriverGroup{}).Select("COALESCE(MAX(position), -1)").Scan(&maxPos)
		group.Position = maxPos + 1
		if err := tx.Create(&group).Error; err != nil {
			return errcode.New("errStorageWriteFailed")
		}
		return nil
	})
}

func (s *DriverGroupStorage) Update(group DriverGroup) error {
	return s.db.DB().Transaction(func(tx *gorm.DB) error {
		var existing []*Driver
		if err := tx.Where("group_id = ?", group.Id).Find(&existing).Error; err != nil {
			return errcode.New("errStorageReadFailed")
		}

		newDriverIds := make(map[uint]bool)
		for _, d := range group.Drivers {
			if d.Id != 0 {
				newDriverIds[d.Id] = true
			}
		}

		var deletedIds []uint
		for _, d := range existing {
			if !newDriverIds[d.Id] {
				deletedIds = append(deletedIds, d.Id)
			}
		}

		if len(deletedIds) > 0 {
			if err := tx.Delete(&Driver{}, "id IN ?", deletedIds).Error; err != nil {
				return errcode.New("errStorageWriteFailed")
			}
		}

		if err := tx.Model(&DriverGroup{}).Where("id = ?", group.Id).Updates(map[string]any{
			"name":               group.Name,
			"type":               group.Type,
			"mutually_exclusive": group.MutuallyExclusive,
		}).Error; err != nil {
			return errcode.New("errStorageWriteFailed")
		}

		for _, d := range group.Drivers {
			d.GroupId = group.Id
			if d.Id == 0 {
				if err := tx.Omit("Incompatibles").Create(d).Error; err != nil {
					return errcode.New("errStorageWriteFailed")
				}
			} else {
				if err := tx.Omit("Incompatibles").Save(d).Error; err != nil {
					return errcode.New("errStorageWriteFailed")
				}
			}
			incompats := make([]*Driver, len(d.IncompatibleIds))
			for i, iid := range d.IncompatibleIds {
				incompats[i] = &Driver{Id: iid}
			}
			if err := tx.Model(d).Association("Incompatibles").Replace(incompats); err != nil {
				return errcode.New("errStorageWriteFailed")
			}
		}

		return nil
	})
}

func (s *DriverGroupStorage) Remove(id uint) error {
	return s.db.DB().Transaction(func(tx *gorm.DB) error {
		result := tx.Delete(&DriverGroup{}, id)
		if result.Error != nil {
			return errcode.New("errStorageWriteFailed")
		}
		if result.RowsAffected == 0 {
			return errcode.New("errStorageNotFound")
		}
		return nil
	})
}

func (s *DriverGroupStorage) Clone(id uint) error {
	return s.db.DB().Transaction(func(tx *gorm.DB) error {
		var original DriverGroup
		if err := tx.Preload("Drivers.Incompatibles").First(&original, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errcode.New("errStorageNotFound")
			}
			return errcode.New("errStorageReadFailed")
		}

		var maxPos int
		tx.Model(&DriverGroup{}).Select("COALESCE(MAX(position), -1)").Scan(&maxPos)

		newGroup := DriverGroup{
			Name:              original.Name + " (copy)",
			Type:              original.Type,
			MutuallyExclusive: original.MutuallyExclusive,
			Position:          maxPos + 1,
		}
		if err := tx.Omit("Drivers").Create(&newGroup).Error; err != nil {
			return errcode.New("errStorageWriteFailed")
		}

		oldToNew := make(map[uint]*Driver, len(original.Drivers))
		for _, d := range original.Drivers {
			newDriver := &Driver{
				GroupId:      newGroup.Id,
				Name:         d.Name,
				Type:         d.Type,
				Path:         d.Path,
				Flags:        d.Flags,
				MinExeTime:   d.MinExeTime,
				AllowRtCodes: d.AllowRtCodes,
			}
			if err := tx.Create(newDriver).Error; err != nil {
				return errcode.New("errStorageWriteFailed")
			}
			oldToNew[d.Id] = newDriver
		}

		for _, d := range original.Drivers {
			if len(d.Incompatibles) == 0 {
				continue
			}
			newDriver := oldToNew[d.Id]
			newIncompats := make([]*Driver, 0, len(d.Incompatibles))
			for _, inc := range d.Incompatibles {
				if mapped, ok := oldToNew[inc.Id]; ok {
					newIncompats = append(newIncompats, mapped)
				}
			}
			if len(newIncompats) > 0 {
				if err := tx.Model(newDriver).Association("Incompatibles").Replace(newIncompats); err != nil {
					return errcode.New("errStorageWriteFailed")
				}
			}
		}

		return nil
	})
}

func (s *DriverGroupStorage) MoveBehind(id uint, index int) error {
	return s.db.DB().Transaction(func(tx *gorm.DB) error {
		var srcPos int
		if err := tx.Model(&DriverGroup{}).Select("position").Where("id = ?", id).Scan(&srcPos).Error; err != nil {
			return errcode.New("errStorageReadFailed")
		}

		destPos := index + 1

		if srcPos == destPos {
			return nil
		}

		if srcPos < destPos {
			if err := tx.Model(&DriverGroup{}).
				Where("position > ? AND position <= ?", srcPos, destPos).
				UpdateColumn("position", gorm.Expr("position - 1")).Error; err != nil {
				return errcode.New("errStorageWriteFailed")
			}
		} else {
			if err := tx.Model(&DriverGroup{}).
				Where("position >= ? AND position < ?", destPos, srcPos).
				UpdateColumn("position", gorm.Expr("position + 1")).Error; err != nil {
				return errcode.New("errStorageWriteFailed")
			}
		}

		if err := tx.Model(&DriverGroup{}).Where("id = ?", id).UpdateColumn("position", destPos).Error; err != nil {
			return errcode.New("errStorageWriteFailed")
		}
		return nil
	})
}
