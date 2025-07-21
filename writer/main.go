package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

var (
	group = "cg1"

	ctx = context.Background()
	rdb *redis.Client

	linesWritten = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "lines_written_total",
			Help: "Total number of lines written",
		},
	)
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	err := rdb.Ping(ctx).Err()
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Redis not available"))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func init() {
	prometheus.MustRegister(linesWritten)
}

func main() {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "redis:6379"
	}

	rdb = redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	defer rdb.Close()

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.HandleFunc("/healthz", healthHandler)
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()

	consumerName, err := os.Hostname()
	if err != nil {
		consumerName = fmt.Sprintf("consumer-%d", time.Now().Unix())
	}
	rdb.XGroupCreateMkStream(ctx, "line_stream", group, "$")

	outFile, err := os.Create("/data/output.txt")
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer outFile.Close()
	writer := bufio.NewWriter(outFile)

	for {
		entries, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    group,
			Consumer: consumerName,
			Streams:  []string{"line_stream", ">"},
			Count:    1,
			Block:    5 * time.Second,
		}).Result()
		if err == redis.Nil || len(entries) == 0 {
			continue
		} else if err != nil {
			log.Printf("Error reading from stream: %v", err)
			continue
		}

		for _, stream := range entries {
			for _, msg := range stream.Messages {
				line, ok := msg.Values["line"].(string)
				if !ok {
					log.Println("Invalid message format")
					continue
				}
				_, err = writer.WriteString(line + "\n")
				if err != nil {
					log.Printf("Failed to write line: %v", err)
					continue
				}
				writer.Flush()
				linesWritten.Inc()
				rdb.XAck(ctx, "line_stream", group, msg.ID)
				fmt.Printf("Consumed and acknowledged: %s\n", line)
			}
		}
	}
}
