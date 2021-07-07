package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/lidofinance/dc4bc/client"
	"github.com/lidofinance/dc4bc/client/types"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "dkg_reinit_log_adpater",
	Short: "DKG reinit log adpater",
}

const (
	flagUserName        = "username"
	flagKeyStorageDBDSN = "key_storage_dbdsn"
	flagOutputFile      = "output"
	flagInputFile       = "input"
)

func init() {
	rootCmd.PersistentFlags().String(flagUserName, "testUser", "Username")
	rootCmd.PersistentFlags().String(flagKeyStorageDBDSN, "./dc4bc_file_storage_keys", "Key storage DBDSN")
	rootCmd.PersistentFlags().StringP(flagOutputFile, "o", "", "Output file")
	rootCmd.PersistentFlags().StringP(flagInputFile, "i", "", "Input file")

}

func adapt() *cobra.Command {
	return &cobra.Command{
		Use:   "adapt",
		Short: "reads a DKG reinit JSON created by release 0.1.4 and apapt it for latest dc4bc.",

		RunE: func(cmd *cobra.Command, args []string) error {
			inputFile, err := cmd.Flags().GetString(flagInputFile)
			if err != nil {
				return fmt.Errorf("failed to read configuration - \"Input file\": %v", err)
			}
			outputFile, err := cmd.Flags().GetString(flagOutputFile)
			if err != nil {
				return fmt.Errorf("failed to read configuration - \"Output file\": %v", err)
			}
			keyStoragePath, err := cmd.Flags().GetString(flagKeyStorageDBDSN)
			if err != nil {
				return fmt.Errorf("failed to read configuration - \"Key storage DBDSN\": %v", err)
			}

			userName, err := cmd.Flags().GetString(flagUserName)
			if err != nil {
				return fmt.Errorf("failed to read configuration - \"Username\": %v", err)
			}
			keyStore, err := client.NewLevelDBKeyStore(userName, keyStoragePath)
			if err != nil {
				return fmt.Errorf("failed to init key store: %w", err)
			}

			data, err := ioutil.ReadFile(inputFile)
			if err != nil {
				return fmt.Errorf("failed to read input file: %v", err)
			}
			var reDKG types.ReDKG

			err = json.Unmarshal(data, &reDKG)
			if err != nil {
				return fmt.Errorf("failed to decode data into reDKG: %v", err)
			}

			adaptedReDKG, err := client.GetAdaptedReDKG(reDKG, userName, keyStore)
			if err != nil {
				return fmt.Errorf("failed to adapt reinit DKG message: %v", err)
			}
			reDKGBz, err := json.Marshal(adaptedReDKG)
			if err != nil {
				return fmt.Errorf("failed to encode adapted reinit DKG message: %v", err)
			}

			if err = ioutil.WriteFile(outputFile, reDKGBz, 0666); err != nil {
				return fmt.Errorf("failed to save adapted reinit DKG JSON: %v", err)
			}
			return nil
		},
	}
}

func main() {
	rootCmd.AddCommand(
		adapt(),
	)
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Failed to execute root command: %v", err)
	}
}
