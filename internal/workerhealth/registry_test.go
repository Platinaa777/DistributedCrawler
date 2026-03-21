package workerhealth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_UpdateHeartbeatAndListLifecycle(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Round(0)
	reg := NewRegistry(2*time.Minute, 10*time.Minute)
	reg.UpdateHeartbeat(HeartbeatInfo{
		WorkerID:   "w1",
		WorkerType: "parser",
		Status:     StatusActive,
		Timestamp:  now,
		StartedAt:  now.Add(-time.Minute),
	})

	snapshots := reg.List(now.Add(time.Minute))
	require.Len(t, snapshots, 1)
	assert.Equal(t, StatusActive, snapshots[0].Status)
	assert.Equal(t, "parser", snapshots[0].WorkerType)
	assert.Greater(t, snapshots[0].Uptime, time.Duration(0))

	reg.UpdateHeartbeat(HeartbeatInfo{
		WorkerID:   "w1",
		WorkerType: "",
		Status:     StatusActive,
		Timestamp:  now.Add(90 * time.Second),
	})
	snapshots = reg.List(now.Add(4 * time.Minute))
	require.Len(t, snapshots, 1)
	assert.Equal(t, StatusInactive, snapshots[0].Status)
}

func TestRegistry_DrainAndForceKillCommands(t *testing.T) {
	t.Parallel()

	reg := NewRegistry(time.Minute, 5*time.Minute)
	commandCh, pending := reg.AttachStream("w1")
	require.Empty(t, pending)

	delivered := reg.RequestDrain("w1", "maintenance")
	assert.True(t, delivered)
	cmd := <-commandCh
	assert.Equal(t, CommandDrain, cmd.Type)
	assert.Equal(t, "maintenance", cmd.Reason)

	delivered = reg.RequestForceKill("w1", "stuck")
	assert.True(t, delivered)
	cmd = <-commandCh
	assert.Equal(t, CommandForceKill, cmd.Type)
	assert.Equal(t, "stuck", cmd.Reason)

	reg.DetachStream("w1")
}

func TestRegistry_PendingCommandsAndCleanup(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Round(0)
	reg := NewRegistry(time.Minute, time.Minute)
	assert.False(t, reg.RequestDrain("offline", "later"))

	_, pending := reg.AttachStream("offline")
	require.Len(t, pending, 1)
	assert.Equal(t, CommandDrain, pending[0].Type)

	reg.UpdateHeartbeat(HeartbeatInfo{
		WorkerID:   "cleanup",
		Status:     StatusDead,
		Timestamp:  now,
		StartedAt:  now.Add(-time.Minute),
	})
	snapshots := reg.List(now.Add(3 * time.Minute))
	require.Len(t, snapshots, 1)
	assert.Equal(t, "offline", snapshots[0].WorkerID)
}

func TestHelpers_NormalizeStatusAndUptime(t *testing.T) {
	t.Parallel()

	assert.Equal(t, StatusUnknown, normalizeStatus("bad"))
	startedAt := time.Now().UTC()
	record := &WorkerRecord{
		Status:          StatusInactive,
		StartedAt:       startedAt,
		LastHeartbeatAt: startedAt.Add(2 * time.Minute),
	}
	assert.Equal(t, 2*time.Minute, uptimeFor(record, startedAt.Add(5*time.Minute)))
	assert.Equal(t, startedAt, safeTime(time.Time{}, startedAt))
	assert.Equal(t, "fallback", firstNonEmpty("", "fallback"))
}
