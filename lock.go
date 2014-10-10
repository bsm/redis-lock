package lock

import (
	"strconv"
	"sync"
	"time"

	"gopkg.in/redis.v2"
)

const minWaitRetry = 10 * time.Millisecond

type LockOptions struct {
	// The maximum duration to lock a key for
	// Default: 10s
	LockTimeout time.Duration

	// The maximum amount of time you are willing to wait to obtain that lock
	// Default: 10s
	WaitTimeout time.Duration

	// The amount of time you are willing to wait between retries, must be at least 10ms
	// Default: 100ms
	WaitRetry time.Duration
}

func (o *LockOptions) normalize() *LockOptions {
	if o == nil {
		o = new(LockOptions)
	}
	if o.LockTimeout < 1 {
		o.LockTimeout = 10 * time.Second
	}
	if o.WaitTimeout < 1 {
		o.WaitTimeout = 10 * time.Second
	}
	if o.WaitRetry < minWaitRetry {
		o.WaitRetry = minWaitRetry
	}
	return o
}

type Lock struct {
	client *redis.Client
	key    string
	opts   *LockOptions

	deadline int64
	mutex    sync.Mutex
}

// ObtainLock is a shortcut for NewLock().Lock()
func ObtainLock(client *redis.Client, key string, opts *LockOptions) (*Lock, error) {
	lock := NewLock(client, key, opts)
	if ok, err := lock.Lock(); err != nil || !ok {
		return nil, err
	}
	return lock, nil
}

// NewLock creates a new distributed lock on key
func NewLock(client *redis.Client, key string, opts *LockOptions) *Lock {
	return &Lock{client: client, key: key, opts: opts.normalize()}
}

// Lock applies the lock, don't forget to defer the Release() function to release the lock after usage
func (l *Lock) Lock() (bool, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	max := time.Now().Add(l.opts.WaitTimeout)
	for max.After(time.Now()) {
		l.deadline = time.Now().Add(l.opts.LockTimeout).UnixNano() + 1

		// Try to obtain a lock via SETNX
		ok, err := l.setnx()
		if err != nil {
			return false, err
		} else if ok {
			return true, nil
		}

		// Check if lock is held by someone else
		held, err := l.get()
		if err != nil {
			return false, err
		}

		// If held lock is expired, try to obtain expired lock via GETSET
		if held.Before(time.Now()) {
			prev, err := l.getset()
			if err != nil {
				return false, err
			} else if prev.Before(time.Now()) {
				return true, nil
			}
		}

		time.Sleep(l.opts.WaitRetry)
	}
	return false, nil
}

// Unlock releases the lock
func (l *Lock) Unlock() error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.deadline > time.Now().UnixNano() {
		return l.del()
	}
	l.deadline = 0
	return nil
}

// Helpers

func nanoTime(u int64) time.Time {
	s := int64(time.Second)
	return time.Unix(u/s, u%s)
}

func toTime(str string, err error) (time.Time, error) {
	if err != nil {
		return time.Time{}, err
	}
	num, _ := strconv.ParseInt(str, 10, 64)
	return nanoTime(num), nil
}

func (l *Lock) setnx() (bool, error) {
	return l.client.SetNX(l.key, strconv.FormatInt(l.deadline, 10)).Result()
}

func (l *Lock) del() error {
	return l.client.Del(l.key).Err()
}

func (l *Lock) get() (time.Time, error) {
	str, err := l.client.Get(l.key).Result()
	return toTime(str, err)
}

func (l *Lock) getset() (time.Time, error) {
	str, err := l.client.GetSet(l.key, strconv.FormatInt(l.deadline, 10)).Result()
	return toTime(str, err)
}
