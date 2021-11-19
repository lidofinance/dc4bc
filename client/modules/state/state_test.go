package state_test

import (
	"os"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/lidofinance/dc4bc/client/modules/state"

	"github.com/stretchr/testify/require"
)

func TestLevelDBState_SaveOffset(t *testing.T) {
	var (
		req    = require.New(t)
		dbPath = "/tmp/dc4bc_test_SaveOffset"
		topic  = "test_topic"
	)
	defer os.RemoveAll(dbPath)

	stg, err := state.NewLevelDBState(dbPath, topic)
	req.NoError(err)

	var offset uint64 = 1
	err = stg.SaveOffset(offset)
	req.NoError(err)

	loadedOffset, err := stg.LoadOffset()
	req.NoError(err)
	req.Equal(offset, loadedOffset)
}

func TestLevelDBState_NewStateFromOld(t *testing.T) {
	var (
		req    = require.New(t)
		dbPath = "/tmp/dc4bc_test_NewStateFromOld"
		topic  = "test_topic"
		re     = regexp.MustCompile(dbPath + `_(?P<ts>\d+)`)
	)
	defer os.RemoveAll(dbPath)

	st, err := state.NewLevelDBState(dbPath, topic)
	req.NoError(err)

	var offset uint64 = 1
	err = st.SaveOffset(offset)
	req.NoError(err)

	loadedOffset, err := st.LoadOffset()
	req.NoError(err)
	req.Equal(offset, loadedOffset)

	timeBefore := time.Now().Unix()
	path, err := st.Reset("")
	timeAfter := time.Now().Unix()

	req.NoError(err)
	submatches := re.FindStringSubmatch(path)
	req.Greater(len(submatches), 0)

	ts, err := strconv.Atoi(submatches[1])
	req.NoError(err)
	req.GreaterOrEqual(int64(ts), timeBefore)
	req.LessOrEqual(int64(ts), timeAfter)

	newLoadedOffset, err := st.LoadOffset()
	req.NoError(err)
	req.NotEqual(newLoadedOffset, loadedOffset)
}
