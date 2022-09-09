package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"git.sr.ht/~spc/go-log"
	"github.com/google/uuid"
	"github.com/redhatinsights/yggdrasil"
	"github.com/redhatinsights/yggdrasil/internal/net"
)

type Client struct {
	t net.Transporter
	m *manager
}

func NewClient(manager *manager, transporter net.Transporter) *Client {
	return &Client{
		m: manager,
		t: transporter,
	}
}

func (c *Client) Connect() error {
	if c.t == nil {
		return fmt.Errorf("cannot connect client: client missing transport")
	}

	c.m.start()

	// start receiving values from the manager and send them using the
	// configured net.Transporter.
	go func() {
		for msg := range c.m.outbound {
			code, metadata, data, err := c.SendDataMessage(&msg.data, msg.data.Metadata)
			if err != nil {
				log.Errorf("cannot send data message: %v", err)
			}
			msg.resp <- struct {
				code     int
				metadata map[string]string
				data     []byte
			}{
				code:     code,
				metadata: metadata,
				data:     data,
			}
		}
	}()

	c.t.SetRxHandler(func(addr string, metadata map[string]interface{}, data []byte) error {
		switch addr {
		case "data":
			var message yggdrasil.Data

			if err := json.Unmarshal(data, &message); err != nil {
				return fmt.Errorf("cannot unmarshal data message: %w", err)
			}
			if err := c.ReceiveDataMessage(&message); err != nil {
				return fmt.Errorf("cannot process data message: %w", err)
			}
		case "control":
			var message yggdrasil.Control

			if err := json.Unmarshal(data, &message); err != nil {
				return fmt.Errorf("cannot unmarshal control message: %w", err)
			}
			if err := c.ReceiveControlMessage(&message); err != nil {
				return fmt.Errorf("cannot process control message: %w", err)
			}
		default:
			return fmt.Errorf("unsupported destination type: %v", addr)
		}
		return nil
	})

	return c.t.Connect()
}

func (c *Client) SendDataMessage(msg *yggdrasil.Data, metadata map[string]string) (int, map[string]string, []byte, error) {
	return c.sendMessage(msg, "data", metadata)
}

func (c *Client) SendConnectionStatusMessage(msg *yggdrasil.ConnectionStatus) error {
	_, _, _, err := c.sendMessage(msg, "control", nil)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) SendEventMessage(msg *yggdrasil.Event, metadata map[string]string) error {
	_, _, _, err := c.sendMessage(msg, "control", metadata)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) sendMessage(msg interface{}, dest string, metadata map[string]string) (int, map[string]string, []byte, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return -1, nil, nil, fmt.Errorf("cannot marshal message: %w", err)
	}
	return c.t.Tx(dest, metadata, data)
}

// ReceiveDataMessage sends a value to a channel for dispatching to worker processes.
func (c *Client) ReceiveDataMessage(msg *yggdrasil.Data) error {
	c.m.inbound <- *msg

	return nil
}

// ReceiveControlMessage unpacks a control message and acts accordingly.
func (c *Client) ReceiveControlMessage(msg *yggdrasil.Control) error {
	switch msg.Type {
	case yggdrasil.MessageTypeCommand:
		var cmd yggdrasil.Command
		if err := json.Unmarshal(msg.Content, &cmd.Content); err != nil {
			return fmt.Errorf("cannot unmarshal command message: %w", err)
		}

		log.Debugf("received message %v", cmd.MessageID)
		log.Tracef("command: %+v", cmd)
		log.Tracef("Control message: %v", cmd)

		switch cmd.Content.Command {
		case yggdrasil.CommandNamePing:
			event := yggdrasil.Event{
				Type:       yggdrasil.MessageTypeEvent,
				MessageID:  uuid.New().String(),
				ResponseTo: cmd.MessageID,
				Version:    1,
				Sent:       time.Now(),
				Content:    string(yggdrasil.EventNamePong),
			}

			data, err := json.Marshal(event)
			if err != nil {
				return fmt.Errorf("cannot marshal event: %w", err)
			}
			if _, _, _, err := c.t.Tx("control", nil, data); err != nil {
				return fmt.Errorf("cannot send data: %w", err)
			}
		case yggdrasil.CommandNameDisconnect:
			log.Info("disconnecting...")
			c.m.disconnectWorkers()
			c.t.Disconnect(500)
		case yggdrasil.CommandNameReconnect:
			log.Info("reconnecting...")
			c.t.Disconnect(500)
			delay, err := strconv.ParseInt(cmd.Content.Arguments["delay"], 10, 64)
			if err != nil {
				return fmt.Errorf("cannot parse data to int: %w", err)
			}
			time.Sleep(time.Duration(delay) * time.Second)

			if err := c.t.Connect(); err != nil {
				return fmt.Errorf("cannot reconnect to broker: %w", err)
			}
		default:
			return fmt.Errorf("unknown command: %v", cmd.Content.Command)
		}
	default:
		return fmt.Errorf("unsupported control message: %v", msg)
	}

	return nil
}

// ConnectionStatus creates a connection-status message using the current state
// of the client.
func (c *Client) ConnectionStatus() (*yggdrasil.ConnectionStatus, error) {
	facts, err := yggdrasil.GetCanonicalFacts()
	if err != nil {
		return nil, fmt.Errorf("cannot get canonical facts: %w", err)
	}

	tagsFilePath := filepath.Join(yggdrasil.SysconfDir, yggdrasil.LongName, "tags.toml")

	var tagMap map[string]string
	if _, err := os.Stat(tagsFilePath); !os.IsNotExist(err) {
		var err error
		tagMap, err = readTagsFile(tagsFilePath)
		if err != nil {
			log.Errorf("cannot load tags: %v", err)
		}
	}

	msg := yggdrasil.ConnectionStatus{
		Type:      yggdrasil.MessageTypeConnectionStatus,
		MessageID: uuid.New().String(),
		Version:   1,
		Sent:      time.Now(),
		Content: struct {
			CanonicalFacts yggdrasil.CanonicalFacts     "json:\"canonical_facts\""
			Dispatchers    map[string]map[string]string "json:\"dispatchers\""
			State          yggdrasil.ConnectionState    "json:\"state\""
			Tags           map[string]string            "json:\"tags,omitempty\""
		}{
			CanonicalFacts: *facts,
			Dispatchers:    c.m.flattenDispatchers(),
			State:          yggdrasil.ConnectionStateOnline,
			Tags:           tagMap,
		},
	}

	return &msg, nil
}
