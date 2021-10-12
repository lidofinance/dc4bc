package airgapped

import (
	"fmt"
	"strings"

	"github.com/lidofinance/dc4bc/dkg"
	"github.com/syndtr/goleveldb/leveldb/util"
)

const (
	blsKeyringPrefix = "bls_keyring"
)

func makeBLSKeyKeyringDBKey(key string) string {
	return fmt.Sprintf("%s_%s", blsKeyringPrefix, key)
}

func (am *Machine) saveBLSKeyring(dkgID string, blsKeyring *dkg.BLSKeyring) error {
	salt, err := am.db.Get([]byte(saltDBKey), nil)
	if err != nil {
		return fmt.Errorf("failed to read salt from db: %w", err)
	}

	blsKeyringBz, err := blsKeyring.Bytes()
	if err != nil {
		return fmt.Errorf("failed to encode bls keyring: %w", err)
	}

	encryptedKeyring, err := encrypt(am.encryptionKey, salt, blsKeyringBz)
	if err != nil {
		return fmt.Errorf("failed to encrypt BLS keyring: %w", err)
	}
	if err := am.db.Put([]byte(makeBLSKeyKeyringDBKey(dkgID)), encryptedKeyring, nil); err != nil {
		return fmt.Errorf("failed to save BLSKeyring into db: %w", err)
	}
	return nil
}

func (am *Machine) loadBLSKeyring(dkgID string) (*dkg.BLSKeyring, error) {
	var (
		blsKeyring   *dkg.BLSKeyring
		blsKeyringBz []byte
		err          error
	)

	salt, err := am.db.Get([]byte(saltDBKey), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to read salt from db: %w", err)
	}

	if blsKeyringBz, err = am.db.Get([]byte(makeBLSKeyKeyringDBKey(dkgID)), nil); err != nil {
		return nil, fmt.Errorf("failed to get bls keyring with dkg id %s: %w", dkgID, err)
	}

	decryptedKeyring, err := decrypt(am.encryptionKey, salt, blsKeyringBz)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt BLS keyring: %w", err)
	}

	if blsKeyring, err = dkg.LoadBLSKeyringFromBytes(am.baseSuite, decryptedKeyring); err != nil {
		return nil, fmt.Errorf("failed to decode bls keyring")
	}
	return blsKeyring, nil
}

func (am *Machine) GetBLSKeyrings() (map[string]*dkg.BLSKeyring, error) {
	var (
		blsKeyring *dkg.BLSKeyring
		err        error
	)

	salt, err := am.db.Get([]byte(saltDBKey), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to read salt from db: %w", err)
	}

	keyrings := make(map[string]*dkg.BLSKeyring)
	iter := am.db.NewIterator(util.BytesPrefix([]byte(blsKeyringPrefix)), nil)
	defer iter.Release()

	for iter.Next() {
		key := iter.Key()
		value := iter.Value()
		decryptedKeyring, err := decrypt(am.encryptionKey, salt, value)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt BLS keyring: %w", err)
		}
		if blsKeyring, err = dkg.LoadBLSKeyringFromBytes(am.baseSuite, decryptedKeyring); err != nil {
			return nil, fmt.Errorf("failed to decode bls keyring: %w", err)
		}
		keyrings[strings.TrimLeft(string(key), blsKeyringPrefix)] = blsKeyring
	}
	return keyrings, iter.Error()
}

type BatchPartialSignatures map[string][][]byte

func (b BatchPartialSignatures) AddPartialSignature(messageID string, partialSignature []byte) {
	b[messageID] = append(b[messageID], partialSignature)
}
