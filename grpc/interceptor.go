package sloggrpc

import (
	"context"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

// Based on or similar to code from:
// https://github.com/open-telemetry/opentelemetry-go-contrib
// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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

		// Log the request
		if cfg.logRequest != nil {
			cfg.logRequest(ctx, cfg.role, call, pr, reqPayload)
		}

		// Call the next interceptor or the actual handler
		before := time.Now()
		resp, err := handler(ctx, req)

		// Log the response
		if cfg.logResponse != nil {
			respPayload := Payload{Payload: resp}
			result := Result{
				Error:   err,
				Elapsed: time.Since(before),
			}
			cfg.logResponse(ctx, cfg.role, call, pr, reqPayload, respPayload, result)
		}

		return resp, err
	}
}

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

		// Log the request
		if cfg.logRequest != nil {
			cfg.logRequest(ctx, cfg.role, call, pr, reqPayload)
		}

		// Call the next interceptor or the actual invocation
		before := time.Now()
		err := invoker(ctx, method, req, resp, cc, callOpts...)

		// Log the response
		if cfg.logResponse != nil {
			respPayload := Payload{Payload: resp}
			result := Result{
				Error:   err,
				Elapsed: time.Since(before),
			}
			cfg.logResponse(ctx, cfg.role, call, pr, reqPayload, respPayload, result)
		}

		return err
	}
}

// clientStream  wraps around the embedded grpc.ClientStream, and intercepts the RecvMsg and
// SendMsg method call.
type clientStream struct {
	grpc.ClientStream
	desc *grpc.StreamDesc

	cfg               *config
	call              Call
	pr                Peer
	receivedMessageID int
	sentMessageID     int
}

func (w *clientStream) RecvMsg(m any) error {
	before := time.Now()
	err := w.ClientStream.RecvMsg(m)
	w.receivedMessageID++

	if w.cfg.logStreamRecv != nil {
		streamInfo := StreamInfo{MsgID: w.receivedMessageID}
		recvPayload := Payload{Payload: m}
		result := Result{
			Error:   err,
			Elapsed: time.Since(before),
		}
		w.cfg.logStreamRecv(w.Context(), w.cfg.role, w.call, streamInfo, w.pr, recvPayload, result)
	}
	return err
}

func (w *clientStream) SendMsg(m any) error {
	before := time.Now()
	err := w.ClientStream.SendMsg(m)
	w.sentMessageID++

	if w.cfg.logStreamSend != nil {
		streamInfo := StreamInfo{MsgID: w.sentMessageID}
		sendPayload := Payload{Payload: m}
		result := Result{
			Error:   err,
			Elapsed: time.Since(before),
		}
		w.cfg.logStreamSend(w.Context(), w.cfg.role, w.call, streamInfo, w.pr, sendPayload, result)
	}
	return err
}

func wrapClientStream(s grpc.ClientStream, desc *grpc.StreamDesc, cfg *config, call Call, pr Peer) *clientStream {
	return &clientStream{
		ClientStream: s,
		desc:         desc,
		cfg:          cfg,
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

		pr := peerAttr(cc.Target())
		call := parseFullMethod(method)

		before := time.Now()
		s, err := streamer(ctx, desc, cc, method, callOpts...)

		if cfg.logStreamStart != nil {
			result := Result{
				Error:   err,
				Elapsed: time.Since(before),
			}
			cfg.logStreamStart(ctx, cfg.role, call, pr, result)
		}

		if err != nil {
			return s, err
		}
		return wrapClientStream(s, desc, cfg, call, pr), nil
	}
}
