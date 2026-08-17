local busyKey = KEYS[1]
local waitKey = KEYS[2]
local readyKey = KEYS[3]
local timeoutKey = KEYS[4]
local chgKey = KEYS[5]
local sessionKey = KEYS[6]
local uchgKey = KEYS[7]

local chgId = ARGV[1]
local queueCount = tonumber(redis.call('HGET', chgKey, "qc") or "0")

if queueCount > 0 then
    return 0
end

local sessionToken = redis.call('HGET', chgKey, "st")
if sessionToken and sessionToken ~= "" then
    redis.call('HDEL', sessionKey, sessionToken)
end

redis.call('ZREM', busyKey, chgId)
redis.call('ZREM', waitKey, chgId)
redis.call('LREM', readyKey, 0, chgId)
redis.call('LREM', timeoutKey, 0, chgId)
redis.call('DEL', uchgKey)
redis.call('DEL', chgKey)

return 1
