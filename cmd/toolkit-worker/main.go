// toolkit-worker is the file-queue worker daemon (internal-network side).
// It scans the shared queue for pending requests and dispatches them to
// configured REST API handlers.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tvmaly/clikit/pkg/filequeue"
	"github.com/tvmaly/clikit/pkg/worker"
)

func main() {
	fs := flag.NewFlagSet("toolkit-worker", flag.ExitOnError)
	queueRoot := fs.String("queue-root", "", "shared drive queue root (required)")
	handlerConfig := fs.String("handler-config", "", "handler registry config file (required)")
	workerID := fs.String("worker-id", "", "unique worker ID (default: hostname-pid)")
	pollInterval := fs.Duration("poll-interval", time.Second, "scan interval")
	staleClaim := fs.Duration("stale-claim", 5*time.Minute, "stale claim threshold")
	retentionDone := fs.Duration("retention-done", 24*time.Hour, "completed request retention")
	retentionDead := fs.Duration("retention-dead", 72*time.Hour, "dead-letter retention")
	cleanupInterval := fs.Duration("cleanup-interval", time.Hour, "cleanup run interval")
	verbose := fs.Bool("verbose", false, "enable debug logging")
	fs.Parse(os.Args[1:])

	if *queueRoot == "" || *handlerConfig == "" {
		fmt.Fprintln(os.Stderr, "--queue-root and --handler-config are required")
		os.Exit(1)
	}

	if *workerID == "" {
		hostname, _ := os.Hostname()
		*workerID = fmt.Sprintf("%s-%d", hostname, os.Getpid())
	}

	if *verbose {
		log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	}

	// Load handler config.
	hc, err := worker.LoadHandlerConfig(*handlerConfig)
	if err != nil {
		log.Fatalf("loading handler config: %v", err)
	}

	// Ensure queue directories.
	if err := filequeue.EnsureQueueDirs(*queueRoot); err != nil {
		log.Fatalf("ensuring queue dirs: %v", err)
	}

	// Build dispatcher.
	d := &worker.Dispatcher{
		QueueRoot:    *queueRoot,
		WorkerID:     *workerID,
		Handlers:     hc,
		HTTPClient:   &http.Client{},
		StaleClaim:   *staleClaim,
		PollInterval: *pollInterval,
	}

	// Graceful shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Printf("received %v, shutting down...", sig)
		cancel()
	}()

	// Periodic cleanup in background.
	go func() {
		ticker := time.NewTicker(*cleanupInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				n, err := worker.RunCleanup(*queueRoot, *retentionDone, *retentionDead)
				if err != nil {
					log.Printf("cleanup error: %v", err)
				} else if *verbose {
					log.Printf("cleanup: deleted %d files", n)
				}
			}
		}
	}()

	log.Printf("toolkit-worker %s starting (queue: %s)", *workerID, *queueRoot)
	if err := d.Run(ctx); err != nil && err != context.Canceled {
		log.Fatalf("worker error: %v", err)
	}
	log.Println("toolkit-worker stopped")
}
