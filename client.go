package lock

const luaScript = `if redis.call("get", KEYS[1]) == ARGV[1] then return redis.call("del", KEYS[1]) else return 0 end`

// Client is an abstraction layer to support a variety of redis clients. A valid client must implement two functions,
// here is an example for a gopkg.in/redis.v2 based wrapper implementation:
//
//  import "gopkg.in/redis.v2"
//
//  type RedisWrapper struct{ *redis.Client }
//
//  func (c *RedisWrapper) SetNxPx(key, value string, pttl int64) (string, error) {
//    cmd := redis.NewStringCmd("set", key, value, "nx", "px", strconv.FormatInt(pttl, 10))
//    c.Process(cmd)
//
//    str, err := cmd.Result()
//    if err == redis.Nil {
//      err = nil
//    }
//    return str, err
//  }
//
//  func (c *RedisWrapper) Eval(script, key, arg string) error {
//    return c.Client.Eval(script, []string{key}, []string{arg}).Err()
//  }
//
type Client interface {
	// SetNxPx must execute a `SET key value NX PX pttl` command and return the result string and error (if occurs)
	SetNxPx(key string, value string, pttl int64) (string, error)

	// Eval must execute a `EVAL script 1 key arg` command and return the error (if one occurs)
	Eval(script string, key string, arg string) error
}
