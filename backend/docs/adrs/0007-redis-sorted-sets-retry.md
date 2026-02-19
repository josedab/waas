# ADR-0007: Redis Sorted Sets for Delayed Retry Scheduling

**Status:** Accepted
**Date:** 2025-06-01

## Context

WaaS implements at-least-once delivery with exponential backoff retries. Failed webhook deliveries must be re-attempted after a computed delay (e.g., 30s, 1m, 5m, 30m, 1h). The retry system must efficiently schedule thousands of concurrent delayed jobs, retrieve only jobs whose delay has elapsed, and handle concurrent consumers without double-processing.

## Decision

We use Redis sorted sets (`ZADD` / `ZRANGEBYSCORE` / `ZREM`) for delayed retry scheduling, where each job's score is its next-retry Unix timestamp.

## Rationale

- **Efficient time-based queries:** `ZRANGEBYSCORE` retrieves all jobs with score ≤ `now` in O(log N + M) time, making it ideal for "what's ready to retry?" queries.
- **Atomic operations:** `ZREM` returns whether the element was actually removed, providing natural distributed locking—only one consumer processes each job.
- **No polling waste:** Unlike polling a database table, sorted set range queries are constant-time relative to result set size, not total queue size.
- **Already deployed:** Redis is already required for rate limiting and caching, so no additional infrastructure.
- **Simple mental model:** Score = timestamp makes the scheduling behavior intuitive and debuggable (`ZRANGEBYSCORE retries -inf +inf WITHSCORES` shows the full schedule).

## Implementation

```
ZADD retry_queue <next_retry_unix_ts> <delivery_id>
ZRANGEBYSCORE retry_queue -inf <now_unix_ts> LIMIT 0 100
ZREM retry_queue <delivery_id>  -- returns 1 if this consumer "wins"
```

The `pkg/queue` package wraps these operations with retry logic and delivery record updates.

## Alternatives Considered

| Alternative | Why Not |
|-------------|---------|
| **RabbitMQ delayed message plugin** | Adds Erlang runtime dependency; plugin must be installed separately; less control over scheduling granularity. |
| **PostgreSQL scheduled jobs table** | Polling-based approach with `SELECT ... WHERE next_retry <= NOW()` creates lock contention at high throughput. |
| **Redis Streams with XAUTOCLAIM** | More complex consumer group management; delayed scheduling requires a separate mechanism. |
| **Go time.AfterFunc** | In-memory only; lost on process restart; doesn't work across multiple delivery engine instances. |

## Consequences

- Retry scheduling depends on Redis persistence (AOF/RDB). Redis crash without persistence loses pending retries; PostgreSQL delivery records serve as the recovery source.
- Clock skew between delivery engine instances can cause minor scheduling imprecision (acceptable for retry delays).
- The `pkg/queue.RetryQueue` interface abstracts the sorted set operations, allowing future migration to a different backend.
