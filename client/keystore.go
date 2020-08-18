package client

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/syndtr/goleveldb/leveldb"
)

const (
	secretsKey = "secrets"
)

type KeyStore interface {
	LoadKeys(userName, password string) (*KeyPair, error)
}

// LevelDBKeyStore is a temporary solution for keeping hot node keys.
// The target state is an encrypted storage with password authentication.
type LevelDBKeyStore struct {
	keystoreDb *leveldb.DB
}

func NewLevelDBKeyStore(username, keystorePath string) (KeyStore, error) {
	db, err := leveldb.OpenFile(keystorePath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open keystore: %w", err)
	}

	keystore := &LevelDBKeyStore{
		keystoreDb: db,
	}

	if _, err := keystore.keystoreDb.Get([]byte(secretsKey), nil); err != nil {
		if err := keystore.initJsonKey(secretsKey, map[string]*KeyPair{
			username: NewKeyPair(),
		}); err != nil {
			return nil, fmt.Errorf("failed to init %s storage: %w", operationsKey, err)
		}
	}

	return keystore, nil
}

func (s *LevelDBKeyStore) LoadKeys(userName, password string) (*KeyPair, error) {
	bz, err := s.keystoreDb.Get([]byte(secretsKey), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to read keystore: %w", err)
	}

	var keyPairs = map[string]*KeyPair{}
	if err := json.Unmarshal(bz, &keyPairs); err != nil {
		return nil, fmt.Errorf("failed to unmarshak key pairs: %w", err)
	}

	keyPair, ok := keyPairs[userName]
	if !ok {
		return nil, fmt.Errorf("no key pair found for user %s", userName)
	}

	return keyPair, nil
}

func (s *LevelDBKeyStore) initJsonKey(key string, data interface{}) error {
	if _, err := s.keystoreDb.Get([]byte(key), nil); err != nil {
		operationsBz, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal storage structure: %w", err)
		}
		err = s.keystoreDb.Put([]byte(key), operationsBz, nil)
		if err != nil {
			return fmt.Errorf("failed to init state: %w", err)
		}
	}

	return nil
}

type KeyPair struct {
	Pub  ed25519.PublicKey
	Priv ed25519.PrivateKey
}

func NewKeyPair() *KeyPair {
	// TODO: implement proper generation.
	pub, priv, _ := ed25519.GenerateKey(nil)
	return &KeyPair{
		Pub:  pub,
		Priv: priv,
	}
}

func (p *KeyPair) GetAddr() string {
	return hex.EncodeToString(p.Pub)
}
