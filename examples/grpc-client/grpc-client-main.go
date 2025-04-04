package main

import (
	"context"
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
		// Add the interceptors
		// We will use the sloggrpc.AppendToAttributesAll option, which is fairly verbose with the attributes.
		// There is also a slimmer sloggrpc.AppendToAttributesDefault, which is what it used if no option is provided.
		// You can also write your own to customize which attributes are added, or rename their keys.
		// There are also other options available: WithInterceptorFilter, WithErrorToLevel, and WithLogger
		grpc.WithChainUnaryInterceptor(sloggrpc.SlogUnaryClientInterceptor(sloggrpc.WithAppendToAttributes(sloggrpc.AppendToAttributesAll))),
		grpc.WithChainStreamInterceptor(sloggrpc.SlogStreamClientInterceptor(sloggrpc.WithAppendToAttributes(sloggrpc.AppendToAttributesAll))),
	)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	client := pb.NewTestClient(conn)

	// Each called RPC below includes an example of the logs generated by the sloggrpc interceptor

	// Test the single/unary req-resp call
	/*
		{
		  "time": "2025-04-03T16:42:07Z",
		  "level": "INFO",
		  "msg": "rpcReq",
		  "grpc_system": "grpc",
		  "grpc_pkg": "com.github.veqryn.slogcontext.grpc.test",
		  "grpc_svc": "Test",
		  "grpc_method": "Unary",
		  "role": "client",
		  "stream_server": false,
		  "stream_client": false,
		  "peer_host": "localhost",
		  "peer_port": 8000,
		  "req": {
			"name": "John",
			"option": 1
		  }
		}
	*/
	resp, err := client.Unary(ctx, &pb.TestReq{
		Name:   "John",
		Option: 1,
	})
	if err != nil {
		panic(err)
	}
	/*
		{
		  "time": "2025-04-03T16:42:07Z",
		  "level": "INFO",
		  "msg": "rpcResp",
		  "code_name": "OK",
		  "code": 0,
		  "grpc_system": "grpc",
		  "grpc_pkg": "com.github.veqryn.slogcontext.grpc.test",
		  "grpc_svc": "Test",
		  "grpc_method": "Unary",
		  "role": "client",
		  "stream_server": false,
		  "stream_client": false,
		  "peer_host": "localhost",
		  "peer_port": 8000,
		  "ms": 27.467792,
		  "resp": {
			"name": "Hello John",
			"option": 2
		  }
		}
	*/

	// Test the client streaming
	/*
		{
		  "time": "2025-04-03T16:42:07Z",
		  "level": "INFO",
		  "msg": "rpcStreamStart",
		  "code_name": "OK",
		  "code": 0,
		  "grpc_system": "grpc",
		  "grpc_pkg": "com.github.veqryn.slogcontext.grpc.test",
		  "grpc_svc": "Test",
		  "grpc_method": "ClientStream",
		  "role": "client",
		  "stream_server": false,
		  "stream_client": true,
		  "peer_host": "localhost",
		  "peer_port": 8000,
		  "ms": 0.0175
		}
	*/
	cStream, err := client.ClientStream(ctx)
	if err != nil {
		panic(err)
	}

	for i := int32(1); i <= 3; i++ {
		/*
			{
			  "time": "2025-04-03T16:42:07Z",
			  "level": "INFO",
			  "msg": "rpcStreamSend",
			  "code_name": "OK",
			  "code": 0,
			  "grpc_system": "grpc",
			  "grpc_pkg": "com.github.veqryn.slogcontext.grpc.test",
			  "grpc_svc": "Test",
			  "grpc_method": "ClientStream",
			  "role": "client",
			  "stream_server": false,
			  "stream_client": true,
			  "peer_host": "localhost",
			  "peer_port": 8000,
			  "desc": {
				"msg_id": 3
			  },
			  "ms": 0.000333,
			  "req": {
				"name": "Bob",
				"option": 3
			  }
			}
		*/
		err = cStream.Send(&pb.TestReq{
			Name:   "Bob",
			Option: i,
		})
		if err != nil {
			panic(err)
		}
	}
	resp, err = cStream.CloseAndRecv()
	if err != nil {
		panic(err)
	}
	/*
		{
		  "time": "2025-04-03T16:42:07Z",
		  "level": "INFO",
		  "msg": "rpcStreamEnd",
		  "code_name": "OK",
		  "code": 0,
		  "grpc_system": "grpc",
		  "grpc_pkg": "com.github.veqryn.slogcontext.grpc.test",
		  "grpc_svc": "Test",
		  "grpc_method": "ClientStream",
		  "role": "client",
		  "stream_server": false,
		  "stream_client": true,
		  "peer_host": "localhost",
		  "peer_port": 8000,
		  "ms": 0.427959,
		  "resp": {
			"name": "Hello Bob, Bob, Bob",
			"option": 4
		  }
		}
	*/

	// Test the server streaming
	/*
		{
		  "time": "2025-04-03T16:42:07Z",
		  "level": "INFO",
		  "msg": "rpcStreamStart",
		  "code_name": "OK",
		  "code": 0,
		  "grpc_system": "grpc",
		  "grpc_pkg": "com.github.veqryn.slogcontext.grpc.test",
		  "grpc_svc": "Test",
		  "grpc_method": "ServerStream",
		  "role": "client",
		  "stream_server": true,
		  "stream_client": false,
		  "peer_host": "localhost",
		  "peer_port": 8000,
		  "ms": 0.010917,
		  "req": {
			"name": "Jane",
			"option": 1
		  }
		}
	*/
	sStream, err := client.ServerStream(ctx, &pb.TestReq{
		Name:   "Jane",
		Option: 1,
	})
	if err != nil {
		panic(err)
	}

	for {
		/*
			{
			  "time": "2025-04-03T16:42:07Z",
			  "level": "INFO",
			  "msg": "rpcStreamRecv",
			  "code_name": "OK",
			  "code": 0,
			  "grpc_system": "grpc",
			  "grpc_pkg": "com.github.veqryn.slogcontext.grpc.test",
			  "grpc_svc": "Test",
			  "grpc_method": "ServerStream",
			  "role": "client",
			  "stream_server": true,
			  "stream_client": false,
			  "peer_host": "localhost",
			  "peer_port": 8000,
			  "desc": {
				"msg_id": 1
			  },
			  "ms": 0.326,
			  "resp": {
				"name": "Hello Jane",
				"option": 1
			  }
			}
		*/
		resp, err = sStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
	}
	/*
		{
		  "time": "2025-04-03T16:42:07Z",
		  "level": "INFO",
		  "msg": "rpcStreamEnd",
		  "code_name": "OK",
		  "code": 0,
		  "grpc_system": "grpc",
		  "grpc_pkg": "com.github.veqryn.slogcontext.grpc.test",
		  "grpc_svc": "Test",
		  "grpc_method": "ServerStream",
		  "role": "client",
		  "stream_server": true,
		  "stream_client": false,
		  "peer_host": "localhost",
		  "peer_port": 8000,
		  "ms": 0.403125
		}
	*/

	// Test bi-direction streaming
	/*
		{
		  "time": "2025-04-03T16:42:07Z",
		  "level": "INFO",
		  "msg": "rpcStreamStart",
		  "code_name": "OK",
		  "code": 0,
		  "grpc_system": "grpc",
		  "grpc_pkg": "com.github.veqryn.slogcontext.grpc.test",
		  "grpc_svc": "Test",
		  "grpc_method": "BidirectionalStream",
		  "role": "client",
		  "stream_server": true,
		  "stream_client": true,
		  "peer_host": "localhost",
		  "peer_port": 8000,
		  "ms": 0.006167
		}
	*/
	bStream, err := client.BidirectionalStream(ctx)
	if err != nil {
		panic(err)
	}

	for i := int32(1); i <= 4; i++ {
		/*
			{
			  "time": "2025-04-03T16:42:07Z",
			  "level": "INFO",
			  "msg": "rpcStreamSend",
			  "code_name": "OK",
			  "code": 0,
			  "grpc_system": "grpc",
			  "grpc_pkg": "com.github.veqryn.slogcontext.grpc.test",
			  "grpc_svc": "Test",
			  "grpc_method": "BidirectionalStream",
			  "role": "client",
			  "stream_server": true,
			  "stream_client": true,
			  "peer_host": "localhost",
			  "peer_port": 8000,
			  "desc": {
				"msg_id": 1
			  },
			  "ms": 0.000792,
			  "req": {
				"name": "Cat",
				"option": 1
			  }
			}
		*/
		err = bStream.Send(&pb.TestReq{
			Name:   "Cat",
			Option: i,
		})
		if err != nil {
			panic(err)
		}

		/*
			{
			  "time": "2025-04-03T16:42:07Z",
			  "level": "INFO",
			  "msg": "rpcStreamRecv",
			  "code_name": "OK",
			  "code": 0,
			  "grpc_system": "grpc",
			  "grpc_pkg": "com.github.veqryn.slogcontext.grpc.test",
			  "grpc_svc": "Test",
			  "grpc_method": "BidirectionalStream",
			  "role": "client",
			  "stream_server": true,
			  "stream_client": true,
			  "peer_host": "localhost",
			  "peer_port": 8000,
			  "desc": {
				"msg_id": 2
			  },
			  "ms": 0.299792,
			  "resp": {
				"name": "Hello Cat",
				"option": 2
			  }
			}
		*/
		resp, err = bStream.Recv()
		if err != nil {
			panic(err)
		}
		i += resp.Option - i
	}

	err = bStream.CloseSend()
	if err != nil {
		panic(err)
	}

	for {
		/*
			{
			  "time": "2025-04-03T16:42:07Z",
			  "level": "INFO",
			  "msg": "rpcStreamRecv",
			  "code_name": "OK",
			  "code": 0,
			  "grpc_system": "grpc",
			  "grpc_pkg": "com.github.veqryn.slogcontext.grpc.test",
			  "grpc_svc": "Test",
			  "grpc_method": "BidirectionalStream",
			  "role": "client",
			  "stream_server": true,
			  "stream_client": true,
			  "peer_host": "localhost",
			  "peer_port": 8000,
			  "desc": {
				"msg_id": 5
			  },
			  "ms": 0.182125,
			  "resp": {
				"name": "Goodbye",
				"option": 5
			  }
			}
		*/
		resp, err = bStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
	}
	/*
		{
		  "time": "2025-04-03T16:42:07Z",
		  "level": "INFO",
		  "msg": "rpcStreamEnd",
		  "code_name": "OK",
		  "code": 0,
		  "grpc_system": "grpc",
		  "grpc_pkg": "com.github.veqryn.slogcontext.grpc.test",
		  "grpc_svc": "Test",
		  "grpc_method": "BidirectionalStream",
		  "role": "client",
		  "stream_server": true,
		  "stream_client": true,
		  "peer_host": "localhost",
		  "peer_port": 8000,
		  "ms": 0.830417
		}
	*/
}
