package worker

import (
	"context"
	"errors"
	"io"
	"time"

	"distributed-crawler/internal/workerhealth"
	crawlergrpc "distributed-crawler/pkg/v1"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type WorkerImplementation struct {
	crawlergrpc.UnimplementedWorkerServiceServer
	registry *workerhealth.Registry
	logger   *zap.Logger
}

func NewImplementation(registry *workerhealth.Registry, logger *zap.Logger) *WorkerImplementation {
	return &WorkerImplementation{
		registry: registry,
		logger:   logger,
	}
}

func (i *WorkerImplementation) WorkerStream(stream crawlergrpc.WorkerService_WorkerStreamServer) error {
	ctx := stream.Context()

	firstHeartbeat, err := stream.Recv()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}

	if firstHeartbeat.GetWorkerId() == "" {
		return status.Error(codes.InvalidArgument, "worker_id is required")
	}

	workerID := firstHeartbeat.GetWorkerId()
	commandCh, pending := i.registry.AttachStream(workerID)
	defer i.registry.DetachStream(workerID)

	sendErr := make(chan error, 1)
	go func() {
		for _, cmd := range pending {
			if err := stream.Send(toProtoCommand(cmd)); err != nil {
				sendErr <- err
				return
			}
		}
		for {
			select {
			case <-ctx.Done():
				sendErr <- ctx.Err()
				return
			case cmd, ok := <-commandCh:
				if !ok {
					sendErr <- nil
					return
				}
				if err := stream.Send(toProtoCommand(cmd)); err != nil {
					sendErr <- err
					return
				}
			}
		}
	}()

	i.handleHeartbeat(firstHeartbeat)

	for {
		heartbeat, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		i.handleHeartbeat(heartbeat)

		select {
		case err := <-sendErr:
			if err == nil || errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		default:
		}
	}
}

func (i *WorkerImplementation) ListWorkers(ctx context.Context, _ *crawlergrpc.ListWorkersRequest) (*crawlergrpc.ListWorkersResponse, error) {
	snapshots := i.registry.List(time.Now().UTC())
	workers := make([]*crawlergrpc.WorkerInfo, 0, len(snapshots))

	for _, worker := range snapshots {
		workers = append(workers, &crawlergrpc.WorkerInfo{
			WorkerId:        worker.WorkerID,
			WorkerType:      worker.WorkerType,
			Status:          toProtoStatus(worker.Status),
			LastHeartbeatAt: timestamppb.New(worker.LastHeartbeatAt),
			ActiveTasks:     worker.ActiveTasks,
			Uptime:          durationpb.New(worker.Uptime),
		})
	}

	return &crawlergrpc.ListWorkersResponse{Workers: workers}, nil
}

func (i *WorkerImplementation) DrainWorker(ctx context.Context, req *crawlergrpc.DrainWorkerRequest) (*crawlergrpc.DrainWorkerResponse, error) {
	if req.GetWorkerId() == "" {
		return nil, status.Error(codes.InvalidArgument, "worker_id is required")
	}

	delivered := i.registry.RequestDrain(req.GetWorkerId(), req.GetReason())
	message := "drain command queued"
	if delivered {
		message = "drain command delivered"
	}

	i.logger.Info("Drain command requested",
		zap.String("worker_id", req.GetWorkerId()),
		zap.String("reason", req.GetReason()),
		zap.Bool("delivered", delivered),
	)

	return &crawlergrpc.DrainWorkerResponse{
		Delivered: delivered,
		Message:   message,
	}, nil
}

func (i *WorkerImplementation) ForceKillWorker(ctx context.Context, req *crawlergrpc.ForceKillWorkerRequest) (*crawlergrpc.ForceKillWorkerResponse, error) {
	if req.GetWorkerId() == "" {
		return nil, status.Error(codes.InvalidArgument, "worker_id is required")
	}

	delivered := i.registry.RequestForceKill(req.GetWorkerId(), req.GetReason())
	message := "force kill command queued"
	if delivered {
		message = "force kill command delivered"
	}

	i.logger.Warn("Force kill requested",
		zap.String("worker_id", req.GetWorkerId()),
		zap.String("reason", req.GetReason()),
		zap.Bool("delivered", delivered),
	)

	return &crawlergrpc.ForceKillWorkerResponse{
		Delivered: delivered,
		Message:   message,
	}, nil
}

func (i *WorkerImplementation) handleHeartbeat(heartbeat *crawlergrpc.WorkerHeartbeat) {
	info := workerhealth.HeartbeatInfo{
		WorkerID:    heartbeat.GetWorkerId(),
		WorkerType:  heartbeat.GetWorkerType(),
		Status:      fromProtoStatus(heartbeat.GetStatus()),
		ActiveTasks: heartbeat.GetActiveTasks(),
		Timestamp:   timestampOrNow(heartbeat.GetTimestamp()),
		StartedAt:   timestampOrZero(heartbeat.GetStartedAt()),
	}

	i.registry.UpdateHeartbeat(info)
}

func toProtoStatus(status workerhealth.Status) crawlergrpc.WorkerStatus {
	switch status {
	case workerhealth.StatusActive:
		return crawlergrpc.WorkerStatus_WORKER_STATUS_ACTIVE
	case workerhealth.StatusInactive:
		return crawlergrpc.WorkerStatus_WORKER_STATUS_INACTIVE
	case workerhealth.StatusDraining:
		return crawlergrpc.WorkerStatus_WORKER_STATUS_DRAINING
	case workerhealth.StatusDead:
		return crawlergrpc.WorkerStatus_WORKER_STATUS_DEAD
	default:
		return crawlergrpc.WorkerStatus_WORKER_STATUS_UNSPECIFIED
	}
}

func fromProtoStatus(status crawlergrpc.WorkerStatus) workerhealth.Status {
	switch status {
	case crawlergrpc.WorkerStatus_WORKER_STATUS_ACTIVE:
		return workerhealth.StatusActive
	case crawlergrpc.WorkerStatus_WORKER_STATUS_INACTIVE:
		return workerhealth.StatusInactive
	case crawlergrpc.WorkerStatus_WORKER_STATUS_DRAINING:
		return workerhealth.StatusDraining
	case crawlergrpc.WorkerStatus_WORKER_STATUS_DEAD:
		return workerhealth.StatusDead
	default:
		return workerhealth.StatusUnknown
	}
}

func toProtoCommand(cmd workerhealth.Command) *crawlergrpc.WorkerCommand {
	return &crawlergrpc.WorkerCommand{
		Type:     toProtoCommandType(cmd.Type),
		Reason:   cmd.Reason,
		IssuedAt: timestamppb.New(cmd.IssuedAt),
	}
}

func toProtoCommandType(cmdType workerhealth.CommandType) crawlergrpc.WorkerCommandType {
	switch cmdType {
	case workerhealth.CommandDrain:
		return crawlergrpc.WorkerCommandType_WORKER_COMMAND_DRAIN
	case workerhealth.CommandForceKill:
		return crawlergrpc.WorkerCommandType_WORKER_COMMAND_FORCE_KILL
	default:
		return crawlergrpc.WorkerCommandType_WORKER_COMMAND_UNSPECIFIED
	}
}

func timestampOrNow(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Now().UTC()
	}
	return ts.AsTime()
}

func timestampOrZero(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime()
}
