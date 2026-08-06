package limiter

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/redis/go-redis/v9"

	_ "embed"
)

//go:embed token_bucket.lua
var tokenBucketScript string

// TokenBucketLimiter is a redis-based rate limiter implementing Token Bucket algorithm.
//
// It's a modified version of https://github.com/redis/docs/blob/main/content/develop/use-cases/rate-limiter/go/token_bucket.go
type TokenBucketLimiter struct {
	client       *redis.Client
	burst        int
	rps          float64
	mu           sync.RWMutex
	scriptSHA    string
	scriptLoaded bool
}

// RateLimitMetadata represents rate limit information for a specific key.
type RateLimitMetadata struct {
	Allowed    bool
	Remaining  float64
	RetryAfter float64
	ResetAfter float64
}

// New returns an initialized TokenBucketLimiter.
func New(client *redis.Client, burst int, rps float64) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		client: client,
		burst:  burst,
		rps:    rps,
	}
}

func (t *TokenBucketLimiter) ensureScriptLoaded(ctx context.Context) error {
	t.mu.RLock()
	loaded := t.scriptLoaded
	t.mu.RUnlock()
	if loaded {
		return nil
	}

	sha, err := t.client.ScriptLoad(ctx, tokenBucketScript).Result()
	if err != nil {
		return err
	}

	t.mu.Lock()
	t.scriptSHA = sha
	t.scriptLoaded = true
	t.mu.Unlock()

	return nil
}

// Allow checks whether the key-user is allowed the request and has not reached
// the limit. It returns allowed, remaining tokens, "retry after" time and "reset after" time.
func (t *TokenBucketLimiter) Allow(ctx context.Context, key string) (RateLimitMetadata, error) {
	if err := t.ensureScriptLoaded(ctx); err != nil {
		return RateLimitMetadata{}, err
	}

	t.mu.RLock()
	script := t.scriptSHA
	t.mu.RUnlock()

	result, err := t.client.EvalSha(ctx, script, []string{key}, t.burst, t.rps).Result()
	if err != nil {
		if !redis.HasErrorPrefix(err, "NOSCRIPT") {
			return RateLimitMetadata{}, fmt.Errorf("token bucket eval failed: %w", err)
		}

		result, err = t.client.Eval(ctx, tokenBucketScript, []string{key}, t.burst, t.rps).Result()
		if err != nil {
			return RateLimitMetadata{}, fmt.Errorf("token bucket eval failed: %w", err)
		}

		t.mu.Lock()
		t.scriptLoaded = false
		t.mu.Unlock()
	}

	// res is []interface{}: [allowed int64, tokens string, retry_after string, reset_after string]
	vals, ok := result.([]any)
	if !ok || len(vals) != 4 {
		return RateLimitMetadata{}, fmt.Errorf("unexpected token bucket result shape: %v", result)
	}

	allowedVal, ok := vals[0].(int64)
	if !ok {
		return RateLimitMetadata{}, fmt.Errorf("unexpected type for allowed: %T", vals[0])
	}
	allowed := allowedVal == 1

	remainingVal, ok := (vals[1].(string))
	if !ok {
		return RateLimitMetadata{}, fmt.Errorf("unexpected type for remaining: %T", vals[1])
	}
	remaining, err := strconv.ParseFloat(remainingVal, 64)
	if err != nil {
		return RateLimitMetadata{}, fmt.Errorf("parsing remaining: %w", err)
	}

	retryAfterVal, ok := (vals[2].(string))
	if !ok {
		return RateLimitMetadata{}, fmt.Errorf("unexpected type for retryAfter: %T", vals[2])
	}
	retryAfter, err := strconv.ParseFloat(retryAfterVal, 64)
	if err != nil {
		return RateLimitMetadata{}, fmt.Errorf("parsing retry_after: %w", err)
	}

	resetAfterVal, ok := (vals[3].(string))
	if !ok {
		return RateLimitMetadata{}, fmt.Errorf("unexpected type for resetAfter: %T", vals[3])
	}
	resetAfter, err := strconv.ParseFloat(resetAfterVal, 64)
	if err != nil {
		return RateLimitMetadata{}, fmt.Errorf("parsing reset_after: %w", err)
	}

	return RateLimitMetadata{
		Allowed:    allowed,
		Remaining:  remaining,
		RetryAfter: retryAfter,
		ResetAfter: resetAfter,
	}, nil
}
