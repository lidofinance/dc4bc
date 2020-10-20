package main

import (
	"encoding/hex"
	"fmt"
	prysmBLS "github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
)

func checkSignature() *cobra.Command {
	return &cobra.Command{
		Use:   "check_signature [signature]",
		Short: "checks a signature on prysm compatibility",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			sig, err := hex.DecodeString(args[0])
			if err != nil {
				log.Fatalf("failed to decode signature bytes from string: %v", err)
			}
			if _, err = prysmBLS.SignatureFromBytes(sig); err != nil {
				log.Fatalf("failed to get prysm sig from bytes: %v", err)
			}
			fmt.Println("Signature is correct")
		},
	}
}

func checkPubKey() *cobra.Command {
	return &cobra.Command{
		Use:   "check_pubkey [pubkey]",
		Short: "checks a pubkey on prysm compatibility",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			pubkey, err := hex.DecodeString(args[0])
			if err != nil {
				log.Fatalf("failed to decode pubkey bytes from string: %v", err)
			}
			if _, err = prysmBLS.PublicKeyFromBytes(pubkey); err != nil {
				log.Fatalf("failed to get prysm pubkey from bytes: %v", err)
			}
			fmt.Println("Public key is correct")
		},
	}
}

func verify() *cobra.Command {
	return &cobra.Command{
		Use:   "verify [signature] [pubkey] [file]",
		Short: "verify signature with Prysm",
		Args:  cobra.ExactArgs(3),
		Run: func(cmd *cobra.Command, args []string) {
			sig, err := hex.DecodeString(args[0])
			if err != nil {
				log.Fatalf("failed to decode signature bytes from string: %v", err)
			}
			prysmSig, err := prysmBLS.SignatureFromBytes(sig)
			if err != nil {
				log.Fatalf("failed to get prysm sig from bytes: %v", err)
			}
			pubkey, err := hex.DecodeString(args[1])
			if err != nil {
				log.Fatalf("failed to decode pubkey bytes from string: %v", err)
			}
			prysmPubKey, err := prysmBLS.PublicKeyFromBytes(pubkey)
			if err != nil {
				log.Fatalf("failed to get prysm pubkey from bytes: %v", err)
			}
			msg, err := ioutil.ReadFile(args[2])
			if err != nil {
				log.Fatalf("failed to read file: %v", err)
			}
			if !prysmSig.Verify(prysmPubKey, msg) {
				log.Fatalf("failed to verify prysm signature")
			}
			fmt.Println("Signature is correct")
		},
	}
}

var rootCmd = &cobra.Command{
	Use:   "./prysmCompatibilityChecker",
	Short: "util to check signatures and pubkeys compatibility with Prysm",
}

func main() {
	rootCmd.AddCommand(
		checkPubKey(),
		checkSignature(),
		verify(),
	)
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Failed to execute root command: %v", err)
	}
}
