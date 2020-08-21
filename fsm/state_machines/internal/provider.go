package internal

import (
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"github.com/depools/dc4bc/fsm/fsm"
	"github.com/depools/dc4bc/fsm/fsm_pool"
)

type DumpedMachineProvider interface {
	fsm_pool.MachineProvider
	WithSetup(state fsm.State, payload *DumpedMachineStatePayload) DumpedMachineProvider
}

// DKG and other stages quorums are separated,
// because unnecessary data may be unset
type DumpedMachineStatePayload struct {
	DkgId                    string
	SignatureProposalPayload *SignatureConfirmation
	DKGProposalPayload       *DKGConfirmation
	SigningProposalPayload   *SigningConfirmation
	PubKeys                  map[string]ed25519.PublicKey
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
		participant, _ = p.SignatureProposalPayload.Quorum[id]
	}
	return
}

func (p *DumpedMachineStatePayload) SigQuorumUpdate(id int, participant *SignatureProposalParticipant) {
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

// Signing quorum

func (p *DumpedMachineStatePayload) SigningQuorumCount() int {
	var count int
	if p.SigningProposalPayload.Quorum != nil {
		count = len(p.SigningProposalPayload.Quorum)
	}
	return count
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
		participant, _ = p.SigningProposalPayload.Quorum[id]
	}
	return
}

func (p *DumpedMachineStatePayload) SigningQuorumUpdate(id int, participant *SigningProposalParticipant) {
	if p.SigningProposalPayload.Quorum != nil {
		p.SigningProposalPayload.Quorum[id] = participant
	}
	return
}

func (p *DumpedMachineStatePayload) SetAddrHexPubKey(addr string, pubKey ed25519.PublicKey) {
	if p.PubKeys == nil {
		p.PubKeys = make(map[string]ed25519.PublicKey)
	}
	hexAddr := hex.EncodeToString([]byte(addr))
	p.PubKeys[hexAddr] = pubKey
	return
}

func (p *DumpedMachineStatePayload) GetPubKeyByAddr(addr string) (ed25519.PublicKey, error) {
	if p.PubKeys == nil {
		return nil, errors.New("{PubKeys} not initialized")
	}
	if addr == "" {
		return nil, errors.New("{addr} cannot be empty")
	}
	pubKey, ok := p.PubKeys[addr]
	if !ok {
		return nil, errors.New("cannot find public key by {addr}")
	}

	return pubKey, nil
}
