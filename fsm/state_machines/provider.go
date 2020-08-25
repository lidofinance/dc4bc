package state_machines

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"github.com/depools/dc4bc/fsm/state_machines/signing_proposal_fsm"
	"strings"

	"github.com/depools/dc4bc/fsm/state_machines/dkg_proposal_fsm"

	"github.com/depools/dc4bc/fsm/fsm"
	"github.com/depools/dc4bc/fsm/fsm_pool"
	"github.com/depools/dc4bc/fsm/state_machines/internal"
	"github.com/depools/dc4bc/fsm/state_machines/signature_proposal_fsm"
)

// Is machine state scope dump will be locked?
type FSMDump struct {
	TransactionId string
	State         fsm.State
	Payload       *internal.DumpedMachineStatePayload
}

type FSMInstance struct {
	machine internal.DumpedMachineProvider
	dump    *FSMDump
}

// Create new fsm with unique id
// transactionId required for unique identify dump
func Create(dkgID string) (*FSMInstance, error) {
	var (
		err error
		i   = &FSMInstance{}
	)

	err = i.InitDump(dkgID)
	if err != nil {
		return nil, err
	}

	fsmPoolProvider := fsm_pool.Init(
		signature_proposal_fsm.New(),
		dkg_proposal_fsm.New(),
		signing_proposal_fsm.New(),
	)

	machine, err := fsmPoolProvider.EntryPointMachine()
	i.machine = machine.(internal.DumpedMachineProvider).
		WithSetup(i.dump.State, i.dump.Payload)
	return i, err
}

// DKGQuorumGet fsm from dump
func FromDump(data []byte) (*FSMInstance, error) {
	var err error

	if len(data) < 2 {
		return nil, errors.New("machine dump is empty")
	}

	fsmPoolProvider := fsm_pool.Init(
		signature_proposal_fsm.New(),
		dkg_proposal_fsm.New(),
		signing_proposal_fsm.New(),
	)

	i := &FSMInstance{
		dump: &FSMDump{},
	}
	err = i.dump.Unmarshal(data)

	// TODO: Add logger
	if err != nil {
		return nil, errors.New("cannot read machine dump")
	}

	machine, err := fsmPoolProvider.MachineByState(i.dump.State)
	if err != nil {
		return nil, err
	}

	i.machine = machine.(internal.DumpedMachineProvider).
		WithSetup(i.dump.State, i.dump.Payload)
	return i, err
}

func (i *FSMInstance) GetPubKeyByAddr(addr string) (ed25519.PublicKey, error) {
	if i.dump == nil {
		return nil, errors.New("dump not initialized")
	}

	return i.dump.Payload.GetPubKeyByAddr(addr)
}

func (i *FSMInstance) Do(event fsm.Event, args ...interface{}) (result *fsm.Response, dump []byte, err error) {
	var dumpErr error

	if i.machine == nil {
		return nil, []byte{}, errors.New("machine is not initialized")
	}

	result, err = i.machine.Do(event, args...)

	// On route errors result will be nil
	if result != nil {
		i.dump.State = result.State

		dump, dumpErr = i.dump.Marshal()
		if dumpErr != nil {
			return result, []byte{}, err
		}
	}

	return result, dump, err
}

func (i *FSMInstance) InitDump(dkgID string) error {
	if i.dump != nil {
		return errors.New("dump already initialized")
	}

	dkgID = strings.TrimSpace(dkgID)

	if dkgID == "" {
		return errors.New("empty {dkgID}")
	}

	i.dump = &FSMDump{
		TransactionId: dkgID,
		State:         fsm.StateGlobalIdle,
		Payload: &internal.DumpedMachineStatePayload{
			DkgId:                    dkgID,
			SignatureProposalPayload: nil,
			DKGProposalPayload:       nil,
		},
	}
	return nil
}

func (i *FSMInstance) State() (fsm.State, error) {
	if i.machine == nil {
		return "", errors.New("machine is not initialized")
	}
	return i.machine.State(), nil
}

func (i *FSMInstance) Id() string {
	if i.dump != nil {
		return i.dump.TransactionId
	}
	return ""
}

func (i *FSMInstance) Dump() ([]byte, error) {
	if i.dump == nil {
		return []byte{}, errors.New("dump is not initialized")
	}
	return i.dump.Marshal()
}

// TODO: Add encryption
func (d *FSMDump) Marshal() ([]byte, error) {
	return json.Marshal(d)
}

// TODO: Add decryption
func (d *FSMDump) Unmarshal(data []byte) error {
	if d == nil {
		return errors.New("dump is not initialized")
	}

	return json.Unmarshal(data, d)
}
