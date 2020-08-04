package state_machines

import (
	"encoding/json"
	"errors"

	"github.com/depools/dc4bc/fsm/fsm"
	"github.com/depools/dc4bc/fsm/fsm_pool"
	"github.com/depools/dc4bc/fsm/state_machines/internal"
	"github.com/depools/dc4bc/fsm/state_machines/signature_construct_fsm"
	"github.com/depools/dc4bc/fsm/state_machines/signature_proposal_fsm"
)

// Is machine state scope dump will be locked?
type FSMDump struct {
	Id      string
	State   fsm.State
	Payload internal.MachineStatePayload
}

type FSMInstance struct {
	machine fsm_pool.MachineProvider
	dump    *FSMDump
}

var (
	fsmPoolProvider *fsm_pool.FSMPool
)

func init() {
	fsmPoolProvider = fsm_pool.Init(
		signature_proposal_fsm.New(),
		signature_construct_fsm.New(),
	)
}

func New(data []byte) (*FSMInstance, error) {
	var err error
	i := &FSMInstance{}
	if len(data) == 0 {
		i.InitDump()
		i.machine, err = fsmPoolProvider.EntryPointMachine()
		return i, err // Create machine
	}

	err = i.dump.Unmarshal(data)

	if err != nil {
		return nil, errors.New("cannot read machine dump")
	}

	i.machine, err = fsmPoolProvider.MachineByState(i.dump.State)
	return i, err
}

func (i *FSMInstance) Do(event fsm.Event, args ...interface{}) (*fsm.Response, []byte, error) {
	// Provide payload as first argument ever
	result, err := i.machine.Do(event, append([]interface{}{i.dump.Payload}, args...)...)

	// On route errors result will be nil
	if result != nil {

		// Proxying combined response, separate payload and data
		if result.Data != nil {
			if r, ok := result.Data.(internal.MachineCombinedResponse); ok {
				i.dump.Payload = *r.Payload
				result.Data = r.Response
			} else {
				return nil, []byte{}, errors.New("cannot cast callback response")
			}
		}

		i.dump.State = result.State
	}
	dump, dumpErr := i.dump.Marshal()
	if dumpErr != nil {
		return result, []byte{}, err
	}

	return result, dump, err
}

func (i *FSMInstance) InitDump() {
	if i.dump == nil {
		i.dump = &FSMDump{
			State: fsm.StateGlobalIdle,
		}
	}
}

// TODO: Add encryption
func (d *FSMDump) Marshal() ([]byte, error) {
	return json.Marshal(d)
}

// TODO: Add decryption
func (d *FSMDump) Unmarshal(data []byte) error {
	return json.Unmarshal(data, d)
}
