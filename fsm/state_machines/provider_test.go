package state_machines

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	sif "github.com/depools/dc4bc/fsm/state_machines/signing_proposal_fsm"
	"log"
	"testing"
	"time"

	"github.com/depools/dc4bc/fsm/fsm"
	dpf "github.com/depools/dc4bc/fsm/state_machines/dkg_proposal_fsm"
	spf "github.com/depools/dc4bc/fsm/state_machines/signature_proposal_fsm"
	"github.com/depools/dc4bc/fsm/types/requests"
	"github.com/depools/dc4bc/fsm/types/responses"
)

type testExternalParticipants struct {
	Addr      string
	PrivKey   *rsa.PrivateKey
	PubKey    *rsa.PublicKey
	DkgPubKey []byte
}

var (
	tm = time.Now()

	dkgId = "1b7a6382afe0fbe2ff127a5779f5e9b042e685cabefeadcf4ef27c6089a56bfb"

	testParticipants = map[int]*testExternalParticipants{}

	testParticipantsListRequest = requests.SignatureProposalParticipantsListRequest{
		Participants: []*requests.SignatureProposalParticipantsEntry{},
		CreatedAt:    tm,
	}

	testFSMDump []byte
)

func init() {

	r := rand.Reader

	for i := 0; i < 3; i++ {
		key, err := rsa.GenerateKey(r, 2048)

		if err != nil {
			log.Fatal("Cannot generate key for user:", err)
			return
		}

		key.Precompute()

		pubKeyMock := make([]byte, 128)

		rand.Read(pubKeyMock)

		participant := &testExternalParticipants{
			Addr:      fmt.Sprintf("User %d", i),
			PrivKey:   key,
			PubKey:    &key.PublicKey,
			DkgPubKey: pubKeyMock,
		}
		testParticipants[i] = participant
	}

	participantsForRequest := make([]*requests.SignatureProposalParticipantsEntry, 0)

	for _, participant := range testParticipants {

		participantsForRequest = append(participantsForRequest, &requests.SignatureProposalParticipantsEntry{
			Addr:      participant.Addr,
			PubKey:    x509.MarshalPKCS1PublicKey(participant.PubKey),
			DkgPubKey: participant.DkgPubKey,
		})
	}
	testParticipantsListRequest.Participants = participantsForRequest
	testParticipantsListRequest.SigningThreshold = len(participantsForRequest)
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

	compareDumpNotZero(t, testFSMDump)
}

func Test_SignatureProposal_Positive(t *testing.T) {
	testFSMInstance, err := FromDump(testFSMDump)

	compareErrNil(t, err)

	compareFSMInstanceNotNil(t, testFSMInstance)

	fsmResponse, dump, err := testFSMInstance.Do(spf.EventInitProposal, testParticipantsListRequest)

	compareErrNil(t, err)

	compareDumpNotZero(t, dump)

	compareFSMResponseNotNil(t, fsmResponse)

	compareState(t, spf.StateAwaitParticipantsConfirmations, fsmResponse.State)

	testParticipantsListResponse, ok := fsmResponse.Data.(responses.SignatureProposalParticipantInvitationsResponse)

	if !ok {
		t.Fatalf("expected response {SignatureProposalParticipantInvitationsResponse}")
	}

	if len(testParticipantsListResponse) != len(testParticipantsListRequest.Participants) {
		t.Fatalf("expected response len {%d}, got {%d}", len(testParticipantsListRequest.Participants), len(testParticipantsListResponse))
	}

	participantsMap := map[int]*responses.SignatureProposalParticipantInvitationEntry{}

	for _, participant := range testParticipantsListResponse {
		if _, ok := participantsMap[participant.ParticipantId]; ok {
			t.Fatalf("expected unique {ParticipantId}")
		}

		if participant.Addr == "" {
			t.Fatalf("expected not empty {Addr}")
		}

		participantsMap[participant.ParticipantId] = participant
	}

	tm = tm.Add(1 * time.Hour)

	participantsCount := len(participantsMap)

	participantCounter := participantsCount

	for _, participant := range participantsMap {
		participantCounter--
		testFSMInstance, err = FromDump(dump)

		compareErrNil(t, err)

		compareFSMInstanceNotNil(t, testFSMInstance)

		fsmResponse, dump, err = testFSMInstance.Do(spf.EventConfirmSignatureProposal, requests.SignatureProposalParticipantRequest{
			ParticipantId: participant.ParticipantId,
			CreatedAt:     tm,
		})

		compareErrNil(t, err)

		compareDumpNotZero(t, dump)

		compareFSMResponseNotNil(t, fsmResponse)

		if participantCounter > 0 {
			compareState(t, spf.StateAwaitParticipantsConfirmations, fsmResponse.State)
		}

	}

	compareState(t, spf.StateSignatureProposalCollected, fsmResponse.State)

	testFSMInstance, err = FromDump(dump)

	fsmResponse, dump, err = testFSMInstance.Do(dpf.EventDKGInitProcess, requests.DefaultRequest{
		CreatedAt: time.Now(),
	})

	compareErrNil(t, err)

	compareDumpNotZero(t, dump)

	compareFSMResponseNotNil(t, fsmResponse)

	compareState(t, dpf.StateDkgCommitsAwaitConfirmations, fsmResponse.State)

	// Commits

	for _, participant := range participantsMap {
		participantCounter--
		testFSMInstance, err = FromDump(dump)

		compareErrNil(t, err)

		compareFSMInstanceNotNil(t, testFSMInstance)

		if _, ok := testParticipants[participant.ParticipantId]; !ok {
			t.Fatalf("not found external user data for response fingerprint")
		}

		commitMock := make([]byte, 128)
		_, err := rand.Read(commitMock)
		if err != nil {
			compareErrNil(t, err)
		}

		fsmResponse, dump, err = testFSMInstance.Do(dpf.EventDKGCommitConfirmationReceived, requests.DKGProposalCommitConfirmationRequest{
			ParticipantId: participant.ParticipantId,
			Commit:        commitMock,
			CreatedAt:     tm,
		})

		compareErrNil(t, err)

		compareDumpNotZero(t, dump)

		compareFSMResponseNotNil(t, fsmResponse)

	}

	compareState(t, dpf.StateDkgDealsAwaitConfirmations, fsmResponse.State)

	// Deals

	for _, participant := range participantsMap {
		participantCounter--
		testFSMInstance, err = FromDump(dump)

		compareErrNil(t, err)

		compareFSMInstanceNotNil(t, testFSMInstance)

		if _, ok := testParticipants[participant.ParticipantId]; !ok {
			t.Fatalf("not found external user data for response fingerprint")
		}

		dealMock := make([]byte, 128)
		_, err := rand.Read(dealMock)
		if err != nil {
			compareErrNil(t, err)
		}

		fsmResponse, dump, err = testFSMInstance.Do(dpf.EventDKGDealConfirmationReceived, requests.DKGProposalDealConfirmationRequest{
			ParticipantId: participant.ParticipantId,
			Deal:          dealMock,
			CreatedAt:     tm,
		})

		compareErrNil(t, err)

		compareDumpNotZero(t, dump)

		compareFSMResponseNotNil(t, fsmResponse)

	}

	compareState(t, dpf.StateDkgResponsesAwaitConfirmations, fsmResponse.State)

	// Responses

	for _, participant := range participantsMap {
		participantCounter--
		testFSMInstance, err = FromDump(dump)

		compareErrNil(t, err)

		compareFSMInstanceNotNil(t, testFSMInstance)

		if _, ok := testParticipants[participant.ParticipantId]; !ok {
			t.Fatalf("not found external user data for response fingerprint")
		}

		responseMock := make([]byte, 128)
		_, err := rand.Read(responseMock)
		if err != nil {
			compareErrNil(t, err)
		}

		fsmResponse, dump, err = testFSMInstance.Do(dpf.EventDKGResponseConfirmationReceived, requests.DKGProposalResponseConfirmationRequest{
			ParticipantId: participant.ParticipantId,
			Response:      responseMock,
			CreatedAt:     tm,
		})

		compareErrNil(t, err)

		compareDumpNotZero(t, dump)

		compareFSMResponseNotNil(t, fsmResponse)

	}

	compareState(t, dpf.StateDkgMasterKeyAwaitConfirmations, fsmResponse.State)

	// Master keys

	masterKeyMock := make([]byte, 128)
	_, err = rand.Read(masterKeyMock)
	if err != nil {
		compareErrNil(t, err)
	}

	for _, participant := range participantsMap {
		participantCounter--
		testFSMInstance, err = FromDump(dump)

		compareErrNil(t, err)

		compareFSMInstanceNotNil(t, testFSMInstance)

		if _, ok := testParticipants[participant.ParticipantId]; !ok {
			t.Fatalf("not found external user data for response fingerprint")
		}

		fsmResponse, dump, err = testFSMInstance.Do(dpf.EventDKGMasterKeyConfirmationReceived, requests.DKGProposalMasterKeyConfirmationRequest{
			ParticipantId: participant.ParticipantId,
			MasterKey:     masterKeyMock,
			CreatedAt:     tm,
		})

		compareErrNil(t, err)

		compareDumpNotZero(t, dump)

		compareFSMResponseNotNil(t, fsmResponse)

	}

	compareState(t, dpf.StateDkgMasterKeyCollected, fsmResponse.State)

	// Signing

	testFSMInstance, err = FromDump(dump)

	compareErrNil(t, err)

	compareFSMInstanceNotNil(t, testFSMInstance)

	fsmResponse, dump, err = testFSMInstance.Do(sif.EventSigningInit, requests.DefaultRequest{
		CreatedAt: time.Now(),
	})

	compareErrNil(t, err)

	compareDumpNotZero(t, dump)

	compareFSMResponseNotNil(t, fsmResponse)

	compareState(t, sif.StateSigningIdle, fsmResponse.State)

	// Start

	testFSMInstance, err = FromDump(dump)

	compareErrNil(t, err)

	compareFSMInstanceNotNil(t, testFSMInstance)

	fsmResponse, dump, err = testFSMInstance.Do(sif.EventSigningStart, requests.SigningProposalStartRequest{
		ParticipantId: 1,
		SrcPayload:    []byte("message to sign"),
		CreatedAt:     time.Now(),
	})

	compareErrNil(t, err)

	compareDumpNotZero(t, dump)

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

func Test_DKGProposal_Positive(t *testing.T) {

}
