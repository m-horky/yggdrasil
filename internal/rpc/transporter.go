package rpc

type RPCEvent string

const (
	RPCEventReceivedDisconnect   RPCEvent = "RECEIVED_DISCONNECT"
	RPCEventUnexpectedDisconnect RPCEvent = "UNEXPECTED_DISCONNECT"
	RPCEventConnectionRestored   RPCEvent = "CONNECTION_RESTORED"
)

type TransporterError string

func (t TransporterError) Error() string {
	return string(t)
}

type RelayHandler func(addr string, id string, metadata map[string]string, data []byte) (responseCode int, responseMetadata map[string]string, responseData []byte, err error)

type Transporter interface {
	// Dispatch sends data to a worker at the given address.
	Dispatch(addr string, id string, metadata map[string]string, data []byte) error

	// SetRelayHandler calls f within a goroutine whenever values are
	// received by the RPC implementation.
	SetRelayHandler(f RelayHandler) error

	// NotifyEvent sends an event to a worker at the given address
	NotifyEvent(addr string, name RPCEvent, detail string, metadata map[string]string) error

	// Listen initializes a listener and starts listening on the given address.
	Listen(addr string) error
}

// response is a struct that can be used internally by rpc.Transporter
// implementations when implementing the Relay method handler.
type response struct {
	code     int
	metadata map[string]string
	data     []byte
}

// request is a struct that can be used internally by rpc.Transporter
// implementations when implementing the Relay method handler.
type request struct {
	destination string
	id          string
	metadata    map[string]string
	data        []byte
	response    chan response
}
