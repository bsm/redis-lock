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

	// Create a new locker with default settings
	locker := lock.New(client, "lock.foo", nil)

	// Try to obtain lock
	hasLock, err := locker.Lock()
	if err != nil {
		panic(err.Error())
	} else if !hasLock {
		fmt.Println("could not obtain lock!")
		return
	}

	// Don't forget to defer Unlock!
	defer locker.Unlock()
	fmt.Println("I have a lock!")

	// Sleep and check if still locked afterwards.
	time.Sleep(200 * time.Millisecond)
	if locker.IsLocked() {
		fmt.Println("My lock has expired!")
	}

	// Renew your lock
	renewed, err := locker.Lock()
	if err != nil {
		panic(err.Error())
	} else if !renewed {
		fmt.Println("could not renew lock!")
		return
	}
	fmt.Println("I have renewed my lock!")

	// Output:
	// I have a lock!
	// My lock has expired!
	// I have renewed my lock!
}
