package sloggrpc

import (
	"context"
	"log/slog"
	"time"

	slogctx "github.com/veqryn/slog-context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// slogRequest logs a grpc request being received/sent
func slogRequest(ctx context.Context, role string, call Call, peer Peer, req Payload) {
	slogctx.LogAttrs(ctx, slog.LevelInfo, "rpcReq",
		slog.Any("call", call),
		slog.Any("peer", peer),
		slog.Any("req", req.Payload),
	)
}

// slogResponse logs a grpc response being sent/received
func slogResponse(ctx context.Context, role string, call Call, peer Peer, req Payload, resp Payload, result Result) {
	level := slog.LevelInfo

	attrs := make([]slog.Attr, 0, 7)
	if result.Error != nil {
		level = slog.LevelWarn
		s, _ := status.FromError(result.Error)
		attrs = append(attrs, slog.String("code_name", s.Code().String()))
		attrs = append(attrs, slog.Int("code", int(s.Code())))
		attrs = append(attrs, slog.String("err", s.Message()))
	} else {
		attrs = append(attrs, slog.String("code_name", codes.OK.String()))
		attrs = append(attrs, slog.Int("code", int(codes.OK)))
	}

	// Use floating point division here for higher precision (instead of Millisecond method).
	attrs = append(attrs, slog.Float64("ms", float64(result.Elapsed)/float64(time.Millisecond)))

	attrs = append(attrs, slog.Any("call", call))
	attrs = append(attrs, slog.Any("peer", peer))
	attrs = append(attrs, slog.Any("resp", resp.Payload))

	slogctx.LogAttrs(ctx, level, "rpcResp", attrs...)
}
