package worker

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	crawlergrpc "distributed-crawler/pkg/v1"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type MonitorStatusFunc func() crawlergrpc.WorkerStatus
type MonitorCountFunc func() int32

type WorkerMonitor struct {
	addr              string
	workerID          string
	workerType        string
	startedAt         time.Time
	activeTasks       MonitorCountFunc
	status            MonitorStatusFunc
	onDrain           func()
	onForceKill       func()
	logger            *zap.Logger
	heartbeatInterval time.Duration
}

func NewWorkerMonitor(
	addr string,
	workerID string,
	workerType string,
	startedAt time.Time,
	activeTasks MonitorCountFunc,
	status MonitorStatusFunc,
	onDrain func(),
	onForceKill func(),
	logger *zap.Logger,
) *WorkerMonitor {
	return &WorkerMonitor{
		addr:              addr,
		workerID:          workerID,
		workerType:        workerType,
		startedAt:         startedAt,
		activeTasks:       activeTasks,
		status:            status,
		onDrain:           onDrain,
		onForceKill:       onForceKill,
		logger:            logger,
		heartbeatInterval: 4 * time.Second,
	}
}

func (m *WorkerMonitor) Run(ctx context.Context) {
	backoff := time.Second
	maxBackoff := 15 * time.Second

	for {
		if ctx.Err() != nil {
			return
		}

		conn, err := grpc.DialContext(ctx, m.addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			m.logger.Warn("Failed to connect to coordinator", zap.Error(err))
			m.sleep(ctx, backoff)
			backoff = minDuration(backoff*2, maxBackoff)
			continue
		}

		client := crawlergrpc.NewWorkerServiceClient(conn)
		stream, err := client.WorkerStream(ctx)
		if err != nil {
			m.logger.Warn("Failed to open worker stream", zap.Error(err))
			conn.Close()
			m.sleep(ctx, backoff)
			backoff = minDuration(backoff*2, maxBackoff)
			continue
		}

		backoff = time.Second
		if err := m.runStream(ctx, stream); err != nil {
			m.logger.Warn("Worker stream closed", zap.Error(err))
		}

		conn.Close()
		m.sleep(ctx, backoff)
		backoff = minDuration(backoff*2, maxBackoff)
	}
}

func (m *WorkerMonitor) runStream(ctx context.Context, stream crawlergrpc.WorkerService_WorkerStreamClient) error {
	ticker := time.NewTicker(m.heartbeatInterval)
	defer ticker.Stop()

	if err := stream.Send(m.buildHeartbeat()); err != nil {
		return err
	}

	errCh := make(chan error, 1)
	go func() {
		for {
			cmd, err := stream.Recv()
			if err != nil {
				errCh <- err
				return
			}
			m.handleCommand(cmd)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errCh:
			return err
		case <-ticker.C:
			if err := stream.Send(m.buildHeartbeat()); err != nil {
				return err
			}
		}
	}
}

func (m *WorkerMonitor) buildHeartbeat() *crawlergrpc.WorkerHeartbeat {
	return &crawlergrpc.WorkerHeartbeat{
		WorkerId:    m.workerID,
		WorkerType:  m.workerType,
		Status:      m.status(),
		ActiveTasks: m.activeTasks(),
		Timestamp:   timestamppb.New(time.Now().UTC()),
		StartedAt:   timestamppb.New(m.startedAt),
	}
}

func (m *WorkerMonitor) handleCommand(cmd *crawlergrpc.WorkerCommand) {
	switch cmd.GetType() {
	case crawlergrpc.WorkerCommandType_WORKER_COMMAND_DRAIN:
		if m.onDrain != nil {
			m.onDrain()
		}
	case crawlergrpc.WorkerCommandType_WORKER_COMMAND_FORCE_KILL:
		if m.onForceKill != nil {
			m.onForceKill()
		}
	}
}

func (m *WorkerMonitor) sleep(ctx context.Context, d time.Duration) {
	select {
	case <-time.After(d):
	case <-ctx.Done():
	}
}

func NewWorkerID(workerType string) (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", err
	}

	suffix, err := randomHex(3)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s-%s-%s", hostname, workerType, suffix), nil
}

func randomHex(n int) (string, error) {
	if n <= 0 {
		return "", nil
	}

	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return hex.EncodeToString(buf), nil
}

func minDuration(a time.Duration, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
