package wc_rotation

import (
	_ "embed"
	"errors"
	"fmt"
	"github.com/lidofinance/dc4bc/pkg/wc_rotation/entity"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/signing"
	"strconv"
)

var (
	// CapellaForkVersion 0x20000092
	CapellaForkVersion = [4]byte{32, 0, 0, 146}

	// DomainBlsToExecutionChange 0x0A000000
	DomainBlsToExecutionChange = [4]byte{10, 0, 0, 0}

	// GenesisValidatorRoot 0x34b7b2d25fa9fd1fc079cad9a4840f94f29ea49918cc997d73aa7a196d37cbbe
	// {beacon api}/eth/v1/beacon/genesis
	GenesisValidatorRoot = [32]byte{52, 183, 178, 210, 95, 169, 253, 31, 192, 121, 202, 217, 164, 132, 15, 148, 242, 158, 164, 153, 24, 204, 153, 125, 115, 170, 122, 25, 109, 55, 203, 190}

	// LidoBlsPubKeyBB 0x8199b7a8c6998aafb30a955794f5d72a454ed1caf51bdbfc3065973153f64eeb64ff07a5b43cb9007cba3e3ec76ed756
	LidoBlsPubKeyBB = [48]byte{129, 153, 183, 168, 198, 153, 138, 175, 179, 10, 149, 87, 148, 245, 215, 42, 69, 78, 209, 202, 245, 27, 219, 252, 48, 101, 151, 49, 83, 246, 78, 235, 100, 255, 7, 165, 180, 60, 185, 0, 124, 186, 62, 62, 199, 110, 215, 86}

	// ToExecutionAddress 0xb9d7934878b5fb9610b3fe8a5e441e8fad7e293f
	ToExecutionAddress = [20]byte{185, 215, 147, 72, 120, 181, 251, 150, 16, 179, 254, 138, 94, 68, 30, 143, 173, 126, 41, 63}

	//go:embed payloads.csv
	ValidatorsIndexesTest string
)

func GetValidatorsIndexes(start, end int) ([]uint64, error) {
	var strids []string

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
	signingDomain, computeDomainErr := signing.ComputeDomain(
		DomainBlsToExecutionChange,
		CapellaForkVersion[:],
		GenesisValidatorRoot[:],
	)

	if computeDomainErr != nil {
		return [32]byte{}, computeDomainErr
	}

	message := &entity.BLSToExecutionChange{
		ValidatorIndex:     validatorIndex,
		FromBlsPubkey:      LidoBlsPubKeyBB,
		ToExecutionAddress: ToExecutionAddress,
	}

	return signing.ComputeSigningRoot(message, signingDomain)
}
