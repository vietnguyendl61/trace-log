package jobs

import (
	"context"
	"log"
	"time"
	"trace-log/internal/telemetry"
)

func CleanupTask(ctx context.Context) {
	// You can now create child spans easily
	_, dbSpan := telemetry.Tracer.Start(ctx, "database.cleanup_old_records")
	defer dbSpan.End()

	// Simulate work
	time.Sleep(1 * time.Second)
	log.Println("Cleanup completed")
}
