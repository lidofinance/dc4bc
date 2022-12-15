package entity

// ForkData
// https://github.com/ethereum/consensus-specs/blob/5337da5dff85cd584c4330b46a881510c1218ca3/specs/phase0/beacon-chain.md#forkdata
type ForkData struct {
	CurrentVersion        [4]byte  `json:"current_version" ssz-size:"4"`
	GenesisValidatorsRoot [32]byte `json:"genesis_validators_root" ssz-size:"32"`
}
