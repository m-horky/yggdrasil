package net

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"git.sr.ht/~spc/go-log"
	"github.com/redhatinsights/yggdrasil"
	"github.com/redhatinsights/yggdrasil/internal/http"
)

// HTTP is a Transporter that sends and receives data and control
// messages by sending HTTP requests to a URL.
type HTTP struct {
	clientID        string
	client          *http.Client
	server          string
	dataHandler     RxHandlerFunc
	pollingInterval time.Duration
	disconnected    atomic.Value
}

func NewHTTPTransport(clientID string, server string, tlsConfig *tls.Config, userAgent string, pollingInterval time.Duration) (*HTTP, error) {
	disconnected := atomic.Value{}
	disconnected.Store(false)
	return &HTTP{
		clientID:        clientID,
		client:          http.NewHTTPClient(tlsConfig.Clone(), userAgent),
		pollingInterval: pollingInterval,
		disconnected:    disconnected,
		server:          server,
	}, nil
}

func (t *HTTP) Connect() error {
	t.disconnected.Store(false)
	go func() {
		for {
			if t.disconnected.Load().(bool) {
				return
			}
			resp, err := t.client.Get(t.getUrl("in", "control"))
			if err != nil {
				log.Tracef("cannot get HTTP request: %v", err)
			}
			if resp != nil {
				data, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					log.Errorf("cannot read response body: %v", err)
					continue
				}
				if t.dataHandler != nil {
					metadata := make(map[string]interface{})
					for k, v := range resp.Header {
						metadata[k] = v
					}
					_ = t.dataHandler("control", metadata, data)
				}
				resp.Body.Close()
			}
			time.Sleep(t.pollingInterval)
		}
	}()

	go func() {
		for {
			if t.disconnected.Load().(bool) {
				return
			}
			resp, err := t.client.Get(t.getUrl("in", "data"))
			if err != nil {
				log.Tracef("cannot get HTTP request: %v", err)
			}
			if resp != nil {
				data, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					log.Errorf("cannot read response body: %v", err)
					continue
				}
				if t.dataHandler != nil {
					metadata := make(map[string]interface{})
					for k, v := range resp.Header {
						metadata[k] = v
					}
					_ = t.dataHandler("data", metadata, data)
				}
				resp.Body.Close()
			}
			time.Sleep(t.pollingInterval)
		}
	}()

	return nil
}

func (t *HTTP) Disconnect(quiesce uint) {
	time.Sleep(time.Millisecond * time.Duration(quiesce))
	t.disconnected.Store(true)
}

func (t *HTTP) Tx(addr string, metadata map[string]string, data []byte) (responseCode int, responseMetadata map[string]string, responseData []byte, err error) {
	resp, err := t.send(data, addr)
	if err != nil {
		return -1, nil, nil, fmt.Errorf("cannot tx: %w", err)
	}
	defer resp.Body.Close()

	responseCode = resp.StatusCode
	responseMetadata = make(map[string]string)
	for k, v := range resp.Header {
		responseMetadata[k] = strings.Join(v, ";")
	}
	responseData, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return -1, nil, nil, fmt.Errorf("cannot read response body: %w", err)
	}
	return
}

func (t *HTTP) SetRxHandler(f RxHandlerFunc) error {
	t.dataHandler = f
	return nil
}

func (t *HTTP) send(message []byte, channel string) (*http.Response, error) {
	if t.disconnected.Load().(bool) {
		return nil, nil
	}
	url := t.getUrl("out", channel)
	headers := map[string]string{
		"Content-Type": "application/json",
	}
	log.Tracef("posting HTTP request body: %s", string(message))
	resp, err := t.client.Post(url, headers, message)
	if err != nil {
		return nil, fmt.Errorf("cannot send data: %w", err)
	}

	return resp, nil
}

func (t *HTTP) getUrl(direction string, channel string) string {
	path := filepath.Join(yggdrasil.PathPrefix, channel, t.clientID, direction)
	return fmt.Sprintf("http://%s/%s", t.server, path)
}
