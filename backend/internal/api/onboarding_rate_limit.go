package api

import (
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/chrismott/miniclass/internal/api/problems"
)

const (
	// guardianOnboardingBurst permits a short legitimate burst before throttling.
	guardianOnboardingBurst = 20
	// guardianOnboardingRefill restores one full burst each refill window.
	guardianOnboardingRefill = 20
	// guardianOnboardingRefillWindow bounds the sustained request rate per client and route.
	guardianOnboardingRefillWindow = time.Minute
	// guardianOnboardingBucketLimit caps in-process memory under a distributed-client flood.
	guardianOnboardingBucketLimit = 10_000
	// guardianOnboardingBucketTTL removes inactive client buckets to reclaim memory.
	guardianOnboardingBucketTTL = 10 * time.Minute
)

// GuardianOnboardingRateLimitSettings controls the in-process pre-database
// limiter. Burst is the maximum number of requests a client may make at once
// for one protected route. Refill requests are added back over RefillWindow,
// so the sustained allowance is Refill requests per RefillWindow. For example,
// the defaults allow 20 immediate requests and then one request every 3 seconds
// for each client-and-route bucket. Zero values use the conservative defaults above.
type GuardianOnboardingRateLimitSettings struct {
	Burst        int
	Refill       int
	RefillWindow time.Duration
	BucketLimit  int
	BucketTTL    time.Duration
}

func defaultGuardianOnboardingRateLimitSettings() GuardianOnboardingRateLimitSettings {
	return GuardianOnboardingRateLimitSettings{
		Burst: guardianOnboardingBurst, Refill: guardianOnboardingRefill,
		RefillWindow: guardianOnboardingRefillWindow, BucketLimit: guardianOnboardingBucketLimit,
		BucketTTL: guardianOnboardingBucketTTL,
	}
}

func (s GuardianOnboardingRateLimitSettings) withDefaults() GuardianOnboardingRateLimitSettings {
	defaults := defaultGuardianOnboardingRateLimitSettings()
	if s.Burst <= 0 {
		s.Burst = defaults.Burst
	}
	if s.Refill <= 0 {
		s.Refill = defaults.Refill
	}
	if s.RefillWindow <= 0 {
		s.RefillWindow = defaults.RefillWindow
	}
	if s.BucketLimit <= 0 {
		s.BucketLimit = defaults.BucketLimit
	}
	if s.BucketTTL <= 0 {
		s.BucketTTL = defaults.BucketTTL
	}
	return s
}

type onboardingTokenBucket struct {
	tokens float64
	seenAt time.Time
}

type guardianOnboardingLimiter struct {
	mu          sync.Mutex
	buckets     map[string]onboardingTokenBucket
	now         func() time.Time
	lastCleanup time.Time
	logger      *slog.Logger
	settings    GuardianOnboardingRateLimitSettings
}

func GuardianOnboardingRateLimit(logger *slog.Logger, settings GuardianOnboardingRateLimitSettings) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	limiter := newGuardianOnboardingLimiter(time.Now, logger, settings)
	return limiter.middleware
}

func newGuardianOnboardingLimiter(now func() time.Time, logger *slog.Logger, settings GuardianOnboardingRateLimitSettings) *guardianOnboardingLimiter {
	if now == nil {
		now = time.Now
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &guardianOnboardingLimiter{buckets: make(map[string]onboardingTokenBucket), now: now, logger: logger, settings: settings.withDefaults()}
}

func (l *guardianOnboardingLimiter) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isGuardianOnboardingRateLimitedRoute(r) {
			next.ServeHTTP(w, r)
			return
		}
		if allowed, retryAfter := l.allow(guardianOnboardingRateLimitKey(r)); !allowed {
			w.Header().Set("Retry-After", strconv.Itoa(max(1, int(retryAfter.Seconds()))))
			l.logger.WarnContext(r.Context(), "guardian onboarding request rate limited",
				slog.String("path", r.URL.Path),
				slog.String("client", guardianOnboardingClientAddress(r)),
			)
			problems.Write(w, problems.New(http.StatusTooManyRequests, problems.RateLimited, "too many onboarding requests"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isGuardianOnboardingRateLimitedRoute(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	switch r.URL.Path {
	case apiBasePath + "/guardian/onboarding/begin",
		apiBasePath + "/guardian/onboarding/invitation/redeem",
		apiBasePath + "/guardian/onboarding/otp/request":
		return true
	default:
		return false
	}
}

func guardianOnboardingRateLimitKey(r *http.Request) string {
	return guardianOnboardingClientAddress(r) + ":" + r.URL.Path
}

func guardianOnboardingClientAddress(r *http.Request) string {
	if r != nil {
		if address, ok := parseRemoteAddr(r.RemoteAddr); ok {
			return address.String()
		}
	}
	return "unknown"
}

func (l *guardianOnboardingLimiter) allow(key string) (bool, time.Duration) {
	now := l.now().UTC()
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.lastCleanup.IsZero() || now.Sub(l.lastCleanup) >= l.settings.RefillWindow {
		l.evictExpired(now)
		l.lastCleanup = now
	}

	bucket, found := l.buckets[key]
	if !found {
		if len(l.buckets) >= l.settings.BucketLimit {
			l.evictOldest()
		}
		bucket = onboardingTokenBucket{tokens: float64(l.settings.Burst), seenAt: now}
	}
	elapsed := now.Sub(bucket.seenAt)
	if elapsed > 0 {
		bucket.tokens = min(float64(l.settings.Burst), bucket.tokens+elapsed.Seconds()*float64(l.settings.Refill)/l.settings.RefillWindow.Seconds())
	}
	bucket.seenAt = now
	if bucket.tokens >= 1 {
		bucket.tokens--
		l.buckets[key] = bucket
		return true, 0
	}
	l.buckets[key] = bucket
	retryAfter := time.Duration(math.Ceil(l.settings.RefillWindow.Seconds()/float64(l.settings.Refill))) * time.Second
	return false, retryAfter
}

func (l *guardianOnboardingLimiter) evictExpired(now time.Time) {
	for key, bucket := range l.buckets {
		if now.Sub(bucket.seenAt) >= l.settings.BucketTTL {
			delete(l.buckets, key)
		}
	}
}

func (l *guardianOnboardingLimiter) evictOldest() {
	var oldestKey string
	var oldest time.Time
	for key, bucket := range l.buckets {
		if oldestKey == "" || bucket.seenAt.Before(oldest) {
			oldestKey, oldest = key, bucket.seenAt
		}
	}
	if oldestKey != "" {
		delete(l.buckets, oldestKey)
	}
}
