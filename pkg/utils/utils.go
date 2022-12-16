package utils

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/lidofinance/dc4bc/storage"
	"os"

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

func ReadLogMessages(inputFilePath string, separator rune, skipHeader bool, messageColumnIndex int) ([]storage.Message, error) {
	inputFile, err := os.Open(inputFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to Open input file: %w", err)
	}
	defer inputFile.Close()

	reader := csv.NewReader(inputFile)
	reader.Comma = separator
	reader.LazyQuotes = true

	lines, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read dump CSV): %w", err)
	}

	if skipHeader {
		lines = lines[1:]
	}

	var message storage.Message
	var messages []storage.Message
	for _, line := range lines {
		if err := json.Unmarshal([]byte(line[messageColumnIndex]), &message); err != nil {
			return nil, fmt.Errorf("failed to unmarshal line `%s`: %w", line[messageColumnIndex], err)
		}

		messages = append(messages, message)
	}

	return messages, nil
}
