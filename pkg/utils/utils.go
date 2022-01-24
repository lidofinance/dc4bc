package utils

import (
	"fmt"
	"github.com/lidofinance/dc4bc/dkg"
	fsmtypes "github.com/lidofinance/dc4bc/fsm/types"
)

func PrepareSignaturesToDump(orig map[string][]fsmtypes.ReconstructedSignature) (*dkg.ExportedSignatures, error) {
	var output = make(dkg.ExportedSignatures)
	for messageID, entries := range orig {
		if len(entries) == 0 {
			return nil, fmt.Errorf("no reconstructed signatures found for message %s", messageID)
		}
		output[messageID] = dkg.ExportedSignatureEntity{
			Payload:   entries[0].SrcPayload,
			Signature: entries[0].Signature,
			File:      entries[0].File,
		}
	}
	return &output, nil
}
