package main

import (
	"fmt"
	"github.com/looplab/fsm"
)

func main() {
	signatureProposalFSM := fsm.NewFSM(
		"idle",
		fsm.Events{
			{Name: "proposal_spotted", Src: []string{"idle"}, Dst: "validate_proposal"},
			{Name: "proposal_valid", Src: []string{"validate_proposal"}, Dst: "proposed"},
			{Name: "proposal_invalid", Src: []string{"validate_proposal"}, Dst: "idle"},
			{Name: "recieve_yay", Src: []string{"proposed"}, Dst: "process_yay"},
			{Name: "receive_nay", Src: []string{"proposed"}, Dst: "process_nay"},
			{Name: "send_nay", Src: []string{"proposed"}, Dst: "proposed"},
			{Name: "send_yay", Src: []string{"proposed"}, Dst: "proposed"},
			{Name: "enough_yays", Src: []string{"process_yay"}, Dst: "signing"},
			{Name: "enough_nays", Src: []string{"process_nay"}, Dst: "abort"},
			{Name: "not_enough_yays", Src: []string{"process_yay"}, Dst: "proposed"},
			{Name: "not_enough_nays", Src: []string{"process_nay"}, Dst: "proposed"},
		},
		fsm.Callbacks{},
	)
	fmt.Print(fsm.Visualize(signatureProposalFSM))

	signatureConstructFSM := fsm.NewFSM(
		"idle",
		fsm.Events{
			{Name: "request_airgapped_sig", Src: []string{"signing"}, Dst: "signing"},
			{Name: "transmit_airgapped_sig", Src: []string{"signing"}, Dst: "signing"},
			{Name: "receive_sig", Src: []string{"signing"}, Dst: "process_sig"},
			{Name: "enough_signature_shares", Src: []string{"process_sig"}, Dst: "reconstruct_signature"},
			{Name: "not_enough_signature_shares", Src: []string{"process_sig"}, Dst: "signing"},
			{Name: "signature_reconstucted", Src: []string{"reconstruct_signature"}, Dst: "publish_signature"},
			{Name: "signature_published", Src: []string{"publish_signature"}, Dst: "fin"},
			{Name: "failed_to_reconstuct_signature", Src: []string{"reconstruct_signature"}, Dst: "signing"},
		},
		fsm.Callbacks{},
	)
	fmt.Print(fsm.Visualize(signatureConstructFSM))

	DkgProposeFSM := fsm.NewFSM(
		"idle",
		fsm.Events{
			{Name: "proposal_spotted", Src: []string{"idle"}, Dst: "validate_proposal"},
			{Name: "proposal_valid", Src: []string{"validate_proposal"}, Dst: "proposed"},
			{Name: "proposal_invalid", Src: []string{"validate_proposal"}, Dst: "idle"},
			{Name: "recieve_yay", Src: []string{"proposed"}, Dst: "process_yay"},
			{Name: "receive_nay", Src: []string{"proposed"}, Dst: "abort"},
			{Name: "send_nay", Src: []string{"proposed"}, Dst: "proposed"},
			{Name: "send_yay", Src: []string{"proposed"}, Dst: "proposed"},
			{Name: "not_enough_yays", Src: []string{"process_yay"}, Dst: "proposed"},
			{Name: "all_yays", Src: []string{"process_yay"}, Dst: "dkg_commitments"},
			{Name: "timeout", Src: []string{"proposed"}, Dst: "abort"},
		},
		fsm.Callbacks{},
	)
	fmt.Print(fsm.Visualize(DkgProposeFSM))

	DkgCommitFSM := fsm.NewFSM(
		"dkg_commitments",
		fsm.Events{
			{Name: "request_airgapped_commitment", Src: []string{"dkg_commitments"}, Dst: "dkg_commitments"},
			{Name: "transmit_airgapped_commitment", Src: []string{"dkg_commitments"}, Dst: "dkg_commitments"},
			{Name: "recieve_commitment", Src: []string{"dkg_commitments"}, Dst: "process_commitment"},
			{Name: "invalid_commitment", Src: []string{"process_commitment"}, Dst: "abort"},
			{Name: "all_commitments", Src: []string{"process_commitment"}, Dst: "dkg_deals"},
			{Name: "not_enough_commitments", Src: []string{"process_commitment"}, Dst: "dkg_commitments"},
			{Name: "timeout", Src: []string{"dkg_commitments"}, Dst: "abort"},
		},
		fsm.Callbacks{},
	)
	fmt.Print(fsm.Visualize(DkgCommitFSM))

	DkgDealsFSM := fsm.NewFSM(
		"dkg_deals",
		fsm.Events{
			{Name: "pass_commitements_and_request_airgapped_deals", Src: []string{"dkg_deals"}, Dst: "dkg_deals"},
			{Name: "transmit_airgapped_deals", Src: []string{"dkg_deals"}, Dst: "dkg_deals"},
			{Name: "transmit_airgapped_error", Src: []string{"dkg_deals"}, Dst: "abort"},
			{Name: "recieve_deal", Src: []string{"dkg_deals"}, Dst: "process_deal"},
			{Name: "not_my_deal", Src: []string{"process_deal"}, Dst: "dkg_deals"},
			{Name: "invalid_deal", Src: []string{"process_deal"}, Dst: "abort"},
			{Name: "enough_deals", Src: []string{"process_deal"}, Dst: "dkg_construct_tss"},
			{Name: "not_enough_deals", Src: []string{"process_deal"}, Dst: "dkg_deals"},
			{Name: "pass_deals_and_request_airgapped_public_key", Src: []string{"dkg_construct_tss"}, Dst: "dkg_construct_tss"},
			{Name: "transmit_airgapped_public_key", Src: []string{"dkg_construct_tss"}, Dst: "fin"},
			{Name: "transmit_airgapped_error", Src: []string{"dkg_construct_tss"}, Dst: "abort"},
			{Name: "timeout", Src: []string{"dkg_deals"}, Dst: "abort"},
		},
		fsm.Callbacks{},
	)
	fmt.Print(fsm.Visualize(DkgDealsFSM))

}
