package dkg

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"

	"github.com/corestario/kyber/pairing"

	"github.com/corestario/kyber"
	"github.com/corestario/kyber/share"
	bls12381 "github.com/depools/kyber-bls12381"
)

type PK2Participant struct {
	ParticipantID int
	Participant   string
	PK            kyber.Point
}

type PKStore []*PK2Participant

func (s *PKStore) Add(newPk *PK2Participant) bool {
	for _, pk := range *s {
		if pk.Participant == newPk.Participant && pk.PK.Equal(newPk.PK) {
			return false
		}
	}
	*s = append(*s, newPk)

	return true
}

func (s PKStore) Len() int           { return len(s) }
func (s PKStore) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s PKStore) Less(i, j int) bool { return s[i].ParticipantID < s[j].ParticipantID }
func (s PKStore) GetPKs() []kyber.Point {
	var out = make([]kyber.Point, len(s))
	for idx, val := range s {
		out[idx] = val.PK
	}
	return out
}

func (s PKStore) GetPKByParticipant(p string) (kyber.Point, error) {
	for _, val := range s {
		if val.Participant == p {
			return val.PK, nil
		}
	}
	return nil, fmt.Errorf("participant %s does not exist", p)
}

func (s PKStore) GetPKByIndex(index int) kyber.Point {
	if index < 0 || index > len(s) {
		return nil
	}
	return s[index].PK
}

func (s PKStore) GetParticipantByIndex(index int) string {
	if index < 0 || index > len(s) {
		return ""
	}
	return s[index].Participant
}

type messageStore struct {
	// Common number of messages of the same type from peers
	messagesCount int

	// Max number of messages of the same type from one peer per round
	maxMessagesFromPeer int

	// Map which stores messages. Key is a peer's address, value is data
	addrToData map[string][]interface{}

	// Map which stores messages (same as addrToData). Key is a peer's index, value is data.
	indexToData map[int][]interface{}
}

func newMessageStore(n int) *messageStore {
	return &messageStore{
		maxMessagesFromPeer: n,
		addrToData:          make(map[string][]interface{}),
		indexToData:         make(map[int][]interface{}),
	}
}

func (ms *messageStore) add(addr string, index int, val interface{}) {
	data := ms.addrToData[addr]
	if len(data) == ms.maxMessagesFromPeer {
		return
	}
	data = append(data, val)
	ms.addrToData[addr] = data

	data = ms.indexToData[index]
	data = append(data, val)
	ms.indexToData[index] = data

	ms.messagesCount++
}

type BLSKeyring struct {
	PubPoly *share.PubPoly
	Share   *share.PriShare
}

type blsKeyringJSON struct {
	Commitments [][]byte `json:"commitments"`
	Share       []byte   `json:"share"`
}

func (b *BLSKeyring) Bytes() ([]byte, error) {
	var shareBuf bytes.Buffer
	shareEnc := gob.NewEncoder(&shareBuf)
	if err := shareEnc.Encode(b.Share); err != nil {
		return nil, fmt.Errorf("failed to encode private key: %v", err)
	}

	_, commitments := b.PubPoly.Info()
	commitmentsBz := make([][]byte, 0, len(commitments))
	for _, commitment := range commitments {
		data, err := commitment.MarshalBinary()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal commitment: %w", err)
		}
		commitmentsBz = append(commitmentsBz, data)
	}
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(commitmentsBz); err != nil {
		return nil, fmt.Errorf("failed to encode commitmentBz: %w", err)
	}

	blsKeyringJSON := blsKeyringJSON{
		Commitments: commitmentsBz,
		Share:       shareBuf.Bytes(),
	}

	return json.Marshal(blsKeyringJSON)
}

func LoadBLSKeyringFromBytes(data []byte) (*BLSKeyring, error) {
	var (
		err            error
		blsKeyringJson blsKeyringJSON
	)
	if err = json.Unmarshal(data, &blsKeyringJson); err != nil {
		return nil, fmt.Errorf("failed to unmarshal blsKeyringJson: %w", err)
	}

	commitments := make([]kyber.Point, 0, len(blsKeyringJson.Commitments))
	for _, commitmentBz := range blsKeyringJson.Commitments {
		commitment := bls12381.NewBLS12381Suite().Point()
		if err := commitment.UnmarshalBinary(commitmentBz); err != nil {
			return nil, fmt.Errorf("failed to unmarshal commitment: %w", err)
		}
		commitments = append(commitments, commitment)
	}

	priShare, privDec := &share.PriShare{V: bls12381.NewBLS12381Suite().(pairing.Suite).G1().Scalar()}, gob.NewDecoder(bytes.NewBuffer(blsKeyringJson.Share))
	if err := privDec.Decode(priShare); err != nil {
		return nil, fmt.Errorf("failed to share: %v", err)
	}

	return &BLSKeyring{
		PubPoly: share.NewPubPoly(bls12381.NewBLS12381Suite(), nil, commitments),
		Share:   priShare,
	}, nil
}
