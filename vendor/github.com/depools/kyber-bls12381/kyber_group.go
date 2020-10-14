package bls

import (
	"crypto/cipher"
	"crypto/sha256"
	"hash"
	"io"
	"reflect"

	vss "github.com/corestario/kyber/share/vss/pedersen"
	"lukechampine.com/frand"

	"github.com/corestario/kyber"
	"github.com/corestario/kyber/util/random"
	"github.com/corestario/kyber/xof/blake2xb"
	bls12381 "github.com/kilic/bls12-381"
)

// GroupChecker allows to verify if a Point is in the correct group or not. For
// curves which don't have a prime order, we need to only consider the points
// lying in the subgroup of prime order. That check returns true if the point is
// correct or not.
type GroupChecker interface {
	kyber.Point
	IsInCorrectGroup() bool
}

type groupBls struct {
	str      string
	newPoint func() kyber.Point
	isPrime  bool
}

func (g *groupBls) String() string {
	return g.str
}

func (g *groupBls) Scalar() kyber.Scalar {
	return NewKyberScalar()
}

func (g *groupBls) ScalarLen() int {
	return g.Scalar().MarshalSize()
}

func (g *groupBls) PointLen() int {
	return g.Point().MarshalSize()
}

func (g *groupBls) Point() kyber.Point {
	return g.newPoint()
}

func (g *groupBls) IsPrimeOrder() bool {
	return g.isPrime
}

func (g *groupBls) Hash() hash.Hash {
	return sha256.New()
}

// XOF returns a newlly instantiated blake2xb XOF function.
func (g *groupBls) XOF(seed []byte) kyber.XOF {
	return blake2xb.New(seed)
}

// RandomStream returns a cipher.Stream which corresponds to a key stream from
// crypto/rand.
func (g *groupBls) RandomStream() cipher.Stream {
	return random.New()
}

func NewGroupG1() kyber.Group {
	return &groupBls{
		str:      "bls12-381.G1",
		newPoint: func() kyber.Point { return NullKyberG1() },
		isPrime:  true,
	}
}

func NewGroupG2() kyber.Group {
	return &groupBls{
		str:      "bls12-381.G2",
		newPoint: func() kyber.Point { return NullKyberG2() },
		isPrime:  false,
	}
}

func NewGroupGT() kyber.Group {
	return &groupBls{
		str:      "bls12-381.GT",
		newPoint: func() kyber.Point { return newEmptyGT() },
		isPrime:  false,
	}
}

type Suite struct {
	e      *bls12381.Engine
	stream cipher.Stream
}

func (s *Suite) String() string {
	return "Suite.String()"
}

func (s *Suite) ScalarLen() int {
	return s.G1().Scalar().MarshalSize()
}

func (s *Suite) Scalar() kyber.Scalar {
	return s.G1().Scalar()
}

func (s *Suite) PointLen() int {
	return s.G1().Point().MarshalSize()
}

func (s *Suite) Point() kyber.Point {
	return s.G1().Point()
}

func NewBLS12381Suite(seed []byte) vss.Suite {
	var stream cipher.Stream
	if len(seed) > 0 {
		stream = random.New(
			frand.NewCustom(seed, 32, 20),
		)
	} else {
		stream = random.New()
	}

	return &Suite{e: bls12381.NewEngine(), stream: stream}
}

func (s *Suite) G1() kyber.Group {
	return NewGroupG1()
}

func (s *Suite) G2() kyber.Group {
	return NewGroupG2()
}

func (s *Suite) GT() kyber.Group {
	return NewGroupGT()
}

// ValidatePairing implements the `pairing.Suite` interface
func (s *Suite) ValidatePairing(p1, p2, p3, p4 kyber.Point) bool {
	s.e.AddPair(p1.(*KyberG1).p, p2.(*KyberG2).p)
	s.e.AddPairInv(p3.(*KyberG1).p, p4.(*KyberG2).p)
	return s.e.Check()
}

func (s *Suite) Pair(p1, p2 kyber.Point) kyber.Point {
	g1point := p1.(*KyberG1).p
	g2point := p2.(*KyberG2).p
	gt := newKyberGT(s.e.AddPair(g1point, g2point).Result())
	return gt
}

// New implements the kyber.Encoding interface.
func (s *Suite) New(t reflect.Type) interface{} {
	panic("Suite.Encoding: deprecated in drand")
}

// Read is the default implementation of kyber.Encoding interface Read.
func (s *Suite) Read(r io.Reader, objs ...interface{}) error {
	panic("Suite.Read(): deprecated in drand")
}

// Write is the default implementation of kyber.Encoding interface Write.
func (s *Suite) Write(w io.Writer, objs ...interface{}) error {
	panic("Suite.Write(): deprecated in drand")
}

// Hash returns a newly instantiated sha256 hash function.
func (s *Suite) Hash() hash.Hash {
	return sha256.New()
}

// XOF returns a newlly instantiated blake2xb XOF function.
func (s *Suite) XOF(seed []byte) kyber.XOF {
	return blake2xb.New(seed)
}

// RandomStream returns a cipher.Stream which corresponds to a key stream from
// crypto/rand.
func (s *Suite) RandomStream() cipher.Stream {
	return s.stream
}
