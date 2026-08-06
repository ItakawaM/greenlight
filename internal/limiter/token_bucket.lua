local key = KEYS[1]
local burst = tonumber(ARGV[1])
local rps = tonumber(ARGV[2])

local time_result = redis.call('TIME')
local now = tonumber(time_result[1]) + tonumber(time_result[2]) / 1e6

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
redis.call('HSET', key, 'tokens', tokens, 'last_refill', last_refill)
redis.call('EXPIRE', key, ttl)

-- Return result: 
-- allowed (1 or 0) 
-- remaining tokens
-- time until at least one token
-- time until full bucket
return {allowed, tostring(tokens), tostring(retry_after), tostring(reset_after)}