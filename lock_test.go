package lock

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-redis/redis"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const testRedisKey = "__bsm_redis_lock_unit_test__"

var _ = Describe("Locker", func() {
	var subject *Locker

	var newLock = func() *Locker {
		return New(redisClient, testRedisKey, &Options{
			RetryCount:  4,
			RetryDelay:  25 * time.Millisecond,
			LockTimeout: time.Second,
		})
	}

	var getTTL = func() (time.Duration, error) {
		return redisClient.PTTL(testRedisKey).Result()
	}

	BeforeEach(func() {
		subject = newLock()
		Expect(subject.IsLocked()).To(BeFalse())
	})

	AfterEach(func() {
		Expect(redisClient.Del(testRedisKey).Err()).NotTo(HaveOccurred())
	})

	It("should normalize options", func() {
		locker := New(redisClient, testRedisKey, &Options{
			LockTimeout: -1,
			RetryCount:  -1,
			RetryDelay:  -1,
		})
		Expect(locker.opts).To(Equal(Options{
			LockTimeout: 5 * time.Second,
			RetryCount:  0,
			RetryDelay:  100 * time.Millisecond,
		}))
	})

	It("should fail obtain with error", func() {
		locker := newLock()
		locker.Lock()
		defer locker.Unlock()

		_, err := Obtain(redisClient, testRedisKey, nil)
		Expect(err).To(Equal(ErrLockNotObtained))
	})

	It("should obtain through short-cut", func() {
		Expect(Obtain(redisClient, testRedisKey, nil)).To(BeAssignableToTypeOf(subject))
	})

	It("should obtain fresh locks", func() {
		Expect(subject.Lock()).To(BeTrue())
		Expect(subject.IsLocked()).To(BeTrue())

		Expect(redisClient.Get(testRedisKey).Result()).To(HaveLen(24))
		Expect(getTTL()).To(BeNumerically("~", time.Second, 10*time.Millisecond))
	})

	It("should retry if enabled", func() {
		Expect(redisClient.Set(testRedisKey, "ABCD", 0).Err()).NotTo(HaveOccurred())
		Expect(redisClient.PExpire(testRedisKey, 30*time.Millisecond).Err()).NotTo(HaveOccurred())

		Expect(subject.Lock()).To(BeTrue())
		Expect(subject.IsLocked()).To(BeTrue())

		Expect(redisClient.Get(testRedisKey).Result()).To(Equal(subject.token))
		Expect(subject.token).To(HaveLen(24))
		Expect(getTTL()).To(BeNumerically("~", time.Second, 10*time.Millisecond))
	})

	It("should not retry if not enabled", func() {
		Expect(redisClient.Set(testRedisKey, "ABCD", 0).Err()).NotTo(HaveOccurred())
		Expect(redisClient.PExpire(testRedisKey, 150*time.Millisecond).Err()).NotTo(HaveOccurred())
		subject.opts.RetryCount = 0

		Expect(subject.Lock()).To(BeFalse())
		Expect(subject.IsLocked()).To(BeFalse())
		Expect(getTTL()).To(BeNumerically("~", 150*time.Millisecond, 10*time.Millisecond))
	})

	It("should give up when retry count reached", func() {
		Expect(redisClient.Set(testRedisKey, "ABCD", 0).Err()).NotTo(HaveOccurred())
		Expect(redisClient.PExpire(testRedisKey, 150*time.Millisecond).Err()).NotTo(HaveOccurred())

		Expect(subject.Lock()).To(BeFalse())
		Expect(subject.IsLocked()).To(BeFalse())
		Expect(subject.token).To(Equal(""))

		Expect(redisClient.Get(testRedisKey).Result()).To(Equal("ABCD"))
		Expect(getTTL()).To(BeNumerically("~", 45*time.Millisecond, 20*time.Millisecond))
	})

	It("should release own locks", func() {
		Expect(subject.Lock()).To(BeTrue())
		Expect(subject.IsLocked()).To(BeTrue())

		Expect(subject.Unlock()).NotTo(HaveOccurred())
		Expect(subject.token).To(Equal(""))
		Expect(subject.IsLocked()).To(BeFalse())
		Expect(redisClient.Get(testRedisKey).Err()).To(Equal(redis.Nil))
	})

	It("should not release someone else's locks", func() {
		Expect(redisClient.Set(testRedisKey, "ABCD", 0).Err()).NotTo(HaveOccurred())
		Expect(subject.IsLocked()).To(BeFalse())

		Expect(subject.Unlock()).NotTo(HaveOccurred())
		Expect(subject.token).To(Equal(""))
		Expect(subject.IsLocked()).To(BeFalse())
		Expect(redisClient.Get(testRedisKey).Val()).To(Equal("ABCD"))
	})

	It("should refresh locks", func() {
		Expect(subject.Lock()).To(BeTrue())
		Expect(subject.IsLocked()).To(BeTrue())

		time.Sleep(50 * time.Millisecond)
		Expect(getTTL()).To(BeNumerically("~", 950*time.Millisecond, 10*time.Millisecond))

		Expect(subject.Lock()).To(BeTrue())
		Expect(subject.IsLocked()).To(BeTrue())
		Expect(getTTL()).To(BeNumerically("~", time.Second, 10*time.Millisecond))
	})

	It("should re-create expired locks on refresh", func() {
		Expect(subject.Lock()).To(BeTrue())
		Expect(subject.IsLocked()).To(BeTrue())
		token := subject.token

		Expect(redisClient.Del(testRedisKey).Err()).NotTo(HaveOccurred())

		Expect(subject.Lock()).To(BeTrue())
		Expect(subject.IsLocked()).To(BeTrue())
		Expect(subject.token).NotTo(Equal(token))
		Expect(getTTL()).To(BeNumerically("~", time.Second, 10*time.Millisecond))
	})

	It("should not re-capture expired locks acquiredby someone else", func() {
		Expect(subject.Lock()).To(BeTrue())
		Expect(subject.IsLocked()).To(BeTrue())
		Expect(redisClient.Set(testRedisKey, "ABCD", 0).Err()).NotTo(HaveOccurred())

		Expect(subject.Lock()).To(BeFalse())
		Expect(subject.IsLocked()).To(BeFalse())
	})

	It("should prevent multiple locks (fuzzing)", func() {
		res := int32(0)
		wg := new(sync.WaitGroup)
		for i := 0; i < 1000; i++ {
			wg.Add(1)
			go func() {
				defer GinkgoRecover()
				defer wg.Done()

				locker := newLock()
				wait := rand.Int63n(int64(50 * time.Millisecond))
				time.Sleep(time.Duration(wait))

				ok, err := locker.Lock()
				if err != nil {
					atomic.AddInt32(&res, 100)
					return
				} else if !ok {
					return
				}
				atomic.AddInt32(&res, 1)
			}()
		}
		wg.Wait()
		Expect(res).To(Equal(int32(1)))
	})

	It("should retry and wait for locks if requested", func() {
		var (
			wg  sync.WaitGroup
			res int32
		)

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer GinkgoRecover()

			err := Run(redisClient, testRedisKey, &Options{RetryCount: 10, RetryDelay: 10 * time.Millisecond}, func() error {
				atomic.AddInt32(&res, 1)
				return nil
			})
			Expect(err).NotTo(HaveOccurred())
		}()

		err := Run(redisClient, testRedisKey, nil, func() error {
			atomic.AddInt32(&res, 1)
			time.Sleep(20 * time.Millisecond)
			return nil
		})
		wg.Wait()

		Expect(err).NotTo(HaveOccurred())
		Expect(res).To(Equal(int32(2)))
	})

	It("should give up retrying after timeout", func() {
		var (
			wg  sync.WaitGroup
			res int32
		)

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer GinkgoRecover()

			err := Run(redisClient, testRedisKey, &Options{RetryCount: 1, RetryDelay: 10 * time.Millisecond}, func() error {
				atomic.AddInt32(&res, 1)
				return nil
			})
			Expect(err).To(Equal(ErrLockNotObtained))
		}()

		err := Run(redisClient, testRedisKey, nil, func() error {
			atomic.AddInt32(&res, 1)
			time.Sleep(100 * time.Millisecond)
			return nil
		})
		wg.Wait()

		Expect(err).NotTo(HaveOccurred())
		Expect(res).To(Equal(int32(1)))
	})

})

// --------------------------------------------------------------------

func TestSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	AfterEach(func() {
		Expect(redisClient.Del(testRedisKey).Err()).NotTo(HaveOccurred())
	})
	RunSpecs(t, "redis-lock")
}

var redisClient *redis.Client

var _ = BeforeSuite(func() {
	redisClient = redis.NewClient(&redis.Options{
		Network: "tcp",
		Addr:    "127.0.0.1:6379", DB: 9,
	})
	Expect(redisClient.Ping().Err()).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	Expect(redisClient.Close()).To(Succeed())
})
