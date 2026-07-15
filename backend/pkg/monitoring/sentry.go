// Package monitoring wires Sentry error reporting. Fail-silent by design:
// when SENTRY_DSN is unset (dev), every function here is a no-op so the
// local stack needs no external services - same pattern as FCM and TTS.
package monitoring

import (
	"time"

	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
)

// InitSentry configures the global Sentry client and tags every event with
// the process name ("api" or "worker"). Returns a flush func to defer in
// main. No-op flush when dsn is blank or init fails - never fatal.
func InitSentry(dsn, process string, log *zap.Logger) func() {
	if dsn == "" {
		return func() {}
	}
	if err := sentry.Init(sentry.ClientOptions{Dsn: dsn}); err != nil {
		log.Warn("sentry init failed - continuing without error reporting", zap.Error(err))
		return func() {}
	}
	sentry.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag("process", process)
	})
	log.Info("sentry error reporting enabled", zap.String("process", process))
	return func() { sentry.Flush(2 * time.Second) }
}

// CaptureErr reports an error with identifying tags (e.g. entry_id).
// No-op when Sentry is not initialized.
func CaptureErr(err error, tags map[string]string) {
	if err == nil {
		return
	}
	sentry.WithScope(func(scope *sentry.Scope) {
		for k, v := range tags {
			scope.SetTag(k, v)
		}
		sentry.CaptureException(err)
	})
}

// RecoverRepanic reports a panic to Sentry, flushes, then re-panics so the
// process still crashes and the host's restart policy applies unchanged.
// Use as: defer monitoring.RecoverRepanic()
func RecoverRepanic() {
	if r := recover(); r != nil {
		sentry.CurrentHub().Recover(r)
		sentry.Flush(2 * time.Second)
		panic(r)
	}
}
