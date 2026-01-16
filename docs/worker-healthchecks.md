The system has a Coordinator and multiple Workers communicating over gRPC.

Your task is to implement:

Worker heartbeat

Each worker must periodically (every 3–5 seconds) send a heartbeat to the Coordinator over gRPC.

The heartbeat must include: worker_id, timestamp, current status, number of active tasks.

The Coordinator must track last_heartbeat_at per worker.

If no heartbeat is received for 10–15 seconds, the worker is marked as inactive.

Worker state management

Worker states: ACTIVE, INACTIVE, DRAINING, DEAD.

State transitions must be explicit and persisted in Coordinator memory/storage.

Graceful shutdown (drain)

Coordinator must be able to send a DRAIN command to a worker via gRPC.

When draining, a worker must stop accepting new tasks, finish current tasks, then shut down gracefully.

Force kill

Coordinator must support a FORCE_KILL command for unresponsive workers.

This is an emergency operation and must be clearly separated from graceful shutdown.

Monitoring API for UI

Expose an API that returns the list of workers with:
worker_id, status, last_heartbeat_at, active_tasks, uptime.

This data will be used by a monitoring page in the UI.

Architecture constraints

Coordinator must NOT directly restart worker processes.

Process restarts are handled by the runtime (Docker / Kubernetes / system supervisor).

gRPC should be used for all worker control and health communication.

Focus on clean state transitions, fault tolerance, and production-like behavior.
Prefer streaming gRPC where appropriate.