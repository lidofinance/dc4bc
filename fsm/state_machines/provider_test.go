package state_machines

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	sif "github.com/lidofinance/dc4bc/fsm/state_machines/signing_proposal_fsm"

	"github.com/lidofinance/dc4bc/fsm/fsm"
	dpf "github.com/lidofinance/dc4bc/fsm/state_machines/dkg_proposal_fsm"
	spf "github.com/lidofinance/dc4bc/fsm/state_machines/signature_proposal_fsm"
	"github.com/lidofinance/dc4bc/fsm/types/requests"
	"github.com/lidofinance/dc4bc/fsm/types/responses"
)

const (
	usernameMockLen = 32
	keysMockLen     = 128
)

type testParticipantsPayload struct {
	Username      string
	HotPrivKey    ed25519.PrivateKey
	HotPubKey     ed25519.PublicKey
	DkgPubKey     []byte
	DkgCommit     []byte
	DkgDeal       []byte
	DkgResponse   []byte
	DkgPartialKey []byte
}

const (
	participantsNumber = 10
	threshold          = 3
)

var (
	tm = time.Now()

	dkgId = "1b7a6382afe0fbe2ff127a5779f5e9b042e685cabefeadcf4ef27c6089a56bfb"

	// map {username} -> {participant}
	testUsernameMapParticipants = map[string]*testParticipantsPayload{}
	// map {dkg_queue_id} -> {participant}
	testIdMapParticipants = map[int]*testParticipantsPayload{}

	testParticipantsListRequest = requests.SignatureProposalParticipantsListRequest{
		Participants: []*requests.SignatureProposalParticipantsEntry{},
		CreatedAt:    tm,
	}

	testSigningId        string
	testSigningInitiator int
	testSigningPayload   = []byte("message to sign")

	testFSMDump = map[fsm.State][]byte{}
)

func init() {
	for i := 0; i < participantsNumber; i++ {

		participant := &testParticipantsPayload{
			Username:      base64.StdEncoding.EncodeToString(genDataMock(usernameMockLen)),
			HotPrivKey:    genDataMock(keysMockLen),
			HotPubKey:     genDataMock(keysMockLen),
			DkgPubKey:     genDataMock(keysMockLen),
			DkgCommit:     genDataMock(keysMockLen),
			DkgDeal:       genDataMock(keysMockLen),
			DkgResponse:   genDataMock(keysMockLen),
			DkgPartialKey: genDataMock(keysMockLen),
		}
		testUsernameMapParticipants[participant.Username] = participant
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

	testFSMDump[spf.StateParticipantsConfirmationsInit], err = testFSMInstance.Dump()
	require.NoError(t, err)

	compareDumpNotZero(t, testFSMDump[spf.StateParticipantsConfirmationsInit])
}

// EventInitProposal
func Test_SignatureProposal_EventInitProposal_Positive(t *testing.T) {
	var fsmResponse *fsm.Response

	testFSMInstance, err := FromDump(testFSMDump[spf.StateParticipantsConfirmationsInit])

	compareErrNil(t, err)

	compareFSMInstanceNotNil(t, testFSMInstance)

	compareState(t, spf.StateParticipantsConfirmationsInit, testFSMInstance.machine.State())

	// Make request
	request := make([]*requests.SignatureProposalParticipantsEntry, 0)

	for _, participant := range testUsernameMapParticipants {

		request = append(request, &requests.SignatureProposalParticipantsEntry{
			Username:  participant.Username,
			PubKey:    participant.HotPubKey,
			DkgPubKey: participant.DkgPubKey,
		})
	}
	testParticipantsListRequest.Participants = request
	testParticipantsListRequest.SigningThreshold = threshold

	fsmResponse, testFSMDump[spf.StateAwaitParticipantsConfirmations], err = testFSMInstance.Do(spf.EventInitProposal, testParticipantsListRequest)

	compareErrNil(t, err)

	compareFSMResponseNotNil(t, fsmResponse)

	compareState(t, spf.StateAwaitParticipantsConfirmations, fsmResponse.State)

	response, ok := fsmResponse.Data.(responses.SignatureProposalParticipantInvitationsResponse)

	if !ok {
		t.Fatalf("expected response {SignatureProposalParticipantInvitationsResponse}")
	}

	if len(response) != len(testParticipantsListRequest.Participants) {
		t.Fatalf("expected response len {%d}, got {%d}", len(testParticipantsListRequest.Participants), len(response))
	}

	for _, participant := range response {
		if _, ok := testIdMapParticipants[participant.ParticipantId]; ok {
			t.Fatalf("expected unique {ParticipantId}")
		}

		if participant.Username == "" {
			t.Fatalf("expected not empty {Username}")
		}

		participantEntry, ok := testUsernameMapParticipants[participant.Username]

		if !ok {
			t.Fatalf("expected exist {Username}")
		}

		testIdMapParticipants[participant.ParticipantId] = participantEntry
	}

	compareDumpNotZero(t, testFSMDump[spf.StateAwaitParticipantsConfirmations])
}

// EventConfirmSignatureProposal
func Test_SignatureProposal_EventConfirmSignatureProposal_Positive(t *testing.T) {
	var (
		fsmResponse      *fsm.Response
		testFSMDumpLocal []byte
	)

	participantsCount := len(testIdMapParticipants)

	participantCounter := participantsCount

	// testFSMDumpLocal = make([]b)
	testFSMDumpLocal = testFSMDump[spf.StateAwaitParticipantsConfirmations]

	for participantId := range testIdMapParticipants {
		participantCounter--
		testFSMInstance, err := FromDump(testFSMDumpLocal)

		compareErrNil(t, err)

		compareFSMInstanceNotNil(t, testFSMInstance)

		inState, _ := testFSMInstance.State()
		compareState(t, spf.StateAwaitParticipantsConfirmations, inState)

		fsmResponse, testFSMDumpLocal, err = testFSMInstance.Do(spf.EventConfirmSignatureProposal, requests.SignatureProposalParticipantRequest{
			ParticipantId: participantId,
			CreatedAt:     time.Now(),
		})

		compareErrNil(t, err)

		compareDumpNotZero(t, testFSMDumpLocal)

		compareFSMResponseNotNil(t, fsmResponse)

		if participantCounter > 0 {
			compareState(t, spf.StateAwaitParticipantsConfirmations, fsmResponse.State)
		}

	}

	compareState(t, spf.StateSignatureProposalCollected, fsmResponse.State)

	testFSMDump[spf.StateSignatureProposalCollected] = testFSMDumpLocal

	compareDumpNotZero(t, testFSMDump[spf.StateSignatureProposalCollected])
}

func Test_SignatureProposal_EventConfirmSignatureProposal_Canceled_Participant(t *testing.T) {
	testFSMInstance, err := FromDump(testFSMDump[spf.StateAwaitParticipantsConfirmations])

	compareErrNil(t, err)

	compareFSMInstanceNotNil(t, testFSMInstance)

	inState, _ := testFSMInstance.State()
	compareState(t, spf.StateAwaitParticipantsConfirmations, inState)

	fsmResponse, testFSMDumpLocal, err := testFSMInstance.Do(spf.EventDeclineProposal, requests.SignatureProposalParticipantRequest{
		ParticipantId: 0,
		CreatedAt:     time.Now(),
	})
	require.NoError(t, err)

	compareErrNil(t, err)

	compareDumpNotZero(t, testFSMDumpLocal)

	compareFSMResponseNotNil(t, fsmResponse)

	compareState(t, spf.StateValidationCanceledByParticipant, fsmResponse.State)
}

func Test_SignatureProposal_EventConfirmSignatureProposal_Canceled_Timeout(t *testing.T) {
	testFSMInstance, err := FromDump(testFSMDump[spf.StateAwaitParticipantsConfirmations])

	compareErrNil(t, err)

	compareFSMInstanceNotNil(t, testFSMInstance)

	inState, _ := testFSMInstance.State()
	compareState(t, spf.StateAwaitParticipantsConfirmations, inState)

	fsmResponse, testFSMDumpLocal, err := testFSMInstance.Do(spf.EventConfirmSignatureProposal, requests.SignatureProposalParticipantRequest{
		ParticipantId: 0,
		CreatedAt:     time.Now().Add(time.Hour * 24 * 8),
	})

	compareErrNil(t, err)

	compareDumpNotZero(t, testFSMDumpLocal)

	compareFSMResponseNotNil(t, fsmResponse)

	compareState(t, spf.StateValidationCanceledByTimeout, fsmResponse.State)
}

func Test_DkgProposal_EventDKGInitProcess_Positive(t *testing.T) {
	var fsmResponse *fsm.Response

	testFSMInstance, err := FromDump(testFSMDump[spf.StateSignatureProposalCollected])

	compareErrNil(t, err)

	compareFSMInstanceNotNil(t, testFSMInstance)

	inState, _ := testFSMInstance.State()
	compareState(t, spf.StateSignatureProposalCollected, inState)

	fsmResponse, testFSMDump[dpf.StateDkgCommitsAwaitConfirmations], err = testFSMInstance.Do(dpf.EventDKGInitProcess, requests.DefaultRequest{
		CreatedAt: time.Now(),
	})

	compareErrNil(t, err)

	compareFSMResponseNotNil(t, fsmResponse)

	compareState(t, dpf.StateDkgCommitsAwaitConfirmations, fsmResponse.State)

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

	compareDumpNotZero(t, testFSMDump[dpf.StateDkgCommitsAwaitConfirmations])
}

// Commits
func Test_DkgProposal_EventDKGCommitConfirmationReceived(t *testing.T) {
	var (
		fsmResponse      *fsm.Response
		testFSMDumpLocal []byte
	)

	testFSMDumpLocal = testFSMDump[dpf.StateDkgCommitsAwaitConfirmations]

	for participantId, participant := range testIdMapParticipants {
		testFSMInstance, err := FromDump(testFSMDumpLocal)

		compareErrNil(t, err)

		compareFSMInstanceNotNil(t, testFSMInstance)

		inState, _ := testFSMInstance.State()
		compareState(t, dpf.StateDkgCommitsAwaitConfirmations, inState)

		fsmResponse, testFSMDumpLocal, err = testFSMInstance.Do(dpf.EventDKGCommitConfirmationReceived, requests.DKGProposalCommitConfirmationRequest{
			ParticipantId: participantId,
			Commit:        participant.DkgCommit,
			CreatedAt:     tm,
		})

		compareErrNil(t, err)

		compareFSMResponseNotNil(t, fsmResponse)

		compareDumpNotZero(t, testFSMDumpLocal)

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

		if responseEntry.Username == "" {
			t.Fatalf("expected {Username} non zero length")
		}

		if len(responseEntry.DkgCommit) == 0 {
			t.Fatalf("expected {DkgCommit} non zero length")
		}

		if !reflect.DeepEqual(testIdMapParticipants[responseEntry.ParticipantId].DkgCommit, responseEntry.DkgCommit) {
			t.Fatalf("expected valid {DkgCommit}")
		}
	}

	testFSMDump[dpf.StateDkgDealsAwaitConfirmations] = testFSMDumpLocal

	compareDumpNotZero(t, testFSMDump[dpf.StateDkgDealsAwaitConfirmations])
}

func Test_DkgProposal_EventDKGCommitConfirmationReceived_Canceled_Error(t *testing.T) {
	testFSMInstance, err := FromDump(testFSMDump[dpf.StateDkgCommitsAwaitConfirmations])

	compareErrNil(t, err)

	compareFSMInstanceNotNil(t, testFSMInstance)

	inState, _ := testFSMInstance.State()
	compareState(t, dpf.StateDkgCommitsAwaitConfirmations, inState)

	fsmResponse, testFSMDumpLocal, err := testFSMInstance.Do(dpf.EventDKGCommitConfirmationError, requests.DKGProposalConfirmationErrorRequest{
		ParticipantId: 0,
		Error:         requests.NewFSMError(errors.New("test error")),
		CreatedAt:     time.Now(),
	})

	compareErrNil(t, err)

	compareFSMResponseNotNil(t, fsmResponse)

	compareDumpNotZero(t, testFSMDumpLocal)

	compareState(t, dpf.StateDkgCommitsAwaitCanceledByError, fsmResponse.State)

}

func Test_DkgProposal_EventDKGCommitConfirmationReceived_Canceled_Timeout(t *testing.T) {
	testFSMInstance, err := FromDump(testFSMDump[dpf.StateDkgCommitsAwaitConfirmations])

	compareErrNil(t, err)

	compareFSMInstanceNotNil(t, testFSMInstance)

	inState, _ := testFSMInstance.State()
	compareState(t, dpf.StateDkgCommitsAwaitConfirmations, inState)

	fsmResponse, testFSMDumpLocal, err := testFSMInstance.Do(dpf.EventDKGCommitConfirmationReceived, requests.DKGProposalCommitConfirmationRequest{
		ParticipantId: 0,
		Commit:        testIdMapParticipants[0].DkgCommit,
		CreatedAt:     time.Now().Add(time.Hour * 24 * 8),
	})

	compareErrNil(t, err)

	compareFSMResponseNotNil(t, fsmResponse)

	compareDumpNotZero(t, testFSMDumpLocal)

	compareState(t, dpf.StateDkgCommitsAwaitCanceledByTimeout, fsmResponse.State)

}

// Deals
func Test_DkgProposal_EventDKGDealConfirmationReceived(t *testing.T) {
	var (
		fsmResponse      *fsm.Response
		testFSMDumpLocal []byte
	)

	testFSMDumpLocal = testFSMDump[dpf.StateDkgDealsAwaitConfirmations]

	for participantId, participant := range testIdMapParticipants {
		testFSMInstance, err := FromDump(testFSMDumpLocal)

		compareErrNil(t, err)

		compareFSMInstanceNotNil(t, testFSMInstance)

		inState, _ := testFSMInstance.State()
		compareState(t, dpf.StateDkgDealsAwaitConfirmations, inState)

		fsmResponse, testFSMDumpLocal, err = testFSMInstance.Do(dpf.EventDKGDealConfirmationReceived, requests.DKGProposalDealConfirmationRequest{
			ParticipantId: participantId,
			Deal:          participant.DkgDeal,
			CreatedAt:     tm,
		})

		compareErrNil(t, err)

		compareFSMResponseNotNil(t, fsmResponse)

		compareDumpNotZero(t, testFSMDumpLocal)

		// Deals reached, next stage
		if fsmResponse.State == dpf.StateDkgResponsesAwaitConfirmations {
			break
		}
	}

	compareState(t, dpf.StateDkgResponsesAwaitConfirmations, fsmResponse.State)

	response, ok := fsmResponse.Data.(responses.DKGProposalDealParticipantResponse)

	if !ok {
		t.Fatalf("expected response {DKGProposalDealParticipantResponse}")
	}

	// Deals count less than total users count by 1 unit
	if len(response) != len(testParticipantsListRequest.Participants) {
		t.Fatalf("expected response len {%d}, got {%d}", len(testParticipantsListRequest.Participants), len(response)-1)
	}

	for _, responseEntry := range response {
		if _, ok := testIdMapParticipants[responseEntry.ParticipantId]; !ok {
			t.Fatalf("expected exist {ParticipantId}")
		}

		if responseEntry.Username == "" {
			t.Fatalf("expected {Username} non zero length")
		}

		if len(responseEntry.DkgDeal) == 0 {
			t.Fatalf("expected {DkgDeal} non zero length")
		}

		if !reflect.DeepEqual(testIdMapParticipants[responseEntry.ParticipantId].DkgDeal, responseEntry.DkgDeal) {
			t.Fatalf("expected valid {DkgDeal}")
		}
	}

	testFSMDump[dpf.StateDkgResponsesAwaitConfirmations] = testFSMDumpLocal

	compareDumpNotZero(t, testFSMDump[dpf.StateDkgResponsesAwaitConfirmations])
}

func Test_DkgProposal_EventDKGDealConfirmationReceived_Canceled_Error(t *testing.T) {
	testFSMInstance, err := FromDump(testFSMDump[dpf.StateDkgDealsAwaitConfirmations])

	compareErrNil(t, err)

	compareFSMInstanceNotNil(t, testFSMInstance)

	inState, _ := testFSMInstance.State()
	compareState(t, dpf.StateDkgDealsAwaitConfirmations, inState)

	fsmResponse, testFSMDumpLocal, err := testFSMInstance.Do(dpf.EventDKGDealConfirmationError, requests.DKGProposalConfirmationErrorRequest{
		ParticipantId: 0,
		Error:         requests.NewFSMError(errors.New("test error")),
		CreatedAt:     time.Now(),
	})

	compareErrNil(t, err)

	compareFSMResponseNotNil(t, fsmResponse)

	compareDumpNotZero(t, testFSMDumpLocal)

	compareState(t, dpf.StateDkgDealsAwaitCanceledByError, fsmResponse.State)

}

func Test_DkgProposal_EventDKGDealConfirmationReceived_Canceled_Timeout(t *testing.T) {
	testFSMInstance, err := FromDump(testFSMDump[dpf.StateDkgDealsAwaitConfirmations])

	compareErrNil(t, err)

	compareFSMInstanceNotNil(t, testFSMInstance)

	inState, _ := testFSMInstance.State()
	compareState(t, dpf.StateDkgDealsAwaitConfirmations, inState)

	fsmResponse, testFSMDumpLocal, err := testFSMInstance.Do(dpf.EventDKGDealConfirmationReceived, requests.DKGProposalDealConfirmationRequest{
		ParticipantId: 0,
		Deal:          testIdMapParticipants[0].DkgDeal,
		CreatedAt:     time.Now().Add(time.Hour * 24 * 8),
	})

	compareErrNil(t, err)

	compareFSMResponseNotNil(t, fsmResponse)

	compareDumpNotZero(t, testFSMDumpLocal)

	compareState(t, dpf.StateDkgDealsAwaitCanceledByTimeout, fsmResponse.State)

}

// Responses
func Test_DkgProposal_EventDKGResponseConfirmationReceived_Positive(t *testing.T) {
	var (
		fsmResponse      *fsm.Response
		testFSMDumpLocal []byte
	)

	testFSMDumpLocal = testFSMDump[dpf.StateDkgResponsesAwaitConfirmations]

	for participantId, participant := range testIdMapParticipants {
		testFSMInstance, err := FromDump(testFSMDumpLocal)

		compareErrNil(t, err)

		compareFSMInstanceNotNil(t, testFSMInstance)

		inState, _ := testFSMInstance.State()
		compareState(t, dpf.StateDkgResponsesAwaitConfirmations, inState)

		fsmResponse, testFSMDumpLocal, err = testFSMInstance.Do(dpf.EventDKGResponseConfirmationReceived, requests.DKGProposalResponseConfirmationRequest{
			ParticipantId: participantId,
			Response:      participant.DkgResponse,
			CreatedAt:     tm,
		})

		compareErrNil(t, err)

		compareFSMResponseNotNil(t, fsmResponse)

		compareDumpNotZero(t, testFSMDumpLocal)
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

		if responseEntry.Username == "" {
			t.Fatalf("expected {Username} non zero length")
		}

		if len(responseEntry.DkgResponse) == 0 {
			t.Fatalf("expected {DkgResponse} non zero length")
		}

		if !reflect.DeepEqual(testIdMapParticipants[responseEntry.ParticipantId].DkgResponse, responseEntry.DkgResponse) {
			t.Fatalf("expected valid {DkgResponse}")
		}
	}

	testFSMDump[dpf.StateDkgMasterKeyAwaitConfirmations] = testFSMDumpLocal

	compareDumpNotZero(t, testFSMDump[dpf.StateDkgMasterKeyAwaitConfirmations])
}

func Test_DkgProposal_EventDKGResponseConfirmationError_Canceled_Error(t *testing.T) {
	testFSMInstance, err := FromDump(testFSMDump[dpf.StateDkgResponsesAwaitConfirmations])

	compareErrNil(t, err)

	compareFSMInstanceNotNil(t, testFSMInstance)

	inState, _ := testFSMInstance.State()
	compareState(t, dpf.StateDkgResponsesAwaitConfirmations, inState)

	fsmResponse, testFSMDumpLocal, err := testFSMInstance.Do(dpf.EventDKGResponseConfirmationError, requests.DKGProposalConfirmationErrorRequest{
		ParticipantId: 0,
		Error:         requests.NewFSMError(errors.New("test error")),
		CreatedAt:     time.Now(),
	})

	compareErrNil(t, err)

	compareFSMResponseNotNil(t, fsmResponse)

	compareDumpNotZero(t, testFSMDumpLocal)

	compareState(t, dpf.StateDkgResponsesAwaitCanceledByError, fsmResponse.State)

}

func Test_DkgProposal_EventDKGResponseConfirmationReceived_Canceled_Timeout(t *testing.T) {
	testFSMInstance, err := FromDump(testFSMDump[dpf.StateDkgResponsesAwaitConfirmations])

	compareErrNil(t, err)

	compareFSMInstanceNotNil(t, testFSMInstance)

	inState, _ := testFSMInstance.State()
	compareState(t, dpf.StateDkgResponsesAwaitConfirmations, inState)

	fsmResponse, testFSMDumpLocal, err := testFSMInstance.Do(dpf.EventDKGResponseConfirmationReceived, requests.DKGProposalResponseConfirmationRequest{
		ParticipantId: 0,
		Response:      testIdMapParticipants[0].DkgResponse,
		CreatedAt:     time.Now().Add(time.Hour * 24 * 8),
	})

	compareErrNil(t, err)

	compareFSMResponseNotNil(t, fsmResponse)

	compareDumpNotZero(t, testFSMDumpLocal)

	compareState(t, dpf.StateDkgResponsesAwaitCanceledByTimeout, fsmResponse.State)

}

// Master keys
func Test_DkgProposal_EventDKGMasterKeyConfirmationReceived_Positive(t *testing.T) {
	var (
		fsmResponse      *fsm.Response
		testFSMDumpLocal []byte
	)

	testFSMDumpLocal = testFSMDump[dpf.StateDkgMasterKeyAwaitConfirmations]

	masterKeyMockup := genDataMock(keysMockLen)

	for participantId := range testIdMapParticipants {
		testFSMInstance, err := FromDump(testFSMDumpLocal)

		compareErrNil(t, err)

		compareFSMInstanceNotNil(t, testFSMInstance)

		inState, _ := testFSMInstance.State()
		compareState(t, dpf.StateDkgMasterKeyAwaitConfirmations, inState)

		fsmResponse, testFSMDumpLocal, err = testFSMInstance.Do(dpf.EventDKGMasterKeyConfirmationReceived, requests.DKGProposalMasterKeyConfirmationRequest{
			ParticipantId: participantId,
			MasterKey:     masterKeyMockup,
			CreatedAt:     tm,
		})

		compareErrNil(t, err)

		compareFSMResponseNotNil(t, fsmResponse)

		compareDumpNotZero(t, testFSMDumpLocal)
	}

	compareState(t, dpf.StateDkgMasterKeyCollected, fsmResponse.State)

	testFSMDump[dpf.StateDkgMasterKeyCollected] = testFSMDumpLocal

	compareDumpNotZero(t, testFSMDump[dpf.StateDkgMasterKeyCollected])
}

func Test_DkgProposal_EventDKGMasterKeyConfirmationError_Canceled_Error(t *testing.T) {
	testFSMInstance, err := FromDump(testFSMDump[dpf.StateDkgMasterKeyAwaitConfirmations])

	compareErrNil(t, err)

	compareFSMInstanceNotNil(t, testFSMInstance)

	inState, _ := testFSMInstance.State()
	compareState(t, dpf.StateDkgMasterKeyAwaitConfirmations, inState)

	fsmResponse, testFSMDumpLocal, err := testFSMInstance.Do(dpf.EventDKGMasterKeyConfirmationError, requests.DKGProposalConfirmationErrorRequest{
		ParticipantId: 0,
		Error:         requests.NewFSMError(errors.New("test error")),
		CreatedAt:     time.Now(),
	})

	compareErrNil(t, err)

	compareFSMResponseNotNil(t, fsmResponse)

	compareDumpNotZero(t, testFSMDumpLocal)

	compareState(t, dpf.StateDkgMasterKeyAwaitCanceledByError, fsmResponse.State)

}

func Test_DkgProposal_EventDKGMasterKeyConfirmationReceived_Canceled_Timeout(t *testing.T) {
	testFSMInstance, err := FromDump(testFSMDump[dpf.StateDkgMasterKeyAwaitConfirmations])

	compareErrNil(t, err)

	compareFSMInstanceNotNil(t, testFSMInstance)

	inState, _ := testFSMInstance.State()
	compareState(t, dpf.StateDkgMasterKeyAwaitConfirmations, inState)

	fsmResponse, testFSMDumpLocal, err := testFSMInstance.Do(dpf.EventDKGMasterKeyConfirmationReceived, requests.DKGProposalMasterKeyConfirmationRequest{
		ParticipantId: 0,
		MasterKey:     genDataMock(keysMockLen),
		CreatedAt:     time.Now().Add(time.Hour * 24 * 8),
	})

	compareErrNil(t, err)

	compareFSMResponseNotNil(t, fsmResponse)

	compareDumpNotZero(t, testFSMDumpLocal)

	compareState(t, dpf.StateDkgMasterKeyAwaitCanceledByTimeout, fsmResponse.State)

}

func Test_DkgProposal_EventDKGMasterKeyConfirmationReceived_Canceled_Mismatched(t *testing.T) {
	testFSMInstance, err := FromDump(testFSMDump[dpf.StateDkgMasterKeyAwaitConfirmations])

	compareErrNil(t, err)

	compareFSMInstanceNotNil(t, testFSMInstance)

	inState, _ := testFSMInstance.State()
	compareState(t, dpf.StateDkgMasterKeyAwaitConfirmations, inState)

	fsmResponse, testFSMDumpLocal, err := testFSMInstance.Do(dpf.EventDKGMasterKeyConfirmationReceived, requests.DKGProposalMasterKeyConfirmationRequest{
		ParticipantId: 0,
		MasterKey:     genDataMock(keysMockLen),
		CreatedAt:     time.Now(),
	})

	compareErrNil(t, err)

	compareFSMResponseNotNil(t, fsmResponse)

	compareDumpNotZero(t, testFSMDumpLocal)

	fsmResponse, testFSMDumpLocal, err = testFSMInstance.Do(dpf.EventDKGMasterKeyConfirmationReceived, requests.DKGProposalMasterKeyConfirmationRequest{
		ParticipantId: 1,
		MasterKey:     genDataMock(keysMockLen),
		CreatedAt:     time.Now(),
	})

	compareErrNil(t, err)

	compareFSMResponseNotNil(t, fsmResponse)

	compareDumpNotZero(t, testFSMDumpLocal)

	compareState(t, dpf.StateDkgMasterKeyAwaitCanceledByError, fsmResponse.State)

}

// Signing
func Test_SigningProposal_EventSigningInit(t *testing.T) {
	var fsmResponse *fsm.Response

	testFSMInstance, err := FromDump(testFSMDump[dpf.StateDkgMasterKeyCollected])

	compareErrNil(t, err)

	compareFSMInstanceNotNil(t, testFSMInstance)

	inState, _ := testFSMInstance.State()
	compareState(t, dpf.StateDkgMasterKeyCollected, inState)

	fsmResponse, testFSMDump[sif.StateSigningIdle], err = testFSMInstance.Do(sif.EventSigningInit, requests.DefaultRequest{
		CreatedAt: time.Now(),
	})

	compareErrNil(t, err)

	compareFSMResponseNotNil(t, fsmResponse)

	compareState(t, sif.StateSigningIdle, fsmResponse.State)

	compareDumpNotZero(t, testFSMDump[sif.StateSigningIdle])

}

// Start
func Test_SigningProposal_EventSigningStart(t *testing.T) {
	var (
		fsmResponse *fsm.Response
	)

	testFSMInstance, err := FromDump(testFSMDump[sif.StateSigningIdle])

	compareErrNil(t, err)

	compareFSMInstanceNotNil(t, testFSMInstance)

	inState, _ := testFSMInstance.State()
	compareState(t, sif.StateSigningIdle, inState)

	fsmResponse, testFSMDump[sif.StateSigningAwaitConfirmations], err = testFSMInstance.Do(sif.EventSigningStart, requests.SigningProposalStartRequest{
		SigningID:     "test-signing-id",
		ParticipantId: 1,
		SrcPayload:    []byte("message to sign"),
		CreatedAt:     time.Now(),
	})

	compareErrNil(t, err)

	compareFSMResponseNotNil(t, fsmResponse)

	compareState(t, sif.StateSigningAwaitConfirmations, fsmResponse.State)

	response, ok := fsmResponse.Data.(responses.SigningProposalParticipantInvitationsResponse)

	if !ok {
		t.Fatalf("expected response {SigningProposalParticipantInvitationsResponse}")
	}

	if len(response.Participants) != len(testParticipantsListRequest.Participants) {
		t.Fatalf("expected response len {%d}, got {%d}", len(testParticipantsListRequest.Participants), len(response.Participants))
	}

	if response.SigningId == "" {
		t.Fatalf("expected field {SigningId}")
	}

	if !reflect.DeepEqual(response.SrcPayload, testSigningPayload) {
		t.Fatalf("expected matched {SrcPayload}")
	}

	testSigningId = response.SigningId
	testSigningInitiator = response.InitiatorId

	compareDumpNotZero(t, testFSMDump[sif.StateSigningAwaitConfirmations])
}

func Test_SigningProposal_EventConfirmSigningConfirmation_Positive(t *testing.T) {
	var (
		fsmResponse      *fsm.Response
		testFSMDumpLocal []byte
	)

	participantsCount := len(testIdMapParticipants)

	participantCounter := participantsCount

	testFSMDumpLocal = testFSMDump[sif.StateSigningAwaitConfirmations]

	confirmedParticipantsCount := 1
	for participantId := range testIdMapParticipants {
		if confirmedParticipantsCount >= threshold {
			break
		}
		participantCounter--

		if testSigningInitiator == participantId {
			continue
		}

		testFSMInstance, err := FromDump(testFSMDumpLocal)

		compareErrNil(t, err)

		compareFSMInstanceNotNil(t, testFSMInstance)

		inState, _ := testFSMInstance.State()
		compareState(t, sif.StateSigningAwaitConfirmations, inState)

		fsmResponse, testFSMDumpLocal, err = testFSMInstance.Do(sif.EventConfirmSigningConfirmation, requests.SigningProposalParticipantRequest{
			SigningId:     testSigningId,
			ParticipantId: participantId,
			CreatedAt:     time.Now(),
		})

		compareErrNil(t, err)

		compareDumpNotZero(t, testFSMDumpLocal)

		compareFSMResponseNotNil(t, fsmResponse)

		confirmedParticipantsCount++

		if confirmedParticipantsCount < threshold {
			compareState(t, sif.StateSigningAwaitConfirmations, fsmResponse.State)
		} else if confirmedParticipantsCount >= threshold {
			compareState(t, sif.StateSigningAwaitPartialSigns, fsmResponse.State)
		}
	}

	compareState(t, sif.StateSigningAwaitPartialSigns, fsmResponse.State)

	response, ok := fsmResponse.Data.(responses.SigningPartialSignsParticipantInvitationsResponse)

	if !ok {
		t.Fatalf("expected response {SigningProposalParticipantInvitationsResponse}")
	}

	if response.SigningId == "" {
		t.Fatalf("expected field {SigningId}")
	}

	if !reflect.DeepEqual(response.SrcPayload, testSigningPayload) {
		t.Fatalf("expected matched {SrcPayload}")
	}

	testFSMDump[sif.StateSigningAwaitPartialSigns] = testFSMDumpLocal

	compareDumpNotZero(t, testFSMDump[sif.StateSigningAwaitPartialSigns])
}

func Test_SigningProposal_EventDeclineProposal_Canceled_Participants(t *testing.T) {
	var (
		fsmResponse      *fsm.Response
		testFSMDumpLocal []byte
		err              error
	)

	testFSMDumpLocal = testFSMDump[sif.StateSigningAwaitConfirmations]

	testFSMInstance, err := FromDump(testFSMDumpLocal)

	compareErrNil(t, err)

	compareFSMInstanceNotNil(t, testFSMInstance)

	inState, _ := testFSMInstance.State()
	compareState(t, sif.StateSigningAwaitConfirmations, inState)

	declinedParticipantsCount := 0
	for participantId, _ := range testIdMapParticipants {
		if declinedParticipantsCount > participantsNumber-threshold {
			break
		}

		if testSigningInitiator == participantId {
			continue
		}

		testFSMInstance, err = FromDump(testFSMDumpLocal)

		compareErrNil(t, err)

		compareFSMInstanceNotNil(t, testFSMInstance)

		inState, _ := testFSMInstance.State()
		compareState(t, sif.StateSigningAwaitConfirmations, inState)

		fsmResponse, testFSMDumpLocal, err = testFSMInstance.Do(sif.EventDeclineSigningConfirmation, requests.SigningProposalParticipantRequest{
			SigningId:     testSigningId,
			ParticipantId: participantId,
			CreatedAt:     time.Now(),
		})

		compareErrNil(t, err)

		compareDumpNotZero(t, testFSMDumpLocal)

		compareFSMResponseNotNil(t, fsmResponse)
		declinedParticipantsCount++

		if declinedParticipantsCount > participantsNumber-threshold {
			compareState(t, sif.StateSigningConfirmationsAwaitCancelledByParticipant, fsmResponse.State)
		} else if declinedParticipantsCount <= participantsNumber-threshold {
			compareState(t, sif.StateSigningAwaitConfirmations, fsmResponse.State)
		}
	}

	compareState(t, sif.StateSigningConfirmationsAwaitCancelledByParticipant, fsmResponse.State)
}

func Test_SigningProposal_EventConfirmSignatureProposal_Canceled_Timeout(t *testing.T) {
	testFSMInstance, err := FromDump(testFSMDump[sif.StateSigningAwaitConfirmations])

	compareErrNil(t, err)

	compareFSMInstanceNotNil(t, testFSMInstance)

	inState, _ := testFSMInstance.State()
	compareState(t, sif.StateSigningAwaitConfirmations, inState)

	fsmResponse, testFSMDumpLocal, err := testFSMInstance.Do(sif.EventConfirmSigningConfirmation, requests.SigningProposalParticipantRequest{
		SigningId:     testSigningId,
		ParticipantId: 0,
		CreatedAt:     time.Now().Add(time.Hour * 24 * 8),
	})

	compareErrNil(t, err)

	compareDumpNotZero(t, testFSMDumpLocal)

	compareFSMResponseNotNil(t, fsmResponse)

	compareState(t, sif.StateSigningConfirmationsAwaitCancelledByTimeout, fsmResponse.State)
}

func Test_SigningProposal_EventSigningPartialKeyReceived_Positive(t *testing.T) {
	var (
		fsmResponse      *fsm.Response
		testFSMDumpLocal []byte
	)

	participantsCount := len(testIdMapParticipants)

	participantCounter := participantsCount

	testFSMDumpLocal = testFSMDump[sif.StateSigningAwaitPartialSigns]

	broadcastParticipantsCount := 0
	for participantId, participant := range testIdMapParticipants {
		if broadcastParticipantsCount >= threshold {
			break
		}
		participantCounter--

		testFSMInstance, err := FromDump(testFSMDumpLocal)

		compareErrNil(t, err)

		compareFSMInstanceNotNil(t, testFSMInstance)

		inState, _ := testFSMInstance.State()
		compareState(t, sif.StateSigningAwaitPartialSigns, inState)

		fsmResponse, testFSMDumpLocal, err = testFSMInstance.Do(sif.EventSigningPartialSignReceived, requests.SigningProposalPartialSignRequest{
			SigningId:     testSigningId,
			ParticipantId: participantId,
			PartialSign:   participant.DkgPartialKey,
			CreatedAt:     time.Now(),
		})

		compareErrNil(t, err)

		compareDumpNotZero(t, testFSMDumpLocal)

		compareFSMResponseNotNil(t, fsmResponse)

		broadcastParticipantsCount++

		if broadcastParticipantsCount < threshold {
			compareState(t, sif.StateSigningAwaitPartialSigns, fsmResponse.State)
		} else if broadcastParticipantsCount >= threshold {
			compareState(t, sif.StateSigningPartialSignsCollected, fsmResponse.State)
		}
	}

	compareState(t, sif.StateSigningPartialSignsCollected, fsmResponse.State)

	response, ok := fsmResponse.Data.(responses.SigningProcessParticipantResponse)

	if !ok {
		t.Fatalf("expected response {SigningProcessParticipantResponse}")
	}

	if response.SigningId == "" {
		t.Fatalf("expected field {SigningId}")
	}

	if !reflect.DeepEqual(response.SrcPayload, testSigningPayload) {
		t.Fatalf("expected matched {SrcPayload}")
	}

	testFSMDump[sif.StateSigningPartialSignsCollected] = testFSMDumpLocal

	compareDumpNotZero(t, testFSMDump[sif.StateSigningPartialSignsCollected])
}

func Test_SigningProposal_EventPartialKeysReceived_Failed_Participants(t *testing.T) {
	var (
		fsmResponse      *fsm.Response
		testFSMDumpLocal []byte
		err              error
	)

	testFSMDumpLocal = testFSMDump[sif.StateSigningAwaitPartialSigns]

	testFSMInstance, err := FromDump(testFSMDumpLocal)

	compareErrNil(t, err)

	compareFSMInstanceNotNil(t, testFSMInstance)

	inState, _ := testFSMInstance.State()
	compareState(t, sif.StateSigningAwaitPartialSigns, inState)

	failedParticipantsCount := 0
	for participantId, _ := range testIdMapParticipants {
		if failedParticipantsCount > participantsNumber-threshold {
			break
		}

		testFSMInstance, err = FromDump(testFSMDumpLocal)

		compareErrNil(t, err)

		compareFSMInstanceNotNil(t, testFSMInstance)

		inState, _ := testFSMInstance.State()
		compareState(t, sif.StateSigningAwaitPartialSigns, inState)

		fsmResponse, testFSMDumpLocal, err = testFSMInstance.Do(sif.EventSigningPartialSignError, requests.SignatureProposalConfirmationErrorRequest{
			Error:         requests.NewFSMError(errors.New("some error")),
			ParticipantId: participantId,
			CreatedAt:     time.Now(),
		})

		compareErrNil(t, err)

		compareDumpNotZero(t, testFSMDumpLocal)

		compareFSMResponseNotNil(t, fsmResponse)
		failedParticipantsCount++

		if failedParticipantsCount > participantsNumber-threshold {
			compareState(t, sif.StateSigningPartialSignsAwaitCancelledByError, fsmResponse.State)
		} else if failedParticipantsCount <= participantsNumber-threshold {
			compareState(t, sif.StateSigningAwaitPartialSigns, fsmResponse.State)
		}
	}

	compareState(t, sif.StateSigningPartialSignsAwaitCancelledByError, fsmResponse.State)
}

func Test_DkgProposal_EventSigningRestart_Positive(t *testing.T) {
	var (
		fsmResponse      *fsm.Response
		testFSMDumpLocal []byte
	)

	testFSMInstance, err := FromDump(testFSMDump[sif.StateSigningPartialSignsCollected])

	compareErrNil(t, err)

	compareFSMInstanceNotNil(t, testFSMInstance)

	inState, _ := testFSMInstance.State()
	compareState(t, sif.StateSigningPartialSignsCollected, inState)

	fsmResponse, testFSMDumpLocal, err = testFSMInstance.Do(sif.EventSigningRestart, requests.DefaultRequest{
		CreatedAt: time.Now(),
	})

	compareErrNil(t, err)

	compareFSMResponseNotNil(t, fsmResponse)

	compareState(t, sif.StateSigningIdle, fsmResponse.State)

	compareDumpNotZero(t, testFSMDumpLocal)
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
	require.NoError(t, err)

	s1, err := testFSMInstance1.State()

	compareErrNil(t, err)

	s2, err := testFSMInstance2.State()
	require.NoError(t, err)

	if s1 == s2 {
		t.Fatalf("MATCH STATES {%s}", s1)
	}

}
