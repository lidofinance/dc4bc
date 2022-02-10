package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/lidofinance/dc4bc/dkg"
	"github.com/lidofinance/dc4bc/pkg/prysm"

	prysmBLS "github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/spf13/cobra"
)

func checkSignature() *cobra.Command {
	return &cobra.Command{
		Use:   "check_signature [signature]",
		Short: "checks a signature on prysm compatibility",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			sig, err := base64.StdEncoding.DecodeString(args[0])
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
			pubkey, err := base64.StdEncoding.DecodeString(args[0])
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
			sig, err := base64.StdEncoding.DecodeString(args[0])
			if err != nil {
				log.Fatalf("failed to decode signature bytes from string: %v", err)
			}
			prysmSig, err := prysmBLS.SignatureFromBytes(sig)
			if err != nil {
				log.Fatalf("failed to get prysm sig from bytes: %v", err)
			}
			pubkey, err := base64.StdEncoding.DecodeString(args[1])
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

func verifyBatch() *cobra.Command {
	return &cobra.Command{
		Use:   "verify_batch [exported_signatures_file] [pubkey] [dir]",
		Short: "verify batch signatures exported with './dc4bc_cli export_signatures' command with Prysm",
		Args:  cobra.ExactArgs(3),
		Run: func(cmd *cobra.Command, args []string) {
			exportedSignaturesFile := args[0]
			pubkeyb64 := args[1]
			dataDir := args[2]

			data, err := ioutil.ReadFile(exportedSignaturesFile)
			if err != nil {
				log.Fatalf("failed to read exported signatures file: %v", err)
			}

			exportedSignatures := make(dkg.ExportedSignatures)

			err = json.Unmarshal(data, &exportedSignatures)
			if err != nil {
				log.Fatalf("failed to unmarshal exported signatures data: %v", err)
			}

			err = prysm.BatchVerification(exportedSignatures, pubkeyb64, dataDir)
			if err != nil {
				log.Fatalln(err)
			}
			fmt.Println("All batch signatures are correct")
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
		verifyBatch(),
	)
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Failed to execute root command: %v", err)
	}
}
