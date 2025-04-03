package sloggrpc

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (c *config) appendCode(attrs []slog.Attr, err error) (slog.Level, []slog.Attr) {
	if err != nil {
		// TODO: allow setting the level by function
		s, _ := status.FromError(err)
		attrs = c.AppendToAttributes(attrs, slog.String("code_name", s.Code().String()))
		attrs = c.AppendToAttributes(attrs, slog.Int("code", int(s.Code())))
		attrs = c.AppendToAttributes(attrs, slog.String("err", s.Message()))
		return c.ErrorToLevel(err), attrs
	}
	attrs = c.AppendToAttributes(attrs, slog.String("code_name", codes.OK.String()))
	attrs = c.AppendToAttributes(attrs, slog.Int("code", int(codes.OK)))
	return slog.LevelInfo, attrs
}

func (c *config) appendCommon(attrs []slog.Attr, role Role, call Call, peer Peer) []slog.Attr {
	attrs = c.AppendToAttributes(attrs, slog.Any("grpc_system", call.System))
	attrs = c.AppendToAttributes(attrs, slog.Any("grpc_pkg", call.Package))
	attrs = c.AppendToAttributes(attrs, slog.Any("grpc_svc", call.Service))
	attrs = c.AppendToAttributes(attrs, slog.Any("grpc_method", call.Method))
	attrs = c.AppendToAttributes(attrs, slog.Any("role", role.Role))
	attrs = c.AppendToAttributes(attrs, slog.Any("stream_server", role.ServerStream))
	attrs = c.AppendToAttributes(attrs, slog.Any("stream_client", role.ClientStream))
	attrs = c.AppendToAttributes(attrs, slog.Any("peer_host", peer.Host))
	attrs = c.AppendToAttributes(attrs, slog.Any("peer_port", peer.Port))
	return attrs
}

func (c *config) appendDurationElapsed(attrs []slog.Attr, durationElapsed time.Duration) []slog.Attr {
	// Use floating point division here for higher precision (instead of Millisecond method).
	return c.AppendToAttributes(attrs, slog.Float64("ms", float64(durationElapsed)/float64(time.Millisecond)))
}

func (c *config) appendStreamInfo(attrs []slog.Attr, streamInfo StreamInfo) []slog.Attr {
	return c.AppendToAttributes(attrs, slog.Any("desc", streamInfo))
}

func (c *config) appendPayload(attrs []slog.Attr, key string, payload Payload) []slog.Attr {
	return c.AppendToAttributes(attrs, slog.Any(key, payload.Payload))
}

func (c *config) logRequest(ctx context.Context, role Role, call Call, peer Peer, req Payload) {
	attrs := c.appendCommon(make([]slog.Attr, 0, 10), role, call, peer)
	attrs = c.appendPayload(attrs, "req", req)

	c.log(ctx, slog.LevelInfo, "rpcReq", attrs...)
}

func (c *config) logResponse(ctx context.Context, role Role, call Call, peer Peer, req Payload, resp Payload, result Result) {
	level, attrs := c.appendCode(make([]slog.Attr, 0, 14), result.Error)
	attrs = c.appendCommon(attrs, role, call, peer)
	attrs = c.appendDurationElapsed(attrs, result.Elapsed)
	attrs = c.appendPayload(attrs, "resp", resp)

	c.log(ctx, level, "rpcResp", attrs...)
}

func (c *config) logStreamStart(ctx context.Context, role Role, call Call, peer Peer, req Payload, result Result) {
	if role.Role == "server" && req.Payload == nil {
		// No need to log the result code/payload, because if the server has
		// received the start connection, it will always be good.
		attrs := c.appendCommon(make([]slog.Attr, 0, 9), role, call, peer)
		c.log(ctx, slog.LevelInfo, "rpcStreamStart", attrs...)
		return
	}

	// Starting on the client side can have an error
	level, attrs := c.appendCode(make([]slog.Attr, 0, 14), result.Error)
	attrs = c.appendCommon(attrs, role, call, peer)
	attrs = c.appendDurationElapsed(attrs, result.Elapsed)
	if req.Payload != nil {
		attrs = c.appendPayload(attrs, "req", req)
	}

	c.log(ctx, level, "rpcStreamStart", attrs...)
}

func (c *config) logStreamClientSendClosed(ctx context.Context, role Role, call Call, peer Peer, result Result) {
	// In full bidirectional streaming, clients can decide whether to end the
	// sending separately from getting an EOF to stop receiving. So log this.
	// In non-bidirectional, they always both close at the same time, so do NOT log this.
	if !role.ClientStream || !role.ServerStream {
		return
	}

	level, attrs := c.appendCode(make([]slog.Attr, 0, 13), result.Error)
	attrs = c.appendCommon(attrs, role, call, peer)
	attrs = c.appendDurationElapsed(attrs, result.Elapsed)

	c.log(ctx, level-4, "rpcStreamClientSendClosed", attrs...)
}

func (c *config) logStreamEnd(ctx context.Context, role Role, call Call, peer Peer, resp Payload, result Result) {
	level, attrs := c.appendCode(make([]slog.Attr, 0, 14), result.Error)
	attrs = c.appendCommon(attrs, role, call, peer)
	attrs = c.appendDurationElapsed(attrs, result.Elapsed)
	if resp.Payload != nil {
		attrs = c.appendPayload(attrs, "resp", resp)
	}

	c.log(ctx, level, "rpcStreamEnd", attrs...)
}

func (c *config) logStreamSend(ctx context.Context, role Role, call Call, streamInfo StreamInfo, peer Peer, req Payload, result Result) {
	level, attrs := c.appendCode(make([]slog.Attr, 0, 15), result.Error)
	attrs = c.appendCommon(attrs, role, call, peer)
	attrs = c.appendStreamInfo(attrs, streamInfo)
	attrs = c.appendDurationElapsed(attrs, result.Elapsed)

	key := "resp"
	if role.Role == "client" {
		key = "req"
	}
	attrs = c.appendPayload(attrs, key, req)

	c.log(ctx, level, "rpcStreamSend", attrs...)
}

func (c *config) logStreamRecv(ctx context.Context, role Role, call Call, streamInfo StreamInfo, peer Peer, resp Payload, result Result) {
	level, attrs := c.appendCode(make([]slog.Attr, 0, 15), result.Error)
	attrs = c.appendCommon(attrs, role, call, peer)
	attrs = c.appendStreamInfo(attrs, streamInfo)
	attrs = c.appendDurationElapsed(attrs, result.Elapsed)

	key := "req"
	if role.Role == "client" {
		key = "resp"
	}
	attrs = c.appendPayload(attrs, key, resp)

	c.log(ctx, level, "rpcStreamRecv", attrs...)
}
