module github.com/depools/dc4bc

go 1.13

require (
	github.com/corestario/kyber v1.6.0
	github.com/golang/mock v1.4.4
	github.com/google/go-cmp v0.5.0
	github.com/google/uuid v1.1.1
	github.com/juju/fslock v0.0.0-20160525022230-4d5c94c67b4b
	github.com/looplab/fsm v0.1.0
	github.com/makiuchi-d/gozxing v0.0.0-20190830103442-eaff64b1ceb7
	github.com/prysmaticlabs/prysm v1.0.0-alpha.29.0.20201014075528-022b6667e5d0
	github.com/segmentio/kafka-go v0.4.2
	github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e
	github.com/spf13/cobra v1.0.0
	github.com/stretchr/testify v1.6.1
	github.com/syndtr/goleveldb v1.0.1-0.20200815110645-5c35d600f0ca
	gocv.io/x/gocv v0.24.0
	golang.org/x/crypto v0.0.0-20200820211705-5c72a883971a
	lukechampine.com/frand v1.3.0
)

replace golang.org/x/crypto => github.com/tendermint/crypto v0.0.0-20180820045704-3764759f34a5

replace github.com/ethereum/go-ethereum => github.com/ethereum/go-ethereum v1.9.22
