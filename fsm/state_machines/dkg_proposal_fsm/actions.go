package dkg_proposal_fsm

import (
	"github.com/depools/dc4bc/fsm/fsm"
	"log"
)

// Pub keys
func (s *DKGProposalFSM) actionDKGPubKeysSent(event fsm.Event, args ...interface{}) (response interface{}, err error) {
	log.Println("I'm actionDKGPubKeysSent")
	return
}

func (s *DKGProposalFSM) actionDKGPubKeyConfirmationReceived(event fsm.Event, args ...interface{}) (response interface{}, err error) {
	log.Println("I'm actionDKGPubKeyConfirmationReceived")
	return
}

func (s *DKGProposalFSM) actionDKGPubKeyConfirmationError(event fsm.Event, args ...interface{}) (response interface{}, err error) {
	log.Println("I'm actionDKGPubKeyConfirmationError")
	return
}

// Commits
func (s *DKGProposalFSM) actionDKGCommitsSent(event fsm.Event, args ...interface{}) (response interface{}, err error) {
	log.Println("I'm actionDKGCommitsSent")
	return
}

func (s *DKGProposalFSM) actionDKGCommitConfirmationReceived(event fsm.Event, args ...interface{}) (response interface{}, err error) {
	log.Println("I'm actionDKGCommitConfirmationReceived")
	return
}

func (s *DKGProposalFSM) actionDKGCommitConfirmationError(event fsm.Event, args ...interface{}) (response interface{}, err error) {
	log.Println("I'm actionDKGCommitConfirmationError")
	return
}

// Deals
func (s *DKGProposalFSM) actionDKGDealsSent(event fsm.Event, args ...interface{}) (response interface{}, err error) {
	log.Println("I'm actionDKGDealsSent")
	return
}

func (s *DKGProposalFSM) actionDKGDealConfirmationReceived(event fsm.Event, args ...interface{}) (response interface{}, err error) {
	log.Println("I'm actionDKGDealConfirmationReceived")
	return
}

func (s *DKGProposalFSM) actionDKGDealConfirmationError(event fsm.Event, args ...interface{}) (response interface{}, err error) {
	log.Println("I'm actionDKGDealConfirmationError")
	return
}
