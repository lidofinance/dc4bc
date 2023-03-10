package wc_rotation

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestWCRotationVariables(t *testing.T) {
	t.Run(`TestGenesisForkVersion`, func(t *testing.T) {
		actual, _ := hex.DecodeString(`20000089`)

		if bytes.Compare(actual, GenesisForkVersion[:]) != 0 {
			t.Errorf("GenesisForkVersion is wrong got = %v, want %v", actual, GenesisForkVersion)
		}
	})

	t.Run(`DomainBlsToExecutionChange`, func(t *testing.T) {
		actual, _ := hex.DecodeString(`0A000000`)

		if bytes.Compare(actual, DomainBlsToExecutionChange[:]) != 0 {
			t.Errorf("DomainBlsToExecutionChange is wrong got = %v, want %v", actual, DomainBlsToExecutionChange)
		}
	})

	t.Run(`GenesisValidatorRoot`, func(t *testing.T) {
		actual, _ := hex.DecodeString(`6cfce3b409fac8249fde5ed10db637d39a12f9546ecf10c0ff64c4332c647aa6`)

		if bytes.Compare(actual, GenesisValidatorRoot[:]) != 0 {
			t.Errorf("GenesisValidatorRoot is wrong got = %v, want %v", actual, GenesisValidatorRoot)
		}
	})

	t.Run(`LidoBlsPubKeyBB`, func(t *testing.T) {
		actual, _ := hex.DecodeString(`8199b7a8c6998aafb30a955794f5d72a454ed1caf51bdbfc3065973153f64eeb64ff07a5b43cb9007cba3e3ec76ed756`)

		if bytes.Compare(actual, LidoBlsPubKeyBB[:]) != 0 {
			t.Errorf("GenesisValidatorRoot is wrong got = %v, want %v", actual, GenesisValidatorRoot)
		}
	})

	t.Run(`ToExecutionAddress`, func(t *testing.T) {
		actual, _ := hex.DecodeString(`b9d7934878b5fb9610b3fe8a5e441e8fad7e293f`)

		if bytes.Compare(actual, ToExecutionAddress[:]) != 0 {
			t.Errorf("ToExecutionAddress is wrong got = %v, want %v", actual, ToExecutionAddress)
		}
	})
}
