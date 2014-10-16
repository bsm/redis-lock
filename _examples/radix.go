package main

import (
	"log"
	"time"

	"github.com/bsm/redis-lock"
	"github.com/fzzy/radix/redis"
)

type RedisWrapper struct{ *redis.Client }

func (c *RedisWrapper) SetNxPx(key, value string, pttl int64) (string, error) {
	repl := c.Cmd("set", key, value, "nx", "px", pttl)
	if repl.Type == redis.StatusReply {
		return repl.Str()
	}
	return "", repl.Err
}

func (c *RedisWrapper) Eval(script, key, arg string) error {
	return c.Client.Cmd("eval", script, 1, key, arg).Err
}

func main() {
	// Connect to Redis
	client, err := redis.DialTimeout("tcp", "127.0.0.1:6379", 10*time.Second)
	if err != nil {
		log.Fatalf("cannot connect to redis: %s", err.Error())
	}
	defer client.Close()

	// Obtain a new lock with default settings
	lock, err := lock.ObtainLock(&RedisWrapper{client}, "lock.foo", nil)
	if err != nil {
		log.Fatalf("redis error: %s", err.Error())
	} else if lock == nil {
		log.Fatalf("could not obtain lock: %s", err.Error())
	}

	// Don't forget to unlock in the end
	defer lock.Unlock()

	// Run something uniquely
	log.Println("I have a lock!")
	time.Sleep(500 * time.Millisecond)
	log.Println("Done...")
}
