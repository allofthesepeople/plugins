// Code generated by goa v2.0.0-wip, DO NOT EDIT.
//
// calc gRPC client encoders and decoders
//
// Command:
// $ goa gen goa.design/plugins/goakit/examples/calc/design -o
// $(GOPATH)/src/goa.design/plugins/goakit/examples/calc

package client

import (
	"context"

	goagrpc "goa.design/goa/grpc"
	calc "goa.design/plugins/goakit/examples/calc/gen/calc"
	calcpb "goa.design/plugins/goakit/examples/calc/gen/grpc/calc/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// BuildAddFunc builds the remote method to invoke for "calc" service "add"
// endpoint.
func BuildAddFunc(grpccli calcpb.CalcClient, cliopts ...grpc.CallOption) goagrpc.RemoteFunc {
	return func(ctx context.Context, reqpb interface{}, opts ...grpc.CallOption) (interface{}, error) {
		for _, opt := range cliopts {
			opts = append(opts, opt)
		}
		return grpccli.Add(ctx, reqpb.(*calcpb.AddRequest), opts...)
	}
}

// EncodeAddRequest encodes requests sent to calc add endpoint.
func EncodeAddRequest(ctx context.Context, v interface{}, md *metadata.MD) (interface{}, error) {
	payload, ok := v.(*calc.AddPayload)
	if !ok {
		return nil, goagrpc.ErrInvalidType("calc", "add", "*calc.AddPayload", v)
	}
	return NewAddRequest(payload), nil
}

// DecodeAddResponse decodes responses from the calc add endpoint.
func DecodeAddResponse(ctx context.Context, v interface{}, hdr, trlr metadata.MD) (interface{}, error) {
	message, ok := v.(*calcpb.AddResponse)
	if !ok {
		return nil, goagrpc.ErrInvalidType("calc", "add", "*calcpb.AddResponse", v)
	}
	res := NewAddResult(message)
	return res, nil
}
