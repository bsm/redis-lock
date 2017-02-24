package lock

import (
	"crypto/rand"
	"encoding/base64"
	"strconv"
	"sync"
	"time"

	"github.com/go-redis/redis"
)

const luaRefresh = `if redis.call("get", KEYS[1]) == ARGV[1] then return redis.call("pexpire", KEYS[1], ARGV[2]) else return 0 end`
const luaRelease = `if redis.call("get", KEYS[1]) == ARGV[1] then return redis.call("del", KEYS[1]) else return 0 end`

// RedisClient is a minimal client interface, supported by gopkg.in/redis.v3
type RedisClient interface {
	SetNX(key string, value interface{}, expiration time.Duration) *redis.BoolCmd
	Eval(script string, keys []string, args ...interface{}) *redis.Cmd
}

type Lock struct {
	client RedisClient
	key    string
	opts   LockOptions

	token string
	mutex sync.Mutex
}

// ObtainLock is a shortcut for NewLock().Lock()
func ObtainLock(client RedisClient, key string, opts *LockOptions) (*Lock, error) {
	lock := NewLock(client, key, opts)
	if ok, err := lock.Lock(); err != nil || !ok {
		return nil, err
	}
	return lock, nil
}

// NewLock creates a new distributed lock on key
func NewLock(client RedisClient, key string, opts *LockOptions) *Lock {
	if opts == nil {
		opts = new(LockOptions)
	}
	return &Lock{client: client, key: key, opts: *opts.normalize()}
}

// IsLocked returns true if a lock is acquired
func (l *Lock) IsLocked() bool {
	l.mutex.Lock()
	locked := l.token != ""
	l.mutex.Unlock()

	return locked
}

// Lock applies the lock, don't forget to defer the Unlock() function to release the lock after usage
func (l *Lock) Lock() (bool, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.token != "" {
		return l.refresh()
	}
	return l.create()
}

// Unlock releases the lock
func (l *Lock) Unlock() error {
	l.mutex.Lock()
	err := l.release()
	l.mutex.Unlock()

	return err
}

// Helpers

func (l *Lock) create() (bool, error) {
	l.reset()

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

func (l *Lock) refresh() (bool, error) {
	ttl := strconv.FormatInt(int64(l.opts.LockTimeout/time.Millisecond), 10)
	status, err := l.client.Eval(luaRefresh, []string{l.key}, l.token, ttl).Result()
	if err != nil {
		return false, err
	} else if status == int64(1) {
		return true, nil
	}
	return l.create()
}

func (l *Lock) obtain(token string) (bool, error) {
	ok, err := l.client.SetNX(l.key, token, l.opts.LockTimeout).Result()
	if err == redis.Nil {
		err = nil
	}
	return ok, err
}

func (l *Lock) release() error {
	defer l.reset()

	err := l.client.Eval(luaRelease, []string{l.key}, l.token).Err()
	if err == redis.Nil {
		err = nil
	}
	return err
}

func (l *Lock) reset() {
	l.token = ""
}

func randomToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(buf), nil
}
