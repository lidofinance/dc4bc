package bls

import (
	"crypto/cipher"
	"encoding/hex"
	"io"

	"github.com/corestario/kyber"
	bls12381 "github.com/kilic/bls12-381"
)

type KyberGT struct {
	f *bls12381.E
}

func newEmptyGT() *KyberGT {
	return newKyberGT(bls12381.NewGT().New())
}
func newKyberGT(f *bls12381.E) *KyberGT {
	return &KyberGT{
		f: f,
	}
}

func (k *KyberGT) Equal(kk kyber.Point) bool {
	return k.f.Equal(kk.(*KyberGT).f)
}

const gtLength = 576

func (k *KyberGT) Null() kyber.Point {
	var zero [gtLength]byte
	k.f, _ = bls12381.NewGT().FromBytes(zero[:])
	return k
}

func (k *KyberGT) Base() kyber.Point {
	panic("not yet available")
	/*var baseReader, _ = blake2b.NewXOF(0, []byte("Quand il y a à manger pour huit, il y en a bien pour dix."))*/
	//_, err := NewGT().rand(baseReader)
	//if err != nil {
	//panic(err)
	//}
	/*return k*/
}

func (k *KyberGT) Pick(rand cipher.Stream) kyber.Point {
	panic("TODO: bls12-381.GT.Pick()")
}

func (k *KyberGT) Set(q kyber.Point) kyber.Point {
	k.f.Set(q.(*KyberGT).f)
	return k
}

func (k *KyberGT) Clone() kyber.Point {
	kk := newEmptyGT()
	kk.Set(k)
	return kk
}

func (k *KyberGT) Add(a, b kyber.Point) kyber.Point {
	aa := a.(*KyberGT)
	bb := b.(*KyberGT)
	bls12381.NewGT().Add(k.f, aa.f, bb.f)
	return k
}

func (k *KyberGT) Sub(a, b kyber.Point) kyber.Point {
	aa := a.(*KyberGT)
	bb := b.(*KyberGT)
	bls12381.NewGT().Sub(k.f, aa.f, bb.f)
	return k
}

func (k *KyberGT) Neg(q kyber.Point) kyber.Point {
	panic("bls12-381: GT is not a full kyber.Point implementation")
}

func (k *KyberGT) Mul(s kyber.Scalar, q kyber.Point) kyber.Point {
	panic("bls12-381: GT is not a full kyber.Point implementation")
}

func (k *KyberGT) MarshalBinary() ([]byte, error) {
	return bls12381.NewGT().ToBytes(k.f), nil
}

func (k *KyberGT) MarshalTo(w io.Writer) (int, error) {
	buf, err := k.MarshalBinary()
	if err != nil {
		return 0, err
	}
	return w.Write(buf)
}

func (k *KyberGT) UnmarshalBinary(buf []byte) error {
	fe12, err := bls12381.NewGT().FromBytes(buf)
	k.f = fe12
	return err
}

func (k *KyberGT) UnmarshalFrom(r io.Reader) (int, error) {
	buf := make([]byte, k.MarshalSize())
	n, err := io.ReadFull(r, buf)
	if err != nil {
		return n, err
	}
	return n, k.UnmarshalBinary(buf)
}

func (k *KyberGT) MarshalSize() int {
	return 576
}

func (k *KyberGT) String() string {
	b, _ := k.MarshalBinary()
	return "bls12-381.GT: " + hex.EncodeToString(b)
}

func (k *KyberGT) EmbedLen() int {
	panic("bls12-381.GT.EmbedLen(): unsupported operation")
}

func (k *KyberGT) Embed(data []byte, rand cipher.Stream) kyber.Point {
	panic("bls12-381.GT.Embed(): unsupported operation")
}

func (k *KyberGT) Data() ([]byte, error) {
	panic("bls12-381.GT.Data(): unsupported operation")
}
