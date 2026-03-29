package jobs

import (
	"context"
	"log"
	"time"
)

func CleanupTask(ctx context.Context) {
	// You can now create child spans easily
	//_, dbSpan := telemetry.Tracer.Start(ctx, "database.cleanup_old_records")
	//defer dbSpan.End()

	// Simulate work
	time.Sleep(2 * time.Second)
	log.Println("Cleanup completed")
}
