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
	// Closure over config
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
		cfg.logRequest(ctx, cfg.role, call, pr, reqPayload)

		before := time.Now()

		// Call the next interceptor or the actual handler
		resp, err := handler(ctx, req)

		respPayload := Payload{Payload: resp}

		result := Result{
			Error:   err,
			Elapsed: time.Since(before),
		}

		// Log the response
		cfg.logResponse(ctx, cfg.role, call, pr, reqPayload, respPayload, result)

		return resp, err
	}
}

// SlogUnaryClientInterceptor returns a grpc.UnaryClientInterceptor suitable
// for use in a grpc.NewClient or grpc.WithChainUnaryInterceptor call.
// This interceptor will log requests and responses.
func SlogUnaryClientInterceptor(opts ...Option) grpc.UnaryClientInterceptor {
	// Closure over config
	cfg := newConfig(opts, "server")

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
		cfg.logRequest(ctx, cfg.role, call, pr, reqPayload)

		before := time.Now()

		// Call the next interceptor or the actual invocation
		err := invoker(ctx, method, req, resp, cc, callOpts...)

		respPayload := Payload{Payload: resp}

		result := Result{
			Error:   err,
			Elapsed: time.Since(before),
		}

		// Log the response
		cfg.logResponse(ctx, cfg.role, call, pr, reqPayload, respPayload, result)

		return err
	}
}
