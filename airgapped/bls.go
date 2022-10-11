package airgapped

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/corestario/kyber/pairing"
	"github.com/corestario/kyber/sign/bls"
	"github.com/corestario/kyber/sign/tbls"
	client "github.com/lidofinance/dc4bc/client/types"
	"github.com/lidofinance/dc4bc/fsm/state_machines/signing_proposal_fsm"
	"github.com/lidofinance/dc4bc/fsm/types/requests"
	"github.com/lidofinance/dc4bc/fsm/types/responses"
	"github.com/lidofinance/dc4bc/pkg/wc_rotation"
)

type SignData struct {
	ID   string
	Data []byte
}

func extractMessageFromBaked(id int) (SignData, error) {
	validatorsIDS := strings.Split(wc_rotation.Payloads, "\n")
	vID, err := strconv.Atoi(validatorsIDS[id])
	if err != nil {
		return SignData{}, fmt.Errorf("failed to parse int from str(%s): %w", validatorsIDS[id], err)
	}
	root, err := wc_rotation.GetSigningRoot(uint64(vID))
	if err != nil {
		return SignData{}, fmt.Errorf("failed to get signed root: %w", err)
	}
	return SignData{
		ID:   fmt.Sprintf("bakedrange%d", id),
		Data: root[:],
	}, nil
}

func getPayloadFromMessages(msgs []requests.MessageToSign) ([]SignData, error) {
	var signData []SignData
	for _, m := range msgs {
		if m.Payload != nil {
			signData = append(signData, SignData{
				ID:   m.MessageID,
				Data: m.Payload,
			})
		} else {
			for i := m.RangeStart; i <= m.RangeEnd; i++ {
				data, err := extractMessageFromBaked(i)
				if err != nil {
					return nil, fmt.Errorf("failed to extractMessageFromBaked: %w", err)
				}
				signData = append(signData, data)
			}
		}
	}
	return signData, nil
}

// handleStateSigningAwaitPartialSigns takes a data to sign as payload and returns a partial sign for the data to broadcast
func (am *Machine) handleStateSigningAwaitPartialSigns(o *client.Operation) error {
	var (
		payload        responses.SigningPartialSignsParticipantInvitationsResponse
		messagesToSign []requests.MessageToSign
		err            error
	)

	if err = json.Unmarshal(o.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	if err = json.Unmarshal(payload.SrcPayload, &messagesToSign); err != nil {
		return fmt.Errorf("failed to unmarshal messages to sign: %w", err)
	}

	signs := make([]requests.PartialSign, 0, len(messagesToSign))
	participantID, err := am.getParticipantID(o.DKGIdentifier)
	if err != nil {
		return fmt.Errorf("failed to get paricipant id: %w", err)
	}

	signData, err := getPayloadFromMessages(messagesToSign)
	if err != nil {
		return fmt.Errorf("failed to extract messages from task: %w", err)
	}

	for _, s := range signData {
		partialSign, err := am.createPartialSign(s.Data, o.DKGIdentifier)
		if err != nil {
			return fmt.Errorf("failed to create partialSign for msg: %w", err)
		}

		signs = append(signs, requests.PartialSign{
			MessageID: s.ID,
			Sign:      partialSign,
		})
	}

	req := requests.SigningProposalBatchPartialSignRequests{
		BatchID:       payload.BatchID,
		ParticipantId: participantID,
		PartialSigns:  signs,
		CreatedAt:     o.CreatedAt,
	}

	reqBz, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to generate fsm request: %w", err)
	}

	o.Event = signing_proposal_fsm.EventSigningPartialSignReceived
	o.ResultMsgs = append(o.ResultMsgs, createMessage(*o, reqBz))
	return nil
}

// createPartialSign returns a partial sign of a given message
// with using of a private part of the reconstructed DKG key of a given DKG round
func (am *Machine) createPartialSign(msg []byte, dkgIdentifier string) ([]byte, error) {
	blsKeyring, err := am.loadBLSKeyring(dkgIdentifier)
	if err != nil {
		return nil, fmt.Errorf("failed to load blsKeyring: %w", err)
	}

	return tbls.Sign(am.baseSuite.(pairing.Suite), blsKeyring.Share, msg)
}

// VerifySign verifies a signature of a message
func (am *Machine) VerifySign(msg []byte, fullSignature []byte, dkgIdentifier string) error {
	blsKeyring, err := am.loadBLSKeyring(dkgIdentifier)
	if err != nil {
		return fmt.Errorf("failed to load blsKeyring: %w", err)
	}

	return bls.Verify(am.baseSuite.(pairing.Suite), blsKeyring.PubPoly.Commit(), msg, fullSignature)
}
