package internal

type MachineStatePayload struct {
	ProposalPayload ProposalConfirmationPrivateQuorum
	SigningPayload  map[string]interface{}
}

// Using combine response for modify data with chain
// User value or pointer? How about memory state?
type MachineCombinedResponse struct {
	Response interface{}
	Payload  *MachineStatePayload
}
