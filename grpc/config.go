package sloggrpc

import (
	"context"
	"io"
	"log/slog"

	slogctx "github.com/veqryn/slog-context"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Based on or similar to code from:
// https://github.com/open-telemetry/opentelemetry-go-contrib
// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// InterceptorFilter is a predicate used to determine whether a given request in
// interceptor info should be instrumented. A InterceptorFilter must return true if
// the request should be traced.
type InterceptorFilter func(*otelgrpc.InterceptorInfo) bool

// AppendToAttributes allows customizing the attributes, including disabling some
type AppendToAttributes func(attrs []slog.Attr, attr slog.Attr) []slog.Attr

// ErrorToLevel defines the mapping between the gRPC return error/code to a log level
type ErrorToLevel func(err error) slog.Level

// Logger defines the logging function to use for the interceptor (same signature as slog.LogAttrs)
type Logger func(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr)

// config is a group of options for this instrumentation.
type config struct {
	InterceptorFilter  InterceptorFilter
	AppendToAttributes AppendToAttributes
	ErrorToLevel       ErrorToLevel
	role               string
	Logger             Logger
}

// Option applies an option value for a config.
type Option interface {
	apply(*config)
}

// newConfig returns a config configured with all the passed Options.
func newConfig(opts []Option, role string) *config {
	c := &config{
		AppendToAttributes: AppendToAttributesDefault,
		Logger:             LoggerDefault,
		role:               role,
	}
	if role == "client" {
		c.ErrorToLevel = ErrorToLevelClientDefault
	} else {
		c.ErrorToLevel = ErrorToLevelServerDefault
	}

	for _, o := range opts {
		o.apply(c)
	}
	return c
}

// ErrorToLevelServerDefault is the helper mapper that maps gRPC return errors/codes to log levels for server side.
func ErrorToLevelServerDefault(err error) slog.Level {
	if err == nil {
		return slog.LevelInfo
	}
	if err == io.EOF {
		return slog.LevelWarn
	}

	s, _ := status.FromError(err)
	switch s.Code() {
	case codes.OK, codes.NotFound, codes.Canceled, codes.AlreadyExists, codes.InvalidArgument, codes.Unauthenticated:
		return slog.LevelInfo

	case codes.DeadlineExceeded, codes.PermissionDenied, codes.ResourceExhausted, codes.FailedPrecondition, codes.Aborted,
		codes.OutOfRange, codes.Unavailable:
		return slog.LevelWarn

	case codes.Unknown, codes.Unimplemented, codes.Internal, codes.DataLoss:
		return slog.LevelError

	default:
		return slog.LevelError
	}
}

// ErrorToLevelClientDefault is the helper mapper that maps gRPC return errors/codes to log levels for client side.
func ErrorToLevelClientDefault(err error) slog.Level {
	if err == nil {
		return slog.LevelInfo
	}
	if err == io.EOF {
		return slog.LevelInfo
	}

	s, _ := status.FromError(err)
	switch s.Code() {
	case codes.OK, codes.Canceled, codes.InvalidArgument, codes.NotFound, codes.AlreadyExists, codes.ResourceExhausted,
		codes.FailedPrecondition, codes.Aborted, codes.OutOfRange:
		return slog.LevelInfo

	case codes.Unknown, codes.DeadlineExceeded, codes.PermissionDenied, codes.Unauthenticated:
		return slog.LevelWarn

	case codes.Unimplemented, codes.Internal, codes.Unavailable, codes.DataLoss:
		return slog.LevelWarn // Maybe make this error level?

	default:
		return slog.LevelWarn
	}
}

// AppendToAttributesAll allows all attributes
var AppendToAttributesAll AppendToAttributes = func(attrs []slog.Attr, attr slog.Attr) []slog.Attr {
	return append(attrs, attr)
}

// AppendToAttributesDefault allows the default attributes
var AppendToAttributesDefault AppendToAttributes = disableFields{
	"grpc_pkg":      {},
	"grpc_system":   {},
	"role":          {},
	"stream_server": {},
	"stream_client": {},
}.appendToAttrs

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

// WithErrorToLevel returns an Option to use the error to level function
func WithErrorToLevel(f ErrorToLevel) Option {
	return interceptorErrorToLevelOption{f: f}
}

type interceptorErrorToLevelOption struct {
	f ErrorToLevel
}

func (o interceptorErrorToLevelOption) apply(c *config) {
	if o.f != nil {
		c.ErrorToLevel = o.f
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

// LoggerDefault returns slogctx.LogAttrs as the default Logger function
var LoggerDefault Logger = slogctx.LogAttrs

// WithLogger returns an Option to use the logger function
func WithLogger(f Logger) Option {
	return interceptorLoggerOption{f: f}
}

type interceptorLoggerOption struct {
	f Logger
}

func (o interceptorLoggerOption) apply(c *config) {
	if o.f != nil {
		c.Logger = o.f
	}
}
