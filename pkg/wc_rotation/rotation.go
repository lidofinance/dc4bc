package wc_rotation

import (
	_ "embed"
	"github.com/lidofinance/dc4bc/pkg/wc_rotation/entity"
)

var (
	// CapellaForkVersion 0x03000000
	//
	// https://github.com/ethereum/consensus-specs/blob/dev/specs/capella/fork.md#configuration
	CapellaForkVersion = [4]byte{3, 0, 0, 0}

	// GenesisForkVersion 0x00000000
	//
	// https://github.com/ethereum/consensus-specs/blob/5337da5dff85cd584c4330b46a881510c1218ca3/specs/phase0/beacon-chain.md#genesis-settings
	GenesisForkVersion = [4]byte{0, 0, 0, 0}

	// DomainBlsToExecutionChange 0x0A000000
	//
	// https://github.com/ethereum/consensus-specs/blob/dev/specs/capella/beacon-chain.md#domain-types
	DomainBlsToExecutionChange = [4]byte{10, 0, 0, 0}

	// GenesisValidatorRoot 0x4b363db94e286120d76eb905340fdd4e54bfe9f06bf33ff6cf5ad27f511bfe95
	// {beacon api}/eth/v1/beacon/genesis
	GenesisValidatorRoot = [32]byte{75, 54, 61, 185, 78, 40, 97, 32, 215, 110, 185, 5, 52, 15, 221, 78, 84, 191, 233, 240, 107, 243, 63, 246, 207, 90, 210, 127, 81, 27, 254, 149}

	// LidoBlsPubKeyBB
	// base64 tnrKcfBLZzA3tUAJt2Dxlh84NuVxQUHIkq/bdewINNzmeE2ccu2K19syjP+P6fE+
	// hex 0xb67aca71f04b673037b54009b760f1961f3836e5714141c892afdb75ec0834dce6784d9c72ed8ad7db328cff8fe9f13e
	//
	// https://blog.lido.fi/lido-withdrawal-key-ceremony/
	LidoBlsPubKeyBB = [48]byte{182, 122, 202, 113, 240, 75, 103, 48, 55, 181, 64, 9, 183, 96, 241, 150, 31, 56, 54, 229, 113, 65, 65, 200, 146, 175, 219, 117, 236, 8, 52, 220, 230, 120, 77, 156, 114, 237, 138, 215, 219, 50, 140, 255, 143, 233, 241, 62}

	// ToExecutionAddress 0xb9d7934878b5fb9610b3fe8a5e441e8fad7e293f
	//
	// https://mainnet.lido.fi/#/lido-dao/0x2e59a20f205bb85a89c53f1936454680651e618e/vote/78/
	ToExecutionAddress = [20]byte{185, 215, 147, 72, 120, 181, 251, 150, 16, 179, 254, 138, 94, 68, 30, 143, 173, 126, 41, 63}

	//go:embed payloads.csv
	ValidatorsIndexes string
)

func GetSigningRoot(validatorIndex uint64) ([32]byte, error) {
	domain, computeDomainErr := computeDomain(
		DomainBlsToExecutionChange,
		CapellaForkVersion,
		GenesisValidatorRoot,
	)

	if computeDomainErr != nil {
		return [32]byte{}, computeDomainErr
	}

	message := &entity.BLSToExecutionChange{
		ValidatorIndex:     validatorIndex,
		FromBlsPubkey:      LidoBlsPubKeyBB,
		ToExecutionAddress: ToExecutionAddress,
	}

	objRoot, err := message.HashTreeRoot()
	if err != nil {
		return [32]byte{}, err
	}

	return (&entity.SigningData{
		ObjectRoot: objRoot,
		Domain:     domain,
	}).HashTreeRoot()
}

// computeDomain returns the domain for the “domain_type“ and “fork_version“.
//
// Spec pseudocode definition:
// def compute_domain(domain_type: DomainType, fork_version: Version=None, genesis_validators_root: Root=None) -> Domain:
// if fork_version is None:
//
//	fork_version = GENESIS_FORK_VERSION
//
// if genesis_validators_root is None:
//
//	genesis_validators_root = Root()  # all bytes zero by default
//
// fork_data_root = compute_fork_data_root(fork_version, genesis_validators_root)
// return Domain(domain_type + fork_data_root[:28])
//
// https://github.com/ethereum/consensus-specs/blob/5337da5dff85cd584c4330b46a881510c1218ca3/specs/phase0/beacon-chain.md#compute_domain
func computeDomain(domainType [4]byte, forkVersion [4]byte, genesisValidatorsRoot [32]byte) ([32]byte, error) {
	if len(forkVersion[:]) == 0 {
		forkVersion = GenesisForkVersion
	}

	if len(genesisValidatorsRoot[:]) == 0 {
		genesisValidatorsRoot = [32]byte{}
	}

	forkDataRoot, err := computeForkDataRoot(forkVersion, genesisValidatorsRoot)
	if err != nil {
		return [32]byte{}, err
	}

	var domain [32]byte
	copy(domain[:], append(domainType[:], forkDataRoot[:28]...))

	return domain, nil
}

// computeForkDataRoot returns the 32byte fork data root for the “current_version“ and “genesis_validators_root“.
// This is used primarily in signature domains to avoid collisions across forks/chains.
//
// Spec pseudocode definition:
//
//		def compute_fork_data_root(current_version: Version, genesis_validators_root: Root) -> Root:
//	   return hash_tree_root(ForkData(
//	       current_version=current_version,
//	       genesis_validators_root=genesis_validators_root,
//	   ))
//
// https://github.com/ethereum/consensus-specs/blob/5337da5dff85cd584c4330b46a881510c1218ca3/specs/phase0/beacon-chain.md#compute_signing_root
func computeForkDataRoot(forkVersion [4]byte, genesisValidatorsRoot [32]byte) ([32]byte, error) {
	r, err := (&entity.ForkData{
		CurrentVersion:        forkVersion,
		GenesisValidatorsRoot: genesisValidatorsRoot,
	}).HashTreeRoot()
	if err != nil {
		return [32]byte{}, err
	}
	return r, nil
}
