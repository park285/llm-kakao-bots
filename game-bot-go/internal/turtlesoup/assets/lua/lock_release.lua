-- ============================================================
-- Script: turtlesoup_lock_release
-- Purpose: WRITE 락 해제 (holder 포함)
-- KEYS[1]: lock key
-- KEYS[2]: holder key
-- ARGV[1]: token
-- Returns: 1 if released, 0 otherwise
-- ============================================================

local lockKey = KEYS[1]
local holderKey = KEYS[2]
local token = ARGV[1]

local deleted = 0
if redis.call("GET", lockKey) == token then
  deleted = redis.call("DEL", lockKey)
end

local holderVal = redis.call("GET", holderKey)
if holderVal then
  local delimPos = string.find(holderVal, "|", 1, true)
  if delimPos then
    local holderToken = string.sub(holderVal, 1, delimPos - 1)
    if holderToken == token then
      redis.call("DEL", holderKey)
    end
  end
end

return deleted
