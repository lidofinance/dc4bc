package wc_rotation

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	_ "embed"

	"github.com/ethereum/go-ethereum/common"
	"github.com/lidofinance/dc4bc/pkg/wc_rotation/entity"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/signing"
)

type LidoConf struct {
	LidoBLSPubKey string
	LidoWC        string
}

type ConfPreset struct {
	TestPreset LidoConf
	ProdPreset LidoConf
}

const (
	DOMAIN_BLS_TO_EXECUTION_CHANGE = `0x0A000000`
	CAPELLA_FORK_VERSION           = `0x03000000`

	ProductionProfileName = "production"
)

var (
	confPreset = ConfPreset{
		TestPreset: LidoConf{
			LidoBLSPubKey: `0xb67aca71f04b673037b54009b760f1961f3836e5714141c892afdb75ec0834dce6784d9c72ed8ad7db328cff8fe9f13e`,
			LidoWC:        `0x010000000000000000000000b9d7934878b5fb9610b3fe8a5e441e8fad7e293f`,
		},
		// TODO: replace with prduction values
		ProdPreset: LidoConf{
			LidoBLSPubKey: `0xb67aca71f04b673037b54009b760f1961f3836e5714141c892afdb75ec0834dce6784d9c72ed8ad7db328cff8fe9f13e`,
			LidoWC:        `0x010000000000000000000000b9d7934878b5fb9610b3fe8a5e441e8fad7e293f`,
		},
	}

	capellaDomain      [4]byte
	capellaForkVersion = common.FromHex(CAPELLA_FORK_VERSION)
	lidoBlsPubKeyBB    [48]byte
	lidoWCBB           [32]byte

	onceDefaultClient sync.Once

	signingDomain    []byte
	computeDomainErr error

	//go:embed payloads.csv
	ValidatorsIndexesTest string

	// TODO: replace with production ready file
	//go:embed payloads.csv
	ValidatorsIndexesProd string

	// change this value with ldflags during the binary build
	// "production" - productions ready preset of the values and validators
	// "test" - testing preset of the values and validators
	Profile string = "test"
)

func GetValidatorsIndexes(start, end int) ([]uint64, error) {
	var strids []string
	if Profile == ProductionProfileName {
		strids = strings.Split(ValidatorsIndexesProd, "\n")
	} else {
		strids = strings.Split(ValidatorsIndexesTest, "\n")
	}

	if end > len(strids) {
		end = len(strids)
	}

	if start >= end {
		return nil, errors.New("invalid range, end should be greater than start")
	}

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
		var lidoBlsPubKeyBLS string
		var lidoWC string
		if Profile == ProductionProfileName {
			lidoBlsPubKeyBLS = confPreset.ProdPreset.LidoBLSPubKey
			lidoWC = confPreset.ProdPreset.LidoWC
		} else {
			lidoBlsPubKeyBLS = confPreset.TestPreset.LidoBLSPubKey
			lidoWC = confPreset.TestPreset.LidoWC
		}
		copy(capellaDomain[:], common.FromHex(DOMAIN_BLS_TO_EXECUTION_CHANGE))
		copy(lidoBlsPubKeyBB[:], common.FromHex(lidoBlsPubKeyBLS))
		copy(lidoWCBB[:], common.FromHex(lidoWC))

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
