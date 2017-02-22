package lock_test

import (
	"fmt"
	"time"

	"github.com/bsm/redis-lock"
	"github.com/go-redis/redis"
)

func Example() {
	// Connect to Redis
	client := redis.NewClient(&redis.Options{
		Network: "tcp",
		Addr:    "127.0.0.1:6379",
	})
	defer client.Close()

	// Obtain a new lock with default settings
	lock, err := lock.ObtainLock(client, "lock.foo", nil)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
		return
	} else if lock == nil {
		fmt.Println("ERROR: could not obtain lock")
		return
	}

	// Don't forget to unlock in the end
	defer lock.Unlock()

	// Run something
	fmt.Println("I have a lock!")
	time.Sleep(200 * time.Millisecond)

	// Renew your lock
	ok, err := lock.Lock()
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
		return
	} else if !ok {
		fmt.Println("ERROR: could not renew lock")
		return
	}
	fmt.Println("I have renewed my lock!")

	// Output:
	// I have a lock!
	// I have renewed my lock!
}
