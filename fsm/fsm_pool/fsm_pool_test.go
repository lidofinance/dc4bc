package fsm_pool

import (
	"github.com/depools/dc4bc/fsm/fsm"
	"testing"
)

type testMachineFSM1 struct {
	*fsm.FSM
	data int
}

const (
	testVal1 = 100
	testVal2 = 17
)

const (
	fsm1Name = "fsm1"
	// Init process from global idle state
	fsm1StateInit = fsm.StateGlobalIdle
	// Set up data
	fsm1StateStage1 = fsm.State("state_fsm1_stage1")
	// Process data
	fsm1StateStage2 = fsm.State("state_fsm1_stage2")
	// Canceled with internal event
	fsm1StateCanceledByInternal = fsm.State("state_fsm1_canceled")
	// Canceled with external event
	fsm1StateCanceled2 = fsm.State("state_fsm1_canceled2")
	// Out endpoint to switch
	fsm1StateOutToFSM2 = fsm.State("state_fsm1_out_to_fsm2")

	// Events
	eventFSM1Init    = fsm.Event("event_fsm1_init")
	eventFSM1Cancel  = fsm.Event("event_fsm1_cancel")
	eventFSM1Process = fsm.Event("event_fsm1_process")

	// Internal events
	eventFSM1Internal         = fsm.Event("event_internal_fsm1")
	eventFSM1CancelByInternal = fsm.Event("event_internal_fsm1_cancel")
	eventFSM1InternalOut2     = fsm.Event("event_internal_fsm1_out")
)

var (
	testing1Events = []fsm.EventDesc{
		// Init
		{Name: eventFSM1Init, SrcState: []fsm.State{fsm1StateInit}, DstState: fsm1StateStage1, IsAuto: true, AutoRunMode: fsm.EventRunAfter},
		{Name: eventFSM1Internal, SrcState: []fsm.State{fsm1StateStage1}, DstState: fsm1StateStage2, IsInternal: true},

		// Cancellation events
		{Name: eventFSM1CancelByInternal, SrcState: []fsm.State{fsm1StateStage2}, DstState: fsm1StateCanceledByInternal, IsInternal: true},
		{Name: eventFSM1Cancel, SrcState: []fsm.State{fsm1StateStage2}, DstState: fsm1StateCanceled2},

		// Out
		{Name: eventFSM1Process, SrcState: []fsm.State{fsm1StateStage2}, DstState: fsm1StateOutToFSM2},
		{Name: eventFSM1InternalOut2, SrcState: []fsm.State{fsm1StateStage2}, DstState: fsm1StateOutToFSM2, IsInternal: true},
	}
)

func NewFSM1() MachineProvider {
	machine := &testMachineFSM1{}

	machine.FSM = fsm.MustNewFSM(
		fsm1Name,
		fsm1StateInit,
		testing1Events,
		fsm.Callbacks{
			eventFSM1Init:         machine.actionFSM1SetUpData,
			eventFSM1InternalOut2: machine.actionFSM1EmitOut2,
			eventFSM1Process:      machine.actionFSM1ProcessData,
		},
	)
	return machine
}

func (m *testMachineFSM1) actionFSM1SetUpData(inEvent fsm.Event, args ...interface{}) (outEvent fsm.Event, response interface{}, err error) {
	m.data = testVal1
	outEvent = eventFSM1Internal
	return
}

func (m *testMachineFSM1) actionFSM1ProcessData(inEvent fsm.Event, args ...interface{}) (outEvent fsm.Event, response interface{}, err error) {
	if len(args) == 1 {
		if val, ok := args[0].(int); ok {
			m.data -= val
		}
	}

	response = m.data
	return
}

func (m *testMachineFSM1) actionFSM1EmitOut2(inEvent fsm.Event, args ...interface{}) (outEvent fsm.Event, response interface{}, err error) {
	return
}

// Second test machine

type testMachineFSM2 struct {
	*fsm.FSM
	data int
}

const (
	fsm2Name = "fsm2"
	// Init process from global idle state
	fsm2StateInit = fsm1StateOutToFSM2
	// Process data
	fsm2StateStage1 = fsm.State("state_fsm2_stage1")
	fsm2StateStage2 = fsm.State("state_fsm2_stage2")
	// Canceled with internal event
	fsm2StateCanceledByInternal = fsm.State("state_fsm2_canceled")
	// Out endpoint to switch
	fsm2StateOutToFSM3 = fsm.State("state_fsm2_out_to_fsm3")

	// Events
	eventFSM2Init    = fsm.Event("event_fsm2_init")
	eventFSM2Process = fsm.Event("event_fsm2_process")

	// Internal events
	eventFSM2Internal         = fsm.Event("event_internal_fsm2")
	eventFSM2CancelByInternal = fsm.Event("event_internal_fsm2_cancel")
	eventFSM2InternalOut      = fsm.Event("event_internal_fsm2_out")
)

var (
	testing2Events = []fsm.EventDesc{
		// Init
		{Name: eventFSM2Init, SrcState: []fsm.State{fsm2StateInit}, DstState: fsm2StateStage1},
		{Name: eventFSM2Internal, SrcState: []fsm.State{fsm2StateStage1}, DstState: fsm2StateStage2, IsInternal: true},

		// Cancellation events
		{Name: eventFSM2CancelByInternal, SrcState: []fsm.State{fsm2StateStage2}, DstState: fsm2StateCanceledByInternal, IsInternal: true},

		// Out
		{Name: eventFSM2Process, SrcState: []fsm.State{fsm2StateStage2}, DstState: fsm.StateGlobalDone},
		{Name: eventFSM2InternalOut, SrcState: []fsm.State{fsm2StateStage2}, DstState: fsm.StateGlobalDone, IsInternal: true},
	}

	testing2Callbacks = fsm.Callbacks{}
)

func NewFSM2() MachineProvider {
	machine := &testMachineFSM1{}

	machine.FSM = fsm.MustNewFSM(
		fsm2Name,
		fsm2StateInit,
		testing2Events,
		testing2Callbacks,
	)
	return machine
}

func (m *testMachineFSM2) actionFSM2SetUpData(event fsm.Event, args ...interface{}) (response interface{}, err error) {
	return
}

func (m *testMachineFSM2) actionFSM2ProcessData(event fsm.Event, args ...interface{}) (response interface{}, err error) {
	return
}

func (m *testMachineFSM2) actionFSM2EmitOut2(event fsm.Event, args ...interface{}) (response interface{}, err error) {
	return
}

var testPoolProvider *FSMPool

func init() {
	testPoolProvider = Init(
		NewFSM1(),
		NewFSM2(),
	)
}

func TestFSMPool_Init_EventsMap(t *testing.T) {
	if len(testPoolProvider.events) == 0 {
		t.Errorf("expected initialized events map")
	}
}

func TestFSMPool_Init_StatesMap(t *testing.T) {
	if len(testPoolProvider.states) == 0 {
		t.Errorf("expected initialized states map")
	}
}

func TestFSMPool_EntryPointMachine(t *testing.T) {
	m, err := testPoolProvider.EntryPointMachine()

	if err != nil || m.Name() != fsm1Name {
		t.Errorf("expected entry point machine")
	}
}

func TestFSMPool_MachineByState(t *testing.T) {
	fsm1States := []fsm.State{
		fsm1StateInit,
		fsm1StateStage1,
		fsm1StateStage2,
	}

	for _, state := range fsm1States {
		machine, err := testPoolProvider.MachineByState(state)
		if err != nil || machine.Name() != fsm1Name {
			t.Errorf("expected machine fsm1 for state \"%s\"", state)
			continue
		}
	}

	fsm2States := []fsm.State{
		fsm2StateInit,
		fsm2StateStage1,
		fsm2StateStage2,
	}

	for _, state := range fsm2States {
		machine, err := testPoolProvider.MachineByState(state)
		if err != nil || machine.Name() != fsm2Name {
			t.Errorf("expected machine fsm2 for state \"%s\"", state)
			continue
		}
	}
}

func TestFSMPool_MachineByEvent(t *testing.T) {
	fsm1Events := []fsm.Event{
		eventFSM1Init,
		eventFSM1Cancel,
		eventFSM1Process,
	}

	for _, event := range fsm1Events {
		machine, err := testPoolProvider.MachineByEvent(event)
		if err != nil || machine.Name() != fsm1Name {
			t.Errorf("expected machine fsm1 for event \"%s\"", event)
			continue
		}
	}

	fsm2Events := []fsm.Event{
		eventFSM2Init,
		eventFSM2Process,
	}

	for _, event := range fsm2Events {
		machine, err := testPoolProvider.MachineByEvent(event)
		if err != nil || machine.Name() != fsm2Name {
			t.Errorf("expected machine fsm2 for event \"%s\"", event)
			continue
		}
	}
}

func TestFSMPool_WorkFlow(t *testing.T) {
	machine, err := testPoolProvider.MachineByState(fsm1StateInit)

	if err != nil || machine.Name() != fsm1Name {
		t.Fatalf("expected machine fsm1 for state \"%s\"", fsm1StateInit)
	}

	if machine.State() != fsm1StateInit {
		t.Fatalf("expected machine  state \"%s\", got \"%s\"", fsm1StateInit, machine.State())
	}

	resp, err := machine.Do(eventFSM1Init)

	if err != nil {
		t.Fatalf("expected response without error, got \"%s\"", err)
	}

	if resp.State != fsm1StateStage2 {
		t.Fatalf("expected machine  state \"%s\", got \"%s\"", fsm1StateStage2, resp.State)
	}

	resp, err = machine.Do(eventFSM1Process, testVal2)

	data, ok := resp.Data.(int)

	if !ok {
		t.Fatalf("expected response data int, got \"%s\"", resp.Data)
	}

	if data != (testVal1 - testVal2) {
		t.Fatalf("expected response data value, got \"%d\"", data)
	}
}
