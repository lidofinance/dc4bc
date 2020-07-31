package fsm

type testMachineFSM struct {
	*FSM
}

const (
	FSM1Name = "fsm1"
	// Init process from global idle state
	FSM1StateInit = StateGlobalIdle
	// Set up data
	FSM1StateStage1 = State("state_fsm1_stage1")
	// Process data
	FSM1StateStage2 = State("state_fsm1_stage2")
	// Cancelled with internal event
	FSM1StateCanceledByInternal = State("state_fsm1_canceled")
	// Cancelled with external event
	FSM1StateCanceled2 = State("state_fsm1_canceled2")
	// Out endpoint to switch
	FSM1StateOutToFSM2 = State("state_fsm1_out_to_fsm2")
	FSM1StateOutToFSM3 = State("state_fsm1_out_to_fsm3")

	// Events
	EventFSM1Init    = Event("event_fsm1_init")
	EventFSM1Cancel  = Event("event_fsm1_cancel")
	EventFSM1Process = Event("event_fsm1_process")

	// Internal events
	EventFSM1Internal         = Event("event_internal_fsm1")
	EventFSM1CancelByInternal = Event("event_internal_fsm1_cancel")
	EventFSM1InternalOut2     = Event("event_internal_fsm1_out")
)

var (
	testing1Events = []EventDesc{
		// Init
		{Name: EventFSM1Init, SrcState: []State{FSM2StateInit}, DstState: FSM1StateStage1},
		{Name: EventFSM1Internal, SrcState: []State{FSM1StateStage1}, DstState: FSM1StateStage2, IsInternal: true},

		// Cancellation events
		{Name: EventFSM1CancelByInternal, SrcState: []State{FSM1StateStage2}, DstState: FSM1StateCanceledByInternal, IsInternal: true},
		{Name: EventFSM1Cancel, SrcState: []State{FSM1StateStage2}, DstState: FSM1StateCanceled2},

		// Out
		{Name: EventFSM1Process, SrcState: []State{FSM1StateStage2}, DstState: FSM1StateOutToFSM2},
		{Name: EventFSM1InternalOut2, SrcState: []State{FSM1StateStage2}, DstState: FSM1StateOutToFSM3, IsInternal: true},
	}

	testing1Callbacks = Callbacks{
		EventFSM1Init:         actionSetUpData,
		EventFSM1InternalOut2: actionEmitOut2,
		EventFSM1Process:      actionProcessData,
	}
)

func actionSetUpData(event Event, args ...interface{}) (response interface{}, err error) {
	return
}

func actionProcessData(event Event, args ...interface{}) (response interface{}, err error) {
	return
}

func actionEmitOut2(event Event, args ...interface{}) (response interface{}, err error) {
	return
}
