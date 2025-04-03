package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	slogctx "github.com/veqryn/slog-context"
	sloggrpc "github.com/veqryn/slog-context/grpc"
	pb "github.com/veqryn/slog-context/grpc/test/gen"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func init() {
	// Create the *slogctx.Handler middleware
	h := slogctx.NewHandler(slog.NewJSONHandler(os.Stdout, nil), nil)
	slog.SetDefault(slog.New(h))
}

func main() {
	ctx := context.TODO()
	slog.Info("Starting client")

	// Create a grpc client connection
	conn, err := grpc.NewClient("localhost:8000",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(sloggrpc.SlogUnaryClientInterceptor()),
		grpc.WithChainStreamInterceptor(sloggrpc.SlogStreamClientInterceptor()),
	)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	client := pb.NewTestClient(conn)

	// Test the single/unary req-resp call
	// {"time":"2025-04-03T16:07:10Z", "level":"INFO", "msg":"rpcReq", "grpc_svc":"Test", "grpc_method":"Unary", "peer_host":"localhost", "peer_port":8000, "req":{"name":"John", "option":1}}
	resp, err := client.Unary(ctx, &pb.TestReq{
		Name:   "John",
		Option: 1,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(resp)

	// Test the client streaming
	cStream, err := client.ClientStream(ctx)
	if err != nil {
		panic(err)
	}

	for i := int32(1); i <= 3; i++ {
		cStream.Send(&pb.TestReq{
			Name:   "Bob",
			Option: i,
		})
	}
	resp, err = cStream.CloseAndRecv()
	if err != nil {
		panic(err)
	}
	fmt.Println(resp)

	// Test the server streaming
	sStream, err := client.ServerStream(ctx, &pb.TestReq{
		Name:   "Jane",
		Option: 1,
	})
	if err != nil {
		panic(err)
	}

	for {
		resp, err = sStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
		fmt.Println(resp)
	}

	// Test bi-direction streaming
	bStream, err := client.BidirectionalStream(ctx)
	if err != nil {
		panic(err)
	}

	for i := int32(1); i <= 4; i++ {
		err = bStream.Send(&pb.TestReq{
			Name:   "Cat",
			Option: i,
		})
		if err != nil {
			panic(err)
		}

		resp, err = bStream.Recv()
		if err != nil {
			panic(err)
		}
		i += resp.Option - i
		fmt.Println(resp)
	}

	err = bStream.CloseSend()
	if err != nil {
		panic(err)
	}

	for {
		resp, err = bStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
		fmt.Println(resp)
	}
}
