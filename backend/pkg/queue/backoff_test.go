package queue

import (
	"context"
	"testing"
	"time"
)

func TestBackoff_DoublesUpToMaxAndResets(t *testing.T) {
	b := NewBackoff(2*time.Second, 60*time.Second)
	want := []time.Duration{2, 4, 8, 16, 32, 60, 60, 60}
	for i, w := range want {
		if got := b.Next(); got != w*time.Second {
			t.Fatalf("step %d: got %v, want %v", i, got, w*time.Second)
		}
	}
	b.Reset()
	if got := b.Next(); got != 2*time.Second {
		t.Fatalf("after Reset: got %v, want 2s", got)
	}
}

func TestBackoff_MinAboveMaxIsCapped(t *testing.T) {
	b := NewBackoff(10*time.Second, 5*time.Second)
	if got := b.Next(); got != 5*time.Second {
		t.Fatalf("delay must never exceed Max, got %v", got)
	}
}

func TestSleep_ElapsesAndCancels(t *testing.T) {
	if !Sleep(context.Background(), time.Millisecond) {
		t.Fatal("an uncancelled sleep must report completion")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	if Sleep(ctx, time.Minute) {
		t.Fatal("a cancelled context must stop the sleep")
	}
	if time.Since(start) > time.Second {
		t.Fatal("cancellation must return promptly")
	}
}
