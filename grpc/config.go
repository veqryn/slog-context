package sloggrpc

import (
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
	InterceptorFilter InterceptorFilter
	role              string
}

// Option applies an option value for a config.
type Option interface {
	apply(*config)
}

// newConfig returns a config configured with all the passed Options.
func newConfig(opts []Option, role string) *config {
	c := &config{
		role: role,
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
