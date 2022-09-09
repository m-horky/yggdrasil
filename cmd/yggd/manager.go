package main

import (
	"fmt"
	"time"

	"git.sr.ht/~spc/go-log"
	"github.com/redhatinsights/yggdrasil"
	"github.com/redhatinsights/yggdrasil/internal/rpc"
)

// manager tracks active worker processes and RPC transport interfaces. It
// receives and sends values over its rx and rx channels to receive and transmit
// data to and from workers over the worker's desired RPC protocol.
//
// manager receives values on its 'tx' channel, and sends them via the correct
// RPC transport for a given worker. It sends values on its 'rx' channel in
// order to relay data from workers to the network destination.
type manager struct {
	transports  map[string]rpc.Transporter
	workers     registry
	dispatchers chan map[string]map[string]string
	inbound     chan yggdrasil.Data
	outbound    chan struct {
		data yggdrasil.Data
		resp chan struct {
			code     int
			metadata map[string]string
			data     []byte
		}
	}
}

func newManager() *manager {
	manager := &manager{
		workers:     registry{},
		dispatchers: make(chan map[string]map[string]string),
		inbound:     make(chan yggdrasil.Data),
		outbound: make(chan struct {
			data yggdrasil.Data
			resp chan struct {
				code     int
				metadata map[string]string
				data     []byte
			}
		}),
	}

	manager.transports = map[string]rpc.Transporter{
		"varlink": rpc.NewVarlinkTransport(),
		"grpc":    rpc.NewGRPCTransport(),
	}

	return manager
}

func (m *manager) start() {
	// start goroutine receiving values from the inbound channel and send them
	// via the appropriate rpc.Transporter.Dispatch method.
	go func() {
		for data := range m.inbound {
			worker := m.workers.get(data.Directive)
			if worker == nil {
				log.Warnf("cannot find worker for directive: %v", data.Directive)
				continue
			}

			transport, present := m.transports[worker.Protocol]
			if !present {
				log.Warnf("unknown RPC protocol: %v", worker.Protocol)
				continue
			}

			// TODO: support for detachedContent

			if err := transport.Dispatch(worker.addr, data.MessageID, data.Metadata, data.Content); err != nil {
				log.Errorf("cannot transmit data to worker: %v", err)
				continue
			}
			log.Debugf("sent message %v to worker %v", data.MessageID, worker.directive)
		}
	}()

	// Set a handler function for receiving data from the RPC transports.
	for _, transport := range m.transports {
		transport.SetRelayHandler(func(addr string, id string, metadata map[string]string, data []byte) (responseCode int, responseMetadata map[string]string, responseData []byte, err error) {
			// TODO: detachedContent

			ch := make(chan struct {
				code     int
				metadata map[string]string
				data     []byte
			})
			m.outbound <- struct {
				data yggdrasil.Data
				resp chan struct {
					code     int
					metadata map[string]string
					data     []byte
				}
			}{
				data: yggdrasil.Data{
					Type:       yggdrasil.MessageTypeData,
					MessageID:  id,
					ResponseTo: "",
					Version:    1,
					Sent:       time.Now(),
					Directive:  addr,
					Metadata:   metadata,
					Content:    data,
				},
				resp: ch,
			}

			select {
			case resp := <-ch:
				responseCode = resp.code
				responseMetadata = resp.metadata
				responseData = resp.data
			case <-time.After(1 * time.Second):
				err = fmt.Errorf("timeout waiting for response")
			}
			return
		})
	}

	// start goroutine listening on each RPC transport
	go func() {
		listen := func(addr string, t rpc.Transporter) {
			if err := t.Listen(addr); err != nil {
				log.Fatalf("error: cannot listen on RPC transport: %v", err)
			}
		}

		for protocol, transport := range m.transports {
			var addr string

			switch protocol {
			case "varlink":
				addr = "@com.redhat.yggdrasil.dispatcher"
			case "grpc":
				addr = DefaultConfig.SocketAddr
			default:
				log.Fatalf("unknown RPC protcol: %v", protocol)
			}

			go listen(addr, transport)
		}
	}()
}

func (m *manager) registerWorker(worker *workerConfig) error {
	if err := m.workers.set(worker.directive, worker); err != nil {
		log.Errorf("cannot register worker: %v", err)
		return err
	}
	log.Infof("worker registered: %v", worker.directive)
	log.Debugf("worker registered: %+v", worker)

	m.dispatchers <- m.flattenDispatchers()

	return nil
}

func (m *manager) unregisterWorker(directive string) {
	m.workers.del(directive)
	log.Infof("unregistered worker: %v", directive)

	m.dispatchers <- m.flattenDispatchers()
}

func (m *manager) notifyWorkers(event rpc.RPCEvent, detail string, metadata map[string]string) {
	for _, worker := range m.workers.all() {
		transport, present := m.transports[worker.Protocol]
		if !present {
			log.Warnf("unknown RPC protocol: %v", worker.Protocol)
			continue
		}

		if err := transport.NotifyEvent(worker.addr, event, detail, metadata); err != nil {
			log.Errorf("cannot tx event: %v", err)
		}
	}
}

func (m *manager) disconnectWorkers() {
	m.notifyWorkers(rpc.RPCEventReceivedDisconnect, "", nil)
}

func (m *manager) flattenDispatchers() map[string]map[string]string {
	dispatchers := make(map[string]map[string]string)
	for directive, worker := range m.workers.all() {
		dispatchers[directive] = worker.Features
	}

	return dispatchers
}
