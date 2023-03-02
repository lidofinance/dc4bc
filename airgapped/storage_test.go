package airgapped

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	client "github.com/lidofinance/dc4bc/client/types"
)

func TestMachine_DropOperationsLog(t *testing.T) {
	testDir := "/tmp/dc4bc_test_drop_log"
	dkgIdentifier := "aaa"

	am, err := NewMachine(testDir)
	if err != nil {
		t.Fatalf("failed to create airgapped machine: %v", err)
	}

	err = am.storeOperation(client.Operation{DKGIdentifier: dkgIdentifier, ID: "id_1"})
	require.NoError(t, err)
	err = am.storeOperation(client.Operation{DKGIdentifier: dkgIdentifier, ID: "id_2"})
	require.NoError(t, err)

	ops, err := am.getOperationsLog(dkgIdentifier)
	require.NoError(t, err)
	require.Len(t, ops, 2)

	err = am.dropRoundOperationLog(dkgIdentifier)
	require.NoError(t, err)

	ops, err = am.getOperationsLog(dkgIdentifier)
	require.NoError(t, err)
	require.Len(t, ops, 0)

	defer os.RemoveAll(fmt.Sprintf("%s/%s-drop_log", testDir, testDB))
}
