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
		return NewLock(redisClient, testRedisKey, &LockOptions{
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
		lock, err := ObtainLock(redisClient, testRedisKey, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(lock).To(BeAssignableToTypeOf(subject))
	})

	It("should obtain fresh locks", func() {
		ok, err := subject.Lock()
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())

		val, err := redisClient.Get(testRedisKey).Result()
		Expect(err).NotTo(HaveOccurred())
		exp, _ := toTime(val, nil)
		Expect(exp).To(BeTemporally(">", time.Now()))
	})

	It("should obtain expired locks", func() {
		err := redisClient.Set(testRedisKey, "1313131313000000000").Err()
		Expect(err).NotTo(HaveOccurred())

		ok, err := subject.Lock()
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())

		val, err := redisClient.Get(testRedisKey).Result()
		Expect(err).NotTo(HaveOccurred())
		exp, _ := toTime(val, nil)
		Expect(exp).To(BeTemporally(">", time.Now()))
	})

	It("should wait for expiring locks if WaitTimeout is set", func() {
		future := time.Now().Add(50 * time.Millisecond).UnixNano()
		err := redisClient.Set(testRedisKey, strconv.FormatInt(future, 10)).Err()
		Expect(err).NotTo(HaveOccurred())

		ok, err := subject.Lock()
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())
		Expect(future).To(BeNumerically("<", time.Now().UnixNano()))

		val, err := redisClient.Get(testRedisKey).Result()
		Expect(err).NotTo(HaveOccurred())
		exp, _ := toTime(val, nil)
		Expect(exp).To(BeTemporally(">", time.Now()))
	})

	It("should wait until WaitTimeout is reached", func() {
		future := time.Now().Add(150 * time.Millisecond).UnixNano()
		err := redisClient.Set(testRedisKey, strconv.FormatInt(future, 10)).Err()
		Expect(err).NotTo(HaveOccurred())

		ok, err := subject.Lock()
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeFalse())
	})

	It("should not wait for expiring locks if WaitTimeout is not set", func() {
		subject.opts.WaitTimeout = 0
		future := time.Now().Add(50 * time.Millisecond).UnixNano()
		err := redisClient.Set(testRedisKey, strconv.FormatInt(future, 10)).Err()
		Expect(err).NotTo(HaveOccurred())

		ok, err := subject.Lock()
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeFalse())
		Expect(future).To(BeNumerically(">", time.Now().UnixNano()))
	})

	It("should release held locks", func() {
		ok, err := subject.Lock()
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())

		Expect(subject.Unlock()).NotTo(HaveOccurred())
		Expect(redisClient.Get(testRedisKey).Err()).To(Equal(redis.Nil))
	})

	It("should prevent multiple locks (fuzzing)", func() {
		locks := make([]*Lock, 40)
		for i := 0; i < 40; i++ {
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

var _ = BeforeSuite(func() {
	redisClient = redis.NewClient(&redis.Options{Network: "tcp", Addr: "127.0.0.1:6379", DB: 9})
	Expect(redisClient.Ping().Err()).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	redisClient.Close()
})
