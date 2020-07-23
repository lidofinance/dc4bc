module dkg

go 1.14

require (
	github.com/corestario/cosmos-utils/client v0.1.0
	github.com/corestario/dkglib v1.0.4
	github.com/cosmos/cosmos-sdk v0.28.2-0.20190827131926-5aacf454e1b6
	github.com/tendermint/tendermint v0.32.8
)

replace golang.org/x/crypto => github.com/tendermint/crypto v0.0.0-20180820045704-3764759f34a5

replace github.com/tendermint/tendermint => github.com/corestario/tendermint v0.5.0

replace github.com/cosmos/cosmos-sdk => github.com/corestario/cosmos-sdk v0.3.0

replace go.dedis.ch/kyber/v3 => github.com/corestario/kyber/v3 v3.0.0-20200218082721-8ed10c357c05
