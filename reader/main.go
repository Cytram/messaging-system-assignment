package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/hpcloud/tail"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

var (
	ctx = context.Background()
	rdb *redis.Client

	linesProduced = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "lines_read_total",
			Help: "Total number of lines read",
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
	prometheus.MustRegister(linesProduced)
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

	t, err := tail.TailFile("/data/input.txt", tail.Config{
		Follow:   true,
		ReOpen:   true,
		Location: &tail.SeekInfo{Offset: 0, Whence: 2}, // start at end of file
	})
	if err != nil {
		log.Fatalf("Failed to tail file: %v", err)
	}

	for line := range t.Lines {
		if line.Err != nil {
			log.Printf("Tail error: %v", line.Err)
			continue
		}

		_, err := rdb.XAdd(ctx, &redis.XAddArgs{
			Stream: "line_stream",
			Values: map[string]interface{}{"line": line.Text},
		}).Result()
		if err != nil {
			log.Printf("Error adding to stream: %v", err)
		} else {
			fmt.Printf("Produced: %s\n", line.Text)
			linesProduced.Inc()
		}
	}
	fmt.Printf("shutdown")
}
