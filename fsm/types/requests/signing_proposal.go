package requests

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lidofinance/dc4bc/pkg/wc_rotation"
)

// MessageToSign is a message to sign on airgapped machine.
// It can either contains a contant as Payload or a range of baked into airgapped messages
type MessageToSign struct {
	MessageID string
	File      string
	Payload   []byte
}

type SigningTask struct {
	MessageID  string
	File       string
	Payload    []byte
	RangeStart int
	RangeEnd   int
}

// States: "stage_signing_idle"
// Events: "event_signing_start_batch"
type SigningBatchProposalStartRequest struct {
	BatchID       string
	ParticipantId int
	CreatedAt     time.Time
	SigningTasks  []SigningTask
}

type PartialSign struct {
	MessageID string
	Sign      []byte
}

// States: "state_signing_await_partial_signs"
// Events: "event_signing_partial_sign_received"
type SigningProposalBatchPartialSignRequests struct {
	BatchID       string
	ParticipantId int
	PartialSigns  []PartialSign
	CreatedAt     time.Time
}

func TasksToMessages(msgs []SigningTask) ([]MessageToSign, error) {
	var signData []MessageToSign
	for _, m := range msgs {
		if m.Payload != nil {
			signData = append(signData, MessageToSign{
				File:      m.File,
				MessageID: m.MessageID,
				Payload:   m.Payload,
			})
		} else {
			for i := m.RangeStart; i <= m.RangeEnd; i++ {
				data, err := ReconstructBakedMessage(i)
				if err != nil {
					return nil, fmt.Errorf("failed to ReconstructBakedMessage: %w", err)
				}
				signData = append(signData, data)
			}
		}
	}
	return signData, nil
}

func ReconstructBakedMessage(id int) (MessageToSign, error) {
	validatorsIDS := strings.Split(wc_rotation.ValidatorsIndexesTest, "\n")
	vID, err := strconv.Atoi(validatorsIDS[id])
	if err != nil {
		return MessageToSign{}, fmt.Errorf("failed to parse int from str(%s): %w", validatorsIDS[id], err)
	}
	root, err := wc_rotation.GetSigningRoot(uint64(vID))
	if err != nil {
		return MessageToSign{}, fmt.Errorf("failed to get signed root: %w", err)
	}
	messageID := fmt.Sprintf("bakedrange%d", id)
	return MessageToSign{
		File:      messageID,
		MessageID: messageID,
		Payload:   root[:],
	}, nil
}
