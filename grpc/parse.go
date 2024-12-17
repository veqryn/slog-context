package sloggrpc

import (
	"net"
	"strconv"
	"strings"
	"time"
)

// Based on or similar to code from:
// https://github.com/open-telemetry/opentelemetry-go-contrib
// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

type Role struct {
	Role         string `json:"role,omitempty"`
	ClientStream bool   `json:"client_stream,omitempty"`
	ServerStream bool   `json:"server_stream,omitempty"`
}

// Payload contains the protobuf message and metadata about it
type Payload struct {
	Payload any `json:"payload,omitempty"`
}

// Result contains information about the end result of the grpc call
type Result struct {
	Error   error         `json:"error,omitempty"` // Call status.FromError() to get the status code, message, and details
	Elapsed time.Duration `json:"elapsed,omitempty"`
}

// Call contains information about the grpc call being made
type Call struct {
	Package string `json:"package,omitempty"`
	Service string `json:"service,omitempty"`
	Method  string `json:"method,omitempty"`
}

type StreamInfo struct {
	MsgID int64 `json:"msg_id,omitempty"` // Incrementing ID
}

// parseFullMethod returns all applicable slog.Attr based on a gRPC's FullMethod.
//
// Parsing is consistent with grpc-go implementation:
// https://github.com/grpc/grpc-go/blob/v1.57.0/internal/grpcutil/method.go#L26-L39
func parseFullMethod(fullMethodName string) Call {
	if !strings.HasPrefix(fullMethodName, "/") {
		// Invalid format, does not follow `/package.service/method`.
		return Call{Service: fullMethodName} // The logs need something
	}
	name := fullMethodName[1:]
	pos := strings.LastIndex(name, "/")
	if pos < 0 {
		// Invalid format, does not follow `/package.service/method`.
		return Call{Service: fullMethodName} // The logs need something
	}
	fullService, method := name[:pos], name[pos+1:]

	pos = strings.LastIndex(fullService, ".")
	if pos < 0 {
		pos = 0
	}
	pkg, service := fullService[:pos], fullService[pos+1:]

	rval := Call{
		Package: pkg,
		Service: service,
		Method:  method,
	}
	if pkg == "" && service == "" && method == "" {
		rval.Service = fullMethodName // The logs need something
	}
	return rval
}

// Peer contains information about the other party
type Peer struct {
	Host string `json:"host,omitempty"`
	Port int    `json:"port,omitempty"`
}

// peerAttr returns attributes about the peer address.
func peerAttr(addr string) Peer {
	host, p, err := net.SplitHostPort(addr)
	if err != nil {
		return Peer{}
	}

	if host == "" {
		host = "127.0.0.1"
	}
	port, _ := strconv.Atoi(p)

	return Peer{
		Host: host,
		Port: port,
	}
}
