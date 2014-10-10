# redis-lock [![Build Status](https://travis-ci.org/bsm/redis-lock.png?branch=master)](https://travis-ci.org/bsm/redis-lock)

Distributed locking implementation using [Redis](http://redis.io). Uses [Go-Redis](https://github.com/go-redis/redis) client.

## Example

```go

import (
    "fmt"

    "github.com/bsm/redis-lock"
    "gopkg.in/redis.v2"
)

func main() {

    // Connect to Redis
    client := redis.NewClient(&redis.Options{Network: "tcp", Addr: "127.0.0.1:6379"})
    defer client.Close()

    // Create a new lock with default settings
    lock, err := lock.ObtainLock(client, "lock.foo", nil)
    if err != nil {
        panic(err.Error())
    } else if lock == nil {
        panic("could not obtain lock!")
    }
    defer lock.Unlock()

    // Perform something unique
    time.Sleep(500*time.Millisecond)
    fmt.Println("done!")

}

```

## Testing

Simply run:

    make

## Licence

    Copyright (c) 2014 Black Square Media

    Permission is hereby granted, free of charge, to any person obtaining
    a copy of this software and associated documentation files (the
    "Software"), to deal in the Software without restriction, including
    without limitation the rights to use, copy, modify, merge, publish,
    distribute, sublicense, and/or sell copies of the Software, and to
    permit persons to whom the Software is furnished to do so, subject to
    the following conditions:

    The above copyright notice and this permission notice shall be
    included in all copies or substantial portions of the Software.

    THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
    EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
    MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
    NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
    LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
    OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
    WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
