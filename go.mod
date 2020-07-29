module p2p.org/dc4bc

go 1.13

require (
	github.com/makiuchi-d/gozxing v0.0.0-20190830103442-eaff64b1ceb7
	github.com/mattn/go-gtk v0.0.0-20191030024613-af2e013261f5
	github.com/mattn/go-pointer v0.0.0-20190911064623-a0a44394634f // indirect
	github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e
	github.com/stretchr/testify v1.6.1 // indirect
	github.com/syndtr/goleveldb v1.0.0
	go.dedis.ch/kyber/v3 v3.0.9
	gocv.io/x/gocv v0.23.0
	golang.org/x/image v0.0.0-20200618115811-c13761719519
	golang.org/x/text v0.3.3 // indirect
	golang.org/x/xerrors v0.0.0-20191204190536-9bdfabe68543 // indirect
)

replace golang.org/x/crypto => github.com/tendermint/crypto v0.0.0-20180820045704-3764759f34a5

replace go.dedis.ch/kyber/v3 => github.com/corestario/kyber/v3 v3.0.0-20200218082721-8ed10c357c05
