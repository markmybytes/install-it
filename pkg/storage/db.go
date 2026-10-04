package storage

import (
	"github.com/glebarez/sqlite"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Database wraps a gorm.DB connection with its file path so it can be reopened
// after the underlying file is replaced (e.g. by Porter import).
type Database struct {
	db   *gorm.DB
	path string
}

// Open creates a new Database backed by a SQLite file at path.
func Open(path string) (*Database, error) {
	db, err := openDB(path)
	if err != nil {
		return nil, err
	}
	return &Database{db: db, path: path}, nil
}

// DB returns the underlying gorm.DB handle.
func (d *Database) DB() *gorm.DB { return d.db }

// Close closes the underlying database connection.
func (d *Database) Close() error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Reopen closes the current connection and opens a new one to the same path.
// This is used after Porter replaces the database file on import.
func (d *Database) Reopen() error {
	if err := d.Close(); err != nil {
		return err
	}
	db, err := openDB(d.path)
	if err != nil {
		return err
	}
	d.db = db
	return nil
}

// Migrate runs all pending database migrations.
func (d *Database) Migrate() error {
	return gormigrate.New(d.db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "2026052601_baseline",
			Migrate: func(tx *gorm.DB) error {
				// Guard for legacy installations that already have raw tables:
				// skip AutoMigrate but let gormigrate record this migration as done.
				if tx.Migrator().HasTable(&DriverGroup{}) {
					return nil
				}
				return tx.AutoMigrate(&DriverGroup{}, &Driver{}, &RuleSet{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable(&RuleSet{}, &Driver{}, &DriverGroup{})
			},
		},
		{
			ID: "2026052901_m2m_and_uint_pks",
			Migrate: func(tx *gorm.DB) error {
				tx.Exec("PRAGMA foreign_keys = ON")
				// Drop old string-PK schema; no user data to preserve (pre-release)
				tx.Migrator().DropTable("rule_sets", "drivers", "driver_groups")
				return tx.AutoMigrate(&DriverGroup{}, &Driver{}, &RuleSet{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable(
					"rule_set_driver_groups", "driver_incompatibles",
					"rule_sets", "drivers", "driver_groups")
			},
		},
		{
			ID: "2026093001_driver_global_exclusive",
			Migrate: func(tx *gorm.DB) error {
				// Additive: AutoMigrate adds the new global_exclusive column
				// and leaves existing columns/rows untouched. No Rollback:
				// a destructive DropColumn would cascade-wipe the
				// driver_incompatibles join edges, and an additive column
				// needs no down-migration.
				return tx.AutoMigrate(&Driver{})
			},
		},
	}).Migrate()
}

func openDB(path string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}
	db.Exec("PRAGMA foreign_keys = ON")
	return db, nil
}
