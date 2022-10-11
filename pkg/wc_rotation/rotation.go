package wc_rotation

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	_ "embed"

	"github.com/ethereum/go-ethereum/common"
	"github.com/lidofinance/dc4bc/pkg/wc_rotation/entity"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/signing"
)

const (
	DOMAIN_BLS_TO_EXECUTION_CHANGE = `0x0A000000`
	CAPELLA_FORK_VERSION           = `0x03000000`

	LIDO_BLS_PUB_KEY = `0xb67aca71f04b673037b54009b760f1961f3836e5714141c892afdb75ec0834dce6784d9c72ed8ad7db328cff8fe9f13e`
	LIDO_WC          = `0x010000000000000000000000b9d7934878b5fb9610b3fe8a5e441e8fad7e293f`
)

var (
	capellaDomain      [4]byte
	capellaForkVersion = common.FromHex(CAPELLA_FORK_VERSION)
	lidoBlsPubKeyBB    [48]byte
	lidoWCBB           [32]byte

	onceDefaultClient sync.Once

	signingDomain    []byte
	computeDomainErr error

	//go:embed payloads.csv
	Payloads string
)

func GetValidatorsIndexes(start, end int) ([]uint64, error) {
	strids := strings.Split(Payloads, "\n")
	ids := make([]uint64, 0, end-start)
	for _, strid := range strids[start:end] {
		id, err := strconv.Atoi(strid)
		if err != nil {
			return nil, fmt.Errorf("failed to parse id into int: %w", err)
		}
		ids = append(ids, uint64(id))
	}
	return ids, nil
}

func GetSigningRoot(validatorIndex uint64) ([32]byte, error) {
	onceDefaultClient.Do(func() {
		copy(capellaDomain[:], common.FromHex(DOMAIN_BLS_TO_EXECUTION_CHANGE))
		copy(lidoBlsPubKeyBB[:], common.FromHex(LIDO_BLS_PUB_KEY))
		copy(lidoWCBB[:], common.FromHex(LIDO_WC))

		signingDomain, computeDomainErr = signing.ComputeDomain(
			capellaDomain,
			capellaForkVersion,
			make([]byte, 32),
		)
	})

	if computeDomainErr != nil {
		return [32]byte{}, computeDomainErr
	}

	message := &entity.BLSToExecutionChange{
		ValidatorIndex:        validatorIndex,
		Pubkey:                lidoBlsPubKeyBB,
		WithdrawalCredentials: lidoWCBB,
	}

	return signing.ComputeSigningRoot(message, signingDomain)
}
