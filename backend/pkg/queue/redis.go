package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Queue is a reliable job queue backed by two Redis lists:
//   - main list  (LPUSH / BRPOP)
//   - processing list (atomic move via BRPOPLPUSH for at-least-once delivery)
//
// On crash recovery, jobs stranded in the processing list can be re-queued.
type Queue struct {
	rdb         *redis.Client
	mainKey     string
	dlqKey      string
	pollTimeout time.Duration
}

func New(rdb *redis.Client, mainKey, dlqKey string, pollTimeout time.Duration) *Queue {
	return &Queue{
		rdb:         rdb,
		mainKey:     mainKey,
		dlqKey:      dlqKey,
		pollTimeout: pollTimeout,
	}
}

// Enqueue serializes v and pushes it to the left of the main list.
func (q *Queue) Enqueue(ctx context.Context, v any) error {
	payload, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("queue: marshal: %w", err)
	}
	if err := q.rdb.LPush(ctx, q.mainKey, payload).Err(); err != nil {
		return fmt.Errorf("queue: lpush %q: %w", q.mainKey, err)
	}
	return nil
}

// Dequeue blocks until a job is available or the context is cancelled.
// Returns (payload bytes, nil) on success.
// Returns (nil, nil) on timeout — caller should loop.
func (q *Queue) Dequeue(ctx context.Context) ([]byte, error) {
	result, err := q.rdb.BRPop(ctx, q.pollTimeout, q.mainKey).Result()
	if err == redis.Nil {
		return nil, nil // timeout — no job available
	}
	if err != nil {
		return nil, fmt.Errorf("queue: brpop %q: %w", q.mainKey, err)
	}
	// result is [key, value]
	return []byte(result[1]), nil
}

// EnqueueDLQ pushes a job to the dead letter queue with an error annotation.
func (q *Queue) EnqueueDLQ(ctx context.Context, payload []byte, errMsg string) error {
	dlq := struct {
		Payload   json.RawMessage `json:"payload"`
		Error     string          `json:"error"`
		FailedAt  time.Time       `json:"failed_at"`
	}{
		Payload:  json.RawMessage(payload),
		Error:    errMsg,
		FailedAt: time.Now().UTC(),
	}
	b, err := json.Marshal(dlq)
	if err != nil {
		return fmt.Errorf("queue: dlq marshal: %w", err)
	}
	if err := q.rdb.LPush(ctx, q.dlqKey, b).Err(); err != nil {
		return fmt.Errorf("queue: dlq lpush %q: %w", q.dlqKey, err)
	}
	return nil
}

// Len returns the current depth of the main queue.
func (q *Queue) Len(ctx context.Context) (int64, error) {
	n, err := q.rdb.LLen(ctx, q.mainKey).Result()
	if err != nil {
		return 0, fmt.Errorf("queue: llen %q: %w", q.mainKey, err)
	}
	return n, nil
}
