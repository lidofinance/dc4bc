package requests

// Event: "event_dkg_pub_keys_sent"
type DKGProposalPubKeysSendingRequest struct {
	PubKeys map[int][]byte
}

// Event: "event_dkg_commits_sent"
type DKGProposalCommitsSendingRequest struct {
	Commits map[int][]byte
}

// Event: "event_dkg_deals_sent"
type DKGProposalDealsSendingRequest struct {
	Deals map[int][]byte
}
