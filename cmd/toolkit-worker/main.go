// toolkit-worker is the file-queue worker daemon (internal-network side).
// It scans the shared queue for pending requests and dispatches them to
// configured REST API handlers.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
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
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	_ = stdout
	fs := flag.NewFlagSet("toolkit-worker", flag.ContinueOnError)
	fs.SetOutput(stderr)
	queueRoot := fs.String("queue-root", "", "shared drive queue root (required)")
	handlerConfig := fs.String("handler-config", "", "handler registry config file (required)")
	workerID := fs.String("worker-id", "", "unique worker ID (default: hostname-pid)")
	pollInterval := fs.Duration("poll-interval", time.Second, "scan interval")
	staleClaim := fs.Duration("stale-claim", 5*time.Minute, "stale claim threshold")
	retentionDone := fs.Duration("retention-done", 24*time.Hour, "completed request retention")
	retentionDead := fs.Duration("retention-dead", 72*time.Hour, "dead-letter retention")
	cleanupInterval := fs.Duration("cleanup-interval", time.Hour, "cleanup run interval")
	verbose := fs.Bool("verbose", false, "enable debug logging")
	once := fs.Bool("once", false, "validate configuration and run one dispatcher scan")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	if *queueRoot == "" || *handlerConfig == "" {
		fmt.Fprintln(stderr, "--queue-root and --handler-config are required")
		return 1
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
		fmt.Fprintf(stderr, "loading handler config: %v\n", err)
		return 1
	}

	// Ensure queue directories.
	if err := filequeue.EnsureQueueDirs(*queueRoot); err != nil {
		fmt.Fprintf(stderr, "ensuring queue dirs: %v\n", err)
		return 1
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
	if *once {
		d.RunOnce(context.Background())
		return 0
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
		fmt.Fprintf(stderr, "worker error: %v\n", err)
		return 1
	}
	log.Println("toolkit-worker stopped")
	return 0
}
