package state_machines

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	sif "github.com/depools/dc4bc/fsm/state_machines/signing_proposal_fsm"
	"reflect"
	"testing"
	"time"

	"github.com/depools/dc4bc/fsm/fsm"
	dpf "github.com/depools/dc4bc/fsm/state_machines/dkg_proposal_fsm"
	spf "github.com/depools/dc4bc/fsm/state_machines/signature_proposal_fsm"
	"github.com/depools/dc4bc/fsm/types/requests"
	"github.com/depools/dc4bc/fsm/types/responses"
)

const (
	addrMockLen = 32
	keysMockLen = 128
)

type testParticipantsPayload struct {
	Addr          string
	HotPrivKey    ed25519.PrivateKey
	HotPubKey     ed25519.PublicKey
	DkgPubKey     []byte
	DkgCommit     []byte
	DkgDeal       []byte
	DkgResponse   []byte
	DkgPartialKey []byte
}

var (
	tm = time.Now()

	dkgId = "1b7a6382afe0fbe2ff127a5779f5e9b042e685cabefeadcf4ef27c6089a56bfb"

	// map {addr} -> {participant}
	testAddrMapParticipants = map[string]*testParticipantsPayload{}
	// map {dkg_queue_id} -> {participant}
	testIdMapParticipants = map[int]*testParticipantsPayload{}

	testParticipantsListRequest = requests.SignatureProposalParticipantsListRequest{
		Participants: []*requests.SignatureProposalParticipantsEntry{},
		CreatedAt:    tm,
	}

	testFSMDump []byte
)

func init() {
	for i := 0; i < 3; i++ {

		participant := &testParticipantsPayload{
			Addr:          base64.StdEncoding.EncodeToString(genDataMock(addrMockLen)),
			HotPrivKey:    genDataMock(keysMockLen),
			HotPubKey:     genDataMock(keysMockLen),
			DkgPubKey:     genDataMock(keysMockLen),
			DkgCommit:     genDataMock(keysMockLen),
			DkgDeal:       genDataMock(keysMockLen),
			DkgResponse:   genDataMock(keysMockLen),
			DkgPartialKey: genDataMock(keysMockLen),
		}
		testAddrMapParticipants[participant.Addr] = participant
	}
}

func TestCreate_Positive(t *testing.T) {
	testFSMInstance, err := Create(dkgId)
	if err != nil {
		t.Fatalf("expected nil error, got {%s}", err)
	}

	if testFSMInstance == nil {
		t.Fatalf("expected {*FSMInstance}")
	}
}

func genDataMock(len int) []byte {
	data := make([]byte, len)
	rand.Read(data)
	return data
}

func compareErrNil(t *testing.T, got error) {
	if got != nil {
		t.Fatalf("expected nil error, got {%s}", got)
	}
}

func compareFSMInstanceNotNil(t *testing.T, got *FSMInstance) {
	if got == nil {
		t.Fatalf("expected {*FSMInstance}")
	}
}

func compareDumpNotZero(t *testing.T, got []byte) {
	if len(got) == 0 {
		t.Fatalf("expected non zero dump, when executed without error")
	}
}

func compareFSMResponseNotNil(t *testing.T, got *fsm.Response) {
	if got == nil {
		t.Fatalf("expected {*fsm.FSMResponse} got nil")
	}
}

func compareState(t *testing.T, expected fsm.State, got fsm.State) {
	if got != expected {
		t.Fatalf("expected state {%s} got {%s}", expected, got)
	}
}

// Test Workflow
func Test_SignatureProposal_Init(t *testing.T) {
	testFSMInstance, err := Create(dkgId)

	compareErrNil(t, err)

	compareFSMInstanceNotNil(t, testFSMInstance)

	transactionId := testFSMInstance.Id()

	if transactionId == "" {
		t.Fatalf("expected {transactionId for dump}")
	}

	if testFSMInstance.machine.Name() != spf.FsmName {
		t.Fatalf("expected machine name {%s}", spf.FsmName)
	}
	compareState(t, spf.StateParticipantsConfirmationsInit, testFSMInstance.machine.State())

	testFSMDump, err = testFSMInstance.Dump()

	compareErrNil(t, err)
}

// EventInitProposal
func Test_SignatureProposal_EventInitProposal(t *testing.T) {
	var fsmResponse *fsm.Response

	testFSMInstance, err := FromDump(testFSMDump)

	compareErrNil(t, err)

	compareFSMInstanceNotNil(t, testFSMInstance)

	// Make request
	request := make([]*requests.SignatureProposalParticipantsEntry, 0)

	for _, participant := range testAddrMapParticipants {

		request = append(request, &requests.SignatureProposalParticipantsEntry{
			Addr:      participant.Addr,
			PubKey:    participant.HotPubKey,
			DkgPubKey: participant.DkgPubKey,
		})
	}
	testParticipantsListRequest.Participants = request
	testParticipantsListRequest.SigningThreshold = len(request)

	fsmResponse, testFSMDump, err = testFSMInstance.Do(spf.EventInitProposal, testParticipantsListRequest)

	compareErrNil(t, err)

	compareDumpNotZero(t, testFSMDump)

	compareFSMResponseNotNil(t, fsmResponse)

	compareState(t, spf.StateAwaitParticipantsConfirmations, fsmResponse.State)

	testParticipantsListResponse, ok := fsmResponse.Data.(responses.SignatureProposalParticipantInvitationsResponse)

	if !ok {
		t.Fatalf("expected response {SignatureProposalParticipantInvitationsResponse}")
	}

	if len(testParticipantsListResponse) != len(testParticipantsListRequest.Participants) {
		t.Fatalf("expected response len {%d}, got {%d}", len(testParticipantsListRequest.Participants), len(testParticipantsListResponse))
	}

	for _, participant := range testParticipantsListResponse {
		if _, ok := testIdMapParticipants[participant.ParticipantId]; ok {
			t.Fatalf("expected unique {ParticipantId}")
		}

		if participant.Addr == "" {
			t.Fatalf("expected not empty {Addr}")
		}

		participantEntry, ok := testAddrMapParticipants[participant.Addr]

		if !ok {
			t.Fatalf("expected exist {Addr}")
		}

		testIdMapParticipants[participant.ParticipantId] = participantEntry
	}

}

// EventConfirmSignatureProposal
func Test_SignatureProposal_EventConfirmSignatureProposal(t *testing.T) {
	var fsmResponse *fsm.Response

	participantsCount := len(testIdMapParticipants)

	participantCounter := participantsCount

	for participantId, _ := range testIdMapParticipants {
		participantCounter--
		testFSMInstance, err := FromDump(testFSMDump)

		compareErrNil(t, err)

		compareFSMInstanceNotNil(t, testFSMInstance)

		fsmResponse, testFSMDump, err = testFSMInstance.Do(spf.EventConfirmSignatureProposal, requests.SignatureProposalParticipantRequest{
			ParticipantId: participantId,
			CreatedAt:     time.Now(),
		})

		compareErrNil(t, err)

		compareDumpNotZero(t, testFSMDump)

		compareFSMResponseNotNil(t, fsmResponse)

		if participantCounter > 0 {
			compareState(t, spf.StateAwaitParticipantsConfirmations, fsmResponse.State)
		}

	}

	compareState(t, spf.StateSignatureProposalCollected, fsmResponse.State)
}

func Test_DkgProposal_Positive(t *testing.T) {
	var fsmResponse *fsm.Response

	testFSMInstance, err := FromDump(testFSMDump)

	compareErrNil(t, err)

	compareFSMInstanceNotNil(t, testFSMInstance)

	fsmResponse, testFSMDump, err = testFSMInstance.Do(dpf.EventDKGInitProcess, requests.DefaultRequest{
		CreatedAt: time.Now(),
	})

	compareErrNil(t, err)

	compareDumpNotZero(t, testFSMDump)

	compareFSMResponseNotNil(t, fsmResponse)

	response, ok := fsmResponse.Data.(responses.DKGProposalPubKeysParticipantResponse)

	if !ok {
		t.Fatalf("expected response {DKGProposalPubKeysParticipantResponse}")
	}

	if len(response) != len(testParticipantsListRequest.Participants) {
		t.Fatalf("expected response len {%d}, got {%d}", len(testParticipantsListRequest.Participants), len(response))
	}

	for _, responseEntry := range response {
		if _, ok := testIdMapParticipants[responseEntry.ParticipantId]; !ok {
			t.Fatalf("expected exist {ParticipantId}")
		}

		if len(responseEntry.DkgPubKey) == 0 {
			t.Fatalf("expected {DkgPubKey} non zero length")
		}

		if !reflect.DeepEqual(testIdMapParticipants[responseEntry.ParticipantId].DkgPubKey, responseEntry.DkgPubKey) {
			t.Fatalf("expected valid {DkgPubKey}")
		}
	}

	compareState(t, dpf.StateDkgCommitsAwaitConfirmations, fsmResponse.State)

	// Commits
}

func Test_DkgProposal_EventDKGCommitConfirmationReceived(t *testing.T) {
	var fsmResponse *fsm.Response

	pCounter := 0
	for participantId, participant := range testIdMapParticipants {
		pCounter--
		testFSMInstance, err := FromDump(testFSMDump)

		compareErrNil(t, err)

		compareFSMInstanceNotNil(t, testFSMInstance)

		fsmResponse, testFSMDump, err = testFSMInstance.Do(dpf.EventDKGCommitConfirmationReceived, requests.DKGProposalCommitConfirmationRequest{
			ParticipantId: participantId,
			Commit:        participant.DkgCommit,
			CreatedAt:     tm,
		})

		compareErrNil(t, err)

		compareDumpNotZero(t, testFSMDump)

		compareFSMResponseNotNil(t, fsmResponse)

	}

	compareState(t, dpf.StateDkgDealsAwaitConfirmations, fsmResponse.State)

	response, ok := fsmResponse.Data.(responses.DKGProposalCommitParticipantResponse)

	if !ok {
		t.Fatalf("expected response {DKGProposalCommitParticipantResponse}")
	}

	if len(response) != len(testParticipantsListRequest.Participants) {
		t.Fatalf("expected response len {%d}, got {%d}", len(testParticipantsListRequest.Participants), len(response))
	}

	for _, responseEntry := range response {
		if _, ok := testIdMapParticipants[responseEntry.ParticipantId]; !ok {
			t.Fatalf("expected exist {ParticipantId}")
		}

		if len(responseEntry.DkgCommit) == 0 {
			t.Fatalf("expected {DkgCommit} non zero length")
		}

		if !reflect.DeepEqual(testIdMapParticipants[responseEntry.ParticipantId].DkgCommit, responseEntry.DkgCommit) {
			t.Fatalf("expected valid {DkgCommit}")
		}
	}
}

// Deals
func Test_DkgProposal_EventDKGDealConfirmationReceived(t *testing.T) {
	var fsmResponse *fsm.Response

	pCounter := 0
	for participantId, participant := range testIdMapParticipants {
		pCounter--
		testFSMInstance, err := FromDump(testFSMDump)

		compareErrNil(t, err)

		compareFSMInstanceNotNil(t, testFSMInstance)

		fsmResponse, testFSMDump, err = testFSMInstance.Do(dpf.EventDKGDealConfirmationReceived, requests.DKGProposalDealConfirmationRequest{
			ParticipantId: participantId,
			Deal:          participant.DkgDeal,
			CreatedAt:     tm,
		})

		compareErrNil(t, err)

		compareDumpNotZero(t, testFSMDump)

		compareFSMResponseNotNil(t, fsmResponse)

	}

	compareState(t, dpf.StateDkgResponsesAwaitConfirmations, fsmResponse.State)

	response, ok := fsmResponse.Data.(responses.DKGProposalDealParticipantResponse)

	if !ok {
		t.Fatalf("expected response {DKGProposalDealParticipantResponse}")
	}

	if len(response) != len(testParticipantsListRequest.Participants) {
		t.Fatalf("expected response len {%d}, got {%d}", len(testParticipantsListRequest.Participants), len(response))
	}

	for _, responseEntry := range response {
		if _, ok := testIdMapParticipants[responseEntry.ParticipantId]; !ok {
			t.Fatalf("expected exist {ParticipantId}")
		}

		if len(responseEntry.DkgDeal) == 0 {
			t.Fatalf("expected {DkgDeal} non zero length")
		}

		if !reflect.DeepEqual(testIdMapParticipants[responseEntry.ParticipantId].DkgDeal, responseEntry.DkgDeal) {
			t.Fatalf("expected valid {DkgDeal}")
		}
	}
}

// Responses
func Test_DkgProposal_EventDKGResponseConfirmationReceived(t *testing.T) {
	var fsmResponse *fsm.Response

	pCounter := 0
	for participantId, participant := range testIdMapParticipants {
		pCounter--
		testFSMInstance, err := FromDump(testFSMDump)

		compareErrNil(t, err)

		compareFSMInstanceNotNil(t, testFSMInstance)

		fsmResponse, testFSMDump, err = testFSMInstance.Do(dpf.EventDKGResponseConfirmationReceived, requests.DKGProposalResponseConfirmationRequest{
			ParticipantId: participantId,
			Response:      participant.DkgResponse,
			CreatedAt:     tm,
		})

		compareErrNil(t, err)

		compareDumpNotZero(t, testFSMDump)

		compareFSMResponseNotNil(t, fsmResponse)

	}

	compareState(t, dpf.StateDkgMasterKeyAwaitConfirmations, fsmResponse.State)

	response, ok := fsmResponse.Data.(responses.DKGProposalResponseParticipantResponse)

	if !ok {
		t.Fatalf("expected response {DKGProposalResponseParticipantResponse}")
	}

	if len(response) != len(testParticipantsListRequest.Participants) {
		t.Fatalf("expected response len {%d}, got {%d}", len(testParticipantsListRequest.Participants), len(response))
	}

	for _, responseEntry := range response {
		if _, ok := testIdMapParticipants[responseEntry.ParticipantId]; !ok {
			t.Fatalf("expected exist {ParticipantId}")
		}

		if len(responseEntry.DkgResponse) == 0 {
			t.Fatalf("expected {DkgResponse} non zero length")
		}

		if !reflect.DeepEqual(testIdMapParticipants[responseEntry.ParticipantId].DkgResponse, responseEntry.DkgResponse) {
			t.Fatalf("expected valid {DkgResponse}")
		}
	}
}

// Master keys
func Test_DkgProposal_EventDKGMasterKeyConfirmationReceived(t *testing.T) {
	var fsmResponse *fsm.Response

	pCounter := 0
	for participantId, participant := range testIdMapParticipants {
		pCounter--
		testFSMInstance, err := FromDump(testFSMDump)

		compareErrNil(t, err)

		compareFSMInstanceNotNil(t, testFSMInstance)

		fsmResponse, testFSMDump, err = testFSMInstance.Do(dpf.EventDKGMasterKeyConfirmationReceived, requests.DKGProposalMasterKeyConfirmationRequest{
			ParticipantId: participantId,
			MasterKey:     participant.DkgPartialKey,
			CreatedAt:     tm,
		})

		compareErrNil(t, err)

		compareDumpNotZero(t, testFSMDump)

		compareFSMResponseNotNil(t, fsmResponse)

	}

	compareState(t, dpf.StateDkgMasterKeyCollected, fsmResponse.State)
	return
}

// Signing
func Test_SigningProposal_Positive(t *testing.T) {
	var fsmResponse *fsm.Response

	testFSMInstance, err := FromDump(testFSMDump)

	compareErrNil(t, err)

	compareFSMInstanceNotNil(t, testFSMInstance)

	fsmResponse, testFSMDump, err = testFSMInstance.Do(sif.EventSigningInit, requests.DefaultRequest{
		CreatedAt: time.Now(),
	})

	compareErrNil(t, err)

	compareDumpNotZero(t, testFSMDump)

	compareFSMResponseNotNil(t, fsmResponse)

	compareState(t, sif.StateSigningIdle, fsmResponse.State)

	// Start

	testFSMInstance, err = FromDump(testFSMDump)

	compareErrNil(t, err)

	compareFSMInstanceNotNil(t, testFSMInstance)

	fsmResponse, testFSMDump, err = testFSMInstance.Do(sif.EventSigningStart, requests.SigningProposalStartRequest{
		ParticipantId: 1,
		SrcPayload:    []byte("message to sign"),
		CreatedAt:     time.Now(),
	})

	compareErrNil(t, err)

	compareDumpNotZero(t, testFSMDump)

	compareFSMResponseNotNil(t, fsmResponse)

	compareState(t, sif.StateSigningAwaitConfirmations, fsmResponse.State)

	testSigningParticipantsListResponse, ok := fsmResponse.Data.(responses.SigningProposalParticipantInvitationsResponse)

	if !ok {
		t.Fatalf("expected response {SigningProposalParticipantInvitationsResponse}")
	}

	if len(testSigningParticipantsListResponse.Participants) != len(testParticipantsListRequest.Participants) {
		t.Fatalf("expected response len {%d}, got {%d}", len(testParticipantsListRequest.Participants), len(testSigningParticipantsListResponse.Participants))
	}

	if testSigningParticipantsListResponse.SigningId == "" {
		t.Fatalf("expected field {SigningId}")
	}

}

func Test_Parallel(t *testing.T) {
	var (
		id1 = "123"
		id2 = "456"
	)
	testFSMInstance1, err := Create(id1)
	compareErrNil(t, err)

	compareFSMInstanceNotNil(t, testFSMInstance1)

	compareState(t, spf.StateParticipantsConfirmationsInit, testFSMInstance1.machine.State())

	testFSMDump1, err := testFSMInstance1.Dump()

	compareErrNil(t, err)

	compareDumpNotZero(t, testFSMDump1)

	/// fsm2
	testFSMInstance2, err := Create(id2)
	compareErrNil(t, err)

	compareFSMInstanceNotNil(t, testFSMInstance2)

	compareState(t, spf.StateParticipantsConfirmationsInit, testFSMInstance2.machine.State())

	testFSMDump2, err := testFSMInstance2.Dump()

	compareErrNil(t, err)

	compareDumpNotZero(t, testFSMDump2)

	testFSMInstance1, err = FromDump(testFSMDump1)

	compareErrNil(t, err)

	testFSMInstance2, err = FromDump(testFSMDump2)

	compareErrNil(t, err)

	_, _, err = testFSMInstance1.Do(spf.EventInitProposal, testParticipantsListRequest)

	s1, err := testFSMInstance1.State()

	compareErrNil(t, err)

	s2, err := testFSMInstance2.State()

	if s1 == s2 {
		t.Fatalf("MATCH STATES {%s}", s1)
	}

}
