package worker

import (
	"context"
	"io"
	"testing"
	"time"

	"distributed-crawler/internal/workerhealth"
	crawlergrpc "distributed-crawler/pkg/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fakeWorkerStream struct {
	ctx       context.Context
	recvItems []*crawlergrpc.WorkerHeartbeat
	recvIndex int
	sent      []*crawlergrpc.WorkerCommand
}

func (f *fakeWorkerStream) Send(cmd *crawlergrpc.WorkerCommand) error {
	f.sent = append(f.sent, cmd)
	return nil
}

func (f *fakeWorkerStream) Recv() (*crawlergrpc.WorkerHeartbeat, error) {
	if f.recvIndex >= len(f.recvItems) {
		return nil, io.EOF
	}
	item := f.recvItems[f.recvIndex]
	f.recvIndex++
	return item, nil
}

func (f *fakeWorkerStream) SetHeader(metadata.MD) error  { return nil }
func (f *fakeWorkerStream) SendHeader(metadata.MD) error { return nil }
func (f *fakeWorkerStream) SetTrailer(metadata.MD)       {}
func (f *fakeWorkerStream) Context() context.Context {
	if f.ctx != nil {
		return f.ctx
	}
	return context.Background()
}
func (f *fakeWorkerStream) SendMsg(any) error { return nil }
func (f *fakeWorkerStream) RecvMsg(any) error { return nil }

func TestListWorkers_ReturnsRegistrySnapshots(t *testing.T) {
	t.Parallel()

	registry := workerhealth.NewRegistry(time.Hour, time.Hour)
	startedAt := time.Now().UTC().Add(-5 * time.Minute).Round(0)
	heartbeatAt := time.Now().UTC().Round(0)
	registry.UpdateHeartbeat(workerhealth.HeartbeatInfo{
		WorkerID:   "parser-1",
		WorkerType: "parser",
		Status:     workerhealth.StatusActive,
		Timestamp:  heartbeatAt,
		StartedAt:  startedAt,
	})

	impl := NewImplementation(registry, zap.NewNop())
	resp, err := impl.ListWorkers(context.Background(), &crawlergrpc.ListWorkersRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Workers, 1)
	assert.Equal(t, "parser-1", resp.Workers[0].WorkerId)
	assert.Equal(t, crawlergrpc.WorkerStatus_WORKER_STATUS_ACTIVE, resp.Workers[0].Status)
	assert.Equal(t, "parser", resp.Workers[0].WorkerType)
	assert.True(t, resp.Workers[0].LastHeartbeatAt.AsTime().Equal(heartbeatAt))
}

func TestDrainWorker_ValidatesInputAndQueuesCommand(t *testing.T) {
	t.Parallel()

	registry := workerhealth.NewRegistry(time.Hour, time.Hour)
	impl := NewImplementation(registry, zap.NewNop())

	resp, err := impl.DrainWorker(context.Background(), &crawlergrpc.DrainWorkerRequest{})
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	okResp, err := impl.DrainWorker(context.Background(), &crawlergrpc.DrainWorkerRequest{
		WorkerId: "worker-1",
		Reason:   "maintenance",
	})
	require.NoError(t, err)
	assert.False(t, okResp.Delivered)
	assert.Equal(t, "drain command queued", okResp.Message)
}

func TestForceKillWorker_QueuesCommand(t *testing.T) {
	t.Parallel()

	registry := workerhealth.NewRegistry(time.Hour, time.Hour)
	impl := NewImplementation(registry, zap.NewNop())

	resp, err := impl.ForceKillWorker(context.Background(), &crawlergrpc.ForceKillWorkerRequest{
		WorkerId: "worker-1",
		Reason:   "stuck",
	})
	require.NoError(t, err)
	assert.False(t, resp.Delivered)
	assert.Equal(t, "force kill command queued", resp.Message)
}

func TestHandleHeartbeat_MapsProtoStatusAndTimestamps(t *testing.T) {
	t.Parallel()

	registry := workerhealth.NewRegistry(time.Hour, time.Hour)
	impl := NewImplementation(registry, zap.NewNop())

	startedAt := time.Now().UTC().Add(-2 * time.Minute).Round(0)
	timestamp := time.Now().UTC().Round(0)
	impl.handleHeartbeat(&crawlergrpc.WorkerHeartbeat{
		WorkerId:   "worker-1",
		WorkerType: "fetcher",
		Status:     crawlergrpc.WorkerStatus_WORKER_STATUS_DRAINING,
		Timestamp:  timestamppb.New(timestamp),
		StartedAt:  timestamppb.New(startedAt),
	})

	resp, err := impl.ListWorkers(context.Background(), &crawlergrpc.ListWorkersRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Workers, 1)
	assert.Equal(t, crawlergrpc.WorkerStatus_WORKER_STATUS_DRAINING, resp.Workers[0].Status)
	assert.True(t, resp.Workers[0].LastHeartbeatAt.AsTime().Equal(timestamp))
}

func TestWorkerStream_ValidatesFirstHeartbeatAndHandlesEOF(t *testing.T) {
	t.Parallel()

	registry := workerhealth.NewRegistry(time.Hour, time.Hour)
	impl := NewImplementation(registry, zap.NewNop())

	err := impl.WorkerStream(&fakeWorkerStream{
		recvItems: []*crawlergrpc.WorkerHeartbeat{{}},
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	stream := &fakeWorkerStream{
		ctx: context.Background(),
		recvItems: []*crawlergrpc.WorkerHeartbeat{{
			WorkerId:   "worker-1",
			WorkerType: "parser",
			Status:     crawlergrpc.WorkerStatus_WORKER_STATUS_ACTIVE,
			Timestamp:  timestamppb.New(time.Now().UTC()),
		}},
	}
	require.NoError(t, impl.WorkerStream(stream))
}

func TestWorkerHelpers_MapUnknownValues(t *testing.T) {
	t.Parallel()

	assert.Equal(t, crawlergrpc.WorkerStatus_WORKER_STATUS_UNSPECIFIED, toProtoStatus(workerhealth.StatusUnknown))
	assert.Equal(t, workerhealth.StatusUnknown, fromProtoStatus(crawlergrpc.WorkerStatus_WORKER_STATUS_UNSPECIFIED))
	assert.Equal(t, crawlergrpc.WorkerCommandType_WORKER_COMMAND_UNSPECIFIED, toProtoCommandType("bad"))
	assert.WithinDuration(t, time.Now().UTC(), timestampOrNow(nil), time.Second)
	assert.True(t, timestampOrZero(nil).IsZero())
}
