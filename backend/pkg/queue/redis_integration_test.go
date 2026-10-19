//go:build integration

package queue

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func TestIntegration_RedisQueueFIFOAndDeadLetters(t *testing.T) {
	addr := os.Getenv("ODE_TEST_REDIS_ADDR")
	if addr != "127.0.0.1:63791" {
		t.Fatal("ODE_TEST_REDIS_ADDR must target isolated regression Redis at 127.0.0.1:63791")
	}
	ctx := context.Background()
	client := redis.NewClient(&redis.Options{Addr: addr})
	defer client.Close()
	key := "ode:qa:" + uuid.NewString()
	dlq := key + ":dlq"
	defer client.Del(ctx, key, dlq)
	q := New(client, key, dlq, time.Second)
	for _, id := range []string{"first", "second"} {
		if err := q.Enqueue(ctx, map[string]string{"id": id}); err != nil {
			t.Fatal(err)
		}
	}
	if n, err := q.Len(ctx); err != nil || n != 2 {
		t.Fatalf("depth=%d err=%v", n, err)
	}
	for _, want := range []string{"first", "second"} {
		payload, err := q.Dequeue(ctx)
		if err != nil {
			t.Fatal(err)
		}
		var value map[string]string
		if err = json.Unmarshal(payload, &value); err != nil || value["id"] != want {
			t.Fatalf("FIFO %s err=%v", payload, err)
		}
	}
	if payload, err := q.Dequeue(ctx); err != nil || payload != nil {
		t.Fatalf("empty timeout payload=%s err=%v", payload, err)
	}
	if err := q.Enqueue(ctx, make(chan int)); err == nil {
		t.Fatal("unsupported payload accepted")
	}
	if err := q.EnqueueDLQ(ctx, []byte(`{"id":"failed"}`), "provider timeout"); err != nil {
		t.Fatal(err)
	}
	raw, err := client.LIndex(ctx, dlq, 0).Result()
	if err != nil {
		t.Fatal(err)
	}
	var record struct {
		Payload  map[string]string `json:"payload"`
		Error    string            `json:"error"`
		FailedAt time.Time         `json:"failed_at"`
	}
	if err = json.Unmarshal([]byte(raw), &record); err != nil || record.Payload["id"] != "failed" || record.Error != "provider timeout" || record.FailedAt.IsZero() {
		t.Fatalf("DLQ invalid %s err=%v", raw, err)
	}
}
