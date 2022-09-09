package main

import (
	"context"
	"encoding/json"

	"git.sr.ht/~spc/go-log"
	dispatcher "github.com/redhatinsights/yggdrasil/protocol/varlink/dispatcher"
	worker "github.com/redhatinsights/yggdrasil/protocol/varlink/worker"
	"github.com/varlink/go/varlink"
)

type echoWorker struct {
	worker.VarlinkInterface
}

// Dispatch implements the Dispatch method of the com.redhat.yggdrasil.worker
// interface.
func (w *echoWorker) Dispatch(ctx context.Context, c worker.VarlinkCall, messageID string, metadata map[string]string, data json.RawMessage) error {
	go func() {
		log.Debugf("received message-id: %v", messageID)
		log.Tracef("received message-metadata: %+v", metadata)
		log.Tracef("received message-data: %v", data)

		message := string(data)
		log.Infoln(message)

		conn, err := varlink.NewConnection(context.Background(), "unix:"+yggSocketAddr)
		if err != nil {
			log.Errorf("cannot create connection: %v", err)
			return
		}

		responseCode, responseMetadata, responseData, err := dispatcher.Relay().Call(context.Background(), conn, "echo", messageID, metadata, data)
		if err != nil {
			log.Errorf("cannot call Relay method: %v", err)
			return
		}
		log.Debugf("received response-code: %v", responseCode)
		log.Debugf("received response-metadata: %v", responseMetadata)
		log.Debugf("received response-data: %v", responseData)
	}()

	return c.ReplyDispatch(ctx, true)
}
