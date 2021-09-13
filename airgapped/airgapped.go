package airgapped

import (
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"sync"

	"github.com/lidofinance/dc4bc/fsm/types/responses"

	vss "github.com/corestario/kyber/share/vss/rabin"

	"github.com/corestario/kyber"
	"github.com/corestario/kyber/encrypt/ecies"
	client "github.com/lidofinance/dc4bc/client/types"
	"github.com/lidofinance/dc4bc/dkg"
	"github.com/lidofinance/dc4bc/fsm/fsm"
	"github.com/lidofinance/dc4bc/fsm/state_machines/dkg_proposal_fsm"
	"github.com/lidofinance/dc4bc/fsm/state_machines/signature_proposal_fsm"
	"github.com/lidofinance/dc4bc/fsm/state_machines/signing_proposal_fsm"
	"github.com/lidofinance/dc4bc/fsm/types/requests"
	"github.com/lidofinance/dc4bc/qr"
	"github.com/syndtr/goleveldb/leveldb"
)

const (
	seedSize = 32
)

type Machine struct {
	sync.Mutex

	ResultQRFolder string

	dkgInstances map[string]*dkg.DKG
	// Used to encrypt local sensitive data, e.g. BLS keyrings.
	encryptionKey []byte
	pubKey        kyber.Point
	secKey        kyber.Scalar
	baseSuite     vss.Suite
	baseSeed      []byte

	qrProcessor qr.Processor
	db          *leveldb.DB
}

func NewMachine(dbPath string) (*Machine, error) {
	var (
		err error
	)

	am := &Machine{
		dkgInstances: make(map[string]*dkg.DKG),
		qrProcessor:  qr.NewCameraProcessor(nil),
	}

	if am.db, err = leveldb.OpenFile(dbPath, nil); err != nil {
		return nil, fmt.Errorf("failed to open db file %s for keys: %w", dbPath, err)
	}

	if err := am.loadBaseSeed(); err != nil {
		return nil, fmt.Errorf("failed to loadBaseSeed: %w", err)
	}

	if _, err = am.db.Get([]byte(operationsLogDBKey), nil); err != nil {
		if err == leveldb.ErrNotFound {
			operationsLogBz, _ := json.Marshal(RoundOperationLog{})
			if err := am.db.Put([]byte(operationsLogDBKey), operationsLogBz, nil); err != nil {
				return nil, fmt.Errorf("failed to init Operation log: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to init Operation log (fatal): %w", err)
		}
	}

	return am, nil
}

func (am *Machine) SetQRProcessorFramesDelay(delay int) {
	am.qrProcessor.SetDelay(delay)
}

func (am *Machine) SetQRProcessorChunkSize(chunkSize int) {
	am.qrProcessor.SetChunkSize(chunkSize)
}

func (am *Machine) SetResultQRFolder(resultQRFolder string) {
	am.ResultQRFolder = resultQRFolder
}

// InitKeys load keys public and private keys for DKG from LevelDB. If keys do not exist, it creates them.
func (am *Machine) InitKeys() error {
	err := am.LoadKeysFromDB()
	if err != nil && err != leveldb.ErrNotFound {
		return fmt.Errorf("failed to load keys from db: %w", err)
	}

	// If keys were not generated yet.
	if err == leveldb.ErrNotFound {
		return am.GenerateKeys()
	}

	return nil
}

func (am *Machine) GenerateKeys() error {
	am.secKey = am.baseSuite.Scalar().Pick(am.baseSuite.RandomStream())
	am.pubKey = am.baseSuite.Point().Mul(am.secKey, nil)
	if err := am.SaveKeysToDB(); err != nil {
		return fmt.Errorf("failed to SaveKeysToDB: %w", err)
	}

	return nil
}

// SetEncryptionKey set a key to encrypt and decrypt sensitive data.
func (am *Machine) SetEncryptionKey(key []byte) {
	am.encryptionKey = key
}

// SensitiveDataRemoved indicates whether sensitive information has been cleared
func (am *Machine) SensitiveDataRemoved() bool {
	return len(am.encryptionKey) == 0
}

// DropSensitiveData remove sensitive data from memory
func (am *Machine) DropSensitiveData() {
	am.Lock()
	defer am.Unlock()

	// There is no guarantee that GC actually deleted a data from memory, but that's ok at this moment
	am.secKey = nil
	am.pubKey = nil
	am.encryptionKey = nil
}

func (am *Machine) ReplayOperationsLog(dkgIdentifier string) error {
	operationsLog, err := am.getOperationsLog(dkgIdentifier)
	if err != nil {
		return fmt.Errorf("failed to getOperationsLog: %w", err)
	}

	for idx, operation := range operationsLog {
		qrPath, err := am.ProcessOperation(operation, false)
		if err != nil {
			return fmt.Errorf("failed to ProcessOperation: %w", err)
		}

		log.Printf("QR code for operation %d was saved to: %s\n", idx, qrPath)
	}

	log.Println("Successfully replayed Operation log")

	return nil
}

func (am *Machine) removeSignatureOperations(o *client.Operation) error {
	var (
		payload responses.SigningProcessParticipantResponse
		err     error
	)

	if err = json.Unmarshal(o.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	removeSignatureOperationsFunc := func(op client.Operation) bool {
		type signingPayload struct {
			SigningId string
		}
		var sp signingPayload
		if strings.HasPrefix(string(op.Type), "state_signing_") {
			if err := json.Unmarshal(op.Payload, &sp); err == nil {
				if sp.SigningId == payload.SigningId {
					return true
				}
			}
		}
		return false
	}
	return am.clearOperationsLog(o.DKGIdentifier, removeSignatureOperationsFunc)
}

func (am *Machine) ProcessOperation(operation client.Operation, storeOperation bool) (string, error) {
	resultOperation, err := am.GetOperationResult(operation)
	if err != nil {
		return "", fmt.Errorf(
			"failed to HandleOperation %s (this error is fatal): %w",
			operation.ID, err)
	}

	if storeOperation {
		if err := am.storeOperation(operation); err != nil {
			return "", fmt.Errorf("failed to storeOperation: %w", err)
		}
	}

	if fsm.State(operation.Type) == signing_proposal_fsm.StateSigningPartialSignsCollected {
		if err := am.removeSignatureOperations(&operation); err != nil {
			return "", fmt.Errorf("failed to remove signature operations: %w", err)
		}
	}

	operationBz, err := json.Marshal(resultOperation)
	if err != nil {
		return "", fmt.Errorf("failed to marshal operation: %w", err)
	}

	qrPath := filepath.Join(am.ResultQRFolder, fmt.Sprintf("dc4bc_qr_%s-response.gif", resultOperation.ID))
	if err = am.qrProcessor.WriteQR(qrPath, operationBz); err != nil {
		return "", fmt.Errorf("failed to write QR: %w", err)
	}

	return qrPath, nil
}

func (am *Machine) DropOperationsLog(dkgIdentifier string) error {
	return am.dropRoundOperationLog(dkgIdentifier)
}

// getParticipantID returns our own participant id for the given DKG round
func (am *Machine) getParticipantID(dkgIdentifier string) (int, error) {
	dkgInstance, ok := am.dkgInstances[dkgIdentifier]
	if !ok {
		return 0, fmt.Errorf("invalid dkg identifier: %s", dkgIdentifier)
	}
	return dkgInstance.ParticipantID, nil
}

// encryptDataForParticipant encrypts a data using the public key of the participant to whom the data is sent
func (am *Machine) encryptDataForParticipant(dkgIdentifier, to string, data []byte) ([]byte, error) {
	dkgInstance, ok := am.dkgInstances[dkgIdentifier]
	if !ok {
		return nil, fmt.Errorf("invalid dkg identifier: %s", dkgIdentifier)
	}

	pk, err := dkgInstance.GetPubKeyByParticipant(to)
	if err != nil {
		return nil, fmt.Errorf("failed to get pk for participant %s: %w", to, err)
	}

	encryptedData, err := ecies.Encrypt(am.baseSuite, pk, data, am.baseSuite.Hash)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt data: %w", err)
	}
	return encryptedData, nil
}

// decryptDataFromParticipant decrypts the data that was sent to us
func (am *Machine) decryptDataFromParticipant(data []byte) ([]byte, error) {
	decryptedData, err := ecies.Decrypt(am.baseSuite, am.secKey, data, am.baseSuite.Hash)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}
	return decryptedData, nil
}

func (am *Machine) GetOperationResult(operation client.Operation) (client.Operation, error) {
	var (
		err error
	)

	// handler gets a pointer to an operation, do necessary things
	// and write a result (or an error) to .Result field of operation
	switch fsm.State(operation.Type) {
	case client.ReinitDKG:
		err = am.handleReinitDKG(&operation)
	case dkg_proposal_fsm.StateDkgCommitsAwaitConfirmations:
		err = am.handleStateDkgCommitsAwaitConfirmations(&operation)
	case dkg_proposal_fsm.StateDkgDealsAwaitConfirmations:
		err = am.handleStateDkgDealsAwaitConfirmations(&operation)
	case dkg_proposal_fsm.StateDkgResponsesAwaitConfirmations:
		err = am.handleStateDkgResponsesAwaitConfirmations(&operation)
	case dkg_proposal_fsm.StateDkgMasterKeyAwaitConfirmations:
		err = am.handleStateDkgMasterKeyAwaitConfirmations(&operation)
	case signing_proposal_fsm.StateSigningAwaitConfirmations:
		err = am.handleStateSigningAwaitConfirmations(&operation)
	case signing_proposal_fsm.StateSigningAwaitPartialSigns:
		err = am.handleStateSigningAwaitPartialSigns(&operation)
	case signing_proposal_fsm.StateSigningPartialSignsCollected:
		err = am.reconstructThresholdSignature(&operation)
	default:
		err = fmt.Errorf("invalid operation type: %s", operation.Type)
	}

	// if we have error after handling the operation, we write the error to the operation, so we can feed it to a FSM
	if err != nil {
		log.Println(fmt.Sprintf("failed to handle operation %s, returning response with error to node: %v",
			operation.Type, err))
		if e := am.writeErrorRequestToOperation(&operation, err); e != nil {
			return operation, fmt.Errorf("failed to write error request to an operation: %w", e)
		}
	}

	return operation, nil
}

// writeErrorRequestToOperation writes error to a operation if some bad things happened
func (am *Machine) writeErrorRequestToOperation(o *client.Operation, handlerError error) error {
	// each type of request should have a required event even error
	// maybe should be global?
	eventToErrorMap := map[fsm.State]fsm.Event{
		signature_proposal_fsm.StateAwaitParticipantsConfirmations: signature_proposal_fsm.EventDeclineProposal,
		dkg_proposal_fsm.StateDkgCommitsAwaitConfirmations:         dkg_proposal_fsm.EventDKGCommitConfirmationError,
		dkg_proposal_fsm.StateDkgDealsAwaitConfirmations:           dkg_proposal_fsm.EventDKGDealConfirmationError,
		dkg_proposal_fsm.StateDkgResponsesAwaitConfirmations:       dkg_proposal_fsm.EventDKGResponseConfirmationError,
		dkg_proposal_fsm.StateDkgMasterKeyAwaitConfirmations:       dkg_proposal_fsm.EventDKGMasterKeyConfirmationError,
		signing_proposal_fsm.StateSigningAwaitConfirmations:        signing_proposal_fsm.EventDeclineSigningConfirmation,
		signing_proposal_fsm.StateSigningAwaitPartialSigns:         signing_proposal_fsm.EventSigningPartialSignError,
		signing_proposal_fsm.StateSigningPartialSignsCollected:     client.SignatureReconstructionFailed,
	}
	pid, err := am.getParticipantID(o.DKGIdentifier)
	if err != nil {
		return fmt.Errorf("failed to get participant id: %w", err)
	}
	req := requests.DKGProposalConfirmationErrorRequest{
		Error:         requests.NewFSMError(handlerError),
		ParticipantId: pid,
		CreatedAt:     o.CreatedAt,
	}
	errorEvent := eventToErrorMap[fsm.State(o.Type)]
	reqBz, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to generate fsm request: %w", err)
	}
	o.Event = errorEvent
	o.ResultMsgs = append(o.ResultMsgs, createMessage(*o, reqBz))
	return nil
}
