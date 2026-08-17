local ueIdKey = KEYS[1]
local uchgKey = KEYS[2]
local chgKey = KEYS[3]
local crawlExecutionIdCountKey = KEYS[4]
local jobExecutionIdCountKey = KEYS[5]
local crawlExecutionJobExecutionKey = KEYS[6]
local queueCountTotalKey = KEYS[7]
local uriremovequeuekey = KEYS[8]
local busyKey = KEYS[9]
local finalizeKey = KEYS[10]

local ueIdVal = ARGV[1]
local eid = ARGV[2]
local uriId = ARGV[3]
local chgId = ARGV[4]
local preserveActiveFetch = ARGV[5] == "true"
local finalizeTime = ARGV[6]

if preserveActiveFetch and redis.call('ZSCORE', busyKey, chgId) then
    local currentUriId = redis.call('HGET', chgKey, "u")
    -- A CHG becomes busy before Frontier finishes prefetch and stores its current
    -- URI. Preserve candidates during that window as well; otherwise abort cleanup
    -- can terminalize the execution before prefetch has saved its counters.
    if (not currentUriId) or currentUriId == "" or currentUriId == uriId then
        return -1
    end
end

local removed = redis.call('ZREM', ueIdKey, ueIdVal)
if removed <= 0 then
    redis.log(redis.LOG_WARNING, "REM", ueIdKey, ueIdVal, removed)
end

if removed > 0 then
    -- Decrement CHG queue count
    redis.call('HINCRBY', chgKey, "qc", -1)

    -- Decrement crawl execution queue count
    local remaining_uri_count = redis.call('HINCRBY', crawlExecutionIdCountKey, eid, -1)

    -- Decrement job execution queue count when the crawl-to-job mapping exists.
    -- Existing queues created before this counter was introduced have no mapping.
    local jobExecutionId = redis.call('HGET', crawlExecutionJobExecutionKey, eid)
    if jobExecutionId then
        local remaining_job_uri_count = redis.call('HINCRBY', jobExecutionIdCountKey, jobExecutionId, -1)
        if remaining_job_uri_count <= 0 then
            redis.call('HDEL', jobExecutionIdCountKey, jobExecutionId)
        end
    end

    if remaining_uri_count <= 0 then
        redis.call('HDEL', crawlExecutionIdCountKey, eid)
        redis.call('HDEL', crawlExecutionJobExecutionKey, eid)
        redis.call('ZADD', finalizeKey, finalizeTime, eid)
    end
    -- Decrement total queue count
    redis.call('DECR', queueCountTotalKey);

    -- Add uri to remove queue
    if uriId ~= '' then
        redis.call('RPUSH', uriremovequeuekey, uriId)
    end

    if redis.call('EXISTS', ueIdKey) == 0 then
        redis.call('ZREM', uchgKey, eid)
    end
end

return removed
