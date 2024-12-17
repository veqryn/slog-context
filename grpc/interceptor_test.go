package sloggrpc_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"

	protogen "github.com/veqryn/slog-context/grpc/test/gen"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

var _ protogen.TestServer = &server{}

type server struct{}

func (s server) Test(ctx context.Context, req *protogen.TestReq) (*protogen.TestResp, error) {
	fmt.Printf("Server Received: %+v\n", req)
	return &protogen.TestResp{Name: "server reply"}, nil
}

func TestUnary(t *testing.T) {
	srv := grpc.NewServer()
	protogen.RegisterTestServer(srv, &server{})

	listener, dialer := getListener()
	defer listener.Close()

	go func() {
		if err := srv.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			t.Error(err)
		}
	}()
	defer srv.Stop()

	ctx := context.Background()
	conn, err := grpc.NewClient("bufnet", grpc.WithContextDialer(dialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	client := protogen.NewTestClient(conn)

	resp, err := client.Test(ctx, &protogen.TestReq{})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(resp)
}

func getListener() (net.Listener, func(ctx context.Context, address string) (net.Conn, error)) {
	resolver.SetDefaultScheme("passthrough")
	listener := bufconn.Listen(bufSize)
	bufDialer := func(ctx context.Context, address string) (net.Conn, error) {
		return listener.Dial()
	}
	return listener, bufDialer
}
