package dkg

import (
	"go.dedis.ch/kyber/v3"
)

type PK2Participant struct {
	Participant string
	PK          kyber.Point
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
func (s PKStore) Less(i, j int) bool { return s[i].Participant < s[j].Participant }
func (s PKStore) GetPKs() []kyber.Point {
	var out = make([]kyber.Point, len(s))
	for idx, val := range s {
		out[idx] = val.PK
	}
	return out
}
