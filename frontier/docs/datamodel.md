```Id=A, Uri=foo1.com, eid=eid1, chgid=chg1, weight=1
Id=B, Uri=foo2.com, eid=eid1, chgid=chg1, weight=1
Id=C, Uri=foo3.com, eid=eid1, chgid=chg2, weight=1
Id=D, Uri=foo4.com, eid=eid1, chgid=chg3, weight=1
Id=E, Uri=foo1.com, eid=eid2, chgid=chg1, weight=1
Id=F, Uri=foo2.com, eid=eid2, chgid=chg1, weight=1
Id=G, Uri=foo3.com, eid=eid2, chgid=chg2, weight=1
Id=H, Uri=foo4.com, eid=eid2, chgid=chg3, weight=1
Id=I, Uri=foo1.com, eid=eid3, chgid=chg1, weight=2
Id=J, Uri=foo2.com, eid=eid3, chgid=chg1, weight=2
Id=K, Uri=foo3.com, eid=eid3, chgid=chg2, weight=2
Id=L, Uri=foo4.com, eid=eid3, chgid=chg3, weight=2
```

```
UEID:chg:eid 0 = seq:fetchTime:UriId
UEID:chg1:eid1 = [1:10:A, 1:10:B]
UEID:chg2:eid1 = [1:10:C]
UEID:chg3:eid1 = [1:10:D]
UEID:chg1:eid2 = [1:10:E, 1:10:F]
UEID:chg2:eid2 = [1:10:G]
UEID:chg3:eid2 = [1:10:H]
UEID:chg1:eid3 = [1:10:I, 1:10:J]
UEID:chg2:eid3 = [1:10:K]
UEID:chg3:eid3 = [1:10:L]
```

```
UCHG:chg = (weight) eid
UCHG:chg1 = [(1) eid1, (1) eid2, (2) eid3]
UCHG:chg2 = [(1) eid1, (1) eid2, (2) eid3]
UCHG:chg3 = [(1) eid1, (1) eid2, (2) eid3]
```


getNext(chg)
```
max = ZREVRANGEBYSCORE chg +inf -inf WITHSCORES LIMIT 0 1 
eid = ZRANGEBYSCORE chg rnd()*max +inf 0 1
url = ZRANGE UEID:chg:eid 0 0
if url == nil
  return nil
else if url.fetchTime >= now()
  return url
else
  return url.fetchTime
```

remove(uri)
```
key = UEID:uri.chg:uri.eid
ZREM key uri.seq:uri.fetchTime:uri.id
if EXISTS key == false
  ZREM UCHG:uri.chg uri.eid
```

add(uri)
```
ZADD UEID:chg:eid 0 seq:fetchTime:uri.id
ZADD UCHG:chg uri.weight uri.eid
```

CrawlHostGroup

## Queue counters

Frontier maintains queue counts in Redis as part of the same Lua operation that
updates a crawl host group:

- `QCT` is the total queued URI count.
- `EIDC` is a hash from crawl-execution ID to queued URI count.
- `JEIDC` is a hash from job-execution ID to queued URI count.
- `EIDJ` is a hash from crawl-execution ID to its owning job-execution ID.

Adding a URI increments all three counters and records the ownership mapping.
Removing a URI decrements the counters only when the URI was actually removed.
Zero-valued crawl and job fields, and ownership mappings for empty crawl
executions, are deleted.

`JEIDC` and `EIDJ` are not backfilled during an upgrade. Queue counts for jobs
that were already active when Frontier was upgraded can therefore be zero or
partial. Jobs started after the upgraded Frontier is running have complete job
queue counts.

The single-execution count RPCs perform one hash lookup. The list-oriented
`QueueCountsForJobExecutions` and `QueueCountsForCrawlExecutions` RPCs accept at
most 100 IDs and perform one `HMGET` against `JEIDC` or `EIDC`. Missing or invalid
stored values are returned as zero.

## gRPC inbound message size

Frontier accepts the optional `MAX_INBOUND_MESSAGE_SIZE` environment variable as
a positive byte count. When it is unset, Frontier does not configure an override
and gRPC-Java's native inbound message-size default remains in effect. Set the
variable through deployment-specific environment configuration or a Kubernetes
overlay when a larger request is required; the Kubernetes base intentionally
does not set it.
