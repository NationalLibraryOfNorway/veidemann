package server

import (
	"context"
	"log/slog"
	"runtime/debug"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RecoveryUnaryInterceptor prevents a panic in one unary RPC from terminating
// the browser-controller process and all active fetches.
func RecoveryUnaryInterceptor(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (resp any, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			slog.Error("Recovered panic from unary gRPC handler",
				"method", info.FullMethod,
				"panic", recovered,
				"stack", string(debug.Stack()),
			)
			resp = nil
			err = status.Error(codes.Internal, "internal server error")
		}
	}()

	return handler(ctx, req)
}
