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
	state      State
	storage    storage.Storage
	qrCodesDir string
}

func NewClient() (*Client, error) {
	return &Client{}, nil
}

func (c *Client) SendMessage() {

}

func (c *Client) Poll(ctx context.Context) {
	tk := time.NewTicker(pollingPeriod)
	for {
		select {
		case <-tk.C:
			offset, err := c.state.GetOffset()
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

				if err := c.state.SetOffset(message.Offset); err != nil {
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
