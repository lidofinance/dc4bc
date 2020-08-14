package client

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/depools/dc4bc/fsm/fsm"
	fsmStateMachines "github.com/depools/dc4bc/fsm/state_machines"
	dkgFSM "github.com/depools/dc4bc/fsm/state_machines/dkg_proposal_fsm"
	"github.com/depools/dc4bc/qr"
	"github.com/depools/dc4bc/storage"
)

const (
	pollingPeriod = time.Second
	QrCodesDir    = "/tmp"
)

type Client struct {
	sync.Mutex
	userName    string
	ctx         context.Context
	fsm         *fsmStateMachines.FSMInstance
	state       State
	storage     storage.Storage
	keyStore    KeyStore
	qrProcessor qr.Processor
}

func NewClient(
	ctx context.Context,
	userName string,
	fsm *fsmStateMachines.FSMInstance,
	state State,
	storage storage.Storage,
	keyStore KeyStore,
	qrProcessor qr.Processor,
) (*Client, error) {
	return &Client{
		ctx:         ctx,
		userName:    userName,
		fsm:         fsm,
		state:       state,
		storage:     storage,
		keyStore:    keyStore,
		qrProcessor: qrProcessor,
	}, nil
}

func (c *Client) SendMessage(message storage.Message) error {
	if _, err := c.storage.Send(message); err != nil {
		return fmt.Errorf("failed to post message: %w", err)
	}

	return nil
}

func (c *Client) Poll() error {
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
				return fmt.Errorf("failed to GetMessages: %w", err)
			}

			for _, message := range messages {
				if err := c.ProcessMessage(message); err != nil {
					log.Println("Failed to process message:", err)
				}
			}
		case <-c.ctx.Done():
			log.Println("Context closed, stop polling...")
			return nil
		}
	}
}

func (c *Client) ProcessMessage(message storage.Message) error {
	fsmReq, err := FSMRequestFromMessage(message)
	if err != nil {
		return fmt.Errorf("failed to get FSMRequestFromMessage: %v", err)
	}

	resp, fsmDump, err := c.fsm.Do(fsm.Event(message.Event), fsmReq)
	if err != nil {
		return fmt.Errorf("failed to Do operation in FSM: %w", err)
	}

	var operation *Operation
	switch resp.State {
	// if the new state is waiting for RPC to airgapped machine
	case
		dkgFSM.StateDkgCommitsAwaitConfirmations,
		dkgFSM.StateDkgDealsAwaitConfirmations,
		dkgFSM.StateDkgResponsesAwaitConfirmations:
		bz, err := json.Marshal(resp.Data)
		if err != nil {
			return fmt.Errorf("failed to marshal FSM response: %w", err)
		}

		operation = &Operation{
			Type:    OperationType(resp.State),
			Payload: bz,
		}
	default:
		log.Printf("State %s does not require an operation", resp.State)
	}

	if operation != nil {
		if err := c.state.PutOperation(operation); err != nil {
			return fmt.Errorf("failed to PutOperation: %w", err)
		}
	}

	if err := c.state.SaveOffset(message.Offset); err != nil {
		return fmt.Errorf("failed to SaveOffset: %w", err)
	}

	if err := c.state.SaveFSM(fsmDump); err != nil {
		return fmt.Errorf("failed to SaveFSM: %w", err)
	}

	return nil
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

	message := storage.Message{
		Sender: c.userName,
		Event:  string(operation.Type),
		Data:   operation.Result,
	}

	sig, err := c.signMessage(message.Bytes())
	if err != nil {
		return fmt.Errorf("failed to sign a message: %w", err)
	}
	message.Signature = sig

	if _, err := c.storage.Send(message); err != nil {
		return fmt.Errorf("failed to post message: %w", err)
	}

	if err := c.state.DeleteOperation(operation.ID); err != nil {
		return fmt.Errorf("failed to DeleteOperation: %w", err)
	}

	return nil
}

func (c *Client) signMessage(message []byte) ([]byte, error) {
	keyPair, err := c.keyStore.LoadKeys(c.userName, "")
	if err != nil {
		return nil, fmt.Errorf("failed to LoadKeys: %w", err)
	}

	return ed25519.Sign(keyPair.Priv, message), nil
}
