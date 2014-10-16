package main

import (
	"log"
	"strconv"
	"time"

	"github.com/bsm/redis-lock"
	"gopkg.in/redis.v2"
)

type RedisWrapper struct{ *redis.Client }

func (c *RedisWrapper) SetNxPx(key, value string, pttl int64) (string, error) {
	cmd := redis.NewStringCmd("set", key, value, "nx", "px", strconv.FormatInt(pttl, 10))
	c.Process(cmd)

	str, err := cmd.Result()
	if err == redis.Nil {
		err = nil
	}
	return str, err
}

func (c *RedisWrapper) Eval(script, key, arg string) error {
	return c.Client.Eval(script, []string{key}, []string{arg}).Err()
}

func main() {
	// Connect to Redis
	client := redis.NewClient(&redis.Options{Network: "tcp", Addr: "127.0.0.1:6379"})
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
