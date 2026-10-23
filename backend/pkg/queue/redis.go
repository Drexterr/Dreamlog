package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Queue is a Redis list job queue (LPUSH / BRPOP).
//
// BRPOP removes a job before it is processed, so a job in flight when a worker
// crashes is lost from Redis. Delivery is made at-least-once by the database
// instead: workers.EntryReconciler requeues any entry left in pending /
// processing past a staleness threshold, and the worker drops duplicate jobs
// for entries another worker already holds.
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
// Returns (nil, nil) on timeout - caller should loop.
func (q *Queue) Dequeue(ctx context.Context) ([]byte, error) {
	result, err := q.rdb.BRPop(ctx, q.pollTimeout, q.mainKey).Result()
	if err == redis.Nil {
		return nil, nil // timeout - no job available
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

// DLQRange returns raw dead-letter elements, newest first (LPUSH order), for
// the inclusive index range [start, stop]; stop = -1 means to the end.
func (q *Queue) DLQRange(ctx context.Context, start, stop int64) ([]string, error) {
	items, err := q.rdb.LRange(ctx, q.dlqKey, start, stop).Result()
	if err != nil {
		return nil, fmt.Errorf("queue: dlq lrange %q: %w", q.dlqKey, err)
	}
	return items, nil
}

// DLQLen returns the number of dead-lettered jobs.
func (q *Queue) DLQLen(ctx context.Context) (int64, error) {
	n, err := q.rdb.LLen(ctx, q.dlqKey).Result()
	if err != nil {
		return 0, fmt.Errorf("queue: dlq llen %q: %w", q.dlqKey, err)
	}
	return n, nil
}

// DLQRemove removes one occurrence of the exact raw element. It reports
// whether this call removed it: LREM is atomic, so of two concurrent callers
// only one gets true - the claim that stops a double requeue.
func (q *Queue) DLQRemove(ctx context.Context, raw string) (bool, error) {
	n, err := q.rdb.LRem(ctx, q.dlqKey, 1, raw).Result()
	if err != nil {
		return false, fmt.Errorf("queue: dlq lrem %q: %w", q.dlqKey, err)
	}
	return n > 0, nil
}

// DLQRestore puts a raw element back on the dead-letter list (after a claimed
// requeue could not be completed).
func (q *Queue) DLQRestore(ctx context.Context, raw string) error {
	if err := q.rdb.LPush(ctx, q.dlqKey, raw).Err(); err != nil {
		return fmt.Errorf("queue: dlq restore %q: %w", q.dlqKey, err)
	}
	return nil
}
