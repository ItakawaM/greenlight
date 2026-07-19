package limiter

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const tokenBucketScript = `
local key = KEYS[1]
local burst = tonumber(ARGV[1])
local rps = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

-- Get current state or initialize
local bucket = redis.call('HMGET', key, 'tokens', 'last_refill')
local tokens = tonumber(bucket[1])
local last_refill = tonumber(bucket[2])

-- Initialize if this is the first request
if tokens == nil then
    tokens = burst
    last_refill = now
end

-- Calculate token refill
local time_passed = now - last_refill
local refills = math.floor(time_passed)

if refills > 0 then
    tokens = math.min(burst, tokens + (refills * rps))
    last_refill = last_refill + refills
end

-- Try to consume a token
local allowed = 0
if tokens >= 1 then
    tokens = tokens - 1
    allowed = 1
end

-- seconds until at least 1 token is available
local retry_after = 0
if tokens < 1 then
    retry_after = (1 - tokens) / rps
end

-- seconds until the bucket is back to full
local reset_after = (burst - tokens) / rps

-- Update state
local ttl = math.ceil(burst / rps) * 2 + 1
redis.call('HMSET', key, 'tokens', tokens, 'last_refill', last_refill)
redis.call('EXPIRE', key, ttl)

-- Return result: allowed (1 or 0) and remaining tokens
return {allowed, tokens, tostring(retry_after), tostring(reset_after)}
`

// TokenBucketLimiter is a redis-based rate limiter implementing Token Bucket algorithm.
//
// It's a modified version of https://github.com/redis/docs/blob/main/content/develop/use-cases/rate-limiter/go/token_bucket.go
type TokenBucketLimiter struct {
	client       *redis.Client
	burst        int
	rps          float64
	scriptSHA    string
	scriptLoaded bool
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
	if t.scriptLoaded {
		return nil
	}

	// could potentially be a data race? not sure
	sha, err := t.client.ScriptLoad(ctx, tokenBucketScript).Result()
	if err != nil {
		return err
	}

	t.scriptSHA = sha
	t.scriptLoaded = true

	return nil
}

// Allow checks whether the key-user is allowed the request and has not reached
// the limit. It returns allowed, remaining tokens, "retry after" time and "reset after" time.
func (t *TokenBucketLimiter) Allow(ctx context.Context, key string) (bool, float64, float64, float64, error) {
	if err := t.ensureScriptLoaded(ctx); err != nil {
		return false, 0, 0, 0, err
	}

	now := float64(time.Now().UnixMicro()) / 1e6

	result, err := t.client.EvalSha(ctx, t.scriptSHA, []string{key}, t.burst, t.rps, now).Result()
	if err != nil {
		if !redis.HasErrorPrefix(err, "NOSCRIPT") {
			return false, 0, 0, 0, fmt.Errorf("token bucket eval failed: %w", err)
		}

		result, err = t.client.Eval(ctx, tokenBucketScript, []string{key}, t.burst, t.rps, now).Result()
		if err != nil {
			return false, 0, 0, 0, fmt.Errorf("token bucket eval failed: %w", err)
		}
		t.scriptLoaded = false
	}

	// res is []interface{}: [allowed int64, tokens int64, retry_after string, reset_after string]
	vals, ok := result.([]any)
	if !ok || len(vals) != 4 {
		return false, 0, 0, 0, fmt.Errorf("unexpected token bucket result shape: %v", result)
	}

	allowed := vals[0].(int64) == 1
	remaining := float64(vals[1].(int64))

	retryAfter, err := strconv.ParseFloat(vals[2].(string), 64)
	if err != nil {
		return false, 0, 0, 0, fmt.Errorf("parsing retry_after: %w", err)
	}

	resetAfter, err := strconv.ParseFloat(vals[3].(string), 64)
	if err != nil {
		return false, 0, 0, 0, fmt.Errorf("parsing reset_after: %w", err)
	}

	return allowed, remaining, retryAfter, resetAfter, nil
}
