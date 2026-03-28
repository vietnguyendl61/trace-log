package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"
	"trace-log/internal/database"
	"trace-log/internal/telemetry"

	"trace-log/internal/cron"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	shutdownTelemetry, err := telemetry.Init(ctx)
	if err != nil {
		log.Fatalf("Failed to init telemetry: %v", err)
	}
	defer shutdownTelemetry(ctx)

	// 2. Init Database (GORM + Pool + Tracing)
	dbCfg := &database.Config{
		DSN:             "postgres://postgres:zxcasdqwe~123456789@localhost:5432/opentelemetry_test?sslmode=disable",
		MaxIdleConns:    10,
		MaxOpenConns:    50,
		ConnMaxLifetime: 1 * time.Hour,
	}
	db, err := database.InitPostgresConnection(dbCfg)
	if err != nil {
		log.Fatalf("Failed to init database: %v", err)
	}
	DB := database.New(db)

	cronManager := cron.New()
	cronManager.Init()
	select {
	case <-ctx.Done():
		fmt.Println("\nShutting down cron bot...")

		cronManager.Stop()
		err = DB.Close()
		if err != nil {
			log.Fatalf("Failed to close database connection: %v", err)
		}
		log.Println("Database closed")
	}

}
