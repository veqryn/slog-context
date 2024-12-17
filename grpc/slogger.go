package sloggrpc

import (
	"context"
	"log/slog"
	"time"

	slogctx "github.com/veqryn/slog-context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func appendCode(attrs []slog.Attr, err error) (slog.Level, []slog.Attr) {
	if err != nil {
		s, _ := status.FromError(err)
		return slog.LevelWarn, append(attrs,
			slog.String("code_name", s.Code().String()),
			slog.Int("code", int(s.Code())),
			slog.String("err", s.Message()),
		)
	}
	return slog.LevelInfo, append(attrs,
		slog.String("code_name", codes.OK.String()),
		slog.Int("code", int(codes.OK)),
	)
}

func slogRequest(ctx context.Context, role Role, call Call, peer Peer, req Payload) {
	slogctx.LogAttrs(ctx, slog.LevelInfo, "rpcReq",
		slog.Any("call", call),
		slog.Any("peer", peer),
		slog.Any("req", req.Payload),
	)
}

func slogResponse(ctx context.Context, role Role, call Call, peer Peer, req Payload, resp Payload, result Result) {
	level, attrs := appendCode(make([]slog.Attr, 0, 7), result.Error)

	attrs = append(attrs,
		slog.Any("call", call),
		slog.Any("peer", peer),
		// Use floating point division here for higher precision (instead of Millisecond method).
		slog.Float64("ms", float64(result.Elapsed)/float64(time.Millisecond)),
		slog.Any("resp", resp.Payload),
	)

	slogctx.LogAttrs(ctx, level, "rpcResp", attrs...)
}

func slogStreamStart(ctx context.Context, role Role, call Call, peer Peer, req Payload, result Result) {
	if role.Role == "server" && req.Payload == nil {
		// No need to log the result, as if the server has received the start connection, it will always be good.
		slogctx.LogAttrs(ctx, slog.LevelInfo, "rpcStreamStart",
			slog.Any("call", call),
			slog.Any("peer", peer),
		)
		return
	}

	// Starting on the client side can have an error
	level, attrs := appendCode(make([]slog.Attr, 0, 7), result.Error)

	attrs = append(attrs,
		slog.Any("call", call),
		slog.Any("peer", peer),
		// Use floating point division here for higher precision (instead of Millisecond method).
		slog.Float64("ms", float64(result.Elapsed)/float64(time.Millisecond)),
	)
	if req.Payload != nil {
		attrs = append(attrs, slog.Any("req", req.Payload))
	}

	slogctx.LogAttrs(ctx, level, "rpcStreamStart", attrs...)
}

func slogStreamClientSendClosed(ctx context.Context, role Role, call Call, peer Peer, result Result) {
	// In full bidirectional streaming, clients can decide whether to end the
	// sending separately from getting an EOF to stop receiving. So log this.
	// In non-bidirectional, they always both close at the same time, so do NOT log this.
	if !role.ClientStream || !role.ServerStream {
		return
	}

	level, attrs := appendCode(make([]slog.Attr, 0, 6), result.Error)

	attrs = append(attrs,
		slog.Any("call", call),
		slog.Any("peer", peer),
		// Use floating point division here for higher precision (instead of Millisecond method).
		slog.Float64("ms", float64(result.Elapsed)/float64(time.Millisecond)),
	)
	slogctx.LogAttrs(ctx, level, "rpcStreamClientSendClosed", attrs...)
}

func slogStreamEnd(ctx context.Context, role Role, call Call, peer Peer, resp Payload, result Result) {
	level, attrs := appendCode(make([]slog.Attr, 0, 7), result.Error)

	attrs = append(attrs,
		slog.Any("call", call),
		slog.Any("peer", peer),
		// Use floating point division here for higher precision (instead of Millisecond method).
		slog.Float64("ms", float64(result.Elapsed)/float64(time.Millisecond)),
	)
	if resp.Payload != nil {
		attrs = append(attrs, slog.Any("req", resp.Payload))
	}

	slogctx.LogAttrs(ctx, level, "rpcStreamEnd", attrs...)
}

func slogStreamSend(ctx context.Context, role Role, call Call, desc StreamInfo, peer Peer, req Payload, result Result) {
	level, attrs := appendCode(make([]slog.Attr, 0, 8), result.Error)

	attrs = append(attrs,
		slog.Any("call", call),
		slog.Any("desc", desc),
		slog.Any("peer", peer),
		// Use floating point division here for higher precision (instead of Millisecond method).
		slog.Float64("ms", float64(result.Elapsed)/float64(time.Millisecond)),
		slog.Any("req", req.Payload),
	)

	slogctx.LogAttrs(ctx, level, "rpcStreamSend", attrs...)
}

func slogStreamRecv(ctx context.Context, role Role, call Call, desc StreamInfo, peer Peer, resp Payload, result Result) {
	level, attrs := appendCode(make([]slog.Attr, 0, 8), result.Error)

	attrs = append(attrs,
		slog.Any("call", call),
		slog.Any("desc", desc),
		slog.Any("peer", peer),
		// Use floating point division here for higher precision (instead of Millisecond method).
		slog.Float64("ms", float64(result.Elapsed)/float64(time.Millisecond)),
		slog.Any("resp", resp.Payload),
	)

	slogctx.LogAttrs(ctx, level, "rpcStreamRecv", attrs...)
}
