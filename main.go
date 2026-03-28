package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/robfig/cron/v3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("cron-bot")

// initTelemetry sets up OpenTelemetry with OTLP (gRPC) exporter
func initTelemetry(ctx context.Context) (func(context.Context) error, error) {
	// Change this to your collector endpoint (e.g. localhost:4317 or Tempo, Jaeger, etc.)
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint("localhost:4317"), // Jaeger OTLP gRPC port
		otlptracegrpc.WithInsecure(),                 // No TLS for local
		otlptracegrpc.WithTimeout(10*time.Second),
	)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("cron-bot"),
			semconv.ServiceVersionKey.String("1.0.0"),
			attribute.String("environment", "development"),
		)),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp.Shutdown, nil
}

// cleanupJob is an example cron task
func cleanupJob(ctx context.Context) {
	// Every cron execution gets its own root trace (no parent context to extract)
	ctx, span := tracer.Start(ctx, "cron.cleanup",
		trace.WithAttributes(
			attribute.String("job.type", "cleanup"),
			attribute.String("cron.schedule", "0 * * * *"), // every hour
		))
	defer span.End()

	log.Println("Starting cleanup job...")

	// Simulate work
	time.Sleep(2 * time.Second)

	// You can create child spans inside the job
	//_, dbSpan := tracer.Start(ctx, "database.cleanup")
	//defer dbSpan.End()
	// ... do database work ...

	span.SetAttributes(attribute.Bool("success", true))
	log.Println("Cleanup job completed")
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Setup OpenTelemetry
	shutdown, err := initTelemetry(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize telemetry: %v", err)
	}
	defer func() {
		if err := shutdown(ctx); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()

	// Create cron scheduler
	c := cron.New(
		cron.WithSeconds(), // optional: support seconds in schedule
		cron.WithLogger(cron.PrintfLogger(log.New(os.Stdout, "cron: ", log.LstdFlags))),
	)

	// Add jobs
	_, err = c.AddFunc("*/10 * * * * *", func() {
		cleanupJob(ctx)
	})
	if err != nil {
		log.Fatalf("Failed to add cron job: %v", err)
	}

	// You can add more jobs...
	// c.AddFunc("0 0 2 * * *", dailyReportJob) // every day at 2:00 AM

	fmt.Println("Cron bot started. Press Ctrl+C to stop.")
	c.Start()
	select {
	case <-ctx.Done():
		fmt.Println("\nShutting down cron bot...")

		c.Stop()
		fmt.Println("Cron bot stopped.")
		stop()
	}

}
