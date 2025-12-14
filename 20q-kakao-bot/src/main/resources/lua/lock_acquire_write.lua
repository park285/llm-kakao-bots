-- KEYS[1]: write lock key
-- KEYS[2]: read hash key
-- ARGV[1]: token
-- ARGV[2]: ttlMillis
if redis.call("EXISTS", KEYS[1]) == 1 or redis.call("EXISTS", KEYS[2]) == 1 then
    return 0
else
    redis.call("PSETEX", KEYS[1], ARGV[2], ARGV[1])
    return 1
end
