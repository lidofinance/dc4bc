package state_machines

import (
	"encoding/json"
	"errors"
	"github.com/depools/dc4bc/fsm/state_machines/dkg_proposal_fsm"
	"strings"

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

var (
	fsmPoolProvider *fsm_pool.FSMPool
)

func init() {
	fsmPoolProvider = fsm_pool.Init(
		signature_proposal_fsm.New(),
		dkg_proposal_fsm.New(),
	)
}

// Transaction id required for unique identify dump
func Create(tid string) (*FSMInstance, error) {
	var err error
	i := &FSMInstance{}
	err = i.InitDump(tid)

	if err != nil {
		return nil, err
	}

	machine, err := fsmPoolProvider.EntryPointMachine()
	i.machine = machine.(internal.DumpedMachineProvider)
	i.machine.SetUpPayload(i.dump.Payload)
	return i, err
}

func FromDump(data []byte) (*FSMInstance, error) {
	var err error

	i := &FSMInstance{}
	err = i.dump.Unmarshal(data)

	if err != nil {
		return nil, errors.New("cannot read machine dump")
	}

	machine, err := fsmPoolProvider.MachineByState(i.dump.State)
	i.machine = machine.(internal.DumpedMachineProvider)
	i.machine.SetUpPayload(i.dump.Payload)
	return i, err
}

func (i *FSMInstance) Do(event fsm.Event, args ...interface{}) (result *fsm.Response, dump []byte, err error) {
	var dumpErr error

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

func (i *FSMInstance) InitDump(tid string) error {
	if i.dump != nil {
		return errors.New("dump already initialized")
	}

	tid = strings.TrimSpace(tid)

	if tid == "" {
		return errors.New("empty transaction id")
	}

	i.dump = &FSMDump{
		State: fsm.StateGlobalIdle,
		Payload: &internal.DumpedMachineStatePayload{
			TransactionId:               tid,
			ConfirmationProposalPayload: nil,
			DKGProposalPayload:          nil,
		},
	}
	return nil
}

// TODO: Add encryption
func (d *FSMDump) Marshal() ([]byte, error) {
	return json.Marshal(d)
}

// TODO: Add decryption
func (d *FSMDump) Unmarshal(data []byte) error {
	return json.Unmarshal(data, d)
}
