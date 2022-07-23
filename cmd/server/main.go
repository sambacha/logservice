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

	logservice "github.com/cesko-digital/vercel-logging"
)

var (
	indexName     string
	flushBytes    int
	flushInterval time.Duration

	listenAddr = "0.0.0.0"
	listenPort = "8080"
)

func init() {
	flag.StringVar(&indexName, "index", "logservice-logs", "Elasticsearch index name")
	flag.IntVar(&flushBytes, "flushBytes", 1e+6, "Flush threshold in bytes")
	flag.DurationVar(&flushInterval, "flushInterval", 5*time.Second, "Flush threshold in duration")
	flag.Parse()
}

func main() {
	log.SetFlags(0)

	if envAddress := os.Getenv("LISTEN_ADDR"); envAddress != "" {
		listenAddr = envAddress
	}

	if envPort := os.Getenv("PORT"); envPort != "" {
		listenPort = envPort
	}

	log.Printf("Server starting at %s:%s...", listenAddr, listenPort)

	svc, err := logservice.New(logservice.Config{
		IndexName:     indexName,
		FlushInterval: flushInterval,
		FlushBytes:    flushBytes,

		ElasticsearchURL:    os.Getenv("ELASTICSEARCH_URL"),
		ElasticsearchAPIKey: os.Getenv("ELASTICSEARCH_API_KEY"),
	})
	if err != nil {
		log.Fatalf("Error creating server: %s", err)
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", listenAddr, listenPort),
		Handler: svc,
	}

	flushCompleted := make(chan struct{})
	srv.RegisterOnShutdown(func() { shutdownServer(svc, flushCompleted) })

	// See https://cloud.google.com/run/docs/samples/cloudrun-sigterm-handler
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
		<-sig

		log.Printf("Server shutting down...")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Fatalf("Error shutting down server: %s", err)
		}
	}()

	if err = srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Error starting server: %s", err)
	}

	<-flushCompleted
}

// shutdownServer calls Writer.Flush() and prints the indexer's statistics.
//
func shutdownServer(svc *logservice.Service, flushCompleted chan struct{}) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	svc.Flush(ctx)

	indexerStats := svc.Stats()
	log.Printf(
		"Indexed [%d] documents from [%d] added with [%d] errors",
		indexerStats.NumFlushed,
		indexerStats.NumAdded,
		indexerStats.NumFailed)

	close(flushCompleted)
}
