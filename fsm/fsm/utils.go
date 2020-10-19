package fsm

import (
	"bytes"
	"fmt"
)

func Visualize(fsm *FSM) string {
	var buf bytes.Buffer

	states := make(map[string]int)

	buf.WriteString(fmt.Sprintf(`digraph fsm {`))
	buf.WriteString("\n")

	// make sure the initial state is at top
	for k, v := range fsm.transitions {
		if k.source == fsm.currentState {
			states[string(k.source)]++
			states[string(v.dstState)]++
			buf.WriteString(fmt.Sprintf(`    "%s" -> "%s" [ label = "%s" ];`, k.source, v.dstState, k.event))
			buf.WriteString("\n")
		}
	}

	for k, v := range fsm.transitions {
		if k.source != fsm.currentState {
			states[string(k.source)]++
			states[string(v.dstState)]++
			buf.WriteString(fmt.Sprintf(`    "%s" -> "%s" [ label = "%s" ];`, k.source, v.dstState, k.event))
			buf.WriteString("\n")
		}
	}

	buf.WriteString("\n")

	for k := range states {
		buf.WriteString(fmt.Sprintf(`    "%s";`, k))
		buf.WriteString("\n")
	}
	buf.WriteString(fmt.Sprintln("}"))

	return buf.String()
}
