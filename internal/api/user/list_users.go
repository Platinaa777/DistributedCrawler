package user

import (
	"context"

	crawlergrpc "distributed-crawler/pkg/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (i *UserImplementation) ListUsers(ctx context.Context, _ *crawlergrpc.ListUsersRequest) (*crawlergrpc.ListUsersResponse, error) {
	users, err := i.userService.ListUsers(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list users")
	}

	result := make([]*crawlergrpc.User, 0, len(users))
	for _, user := range users {
		result = append(result, toProtoUser(user))
	}

	return &crawlergrpc.ListUsersResponse{Users: result}, nil
}
