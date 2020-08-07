package internal

type MachineStatePayload struct {
	ConfirmationProposalPayload ConfirmationProposalPrivateQuorum
	DKGProposalPayload          DKGProposalPrivateQuorum
}

// Using combine response for modify data with chain
// User value or pointer? How about memory state?
type MachineCombinedResponse struct {
	Response interface{}
	Payload  *MachineStatePayload
}
