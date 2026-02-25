package interceptor

import (
	"context"
	"distributed-crawler/internal/infra/logger"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func LogInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	now := time.Now()

	res, err := handler(ctx, req)
	if err != nil {
		logger.Error(
			err.Error(), 
			zap.String("method", info.FullMethod),
		)
	}

	logger.Info("request", 
		zap.String("method", info.FullMethod), 
		zap.Duration("duration", 
		time.Since(now)),
	)

	return res, err
}
