package lock

import (
	"crypto/rand"
	"encoding/base64"
	"strconv"
	"sync"
	"time"

	"gopkg.in/redis.v2"
)

type Lock struct {
	client *redis.Client
	key    string
	opts   *LockOptions

	token string
	mutex sync.Mutex
}

const luaScript = `
if redis.call("get", KEYS[1]) == ARGV[1] then
	return redis.call("del", KEYS[1])
else
  return 0
end
`

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
	ttl := strconv.FormatInt(int64(l.opts.LockTimeout/time.Millisecond), 10)
	cmd := redis.NewStringCmd("set", l.key, token, "nx", "px", ttl)
	l.client.Process(cmd)
	str, err := cmd.Result()
	if str == "OK" {
		return true, nil
	} else if err == redis.Nil {
		return false, nil
	}
	return false, err
}

func (l *Lock) release() error {
	return l.client.Eval(luaScript, []string{l.key}, []string{l.token}).Err()
}

func randomToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(buf), nil
}
