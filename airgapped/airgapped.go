package airgapped

import (
	"encoding/json"
	"fmt"
	client "github.com/depools/dc4bc/client/types"
	"github.com/depools/dc4bc/dkg"
	"github.com/depools/dc4bc/fsm/fsm"
	"github.com/depools/dc4bc/fsm/state_machines/dkg_proposal_fsm"
	"github.com/depools/dc4bc/fsm/types/requests"
	"github.com/depools/dc4bc/qr"
	"github.com/syndtr/goleveldb/leveldb"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/encrypt/ecies"
	"go.dedis.ch/kyber/v3/pairing/bn256"
	"log"
	"sync"
)

const (
	resultQRFolder  = "result_qr_codes"
	pubKeyDBKey     = "public_key"
	privateKeyDBKey = "private_key"
)

type AirgappedMachine struct {
	sync.Mutex

	dkgInstances map[string]*dkg.DKG
	qrProcessor  qr.Processor

	pubKey kyber.Point
	secKey kyber.Scalar
	suite  *bn256.Suite

	db *leveldb.DB
}

func NewAirgappedMachine(dbPath string) (*AirgappedMachine, error) {
	var (
		err error
	)

	am := &AirgappedMachine{
		dkgInstances: make(map[string]*dkg.DKG),
		qrProcessor:  qr.NewCameraProcessor(),
	}

	am.suite = bn256.NewSuiteG2()

	if am.db, err = leveldb.OpenFile(dbPath, nil); err != nil {
		return nil, fmt.Errorf("failed to open db file %s for keys: %w", dbPath, err)
	}

	err = am.loadKeysFromDB(dbPath)
	if err != nil && err != leveldb.ErrNotFound {
		return nil, fmt.Errorf("failed to load keys from db %s: %w", dbPath, err)
	}
	// if keys were not generated yet
	if err == leveldb.ErrNotFound {
		am.secKey = am.suite.Scalar().Pick(am.suite.RandomStream())
		am.pubKey = am.suite.Point().Mul(am.secKey, nil)

		return am, am.saveKeysToDB(dbPath)
	}

	return am, nil
}

func (am *AirgappedMachine) loadKeysFromDB(dbPath string) error {
	pubKeyBz, err := am.db.Get([]byte(pubKeyDBKey), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return err
		}
		return fmt.Errorf("failed to get public key from db %s: %w", dbPath, err)
	}

	privateKeyBz, err := am.db.Get([]byte(privateKeyDBKey), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return err
		}
		return fmt.Errorf("failed to get private key from db %s: %w", dbPath, err)
	}

	am.pubKey = am.suite.Point()
	if err = am.pubKey.UnmarshalBinary(pubKeyBz); err != nil {
		return fmt.Errorf("failed to unmarshal public key: %w", err)
	}

	am.secKey = am.suite.Scalar()
	if err = am.secKey.UnmarshalBinary(privateKeyBz); err != nil {
		return fmt.Errorf("failed to unmarshal private key: %w", err)
	}
	return nil
}

func (am *AirgappedMachine) saveKeysToDB(dbPath string) error {

	pubKeyBz, err := am.pubKey.MarshalBinary()
	if err != nil {
		return fmt.Errorf("failed to marshal pub key: %w", err)
	}
	privateKeyBz, err := am.secKey.MarshalBinary()
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}

	tx, err := am.db.OpenTransaction()
	if err != nil {
		return fmt.Errorf("failed to open transcation for db %s: %w", dbPath, err)
	}
	defer tx.Discard()

	if err = tx.Put([]byte(pubKeyDBKey), pubKeyBz, nil); err != nil {
		return fmt.Errorf("failed to put pub key into db %s: %w", dbPath, err)
	}

	if err = tx.Put([]byte(privateKeyDBKey), privateKeyBz, nil); err != nil {
		return fmt.Errorf("failed to put private key into db %s: %w", dbPath, err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit tx for saving keys into db %s: %w", dbPath, err)
	}
	return nil
}

func (am *AirgappedMachine) getParticipantID(dkgIdentifier string) (int, error) {
	dkgInstance, ok := am.dkgInstances[dkgIdentifier]
	if !ok {
		return 0, fmt.Errorf("invalid dkg identifier: %s", dkgIdentifier)
	}
	return dkgInstance.ParticipantID, nil
}

func (am *AirgappedMachine) encryptData(dkgIdentifier, to string, data []byte) ([]byte, error) {
	dkgInstance, ok := am.dkgInstances[dkgIdentifier]
	if !ok {
		return nil, fmt.Errorf("invalid dkg identifier: %s", dkgIdentifier)
	}

	pk, err := dkgInstance.GetPubKeyByParticipant(to)
	if err != nil {
		return nil, fmt.Errorf("failed to get pk for participant %s: %w", to, err)
	}

	encryptedData, err := ecies.Encrypt(am.suite, pk, data, am.suite.Hash)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt data: %w", err)
	}
	return encryptedData, nil
}

func (am *AirgappedMachine) decryptData(data []byte) ([]byte, error) {
	decryptedData, err := ecies.Decrypt(am.suite, am.secKey, data, am.suite.Hash)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}
	return decryptedData, nil
}

func (am *AirgappedMachine) HandleOperation(operation client.Operation) ([]client.Operation, error) {
	var (
		err error
		// output operations (cause of deals)
		operations []client.Operation
	)

	am.Lock()
	defer am.Unlock()

	// handler gets a pointer to an operation, do necessary things
	// and write a result (or an error) to .Result field of operation
	switch fsm.State(operation.Type) {
	case dkg_proposal_fsm.StateDkgCommitsAwaitConfirmations:
		err = am.handleStateDkgCommitsAwaitConfirmations(&operation)
	case dkg_proposal_fsm.StateDkgDealsAwaitConfirmations:
		operations, err = am.handleStateDkgDealsAwaitConfirmations(operation)
	case dkg_proposal_fsm.StateDkgResponsesAwaitConfirmations:
		err = am.handleStateDkgResponsesAwaitConfirmations(&operation)
	case dkg_proposal_fsm.StateDkgMasterKeyAwaitConfirmations:
		err = am.handleStateDkgMasterKeyAwaitConfirmations(&operation)
	default:
		err = fmt.Errorf("invalid operation type: %s", operation.Type)
	}

	// if we have error after handling the operation, we write the error to the operation, so we can feed it to a FSM
	if err != nil {
		log.Println(fmt.Sprintf("failed to handle operation %s, returning response with errot to client: %v",
			operation.Type, err))
		if e := am.writeErrorRequestToOperation(&operation, err); e != nil {
			return nil, fmt.Errorf("failed to write error request to an operation: %w", e)
		}
	}

	if len(operation.Result) > 0 {
		operations = append(operations, operation)
	}

	return operations, nil
}

// HandleQR - gets an operation from a QR code, do necessary things for the operation and returns paths to QR-code images
func (am *AirgappedMachine) HandleQR() ([]string, error) {
	var (
		err error

		// input operation
		operation client.Operation
		qrData    []byte

		// output operations (cause of deals)
		operations []client.Operation
	)

	if qrData, err = am.qrProcessor.ReadQR(); err != nil {
		return nil, fmt.Errorf("failed to read QR: %w", err)
	}
	if err = json.Unmarshal(qrData, &operation); err != nil {
		return nil, fmt.Errorf("failed to unmarshal operation: %w", err)
	}

	if operations, err = am.HandleOperation(operation); err != nil {
		return nil, err
	}

	qrPath := "%s/%s_%s_%s.png"
	qrPaths := make([]string, 0, len(operations))
	for _, o := range operations {
		operationBz, err := json.Marshal(o)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal operation: %w", err)
		}

		if err = am.qrProcessor.WriteQR(fmt.Sprintf(qrPath, resultQRFolder, o.Type, o.ID, o.To), operationBz); err != nil {
			return nil, fmt.Errorf("failed to write QR")
		}
		qrPaths = append(qrPaths, qrPath)
	}

	return qrPaths, nil
}

func (am *AirgappedMachine) writeErrorRequestToOperation(o *client.Operation, handlerError error) error {
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
	o.Result = reqBz
	o.Event = errorEvent
	return nil
}
