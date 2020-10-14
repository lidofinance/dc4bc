package bls

import (
	"math/big"

	"github.com/corestario/kyber"
	"github.com/corestario/kyber/group/mod"
)

var curveOrder, _ = new(big.Int).SetString("73eda753299d7d483339d80809a1d80553bda402fffe5bfeffffffff00000001", 16)

func NewKyberScalar() kyber.Scalar {
	return mod.NewInt64(0, curveOrder)
}
