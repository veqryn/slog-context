package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"os"
	"strings"

	slogctx "github.com/veqryn/slog-context"
	sloggrpc "github.com/veqryn/slog-context/grpc"
	pb "github.com/veqryn/slog-context/grpc/test/gen"
	"google.golang.org/grpc"
)

func init() {
	// Create the *slogctx.Handler middleware
	h := slogctx.NewHandler(slog.NewJSONHandler(os.Stdout, nil), nil)
	slog.SetDefault(slog.New(h))
}

func main() {
	ctx := context.TODO()
	slog.Info("Starting server. Please run: grpcurl localhost:8080/hello") // TODO: fix

	// Create api app
	app := &Api{}

	// Create a listener on TCP port for gRPC:
	lis, err := net.Listen("tcp", ":8000")
	if err != nil {
		slogctx.Error(ctx, "Unable to create grpc listener", slogctx.Err(err))
		panic(err)
	}

	// Create a gRPC server, and register our app as the handler/server for the service interface
	// https://github.com/grpc-ecosystem/go-grpc-middleware
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			sloggrpc.SlogUnaryServerInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			sloggrpc.SlogStreamServerInterceptor(),
		),
	)
	pb.RegisterTestServer(grpcServer, app)

	// Start gRPC server
	serveErr := grpcServer.Serve(lis)
	if serveErr != nil && !errors.Is(serveErr, grpc.ErrServerStopped) {
		panic(serveErr)
	}
}

// GRPC setup
var _ pb.TestServer = &Api{}

type Api struct{}

func (a Api) Unary(ctx context.Context, req *pb.TestReq) (*pb.TestResp, error) {
	return &pb.TestResp{
		Name:   "Hello " + req.Name,
		Option: req.Option + 1,
	}, nil
}

func (a Api) ClientStream(stream grpc.ClientStreamingServer[pb.TestReq, pb.TestResp]) error {
	var reqNames []string
	var reqOptions int32
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
		reqNames = append(reqNames, req.Name)
		reqOptions += req.Option
	}

	return stream.SendAndClose(&pb.TestResp{
		Name:   "Hello " + strings.Join(reqNames, ", "),
		Option: reqOptions + 1,
	})
}

func (a Api) ServerStream(req *pb.TestReq, stream grpc.ServerStreamingServer[pb.TestResp]) error {
	for i := int32(0); i < req.Option; i++ {
		err := stream.Send(&pb.TestResp{
			Name:   "Hello " + req.Name,
			Option: req.Option + i,
		})
		if err != nil {
			panic(err)
		}
	}
	return nil
}

func (a Api) BidirectionalStream(stream grpc.BidiStreamingServer[pb.TestReq, pb.TestResp]) error {
	var i int32
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}

		i = req.Option + 1
		err = stream.Send(&pb.TestResp{
			Name:   "Hello " + req.Name,
			Option: i,
		})
		if err != nil {
			panic(err)
		}
	}
	return stream.Send(&pb.TestResp{
		Name:   "Goodbye",
		Option: i + 1,
	})
}
