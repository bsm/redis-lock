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

	// Obtain a new locker with default settings
	locker, err := lock.ObtainLock(client, "locker.foo", nil)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
		return
	} else if locker == nil {
		fmt.Println("ERROR: could not obtain locker")
		return
	}

	// Don't forget to unlock in the end
	defer locker.Unlock()

	// Run something
	fmt.Println("I have a locker!")
	time.Sleep(200 * time.Millisecond)

	// Renew your locker
	ok, err := locker.Lock()
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
		return
	} else if !ok {
		fmt.Println("ERROR: could not renew locker")
		return
	}
	fmt.Println("I have renewed my locker!")

	// Output:
	// I have a locker!
	// I have renewed my locker!
}
