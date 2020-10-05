package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/depools/dc4bc/client"
	"github.com/depools/dc4bc/qr"
	"github.com/depools/dc4bc/storage"

	"github.com/spf13/cobra"
)

const (
	flagUserName     = "username"
	flagListenAddr   = "listen_addr"
	flagStateDBDSN   = "state_dbdsn"
	flagStorageDBDSN = "storage_dbdsn"
	flagStoreDBDSN   = "key_store_dbdsn"
	flagFramesDelay  = "frames_delay"
	flagChunkSize    = "chunk_size"
)

func init() {
	rootCmd.PersistentFlags().String(flagUserName, "testUser", "Username")
	rootCmd.PersistentFlags().String(flagListenAddr, "localhost:8080", "Listen Address")
	rootCmd.PersistentFlags().String(flagStateDBDSN, "./dc4bc_client_state", "State DBDSN")
	rootCmd.PersistentFlags().String(flagStorageDBDSN, "./dc4bc_file_storage", "Storage DBDSN")
	rootCmd.PersistentFlags().String(flagStoreDBDSN, "./dc4bc_key_store", "Key Store DBDSN")
	rootCmd.PersistentFlags().Int(flagFramesDelay, 10, "Delay times between frames in 100ths of a second")
	rootCmd.PersistentFlags().Int(flagChunkSize, 256, "QR-code's chunk size")
}

func genKeyPairCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "gen_keys",
		Short: "generates a keypair to sign and verify messages",
		RunE: func(cmd *cobra.Command, args []string) error {
			userName, err := cmd.Flags().GetString(flagUserName)
			if err != nil {
				return fmt.Errorf("failed to read configuration: %v", err)
			}
			keyStoreDBDSN, err := cmd.Flags().GetString(flagStoreDBDSN)
			if err != nil {
				log.Fatalf("failed to read configuration: %v", err)
			}
			keyPair := client.NewKeyPair()
			keyStore, err := client.NewLevelDBKeyStore(userName, keyStoreDBDSN)
			if err != nil {
				return fmt.Errorf("failed to init key store: %w", err)
			}
			if err = keyStore.PutKeys(userName, keyPair); err != nil {
				return fmt.Errorf("failed to save keypair: %w", err)
			}
			fmt.Printf("keypair generated for user %s and saved to %s\n", userName, keyStoreDBDSN)
			return nil
		},
	}
}

func startClientCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "starts dc4bc client",
		Run: func(cmd *cobra.Command, args []string) {
			userName, err := cmd.Flags().GetString(flagUserName)
			if err != nil {
				log.Fatalf("failed to read configuration: %v", err)
			}

			listenAddr, err := cmd.Flags().GetString(flagListenAddr)
			if err != nil {
				log.Fatalf("failed to read configuration: %v", err)
			}

			stateDBDSN, err := cmd.Flags().GetString(flagStateDBDSN)
			if err != nil {
				log.Fatalf("failed to read configuration: %v", err)
			}

			framesDelay, err := cmd.Flags().GetInt(flagFramesDelay)
			if err != nil {
				log.Fatalf("failed to read configuration: %v", err)
			}

			chunkSize, err := cmd.Flags().GetInt(flagChunkSize)
			if err != nil {
				log.Fatalf("failed to read configuration: %v", err)
			}

			storageDBDSN, err := cmd.Flags().GetString(flagStorageDBDSN)
			if err != nil {
				log.Fatalf("failed to read configuration: %v", err)
			}

			keyStoreDBDSN, err := cmd.Flags().GetString(flagStoreDBDSN)
			if err != nil {
				log.Fatalf("failed to read configuration: %v", err)
			}

			ctx := context.Background()
			ctx, cancel := context.WithCancel(ctx)

			state, err := client.NewLevelDBState(stateDBDSN)
			if err != nil {
				log.Fatalf("Failed to init state client: %v", err)
			}

			stg, err := storage.NewKafkaStorage(ctx, storageDBDSN)
			if err != nil {
				log.Fatalf("Failed to init storage client: %v", err)
			}

			keyStore, err := client.NewLevelDBKeyStore(userName, keyStoreDBDSN)
			if err != nil {
				log.Fatalf("Failed to init key store: %v", err)
			}

			processor := qr.NewCameraProcessor()
			processor.SetDelay(framesDelay)
			processor.SetChunkSize(chunkSize)

			cli, err := client.NewClient(ctx, userName, state, stg, keyStore, processor)
			if err != nil {
				log.Fatalf("Failed to init client: %v", err)
			}

			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigs

				log.Println("Received signal, stopping client...")
				cancel()

				log.Println("BaseClient stopped, exiting")
				os.Exit(0)
			}()

			go func() {
				if err := cli.StartHTTPServer(listenAddr); err != nil {
					log.Fatalf("HTTP server error: %v", err)
				}
			}()
			cli.GetLogger().Log("starting to poll messages from append-only log...")
			if err = cli.Poll(); err != nil {
				log.Fatalf("error while handling operations: %v", err)
			}
			cli.GetLogger().Log("polling is stopped")
		},
	}
}

var rootCmd = &cobra.Command{
	Use:   "dc4bc_d",
	Short: "dc4bc client daemon implementation",
}

func main() {
	rootCmd.AddCommand(
		startClientCommand(),
		genKeyPairCommand(),
	)
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Failed to execute root command: %v", err)
	}
}
