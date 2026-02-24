package rawproto

import (
	"context"
	"io"

	encodingapi "go.uber.org/yarpc/api/encoding"
	"go.uber.org/yarpc/api/transport"
)

// Encoding is the wire encoding this package registers under.
const Encoding transport.Encoding = "proto"

// UnaryHandler handles a single unary request as raw bytes.
type UnaryHandler func(ctx context.Context, body []byte) ([]byte, error)

// Procedure builds a transport.Procedure that accepts proto-encoded
// requests but skips all protobuf marshal/unmarshal.
func Procedure(name string, handler UnaryHandler) []transport.Procedure {
	return []transport.Procedure{
		{
			Name:        name,
			Encoding:    Encoding,
			HandlerSpec: transport.NewUnaryHandlerSpec(rawProtoHandler{handler}),
		},
	}
}

// OnewayHandler handles a single oneway request as raw bytes.
type OnewayHandler func(ctx context.Context, body []byte) error

// OnewayProcedure builds a oneway transport.Procedure under proto encoding.
func OnewayProcedure(name string, handler OnewayHandler) []transport.Procedure {
	return []transport.Procedure{
		{
			Name:        name,
			Encoding:    Encoding,
			HandlerSpec: transport.NewOnewayHandlerSpec(rawProtoOnewayHandler{handler}),
		},
	}
}

type rawProtoHandler struct{ handler UnaryHandler }

func (h rawProtoHandler) Handle(ctx context.Context, req *transport.Request, rw transport.ResponseWriter) error {
	ctx, call := encodingapi.NewInboundCall(ctx)
	if err := call.ReadFromRequest(req); err != nil {
		return err
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return err
	}

	resBody, appErr := h.handler(ctx, body)

	if err := call.WriteToResponse(rw); err != nil {
		return err
	}

	if len(resBody) > 0 {
		if _, err := rw.Write(resBody); err != nil {
			return err
		}
	}

	if appErr != nil {
		rw.SetApplicationError()
		return appErr
	}
	return nil
}

type rawProtoOnewayHandler struct{ handler OnewayHandler }

func (h rawProtoOnewayHandler) HandleOneway(ctx context.Context, req *transport.Request) error {
	ctx, call := encodingapi.NewInboundCall(ctx)
	if err := call.ReadFromRequest(req); err != nil {
		return err
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return err
	}

	return h.handler(ctx, body)
}
