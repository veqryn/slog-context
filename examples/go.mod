module github.com/veqryn/slog-context/examples

go 1.21

require (
	github.com/veqryn/slog-context v0.5.0
	github.com/veqryn/slog-context/otel v0.3.1-0.20240103074521-0aea7a966d3b
	go.opentelemetry.io/otel v1.21.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.21.0
	go.opentelemetry.io/otel/sdk v1.21.0
	go.opentelemetry.io/otel/trace v1.21.0
)

require (
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	go.opentelemetry.io/otel/metric v1.21.0 // indirect
	golang.org/x/sys v0.15.0 // indirect
)
