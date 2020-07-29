package client

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"p2p.org/dc4bc/storage"

	"p2p.org/dc4bc/qr"
)

const pollingPeriod = time.Second

type Client struct {
	fsm        interface{}
	state      State
	storage    storage.Storage
	qrCodesDir string
}

func NewClient() (*Client, error) {
	return &Client{}, nil
}

func (c *Client) PostMessage(message storage.Message) error {
	if err := c.storage.Post(message); err != nil {
		return fmt.Errorf("failed to post message: %w", err)
	}

	return nil
}

func (c *Client) Poll(ctx context.Context) {
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
		case <-ctx.Done():
			return
		}
	}
}

func (c *Client) GetOperationsList() (map[string]*Operation, error) {
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

	operationQRPath := filepath.Join(c.qrCodesDir, operationID)
	if err := qr.WriteQR(operationQRPath, operationJSON); err != nil {
		return "", fmt.Errorf("failed to WriteQR: %w", err)
	}

	return operationQRPath, nil
}

// ReadProcessedOperation reads the processed operation from camera, checks that
// the processed operation has its unprocessed counterpart in our state,
// posts a Message to the storage and deletes the operation from our state.
func (c *Client) ReadProcessedOperation() error {
	bz, err := qr.ReadQRFromCamera()
	if err != nil {
		return fmt.Errorf("failed to ReadQRFromCamera: %s", err)
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
	if err := c.storage.Post(message); err != nil {
		return fmt.Errorf("failed to post message: %w", err)
	}

	if err := c.state.DeleteOperation(operation.ID); err != nil {
		return fmt.Errorf("failed to DeleteOperation: %w", err)
	}

	return nil
}
