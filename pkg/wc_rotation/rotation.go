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
	// CapellaForkVersion 0x03000000
	CapellaForkVersion = [4]byte{3, 0, 0, 0}
	// DomainBlsToExecutionChange 0x0A000000
	DomainBlsToExecutionChange = [4]byte{10, 0, 0, 0}

	// GenesisValidatorRoot 0x4b363db94e286120d76eb905340fdd4e54bfe9f06bf33ff6cf5ad27f511bfe95
	// {beacon api}/eth/v1/beacon/genesis
	GenesisValidatorRoot = [32]byte{75, 54, 61, 185, 78, 40, 97, 32, 215, 110, 185, 5, 52, 15, 221, 78, 84, 191, 233, 240, 107, 243, 63, 246, 207, 90, 210, 127, 81, 27, 254, 149}

	// LidoBlsPubKeyBB 0xb67aca71f04b673037b54009b760f1961f3836e5714141c892afdb75ec0834dce6784d9c72ed8ad7db328cff8fe9f13e
	LidoBlsPubKeyBB = [48]byte{182, 122, 202, 113, 240, 75, 103, 48, 55, 181, 64, 9, 183, 96, 241, 150, 31, 56, 54, 229, 113, 65, 65, 200, 146, 175, 219, 117, 236, 8, 52, 220, 230, 120, 77, 156, 114, 237, 138, 215, 219, 50, 140, 255, 143, 233, 241, 62}
	// ToExecutionAddress 0xb9d7934878b5fb9610b3fe8a5e441e8fad7e293f
	ToExecutionAddress = [20]byte{185, 215, 147, 72, 120, 181, 251, 150, 16, 179, 254, 138, 94, 68, 30, 143, 173, 126, 41, 63}

	//go:embed payloads.csv
	ValidatorsIndexes string
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
