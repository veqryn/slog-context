package sloggrpc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"testing"

	slogctx "github.com/veqryn/slog-context"
	protogen "github.com/veqryn/slog-context/grpc/test/gen"
	"github.com/veqryn/slog-context/internal/test"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

var _ protogen.TestServer = &server{}

type server struct {
	responseName string
	responseErr  error
}

func (s server) Unary(ctx context.Context, req *protogen.TestReq) (*protogen.TestResp, error) {
	fmt.Printf("Server Received: %+v\n", req)
	return &protogen.TestResp{Name: s.responseName}, s.responseErr
}

func (s server) ClientStream(g grpc.ClientStreamingServer[protogen.TestReq, protogen.TestResp]) error {
	// TODO implement me
	panic("implement me")
}

func (s server) ServerStream(req *protogen.TestReq, g grpc.ServerStreamingServer[protogen.TestResp]) error {
	// TODO implement me
	panic("implement me")
}

func (s server) BidirectionalStream(g grpc.BidiStreamingServer[protogen.TestReq, protogen.TestResp]) error {
	// TODO implement me
	panic("implement me")
}

func TestUnary(t *testing.T) {
	serverLogger := &test.Handler{}
	slog.SetDefault(slog.New(slogctx.NewHandler(serverLogger, nil)))

	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(SlogUnaryServerInterceptor(WithAppendToAttributes(testAllAppendToAttributes.appendToAttrs))),
	)

	app := &server{}
	protogen.RegisterTestServer(srv, app)

	listener, dialer := getListener()
	defer listener.Close()

	go func() {
		if err := srv.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			t.Error(err)
		}
	}()
	defer srv.Stop()

	clientLogger := &test.Handler{}
	clientCtx := slogctx.NewCtx(context.Background(), slog.New(clientLogger))

	conn, err := grpc.NewClient("bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(SlogUnaryClientInterceptor(WithAppendToAttributes(testAllAppendToAttributes.appendToAttrs))),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	client := protogen.NewTestClient(conn)

	// Run the test
	app.responseName = "serverResponse"
	app.responseErr = nil
	_, err = client.Unary(clientCtx, &protogen.TestReq{Name: "clientRequest"})
	if err != nil {
		t.Fatal(err)
	}

	serverJson, err := serverLogger.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	// fmt.Println(string(serverJson))
	serverExpected := `{"time":"2023-09-29T13:00:59Z","level":"INFO","msg":"rpcReq","grpc_system":"grpc","grpc_pkg":"com.github.veqryn.slogcontext.grpc.test","grpc_svc":"Test","grpc_method":"Unary","role":"server","stream_server":false,"stream_client":false,"peer_host":"","peer_port":0,"req":{"name":"clientRequest"}}
{"time":"2023-09-29T13:00:59Z","level":"INFO","msg":"rpcResp","code_name":"OK","code":0,"grpc_system":"grpc","grpc_pkg":"com.github.veqryn.slogcontext.grpc.test","grpc_svc":"Test","grpc_method":"Unary","role":"server","stream_server":false,"stream_client":false,"peer_host":"","peer_port":0,"resp":{"name":"serverResponse"}}
`
	if string(serverJson) != serverExpected {
		t.Error("Expected:", serverExpected, "\nGot:", string(serverJson))
	}

	clientJson, err := clientLogger.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	// fmt.Println(string(clientJson))
	clientExpected := `{"time":"2023-09-29T13:00:59Z","level":"INFO","msg":"rpcReq","grpc_system":"grpc","grpc_pkg":"com.github.veqryn.slogcontext.grpc.test","grpc_svc":"Test","grpc_method":"Unary","role":"client","stream_server":false,"stream_client":false,"peer_host":"","peer_port":0,"req":{"name":"clientRequest"}}
{"time":"2023-09-29T13:00:59Z","level":"INFO","msg":"rpcResp","code_name":"OK","code":0,"grpc_system":"grpc","grpc_pkg":"com.github.veqryn.slogcontext.grpc.test","grpc_svc":"Test","grpc_method":"Unary","role":"client","stream_server":false,"stream_client":false,"peer_host":"","peer_port":0,"resp":{"name":"serverResponse"}}
`
	if string(clientJson) != clientExpected {
		t.Error("Expected:", clientExpected, "\nGot:", string(clientJson))
	}

	// Test an error response
	// Reset the loggers
	serverLogger.Records = nil
	clientLogger.Records = nil

	app.responseName = ""
	app.responseErr = status.New(codes.InvalidArgument, "missing name").Err()
	_, err = client.Unary(clientCtx, &protogen.TestReq{})
	if err == nil {
		t.Fatal("expected error")
	}

	serverJson, err = serverLogger.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	//fmt.Println(string(serverJson))
	serverExpected = `{"time":"2023-09-29T13:00:59Z","level":"INFO","msg":"rpcReq","grpc_system":"grpc","grpc_pkg":"com.github.veqryn.slogcontext.grpc.test","grpc_svc":"Test","grpc_method":"Unary","role":"server","stream_server":false,"stream_client":false,"peer_host":"","peer_port":0,"req":{}}
{"time":"2023-09-29T13:00:59Z","level":"WARN","msg":"rpcResp","code_name":"InvalidArgument","code":3,"err":"missing name","grpc_system":"grpc","grpc_pkg":"com.github.veqryn.slogcontext.grpc.test","grpc_svc":"Test","grpc_method":"Unary","role":"server","stream_server":false,"stream_client":false,"peer_host":"","peer_port":0,"resp":{}}
`
	if string(serverJson) != serverExpected {
		t.Error("Expected:", serverExpected, "\nGot:", string(serverJson))
	}

	clientJson, err = clientLogger.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	//fmt.Println(string(clientJson))
	clientExpected = `{"time":"2023-09-29T13:00:59Z","level":"INFO","msg":"rpcReq","grpc_system":"grpc","grpc_pkg":"com.github.veqryn.slogcontext.grpc.test","grpc_svc":"Test","grpc_method":"Unary","role":"client","stream_server":false,"stream_client":false,"peer_host":"","peer_port":0,"req":{}}
{"time":"2023-09-29T13:00:59Z","level":"WARN","msg":"rpcResp","code_name":"InvalidArgument","code":3,"err":"missing name","grpc_system":"grpc","grpc_pkg":"com.github.veqryn.slogcontext.grpc.test","grpc_svc":"Test","grpc_method":"Unary","role":"client","stream_server":false,"stream_client":false,"peer_host":"","peer_port":0,"resp":{}}
`
	if string(clientJson) != clientExpected {
		t.Error("Expected:", clientExpected, "\nGot:", string(clientJson))
	}
}

func getListener() (net.Listener, func(ctx context.Context, address string) (net.Conn, error)) {
	const bufSize = 1024 * 1024
	resolver.SetDefaultScheme("passthrough")
	listener := bufconn.Listen(bufSize)
	bufDialer := func(ctx context.Context, address string) (net.Conn, error) {
		return listener.Dial()
	}
	return listener, bufDialer
}

var testAllAppendToAttributes = disableFields{"ms": {}}
var testFewAppendToAttributes = disableFields{
	"ms":          {},
	"grpc_system": {},
	"grpc_pkg":    {},
	"grpc_svc":    {},
	"grpc_method": {},
	"peer_host":   {},
	"peer_port":   {},
}
