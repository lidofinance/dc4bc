package fsmservice

import (
	"encoding/json"
	"fmt"
	"github.com/lidofinance/dc4bc/client/api/dto"
	"github.com/lidofinance/dc4bc/client/modules/state"
	"github.com/lidofinance/dc4bc/fsm/state_machines"
	"github.com/lidofinance/dc4bc/storage"
	"github.com/lidofinance/dc4bc/storage/kafka_storage"
)

const (
	FSMStateKey = "fsm_state"
)

type FSMService interface {
	GetFSMInstance(dkgRoundID string) (*state_machines.FSMInstance, error)
	GetFSMDump(dto *dto.DkgIdDTO) (*state_machines.FSMDump, error)
	GetFSMList() (map[string]string, error)
	ResetFSMState(dto *dto.ResetStateDTO) (string, error)
	SaveFSM(dkgRoundID string, dump []byte) error
}

type FSM struct {
	state    state.State
	storage  storage.Storage
	stateKey string
}

func NewFSMService(state state.State, storage storage.Storage, stateNamespace string) FSMService {
	return &FSM{
		state:    state,
		storage:  storage,
		stateKey: fmt.Sprintf("%s_%s", stateNamespace, FSMStateKey),
	}
}

func (fsm *FSM) getStateKey() string {
	return fsm.stateKey
}

func (fsm *FSM) getAllFSMData() (fsmInstances map[string][]byte, err error) {
	bz, err := fsm.state.Get(fsm.getStateKey())
	if err != nil {
		return nil, fmt.Errorf("failed to get FSM instances: %w", err)
	}

	fsmInstances = map[string][]byte{}
	if len(bz) > 0 {
		if err := json.Unmarshal(bz, &fsmInstances); err != nil {
			return nil, fmt.Errorf("failed to unmarshal FSM instances: %w", err)
		}
	}
	return fsmInstances, nil
}

func (fsm *FSM) loadFSM(dkgRoundID string) (*state_machines.FSMInstance, bool, error) {
	fsmInstances, err := fsm.getAllFSMData()
	if err != nil {
		return nil, false, fmt.Errorf("failed to get fsm instances: %w", err)
	}

	fsmInstanceBz, ok := fsmInstances[dkgRoundID]
	if !ok {
		return nil, false, nil
	}

	fsmInstance, err := state_machines.FromDump(fsmInstanceBz)
	if err != nil {
		return nil, false, fmt.Errorf("failed to restore FSM instance from dump: %w", err)
	}

	return fsmInstance, ok, nil
}

func (fsm *FSM) SaveFSM(dkgRoundID string, dump []byte) error {
	fsmInstances, err := fsm.getAllFSMData()
	if err != nil {
		return fmt.Errorf("failed to get fsm instances: %w", err)
	}

	fsmInstances[dkgRoundID] = dump

	fsmInstancesBz, err := json.Marshal(fsmInstances)
	if err != nil {
		return fmt.Errorf("failed to marshal FSM instances: %w", err)
	}

	if err := fsm.state.Set(fsm.getStateKey(), fsmInstancesBz); err != nil {
		return fmt.Errorf("failed to save fsm state: %w", err)
	}

	return nil
}

// GetFSMInstance returns FSM for a necessary DKG round.
func (fsm *FSM) GetFSMInstance(dkgRoundID string) (*state_machines.FSMInstance, error) {
	var err error
	fsmInstance, ok, err := fsm.loadFSM(dkgRoundID)
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

		if err := fsm.SaveFSM(dkgRoundID, bz); err != nil {
			return nil, fmt.Errorf("failed to SaveFSM: %w", err)
		}
	}

	return fsmInstance, nil
}

func (fsm *FSM) GetFSMDump(dto *dto.DkgIdDTO) (*state_machines.FSMDump, error) {
	fsmInstance, err := fsm.GetFSMInstance(dto.DkgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get FSM instance for DKG round ID %s: %w", dto.DkgID, err)
	}
	return fsmInstance.FSMDump(), nil
}

func (fsm *FSM) GetAllFSM() (map[string]*state_machines.FSMInstance, error) {
	fsmInstancesBz, err := fsm.getAllFSMData()
	if err != nil {
		return nil, fmt.Errorf("failed to get fsm instances: %w", err)
	}

	fsmInstances := make(map[string]*state_machines.FSMInstance, len(fsmInstancesBz))
	for k, v := range fsmInstancesBz {
		fsmInstances[k], err = state_machines.FromDump(v)
		if err != nil {
			return nil, fmt.Errorf("failed to restore FSM instance from dump: %w", err)
		}
	}
	return fsmInstances, nil
}

func (fsm *FSM) GetFSMList() (map[string]string, error) {
	fsmInstances, err := fsm.GetAllFSM()

	if err != nil {
		return nil, fmt.Errorf("failed to get all FSM instances: %w", err)
	}

	fsmInstancesStates := make(map[string]string, len(fsmInstances))
	for k, v := range fsmInstances {
		fsmState, err := v.State()
		if err != nil {
			return nil, fmt.Errorf("failed to get FSM state: %w", err)
		}
		fsmInstancesStates[k] = fsmState.String()
	}

	return fsmInstancesStates, nil
}

func (fsm *FSM) ResetFSMState(dto *dto.ResetStateDTO) (string, error) {

	if err := fsm.storage.IgnoreMessages(dto.Messages, dto.UseOffset); err != nil {
		return "", fmt.Errorf("failed to ignore messages while resetting state: %w", err)
	}

	switch stg := fsm.storage.(type) {
	case *kafka_storage.KafkaStorage:
		if err := stg.SetConsumerGroup(dto.KafkaConsumerGroup); err != nil {
			return "", fmt.Errorf("failed to set consumer group while reseting state: %w", err)
		}
	}

	newstatepath, err := fsm.state.Reset(dto.NewStateDBDSN)
	if err != nil {
		return "", fmt.Errorf("failed to create new state from old: %w", err)
	}

	return newstatepath, err
}
