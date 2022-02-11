// Code generated by github.com/varlink/go/cmd/varlink-go-interface-generator, DO NOT EDIT.

// Interface to interact with yggd, the dispatcher service.
package comredhatyggdrasildispatcher

import (
	"context"
	"encoding/json"
	"github.com/varlink/go/varlink"
)

// Generated type declarations

// The destination could not be handled by the dispatch service.
type InvalidDestination struct{}

func (e InvalidDestination) Error() string {
	s := "com.redhat.yggdrasil.dispatcher.InvalidDestination"
	return s
}

func Dispatch_Error(err error) error {
	if e, ok := err.(*varlink.Error); ok {
		switch e.Name {
		case "com.redhat.yggdrasil.dispatcher.InvalidDestination":
			errorRawParameters := e.Parameters.(*json.RawMessage)
			if errorRawParameters == nil {
				return e
			}
			var param InvalidDestination
			err := json.Unmarshal(*errorRawParameters, &param)
			if err != nil {
				return e
			}
			return &param
		}
	}
	return err
}

// Generated client method calls

// Relay sends data to the specified destination.
type Relay_methods struct{}

func Relay() Relay_methods { return Relay_methods{} }

func (m Relay_methods) Call(ctx context.Context, c *varlink.Connection, destination_in_ string, id_in_ string, metadata_in_ map[string]string, data_in_ json.RawMessage) (code_out_ *int64, metadata_out_ *map[string]string, data_out_ *json.RawMessage, err_ error) {
	receive, err_ := m.Send(ctx, c, 0, destination_in_, id_in_, metadata_in_, data_in_)
	if err_ != nil {
		return
	}
	code_out_, metadata_out_, data_out_, _, err_ = receive(ctx)
	return
}

func (m Relay_methods) Send(ctx context.Context, c *varlink.Connection, flags uint64, destination_in_ string, id_in_ string, metadata_in_ map[string]string, data_in_ json.RawMessage) (func(ctx context.Context) (*int64, *map[string]string, *json.RawMessage, uint64, error), error) {
	var in struct {
		Destination string            `json:"destination"`
		Id          string            `json:"id"`
		Metadata    map[string]string `json:"metadata"`
		Data        json.RawMessage   `json:"data"`
	}
	in.Destination = destination_in_
	in.Id = id_in_
	in.Metadata = map[string]string(metadata_in_)
	in.Data = data_in_
	receive, err := c.Send(ctx, "com.redhat.yggdrasil.dispatcher.Relay", in, flags)
	if err != nil {
		return nil, err
	}
	return func(context.Context) (code_out_ *int64, metadata_out_ *map[string]string, data_out_ *json.RawMessage, flags uint64, err error) {
		var out struct {
			Code     *int64             `json:"code,omitempty"`
			Metadata *map[string]string `json:"metadata,omitempty"`
			Data     *json.RawMessage   `json:"data,omitempty"`
		}
		flags, err = receive(ctx, &out)
		if err != nil {
			err = Dispatch_Error(err)
			return
		}
		code_out_ = out.Code
		metadata_out_ = out.Metadata
		data_out_ = out.Data
		return
	}, nil
}

func (m Relay_methods) Upgrade(ctx context.Context, c *varlink.Connection, destination_in_ string, id_in_ string, metadata_in_ map[string]string, data_in_ json.RawMessage) (func(ctx context.Context) (code_out_ *int64, metadata_out_ *map[string]string, data_out_ *json.RawMessage, flags uint64, conn varlink.ReadWriterContext, err_ error), error) {
	var in struct {
		Destination string            `json:"destination"`
		Id          string            `json:"id"`
		Metadata    map[string]string `json:"metadata"`
		Data        json.RawMessage   `json:"data"`
	}
	in.Destination = destination_in_
	in.Id = id_in_
	in.Metadata = map[string]string(metadata_in_)
	in.Data = data_in_
	receive, err := c.Upgrade(ctx, "com.redhat.yggdrasil.dispatcher.Relay", in)
	if err != nil {
		return nil, err
	}
	return func(context.Context) (code_out_ *int64, metadata_out_ *map[string]string, data_out_ *json.RawMessage, flags uint64, conn varlink.ReadWriterContext, err error) {
		var out struct {
			Code     *int64             `json:"code,omitempty"`
			Metadata *map[string]string `json:"metadata,omitempty"`
			Data     *json.RawMessage   `json:"data,omitempty"`
		}
		flags, conn, err = receive(ctx, &out)
		if err != nil {
			err = Dispatch_Error(err)
			return
		}
		code_out_ = out.Code
		metadata_out_ = out.Metadata
		data_out_ = out.Data
		return
	}, nil
}

// Generated service interface with all methods

type comredhatyggdrasildispatcherInterface interface {
	Relay(ctx context.Context, c VarlinkCall, destination_ string, id_ string, metadata_ map[string]string, data_ json.RawMessage) error
}

// Generated service object with all methods

type VarlinkCall struct{ varlink.Call }

// Generated reply methods for all varlink errors

// The destination could not be handled by the dispatch service.
func (c *VarlinkCall) ReplyInvalidDestination(ctx context.Context) error {
	var out InvalidDestination
	return c.ReplyError(ctx, "com.redhat.yggdrasil.dispatcher.InvalidDestination", &out)
}

// Generated reply methods for all varlink methods

func (c *VarlinkCall) ReplyRelay(ctx context.Context, code_ *int64, metadata_ *map[string]string, data_ *json.RawMessage) error {
	var out struct {
		Code     *int64             `json:"code,omitempty"`
		Metadata *map[string]string `json:"metadata,omitempty"`
		Data     *json.RawMessage   `json:"data,omitempty"`
	}
	out.Code = code_
	out.Metadata = metadata_
	out.Data = data_
	return c.Reply(ctx, &out)
}

// Generated dummy implementations for all varlink methods

// Relay sends data to the specified destination.
func (s *VarlinkInterface) Relay(ctx context.Context, c VarlinkCall, destination_ string, id_ string, metadata_ map[string]string, data_ json.RawMessage) error {
	return c.ReplyMethodNotImplemented(ctx, "com.redhat.yggdrasil.dispatcher.Relay")
}

// Generated method call dispatcher

func (s *VarlinkInterface) VarlinkDispatch(ctx context.Context, call varlink.Call, methodname string) error {
	switch methodname {
	case "Relay":
		var in struct {
			Destination string            `json:"destination"`
			Id          string            `json:"id"`
			Metadata    map[string]string `json:"metadata"`
			Data        json.RawMessage   `json:"data"`
		}
		err := call.GetParameters(&in)
		if err != nil {
			return call.ReplyInvalidParameter(ctx, "parameters")
		}
		return s.comredhatyggdrasildispatcherInterface.Relay(ctx, VarlinkCall{call}, in.Destination, in.Id, map[string]string(in.Metadata), in.Data)

	default:
		return call.ReplyMethodNotFound(ctx, methodname)
	}
}

// Generated varlink interface name

func (s *VarlinkInterface) VarlinkGetName() string {
	return `com.redhat.yggdrasil.dispatcher`
}

// Generated varlink interface description

func (s *VarlinkInterface) VarlinkGetDescription() string {
	return `# Interface to interact with yggd, the dispatcher service.
interface com.redhat.yggdrasil.dispatcher

# Relay sends data to the specified destination.
method Relay(destination: string, id: string, metadata: [string]string, data: object) -> (code: ?int, metadata: ?[string]string, data: ?object)

# The destination could not be handled by the dispatch service.
error InvalidDestination ()
`
}

// Generated service interface

type VarlinkInterface struct {
	comredhatyggdrasildispatcherInterface
}

func VarlinkNew(m comredhatyggdrasildispatcherInterface) *VarlinkInterface {
	return &VarlinkInterface{m}
}
