package rpc

import (
	"context"
	"fmt"
	"net"
	"time"

	"git.sr.ht/~spc/go-log"
	pb "github.com/redhatinsights/yggdrasil/protocol/grpc"
	"google.golang.org/grpc"
)

// grpcServer is an implementation of the Dispatcher gRPC server interface.
type grpcServer struct {
	pb.UnimplementedDispatcherServer
	outbound chan request
}

// Send implements the gRPC Dispatcher Send method.
//
// gRPC Data protobuf messages are received by this function, converted into
// a rpc.request struct and sent to a channel. The routine then attempts to
// receive a value from the request's response channel, timing out after 1
// second.
func (g *grpcServer) Send(ctx context.Context, r *pb.Data) (*pb.Receipt, error) {
	log.Debugf("grpc-server-Send received message-id: %v", r.GetMessageId())
	ch := make(chan response)
	g.outbound <- request{
		destination: r.GetDirective(),
		id:          r.GetMessageId(),
		metadata:    r.GetMetadata(),
		data:        r.GetContent(),
		response:    ch,
	}
	log.Debugf("grpc-server-Send sent message request with id: %v", r.GetMessageId())

	select {
	case resp := <-ch:
		log.Debugf("received response: %v", resp)
	case <-time.After(1 * time.Second):
		log.Infof("grpc-server-Send no response received on channel")
	}
	close(ch)

	return &pb.Receipt{}, nil
}

// GRPCTransport implements the rpc.Transporter interface using gRPC as an
// underlying RPC protocol.
type GRPCTransport struct {
	server       *grpcServer
	relayHandler RelayHandler
}

func NewGRPCTransport() *GRPCTransport {
	return &GRPCTransport{
		server: &grpcServer{
			outbound: make(chan request),
		},
	}
}

// Dispatch implements the rpc.Transporter Dispatch method.
func (g *GRPCTransport) Dispatch(addr string, id string, metadata map[string]string, data []byte) error {
	conn, err := grpc.Dial("unix:"+addr, grpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("cannot connect to gRPC address %v: %v", addr, err)
	}
	defer conn.Close()
	log.Debugf("grpc-transport-Dispatch worker dialed on address %v", "unix:"+addr)

	c := pb.NewWorkerClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	msg := pb.Data{
		MessageId:  id,
		ResponseTo: "",
		Directive:  "",
		Metadata:   metadata,
		Content:    data,
	}
	_, err = c.Send(ctx, &msg)
	if err != nil {
		return fmt.Errorf("cannot send message: %v", err)
	}

	log.Debugf("grpc-transport-Dispatch dispatched message-id: %v", id)

	return nil
}

// SetRelayHandler implements the rpc.Transporter interface's Rx method.
func (g *GRPCTransport) SetRelayHandler(f RelayHandler) error {
	g.relayHandler = f
	return nil
}

// NotifyEvent implements the rpc.Transporter interface's NotifyEvent method. It sends
// the specified event name to the worker at the given address.
func (g *GRPCTransport) NotifyEvent(addr string, name RPCEvent, detail string, metadata map[string]string) error {
	return fmt.Errorf("not implemented")
}

// Listen implements the rpc.Transporter interface's Listen method. It creates a
// new gRPC server, registers an object as a dispatcher service and starts
// serving requests on the given address.
func (g *GRPCTransport) Listen(addr string) error {
	server := grpc.NewServer()

	pb.RegisterDispatcherServer(server, g.server)
	listener, err := net.Listen("unix", addr)
	if err != nil {
		return fmt.Errorf("cannot listen to socket: %w", err)
	}

	log.Infof("gRPC transport listening on socket: %v\n", addr)

	go func() {
		for r := range g.server.outbound {
			log.Debugf("grpc-transport-RelayHandler received message-id: %v", r.id)
			code, metadata, data, err := g.relayHandler(r.destination, r.id, r.metadata, r.data)
			if err != nil {
				log.Errorf("cannot call handlerFunc: %v", err)
				continue
			}
			resp := response{
				code:     code,
				metadata: metadata,
				data:     data,
			}
			r.response <- resp
			log.Debugf("grpc-transport-RelayHandler sent response to message-id: %v", r.id)
			log.Tracef("grpc-transport-RelayHandler response: %v", resp)
		}
	}()

	return server.Serve(listener)
}
