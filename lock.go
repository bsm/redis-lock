package lock

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"
)

type Lock struct {
	client Client
	key    string
	opts   *LockOptions

	token string
	mutex sync.Mutex
}

// ObtainLock is a shortcut for NewLock().Lock()
func ObtainLock(client Client, key string, opts *LockOptions) (*Lock, error) {
	lock := NewLock(client, key, opts)
	if ok, err := lock.Lock(); err != nil || !ok {
		return nil, err
	}
	return lock, nil
}

// NewLock creates a new distributed lock on key
func NewLock(client Client, key string, opts *LockOptions) *Lock {
	return &Lock{client: client, key: key, opts: opts.normalize()}
}

// Lock applies the lock, don't forget to defer the Release() function to release the lock after usage
func (l *Lock) Lock() (bool, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	// Create a random token
	token, err := randomToken()
	if err != nil {
		return false, err
	}

	// Calculate the timestamp we are willing to wait for
	stop := time.Now().Add(l.opts.WaitTimeout)
	for {
		// Try to obtain a lock
		ok, err := l.obtain(token)
		if err != nil {
			return false, err
		} else if ok {
			l.token = token
			return true, nil
		}

		if time.Now().Add(l.opts.WaitRetry).After(stop) {
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

	err := l.release()
	l.token = ""
	return err
}

// Helpers

func (l *Lock) obtain(token string) (bool, error) {
	str, err := l.client.SetNxPx(l.key, token, int64(l.opts.LockTimeout/time.Millisecond))
	return str == "OK", err
}

func (l *Lock) release() error {
	return l.client.Eval(luaScript, l.key, l.token)
}

func randomToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(buf), nil
}
