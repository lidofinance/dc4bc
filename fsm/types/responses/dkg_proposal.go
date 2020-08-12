package responses

type DKGProposalPubKeyParticipantResponse []*DKGProposalPubKeyParticipantEntry

type DKGProposalPubKeyParticipantEntry struct {
	ParticipantId int
	Title         string
	PubKey        []byte
}

type DKGProposalCommitParticipantResponse []*DKGProposalCommitParticipantEntry

type DKGProposalCommitParticipantEntry struct {
	ParticipantId int
	Title         string
	Commit        []byte
}

type DKGProposalDealParticipantResponse []*DKGProposalDealParticipantEntry

type DKGProposalDealParticipantEntry struct {
	ParticipantId int
	Title         string
	Deal          []byte
}

type DKGProposalResponsesParticipantResponse []*DKGProposalResponsesParticipantEntry

type DKGProposalResponsesParticipantEntry struct {
	ParticipantId int
	Title         string
	Responses     []byte
}
