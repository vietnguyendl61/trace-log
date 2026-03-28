package cron

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"
	"trace-log/internal/cron/jobs"
	"trace-log/internal/telemetry"

	"github.com/robfig/cron/v3"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Manager manages all cron jobs
type Manager struct {
	c *cron.Cron
}

// New creates a new cron manager
func New() *Manager {
	return &Manager{
		c: cron.New(
			cron.WithSeconds(),
			cron.WithLogger(cron.PrintfLogger(log.New(os.Stdout, "cron: ", log.LstdFlags))),
		),
	}
}

// Init adds jobs and starts the scheduler
func (m *Manager) Init() {
	// Example: Add jobs with auto span wrapper (from previous examples)
	_, err := m.c.AddJob("*/10 * * * * *", wrapJob("cleanup", jobs.CleanupTask))
	if err != nil {
		log.Println("error adding cron job:", err)
		return
	}

	m.c.Start()
	log.Println("✅ Cron scheduler started")
}

// Stop gracefully stops the cron
func (m *Manager) Stop() {
	log.Println("Stopping cron scheduler...")
	m.c.Stop()
	log.Println("Cron scheduler stopped")
}

// wrapJob automatically creates a root span for every cron execution
func wrapJob(name string, jobFunc func(context.Context)) cron.Job {
	return cron.FuncJob(func() {
		ctx := context.Background() // Each cron run = new root trace

		ctx, span := telemetry.Tracer.Start(ctx, "cron."+name,
			trace.WithAttributes(
				attribute.String("job.name", name),
				attribute.String("job.type", "scheduled"),
			))
		defer span.End()

		start := time.Now()
		defer func() {
			if r := recover(); r != nil {
				span.RecordError(fmt.Errorf("panic: %v", r))
				span.SetStatus(codes.Error, "panic occurred")
			}
			span.SetAttributes(attribute.Int64("duration_ms", time.Since(start).Milliseconds()))
		}()

		jobFunc(ctx) // Pass the context so child spans can be created inside
	})
}
