package main

import (
	"log"
	"time"

	"github.com/bsm/redis-lock"
	"gopkg.in/redis.v2"
)

func main() {
	// Connect to Redis
	client := redis.NewClient(&redis.Options{Network: "tcp", Addr: "127.0.0.1:6379"})
	defer client.Close()

	// Obtain a new lock with default settings
	lock, err := lock.ObtainLock(client, "lock.foo", nil)
	if err != nil {
		log.Fatalf("redis error: %s", err.Error())
	} else if lock == nil {
		log.Fatalln("could not obtain lock")
	}

	// Don't forget to unlock in the end
	defer lock.Unlock()

	// Run something
	log.Println("I have a lock!")
	time.Sleep(200 * time.Millisecond)

	// Renew your lock
	ok, err := lock.Lock()
	if err != nil {
		log.Fatalf("redis error: %s", err.Error())
	} else if !ok {
		log.Fatalln("could not renew lock")
	}

	// Run something else
	log.Println("I have renewed my lock!")
	time.Sleep(300 * time.Millisecond)

	log.Println("Done...")
}
