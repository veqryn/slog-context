package sloggrpc

import (
	"context"
	"io"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

// Based on or similar to code from:
// https://github.com/open-telemetry/opentelemetry-go-contrib
// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// SlogUnaryClientInterceptor returns a grpc.UnaryClientInterceptor suitable
// for use in a grpc.NewClient or grpc.WithChainUnaryInterceptor call.
// This interceptor will log requests and responses.
func SlogUnaryClientInterceptor(opts ...Option) grpc.UnaryClientInterceptor {
	// Closure over config
	cfg := newConfig(opts, "client")

	return func(
		ctx context.Context,
		method string,
		req any,
		resp any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		callOpts ...grpc.CallOption,
	) error {
		// See if we should skip intercepting this call
		i := &otelgrpc.InterceptorInfo{
			Method: method,
			Type:   otelgrpc.UnaryClient,
		}
		if cfg.InterceptorFilter != nil && !cfg.InterceptorFilter(i) {
			return invoker(ctx, method, req, resp, cc, callOpts...)
		}

		pr := peerAttr(cc.Target())
		call := parseFullMethod(method)
		reqPayload := Payload{Payload: req}
		role := Role{Role: cfg.role}

		// Log the request
		cfg.logRequest(ctx, role, call, pr, reqPayload)

		// Call the next interceptor or the actual invocation
		before := time.Now()
		err := invoker(ctx, method, req, resp, cc, callOpts...)

		// Log the response
		respPayload := Payload{Payload: resp}
		result := Result{
			Error:   err,
			Elapsed: time.Since(before),
		}
		cfg.logResponse(ctx, role, call, pr, reqPayload, respPayload, result)

		return err
	}
}

// clientStream  wraps around the embedded grpc.ClientStream, and intercepts the RecvMsg and
// SendMsg method call.
type clientStream struct {
	grpc.ClientStream

	cfg       *config
	role      Role
	call      Call
	pr        Peer
	start     time.Time
	messageID atomic.Int64
}

func (w *clientStream) RecvMsg(m any) error {
	before := time.Now()
	err := w.ClientStream.RecvMsg(m)

	// With client-streaming-only, CloseAndRecv sends the CloseSend before the
	// first and only RecvMsg call. So log the end with the payload.
	if !w.role.ServerStream {
		recvPayload := Payload{Payload: m}
		result := Result{
			Error:   err,
			Elapsed: time.Since(w.start),
		}
		w.cfg.logStreamEnd(w.Context(), w.role, w.call, w.pr, recvPayload, result)
		return err
	}

	id := w.messageID.Add(1)
	recvPayload := Payload{Payload: m}

	if err == io.EOF {
		// With server-streaming-only, the CloseSend is sent right away,
		// so we can only tell that the stream is done when we receive the io.EOF.
		// With bidirectional-streaming, the server has stopped streaming if io.EOF.
		result := Result{
			Error:   nil,
			Elapsed: time.Since(w.start),
		}
		w.cfg.logStreamEnd(w.Context(), w.role, w.call, w.pr, recvPayload, result)
		return err
	}

	// Server is still sending, so log it was received
	streamInfo := StreamInfo{MsgID: id}
	result := Result{
		Error:   err,
		Elapsed: time.Since(before),
	}
	w.cfg.logStreamRecv(w.Context(), w.role, w.call, streamInfo, w.pr, recvPayload, result)
	return err
}

func (w *clientStream) SendMsg(m any) error {
	before := time.Now()
	err := w.ClientStream.SendMsg(m)

	// If client streaming, there will be many request payloads, and we will log each.
	// Otherwise, log the start with the first (and only) payload sent.
	if !w.role.ClientStream {
		sendPayload := Payload{Payload: m}
		result := Result{
			Error:   err,
			Elapsed: time.Since(w.start),
		}
		w.cfg.logStreamStart(w.Context(), w.role, w.call, w.pr, sendPayload, result)
		return err
	}

	id := w.messageID.Add(1)

	// We don't mind if the error is io.EOF, because if it is, then it is unexpected,
	// and we should log the situation (the server has stopped receiving prematurely).
	streamInfo := StreamInfo{MsgID: id}
	sendPayload := Payload{Payload: m}
	result := Result{
		Error:   err,
		Elapsed: time.Since(before),
	}
	w.cfg.logStreamSend(w.Context(), w.role, w.call, streamInfo, w.pr, sendPayload, result)
	return err
}

func (w *clientStream) CloseSend() error {
	err := w.ClientStream.CloseSend()

	// With server-streaming-only, the CloseSend is sent right away.
	// With client-streaming-only, the CloseSend is sent before the final RecvMsg's.
	result := Result{
		Error:   err,
		Elapsed: time.Since(w.start),
	}
	w.cfg.logStreamClientSendClosed(w.Context(), w.role, w.call, w.pr, result)
	return err
}

func wrapClientStream(s grpc.ClientStream, cfg *config, role Role, before time.Time, call Call, pr Peer) *clientStream {
	return &clientStream{
		ClientStream: s,
		cfg:          cfg,
		role:         role,
		start:        before,
		call:         call,
		pr:           pr,
	}
}

// SlogStreamClientInterceptor returns a grpc.StreamClientInterceptor suitable
// for use in a grpc.NewClient or grpc.WithChainStreamInterceptor call.
// This interceptor will log stream start, sends and receives.
func SlogStreamClientInterceptor(opts ...Option) grpc.StreamClientInterceptor {
	cfg := newConfig(opts, "client")

	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		callOpts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		// See if we should skip intercepting this call
		i := &otelgrpc.InterceptorInfo{
			Method: method,
			Type:   otelgrpc.StreamClient,
		}
		if cfg.InterceptorFilter != nil && !cfg.InterceptorFilter(i) {
			return streamer(ctx, desc, cc, method, callOpts...)
		}

		role := Role{
			Role:         cfg.role,
			ClientStream: desc.ClientStreams,
			ServerStream: desc.ServerStreams,
		}
		pr := peerAttr(cc.Target())
		call := parseFullMethod(method)

		before := time.Now()
		s, err := streamer(ctx, desc, cc, method, callOpts...)

		// If client streaming, there will be many request payloads, so SendMsg will log them.
		// Otherwise, log the start with the first (and only) payload sent.
		if role.ClientStream {
			result := Result{
				Error:   err,
				Elapsed: time.Since(before),
			}
			cfg.logStreamStart(ctx, role, call, pr, Payload{}, result)
		}

		if err != nil {
			return s, err
		}
		return wrapClientStream(s, cfg, role, before, call, pr), nil
	}
}

// SlogUnaryServerInterceptor returns a grpc.UnaryServerInterceptor suitable
// for use in a grpc.NewServer or grpc.ChainUnaryInterceptor call.
// This interceptor will log requests and responses.
func SlogUnaryServerInterceptor(opts ...Option) grpc.UnaryServerInterceptor {
	// Closure:
	cfg := newConfig(opts, "server")

	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		// See if we should skip intercepting this call
		i := &otelgrpc.InterceptorInfo{
			UnaryServerInfo: info,
			Type:            otelgrpc.StreamServer,
		}
		if cfg.InterceptorFilter != nil && !cfg.InterceptorFilter(i) {
			return handler(ctx, req)
		}

		pr := Peer{}
		if p, ok := peer.FromContext(ctx); ok {
			pr = peerAttr(p.Addr.String())
		}

		call := parseFullMethod(info.FullMethod)
		reqPayload := Payload{Payload: req}
		role := Role{Role: cfg.role}

		// Log the request
		cfg.logRequest(ctx, role, call, pr, reqPayload)

		// Call the next interceptor or the actual handler
		before := time.Now()
		resp, err := handler(ctx, req)

		// Log the response
		respPayload := Payload{Payload: resp}
		result := Result{
			Error:   err,
			Elapsed: time.Since(before),
		}
		cfg.logResponse(ctx, role, call, pr, reqPayload, respPayload, result)

		return resp, err
	}
}

// serverStream wraps around the embedded grpc.ServerStream, and intercepts the RecvMsg and
// SendMsg method call.
type serverStream struct {
	grpc.ServerStream

	cfg       *config
	role      Role
	call      Call
	pr        Peer
	start     time.Time
	messageID atomic.Int64
}

func (w *serverStream) RecvMsg(m any) error {
	before := time.Now()
	err := w.ServerStream.RecvMsg(m)

	// If client streaming, there will be many request payloads, and we will log each.
	// Otherwise, log the start with the first (and only) payload received.
	if !w.role.ClientStream {
		recvPayload := Payload{Payload: m}
		result := Result{
			Error:   err,
			Elapsed: time.Since(before),
		}
		w.cfg.logStreamStart(w.Context(), w.role, w.call, w.pr, recvPayload, result)
		return err
	}

	// With client streaming, we will know it has ended with io.EOF
	if err == io.EOF {
		result := Result{
			Error:   nil,
			Elapsed: time.Since(before),
		}
		w.cfg.logStreamClientSendClosed(w.Context(), w.role, w.call, w.pr, result)
		return err
	}

	id := w.messageID.Add(1)

	streamInfo := StreamInfo{MsgID: id}
	recvPayload := Payload{Payload: m}
	result := Result{
		Error:   err,
		Elapsed: time.Since(before),
	}
	w.cfg.logStreamRecv(w.Context(), w.role, w.call, streamInfo, w.pr, recvPayload, result)
	return err
}

func (w *serverStream) SendMsg(m any) error {
	before := time.Now()
	err := w.ServerStream.SendMsg(m)
	sendPayload := Payload{Payload: m}

	// If server streaming, there will be many sent/response payloads, so SendMsg will log them all.
	// Otherwise, log the end with the first (and only) payload sent.
	if !w.role.ServerStream {
		result := Result{
			Error:   err,
			Elapsed: time.Since(w.start),
		}
		w.cfg.logStreamEnd(w.Context(), w.role, w.call, w.pr, sendPayload, result)
		return err
	}

	id := w.messageID.Add(1)

	streamInfo := StreamInfo{MsgID: id}
	result := Result{
		Error:   err,
		Elapsed: time.Since(before),
	}
	w.cfg.logStreamSend(w.Context(), w.role, w.call, streamInfo, w.pr, sendPayload, result)
	return err
}

func wrapServerStream(ss grpc.ServerStream, cfg *config, role Role, before time.Time, call Call, pr Peer) *serverStream {
	return &serverStream{
		ServerStream: ss,
		cfg:          cfg,
		role:         role,
		start:        before,
		call:         call,
		pr:           pr,
	}
}

// SlogStreamServerInterceptor returns a grpc.StreamServerInterceptor suitable
// for use in a grpc.NewServer call.
// This interceptor will log stream start, sends and receives.
func SlogStreamServerInterceptor(opts ...Option) grpc.StreamServerInterceptor {
	cfg := newConfig(opts, "server")

	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		// See if we should skip intercepting this call
		i := &otelgrpc.InterceptorInfo{
			StreamServerInfo: info,
			Type:             otelgrpc.StreamServer,
		}
		if cfg.InterceptorFilter != nil && !cfg.InterceptorFilter(i) {
			return handler(srv, ss)
		}

		role := Role{
			Role:         cfg.role,
			ClientStream: info.IsClientStream,
			ServerStream: info.IsServerStream,
		}
		pr := Peer{}
		if p, ok := peer.FromContext(ss.Context()); ok {
			pr = peerAttr(p.Addr.String())
		}
		call := parseFullMethod(info.FullMethod)

		// If client streaming, there will be many request payloads, so RecvMsg will log them all.
		// Otherwise, log the start with the first (and only) payload received.
		if role.ClientStream {
			cfg.logStreamStart(ss.Context(), role, call, pr, Payload{}, Result{}) // Empty result, since starting the stream was a success
		}

		before := time.Now()
		err := handler(srv, wrapServerStream(ss, cfg, role, before, call, pr))

		// If server streaming, there will be many sent/response payloads, so SendMsg will log them all.
		// Otherwise, log the end with the first (and only) payload sent.
		if role.ServerStream {
			result := Result{
				Error:   err,
				Elapsed: time.Since(before),
			}
			cfg.logStreamEnd(ss.Context(), role, call, pr, Payload{}, result)
		}
		return err
	}
}
