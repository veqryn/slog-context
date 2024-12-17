package sloggrpc

import (
	"context"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

// Based on or similar to code from:
// https://github.com/open-telemetry/opentelemetry-go-contrib
// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// InterceptorFilter is a predicate used to determine whether a given request in
// interceptor info should be instrumented. A InterceptorFilter must return true if
// the request should be traced.
type InterceptorFilter func(*otelgrpc.InterceptorInfo) bool

// config is a group of options for this instrumentation.
type config struct {
	InterceptorFilter         InterceptorFilter
	role                      string
	logRequest                func(ctx context.Context, role Role, call Call, peer Peer, req Payload)
	logResponse               func(ctx context.Context, role Role, call Call, peer Peer, req Payload, resp Payload, result Result)
	logStreamStart            func(ctx context.Context, role Role, call Call, peer Peer, result Result)
	logStreamClientSendClosed func(ctx context.Context, role Role, call Call, peer Peer, result Result)
	logStreamEnd              func(ctx context.Context, role Role, call Call, peer Peer, result Result)
	logStreamSend             func(ctx context.Context, role Role, call Call, si StreamInfo, peer Peer, req Payload, result Result)
	logStreamRecv             func(ctx context.Context, role Role, call Call, si StreamInfo, peer Peer, resp Payload, result Result)
}

// Option applies an option value for a config.
type Option interface {
	apply(*config)
}

// newConfig returns a config configured with all the passed Options.
func newConfig(opts []Option, role string) *config {
	c := &config{
		role:                      role,
		logRequest:                slogRequest,
		logResponse:               slogResponse,
		logStreamStart:            slogStreamStart,
		logStreamClientSendClosed: slogStreamClientSendClosed,
		logStreamEnd:              slogStreamEnd,
		logStreamSend:             slogStreamSend,
		logStreamRecv:             slogStreamRecv,
	}

	for _, o := range opts {
		o.apply(c)
	}
	return c
}

// WithInterceptorFilter returns an Option to use the request filter.
func WithInterceptorFilter(f InterceptorFilter) Option {
	return interceptorFilterOption{f: f}
}

type interceptorFilterOption struct {
	f InterceptorFilter
}

func (o interceptorFilterOption) apply(c *config) {
	if o.f != nil {
		c.InterceptorFilter = o.f
	}
}

// InterceptorFilterIgnoreReflection returns an InterceptorFilter that will
// ignore all grpc Reflection calls.
func InterceptorFilterIgnoreReflection(ii *otelgrpc.InterceptorInfo) bool {
	call := parseFullMethod(FullMethod(ii))
	if call.Service == "ServerReflection" {
		return false
	}
	return true
}

// FullMethod returns the full grpc method name, when given an *otelgrpc.InterceptorInfo
func FullMethod(ii *otelgrpc.InterceptorInfo) string {
	if ii.UnaryServerInfo != nil && ii.UnaryServerInfo.FullMethod != "" {
		return ii.UnaryServerInfo.FullMethod
	}
	if ii.StreamServerInfo != nil && ii.StreamServerInfo.FullMethod != "" {
		return ii.StreamServerInfo.FullMethod
	}
	return ii.Method
}
