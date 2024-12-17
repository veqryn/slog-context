package sloggrpc

import (
	"encoding/json"
	"log/slog"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func ReplaceAttrJsonPB(group []string, a slog.Attr) slog.Attr {
	// Specifically use protobuf->json defined by the protobuf spec
	// for any protobuf messages.
	if a.Value.Kind() == slog.KindAny {
		if pbm, ok := a.Value.Any().(proto.Message); ok {
			return slog.Any(a.Key, JsonPB(pbm))
		}
	}
	return a
}

// JsonPB turns a protobuf message into json, and should be used when logging a
// protobuf message, instead of the "encoding/json" package.
// It will marshal the protobuf using jsonpb (the official Proto<->JSON spec),
// instead of a json package that doesn't know how to handle well known types.
func JsonPB(m proto.Message) any {
	jsonpb, err := (&protojson.MarshalOptions{UseProtoNames: true}).Marshal(m)
	if err != nil {
		return m // Let the default log handler deal with it
	}
	// jsontext.Value tells the json marshaller it has already
	// been marshalled to bytes
	return json.RawMessage(jsonpb)
}

// TODO: have a way to truncate a protobuf message but keep it valid json
// TODO: have a way to not log the protobuf message resp, or req, separately, and maybe only on some calls. maybe log each field as its own attribute to enable this.
// TODO: have a way to replace the log functions
// TODO: tests
