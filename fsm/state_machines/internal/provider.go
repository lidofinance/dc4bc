package internal

import "github.com/depools/dc4bc/fsm/fsm_pool"

type DumpedMachineProvider interface {
	fsm_pool.MachineProvider
	SetUpPayload(payload *DumpedMachineStatePayload)
}

// DKG and other stages quorums are separated,
// because unnecessary data may be unset
type DumpedMachineStatePayload struct {
	TransactionId            string
	SignatureProposalPayload *SignatureConfirmation
	DKGProposalPayload       *DKGConfirmation
}

// Signature quorum

func (p *DumpedMachineStatePayload) SigQuorumCount() int {
	var count int
	if p.SignatureProposalPayload.Quorum != nil {
		count = len(p.SignatureProposalPayload.Quorum)
	}
	return count
}

func (p *DumpedMachineStatePayload) SigQuorumExists(id string) bool {
	var exists bool
	if p.SignatureProposalPayload.Quorum != nil {
		_, exists = p.SignatureProposalPayload.Quorum[id]
	}
	return exists
}

func (p *DumpedMachineStatePayload) SigQuorumGet(id string) (participant *SignatureProposalParticipant) {
	if p.SignatureProposalPayload.Quorum != nil {
		participant, _ = p.SignatureProposalPayload.Quorum[id]
	}
	return
}

func (p *DumpedMachineStatePayload) SigQuorumUpdate(id string, participant *SignatureProposalParticipant) {
	if p.SignatureProposalPayload.Quorum != nil {
		p.SignatureProposalPayload.Quorum[id] = participant
	}
	return
}

// DKG quorum

func (p *DumpedMachineStatePayload) DKGQuorumCount() int {
	var count int
	if p.DKGProposalPayload.Quorum != nil {
		count = len(p.DKGProposalPayload.Quorum)
	}
	return count
}

func (p *DumpedMachineStatePayload) DKGQuorumExists(id int) bool {
	var exists bool
	if p.DKGProposalPayload.Quorum != nil {
		_, exists = p.DKGProposalPayload.Quorum[id]
	}
	return exists
}

func (p *DumpedMachineStatePayload) DKGQuorumGet(id int) (participant *DKGProposalParticipant) {
	if p.DKGProposalPayload.Quorum != nil {
		participant, _ = p.DKGProposalPayload.Quorum[id]
	}
	return
}

func (p *DumpedMachineStatePayload) DKGQuorumUpdate(id int, participant *DKGProposalParticipant) {
	if p.DKGProposalPayload.Quorum != nil {
		p.DKGProposalPayload.Quorum[id] = participant
	}
	return
}
