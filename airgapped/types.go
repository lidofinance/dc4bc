package airgapped

import (
	"fmt"
	"github.com/depools/dc4bc/dkg"
	"github.com/syndtr/goleveldb/leveldb/util"
	"strings"
)

const (
	blsKeyringPrefix = "bls_keyring"
)

func makeBLSKeyKeyringDBKey(key string) string {
	return fmt.Sprintf("%s_%s", blsKeyringPrefix, key)
}

func (am *AirgappedMachine) saveBLSKeyring(dkgID string, blsKeyring *dkg.BLSKeyring) error {
	blsKeyringBz, err := blsKeyring.Bytes()
	if err != nil {
		return fmt.Errorf("failed to encode bls keyring: %w", err)
	}
	if err := am.db.Put([]byte(makeBLSKeyKeyringDBKey(dkgID)), blsKeyringBz, nil); err != nil {
		return fmt.Errorf("failed to save BLSKeyring into db: %w", err)
	}
	return nil
}

func (am *AirgappedMachine) loadBLSKeyring(dkgID string) (*dkg.BLSKeyring, error) {
	var (
		blsKeyring   *dkg.BLSKeyring
		blsKeyringBz []byte
		err          error
	)

	if blsKeyringBz, err = am.db.Get([]byte(makeBLSKeyKeyringDBKey(dkgID)), nil); err != nil {
		return nil, fmt.Errorf("failed to get bls keyring with dkg id %s: %w", dkgID, err)
	}
	if blsKeyring, err = dkg.LoadBLSKeyringFromBytes(blsKeyringBz); err != nil {
		return nil, fmt.Errorf("failed to decode bls keyring")
	}
	return blsKeyring, nil
}

func (am *AirgappedMachine) GetBLSKeyrings() (map[string]*dkg.BLSKeyring, error) {
	var (
		blsKeyring *dkg.BLSKeyring
		err        error
	)

	keyrings := make(map[string]*dkg.BLSKeyring)
	iter := am.db.NewIterator(util.BytesPrefix([]byte(blsKeyringPrefix)), nil)
	defer iter.Release()

	for iter.Next() {
		key := iter.Key()
		value := iter.Value()
		if blsKeyring, err = dkg.LoadBLSKeyringFromBytes(value); err != nil {
			return nil, fmt.Errorf("failed to decode bls keyring: %w", err)
		}
		keyrings[strings.TrimLeft(string(key), blsKeyringPrefix)] = blsKeyring
	}
	return keyrings, iter.Error()
}
