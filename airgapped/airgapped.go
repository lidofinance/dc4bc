package airgapped

import (
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"sync"

	vss "github.com/corestario/kyber/share/vss/rabin"

	"github.com/corestario/kyber"
	"github.com/corestario/kyber/encrypt/ecies"
	client "github.com/depools/dc4bc/client/types"
	"github.com/depools/dc4bc/dkg"
	"github.com/depools/dc4bc/fsm/fsm"
	"github.com/depools/dc4bc/fsm/state_machines/dkg_proposal_fsm"
	"github.com/depools/dc4bc/fsm/state_machines/signature_proposal_fsm"
	"github.com/depools/dc4bc/fsm/state_machines/signing_proposal_fsm"
	"github.com/depools/dc4bc/fsm/types/requests"
	"github.com/depools/dc4bc/qr"
	"github.com/syndtr/goleveldb/leveldb"
)

const (
	seedSize = 32
)

type Machine struct {
	sync.Mutex

	dkgInstances map[string]*dkg.DKG
	qrProcessor  qr.Processor

	encryptionKey []byte
	pubKey        kyber.Point
	secKey        kyber.Scalar
	baseSuite     vss.Suite
	baseSeed      []byte

	db             *leveldb.DB
	resultQRFolder string
}

func NewMachine(dbPath string) (*Machine, error) {
	var (
		err error
	)

	am := &Machine{
		dkgInstances: make(map[string]*dkg.DKG),
		qrProcessor:  qr.NewCameraProcessor(),
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

func (am *Machine) CloseCameraReader() {
	am.qrProcessor.CloseCameraReader()
}

func (am *Machine) SetQRProcessorChunkSize(chunkSize int) {
	am.qrProcessor.SetChunkSize(chunkSize)
}

func (am *Machine) SetResultQRFolder(resultQRFolder string) {
	am.resultQRFolder = resultQRFolder
}

// InitKeys load keys public and private keys for DKG from LevelDB. If keys does not exist, creates them.
func (am *Machine) InitKeys() error {
	err := am.LoadKeysFromDB()
	if err != nil && err != leveldb.ErrNotFound {
		return fmt.Errorf("failed to load keys from db: %w", err)
	}
	// if keys were not generated yet
	if err == leveldb.ErrNotFound {
		am.secKey = am.baseSuite.Scalar().Pick(am.baseSuite.RandomStream())
		am.pubKey = am.baseSuite.Point().Mul(am.secKey, nil)
		return am.SaveKeysToDB()
	}

	return nil
}

// SetEncryptionKey set a key to encrypt and decrypt a sensitive data
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

	for _, operation := range operationsLog {
		if _, err := am.HandleOperation(operation); err != nil {
			return fmt.Errorf(
				"failed to HandleOperation %s (this error is fatal, the state can not be recovered): %w",
				operation.ID, err)
		}
	}

	log.Println("Successfully replayed Operation log")

	return nil
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

// HandleOperation handles and processes an operation
func (am *Machine) HandleOperation(operation client.Operation) (client.Operation, error) {
	if err := am.storeOperation(operation); err != nil {
		return client.Operation{}, fmt.Errorf("failed to storeOperation: %w", err)
	}

	return am.handleOperation(operation)
}

func (am *Machine) handleOperation(operation client.Operation) (client.Operation, error) {
	var (
		err error
	)

	// handler gets a pointer to an operation, do necessary things
	// and write a result (or an error) to .Result field of operation
	switch fsm.State(operation.Type) {
	case signature_proposal_fsm.StateAwaitParticipantsConfirmations:
		err = am.handleStateAwaitParticipantsConfirmations(&operation)
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
		log.Println(fmt.Sprintf("failed to handle operation %s, returning response with error to client: %v",
			operation.Type, err))
		if e := am.writeErrorRequestToOperation(&operation, err); e != nil {
			return operation, fmt.Errorf("failed to write error request to an operation: %w", e)
		}
	}

	return operation, nil
}

// HandleQR - gets an operation from a QR code, do necessary things for the operation and returns paths to QR-code images
func (am *Machine) HandleQR() (string, error) {
	var (
		err error

		// input operation
		operation client.Operation
		qrData    []byte

		resultOperation client.Operation
	)

	if qrData, err = am.qrProcessor.ReadQR(); err != nil {
		return "", fmt.Errorf("failed to read QR: %w", err)
	}
	if err = json.Unmarshal(qrData, &operation); err != nil {
		return "", fmt.Errorf("failed to unmarshal operation: %w", err)
	}

	if resultOperation, err = am.HandleOperation(operation); err != nil {
		return "", err
	}

	operationBz, err := json.Marshal(resultOperation)
	if err != nil {
		return "", fmt.Errorf("failed to marshal operation: %w", err)
	}

	qrPath := filepath.Join(am.resultQRFolder, fmt.Sprintf("dc4bc_qr_%s-response.gif", resultOperation.ID))
	if err = am.qrProcessor.WriteQR(qrPath, operationBz); err != nil {
		return "", fmt.Errorf("failed to write QR: %w", err)
	}

	return qrPath, nil
}

// writeErrorRequestToOperation writes error to a operation if some bad things happened
func (am *Machine) writeErrorRequestToOperation(o *client.Operation, handlerError error) error {
	// each type of request should have a required event even error
	// maybe should be global?
	eventToErrorMap := map[fsm.State]fsm.Event{
		dkg_proposal_fsm.StateDkgCommitsAwaitConfirmations:   dkg_proposal_fsm.EventDKGCommitConfirmationError,
		dkg_proposal_fsm.StateDkgDealsAwaitConfirmations:     dkg_proposal_fsm.EventDKGDealConfirmationError,
		dkg_proposal_fsm.StateDkgResponsesAwaitConfirmations: dkg_proposal_fsm.EventDKGResponseConfirmationError,
		dkg_proposal_fsm.StateDkgMasterKeyAwaitConfirmations: dkg_proposal_fsm.EventDKGMasterKeyConfirmationError,
	}
	pid, err := am.getParticipantID(o.DKGIdentifier)
	if err != nil {
		return fmt.Errorf("failed to get participant id: %w", err)
	}
	req := requests.DKGProposalConfirmationErrorRequest{
		Error:         handlerError,
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
