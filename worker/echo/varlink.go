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

func (w *echoWorker) Dispatch(ctx context.Context, c worker.VarlinkCall, messageID string, metadata map[string]string, data json.RawMessage) error {
	go func() {
		conn, err := varlink.NewConnection(context.Background(), "unix:@com.redhat.yggdrasil.dispatcher")
		if err != nil {
			log.Errorf("cannot create connection: %v", err)
			return
		}

		respCode, respMetadata, respData, err := dispatcher.Relay().Call(context.Background(), conn, "", messageID, metadata, data)
		if err != nil {
			log.Errorf("cannot call Relay method: %v", err)
			return
		}

		log.Infof("received response code: %v", respCode)
		log.Infof("received response metadata: %+v", respMetadata)
		log.Infof("received response data: %v", respData)
	}()

	return c.ReplyDispatch(context.Background(), true)
}
