package wc_rotation

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"testing"
)

func TestWCRotationVariables(t *testing.T) {
	t.Run(`TestCapellaForkVersion`, func(t *testing.T) {
		actual, _ := hex.DecodeString(`03000000`)

		if bytes.Compare(actual, CapellaForkVersion[:]) != 0 {
			t.Errorf("CapellaForkVersion is wrong got = %v, want %v", actual, CapellaForkVersion)
		}
	})

	t.Run(`TestGenesisForkVersion`, func(t *testing.T) {
		actual, _ := hex.DecodeString(`00000000`)

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
		actual, _ := hex.DecodeString(`4b363db94e286120d76eb905340fdd4e54bfe9f06bf33ff6cf5ad27f511bfe95`)

		if bytes.Compare(actual, GenesisValidatorRoot[:]) != 0 {
			t.Errorf("GenesisValidatorRoot is wrong got = %v, want %v", actual, GenesisValidatorRoot)
		}
	})

	t.Run(`LidoBlsPubKeyBB`, func(t *testing.T) {
		p, _ := base64.StdEncoding.DecodeString("tnrKcfBLZzA3tUAJt2Dxlh84NuVxQUHIkq/bdewINNzmeE2ccu2K19syjP+P6fE+")
		actual := hex.EncodeToString(p)
		expected := `b67aca71f04b673037b54009b760f1961f3836e5714141c892afdb75ec0834dce6784d9c72ed8ad7db328cff8fe9f13e`

		if actual != expected {
			t.Errorf("LidoBlsPubKeyBB is wrong got = %v, want %v", actual, expected)
		}

		if bytes.Compare(p, LidoBlsPubKeyBB[:]) != 0 {
			t.Errorf("LidoBlsPubKeyBB is wrong got = %v, want %v", actual, LidoBlsPubKeyBB)
		}
	})

	t.Run(`ToExecutionAddress`, func(t *testing.T) {
		actual, _ := hex.DecodeString(`b9d7934878b5fb9610b3fe8a5e441e8fad7e293f`)

		if bytes.Compare(actual, ToExecutionAddress[:]) != 0 {
			t.Errorf("ToExecutionAddress is wrong got = %v, want %v", actual, ToExecutionAddress)
		}
	})
}
