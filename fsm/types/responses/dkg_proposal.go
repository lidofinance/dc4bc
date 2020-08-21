package responses

type DKGProposalPubKeysParticipantResponse []*DKGProposalPubKeysParticipantEntry

type DKGProposalPubKeysParticipantEntry struct {
	ParticipantId int
	Addr          string
	DkgPubKey     []byte
}

type DKGProposalCommitParticipantResponse []*DKGProposalCommitParticipantEntry

type DKGProposalCommitParticipantEntry struct {
	ParticipantId int
	Addr          string
	DkgCommit     []byte
}

type DKGProposalDealParticipantResponse []*DKGProposalDealParticipantEntry

type DKGProposalDealParticipantEntry struct {
	ParticipantId int
	Addr          string
	DkgDeal       []byte
}

type DKGProposalResponseParticipantResponse []*DKGProposalResponseParticipantEntry

type DKGProposalResponseParticipantEntry struct {
	ParticipantId int
	Addr          string
	DkgResponse   []byte
}

type DKGProposalResponsesParticipantResponse []*DKGProposalResponsesParticipantEntry

type DKGProposalResponsesParticipantEntry struct {
	ParticipantId int
	Title         string
	Responses     []byte
}
