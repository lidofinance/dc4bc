package main

import (
	"github.com/spf13/cobra"
	"log"
)

const (
	flagUserName     = "username"
	flagListenAddr   = "listen_addr"
	flagStateDBDSN   = "state_dbdsn"
	flagStorageDBDSN = "storage_dbdsn"
	flagStoreDBDSN   = "key_store_dbdsn"
)

func init() {
	rootCmd.PersistentFlags().String(flagUserName, "testUser", "Username")
	rootCmd.PersistentFlags().String(flagListenAddr, "localhost:8080", "Listen Address")
	rootCmd.PersistentFlags().String(flagStateDBDSN, "./dc4bc_client_state", "State DBDSN")
	rootCmd.PersistentFlags().String(flagStorageDBDSN, "./dc4bc_file_storage", "Storage DBDSN")
	rootCmd.PersistentFlags().String(flagStoreDBDSN, "./dc4bc_key_store", "Key Store DBDSN")
}

var rootCmd = &cobra.Command{
	Use:   "dc4bc_cli",
	Short: "dc4bc client cli utilities implementation",
}

func main() {
	rootCmd.AddCommand(
		getOperationsCommand(),
		getOperationQRPathCommand(),
		readOperationFromCameraCommand(),
		startDKGCommand(),
		proposeSignMessageCommand(),
		getAddressCommand(),
		getPubKeyCommand(),
	)
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Failed to execute root command: %v", err)
	}
}
