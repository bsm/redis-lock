package lock

import (
	"strconv"
	"sync"
	"time"

	"gopkg.in/redis.v2"
)

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
	for {
		deadline := time.Now().Add(l.opts.LockTimeout).UnixNano() + 1

		// Try to obtain a lock via SETNX
		ok, err := l.setnx(deadline)
		if err != nil {
			return false, err
		} else if ok {
			l.deadline = deadline
			return true, nil
		}

		// Check if lock is held by someone else
		held, err := l.get()
		if err != nil {
			return false, err
		}

		// If held lock is expired, try to obtain expired lock via GETSET
		if held.Before(time.Now()) {
			prev, err := l.getset(deadline)
			if err != nil {
				return false, err
			} else if prev.Before(time.Now()) {
				l.deadline = deadline
				return true, nil
			}
		}

		if time.Now().Add(l.opts.WaitRetry).After(max) {
			break
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

func (l *Lock) setnx(deadline int64) (bool, error) {
	return l.client.SetNX(l.key, strconv.FormatInt(deadline, 10)).Result()
}

func (l *Lock) del() error {
	return l.client.Del(l.key).Err()
}

func (l *Lock) get() (time.Time, error) {
	str, err := l.client.Get(l.key).Result()
	return toTime(str, err)
}

func (l *Lock) getset(deadline int64) (time.Time, error) {
	str, err := l.client.GetSet(l.key, strconv.FormatInt(deadline, 10)).Result()
	return toTime(str, err)
}
