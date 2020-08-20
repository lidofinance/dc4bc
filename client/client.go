package client

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/depools/dc4bc/airgapped"
	"github.com/depools/dc4bc/client/types"
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/depools/dc4bc/fsm/state_machines/signature_proposal_fsm"
	spf "github.com/depools/dc4bc/fsm/state_machines/signature_proposal_fsm"

	"github.com/depools/dc4bc/fsm/state_machines"

	"github.com/depools/dc4bc/fsm/fsm"
	dpf "github.com/depools/dc4bc/fsm/state_machines/dkg_proposal_fsm"
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
	address     string
	pubKey      ed25519.PublicKey
	ctx         context.Context
	state       State
	storage     storage.Storage
	keyStore    KeyStore
	qrProcessor qr.Processor
	airgapped   *airgapped.AirgappedMachine
}

func NewClient(
	ctx context.Context,
	userName string,
	state State,
	storage storage.Storage,
	keyStore KeyStore,
	qrProcessor qr.Processor,
	airgappedMachine *airgapped.AirgappedMachine,
) (*Client, error) {
	keyPair, err := keyStore.LoadKeys(userName, "")
	if err != nil {
		return nil, fmt.Errorf("failed to LoadKeys: %w", err)
	}

	return &Client{
		ctx:         ctx,
		userName:    userName,
		address:     keyPair.GetAddr(),
		pubKey:      keyPair.Pub,
		state:       state,
		storage:     storage,
		keyStore:    keyStore,
		qrProcessor: qrProcessor,
		airgapped:   airgappedMachine,
	}, nil
}

func (c *Client) GetAddr() string {
	return c.address
}

func (c *Client) GetPubKey() ed25519.PublicKey {
	return c.pubKey
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
					log.Println("Failed to process message:", c.userName, err)
					fmt.Println("Not processed", c.userName, message.Event)
				} else {
					fmt.Println("Processed", c.userName, message.Event)
				}
			}

			operations, err := c.GetOperations()
			if err != nil {
				log.Printf("Failed to get operations: %v", err)
			}
			for _, operation := range operations {
				processedOperations, err := c.airgapped.HandleOperation(*operation)
				if err != nil {
					return fmt.Errorf("failed to process operation in airgapped: %w", err)
				}
				for _, po := range processedOperations {
					if err = c.handleProcessedOperation(po); err != nil {
						return fmt.Errorf("failed to handle processed operation")
					}
				}
			}
		case <-c.ctx.Done():
			log.Println("Context closed, stop polling...")
			return nil
		}
	}
}

func (c *Client) SendMessage(message storage.Message) error {
	if _, err := c.storage.Send(message); err != nil {
		return fmt.Errorf("failed to post message: %w", err)
	}

	return nil
}

func (c *Client) ProcessMessage(message storage.Message) error {
	fsmInstance, err := c.getFSMInstance(message.DkgRoundID)
	if err != nil {
		return fmt.Errorf("failed to getFSMInstance: %w", err)
	}
	state, _ := fsmInstance.State()
	fmt.Printf("Do msg %s for username %s with init state: %s\n", message.Event, c.userName, state)

	if fsm.Event(message.Event) != signature_proposal_fsm.EventInitProposal {
		if err := c.verifyMessage(fsmInstance, message); err != nil {
			return fmt.Errorf("failed to verifyMessage %+v: %w", message, err)
		}
	}

	fsmReq, err := types.FSMRequestFromMessage(message)
	if err != nil {
		return fmt.Errorf("failed to get FSMRequestFromMessage: %v", err)
	}

	resp, fsmDump, err := fsmInstance.Do(fsm.Event(message.Event), fsmReq)
	if err != nil {
		return fmt.Errorf("failed to Do operation in FSM: %w", err)
	}

	state, _ = fsmInstance.State()
	fmt.Printf("Done msg %s for username %s with state after Do: %s\n", message.Event, c.userName, state)

	var operation *types.Operation
	switch resp.State {
	// if the new state is waiting for RPC to airgapped machine
	case
		spf.StateAwaitParticipantsConfirmations,
		dpf.StateDkgCommitsAwaitConfirmations,
		dpf.StateDkgDealsAwaitConfirmations,
		dpf.StateDkgResponsesAwaitConfirmations:
		bz, err := json.Marshal(resp.Data)
		if err != nil {
			return fmt.Errorf("failed to marshal FSM response: %w", err)
		}

		operation = &types.Operation{
			Type:          types.OperationType(resp.State),
			Payload:       bz,
			DKGIdentifier: message.DkgRoundID,
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

	if err := c.state.SaveFSM(message.DkgRoundID, fsmDump); err != nil {
		return fmt.Errorf("failed to SaveFSM: %w", err)
	}

	return nil
}

func (c *Client) GetOperations() (map[string]*types.Operation, error) {
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

	var operation types.Operation
	if err = json.Unmarshal(bz, &operation); err != nil {
		return fmt.Errorf("failed to unmarshal processed operation")
	}

	return c.handleProcessedOperation(operation)
}

func (c *Client) handleProcessedOperation(operation types.Operation) error {
	storedOperation, err := c.state.GetOperationByID(operation.ID)
	if err != nil {
		return fmt.Errorf("failed to find matching operation: %w", err)
	}

	if err := storedOperation.Check(&operation); err != nil {
		return fmt.Errorf("processed operation does not match stored operation: %w", err)
	}

	message := storage.Message{
		Event:      string(operation.Type),
		Data:       operation.Result,
		DkgRoundID: operation.DKGIdentifier,
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

func (c *Client) getFSMInstance(dkgRoundID string) (*state_machines.FSMInstance, error) {
	var err error
	fsmInstance, ok, err := c.state.LoadFSM(dkgRoundID)
	if err != nil {
		return nil, fmt.Errorf("failed to LoadFSM: %w", err)
	}

	if !ok {
		fsmInstance, err = state_machines.Create(dkgRoundID)
		if err != nil {
			return nil, fmt.Errorf("failed to create FSM instance: %w", err)
		}
		bz, err := fsmInstance.Dump()
		if err != nil {
			return nil, fmt.Errorf("failed to Dump FSM instance: %w", err)
		}
		if err := c.state.SaveFSM(dkgRoundID, bz); err != nil {
			return nil, fmt.Errorf("failed to SaveFSM: %w", err)
		}
	}

	return fsmInstance, nil
}

func (c *Client) signMessage(message []byte) ([]byte, error) {
	keyPair, err := c.keyStore.LoadKeys(c.userName, "")
	if err != nil {
		return nil, fmt.Errorf("failed to LoadKeys: %w", err)
	}

	return ed25519.Sign(keyPair.Priv, message), nil
}

func (c *Client) verifyMessage(fsmInstance *state_machines.FSMInstance, message storage.Message) error {
	senderPubKey, err := fsmInstance.GetPubKeyByAddr(message.SenderAddr)
	if err != nil {
		return fmt.Errorf("failed to GetPubKeyByAddr: %w", err)
	}

	if !ed25519.Verify(senderPubKey, message.Bytes(), message.Signature) {
		return errors.New("signature is corrupt")
	}

	return nil
}
