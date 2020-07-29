package fsm

import (
	"errors"
	"sync"
)

//
//  fsmInstance, err := fsm.New(scope)
//  if err != nil {
//     log.Println(err)
//     return
//  }
//
//  fsmInstance.Do(event, args)
//

// Temporary global finish state for deprecating operations
const (
	StateGlobalIdle = "__idle"
	StateGlobalDone = "__done"
)

// FSMResponse returns result for processing with client events
type FSMResponse struct {
	// Returns machine execution result state
	State string
	// Must be cast, according to mapper event_name->response_type
	Data interface{}
}

type FSM struct {
	name         string
	initialState string
	currentState string

	// May be mapping must require pair source + event?
	transitions map[trKey]*trEvent

	callbacks Callbacks

	initialEvent string

	// Finish states, for switch machine or fin,
	// These states cannot be linked as SrcState in this machine
	finStates map[string]bool

	// stateMu guards access to the currentState state.
	stateMu sync.RWMutex
	// eventMu guards access to State() and Transition().
	eventMu sync.Mutex
}

// Transition key source + dst
type trKey struct {
	source string
	event  string
}

// Transition lightweight event description
type trEvent struct {
	dstState   string
	isInternal bool
}

type EventDesc struct {
	Name string

	SrcState []string

	// Dst state changes after callback
	DstState string

	// Internal events, cannot be emitted from external call
	IsInternal bool
}

type Callback func(event string, args ...interface{}) (interface{}, error)

type Callbacks map[string]Callback

// TODO: Exports
func MustNewFSM(name, initial string, events []EventDesc, callbacks map[string]Callback) *FSM {
	// Add validation, chains building

	if name == "" {
		panic("name cannot be empty")
	}

	if initial == "" {
		panic("initialState state cannot be empty")
	}

	// to remove
	if len(events) == 0 {
		panic("cannot init fsm with empty events")
	}

	f := &FSM{
		name:         name,
		currentState: initial,
		initialState: initial,
		transitions:  make(map[trKey]*trEvent),
		finStates:    make(map[string]bool),
		callbacks:    make(map[string]Callback),
	}

	allEvents := make(map[string]bool)

	// Required for find finStates
	allSources := make(map[string]bool)
	allStates := make(map[string]bool)

	// Validate events
	for _, event := range events {

		if event.Name == "" {
			panic("cannot init empty event")
		}

		// TODO: Check transition when all events added
		if len(event.SrcState) == 0 {
			panic("event must have min one source available state")
		}

		if event.DstState == "" {
			panic("event dest cannot be empty, use StateGlobalDone for finish or external state")
		}

		if _, ok := allEvents[event.Name]; ok {
			panic("duplicate event")
		}

		allEvents[event.Name] = true
		allStates[event.DstState] = true

		for _, sourceState := range event.SrcState {
			tKey := trKey{
				sourceState,
				event.Name,
			}

			if sourceState == StateGlobalDone {
				panic("StateGlobalDone cannot set as source state")
			}

			if _, ok := f.transitions[tKey]; ok {
				panic("duplicate dst for pair `source + event`")
			}

			f.transitions[tKey] = &trEvent{event.DstState, event.IsInternal}

			// For using provider, event must use with IsGlobal = true
			if sourceState == initial {
				if f.initialEvent != "" {
					panic("machine entry event already exist")
				}
				f.initialEvent = event.Name
			}

			allSources[sourceState] = true
		}
	}

	if len(allStates) < 2 {
		panic("machine must contain at least two states")
	}

	// Validate callbacks
	for event, callback := range callbacks {
		if event == "" {
			panic("callback name cannot be empty")
		}

		if _, ok := allEvents[event]; !ok {
			panic("callback has no event")
		}

		f.callbacks[event] = callback
	}

	for state := range allStates {
		if state == StateGlobalIdle {
			continue
		}
		// Exit states cannot be a source in this machine
		if _, exists := allSources[state]; !exists || state == StateGlobalDone {
			f.finStates[state] = true
		}
	}

	if len(f.finStates) == 0 {
		panic("cannot initialize machine without final states")
	}

	return f
}

func (f *FSM) Do(event string, args ...interface{}) (resp *FSMResponse, err error) {
	f.eventMu.Lock()
	defer f.eventMu.Unlock()

	trEvent, ok := f.transitions[trKey{f.currentState, event}]
	if !ok {
		return nil, errors.New("cannot execute event for this state")
	}
	if trEvent.isInternal {
		return nil, errors.New("event is internal")
	}

	resp = &FSMResponse{
		State: f.State(),
	}

	if callback, ok := f.callbacks[event]; ok {
		resp.Data, err = callback(event, args...)
		// Do not try change state on error
		if err != nil {
			return resp, err
		}
	}

	err = f.setState(event)
	return
}

// State returns the currentState state of the FSM.
func (f *FSM) State() string {
	f.stateMu.RLock()
	defer f.stateMu.RUnlock()
	return f.currentState
}

// setState allows the user to move to the given state from currentState state.
// The call does not trigger any callbacks, if defined.
func (f *FSM) setState(event string) error {
	f.stateMu.Lock()
	defer f.stateMu.Unlock()

	trEvent, ok := f.transitions[trKey{f.currentState, event}]
	if !ok {
		return errors.New("cannot change state")
	}

	f.currentState = trEvent.dstState

	return nil
}

func (f *FSM) Name() string {
	return f.name
}

func (f *FSM) InitialState() string {
	return f.initialState
}

// Check entry event for available emitting as global entry event
func (f *FSM) GlobalInitialEvent() (event string) {
	if initialEvent, exists := f.transitions[trKey{StateGlobalIdle, f.initialEvent}]; exists {
		if !initialEvent.isInternal {
			event = f.initialEvent
		}
	}
	return
}

func (f *FSM) EntryEvent() (event string) {
	if entryEvent, exists := f.transitions[trKey{f.initialState, f.initialEvent}]; exists {
		if !entryEvent.isInternal {
			event = f.initialEvent
		}
	}
	return
}

func (f *FSM) EventsList() (events []string) {
	if len(f.transitions) > 0 {
		for trKey, trEvent := range f.transitions {
			if !trEvent.isInternal {
				events = append(events, trKey.event)
			}
		}
	}
	return
}

func (f *FSM) StatesList() (states []string) {
	allStates := map[string]bool{}
	if len(f.transitions) > 0 {
		for trKey, _ := range f.transitions {
			allStates[trKey.source] = true
		}
	}

	if len(allStates) > 0 {
		for state := range allStates {
			states = append(states, state)
		}
	}

	return
}

func (f *FSM) IsFinState(state string) bool {
	_, exists := f.finStates[state]
	return exists
}
