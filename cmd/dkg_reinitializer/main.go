package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/lidofinance/dc4bc/pkg/utils"
	"github.com/spf13/cobra"

	"github.com/lidofinance/dc4bc/client/services/node"
	"github.com/lidofinance/dc4bc/client/types"
	"github.com/lidofinance/dc4bc/storage"
)

const (
	flagInputFile   = "input"
	flagOutputFile  = "output"
	flagKeysFile    = "keys"
	flagSeparator   = "separator"
	flagColumnIndex = "column"
	flagSkipHeader  = "skip-header"
	flagAdapt014    = "adapt_0_1_4"
)

var rootCmd = &cobra.Command{
	Use:   "dkg_reinitializer",
	Short: "DKG reinitializer tool",
}

func init() {
	rootCmd.PersistentFlags().StringP(flagInputFile, "i", "", "Input file")
	rootCmd.PersistentFlags().StringP(flagOutputFile, "o", "./reinit.json", "Output file")
	rootCmd.PersistentFlags().StringP(flagKeysFile, "k", "./keys.json", "File with new keys (JSON)")
	rootCmd.PersistentFlags().StringP(flagSeparator, "s", ";", "Separator")
	rootCmd.PersistentFlags().IntP(flagColumnIndex, "p", 4, "Column index (with message JSON)")
	rootCmd.PersistentFlags().Bool(flagSkipHeader, false, "Skip header (if present)")
	rootCmd.PersistentFlags().Bool(flagAdapt014, true, "Adapt 0.1.4 dump")
}

func reinit() *cobra.Command {
	return &cobra.Command{
		Use:   "reinit",
		Short: "reads the input file (CSV-encoded) and returns DKG reinit JSON.",
		RunE: func(cmd *cobra.Command, args []string) error {
			messages, err := readMessages(cmd)
			if err != nil {
				return fmt.Errorf("failed to readMessages: %w", err)
			}

			// Load the new communication public keys.
			newKeysFilePath, _ := cmd.Flags().GetString(flagKeysFile)
			newKeysFile, err := os.Open(newKeysFilePath)
			if err != nil {
				return fmt.Errorf("failed to Open keys file: %w", err)
			}
			defer newKeysFile.Close()

			var newCommPubKeys = map[string][]byte{}
			var dec = json.NewDecoder(newKeysFile)
			if err := dec.Decode(&newCommPubKeys); err != nil {
				return fmt.Errorf("failed to json.Decode keys: %w", err)
			}

			// Generate the re-DKG message.
			reDKG, err := types.GenerateReDKGMessage(messages, newCommPubKeys)
			if err != nil {
				return fmt.Errorf("failed to generate reDKG message:  %w", err)
			}

			// Adapt from 0.1.4 if required.
			if adapt014, _ := cmd.Flags().GetBool(flagAdapt014); adapt014 {
				reDKG, err = node.GetAdaptedReDKG(reDKG)
				if err != nil {
					return fmt.Errorf("failed to adapt reinit DKG message from 0.1.4:  %w", err)
				}
			}

			// Save to disk.
			reDKGBz, err := json.MarshalIndent(reDKG, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to encode reinit DKG message:  %w", err)
			}

			outputFile, _ := cmd.Flags().GetString(flagOutputFile)
			if len(outputFile) == 0 {
				fmt.Println(string(reDKGBz))
				return nil
			}

			if err = ioutil.WriteFile(outputFile, reDKGBz, 0666); err != nil {
				return fmt.Errorf("failed to save reinit DKG JSON: %w", err)
			}

			return nil
		},
	}
}

func readMessages(cmd *cobra.Command) ([]storage.Message, error) {
	inputFilePath, _ := cmd.Flags().GetString(flagInputFile)
	separator, _ := cmd.Flags().GetString(flagSeparator)
	if len(separator) < 1 {
		return nil, errors.New("invalid (empty) separator")
	}

	columnIndex, _ := cmd.Flags().GetInt(flagColumnIndex)
	if columnIndex < 0 {
		return nil, errors.New("invalid (negative) column index")
	}

	skipHeader, _ := cmd.Flags().GetBool(flagSkipHeader)
	return utils.ReadLogMessages(inputFilePath, rune(separator[0]), skipHeader, columnIndex)
}

func main() {
	rootCmd.AddCommand(
		reinit(),
	)
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(fmt.Errorf("Failed to execute root command:  %w", err))
	}
}
