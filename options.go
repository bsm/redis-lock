package lock

import "time"

const (
	minWaitRetry   = 10 * time.Millisecond
	minLockTimeout = 5 * time.Second
)

// Options describe the options for the lock
type Options struct {
	// The maximum duration to lock a key for
	// Default: 5s
	LockTimeout time.Duration

	// The maximum amount of time you are willing to wait to obtain that lock
	// Default: 0 = do not wait
	WaitTimeout time.Duration

	// In case WaitTimeout is activated, this it the amount of time you are willing
	// to wait between retries.
	// Default: 100ms, must be at least 10ms
	WaitRetry time.Duration

	// In case RetriesCount is activated, this it the count of retries.
	// Default: 0
	RetriesCount int
}

func (o *Options) normalize() *Options {
	if o.LockTimeout < 1 {
		o.LockTimeout = minLockTimeout
	}
	if o.WaitRetry < minWaitRetry {
		o.WaitRetry = minWaitRetry
	}
	if o.RetriesCount < 0 {
		o.RetriesCount = 0
	}
	if o.WaitTimeout < 0 {
		o.WaitTimeout = 0
	}
	if o.RetriesCount > 0 && o.WaitTimeout <= 0 {
		o.WaitTimeout = o.WaitRetry * time.Duration(o.RetriesCount)
	}
	return o
}
