package database

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Open creates (if needed) and opens the SQLite database with production-minded
// pragmas for foreign keys, busy timeout, and WAL mode.
//
// SQLite is appropriate for this local assessment. Higher-write-concurrency
// production systems should prefer PostgreSQL (or similar).
func Open(path string, logLevel string) (*gorm.DB, error) {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create data directory %q: %w", dir, err)
		}
	}

	// glebarez/modernc accepts a file path; pragmas are applied explicitly below
	// because DSN query flags are not portable across SQLite drivers.
	dsn := filepath.ToSlash(path)

	gormLog := logger.Default.LogMode(logger.Warn)
	switch logLevel {
	case "debug":
		gormLog = logger.Default.LogMode(logger.Info)
	case "error":
		gormLog = logger.Default.LogMode(logger.Error)
	case "silent":
		gormLog = logger.Default.LogMode(logger.Silent)
	}

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger:                 gormLog,
		SkipDefaultTransaction: false,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get underlying sql.DB: %w", err)
	}

	// SQLite writers serialize; keep a small pool to avoid SQLITE_BUSY storms.
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(0)

	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA busy_timeout = 5000",
		"PRAGMA journal_mode = WAL",
	}
	for _, p := range pragmas {
		if err := db.Exec(p).Error; err != nil {
			return nil, fmt.Errorf("apply %s: %w", p, err)
		}
	}

	var fk int
	if err := db.Raw("PRAGMA foreign_keys").Scan(&fk).Error; err != nil {
		return nil, fmt.Errorf("verify foreign_keys pragma: %w", err)
	}
	if fk != 1 {
		return nil, fmt.Errorf("SQLite foreign_keys not enabled (got %d)", fk)
	}

	return db, nil
}
