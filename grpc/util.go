package sloggrpc

import (
	"encoding/json"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// JsonPB turns a protobuf message into json, and should be used when logging a
// protobuf message, instead of the "encoding/json" package.
// It will marshal the protobuf using jsonpb (the official Proto<->JSON spec),
// instead of a json package that doesn't know how to handle well known types.
func JsonPB(m proto.Message) any {
	jsonpb, err := (&protojson.MarshalOptions{UseProtoNames: true}).Marshal(m)
	if err != nil {
		return m // Let the default log handler deal with it
	}
	return json.RawMessage(jsonpb)
}
