package main

import (
	"context"
	"fmt"
	"github.com/lidofinance/dc4bc/client/api/http_api"
	"github.com/lidofinance/dc4bc/client/modules/keystore"
	"github.com/lidofinance/dc4bc/client/services"
	"github.com/lidofinance/dc4bc/client/services/node"
	"log"
	"os"
	"os/signal"
	"syscall"

	apiconfig "github.com/lidofinance/dc4bc/client/config"
	"github.com/lidofinance/dc4bc/fsm/config"
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
	flagKafkaConsumerGroup       = "kafka_consumer_group"
	flagKafkaTimeout             = "kafka_timeout"
	flagStoreDBDSN               = "key_store_dbdsn"
	flagChunkSize                = "chunk_size"
	flagConfig                   = "config"
	flagSkipCommKeysVerification = "skip_comm_keys_verification"
	flagStorageIgnoreMessages    = "storage_ignore_messages"
	flagOffsetsToIgnoreMessages  = "offsets_to_ignore_messages"
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
	rootCmd.PersistentFlags().String(flagKafkaConsumerGroup, "", "Kafka consumer group")
	rootCmd.PersistentFlags().String(flagKafkaTimeout, "60s", "Kafka I/O Timeout")
	rootCmd.PersistentFlags().String(flagStoreDBDSN, "./dc4bc_key_store", "Key Store DBDSN")
	rootCmd.PersistentFlags().Int(flagFramesDelay, 10, "Delay times between frames in 100ths of a second")
	rootCmd.PersistentFlags().Int(flagChunkSize, 256, "QR-code's chunk size")
	rootCmd.PersistentFlags().StringVar(&cfgFile, flagConfig, "", "path to your config file")
	rootCmd.PersistentFlags().Bool(flagSkipCommKeysVerification, false, "verify messages from append-log or not")
	rootCmd.PersistentFlags().String(flagStorageIgnoreMessages, "", "Messages ids or offsets separated by comma (id_1,id_2,...,id_n) to ignore when reading from storage")
	rootCmd.PersistentFlags().Bool(flagOffsetsToIgnoreMessages, false, "Consider values provided in "+flagStorageIgnoreMessages+" flag to be message offsets instead of ids")

	exitIfError(viper.BindPFlag(flagUserName, rootCmd.PersistentFlags().Lookup(flagUserName)))
	exitIfError(viper.BindPFlag(flagListenAddr, rootCmd.PersistentFlags().Lookup(flagListenAddr)))
	exitIfError(viper.BindPFlag(flagStateDBDSN, rootCmd.PersistentFlags().Lookup(flagStateDBDSN)))
	exitIfError(viper.BindPFlag(flagStorageDBDSN, rootCmd.PersistentFlags().Lookup(flagStorageDBDSN)))
	exitIfError(viper.BindPFlag(flagStorageTopic, rootCmd.PersistentFlags().Lookup(flagStorageTopic)))
	exitIfError(viper.BindPFlag(flagKafkaProducerCredentials, rootCmd.PersistentFlags().Lookup(flagKafkaProducerCredentials)))
	exitIfError(viper.BindPFlag(flagKafkaConsumerCredentials, rootCmd.PersistentFlags().Lookup(flagKafkaConsumerCredentials)))
	exitIfError(viper.BindPFlag(flagKafkaTrustStorePath, rootCmd.PersistentFlags().Lookup(flagKafkaTrustStorePath)))
	exitIfError(viper.BindPFlag(flagKafkaConsumerGroup, rootCmd.PersistentFlags().Lookup(flagKafkaConsumerGroup)))
	exitIfError(viper.BindPFlag(flagKafkaTimeout, rootCmd.PersistentFlags().Lookup(flagKafkaTimeout)))
	exitIfError(viper.BindPFlag(flagStoreDBDSN, rootCmd.PersistentFlags().Lookup(flagStoreDBDSN)))
	exitIfError(viper.BindPFlag(flagFramesDelay, rootCmd.PersistentFlags().Lookup(flagFramesDelay)))
	exitIfError(viper.BindPFlag(flagChunkSize, rootCmd.PersistentFlags().Lookup(flagChunkSize)))
	exitIfError(viper.BindPFlag(flagUserName, rootCmd.PersistentFlags().Lookup(flagUserName)))
	exitIfError(viper.BindPFlag(flagSkipCommKeysVerification, rootCmd.PersistentFlags().Lookup(flagSkipCommKeysVerification)))
	exitIfError(viper.BindPFlag(flagStorageIgnoreMessages, rootCmd.PersistentFlags().Lookup(flagStorageIgnoreMessages)))
	exitIfError(viper.BindPFlag(flagOffsetsToIgnoreMessages, rootCmd.PersistentFlags().Lookup(flagOffsetsToIgnoreMessages)))
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

func prepareConfig() (*apiconfig.Config, error) {
	cfg := apiconfig.Config{}
	qrCfg := apiconfig.QrProcessorConfig{}
	kafkaCfg := apiconfig.KafkaStorageConfig{}
	httpCfg := apiconfig.HttpApiConfig{}

	for _, c := range []interface{}{&cfg, &qrCfg, &kafkaCfg, &httpCfg} {
		err := viper.Unmarshal(c)
		if err != nil {
			return nil, fmt.Errorf("failed to parse cli arguments: %w", err)
		}
	}

	cfg.QrProcessorConfig = &qrCfg
	cfg.HttpApiConfig = &httpCfg
	cfg.KafkaStorageConfig = &kafkaCfg

	return &cfg, nil
}

func genKeyPairCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "gen_keys",
		Short: "generates a keypair to sign and verify messages",
		RunE: func(cmd *cobra.Command, args []string) error {
			username := viper.GetString(flagUserName)

			if len(username) < config.UsernameMinLength {
				return fmt.Errorf("\"username\" minimum length is %d", config.UsernameMinLength)
			}

			if len(username) > config.UsernameMaxLength {
				return fmt.Errorf("\"username\" maximum length is %d", config.UsernameMaxLength)
			}

			keyStoreDBDSN := viper.GetString(flagStoreDBDSN)

			keyPair := keystore.NewKeyPair()
			keyStore, err := keystore.NewLevelDBKeyStore(username, keyStoreDBDSN)
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

func startClientCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "starts dc4bc node",
		RunE: func(cmd *cobra.Command, args []string) error {

			cfg, err := prepareConfig()
			if err != nil {
				log.Fatalln("failed to prepare config: ", err)
			}

			ctx := context.Background()
			ctx, cancel := context.WithCancel(ctx)

			sp, err := services.CreateServiceProviderWithCfg(cfg)
			if err != nil {
				log.Fatalf("failed to init service provider: %+v", err)
			}

			cli, err := node.NewNode(ctx, cfg, sp)
			if err != nil {
				log.Fatalf("failed to init node: %+v", err)
			}

			cli.SetSkipCommKeysVerification(viper.GetBool(flagSkipCommKeysVerification))

			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigs

				log.Println("Received signal, stopping node...")
				cancel()

				log.Println("BaseClient stopped, exiting")
				os.Exit(0)
			}()

			server := http_api.RESTApiProvider{}

			err = server.NewServer(cfg, cli)
			if err != nil {
				log.Fatalf("initializing HTTP server error: %+v", err)
			}

			go func() {
				if err := server.Start(); err != nil {
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
	Short: "dc4bc node daemon implementation",
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
