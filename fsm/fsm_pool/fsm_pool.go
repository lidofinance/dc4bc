package fsm_pool

import (
	"errors"
	"github.com/p2p-org/dc4bc/fsm/fsm"
)

type IStateMachine interface {
	// Returns machine state from scope dump
	// For nil argument returns fsm with process initiation
	// Get() IStateMachine

	Name() string

	InitialState() string

	// Process event
	Do(event string, args ...interface{}) (*fsm.FSMResponse, error)

	GlobalInitialEvent() string

	EventsList() []string

	StatesSourcesList() []string

	IsFinState(state string) bool
}

type FSMMapper map[string]IStateMachine

type FSMRouteMapper map[string]string

type FSMPoolProvider struct {
	fsmInitialEvent string
	// Pool mapper by names
	mapper FSMMapper
	events FSMRouteMapper
	states FSMRouteMapper
}

func Init(machines ...IStateMachine) *FSMPoolProvider {
	if len(machines) == 0 {
		panic("cannot initialize empty pool")
	}
	p := &FSMPoolProvider{
		mapper: make(FSMMapper),
		events: make(FSMRouteMapper),
		states: make(FSMRouteMapper),
	}

	allInitStatesMap := make(map[string]string)

	// Fill up mapper
	for _, machine := range machines {

		if machine == nil {
			panic("machine not initialized, got nil")
		}

		machineName := machine.Name()

		if machineName == "" {
			panic("machine name cannot be empty")
		}

		if _, exists := p.mapper[machineName]; exists {
			panic("duplicate machine name")
		}

		allInitStatesMap[machine.InitialState()] = machineName

		machineEvents := machine.EventsList()
		for _, event := range machineEvents {
			if _, exists := p.events[event]; exists {
				panic("duplicate public event")
			}
			p.events[event] = machineName
		}

		// Setup entry event for machines pool if available
		if initialEvent := machine.GlobalInitialEvent(); initialEvent != "" {
			if p.fsmInitialEvent != "" {
				panic("duplicate entry event initialization")
			}

			p.fsmInitialEvent = initialEvent
		}

		p.mapper[machineName] = machine
	}

	// Second iteration, all initial states filled up
	// Fill up states with initial and exit states checking
	for _, machine := range machines {
		machineName := machine.Name()
		machineStates := machine.StatesSourcesList()
		for _, state := range machineStates {
			if machine.IsFinState(state) {
				// If state is initial for another machine,
				if initMachineName, exists := allInitStatesMap[state]; exists {
					p.states[allInitStatesMap[state]] = initMachineName
					continue
				}
			}
			if name, exists := p.states[state]; exists && name != machineName {
				panic("duplicate state for machines")
			}

			p.states[state] = machineName
		}
	}

	if p.fsmInitialEvent == "" {
		panic("machines pool entry event not set")
	}
	return p
}

func (p *FSMPoolProvider) EntryPointMachine() (IStateMachine, error) {
	// StateGlobalIdle
	// TODO: Short code
	entryStateMachineName := p.events[p.fsmInitialEvent]

	machine, exists := p.mapper[entryStateMachineName]

	if !exists || machine == nil {
		return nil, errors.New("cannot init machine with entry point")
	}
	return machine, nil
}

func (p *FSMPoolProvider) MachineByEvent(event string) (IStateMachine, error) {
	eventMachineName := p.events[event]
	machine, exists := p.mapper[eventMachineName]

	if !exists || machine == nil {
		return nil, errors.New("cannot init machine for event")
	}
	return machine, nil
}

func (p *FSMPoolProvider) MachineByState(state string) (IStateMachine, error) {
	eventMachineName := p.states[state]
	machine, exists := p.mapper[eventMachineName]

	if !exists || machine == nil {
		return nil, errors.New("cannot init machine for state")
	}
	return machine, nil
}
