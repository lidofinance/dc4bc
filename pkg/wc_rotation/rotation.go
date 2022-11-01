package wc_rotation

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/signing"

	_ "embed"

	"github.com/lidofinance/dc4bc/pkg/wc_rotation/entity"

	"github.com/ethereum/go-ethereum/common"
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
	ProductionProfileName = "production"

	// TestNet
	T_ValidatorID     = 393395
	T_ValidatorPubKey = `0xb645415eb26d640205954202df61074f03a3c23cada71a2fec21a42f55a2ea37673b32a812228fdd93281a9c2a9498b3`

	T_BLSPubKey = `0x8a1740906df44d772f76c6a4ce34203a76fac9f034832598ccf757a5cc45262b77c6d8071430fd8fd7e6fbc4ad9fec0d`
	//  8a1740906df44d772f76c6a4ce34203a76fac9f034832598ccf757a5cc45262b77c6d8071430fd8fd7e6fbc4ad9fec0d
	T_EXECUTION_ADDRESS_GOERLY = `0x01000000000000000000000017E4C873e6EE44381e79aE0b1A3BDc0eF540a9e0`
)

var (
	// `0x03000000`
	CAPELLA_FORK_VERSION = [4]byte{3, 0, 0, 0}
	// `0x0A000000`
	DOMAIN_BLS_TO_EXECUTION_CHANGE = [4]byte{10, 0, 0, 0}
	TESTNET_GENESIS_FORK_VERSION   = [4]byte{0, 0, 16, 32}

	confPreset = ConfPreset{
		TestPreset: LidoConf{
			LidoBLSPubKey: T_BLSPubKey,
			LidoWC:        T_EXECUTION_ADDRESS_GOERLY,
		},
		// TODO: replace with prduction values
		ProdPreset: LidoConf{
			LidoBLSPubKey: `0xb67aca71f04b673037b54009b760f1961f3836e5714141c892afdb75ec0834dce6784d9c72ed8ad7db328cff8fe9f13e`,
			LidoWC:        `0x010000000000000000000000b9d7934878b5fb9610b3fe8a5e441e8fad7e293f`,
		},
	}

	forkVersion        [4]byte
	lidoBlsPubKeyBB    [48]byte
	ToExecutionAddress [32]byte

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
			forkVersion = CAPELLA_FORK_VERSION
		} else {
			lidoBlsPubKeyBLS = confPreset.TestPreset.LidoBLSPubKey
			lidoWC = confPreset.TestPreset.LidoWC
			forkVersion = TESTNET_GENESIS_FORK_VERSION
		}
		copy(lidoBlsPubKeyBB[:], common.FromHex(lidoBlsPubKeyBLS))
		copy(ToExecutionAddress[:], common.FromHex(lidoWC))

		signingDomain, computeDomainErr = signing.ComputeDomain(
			DOMAIN_BLS_TO_EXECUTION_CHANGE,
			forkVersion[:],
			make([]byte, 32),
		)
	})

	if computeDomainErr != nil {
		return [32]byte{}, computeDomainErr
	}

	message := &entity.BLSToExecutionChange{
		ValidatorIndex:     validatorIndex,
		FromBlsPubkey:      lidoBlsPubKeyBB,
		ToExecutionAddress: ToExecutionAddress,
	}

	return signing.ComputeSigningRoot(message, signingDomain)
}
