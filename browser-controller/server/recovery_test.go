package server

import (
	"context"
	"strings"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRecoveryUnaryInterceptorRecoversHandlerPanic(t *testing.T) {
	const panicDetail = "sensitive panic detail"
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Panic"}

	response, err := RecoveryUnaryInterceptor(
		context.Background(),
		"request",
		info,
		func(context.Context, any) (any, error) {
			panic(panicDetail)
		},
	)
	if response != nil {
		t.Fatalf("response = %v, want nil", response)
	}
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}
	if status.Convert(err).Message() != "internal server error" {
		t.Fatalf("status message = %q, want generic internal error", status.Convert(err).Message())
	}
	if strings.Contains(err.Error(), panicDetail) {
		t.Fatal("panic detail leaked through the returned error")
	}

	want := "still serving"
	response, err = RecoveryUnaryInterceptor(
		context.Background(),
		"request",
		&grpc.UnaryServerInfo{FullMethod: "/test.Service/Healthy"},
		func(context.Context, any) (any, error) {
			return want, nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if response != want {
		t.Fatalf("response = %v, want %q", response, want)
	}
}
