package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

func main() {
	numMessages := 100000
	numWorkers := 100
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "redis:6379"
	}

	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	defer rdb.Close()

	start := time.Now()
	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for i := 0; i < numWorkers; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numMessages/numWorkers; j++ {
				line := fmt.Sprintf("worker-%d-message-%d-%d", id, j, rand.Intn(1000000))
				_, err := rdb.XAdd(ctx, &redis.XAddArgs{
					Stream: "line_stream",
					Values: map[string]interface{}{"line": line},
				}).Result()
				if err != nil {
					log.Printf("Worker %d failed to push: %v", id, err)
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)
	fmt.Printf("Inserted %d messages with %d workers in %v\n", numMessages, numWorkers, duration)
}
