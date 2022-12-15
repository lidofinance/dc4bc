package entity

// SigningData
// https://github.com/ethereum/consensus-specs/blob/5337da5dff85cd584c4330b46a881510c1218ca3/specs/phase0/beacon-chain.md#signingdata
type SigningData struct {
	ObjectRoot [32]byte `json:"object_root" ssz-size:"32"`
	Domain     [32]byte `json:"domain" ssz-size:"32"`
}
