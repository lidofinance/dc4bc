package state_machines

import (
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

var (
	tm                          = time.Now()
	testParticipantsListRequest = requests.SignatureProposalParticipantsListRequest{
		Participants: []*requests.SignatureProposalParticipantsEntry{
			{
				"User 1",
				[]byte("pubkey123123"),
			},
			{
				"User 2",
				[]byte("pubkey456456"),
			},
			{
				"User 3",
				[]byte("pubkey789789"),
			},
		},
		CreatedAt: &tm,
	}
)

func TestCreate_Positive(t *testing.T) {
	testFSMInstance, err := Create(testTransactionId)
	if err != nil {
		t.Fatalf("expected nil error")
	}

	if testFSMInstance == nil {
		t.Fatalf("expected {*FSMInstance}")
	}
}

func TestCreate_Negative(t *testing.T) {
	_, err := Create("")
	if err == nil {
		t.Fatalf("expected error for empty {transactionId}")
	}
}

func Test_Workflow(t *testing.T) {
	testFSMInstance, err := Create(testTransactionId)
	if err != nil {
		t.Fatalf("expected nil error, got {%s}", err)
	}

	if testFSMInstance == nil {
		t.Fatalf("expected {*FSMInstance}")
	}

	if testFSMInstance.machine.Name() != spf.FsmName {
		t.Fatalf("expected machine name {%s}", spf.FsmName)
	}

	if testFSMInstance.machine.State() != spf.StateParticipantsConfirmationsInit {
		t.Fatalf("expected inital state {%s}", spf.StateParticipantsConfirmationsInit)
	}

	fsmResponse, dump, err := testFSMInstance.Do(spf.EventInitProposal, testParticipantsListRequest)

	if err != nil {
		t.Fatalf("expected nil error")
	}

	if len(dump) == 0 {
		t.Fatalf("expected non zero dump, when executed without error")
	}

	if fsmResponse == nil {
		t.Fatalf("expected {*fsm.FSMResponse}")
	}

	if fsmResponse.State != spf.StateAwaitParticipantsConfirmations {
		t.Fatalf("expected state {%s}", spf.StateAwaitParticipantsConfirmations)
	}

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
			t.Fatalf("expected not empty {EncryptedInvitation}")
		}

		if participant.PubKeyFingerprint == "" {
			t.Fatalf("expected not empty {PubKeyFingerprint}")
		}

		participantsMap[participant.ParticipantId] = participant
	}

	tm = tm.Add(10 * time.Second)

	testFSMInstance, err = FromDump(dump)

	if err != nil {
		t.Fatalf("expected nil error, got {%s}", err)
	}

	if testFSMInstance == nil {
		t.Fatalf("expected {*FSMInstance}")
	}

	for _, participant := range participantsMap {
		response, _, err := testFSMInstance.Do(spf.EventConfirmProposal, requests.SignatureProposalParticipantRequest{
			PubKeyFingerprint:   participant.PubKeyFingerprint,
			EncryptedInvitation: "lll",
			CreatedAt:           &tm,
		})
		log.Println(response, err)
	}
}
