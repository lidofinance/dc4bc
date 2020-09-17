package airgapped

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
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
	bls12381 "github.com/depools/kyber-bls12381"
	"github.com/syndtr/goleveldb/leveldb"
)

const (
	resultQRFolder        = "result_qr_codes"
	pubKeyDBKey           = "public_key"
	privateKeyDBKey       = "private_key"
	participantAddressKey = "participant_address"
)

type AirgappedMachine struct {
	sync.Mutex

	ParticipantAddress string

	dkgInstances map[string]*dkg.DKG
	qrProcessor  qr.Processor

	encryptionKey []byte
	pubKey        kyber.Point
	secKey        kyber.Scalar
	suite         vss.Suite

	db *leveldb.DB
}

func NewAirgappedMachine(dbPath string) (*AirgappedMachine, error) {
	var (
		err error
	)

	if err := os.MkdirAll(resultQRFolder, 0777); err != nil {
		if err != os.ErrExist {
			return nil, fmt.Errorf("failed to create folder %s: %w", resultQRFolder, err)
		}
	}

	am := &AirgappedMachine{
		dkgInstances: make(map[string]*dkg.DKG),
		qrProcessor:  qr.NewCameraProcessor(),
	}

	am.suite = bls12381.NewBLS12381Suite()

	if am.db, err = leveldb.OpenFile(dbPath, nil); err != nil {
		return nil, fmt.Errorf("failed to open db file %s for keys: %w", dbPath, err)
	}

	if err = am.loadAddressFromDB(dbPath); err != nil {
		return nil, fmt.Errorf("failed to load address from db")
	}

	return am, nil
}

func (am *AirgappedMachine) InitKeys() error {
	err := am.LoadKeysFromDB()
	if err != nil && err != leveldb.ErrNotFound {
		return fmt.Errorf("failed to load keys from db: %w", err)
	}
	// if keys were not generated yet
	if err == leveldb.ErrNotFound {
		am.secKey = am.suite.Scalar().Pick(am.suite.RandomStream())
		am.pubKey = am.suite.Point().Mul(am.secKey, nil)

		return am.SaveKeysToDB()
	}
	return nil
}

func (am *AirgappedMachine) SetAddress(address string) error {
	am.ParticipantAddress = address
	return am.saveAddressToDB(address)
}

func (am *AirgappedMachine) GetAddress() string {
	return am.ParticipantAddress
}

func (am *AirgappedMachine) SetEncryptionKey(key []byte) {
	am.encryptionKey = key
}

func (am *AirgappedMachine) SensitiveDataRemoved() bool {
	return len(am.encryptionKey) == 0
}

func (am *AirgappedMachine) DropSensitiveData() {
	am.Lock()
	defer am.Unlock()

	am.secKey = nil
	am.pubKey = nil
	am.encryptionKey = nil
}

func (am *AirgappedMachine) LoadKeysFromDB() error {
	pubKeyBz, err := am.db.Get([]byte(pubKeyDBKey), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return err
		}
		return fmt.Errorf("failed to get public key from db: %w", err)
	}

	privateKeyBz, err := am.db.Get([]byte(privateKeyDBKey), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return err
		}
		return fmt.Errorf("failed to get private key from db: %w", err)
	}

	decryptedPubKey, err := decrypt(am.encryptionKey, pubKeyBz)
	if err != nil {
		return err
	}
	decryptedPrivateKey, err := decrypt(am.encryptionKey, privateKeyBz)
	if err != nil {
		return err
	}

	am.pubKey = am.suite.Point()
	if err = am.pubKey.UnmarshalBinary(decryptedPubKey); err != nil {
		return fmt.Errorf("failed to unmarshal public key: %w", err)
	}

	am.secKey = am.suite.Scalar()
	if err = am.secKey.UnmarshalBinary(decryptedPrivateKey); err != nil {
		return fmt.Errorf("failed to unmarshal private key: %w", err)
	}
	return nil
}

func (am *AirgappedMachine) loadAddressFromDB(dbPath string) error {
	address, err := am.db.Get([]byte(participantAddressKey), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil
		}
		return fmt.Errorf("failed to get address from db %s: %w", dbPath, err)
	}
	am.ParticipantAddress = string(address)
	return nil
}

func (am *AirgappedMachine) saveAddressToDB(address string) error {
	return am.db.Put([]byte(participantAddressKey), []byte(address), nil)
}

func (am *AirgappedMachine) SaveKeysToDB() error {
	pubKeyBz, err := am.pubKey.MarshalBinary()
	if err != nil {
		return fmt.Errorf("failed to marshal pub key: %w", err)
	}
	privateKeyBz, err := am.secKey.MarshalBinary()
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}

	encryptedPubKey, err := encrypt(am.encryptionKey, pubKeyBz)
	if err != nil {
		return err
	}
	encryptedPrivateKey, err := encrypt(am.encryptionKey, privateKeyBz)
	if err != nil {
		return err
	}

	tx, err := am.db.OpenTransaction()
	if err != nil {
		return fmt.Errorf("failed to open transcation for db: %w", err)
	}
	defer tx.Discard()

	if err = tx.Put([]byte(pubKeyDBKey), encryptedPubKey, nil); err != nil {
		return fmt.Errorf("failed to put pub key into db: %w", err)
	}

	if err = tx.Put([]byte(privateKeyDBKey), encryptedPrivateKey, nil); err != nil {
		return fmt.Errorf("failed to put private key into db: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit tx for saving keys into db: %w", err)
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

func (am *AirgappedMachine) encryptDataForParticipant(dkgIdentifier, to string, data []byte) ([]byte, error) {
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

func (am *AirgappedMachine) decryptDataFromParticipant(data []byte) ([]byte, error) {
	decryptedData, err := ecies.Decrypt(am.suite, am.secKey, data, am.suite.Hash)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}
	return decryptedData, nil
}

func (am *AirgappedMachine) HandleOperation(operation client.Operation) (client.Operation, error) {
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
func (am *AirgappedMachine) HandleQR() ([]string, error) {
	var (
		err error

		// input operation
		operation client.Operation
		qrData    []byte

		resultOperation client.Operation
	)

	if qrData, err = qr.ReadDataFromQRChunks(am.qrProcessor); err != nil {
		return nil, fmt.Errorf("failed to read QR: %w", err)
	}
	if err = json.Unmarshal(qrData, &operation); err != nil {
		return nil, fmt.Errorf("failed to unmarshal operation: %w", err)
	}

	if resultOperation, err = am.HandleOperation(operation); err != nil {
		return nil, err
	}

	operationBz, err := json.Marshal(resultOperation)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal operation: %w", err)
	}

	chunks, err := qr.DataToChunks(operationBz)
	if err != nil {
		return nil, fmt.Errorf("failed to divide a data on chunks: %w", err)
	}
	qrPaths := make([]string, 0, len(chunks))

	for idx, chunk := range chunks {
		qrPath := fmt.Sprintf("%s/%s_%s_%s-%d.png", resultQRFolder, resultOperation.Type, resultOperation.ID,
			resultOperation.To, idx)
		if err = am.qrProcessor.WriteQR(qrPath, chunk); err != nil {
			return nil, fmt.Errorf("failed to write QR: %w", err)
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
	o.Event = errorEvent
	o.ResultMsgs = append(o.ResultMsgs, createMessage(*o, reqBz))
	return nil
}
