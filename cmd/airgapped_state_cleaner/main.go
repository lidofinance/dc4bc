package main

import (
	"encoding/json"
	"fmt"
	"github.com/lidofinance/dc4bc/airgapped"
	client "github.com/lidofinance/dc4bc/client/types"
	"github.com/syndtr/goleveldb/leveldb"
	"os"
)

const operationsLogDBKey = "operations_log"

func main() {
	stateDir := os.Args[1]
	dkgRoundID := os.Args[2]

	db, err := leveldb.OpenFile(stateDir, nil)
	if err != nil {
		fmt.Printf("failed to open db file %s for keys: %v\n", stateDir, err)
		return
	}
	defer db.Close()

	roundOperationsLog, err := getRoundOperationLog(db)
	if err != nil {
		fmt.Printf("failed to getRoundOperationLog: %v\n", err)
		return
	}

	roundOperations := roundOperationsLog[dkgRoundID]
	var cleanRoundOperations []client.Operation

	knownOperations := map[string]bool{}
	for _, operation := range roundOperations {
		if knownOperations[operation.ID] {
			continue
		}

		cleanRoundOperations = append(cleanRoundOperations, operation)
		knownOperations[operation.ID] = true
	}

	roundOperationsLog[dkgRoundID] = cleanRoundOperations
	roundOperationsLogBz, err := json.Marshal(roundOperationsLog)
	if err != nil {
		fmt.Printf("failed to marshal operationsLog: %v\n", err)
		return
	}

	if err := db.Put([]byte(operationsLogDBKey), roundOperationsLogBz, nil); err != nil {
		fmt.Printf("failed to put updated operationsLog: %v\n", err)
	}
}

func getRoundOperationLog(db *leveldb.DB) (airgapped.RoundOperationLog, error) {
	operationsLogBz, err := db.Get([]byte(operationsLogDBKey), nil)
	if err != nil {
		return nil, err
	}

	var roundOperationsLog airgapped.RoundOperationLog
	if err := json.Unmarshal(operationsLogBz, &roundOperationsLog); err != nil {
		return nil, fmt.Errorf("failed to unmarshal stored operationsLog: %w", err)
	}

	return roundOperationsLog, nil
}