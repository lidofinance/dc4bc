package fsm

const (
	FSM1Name = "fsm1"
	// Init process from global idle state
	FSM1StateInit = StateGlobalIdle
	// Set up data
	FSM1StateStage1 = "state_fsm1_stage1"
	// Process data
	FSM1StateStage2 = "state_fsm1_stage2"
	// Cancelled with internal event
	FSM1StateCanceledByInternal = "state_fsm1_canceled1"
	// Cancelled with external event
	FSM1StateCanceled2 = "state_fsm1_canceled2"
	// Out endpoint to switch
	FSM1StateOutToFSM2 = "state_fsm1_out_to_fsm2"
	FSM1StateOutToFSM3 = "state_fsm1_out_to_fsm3"

	// Events
	EventFSM1Init    = "event_fsm1_init"
	EventFSM1Cancel  = "event_fsm1_cancel"
	EventFSM1Process = "event_fsm1_process"

	// Internal events
	EventFSM1Internal         = "event_internal_fsm1"
	EventFSM1CancelByInternal = "event_internal_fsm1_cancel"
	EventFSM1InternalOut2     = "event_internal_fsm1_out"
)

var (
	testingEvents = []Event{
		// Init
		{Name: EventFSM1Init, SrcState: []string{FSM1StateInit}, DstState: FSM1StateStage1},
		{Name: EventFSM1Internal, SrcState: []string{FSM1StateStage1}, DstState: FSM1StateStage2, IsInternal: true},

		// Cancellation events
		{Name: EventFSM1CancelByInternal, SrcState: []string{FSM1StateStage2}, DstState: FSM1StateCanceledByInternal, IsInternal: true},
		{Name: EventFSM1Cancel, SrcState: []string{FSM1StateStage2}, DstState: FSM1StateCanceled2},

		// Out
		{Name: EventFSM1Process, SrcState: []string{FSM1StateStage2}, DstState: FSM1StateOutToFSM2},
		{Name: EventFSM1InternalOut2, SrcState: []string{FSM1StateStage2}, DstState: FSM1StateOutToFSM3, IsInternal: true},
	}

	testingCallbacks = Callbacks{
		EventFSM1Init:         actionSetUpData,
		EventFSM1InternalOut2: actionEmitOut2,
		EventFSM1Process:      actionProcessData,
	}
)

type testMachineFSM struct {
	*FSM
}

/*func new() fsm_pool.IStateMachine {
	machine := &testMachineFSM{}
	machine.FSM = MustNewFSM(
		FSM1Name,
		FSM1StateInit,
		testingEvents,
		testingCallbacks,
	)
	return machine
}*/

func actionSetUpData(event string, args ...interface{}) (response interface{}, err error) {
	return
}

func actionProcessData(event string, args ...interface{}) (response interface{}, err error) {
	return
}

func actionEmitOut2(event string, args ...interface{}) (response interface{}, err error) {
	return
}
