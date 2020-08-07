package dkg_proposal_fsm

import (
	"github.com/depools/dc4bc/fsm/fsm"
	"log"
)

// Pub keys
func (m *DKGProposalFSM) actionDKGPubKeysSent(event fsm.Event, args ...interface{}) (response interface{}, err error) {
	log.Println("I'm actionDKGPubKeysSent")
	return
}

func (m *DKGProposalFSM) actionDKGPubKeyConfirmationReceived(event fsm.Event, args ...interface{}) (response interface{}, err error) {
	log.Println("I'm actionDKGPubKeyConfirmationReceived")
	return
}

func (m *DKGProposalFSM) actionDKGPubKeyConfirmationError(event fsm.Event, args ...interface{}) (response interface{}, err error) {
	log.Println("I'm actionDKGPubKeyConfirmationError")
	return
}

// Commits
func (m *DKGProposalFSM) actionDKGCommitsSent(event fsm.Event, args ...interface{}) (response interface{}, err error) {
	log.Println("I'm actionDKGCommitsSent")
	return
}

func (m *DKGProposalFSM) actionDKGCommitConfirmationReceived(event fsm.Event, args ...interface{}) (response interface{}, err error) {
	log.Println("I'm actionDKGCommitConfirmationReceived")
	return
}

func (m *DKGProposalFSM) actionDKGCommitConfirmationError(event fsm.Event, args ...interface{}) (response interface{}, err error) {
	log.Println("I'm actionDKGCommitConfirmationError")
	return
}

// Deals
func (m *DKGProposalFSM) actionDKGDealsSent(event fsm.Event, args ...interface{}) (response interface{}, err error) {
	log.Println("I'm actionDKGDealsSent")
	return
}

func (m *DKGProposalFSM) actionDKGDealConfirmationReceived(event fsm.Event, args ...interface{}) (response interface{}, err error) {
	log.Println("I'm actionDKGDealConfirmationReceived")
	return
}

func (m *DKGProposalFSM) actionDKGDealConfirmationError(event fsm.Event, args ...interface{}) (response interface{}, err error) {
	log.Println("I'm actionDKGDealConfirmationError")
	return
}
