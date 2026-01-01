package testhelper

import (
	"context"
	"net"
	"sync"
	"testing"

	"google.golang.org/grpc"
)

// StartTestGRPCServer: 테스트용 gRPC 서버를 시작하고, grpc:// baseURL과 stop 함수를 반환합니다.
func StartTestGRPCServer(t *testing.T, register func(s *grpc.Server)) (baseURL string, stop func()) {
	t.Helper()

	lis, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}

	server := grpc.NewServer()
	if register != nil {
		register(server)
	}

	go func() {
		_ = server.Serve(lis)
	}()

	var once sync.Once
	stop = func() {
		once.Do(func() {
			server.Stop()
			_ = lis.Close()
		})
	}

	t.Cleanup(stop)
	return "grpc://" + lis.Addr().String(), stop
}
