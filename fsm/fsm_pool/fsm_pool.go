package fsm_pool

import (
	"errors"

	"github.com/depools/dc4bc/fsm/fsm"
)

type MachineProvider interface {
	// Returns machine state from scope dump
	// For nil argument returns fsm with process initiation
	// Get() MachineProvider

	Name() string

	InitialState() fsm.State

	// Process event
	Do(event fsm.Event, args ...interface{}) (*fsm.Response, error)

	GlobalInitialEvent() fsm.Event

	EventsList() []fsm.Event

	StatesSourcesList() []fsm.State

	IsFinState(state fsm.State) bool
}

type FSMMapper map[string]MachineProvider

type FSMEventsMapper map[fsm.Event]string

type FSMStatesMapper map[fsm.State]string

type FSMPool struct {
	fsmInitialEvent fsm.Event
	// Pool mapper by names
	mapper FSMMapper
	events FSMEventsMapper
	states FSMStatesMapper
}

func Init(machines ...MachineProvider) *FSMPool {
	if len(machines) == 0 {
		panic("cannot initialize empty pool")
	}
	p := &FSMPool{
		mapper: make(FSMMapper),
		events: make(FSMEventsMapper),
		states: make(FSMStatesMapper),
	}

	allInitStatesMap := make(map[fsm.State]string)

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
					p.states[state] = initMachineName
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

func (p *FSMPool) EntryPointMachine() (MachineProvider, error) {
	// StateGlobalIdle
	// TODO: Short code
	entryStateMachineName := p.events[p.fsmInitialEvent]

	machine, exists := p.mapper[entryStateMachineName]

	if !exists || machine == nil {
		return nil, errors.New("cannot init machine with entry point")
	}
	return machine, nil
}

func (p *FSMPool) MachineByEvent(event fsm.Event) (MachineProvider, error) {
	eventMachineName := p.events[event]
	machine, exists := p.mapper[eventMachineName]

	if !exists || machine == nil {
		return nil, errors.New("cannot init machine for event")
	}
	return machine, nil
}

func (p *FSMPool) MachineByState(state fsm.State) (MachineProvider, error) {
	eventMachineName := p.states[state]
	machine, exists := p.mapper[eventMachineName]

	if !exists || machine == nil {
		return nil, errors.New("cannot init machine for state")
	}
	return machine, nil
}
