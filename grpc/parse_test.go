package sloggrpc

import (
	"testing"
)

// Based on or similar to code from:
// https://github.com/open-telemetry/opentelemetry-go-contrib
// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

func TestParseFullMethod(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input    string
		expected Call
	}{
		{
			input:    "",
			expected: Call{},
		},
		{
			input: "/",
			expected: Call{
				Package: "",
				Service: "/",
				Method:  "",
			},
		},
		{
			input: "/slash_but_no_second_slash",
			expected: Call{
				Package: "",
				Service: "/slash_but_no_second_slash",
				Method:  "",
			},
		},
		{
			input: "/service_only/",
			expected: Call{
				Package: "",
				Service: "service_only",
				Method:  "",
			},
		},
		{
			input: "//method_only",
			expected: Call{
				Package: "",
				Service: "",
				Method:  "method_only",
			},
		},
		{
			input: "/service/method",
			expected: Call{
				Package: "",
				Service: "service",
				Method:  "method",
			},
		},
		{
			input: "/package.lib.service/method",
			expected: Call{
				Package: "package.lib",
				Service: "service",
				Method:  "method",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			call := parseFullMethod(tc.input)
			if call != tc.expected {
				t.Errorf("Got: %v; Want: %v", call, tc.expected)
			}
		})
	}
}

func TestPeerAttr(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input    string
		expected Peer
	}{
		{
			input:    "",
			expected: Peer{},
		},
		{
			input: "1.1.1.1:8080",
			expected: Peer{
				Host: "1.1.1.1",
				Port: 8080,
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			peer := peerAttr(tc.input)
			if peer != tc.expected {
				t.Errorf("Got: %v; Want: %v", peer, tc.expected)
			}
		})
	}
}
