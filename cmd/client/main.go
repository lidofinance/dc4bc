package main

import (
	"context"
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
	flagListenAddr   = "listen_addr"
	flagStateDBDSN   = "state_dbdsn"
	flagStorageDBDSN = "storage_dbdsn"
)

func init() {
	rootCmd.PersistentFlags().String(flagListenAddr, "localhost:8080", "Listen address")
	rootCmd.PersistentFlags().String(flagStateDBDSN, "./dc4bc_client_state", "State DBDSN")
	rootCmd.PersistentFlags().String(flagStorageDBDSN, "./dc4bc_file_storage", "Storage DBDSN")
}

var rootCmd = &cobra.Command{
	Use:   "dc4bc_client",
	Short: "dc4bc client implementation",
	Run: func(cmd *cobra.Command, args []string) {
		listenAddr, err := cmd.PersistentFlags().GetString(flagListenAddr)
		if err != nil {
			log.Fatalf("failed to read configuration: %v", err)
		}

		stateDBDSN, err := cmd.PersistentFlags().GetString(flagStateDBDSN)
		if err != nil {
			log.Fatalf("failed to read configuration: %v", err)
		}

		storageDBDSN, err := cmd.PersistentFlags().GetString(flagStorageDBDSN)
		if err != nil {
			log.Fatalf("failed to read configuration: %v", err)
		}

		ctx := context.Background()
		ctx, cancel := context.WithCancel(ctx)

		state, err := client.NewLevelDBState(stateDBDSN)
		if err != nil {
			log.Fatalf("Failed to init state client: %v", err)
		}

		stg, err := storage.NewFileStorage(storageDBDSN)
		if err != nil {
			log.Fatalf("Failed to init storage client: %v", err)
		}

		processor := qr.NewCameraProcessor()

		// TODO: create state machine.

		cli, err := client.NewClient(ctx, nil, state, stg, processor)
		if err != nil {
			log.Fatalf("Failed to init client: %v", err)
		}

		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigs

			log.Println("Received signal, stopping client...")
			cancel()

			log.Println("Client stopped, exiting")
			os.Exit(0)
		}()

		log.Printf("Client started on %s...", listenAddr)
		if err := cli.StartHTTPServer(listenAddr); err != nil {
			log.Fatalf("Client error: %v", err)
		}
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Failed to execute root command: %v", err)
	}
}
