package lock

import (
	"math/rand"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/redis.v2"
)

const testRedisKey = "__bsm_redis_lock_unit_test__"

var _ = Describe("Lock", func() {
	var subject *Lock

	var newLock = func() *Lock {
		return NewLock(&redisWrapper{redisClient}, testRedisKey, &LockOptions{
			WaitTimeout: 100 * time.Millisecond,
			LockTimeout: time.Second,
		})
	}
	var process = func(l *Lock) int {
		time.Sleep(time.Duration(rand.Int63n(int64(10 * time.Millisecond))))
		ok, err := l.Lock()
		if err != nil {
			return 100
		} else if !ok {
			return 0
		}

		defer l.Unlock()
		time.Sleep(200 * time.Millisecond)
		return 1
	}

	BeforeEach(func() {
		subject = newLock()
	})

	It("should obtain through short-cut", func() {
		lock, err := ObtainLock(&redisWrapper{redisClient}, testRedisKey, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(lock).To(BeAssignableToTypeOf(subject))
	})

	It("should obtain fresh locks", func() {
		ok, err := subject.Lock()
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())

		val := redisClient.Get(testRedisKey).Val()
		Expect(val).To(HaveLen(24))

		ttl := redisClient.PTTL(testRedisKey).Val()
		Expect(ttl).To(BeNumerically("~", time.Second, 10*time.Millisecond))
	})

	It("should wait for expiring locks if WaitTimeout is set", func() {
		Expect(redisClient.Set(testRedisKey, "ABCD").Err()).NotTo(HaveOccurred())
		Expect(redisClient.PExpire(testRedisKey, 50*time.Millisecond).Err()).NotTo(HaveOccurred())

		ok, err := subject.Lock()
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())

		val := redisClient.Get(testRedisKey).Val()
		Expect(val).To(HaveLen(24))
		Expect(subject.token).To(Equal(val))

		ttl := redisClient.PTTL(testRedisKey).Val()
		Expect(ttl).To(BeNumerically("~", time.Second, 10*time.Millisecond))
	})

	It("should wait until WaitTimeout is reached, then give up", func() {
		Expect(redisClient.Set(testRedisKey, "ABCD").Err()).NotTo(HaveOccurred())
		Expect(redisClient.PExpire(testRedisKey, 150*time.Millisecond).Err()).NotTo(HaveOccurred())

		ok, err := subject.Lock()
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeFalse())
		Expect(subject.token).To(Equal(""))

		val := redisClient.Get(testRedisKey).Val()
		Expect(val).To(Equal("ABCD"))

		ttl := redisClient.PTTL(testRedisKey).Val()
		Expect(ttl).To(BeNumerically("~", 50*time.Millisecond, 10*time.Millisecond))
	})

	It("should not wait for expiring locks if WaitTimeout is not set", func() {
		Expect(redisClient.Set(testRedisKey, "ABCD").Err()).NotTo(HaveOccurred())
		Expect(redisClient.PExpire(testRedisKey, 150*time.Millisecond).Err()).NotTo(HaveOccurred())
		subject.opts.WaitTimeout = 0

		ok, err := subject.Lock()
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeFalse())

		ttl := redisClient.PTTL(testRedisKey).Val()
		Expect(ttl).To(BeNumerically("~", 150*time.Millisecond, 10*time.Millisecond))
	})

	It("should release own locks", func() {
		ok, err := subject.Lock()
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())

		Expect(subject.Unlock()).NotTo(HaveOccurred())
		Expect(subject.token).To(Equal(""))
		Expect(redisClient.Get(testRedisKey).Err()).To(Equal(redis.Nil))
	})

	It("should not release someone else's locks", func() {
		Expect(redisClient.Set(testRedisKey, "ABCD").Err()).NotTo(HaveOccurred())

		Expect(subject.Unlock()).NotTo(HaveOccurred())
		Expect(subject.token).To(Equal(""))
		Expect(redisClient.Get(testRedisKey).Val()).To(Equal("ABCD"))
	})

	It("should prevent multiple locks (fuzzing)", func() {
		locks := make([]*Lock, 1000)
		for i := 0; i < 1000; i++ {
			locks[i] = newLock()
		}

		res := int32(0)
		wg := new(sync.WaitGroup)
		for _, lock := range locks {
			wg.Add(1)
			go func(l *Lock) {
				defer wg.Done()
				atomic.AddInt32(&res, int32(process(l)))
			}(lock)
		}
		wg.Wait()
		Expect(res).To(Equal(int32(1)))
	})

})

/*************************************************************************
 * GINKGO TEST HOOK
 *************************************************************************/

func TestSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	AfterEach(func() {
		Expect(redisClient.Del(testRedisKey).Err()).NotTo(HaveOccurred())
	})
	RunSpecs(t, "redis-lock")
}

var redisClient *redis.Client

type redisWrapper struct{ *redis.Client }

func (c *redisWrapper) SetNxPx(key, value string, pttl int64) (string, error) {
	cmd := redis.NewStringCmd("set", key, value, "nx", "px", strconv.FormatInt(pttl, 10))
	c.Process(cmd)

	str, err := cmd.Result()
	if err == redis.Nil {
		err = nil
	}
	return str, err
}

func (c *redisWrapper) Eval(script, key, arg string) error {
	return c.Client.Eval(script, []string{key}, []string{arg}).Err()
}

var _ = BeforeSuite(func() {
	redisClient = redis.NewClient(&redis.Options{Network: "tcp", Addr: "127.0.0.1:6379", DB: 9})
	Expect(redisClient.Ping().Err()).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	redisClient.Close()
})
