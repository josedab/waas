# ADR-0003: Redis over RabbitMQ for Message Queuing

**Status:** Accepted
**Date:** 2025-06-01

## Context

WaaS requires an asynchronous message queue to decouple HTTP event ingestion from webhook delivery. The queue must support delayed retries with exponential backoff, dead-letter semantics, and priority processing. The solution must work well for both single-node Docker Compose deployments and multi-node Kubernetes clusters.

## Decision

We chose Redis (with sorted sets for delayed scheduling) as the queue backend instead of a dedicated message broker like RabbitMQ or Kafka.

## Rationale

- **Operational simplicity:** Redis is already required for rate-limit counters and caching, so no additional infrastructure component is needed.
- **Sorted set scheduling:** `ZRANGEBYSCORE` on Redis sorted sets provides efficient O(log N) delayed retry scheduling—jobs are scored by their next-retry timestamp.
- **Self-hosted friendliness:** A single Redis instance is simpler to operate than a RabbitMQ cluster with Erlang runtime dependencies.
- **Sufficient guarantees:** At-least-once delivery is achieved by combining Redis queue operations with PostgreSQL delivery records. The delivery engine re-enqueues failed jobs on crash recovery.
- **Low latency:** Redis provides sub-millisecond enqueue/dequeue operations suitable for real-time webhook delivery.

## Alternatives Considered

| Alternative | Why Not |
|-------------|---------|
| **RabbitMQ** | More operational overhead (Erlang runtime, clustering, management UI). Overkill for single-node deployments. |
| **Kafka** | Designed for high-throughput streaming, not request-response webhook delivery. Heavy infrastructure for small deployments. |
| **PostgreSQL LISTEN/NOTIFY** | Limited throughput for high-volume queue operations; no built-in delayed scheduling. |
| **NATS** | Viable but less mature ecosystem; would add another infrastructure dependency. |

## Consequences

- Queue-related changes must consider Redis sorted set semantics (scores as timestamps).
- Delivery guarantees depend on the combination of Redis queues and PostgreSQL delivery records.
- At very high scale (>100K msg/s), a dedicated broker may be needed—this can be added as an alternative queue backend behind the `pkg/queue` interface.
- Redis persistence (AOF/RDB) should be configured to minimize message loss on crash.
