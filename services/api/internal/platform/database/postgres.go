// Package database provides PostgreSQL connectivity via GORM.
package database

import (
	"fmt"
	"log/slog"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Config holds the settings required to open a database connection.
type Config struct {
	// DSN is the PostgreSQL connection string.
	DSN string
}

// Open establishes a GORM PostgreSQL connection using the settings in cfg.
func Open(cfg Config, log *slog.Logger) (*gorm.DB, error) {
	return openPostgres(cfg.DSN, log)
}

// openPostgres opens a standard PostgreSQL connection via gorm+pgx.
func openPostgres(dsn string, log *slog.Logger) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("database: open: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("database: get sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("database: ping: %w", err)
	}

	log.Info("database connected")
	return db, nil
}
