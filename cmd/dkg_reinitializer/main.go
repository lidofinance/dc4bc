package main

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/lidofinance/dc4bc/client/services/node"

	"github.com/lidofinance/dc4bc/client/types"
	"github.com/lidofinance/dc4bc/storage"
	"github.com/spf13/cobra"
)

const (
	flagInputFile   = "input"
	flagOutputFile  = "output"
	flagKeysFile    = "keys"
	flagSeparator   = "separator"
	flagColumnIndex = "column"
	flagSkipHeader  = "skip-header"
	flagAdapt140    = "adapt_1_4_0"
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
	rootCmd.PersistentFlags().Bool(flagAdapt140, true, "Adapt 1.4.0 dump")
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
				return fmt.Errorf("failed to generate reDKG message: %v", err)
			}

			// Adapt from 1.4.0 if required.
			if adapt140, _ := cmd.Flags().GetBool(flagAdapt140); adapt140 {
				reDKG, err = node.GetAdaptedReDKG(reDKG)
				if err != nil {
					return fmt.Errorf("failed to adapt reinit DKG message from 1.4.0: %v", err)
				}
			}

			// Save to disk.
			reDKGBz, err := json.MarshalIndent(reDKG, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to encode reinit DKG message: %v", err)
			}

			outputFile, _ := cmd.Flags().GetString(flagOutputFile)
			if len(outputFile) == 0 {
				fmt.Println(string(reDKGBz))
				return nil
			}

			if err = ioutil.WriteFile(outputFile, reDKGBz, 0666); err != nil {
				return fmt.Errorf("failed to save reinit DKG JSON: %v", err)
			}

			return nil
		},
	}
}

func readMessages(cmd *cobra.Command) ([]storage.Message, error) {
	inputFilePath, _ := cmd.Flags().GetString(flagInputFile)
	inputFile, err := os.Open(inputFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to Open input file: %w", err)
	}
	defer inputFile.Close()

	separator, _ := cmd.Flags().GetString(flagSeparator)
	if len(separator) < 1 {
		return nil, errors.New("invalid (empty) separator")
	}

	columnIndex, _ := cmd.Flags().GetInt(flagColumnIndex)
	if columnIndex < 0 {
		return nil, errors.New("invalid (negative) column index")
	}

	reader := csv.NewReader(inputFile)
	reader.Comma = rune(separator[0])
	reader.LazyQuotes = true

	lines, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read dump CSV): %w", err)
	}

	skipHeader, _ := cmd.Flags().GetBool(flagSkipHeader)
	if skipHeader {
		lines = lines[1:]
	}

	var message storage.Message
	var messages []storage.Message
	for _, line := range lines {
		if err := json.Unmarshal([]byte(line[columnIndex]), &message); err != nil {
			return nil, fmt.Errorf("failed to unmarshal line `%s`: %w", line[columnIndex], err)
		}

		messages = append(messages, message)
	}

	return messages, nil
}

func main() {
	rootCmd.AddCommand(
		reinit(),
	)
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Failed to execute root command: %v", err)
	}
}
