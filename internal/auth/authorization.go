package auth

import (
	"context"

	"distributed-crawler/internal/domain/auth/models"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func RBACInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if isPublicEndpoint(info.FullMethod) {
			return handler(ctx, req)
		}

		roleValue, ok := GetUserRoleFromContext(ctx)
		if !ok {
			return nil, status.Error(codes.PermissionDenied, "missing role")
		}

		role, err := models.ParseRole(roleValue)
		if err != nil {
			return nil, status.Error(codes.PermissionDenied, "invalid role")
		}

		requiredRole, ok := requiredRoleForMethod(info.FullMethod)
		if !ok {
			return nil, status.Error(codes.PermissionDenied, "access denied")
		}

		if role.Level() < requiredRole.Level() {
			return nil, status.Error(codes.PermissionDenied, "access denied")
		}

		return handler(ctx, req)
	}
}

func requiredRoleForMethod(method string) (models.Role, bool) {
	switch method {
	case "/crawler.v1.CrawlerService/ListJobs",
		"/crawler.v1.CrawlerService/GetJob",
		"/crawler.v1.CrawlerService/ListTasksByJob",
		"/crawler.v1.CrawlerService/GetTask",
		"/crawler.v1.CrawlerService/GetTaskFileURL",
		"/crawler.v1.CrawlerService/GetTaskAnalytics",
		"/crawler.v1.CrawlerService/GetJobExportFileURL":
		return models.RoleRead, true
	case "/crawler.v1.CrawlerService/CreateJob",
		"/crawler.v1.PreviewService/CreatePreview",
		"/crawler.v1.PreviewService/GetPreview":
		return models.RoleReadWrite, true
	case "/crawler.v1.WorkerService/ListWorkers",
		"/crawler.v1.WorkerService/DrainWorker",
		"/crawler.v1.WorkerService/ForceKillWorker",
		"/crawler.v1.UserService/ListUsers",
		"/crawler.v1.UserService/UpdateUserRole",
		"/crawler.v1.QueueAdminService/ListQueueEndpoints",
		"/crawler.v1.QueueAdminService/CreateQueueEndpoint",
		"/crawler.v1.QueueAdminService/UpdateQueueEndpoint",
		"/crawler.v1.QueueAdminService/DeleteQueueEndpoint",
		"/crawler.v1.QueueAdminService/ListQueueRoutingRules",
		"/crawler.v1.QueueAdminService/UpsertQueueRoutingRules":
		return models.RoleAdministrator, true
	default:
		return "", false
	}
}
