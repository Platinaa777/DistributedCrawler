# Coordinator (gRPC Server) Improvement Suggestions

## Current State (from code)
Your coordinator currently supports:
- Worker heartbeat stream: `WorkerStream`
- Worker list view: `ListWorkers`
- Manual control commands:
  - `DrainWorker`
  - `ForceKillWorker`

Current limitations:
- Coordinator is mostly manual, not policy-driven.
- Control is only per `worker_id` (no fleet/group operations).
- No command acknowledgement lifecycle (`issued -> received -> started -> finished/failed`).
- No built-in automation for remediation (stuck drain, repeated reconnects, overload).
- `active_tasks` exists in proto but is not actually propagated end-to-end now.

## First Mandatory Fix (before adding new smart features)

### 1) Propagate `active_tasks` correctly
You already have these fields in proto:
- `WorkerHeartbeat.active_tasks`
- `WorkerInfo.active_tasks`

But current implementation does not fill them.

What to change:
- `internal/worker/monitoring.go`
  - In `buildHeartbeat()`, set `ActiveTasks` from worker runtime state.
- `internal/workerhealth/registry.go`
  - Store active task count in `WorkerRecord` and snapshot.
- `internal/api/worker/worker_impl.go`
  - Map active task count into `ListWorkers` response.

Why this matters:
- Without real load signals, coordinator cannot make smart decisions.

## Smart Coordinator Functions to Add

## Priority A (high impact, low complexity)

### 2) `DrainWorkerGroup`
Drain all workers by type (fetch/parser/export/scheduler), with optional limit.

Why:
- Rolling restart/maintenance is currently painful one worker at a time.

Suggested RPC:
- `DrainWorkerGroup(DrainWorkerGroupRequest) returns (DrainWorkerGroupResponse)`

Request fields:
- `worker_type`
- `reason`
- `max_workers` (optional; for progressive drain)

Response fields:
- `matched`
- `delivered`
- `queued`
- `worker_ids`

---

### 3) `ResumeWorker` and `ResumeWorkerGroup`
Reverse draining state and allow intake again.

Why:
- Current model has drain/kill only, no explicit resume command.

Suggested RPCs:
- `ResumeWorker(ResumeWorkerRequest) returns (ResumeWorkerResponse)`
- `ResumeWorkerGroup(ResumeWorkerGroupRequest) returns (ResumeWorkerGroupResponse)`

Worker-side behavior:
- For fetch/parser: recreate consume context and restart intake loop.
- For export/scheduler: set `accepting=true`.

---

### 4) `GetWorkerDetails`
Detailed worker endpoint (not just list view).

Why:
- `ListWorkers` is useful for table view, but insufficient for diagnostics.

Suggested RPC:
- `GetWorkerDetails(GetWorkerDetailsRequest) returns (GetWorkerDetailsResponse)`

Useful fields:
- worker identity and status
- active tasks
- last heartbeat latency/skew
- last N commands with status
- reconnect count (last 1h)
- optional notes/labels

## Priority B (smart behavior and automation)

### 5) `AcknowledgeCommand` (worker -> coordinator)
Add command lifecycle tracking.

Why:
- Today coordinator does not know if drain was accepted, started, finished, or failed.

Suggested RPC:
- `AcknowledgeCommand(AcknowledgeCommandRequest) returns (AcknowledgeCommandResponse)`

Lifecycle states:
- `RECEIVED`
- `EXECUTING`
- `COMPLETED`
- `FAILED`

Add `command_id` to `WorkerCommand`.

---

### 6) `GetCoordinatorRecommendations`
Coordinator returns recommended actions based on live state.

Why:
- This is the core "smarter coordinator" feature.

Suggested RPC:
- `GetCoordinatorRecommendations(GetCoordinatorRecommendationsRequest) returns (GetCoordinatorRecommendationsResponse)`

Recommendation types:
- `DRAIN_WORKER` (flapping worker, long heartbeat gap)
- `RESUME_WORKER` (recovered and stable)
- `SCALE_FETCH_UP` / `SCALE_PARSER_UP` (if backlog/latency high)
- `NO_ACTION`

Input signals:
- worker statuses
- worker active tasks
- queue backlog/consumer lag (RabbitMQ)
- error rates from telemetry

---

### 7) `ApplyCoordinatorPolicy`
Store and apply policy rules for automatic actions.

Suggested RPC:
- `UpdateCoordinatorPolicy(UpdateCoordinatorPolicyRequest) returns (UpdateCoordinatorPolicyResponse)`
- `GetCoordinatorPolicy(GetCoordinatorPolicyRequest) returns (GetCoordinatorPolicyResponse)`

Policy examples:
- mark worker `INACTIVE` after 12s without heartbeat (already exists)
- auto-drain if reconnects > N in M minutes
- auto-force-kill if draining exceeds timeout and active tasks not decreasing
- max parallel drains per worker_type

## Priority C (operator and UI quality)

### 8) `StreamWorkerEvents`
Server-streaming of worker events for real-time UI and audit.

Suggested RPC:
- `StreamWorkerEvents(StreamWorkerEventsRequest) returns (stream WorkerEvent)`

Event types:
- heartbeat received
- status changed
- command issued
- command acked
- command timeout
- policy action executed

---

### 9) `ListCommands` / `RetryCommand` / `CancelCommand`
Command control plane for operators.

Why:
- Needed for recovery from transient coordinator/network issues.

---

### 10) `SetWorkerMetadata`
Attach labels/metadata (zone, node, purpose, version).

Why:
- Enables smarter group targeting and safer maintenance.

## Suggested Proto Additions (example)
```proto
message DrainWorkerGroupRequest {
  string worker_type = 1;
  string reason = 2;
  optional int32 max_workers = 3;
}

message ResumeWorkerRequest {
  string worker_id = 1;
  string reason = 2;
}

message WorkerCommand {
  string command_id = 1;
  WorkerCommandType type = 2;
  string reason = 3;
  google.protobuf.Timestamp issued_at = 4;
}

enum WorkerCommandState {
  WORKER_COMMAND_STATE_UNSPECIFIED = 0;
  WORKER_COMMAND_STATE_RECEIVED = 1;
  WORKER_COMMAND_STATE_EXECUTING = 2;
  WORKER_COMMAND_STATE_COMPLETED = 3;
  WORKER_COMMAND_STATE_FAILED = 4;
}

message AcknowledgeCommandRequest {
  string worker_id = 1;
  string command_id = 2;
  WorkerCommandState state = 3;
  optional string message = 4;
  google.protobuf.Timestamp timestamp = 5;
}
```

## Internal Implementation Functions to Add

Coordinator-side (`internal/workerhealth/registry.go`):
- `UpdateActiveTasks(workerID string, active int32)`
- `RequestResume(workerID, reason string) bool`
- `RequestDrainGroup(workerType, reason string, max int) DrainGroupResult`
- `RecordCommandAck(workerID, commandID string, state CommandState, message string)`
- `ListWorkerEvents(since time.Time, filter EventFilter) []WorkerEvent`
- `EvaluatePolicies(now time.Time) []RecommendedAction`

API-side (`internal/api/worker/worker_impl.go`):
- `ResumeWorker(...)`
- `DrainWorkerGroup(...)`
- `GetWorkerDetails(...)`
- `AcknowledgeCommand(...)`
- `GetCoordinatorRecommendations(...)`

Worker-side (`internal/worker/monitoring.go` and `internal/app/worker_app.go`):
- Include active tasks in heartbeat.
- Handle new command `RESUME`.
- Send command acknowledgements.

## Practical Smart Logic (simple, robust)
Start with deterministic rules (no ML):
- If `heartbeat_gap > inactive_threshold` -> mark `INACTIVE`.
- If worker reconnects > 5 in 2 minutes -> recommendation `DRAIN_WORKER`.
- If worker in `DRAINING` for > 2 minutes and active tasks not dropping -> recommendation `FORCE_KILL`.
- If fetch backlog high and parser stable -> recommendation `SCALE_FETCH_UP`.
- If parser backlog high and fetch stable -> recommendation `SCALE_PARSER_UP`.

## Incremental Rollout Plan

### Phase 1 (quick win)
- Fix active_tasks propagation.
- Add `DrainWorkerGroup`.
- Add `ResumeWorker`.

### Phase 2
- Add command IDs + ack lifecycle.
- Add `GetWorkerDetails` and command history.

### Phase 3
- Add policy engine + recommendations endpoint.
- Add event stream for UI/audit.

## Important Design Notes
- Keep `Registry` state thread-safe and bounded in memory (ring buffer for events/history).
- Persist critical command history if you need post-restart audit.
- Continue graceful degradation: if recommendation engine fails, manual control endpoints must still work.
- Keep default policy conservative (recommend first, auto-act second).

## Summary
To make coordinator truly smart, move from manual per-worker control to:
- load-aware signals (`active_tasks`, backlog),
- group operations,
- command lifecycle tracking,
- policy-based recommendations/automation,
- real-time event visibility.

This gives you an operator-safe path: first observability, then recommendations, then optional automation.
