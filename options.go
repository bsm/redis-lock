package lock

import "time"

// Options describe the options for the lock
type Options struct {
	// The maximum duration to lock a key for
	// Default: 5s
	LockTimeout time.Duration

	// The maximum amount of time you are willing to wait to obtain that lock.
	// Default: 0 = do not wait
	WaitTimeout time.Duration

	// WaitRetry is the amount of time you are willing to wait between retries.
	// Default: 100ms
	WaitRetry time.Duration
}

func (o *Options) normalize() *Options {
	if o.LockTimeout < 1 {
		o.LockTimeout = 5 * time.Second
	}
	if o.WaitTimeout < 0 {
		o.WaitTimeout = 0
	}
	if o.WaitRetry < 1 {
		o.WaitRetry = 100 * time.Millisecond
	}
	return o
}
