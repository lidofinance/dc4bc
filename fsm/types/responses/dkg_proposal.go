package responses

type DKGProposalPubKeysParticipantResponse []*DKGProposalPubKeysParticipantEntry

type DKGProposalPubKeysParticipantEntry struct {
	ParticipantId int
	Username      string
	DkgPubKey     []byte
}

type DKGProposalCommitParticipantResponse []*DKGProposalCommitParticipantEntry

type DKGProposalCommitParticipantEntry struct {
	ParticipantId int
	Username      string
	DkgCommit     []byte
}

type DKGProposalDealParticipantResponse []*DKGProposalDealParticipantEntry

type DKGProposalDealParticipantEntry struct {
	ParticipantId int
	Username      string
	DkgDeal       []byte
}

type DKGProposalResponseParticipantResponse []*DKGProposalResponseParticipantEntry

type DKGProposalResponseParticipantEntry struct {
	ParticipantId int
	Username      string
	DkgResponse   []byte
}
