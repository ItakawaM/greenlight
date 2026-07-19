package limiter

import (
	"context"
	"crypto/sha1"
	"fmt"
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

-- Update state
local ttl = math.ceil(burst / rps) * 2 + 1
redis.call('HMSET', key, 'tokens', tokens, 'last_refill', last_refill)
redis.call('EXPIRE', key, ttl)

local wait = 0
if allowed == 0 then
    local tokens_needed = 1 - tokens
    local seconds_needed = math.ceil(tokens_needed / rps)
    local time_since_refill = now - last_refill
    wait = seconds_needed - time_since_refill
end

-- Return result: allowed (1 or 0), remaining tokens and wait time
return {allowed, tokens, wait}
`

type TokenBucketLimiter struct {
	client       *redis.Client
	burst        int
	rps          float64
	scriptSHA    string
	scriptLoaded bool
}

func New(client *redis.Client, burst int, rps float64) *TokenBucketLimiter {
	h := sha1.New()
	h.Write([]byte(tokenBucketScript))
	sha := fmt.Sprintf("%x", h.Sum(nil))

	return &TokenBucketLimiter{
		client:    client,
		burst:     burst,
		rps:       rps,
		scriptSHA: sha,
	}
}

func (t *TokenBucketLimiter) ensureScriptLoaded(ctx context.Context) error {
	if t.scriptLoaded {
		return nil
	}

	sha, err := t.client.ScriptLoad(ctx, tokenBucketScript).Result()
	if err != nil {
		return err
	}

	t.scriptSHA = sha
	t.scriptLoaded = true

	return nil
}

func (t *TokenBucketLimiter) Allow(ctx context.Context, key string) (bool, float64, error) {
	if err := t.ensureScriptLoaded(ctx); err != nil {
		return false, 0.0, err
	}

	now := float64(time.Now().UnixMicro()) / 1e6

	result, err := t.client.EvalSha(ctx, t.scriptSHA, []string{key}, t.burst, t.rps, now).Int64Slice()
	if err != nil {
		if !redis.HasErrorPrefix(err, "NOSCRIPT") {
			return false, 0, fmt.Errorf("token bucket eval failed: %w", err)
		}

		result, err = t.client.Eval(ctx, tokenBucketScript, []string{key}, t.burst, t.rps, now).Int64Slice()
		if err != nil {
			return false, 0, fmt.Errorf("token bucket eval failed: %w", err)
		}
		t.scriptLoaded = false
	}

	allowed := result[0] == 1
	remaining := float64(result[1])

	return allowed, remaining, nil
}
