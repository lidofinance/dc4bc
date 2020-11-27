package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/lidofinance/dc4bc/client"
	"github.com/lidofinance/dc4bc/qr"
	"github.com/lidofinance/dc4bc/storage"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	flagUserName                 = "username"
	flagListenAddr               = "listen_addr"
	flagStateDBDSN               = "state_dbdsn"
	flagStorageDBDSN             = "storage_dbdsn"
	flagFramesDelay              = "frames_delay"
	flagStorageTopic             = "storage_topic"
	flagKafkaProducerCredentials = "producer_credentials"
	flagKafkaConsumerCredentials = "consumer_credentials"
	flagKafkaTrustStorePath      = "kafka_truststore_path"
	flagStoreDBDSN               = "key_store_dbdsn"
	flagChunkSize                = "chunk_size"
	flagConfig                   = "config"
)

var (
	cfgFile string
)

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().String(flagUserName, "testUser", "Username")
	rootCmd.PersistentFlags().String(flagListenAddr, "localhost:8080", "Listen Address")
	rootCmd.PersistentFlags().String(flagStateDBDSN, "./dc4bc_client_state", "State DBDSN")
	rootCmd.PersistentFlags().String(flagStorageDBDSN, "./dc4bc_file_storage", "Storage DBDSN")
	rootCmd.PersistentFlags().String(flagStorageTopic, "messages", "Storage Topic (Kafka)")
	rootCmd.PersistentFlags().String(flagKafkaProducerCredentials, "producer:producerpass", "Producer credentials for Kafka: username:password")
	rootCmd.PersistentFlags().String(flagKafkaConsumerCredentials, "consumer:consumerpass", "Consumer credentials for Kafka: username:password")
	rootCmd.PersistentFlags().String(flagKafkaTrustStorePath, "certs/ca.pem", "Path to kafka truststore")
	rootCmd.PersistentFlags().String(flagStoreDBDSN, "./dc4bc_key_store", "Key Store DBDSN")
	rootCmd.PersistentFlags().Int(flagFramesDelay, 10, "Delay times between frames in 100ths of a second")
	rootCmd.PersistentFlags().Int(flagChunkSize, 256, "QR-code's chunk size")
	rootCmd.PersistentFlags().StringVar(&cfgFile, flagConfig, "", "path to your config file")

	exitIfError(viper.BindPFlag(flagUserName, rootCmd.PersistentFlags().Lookup(flagUserName)))
	exitIfError(viper.BindPFlag(flagListenAddr, rootCmd.PersistentFlags().Lookup(flagListenAddr)))
	exitIfError(viper.BindPFlag(flagStateDBDSN, rootCmd.PersistentFlags().Lookup(flagStateDBDSN)))
	exitIfError(viper.BindPFlag(flagStorageDBDSN, rootCmd.PersistentFlags().Lookup(flagStorageDBDSN)))
	exitIfError(viper.BindPFlag(flagStorageTopic, rootCmd.PersistentFlags().Lookup(flagStorageTopic)))
	exitIfError(viper.BindPFlag(flagKafkaProducerCredentials, rootCmd.PersistentFlags().Lookup(flagKafkaProducerCredentials)))
	exitIfError(viper.BindPFlag(flagKafkaConsumerCredentials, rootCmd.PersistentFlags().Lookup(flagKafkaConsumerCredentials)))
	exitIfError(viper.BindPFlag(flagKafkaTrustStorePath, rootCmd.PersistentFlags().Lookup(flagKafkaTrustStorePath)))
	exitIfError(viper.BindPFlag(flagStoreDBDSN, rootCmd.PersistentFlags().Lookup(flagStoreDBDSN)))
	rootCmd.PersistentFlags().Int(flagFramesDelay, 10, "Delay times between frames in 100ths of a second")
	exitIfError(viper.BindPFlag(flagChunkSize, rootCmd.PersistentFlags().Lookup(flagChunkSize)))
	exitIfError(viper.BindPFlag(flagUserName, rootCmd.PersistentFlags().Lookup(flagUserName)))
}

func exitIfError(err error) {
	if err != nil {
		log.Fatalf("fatal error: %v", err)
	}
}

func initConfig() {
	if cfgFile == "" {
		return
	}

	viper.SetConfigFile(cfgFile)
	exitIfError(viper.ReadInConfig())
}

func genKeyPairCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "gen_keys",
		Short: "generates a keypair to sign and verify messages",
		RunE: func(cmd *cobra.Command, args []string) error {
			username := viper.GetString(flagUserName)
			keyStoreDBDSN := viper.GetString(flagStoreDBDSN)

			keyPair := client.NewKeyPair()
			keyStore, err := client.NewLevelDBKeyStore(username, keyStoreDBDSN)
			if err != nil {
				return fmt.Errorf("failed to init key store: %w", err)
			}
			if err = keyStore.PutKeys(username, keyPair); err != nil {
				return fmt.Errorf("failed to save keypair: %w", err)
			}
			fmt.Printf("keypair generated for user %s and saved to %s\n", username, keyStoreDBDSN)
			return nil
		},
	}
}

func parseKafkaAuthCredentials(creds string) (*storage.KafkaAuthCredentials, error) {
	credsSplited := strings.SplitN(creds, ":", 2)
	if len(credsSplited) == 1 {
		return nil, fmt.Errorf("failed to parse credentials")
	}
	return &storage.KafkaAuthCredentials{
		Username: credsSplited[0],
		Password: credsSplited[1],
	}, nil
}

func startClientCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "starts dc4bc client",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			ctx, cancel := context.WithCancel(ctx)

			storageTopic := viper.GetString(flagStorageTopic)
			stateDBDSN := viper.GetString(flagStateDBDSN)
			state, err := client.NewLevelDBState(stateDBDSN, storageTopic)
			if err != nil {
				return fmt.Errorf("failed to init state client: %w", err)
			}

			kafkaTrustStorePath := viper.GetString(flagKafkaTrustStorePath)
			tlsConfig, err := storage.GetTLSConfig(kafkaTrustStorePath)
			if err != nil {
				return fmt.Errorf("faile to create tls config: %w", err)
			}

			producerCredentials := viper.GetString(flagKafkaProducerCredentials)
			producerCreds, err := parseKafkaAuthCredentials(producerCredentials)
			if err != nil {
				return fmt.Errorf("failed to parse kafka credentials: %w", err)
			}

			consumerCredentials := viper.GetString(flagKafkaConsumerCredentials)
			consumerCreds, err := parseKafkaAuthCredentials(consumerCredentials)
			if err != nil {
				return fmt.Errorf("failed to parse kafka credentials: %w", err)
			}

			storageDBDSN := viper.GetString(flagStorageDBDSN)
			stg, err := storage.NewKafkaStorage(ctx, storageDBDSN, storageTopic, tlsConfig, producerCreds, consumerCreds)
			if err != nil {
				return fmt.Errorf("failed to init storage client: %w", err)
			}

			username := viper.GetString(flagUserName)
			keyStoreDBDSN := viper.GetString(flagStoreDBDSN)
			keyStore, err := client.NewLevelDBKeyStore(username, keyStoreDBDSN)
			if err != nil {
				return fmt.Errorf("failed to init key store: %w", err)
			}

			framesDelay := viper.GetInt(flagFramesDelay)
			chunkSize := viper.GetInt(flagChunkSize)

			processor := qr.NewCameraProcessor()
			processor.SetDelay(framesDelay)
			processor.SetChunkSize(chunkSize)

			cli, err := client.NewClient(ctx, username, state, stg, keyStore, processor)
			if err != nil {
				return fmt.Errorf("failed to init client: %w", err)
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

			listenAddress := viper.GetString(flagListenAddr)

			go func() {
				if err := cli.StartHTTPServer(listenAddress); err != nil {
					log.Fatalf("HTTP server error: %v", err)
				}
			}()
			cli.GetLogger().Log("Client started to poll messages from append-only log")
			cli.GetLogger().Log("Waiting for messages from append-only log...")
			if err = cli.Poll(); err != nil {
				return fmt.Errorf("error while handling operations: %w", err)
			}
			cli.GetLogger().Log("polling is stopped")
			return nil
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
