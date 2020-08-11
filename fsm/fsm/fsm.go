package fsm

import (
	"errors"
	"fmt"
	"strings"
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
	StateGlobalIdle = State("__idle")
	StateGlobalDone = State("__done")
)

const (
	EventRunDefault EventRunMode = iota
	EventRunBefore
	EventRunAfter
)

type State string

func (s *State) String() string {
	return string(*s)
}

type Event string

func (e *Event) String() string {
	return string(*e)
}

func (e *Event) IsEmpty() bool {
	return e.String() == ""
}

type EventRunMode uint8

// Response returns result for processing with clientMocks events
type Response struct {
	// Returns machine execution result state
	State State
	// Must be cast, according to mapper event_name->response_type
	Data interface{}
}

type FSM struct {
	name         string
	initialState State
	currentState State

	// May be mapping must require pair source + event?
	transitions map[trKey]*trEvent

	autoTransitions map[State]*trEvent

	callbacks Callbacks

	initialEvent Event

	// Finish states, for switch machine or fin,
	// These states cannot be linked as SrcState in this machine
	finStates map[State]bool

	// stateMu guards access to the currentState state.
	stateMu sync.RWMutex
	// eventMu guards access to State() and Transition().
	eventMu sync.Mutex
}

// Transition key source + dst
type trKey struct {
	source State
	event  Event
}

// Transition lightweight event description
type trEvent struct {
	event      Event
	dstState   State
	isInternal bool
	isAuto     bool
	runMode    EventRunMode
}

type EventDesc struct {
	Name Event

	SrcState []State

	// Dst state changes after callback
	DstState State

	// Internal events, cannot be emitted from external call
	IsInternal bool

	// Event must run without manual call
	IsAuto bool

	AutoRunMode EventRunMode
}

type Callback func(event Event, args ...interface{}) (Event, interface{}, error)

type Callbacks map[Event]Callback

// TODO: Exports
func MustNewFSM(machineName string, initialState State, events []EventDesc, callbacks Callbacks) *FSM {
	machineName = strings.TrimSpace(machineName)
	initialState = State(strings.TrimSpace(initialState.String()))

	if machineName == "" {
		panic("machine name cannot be empty")
	}

	if initialState == "" {
		panic("initial state state cannot be empty")
	}

	// to remove
	if len(events) == 0 {
		panic("cannot init fsm with empty events")
	}

	f := &FSM{
		name:            machineName,
		currentState:    initialState,
		initialState:    initialState,
		transitions:     make(map[trKey]*trEvent),
		autoTransitions: make(map[State]*trEvent),
		finStates:       make(map[State]bool),
		callbacks:       make(map[Event]Callback),
	}

	allEvents := make(map[Event]bool)

	// Required for find finStates
	allSources := make(map[State]bool)
	allStates := make(map[State]bool)

	// Validate events
	for _, event := range events {
		event.Name = Event(strings.TrimSpace(event.Name.String()))
		event.DstState = State(strings.TrimSpace(event.DstState.String()))

		if event.Name == "" {
			panic("cannot init empty event")
		}

		if event.DstState == "" {
			panic("event dest cannot be empty, use StateGlobalDone for finish or external state")
		}

		if _, ok := allEvents[event.Name]; ok {
			panic(fmt.Sprintf("duplicate event \"%s\"", event.Name))
		}

		allEvents[event.Name] = true
		allStates[event.DstState] = true

		trimmedSourcesCounter := 0

		for _, sourceState := range event.SrcState {
			sourceState := State(strings.TrimSpace(sourceState.String()))

			if sourceState == "" {
				continue
			}

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

			if event.IsAuto && event.AutoRunMode == EventRunDefault {
				event.AutoRunMode = EventRunAfter
			}

			trEvent := &trEvent{
				tKey.event,
				event.DstState,
				event.IsInternal,
				event.IsAuto,
				event.AutoRunMode,
			}

			f.transitions[tKey] = trEvent

			// For using provider, event must use with IsGlobal = true
			if sourceState == initialState {
				if f.initialEvent == "" {
					f.initialEvent = event.Name
				}
			}

			if event.IsAuto {
				if event.AutoRunMode != EventRunBefore && event.AutoRunMode != EventRunAfter {
					panic("{AutoRunMode} not set for auto event")
				}

				if _, ok := f.autoTransitions[sourceState]; ok {
					panic(fmt.Sprintf(
						"auto event \"%s\" already exists for state \"%s\"",
						event.Name,
						sourceState,
					))
				}
				f.autoTransitions[sourceState] = trEvent
			}

			allSources[sourceState] = true
			trimmedSourcesCounter++
		}

		if trimmedSourcesCounter == 0 {
			panic("event must have minimum one source available state")
		}
	}

	if len(allStates) < 2 {
		panic("machine must contain at least two states")
	}

	// Validate callbacks
	for event, callback := range callbacks {
		if event == "" {
			panic("callback machineName cannot be empty")
		}

		if _, ok := allEvents[event]; !ok {
			panic("callback has empty event")
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

func (f *FSM) DoInternal(event Event, args ...interface{}) (resp *Response, err error) {
	trEvent, ok := f.transitions[trKey{f.currentState, event}]
	if !ok {
		return nil, errors.New(fmt.Sprintf("cannot execute event \"%s\" for state \"%s\"", event, f.currentState))
	}

	return f.do(trEvent, args...)
}

func (f *FSM) Do(event Event, args ...interface{}) (resp *Response, err error) {
	trEvent, ok := f.transitions[trKey{f.currentState, event}]
	if !ok {
		return nil, errors.New(fmt.Sprintf("cannot execute event \"%s\" for state \"%s\"", event, f.currentState))
	}
	if trEvent.isInternal {
		return nil, errors.New("event is internal")
	}

	return f.do(trEvent, args...)
}
func (f *FSM) do(trEvent *trEvent, args ...interface{}) (resp *Response, err error) {
	var outEvent Event
	// f.eventMu.Lock()
	// defer f.eventMu.Unlock()

	// Process auto event
	if autoEvent, ok := f.autoTransitions[f.State()]; ok {
		autoEventResp := &Response{
			State: f.State(),
		}
		if autoEvent.runMode == EventRunBefore {
			if callback, ok := f.callbacks[autoEvent.event]; ok {
				outEvent, autoEventResp.Data, err = callback(autoEvent.event, args...)
				if err != nil {
					return autoEventResp, err
				}
			}
			if outEvent.IsEmpty() || autoEvent.event == outEvent {
				err = f.SetState(autoEvent.event)
			} else {
				err = f.SetState(outEvent)
			}
			if err != nil {
				return autoEventResp, err
			}
		}
		outEvent = ""
	}

	resp = &Response{
		State: f.State(),
	}

	if callback, ok := f.callbacks[trEvent.event]; ok {
		outEvent, resp.Data, err = callback(trEvent.event, args...)
		// Do not try change state on error
		if err != nil {
			return resp, err
		}
	}

	// Set state when callback executed
	if outEvent.IsEmpty() || trEvent.event == outEvent {
		err = f.SetState(trEvent.event)
	} else {
		err = f.SetState(outEvent)
	}

	// Process auto event
	if autoEvent, ok := f.autoTransitions[f.State()]; ok {
		autoEventResp := &Response{
			State: f.State(),
		}
		if autoEvent.runMode == EventRunAfter {
			if callback, ok := f.callbacks[autoEvent.event]; ok {
				outEvent, autoEventResp.Data, err = callback(autoEvent.event, args...)
				if err != nil {
					return autoEventResp, err
				}
			}
			if outEvent.IsEmpty() || autoEvent.event == outEvent {
				err = f.SetState(autoEvent.event)
			} else {
				err = f.SetState(outEvent)
			}
			if err != nil {
				return autoEventResp, err
			}
		}
		outEvent = ""
	}

	resp.State = f.State()

	return
}

// State returns the currentState state of the FSM.
func (f *FSM) State() State {
	f.stateMu.RLock()
	defer f.stateMu.RUnlock()
	return f.currentState
}

// SetState allows the user to move to the given state from currentState state.
// The call does not trigger any callbacks, if defined.
func (f *FSM) SetState(event Event) error {
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

func (f *FSM) InitialState() State {
	return f.initialState
}

// Check entry event for available emitting as global entry event
func (f *FSM) GlobalInitialEvent() (event Event) {
	if initialEvent, exists := f.transitions[trKey{StateGlobalIdle, f.initialEvent}]; exists {
		if !initialEvent.isInternal {
			event = f.initialEvent
		}
	}
	return
}

func (f *FSM) EntryEvent() (event Event) {
	if entryEvent, exists := f.transitions[trKey{f.initialState, f.initialEvent}]; exists {
		if !entryEvent.isInternal {
			event = f.initialEvent
		}
	}
	return
}

func (f *FSM) EventsList() (events []Event) {
	var eventsMap = map[Event]bool{}
	if len(f.transitions) > 0 {
		for trKey, trEvent := range f.transitions {
			if !trEvent.isInternal {
				eventsMap[trKey.event] = true
				if _, exists := eventsMap[trKey.event]; !exists {

					events = append(events, trKey.event)
				}
			}
		}
	}

	if len(eventsMap) > 0 {
		for event := range eventsMap {
			events = append(events, event)
		}
	}

	return
}

func (f *FSM) StatesSourcesList() (states []State) {
	var allStates = map[State]bool{}
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

func (f *FSM) IsFinState(state State) bool {
	_, exists := f.finStates[state]
	return exists
}
