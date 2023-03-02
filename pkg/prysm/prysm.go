package prysm

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"path"

	prysmBLS "github.com/prysmaticlabs/prysm/v3/crypto/bls"

	"github.com/lidofinance/dc4bc/dkg"
)

func BatchVerification(exportedSignatures dkg.ExportedSignatures, pubkeyb64 string, dataDir string) error {
	pubkey, err := base64.StdEncoding.DecodeString(pubkeyb64)
	if err != nil {
		return fmt.Errorf("failed to decode pubkey bytes from string: %w", err)
	}

	prysmPubKey, err := prysmBLS.PublicKeyFromBytes(pubkey)
	if err != nil {
		return fmt.Errorf("failed to get prysm pubkey from bytes: %w", err)
	}

	for _, signature := range exportedSignatures {

		prysmSig, err := prysmBLS.SignatureFromBytes(signature.Signature)
		if err != nil {
			return fmt.Errorf("failed to get prysm sig from bytes(filename - %s): %w", signature.File, err)
		}

		msg, err := ioutil.ReadFile(path.Join(dataDir, signature.File))
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		if !prysmSig.Verify(prysmPubKey, msg) {
			return fmt.Errorf("failed to verify prysm signature for file - %s", signature.File)
		}
	}
	return nil
}
