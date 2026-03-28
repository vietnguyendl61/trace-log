package database

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type Config struct {
	DSN             string
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
}

type Database struct {
	postgresDB *gorm.DB
}

func New(postgresDB *gorm.DB) *Database {
	return &Database{
		postgresDB: postgresDB,
	}
}

func (d *Database) Close() error {
	if d.postgresDB == nil {
		return nil
	}
	sqlDB, err := d.postgresDB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func InitPostgresConnection(cfg *Config) (*gorm.DB, error) {
	if cfg.DSN == "" {
		return nil, errors.New("dsn is required")
	}

	gormDB, err := gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true, // optional: use singular table names
		},
	})
	if err != nil {
		return nil, errors.New(fmt.Sprintf("failed to connect to postgres database: %v", err))
	}

	// Enable OpenTelemetry tracing for all GORM operations
	if err := gormDB.Use(otelgorm.NewPlugin(
		otelgorm.WithDBName("cron-bot-db"),
	)); err != nil {
		return nil, errors.New(fmt.Sprintf("failed to connect to postgres database: %v", err))
	}

	// Get underlying *sql.DB for connection pool configuration
	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, errors.New(fmt.Sprintf("failed to connect to postgres database: %v", err))
	}

	// Connection Pool Settings
	if cfg.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}

	// Optional: Health check
	if err := sqlDB.PingContext(context.Background()); err != nil {
		return nil, errors.New(fmt.Sprintf("failed to ping postgres database: %v", err))
	}

	log.Println("✅ GORM + PostgreSQL connected with OpenTelemetry tracing and connection pool")
	return gormDB, nil
}
