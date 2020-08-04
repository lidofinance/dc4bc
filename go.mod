module github.com/depools/dc4bc

go 1.13

require (
	github.com/golang/mock v1.4.4
	github.com/google/uuid v1.1.1
	github.com/juju/fslock v0.0.0-20160525022230-4d5c94c67b4b
	github.com/looplab/fsm v0.1.0
	github.com/makiuchi-d/gozxing v0.0.0-20190830103442-eaff64b1ceb7
	github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e
	github.com/stretchr/testify v1.6.1
	github.com/syndtr/goleveldb v1.0.0
	go.dedis.ch/kyber/v3 v3.0.9
	gocv.io/x/gocv v0.23.0
	golang.org/x/image v0.0.0-20200618115811-c13761719519
	golang.org/x/text v0.3.3 // indirect
	golang.org/x/xerrors v0.0.0-20191204190536-9bdfabe68543 // indirect
)

replace (
	go.dedis.ch/kyber/v3 => github.com/corestario/kyber/v3 v3.0.0-20200218082721-8ed10c357c05
	golang.org/x/crypto => github.com/tendermint/crypto v0.0.0-20180820045704-3764759f34a5
)
