package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"git.sr.ht/~spc/go-log"
	dispatcher "github.com/redhatinsights/yggdrasil/protocol/varlink/dispatcher"
	worker "github.com/redhatinsights/yggdrasil/protocol/varlink/worker"
	"github.com/varlink/go/varlink"
)

// varlinkService is an implementation of the varlink Dispatcher interface.
type varlinkService struct {
	dispatcher.VarlinkInterface
	outbound chan request
}

// Relay implements the varlink Dispatcher Relay method.
//
// Parameter values are converted into an rpc.request struct and sent to a
// channel. The routine then attempts to receive a value from the request's
// response channel, timing out after 1 second.
func (v *varlinkService) Relay(ctx context.Context, c dispatcher.VarlinkCall, destination string, messageID string, metadata map[string]string, data json.RawMessage) error {
	log.Debugf("varlink-service-Relay received message-id: %v", messageID)
	ch := make(chan response)
	v.outbound <- request{
		destination: destination,
		id:          messageID,
		metadata:    metadata,
		data:        data,
		response:    ch,
	}
	log.Debugf("varlink-service-Relay sent message request with id: %v", messageID)

	var (
		responseCode     int64
		responseMetadata map[string]string
		responseData     json.RawMessage
	)
	select {
	case r := <-ch:
		responseCode = int64(r.code)
		responseMetadata = r.metadata
		responseData = r.data
	case <-time.After(1 * time.Second):
		log.Errorf("varlink-service-Relay no response received on channel")
	}
	close(ch)

	return c.ReplyRelay(ctx, &responseCode, &responseMetadata, &responseData)
}

// VarlinkTransport implements the rpc.Transporter interface using VARLINK as an
// underlying RPC protocol.
type VarlinkTransport struct {
	server       *varlinkService
	relayHandler RelayHandler
}

func NewVarlinkTransport() *VarlinkTransport {
	return &VarlinkTransport{
		server: &varlinkService{
			outbound: make(chan request),
		},
	}
}

// Dispatch implements the rpc.Transporter Dispatch method.
func (v *VarlinkTransport) Dispatch(addr string, id string, metadata map[string]string, data []byte) error {
	conn, err := varlink.NewConnection(context.Background(), "unix:"+addr)
	if err != nil {
		return fmt.Errorf("cannot connect to VARLINK address %v: %v", addr, err)
	}
	defer conn.Close()
	log.Debugf("varlink-transport-Dispatch connected to varlink address %v", "unix:"+addr)

	accepted, err := worker.Dispatch().Call(context.Background(), conn, id, metadata, data)
	if err != nil {
		return fmt.Errorf("cannot call dispatch method: %v", err)
	}

	if !accepted {
		return fmt.Errorf("cannot dispatch work: work rejected")
	}
	log.Debugf("varlink-transport-Dispatch dispatched message-id: %v", id)

	return nil
}

// SetRelayHandler implements the rpc.Transporter interface's Rx method.
func (v *VarlinkTransport) SetRelayHandler(f RelayHandler) error {
	v.relayHandler = f
	return nil
}

func (v *VarlinkTransport) NotifyEvent(addr string, name RPCEvent, detail string, metadata map[string]string) error {
	return fmt.Errorf("not implemented")
}

func (v *VarlinkTransport) Listen(addr string) error {
	service, err := varlink.NewService(
		"Red Hat",
		"yggdrasil",
		"1",
		"https://github.com/RedHatInsights/yggdrasil",
	)
	if err != nil {
		return fmt.Errorf("cannot create VARLINK service: %w", err)
	}
	if err := service.RegisterInterface(dispatcher.VarlinkNew(v.server)); err != nil {
		return fmt.Errorf("cannot register VARLINK interface: %w", err)
	}

	log.Infof("VARLINK transport listening on socket: unix:%v\n", addr)

	go func() {
		for r := range v.server.outbound {
			log.Debugf("varlink-transport-RelayHandler received message-id: %v", r.id)
			code, metadata, data, err := v.relayHandler(r.destination, r.id, r.metadata, r.data)
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
			log.Debugf("varlink-transport-RelayHandler sent response to message-id: %v", r.id)
			log.Tracef("varlink-transport-RelayHandler response: %v", resp)
		}
	}()

	return service.Listen(context.Background(), "unix:"+addr, 0)
}
