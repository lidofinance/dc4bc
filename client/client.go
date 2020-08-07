package client

import (
	"context"
	"encoding/json"
	"fmt"
	"go.dedis.ch/kyber/v3"
	"log"
	"path/filepath"
	"time"

	fsmStateMachines "github.com/depools/dc4bc/fsm/state_machines"
	"github.com/depools/dc4bc/qr"
	"github.com/depools/dc4bc/storage"
	sign "go.dedis.ch/kyber/v3/sign/schnorr"
	"go.dedis.ch/kyber/v3/util/key"
)

const (
	pollingPeriod = time.Second
	QrCodesDir    = "/tmp"
)

type Client struct {
	ctx         context.Context
	fsm         *fsmStateMachines.FSMInstance
	state       State
	storage     storage.Storage
	qrProcessor qr.Processor

	// these just a template
	suite       key.Suite
	authKeyPair *key.Pair
}

func NewClient(
	ctx context.Context,
	fsm *fsmStateMachines.FSMInstance,
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

				fsmReq, err := FSMRequestFromBytes(message.Data)
				if err != nil {
					panic(err)
				}

				resp, fsmDump, err := c.fsm.Do(fsmReq.Event, fsmReq.Args...)
				if err != nil {
					panic(err)
				}

				var operation *Operation

				if resp.IsOpRequired {
					operation = &Operation{
						Type:    OperationType(resp.State),
						Payload: resp.Data, // TODO:marshall
					}
				}

				// I.e., if FSM returned an Operation for us.
				if operation != nil {
					if err := c.state.PutOperation(operation); err != nil {
						panic(err)
					}
				}

				if err := c.state.SaveOffset(message.Offset); err != nil {
					panic(err)
				}

				if err := c.state.SaveFSM(fsmDump); err != nil {
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

func (c *Client) getOperationJSON(operationID string) ([]byte, error) {
	operation, err := c.state.GetOperationByID(operationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get operation: %w", err)
	}

	operationJSON, err := json.Marshal(operation)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal operation: %w", err)
	}
	return operationJSON, nil
}

// GetOperationQRPath returns a path to the image with the QR generated
// for the specified operation. It is supposed that the user will open
// this file herself.
func (c *Client) GetOperationQRPath(operationID string) (string, error) {
	operationJSON, err := c.getOperationJSON(operationID)
	if err != nil {
		return "", fmt.Errorf("failed to get operation in JSON: %w", err)
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

	return c.handleProcessedOperation(operation)
}

func (c *Client) handleProcessedOperation(operation Operation) error {
	storedOperation, err := c.state.GetOperationByID(operation.ID)
	if err != nil {
		return fmt.Errorf("failed to find matching operation: %w", err)
	}

	if err := storedOperation.Check(&operation); err != nil {
		return fmt.Errorf("processed operation does not match stored operation: %w", err)
	}

	sig, err := c.signMessage(operation.Result)
	if err != nil {
		return fmt.Errorf("failed to sign a message: %w", err)
	}
	message := storage.Message{
		Data:      operation.Result,
		Signature: sig,
	}

	if _, err := c.storage.Send(message); err != nil {
		return fmt.Errorf("failed to post message: %w", err)
	}

	if err := c.state.DeleteOperation(operation.ID); err != nil {
		return fmt.Errorf("failed to DeleteOperation: %w", err)
	}

	return nil
}

// it's just a template
func (c *Client) signMessage(msg []byte) ([]byte, error) {
	s, err := sign.Sign(c.suite, c.authKeyPair.Private, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to sign a message: %w", err)
	}
	return s, nil
}

// it's just a template
func (c *Client) verifyMessage(participant string, msg, signature []byte) error {
	return sign.Verify(c.suite, c.getPublicKeyOfParticipant(participant), msg, signature)
}

// func should return public key of participant for checking his message signature
func (c *Client) getPublicKeyOfParticipant(participant string) kyber.Point {
	return nil
}
