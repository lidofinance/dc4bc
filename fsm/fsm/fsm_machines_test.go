package fsm

import (
	"log"
	"testing"
)

var testingFSM *FSM

func init() {
	testingFSM = MustNewFSM(
		FSM1Name,
		FSM1StateInit,
		testingEvents,
		testingCallbacks,
	)
}

func compareRecoverStr(t *testing.T, r interface{}, assertion string) {
	if r == nil {
		return
	}
	msg, ok := r.(string)
	if !ok {
		t.Error("not asserted recover:", r)
	}
	if msg != assertion {
		t.Error("not asserted recover:", msg)
	}
}

func compareArrays(src, dst []string) bool {
	if len(src) != len(dst) {
		return false
	}
	// create a map of string -> int
	diff := make(map[string]int, len(src))
	for _, _x := range src {
		// 0 value for int is 0, so just increment a counter for the string
		diff[_x]++
	}
	for _, _y := range dst {
		// If the string _y is not in diff bail out early
		if _, ok := diff[_y]; !ok {
			return false
		}
		diff[_y] -= 1
		if diff[_y] == 0 {
			delete(diff, _y)
		}
	}
	if len(diff) == 0 {
		return true
	}
	return false
}

func TestMustNewFSM_Empty_Name_Panic(t *testing.T) {
	defer func() {
		compareRecoverStr(t, recover(), "machine name cannot be empty")
	}()
	testingFSM = MustNewFSM(
		"",
		"init_state",
		[]Event{},
		nil,
	)

	t.Errorf("did not panic on empty machine name")
}

func TestMustNewFSM_Empty_Initial_State_Panic(t *testing.T) {
	defer func() {
		compareRecoverStr(t, recover(), "initial state state cannot be empty")
	}()

	testingFSM = MustNewFSM(
		"fsm",
		"",
		[]Event{},
		nil,
	)

	t.Errorf("did not panic on empty initial")
}

func TestMustNewFSM_Empty_Events_Panic(t *testing.T) {
	defer func() {
		compareRecoverStr(t, recover(), "cannot init fsm with empty events")
	}()

	testingFSM = MustNewFSM(
		"fsm",
		"init_state",
		[]Event{},
		nil,
	)

	t.Errorf("did not panic on empty events list")
}

func TestMustNewFSM_Event_Empty_Name_Panic(t *testing.T) {
	defer func() {
		compareRecoverStr(t, recover(), "cannot init empty event")
	}()

	testingFSM = MustNewFSM(
		"fsm",
		"init_state",
		[]Event{
			{Name: "", SrcState: []string{"init_state"}, DstState: StateGlobalDone},
		},
		nil,
	)

	t.Errorf("did not panic on empty event name")
}

func TestMustNewFSM_Event_Empty_Source_Panic(t *testing.T) {
	defer func() {
		compareRecoverStr(t, recover(), "event must have minimum one source available state")
	}()

	testingFSM = MustNewFSM(
		"fsm",
		"init_state",
		[]Event{
			{Name: "event", SrcState: []string{}, DstState: StateGlobalDone},
		},
		nil,
	)

	t.Errorf("did not panic on empty event sources")
}

func TestMustNewFSM_States_Min_Panic(t *testing.T) {
	defer func() {
		compareRecoverStr(t, recover(), "machine must contain at least two states")
	}()

	testingFSM = MustNewFSM(
		"fsm",
		"init_state",
		[]Event{
			{Name: "event", SrcState: []string{"init_state"}, DstState: StateGlobalDone},
		},
		nil,
	)

	t.Errorf("did not panic on less than two states")
}

func TestMustNewFSM_State_Entry_Conflict_Panic(t *testing.T) {
	defer func() {
		compareRecoverStr(t, recover(), "machine entry event already exist")
	}()

	testingFSM = MustNewFSM(
		"fsm",
		"init_state",
		[]Event{
			{Name: "event1", SrcState: []string{"init_state"}, DstState: "state"},
			{Name: "event2", SrcState: []string{"init_state"}, DstState: "state"},
		},
		nil,
	)

	t.Errorf("did not panic on initialize with conflict in entry state")
}

func TestMustNewFSM_State_Final_Not_Found_Panic(t *testing.T) {
	defer func() {
		compareRecoverStr(t, recover(), "cannot initialize machine without final states")
	}()

	testingFSM = MustNewFSM(
		"fsm",
		"init_state",
		[]Event{
			{Name: "event1", SrcState: []string{"init_state"}, DstState: "state2"},
			{Name: "event2", SrcState: []string{"state2"}, DstState: "init_state"},
		},
		nil,
	)

	t.Errorf("did not panic on initialize without final state")
}

func TestFSM_Name(t *testing.T) {
	if testingFSM.Name() != FSM1Name {
		t.Errorf("expected machine name \"%s\"", FSM1Name)
	}
}

func TestFSM_EntryEvent(t *testing.T) {
	if testingFSM.InitialState() != FSM1StateInit {
		t.Errorf("expected initial state \"%s\"", FSM1StateInit)
	}
}

func TestFSM_EventsList(t *testing.T) {
	eventsList := []string{
		EventFSM1Init,
		EventFSM1Cancel,
		EventFSM1Process,
	}

	if !compareArrays(testingFSM.EventsList(), eventsList) {
		t.Error("expected public events", eventsList)
	}

}

func TestFSM_StatesList(t *testing.T) {
	log.Println(testingFSM.StatesSourcesList())
	statesList := []string{
		FSM1StateInit,
		FSM1StateStage1,
		FSM1StateStage2,
	}

	if !compareArrays(testingFSM.StatesSourcesList(), statesList) {
		t.Error("expected states", statesList)
	}
}
