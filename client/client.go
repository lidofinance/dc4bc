package client

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/depool/dc4bc/qr"
	"github.com/depool/dc4bc/storage"
)

const (
	pollingPeriod = time.Second
	QrCodesDir    = "/tmp"
)

type Client struct {
	ctx         context.Context
	fsm         interface{}
	state       State
	storage     storage.Storage
	qrProcessor qr.Processor
}

func NewClient(
	ctx context.Context,
	fsm interface{},
	state State,
	storage storage.Storage,
	qrProcessor qr.Processor,
) (*Client, error) {
	return &Client{
		ctx:         ctx,
		fsm:         fsm,
		state:       state,
		storage:     storage,
		qrProcessor: qrProcessor,
	}, nil
}

func (c *Client) SendMessage(message storage.Message) error {
	if _, err := c.storage.Send(message); err != nil {
		return fmt.Errorf("failed to post message: %w", err)
	}

	return nil
}

func (c *Client) Poll() {
	tk := time.NewTicker(pollingPeriod)
	for {
		select {
		case <-tk.C:
			offset, err := c.state.LoadOffset()
			if err != nil {
				panic(err)
			}

			messages, err := c.storage.GetMessages(offset)
			if err != nil {
				panic(err)
			}

			for _, message := range messages {
				log.Println("Message:", message)

				// Feed the message to the FSM, get a possibly empty operation.
				var operation *Operation

				// I.e., if FSM returned an Operation for us.
				if operation != nil {
					if err := c.state.PutOperation(operation); err != nil {
						panic(err)
					}
				}

				if err := c.state.SaveOffset(message.Offset); err != nil {
					panic(err)
				}

				if err := c.state.SaveFSM(c.fsm); err != nil {
					panic(err)
				}
			}
		case <-c.ctx.Done():
			return
		}
	}
}

func (c *Client) GetOperations() (map[string]*Operation, error) {
	return c.state.GetOperations()
}

// GetOperationQRPath returns a path to the image with the QR generated
// for the specified operation. It is supposed that the user will open
// this file herself.
func (c *Client) GetOperationQRPath(operationID string) (string, error) {
	operation, err := c.state.GetOperationByID(operationID)
	if err != nil {
		return "", fmt.Errorf("failed to get operation: %w", err)
	}

	operationJSON, err := json.Marshal(operation)
	if err != nil {
		return "", fmt.Errorf("failed to marshal operation: %w", err)
	}

	operationQRPath := filepath.Join(QrCodesDir, operationID)
	if err := c.qrProcessor.WriteQR(operationQRPath, operationJSON); err != nil {
		return "", fmt.Errorf("failed to WriteQR: %w", err)
	}

	return operationQRPath, nil
}

// ReadProcessedOperation reads the processed operation from camera, checks that
// the processed operation has its unprocessed counterpart in our state,
// posts a Message to the storage and deletes the operation from our state.
func (c *Client) ReadProcessedOperation() error {
	bz, err := c.qrProcessor.ReadQR()
	if err != nil {
		return fmt.Errorf("failed to ReadQR: %s", err)
	}

	var operation Operation
	if err = json.Unmarshal(bz, &operation); err != nil {
		return fmt.Errorf("failed to unmarshal processed operation")
	}

	storedOperation, err := c.state.GetOperationByID(operation.ID)
	if err != nil {
		return fmt.Errorf("failed to find matching operation: %w", err)
	}

	if err := storedOperation.Check(&operation); err != nil {
		return fmt.Errorf("processed operation does not match stored operation: %w", err)
	}

	var message storage.Message
	if _, err := c.storage.Send(message); err != nil {
		return fmt.Errorf("failed to post message: %w", err)
	}

	if err := c.state.DeleteOperation(operation.ID); err != nil {
		return fmt.Errorf("failed to DeleteOperation: %w", err)
	}

	return nil
}
