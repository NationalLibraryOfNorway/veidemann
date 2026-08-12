# veidemann-controller

Api server and controller for Veidemann

## Job queue counts

Controller forwards `QueueCountForJobExecution` to Frontier using only the job
execution ID. This replaces the former batch crawl-execution count RPC. Because
the API change is breaking, Dashboard, Controller, and Frontier must be upgraded
together.

Frontier does not backfill the underlying Redis job counters. Jobs active during
the rollout can report zero or partial queue counts; jobs started after Frontier
is upgraded report complete counts.

## gRPC inbound message size

Controller accepts the optional `MAX_INBOUND_MESSAGE_SIZE` environment variable
as a positive byte count. When it is unset, Controller does not configure a
server override and gRPC-Java's native inbound message-size default remains in
effect. Set it through deployment-specific environment configuration or a
Kubernetes overlay when needed; the Kubernetes base intentionally leaves it
unset. Frontier supports the same variable independently, so configure each
server that needs a larger inbound limit.
