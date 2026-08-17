local sourceExecutionQueue = KEYS[1]
local sourceExecutionWeights = KEYS[2]
local sourceChg = KEYS[3]
local targetExecutionQueue = KEYS[4]
local targetExecutionWeights = KEYS[5]
local targetChg = KEYS[6]
local waitQueue = KEYS[7]

local sourceMember = ARGV[1]
local targetMember = ARGV[2]
local executionId = ARGV[3]
local sourceChgId = ARGV[4]
local targetChgId = ARGV[5]
local weight = tonumber(ARGV[6])
local readyTime = ARGV[7]

if sourceChgId == targetChgId then
    return 0
end

-- A retry after an acknowledged move is a no-op.
if redis.call('ZSCORE', targetExecutionQueue, targetMember) then
    return 0
end

if redis.call('ZREM', sourceExecutionQueue, sourceMember) == 0 then
    return -1
end

redis.call('HINCRBY', sourceChg, 'qc', -1)
if redis.call('EXISTS', sourceExecutionQueue) == 0 then
    redis.call('ZREM', sourceExecutionWeights, executionId)
end

local targetExists = redis.call('EXISTS', targetChg)
redis.call('ZADD', targetExecutionQueue, 0, targetMember)
redis.call('ZADD', targetExecutionWeights, weight, executionId)
redis.call('HINCRBY', targetChg, 'qc', 1)
if targetExists == 0 then
    redis.call('ZADD', waitQueue, readyTime, targetChgId)
end

return 1
