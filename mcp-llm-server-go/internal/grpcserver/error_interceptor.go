package grpcserver

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func errorMapperInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		resp, err := handler(ctx, req)
		if err == nil {
			return resp, nil
		}

		// status.Error(...)를 이미 반환한 경우, 중복 매핑으로 code/message가 변형되는 것 방지함.
		if _, ok := status.FromError(err); ok {
			return resp, err
		}

		return resp, statusFromError(err)
	}
}
