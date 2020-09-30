package dkg

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"

	"github.com/corestario/kyber"
	"github.com/corestario/kyber/share"
	dkg "github.com/corestario/kyber/share/dkg/pedersen"
	vss "github.com/corestario/kyber/share/vss/pedersen"
	"github.com/google/go-cmp/cmp"
	"lukechampine.com/frand"
)

// TODO: dump necessary data on disk
type DKG struct {
	sync.Mutex
	instance      *dkg.DistKeyGenerator
	deals         map[string]*dkg.Deal
	commits       map[string][]kyber.Point
	responses     *messageStore
	pubkeys       PKStore
	pubKey        kyber.Point
	secKey        kyber.Scalar
	suite         vss.Suite
	ParticipantID int

	N         int
	Threshold int
}

func Init(suite vss.Suite, pubKey kyber.Point, secKey kyber.Scalar) *DKG {
	var (
		d DKG
	)

	d.suite = suite
	d.secKey = secKey
	d.pubKey = pubKey

	d.deals = make(map[string]*dkg.Deal)
	d.commits = make(map[string][]kyber.Point)

	return &d
}

func (d *DKG) Equals(other *DKG) error {
	for addr, commits := range d.commits {
		otherCommits := other.commits[addr]
		for idx := range commits {
			if !commits[idx].Equal(otherCommits[idx]) {
				return fmt.Errorf("commits from %s are not equal (idx %d): %v != %v", addr, idx, commits[idx], otherCommits[idx])
			}
		}
	}

	for addr, deal := range d.deals {
		otherDeal := other.deals[addr]
		if !cmp.Equal(deal.Deal, otherDeal.Deal) {
			return fmt.Errorf("deals from %s are not equal: %+v != %+v", addr, deal.Deal, otherDeal.Deal)
		}
	}

	return nil
}

func (d *DKG) GetPubKey() kyber.Point {
	return d.pubKey
}

func (d *DKG) GetSecKey() kyber.Scalar {
	return d.secKey
}

func (d *DKG) GetPubKeyByParticipant(participant string) (kyber.Point, error) {
	pk, err := d.pubkeys.GetPKByParticipant(participant)
	if err != nil {
		return nil, fmt.Errorf("failed to get pk for participant %s: %w", participant, err)
	}
	return pk, nil
}

func (d *DKG) GetParticipantByIndex(index int) string {
	return d.pubkeys.GetParticipantByIndex(index)
}

func (d *DKG) GetPKByIndex(index int) kyber.Point {
	return d.pubkeys.GetPKByIndex(index)
}

func (d *DKG) StorePubKey(participant string, pid int, pk kyber.Point) bool {
	d.Lock()
	defer d.Unlock()

	return d.pubkeys.Add(&PK2Participant{
		Participant:   participant,
		PK:            pk,
		ParticipantID: pid,
	})
}

func (d *DKG) calcParticipantID() int {
	for idx, p := range d.pubkeys {
		if p.PK.Equal(d.pubKey) {
			return idx
		}
	}
	return -1
}

func (d *DKG) InitDKGInstance(seed []byte) (err error) {
	sort.Sort(d.pubkeys)

	publicKeys := d.pubkeys.GetPKs()

	participantsCount := len(publicKeys)

	participantID := d.calcParticipantID()

	if participantID < 0 {
		return fmt.Errorf("failed to determine participant index")
	}

	d.ParticipantID = participantID

	d.responses = newMessageStore(int(math.Pow(float64(participantsCount)-1, 2)))

	reader := frand.NewCustom(seed, 32, 20)

	d.instance, err = dkg.NewDistKeyGenerator(d.suite, d.secKey, publicKeys, d.Threshold, reader)
	if err != nil {
		return err
	}
	return nil
}

func (d *DKG) GetCommits() []kyber.Point {
	return d.instance.GetDealer().Commits()
}

func (d *DKG) StoreCommits(participant string, commits []kyber.Point) {
	d.Lock()
	defer d.Unlock()

	d.commits[participant] = commits
}

func (d *DKG) GetDeals() (map[int]*dkg.Deal, error) {
	deals, err := d.instance.Deals()
	if err != nil {
		return nil, err
	}
	return deals, nil
}

func (d *DKG) StoreDeal(participant string, deal *dkg.Deal) {
	d.Lock()
	defer d.Unlock()

	d.deals[participant] = deal
}

func (d *DKG) ProcessDeals() ([]*dkg.Response, error) {
	responses := make([]*dkg.Response, 0)
	for _, deal := range d.deals {
		if deal.Index == uint32(d.ParticipantID) {
			continue
		}
		resp, err := d.instance.ProcessDeal(deal)
		if err != nil {
			return nil, err
		}

		// Commits verification.
		allVerifiers := d.instance.Verifiers()
		verifier := allVerifiers[deal.Index]
		commitsOK, err := d.processDealCommits(verifier, deal)
		if err != nil {
			return nil, err
		}

		// If something goes wrong, party complains.
		if !resp.Response.Status || !commitsOK {
			return nil, fmt.Errorf("failed to process deals")
		}
		responses = append(responses, resp)
	}
	return responses, nil
}

func (d *DKG) StoreResponses(participant string, responses []*dkg.Response) {
	d.Lock()
	defer d.Unlock()

	for _, resp := range responses {
		d.responses.add(participant, int(resp.Response.Index), resp)
	}
}

func (d *DKG) ProcessResponses() error {
	for _, peerResponses := range d.responses.indexToData {
		for _, response := range peerResponses {
			resp := response.(*dkg.Response)
			if int(resp.Response.Index) == d.ParticipantID {
				continue
			}

			_, err := d.instance.ProcessResponse(resp)
			if err != nil {
				return fmt.Errorf("failed to ProcessResponse: %w", err)
			}
		}
	}

	if !d.instance.Certified() {
		return fmt.Errorf("praticipant %v is not certified", d.ParticipantID)
	}

	return nil
}

func (d *DKG) processDealCommits(verifier *vss.Verifier, deal *dkg.Deal) (bool, error) {
	decryptedDeal, err := verifier.DecryptDeal(deal.Deal)
	if err != nil {
		return false, err
	}

	participant := d.pubkeys.GetParticipantByIndex(int(deal.Index))

	commitsData, ok := d.commits[participant]

	if !ok {
		return false, err
	}
	var originalCommits []kyber.Point
	for _, commitData := range commitsData {
		commit, ok := commitData.(kyber.Point)
		if !ok {
			return false, fmt.Errorf("failed to cast commit data to commit type")
		}
		originalCommits = append(originalCommits, commit)
	}

	if len(originalCommits) != len(decryptedDeal.Commitments) {
		return false, errors.New("number of original commitments and number of commitments in the deal are not met")
	}

	for i := range originalCommits {
		if !originalCommits[i].Equal(decryptedDeal.Commitments[i]) {
			return false, errors.New("commits are different")
		}
	}

	return true, nil
}

func (d *DKG) GetDistKeyShare() (*dkg.DistKeyShare, error) {
	return d.instance.DistKeyShare()
}

func (d *DKG) GetDistributedPublicKey() (kyber.Point, error) {
	distKeyShare, err := d.instance.DistKeyShare()
	if err != nil {
		return nil, fmt.Errorf("failed to get distKeyShare")
	}
	return distKeyShare.Public(), nil
}

func (d *DKG) GetBLSKeyring() (*BLSKeyring, error) {
	if d.instance == nil || !d.instance.Certified() {
		return nil, fmt.Errorf("dkg instance is not ready")
	}

	distKeyShare, err := d.instance.DistKeyShare()
	if err != nil {
		return nil, fmt.Errorf("failed to get DistKeyShare: %v", err)
	}

	masterPubKey := share.NewPubPoly(d.suite, nil, distKeyShare.Commitments())

	return &BLSKeyring{
		PubPoly: masterPubKey,
		Share:   distKeyShare.PriShare(),
	}, nil
}
