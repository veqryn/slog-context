package sloggrpc

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

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

	// Test: Send a good request, get a good response
	app.Reset()
	app.responseName = []string{"serverResponse"}
	app.responseErr = []error{nil}
	resp, err := client.Unary(clientCtx, &protogen.TestReq{Name: "clientRequest"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Name != "serverResponse" {
		t.Fatal("Expected serverResponse, got ", resp.Name)
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

	// Test: Send a bad request, get an error response
	// Reset the loggers
	serverLogger.Records = nil
	clientLogger.Records = nil

	app.Reset()
	app.responseName = []string{"serverError"}
	app.responseErr = []error{status.New(codes.InvalidArgument, "missing name").Err()}
	resp, err = client.Unary(clientCtx, &protogen.TestReq{})
	if err == nil {
		t.Fatal("expected an error")
	}
	if resp != nil {
		t.Fatal("Expected nil; Got:", resp)
	}

	serverJson, err = serverLogger.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	//fmt.Println(string(serverJson))
	serverExpected = `{"time":"2023-09-29T13:00:59Z","level":"INFO","msg":"rpcReq","grpc_system":"grpc","grpc_pkg":"com.github.veqryn.slogcontext.grpc.test","grpc_svc":"Test","grpc_method":"Unary","role":"server","stream_server":false,"stream_client":false,"peer_host":"","peer_port":0,"req":{}}
{"time":"2023-09-29T13:00:59Z","level":"WARN","msg":"rpcResp","code_name":"InvalidArgument","code":3,"err":"missing name","grpc_system":"grpc","grpc_pkg":"com.github.veqryn.slogcontext.grpc.test","grpc_svc":"Test","grpc_method":"Unary","role":"server","stream_server":false,"stream_client":false,"peer_host":"","peer_port":0,"resp":{"name":"serverError"}}
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

func TestClientStreaming(t *testing.T) {
	serverLogger := &test.Handler{}
	slog.SetDefault(slog.New(slogctx.NewHandler(serverLogger, nil)))

	srv := grpc.NewServer(
		grpc.ChainStreamInterceptor(SlogStreamServerInterceptor(WithAppendToAttributes(testFewAppendToAttributes.appendToAttrs))),
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
		grpc.WithChainStreamInterceptor(SlogStreamClientInterceptor(WithAppendToAttributes(testFewAppendToAttributes.appendToAttrs))),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	client := protogen.NewTestClient(conn)

	// Test: Create a stream, send 2 good requests, close and get 1 good response
	app.Reset()
	app.responseName = []string{"serverResponse1"}
	app.responseErr = []error{nil}
	app.maxReceives = 10

	stream, err := client.ClientStream(clientCtx)
	if err != nil {
		t.Fatal(err)
	}

	err = stream.Send(&protogen.TestReq{Name: "clientRequest1"})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond) // GRPC buffers under the hood, so let the server catch up

	err = stream.Send(&protogen.TestReq{Name: "clientRequest2"})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond) // GRPC buffers under the hood, so let the server catch up

	_, err = stream.CloseAndRecv()
	if err != nil {
		t.Fatal(err)
	}

	serverJson, err := serverLogger.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	// fmt.Println(string(serverJson))
	serverExpected := `{"time":"2023-09-29T13:00:59Z","level":"INFO","msg":"rpcStreamStart","role":"server","stream_server":false,"stream_client":true}
{"time":"2023-09-29T13:00:59Z","level":"INFO","msg":"rpcStreamRecv","code_name":"OK","code":0,"role":"server","stream_server":false,"stream_client":true,"desc":{"msg_id":1},"req":{"name":"clientRequest1"}}
{"time":"2023-09-29T13:00:59Z","level":"INFO","msg":"rpcStreamRecv","code_name":"OK","code":0,"role":"server","stream_server":false,"stream_client":true,"desc":{"msg_id":2},"req":{"name":"clientRequest2"}}
{"time":"2023-09-29T13:00:59Z","level":"INFO","msg":"rpcStreamEnd","code_name":"OK","code":0,"role":"server","stream_server":false,"stream_client":true,"resp":{"name":"serverResponse1"}}
`
	if string(serverJson) != serverExpected {
		t.Error("Expected:", serverExpected, "\nGot:", string(serverJson))
	}

	clientJson, err := clientLogger.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	// fmt.Println(string(clientJson))
	clientExpected := `{"time":"2023-09-29T13:00:59Z","level":"INFO","msg":"rpcStreamStart","code_name":"OK","code":0,"role":"client","stream_server":false,"stream_client":true}
{"time":"2023-09-29T13:00:59Z","level":"INFO","msg":"rpcStreamSend","code_name":"OK","code":0,"role":"client","stream_server":false,"stream_client":true,"desc":{"msg_id":1},"req":{"name":"clientRequest1"}}
{"time":"2023-09-29T13:00:59Z","level":"INFO","msg":"rpcStreamSend","code_name":"OK","code":0,"role":"client","stream_server":false,"stream_client":true,"desc":{"msg_id":2},"req":{"name":"clientRequest2"}}
{"time":"2023-09-29T13:00:59Z","level":"INFO","msg":"rpcStreamEnd","code_name":"OK","code":0,"role":"client","stream_server":false,"stream_client":true,"resp":{"name":"serverResponse1"}}
`
	if string(clientJson) != clientExpected {
		t.Error("Expected:", clientExpected, "\nGot:", string(clientJson))
	}

	// Test: Create a stream, send 1 good request then 1 bad request, close, get a response and an error
	// Reset the loggers
	serverLogger.Records = nil
	clientLogger.Records = nil

	app.Reset()
	app.responseName = []string{"serverError"}
	app.responseErr = []error{status.New(codes.InvalidArgument, "missing name").Err()}
	app.maxReceives = 10

	stream, err = client.ClientStream(clientCtx)
	if err != nil {
		t.Fatal(err)
	}

	err = stream.Send(&protogen.TestReq{Name: "clientRequest1"})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond) // GRPC buffers under the hood, so let the server catch up

	err = stream.Send(&protogen.TestReq{})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond) // GRPC buffers under the hood, so let the server catch up

	_, err = stream.CloseAndRecv()
	if err == nil {
		t.Fatal("expected an error")
	}

	serverJson, err = serverLogger.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	// fmt.Println(string(serverJson))
	serverExpected = `{"time":"2023-09-29T13:00:59Z","level":"INFO","msg":"rpcStreamStart","role":"server","stream_server":false,"stream_client":true}
{"time":"2023-09-29T13:00:59Z","level":"INFO","msg":"rpcStreamRecv","code_name":"OK","code":0,"role":"server","stream_server":false,"stream_client":true,"desc":{"msg_id":1},"req":{"name":"clientRequest1"}}
{"time":"2023-09-29T13:00:59Z","level":"INFO","msg":"rpcStreamRecv","code_name":"OK","code":0,"role":"server","stream_server":false,"stream_client":true,"desc":{"msg_id":2},"req":{}}
{"time":"2023-09-29T13:00:59Z","level":"WARN","msg":"rpcStreamEnd","code_name":"InvalidArgument","code":3,"err":"missing name","role":"server","stream_server":false,"stream_client":true,"resp":{"name":"serverError"}}
`
	if string(serverJson) != serverExpected {
		t.Error("Expected:", serverExpected, "\nGot:", string(serverJson))
	}

	clientJson, err = clientLogger.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	// fmt.Println(string(clientJson))
	clientExpected = `{"time":"2023-09-29T13:00:59Z","level":"INFO","msg":"rpcStreamStart","code_name":"OK","code":0,"role":"client","stream_server":false,"stream_client":true}
{"time":"2023-09-29T13:00:59Z","level":"INFO","msg":"rpcStreamSend","code_name":"OK","code":0,"role":"client","stream_server":false,"stream_client":true,"desc":{"msg_id":1},"req":{"name":"clientRequest1"}}
{"time":"2023-09-29T13:00:59Z","level":"INFO","msg":"rpcStreamSend","code_name":"OK","code":0,"role":"client","stream_server":false,"stream_client":true,"desc":{"msg_id":2},"req":{}}
{"time":"2023-09-29T13:00:59Z","level":"WARN","msg":"rpcStreamEnd","code_name":"InvalidArgument","code":3,"err":"missing name","role":"client","stream_server":false,"stream_client":true,"resp":{"name":"serverError"}}
`
	if string(clientJson) != clientExpected {
		t.Error("Expected:", clientExpected, "\nGot:", string(clientJson))
	}

	// Test: Create a stream, send 1 bad then 1 good requests, receive an EOF, close, then get an error
	// Reset the loggers
	serverLogger.Records = nil
	clientLogger.Records = nil

	app.Reset()
	app.responseName = []string{""}
	app.responseErr = []error{status.New(codes.InvalidArgument, "missing name").Err()}
	app.maxReceives = 1

	stream, err = client.ClientStream(clientCtx)
	if err != nil {
		t.Fatal(err)
	}

	err = stream.Send(&protogen.TestReq{Name: "clientRequest1"})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond) // GRPC buffers under the hood, so let the server catch up

	err = stream.Send(&protogen.TestReq{Name: "clientRequest2"})
	if err != io.EOF {
		t.Fatal("expected EOF")
	}
	time.Sleep(time.Millisecond) // GRPC buffers under the hood, so let the server catch up

	_, err = stream.CloseAndRecv()
	if err == nil {
		t.Fatal("expected an error")
	}

	serverJson, err = serverLogger.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	// fmt.Println(string(serverJson))
	serverExpected = `{"time":"2023-09-29T13:00:59Z","level":"INFO","msg":"rpcStreamStart","role":"server","stream_server":false,"stream_client":true}
{"time":"2023-09-29T13:00:59Z","level":"INFO","msg":"rpcStreamRecv","code_name":"OK","code":0,"role":"server","stream_server":false,"stream_client":true,"desc":{"msg_id":1},"req":{"name":"clientRequest1"}}
{"time":"2023-09-29T13:00:59Z","level":"WARN","msg":"rpcStreamEnd","code_name":"InvalidArgument","code":3,"err":"missing name","role":"server","stream_server":false,"stream_client":true}
`
	if string(serverJson) != serverExpected {
		t.Error("Expected:", serverExpected, "\nGot:", string(serverJson))
	}

	clientJson, err = clientLogger.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	// fmt.Println(string(clientJson))
	clientExpected = `{"time":"2023-09-29T13:00:59Z","level":"INFO","msg":"rpcStreamStart","code_name":"OK","code":0,"role":"client","stream_server":false,"stream_client":true}
{"time":"2023-09-29T13:00:59Z","level":"INFO","msg":"rpcStreamSend","code_name":"OK","code":0,"role":"client","stream_server":false,"stream_client":true,"desc":{"msg_id":1},"req":{"name":"clientRequest1"}}
{"time":"2023-09-29T13:00:59Z","level":"WARN","msg":"rpcStreamSend","code_name":"Unknown","code":2,"err":"EOF","role":"client","stream_server":false,"stream_client":true,"desc":{"msg_id":2},"req":{"name":"clientRequest2"}}
{"time":"2023-09-29T13:00:59Z","level":"WARN","msg":"rpcStreamEnd","code_name":"InvalidArgument","code":3,"err":"missing name","role":"client","stream_server":false,"stream_client":true,"resp":{}}
`
	if string(clientJson) != clientExpected {
		t.Error("Expected:", clientExpected, "\nGot:", string(clientJson))
	}
}

var _ protogen.TestServer = &server{}

type server struct {
	index        int
	responseName []string
	responseErr  []error
	maxReceives  int
}

func (s *server) Reset() {
	s.index = 0
	s.responseName = nil
	s.responseErr = nil
}

func (s *server) Unary(ctx context.Context, req *protogen.TestReq) (*protogen.TestResp, error) {
	// fmt.Printf("Server Received: %+v\n", req)
	rval, rerr := &protogen.TestResp{Name: s.responseName[s.index]}, s.responseErr[s.index]
	s.index++
	return rval, rerr
}

func (s *server) ClientStream(stream grpc.ClientStreamingServer[protogen.TestReq, protogen.TestResp]) error {
	for i := 0; i < s.maxReceives; i++ {
		_, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
		// fmt.Printf("Server Received: %+v\n", req)
	}
	rval, rerr := &protogen.TestResp{Name: s.responseName[s.index]}, s.responseErr[s.index]
	s.index++
	if rval.Name == "" && rerr != nil {
		return rerr
	}
	err := stream.SendAndClose(rval)
	if err != nil {
		panic(err)
	}
	return rerr
}

func (s *server) ServerStream(req *protogen.TestReq, stream grpc.ServerStreamingServer[protogen.TestResp]) error {
	// TODO implement me
	panic("implement me")
}

func (s *server) BidirectionalStream(stream grpc.BidiStreamingServer[protogen.TestReq, protogen.TestResp]) error {
	// TODO implement me
	panic("implement me")
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
