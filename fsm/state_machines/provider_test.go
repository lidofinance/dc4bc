package state_machines

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"github.com/depools/dc4bc/fsm/fsm"
	dpf "github.com/depools/dc4bc/fsm/state_machines/dkg_proposal_fsm"
	spf "github.com/depools/dc4bc/fsm/state_machines/signature_proposal_fsm"
	"github.com/depools/dc4bc/fsm/types/requests"
	"github.com/depools/dc4bc/fsm/types/responses"
	"log"
	"testing"
	"time"
)

const (
	testTransactionId = "d8a928b2043db77e340b523547bf16cb4aa483f0645fe0a290ed1f20aab76257"
)

type testExternalParticipants struct {
	Title   string
	PrivKey *rsa.PrivateKey
	PubKey  *rsa.PublicKey
}

var (
	tm = time.Now()

	testParticipants = map[string]*testExternalParticipants{}

	testParticipantsListRequest = requests.SignatureProposalParticipantsListRequest{
		Participants: []*requests.SignatureProposalParticipantsEntry{},
		CreatedAt:    &tm,
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

		marshaledPubKey := x509.MarshalPKCS1PublicKey(&key.PublicKey)
		hash := sha1.Sum(marshaledPubKey)

		fingerprint := base64.StdEncoding.EncodeToString(hash[:])

		participant := &testExternalParticipants{
			Title:   fmt.Sprintf("User %d", i),
			PrivKey: key,
			PubKey:  &key.PublicKey,
		}
		testParticipants[fingerprint] = participant
	}

	participantsForRequest := make([]*requests.SignatureProposalParticipantsEntry, 0)

	for _, participant := range testParticipants {

		participantsForRequest = append(participantsForRequest, &requests.SignatureProposalParticipantsEntry{
			Title:  participant.Title,
			PubKey: x509.MarshalPKCS1PublicKey(participant.PubKey),
		})
	}
	testParticipantsListRequest.Participants = participantsForRequest
}

func TestCreate_Positive(t *testing.T) {
	testFSMInstance, err := Create()
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
	testFSMInstance, err := Create()

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

		if participant.Title == "" {
			t.Fatalf("expected not empty {Title}")
		}

		if participant.EncryptedInvitation == "" {
			t.Fatalf("expected not empty {DecryptedInvitation}")
		}

		if participant.PubKeyFingerprint == "" {
			t.Fatalf("expected not empty {PubKeyFingerprint}")
		}

		participantsMap[participant.ParticipantId] = participant
	}

	tm = tm.Add(10 * time.Hour)

	participantsCount := len(participantsMap)

	participantCounter := participantsCount

	for _, participant := range participantsMap {
		participantCounter--
		testFSMInstance, err = FromDump(dump)

		compareErrNil(t, err)

		compareFSMInstanceNotNil(t, testFSMInstance)

		if _, ok := testParticipants[participant.PubKeyFingerprint]; !ok {
			t.Fatalf("not found external user data for response fingerprint")
		}

		r := rand.Reader
		encrypted, err := rsa.DecryptPKCS1v15(r, testParticipants[participant.PubKeyFingerprint].PrivKey, []byte(participant.EncryptedInvitation))

		if err != nil {
			t.Fatalf("cannot encrypt {DecryptedInvitation} with private key")
		}

		fsmResponse, dump, err = testFSMInstance.Do(spf.EventConfirmProposal, requests.SignatureProposalParticipantRequest{
			PubKeyFingerprint:   participant.PubKeyFingerprint,
			DecryptedInvitation: string(encrypted),
			CreatedAt:           &tm,
		})

		compareErrNil(t, err)

		compareDumpNotZero(t, dump)

		compareFSMResponseNotNil(t, fsmResponse)

		if participantCounter > 0 {
			compareState(t, spf.StateAwaitParticipantsConfirmations, fsmResponse.State)
		} else {
			compareState(t, dpf.StateDkgInitial, fsmResponse.State)
		}

	}

	// PubKeys

	for _, participant := range participantsMap {
		participantCounter--
		testFSMInstance, err = FromDump(dump)

		compareErrNil(t, err)

		compareFSMInstanceNotNil(t, testFSMInstance)

		if _, ok := testParticipants[participant.PubKeyFingerprint]; !ok {
			t.Fatalf("not found external user data for response fingerprint")
		}

		pubKeyMock := make([]byte, 128)
		_, err := rand.Read(pubKeyMock)
		if err != nil {
			compareErrNil(t, err)
		}

		fsmResponse, dump, err = testFSMInstance.Do(dpf.EventDKGPubKeyConfirmationReceived, requests.DKGProposalPubKeyConfirmationRequest{
			ParticipantId: participant.ParticipantId,
			PubKey:        pubKeyMock,
			CreatedAt:     &tm,
		})

		compareErrNil(t, err)

		compareDumpNotZero(t, dump)

		compareFSMResponseNotNil(t, fsmResponse)

		/*if participantCounter > 0 {
			compareState(t, dpf.StateDkgPubKeysAwaitConfirmations, fsmResponse.State)
		} else {
			compareState(t, dpf.StateDkgCommitsAwaitConfirmations, fsmResponse.State)
		}*/
	}

	compareState(t, dpf.StateDkgCommitsAwaitConfirmations, fsmResponse.State)

	// Commits

	for _, participant := range participantsMap {
		participantCounter--
		testFSMInstance, err = FromDump(dump)

		compareErrNil(t, err)

		compareFSMInstanceNotNil(t, testFSMInstance)

		if _, ok := testParticipants[participant.PubKeyFingerprint]; !ok {
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
			CreatedAt:     &tm,
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

		if _, ok := testParticipants[participant.PubKeyFingerprint]; !ok {
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
			CreatedAt:     &tm,
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

		if _, ok := testParticipants[participant.PubKeyFingerprint]; !ok {
			t.Fatalf("not found external user data for response fingerprint")
		}

		responseMock := make([]byte, 128)
		_, err := rand.Read(responseMock)
		if err != nil {
			compareErrNil(t, err)
		}

		fsmResponse, dump, err = testFSMInstance.Do(dpf.EventDKGDealConfirmationReceived, requests.DKGProposalDealConfirmationRequest{
			ParticipantId: participant.ParticipantId,
			Deal:          responseMock,
			CreatedAt:     &tm,
		})

		compareErrNil(t, err)

		compareDumpNotZero(t, dump)

		compareFSMResponseNotNil(t, fsmResponse)

	}

	compareState(t, fsm.StateGlobalDone, fsmResponse.State)
}

/*
func Test_SignatureProposal_Negative_By_Decline(t *testing.T) {
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

		if participant.Title == "" {
			t.Fatalf("expected not empty {Title}")
		}

		if participant.EncryptedInvitation == "" {
			t.Fatalf("expected not empty {DecryptedInvitation}")
		}

		if participant.PubKeyFingerprint == "" {
			t.Fatalf("expected not empty {PubKeyFingerprint}")
		}

		participantsMap[participant.ParticipantId] = participant
	}

	tm = tm.Add(10 * time.Second)

	participantsCount := len(participantsMap)

	participantCounter := participantsCount

	for _, participant := range participantsMap {
		participantCounter--
		testFSMInstance, err = FromDump(dump)

		compareErrNil(t, err)

		compareFSMInstanceNotNil(t, testFSMInstance)

		if _, ok := testParticipants[participant.PubKeyFingerprint]; !ok {
			t.Fatalf("not found external user data for response fingerprint")
		}

		r := rand.Reader
		encrypted, err := rsa.DecryptPKCS1v15(r, testParticipants[participant.PubKeyFingerprint].PrivKey, []byte(participant.EncryptedInvitation))

		if err != nil {
			t.Fatalf("cannot encrypt {DecryptedInvitation} with private key")
		}

		fsmResponse, dump, err = testFSMInstance.Do(spf.EventDeclineProposal, requests.SignatureProposalParticipantRequest{
			PubKeyFingerprint:   participant.PubKeyFingerprint,
			DecryptedInvitation: string(encrypted),
			CreatedAt:           &tm,
		})

		compareErrNil(t, err)

		compareDumpNotZero(t, dump)

		compareFSMResponseNotNil(t, fsmResponse)

		compareState(t, spf.StateValidationCanceledByParticipant,  fsmResponse.State)


	}
}
*/
