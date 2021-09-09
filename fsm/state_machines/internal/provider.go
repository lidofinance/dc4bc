package internal

import (
	"crypto/ed25519"
	"errors"
	"github.com/lidofinance/dc4bc/fsm/fsm"
	"github.com/lidofinance/dc4bc/fsm/fsm_pool"
)

type DumpedMachineProvider interface {
	fsm_pool.MachineProvider
	WithSetup(state fsm.State, payload *DumpedMachineStatePayload) DumpedMachineProvider
}

// DKG and other stages quorums are separated,
// because unnecessary data may be unset
type DumpedMachineStatePayload struct {
	DkgId                    string
	Threshold                int
	SignatureProposalPayload *SignatureConfirmation
	DKGProposalPayload       *DKGConfirmation
	SigningProposalPayload   *SigningConfirmation
	PubKeys                  map[string]ed25519.PublicKey
	IDs                      map[string]int
}

// Signature quorum

func (p *DumpedMachineStatePayload) SigQuorumCount() int {
	var count int
	if p.SignatureProposalPayload.Quorum != nil {
		count = len(p.SignatureProposalPayload.Quorum)
	}
	return count
}

func (p *DumpedMachineStatePayload) SigQuorumExists(id int) bool {
	var exists bool
	if p.SignatureProposalPayload.Quorum != nil {
		_, exists = p.SignatureProposalPayload.Quorum[id]
	}
	return exists
}

func (p *DumpedMachineStatePayload) SigQuorumGet(id int) (participant *SignatureProposalParticipant) {
	if p.SignatureProposalPayload.Quorum != nil {
		participant = p.SignatureProposalPayload.Quorum[id]
	}
	return participant
}

func (p *DumpedMachineStatePayload) SigQuorumUpdate(id int, participant *SignatureProposalParticipant) {
	if p.SignatureProposalPayload.Quorum != nil {
		p.SignatureProposalPayload.Quorum[id] = participant
	}
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
		participant = p.DKGProposalPayload.Quorum[id]
	}
	return participant
}

func (p *DumpedMachineStatePayload) DKGQuorumUpdate(id int, participant *DKGProposalParticipant) {
	if p.DKGProposalPayload.Quorum != nil {
		p.DKGProposalPayload.Quorum[id] = participant
	}
}

// Signing quorum

func (p *DumpedMachineStatePayload) SigningQuorumCount() int {
	var count int
	if p.SigningProposalPayload.Quorum != nil {
		count = len(p.SigningProposalPayload.Quorum)
	}
	return count
}

func (p *DumpedMachineStatePayload) GetThreshold() int {
	return p.Threshold
}

func (p *DumpedMachineStatePayload) SigningQuorumExists(id int) bool {
	var exists bool
	if p.SigningProposalPayload.Quorum != nil {
		_, exists = p.SigningProposalPayload.Quorum[id]
	}
	return exists
}

func (p *DumpedMachineStatePayload) SigningQuorumGet(id int) (participant *SigningProposalParticipant) {
	if p.SigningProposalPayload.Quorum != nil {
		participant = p.SigningProposalPayload.Quorum[id]
	}
	return
}

func (p *DumpedMachineStatePayload) SigningQuorumUpdate(id int, participant *SigningProposalParticipant) {
	if p.SigningProposalPayload.Quorum != nil {
		p.SigningProposalPayload.Quorum[id] = participant
	}
}

func (p *DumpedMachineStatePayload) SetPubKeyUsername(username string, pubKey ed25519.PublicKey) {
	if p.PubKeys == nil {
		p.PubKeys = make(map[string]ed25519.PublicKey)
	}
	p.PubKeys[username] = pubKey
}

func (p *DumpedMachineStatePayload) SetIDUsername(username string, id int) {
	if p.IDs == nil {
		p.IDs = make(map[string]int)
	}
	p.IDs[username] = id
}

func (p *DumpedMachineStatePayload) GetPubKeyByUsername(username string) (ed25519.PublicKey, error) {
	if p.PubKeys == nil {
		return nil, errors.New("{PubKeys} not initialized")
	}
	if username == "" {
		return nil, errors.New("{username} cannot be empty")
	}
	pubKey, ok := p.PubKeys[username]
	if !ok {
		return nil, errors.New("cannot find public key by {username}")
	}

	return pubKey, nil
}

func (p *DumpedMachineStatePayload) GetIDByUsername(username string) (int, error) {
	if p.IDs == nil {
		return -1, errors.New("{IDs} not initialized")
	}
	if username == "" {
		return -1, errors.New("{username} cannot be empty")
	}
	id, ok := p.IDs[username]
	if !ok {
		return -1, errors.New("cannot find id by {username}")
	}
	return id, nil
}
