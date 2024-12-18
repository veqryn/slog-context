package sloggrpc

import (
	"context"
	"log/slog"

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

type AppendToAttributes func(attrs []slog.Attr, attr slog.Attr) []slog.Attr

// config is a group of options for this instrumentation.
type config struct {
	InterceptorFilter  InterceptorFilter
	AppendToAttributes AppendToAttributes
	role               string
	log                func(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr)
}

// Option applies an option value for a config.
type Option interface {
	apply(*config)
}

// newConfig returns a config configured with all the passed Options.
func newConfig(opts []Option, role string) *config {
	c := &config{
		AppendToAttributes: defaultAppendToAttributes.appendToAttrs, // TODO: make into an option
		role:               role,
		log:                slog.LogAttrs, // TODO: make into an option
	}

	for _, o := range opts {
		o.apply(c)
	}
	return c
}

var defaultAppendToAttributes = disableFields{
	"grpc_pkg":      {},
	"grpc_system":   {},
	"role":          {},
	"stream_server": {},
	"stream_client": {},
}

type disableFields map[string]struct{}

func (df disableFields) appendToAttrs(attrs []slog.Attr, attr slog.Attr) []slog.Attr {
	if _, ok := df[attr.Key]; ok {
		return attrs
	}
	return append(attrs, attr)
}

// WithAppendToAttributes returns an Option to use the appending function
func WithAppendToAttributes(f AppendToAttributes) Option {
	return interceptorAppendToAttributesOption{f: f}
}

type interceptorAppendToAttributesOption struct {
	f AppendToAttributes
}

func (o interceptorAppendToAttributesOption) apply(c *config) {
	if o.f != nil {
		c.AppendToAttributes = o.f
	}
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
	return call.Service != "ServerReflection"
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
