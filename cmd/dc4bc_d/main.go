package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"syscall"

	"github.com/lidofinance/dc4bc/client"
	"github.com/lidofinance/dc4bc/qr"
	"github.com/lidofinance/dc4bc/storage"

	"github.com/spf13/cobra"
)

const (
	flagUserName                 = "username"
	flagListenAddr               = "listen_addr"
	flagStateDBDSN               = "state_dbdsn"
	flagStorageDBDSN             = "storage_dbdsn"
	flagStorageTopic             = "storage_topic"
	flagKafkaProducerCredentials = "producer_credentials"
	flagKafkaConsumerCredentials = "consumer_credentials"
	flagKafkaTrustStorePath      = "kafka_truststore_path"
	flagStoreDBDSN               = "key_store_dbdsn"
	flagFramesDelay              = "frames_delay"
	flagChunkSize                = "chunk_size"
	flagConfigPath               = "config_path"
)

func init() {
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
	rootCmd.PersistentFlags().String(flagConfigPath, "", "Path to a config file")
}

type config struct {
	Username            string `json:"username"`
	ListenAddress       string `json:"listen_address"`
	StateDBDSN          string `json:"state_dbdsn"`
	StorageDBDSN        string `json:"storage_dbdsn"`
	StorageTopic        string `json:"storage_topic"`
	KeyStoreDBDSN       string `json:"key_store_dbdsn"`
	FramesDelay         int    `json:"frames_delay"`
	ChunkSize           int    `json:"chunk_size"`
	ProducerCredentials string `json:"producer_credentials"`
	ConsumerCredentials string `json:"consumer_credentials"`
	KafkaTrustStorePath string `json:"kafka_truststore_path"`
}

func readConfig(path string) (config, error) {
	var cfg config
	configBz, err := ioutil.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("failed to read config file: %w", err)
	}
	if err = json.Unmarshal(configBz, &cfg); err != nil {
		return cfg, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return cfg, nil
}

func checkConfig(cfg *config) error {
	v := reflect.ValueOf(cfg)
	v = v.Elem()
	t := reflect.TypeOf(*cfg)

	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).IsZero() {
			return fmt.Errorf("%s cannot be empty", t.Field(i).Tag.Get("json"))
		}
	}
	if cfg.FramesDelay < 0 {
		return fmt.Errorf("frames_delay cannot be less than zero")
	}
	if cfg.ChunkSize < 0 {
		return fmt.Errorf("chunk_size cannot be less than zero")
	}
	return nil
}

func loadConfig(cmd *cobra.Command) (*config, error) {
	var cfg config
	cfgPath, err := cmd.Flags().GetString(flagConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration: %v", err)
	}
	if cfgPath != "" {
		cfg, err = readConfig(cfgPath)
		if err != nil {
			return nil, err
		}
	} else {
		cfg.Username, err = cmd.Flags().GetString(flagUserName)
		if err != nil {
			return nil, fmt.Errorf("failed to read configuration: %v", err)
		}
		cfg.KeyStoreDBDSN, err = cmd.Flags().GetString(flagStoreDBDSN)
		if err != nil {
			return nil, fmt.Errorf("failed to read configuration: %v", err)
		}

		cfg.ListenAddress, err = cmd.Flags().GetString(flagListenAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to read configuration: %v", err)
		}

		cfg.StateDBDSN, err = cmd.Flags().GetString(flagStateDBDSN)
		if err != nil {
			return nil, fmt.Errorf("failed to read configuration: %v", err)
		}

		cfg.FramesDelay, err = cmd.Flags().GetInt(flagFramesDelay)
		if err != nil {
			return nil, fmt.Errorf("failed to read configuration: %v", err)
		}

		cfg.ChunkSize, err = cmd.Flags().GetInt(flagChunkSize)
		if err != nil {
			return nil, fmt.Errorf("failed to read configuration: %v", err)
		}

		cfg.StorageDBDSN, err = cmd.Flags().GetString(flagStorageDBDSN)
		if err != nil {
			return nil, fmt.Errorf("failed to read configuration: %v", err)
		}

		cfg.StorageTopic, err = cmd.Flags().GetString(flagStorageTopic)
		if err != nil {
			return nil, fmt.Errorf("failed to read configuration: %v", err)
		}

		cfg.KafkaTrustStorePath, err = cmd.Flags().GetString(flagKafkaTrustStorePath)
		if err != nil {
			log.Fatalf("failed to read configuration: %v", err)
		}
		cfg.ProducerCredentials, err = cmd.Flags().GetString(flagKafkaProducerCredentials)
		if err != nil {
			log.Fatalf("failed to read configuration: %v", err)
		}
		cfg.ConsumerCredentials, err = cmd.Flags().GetString(flagKafkaConsumerCredentials)
		if err != nil {
			log.Fatalf("failed to read configuration: %v", err)
		}
	}
	if err = checkConfig(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func genKeyPairCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "gen_keys",
		Short: "generates a keypair to sign and verify messages",
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := loadConfig(cmd)
			if err != nil {
				log.Fatalf("failed to load config: %v", err)
			}

			keyPair := client.NewKeyPair()
			keyStore, err := client.NewLevelDBKeyStore(cfg.Username, cfg.KeyStoreDBDSN)
			if err != nil {
				log.Fatalf("failed to init key store: %v", err)
			}
			if err = keyStore.PutKeys(cfg.Username, keyPair); err != nil {
				log.Fatalf("failed to save keypair: %v", err)
			}
			fmt.Printf("keypair generated for user %s and saved to %s\n", cfg.Username, cfg.KeyStoreDBDSN)
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
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := loadConfig(cmd)
			if err != nil {
				log.Fatalf("failed to load config: %v", err)
			}

			ctx := context.Background()
			ctx, cancel := context.WithCancel(ctx)

			state, err := client.NewLevelDBState(cfg.StateDBDSN)
			if err != nil {
				log.Fatalf("Failed to init state client: %v", err)
			}

			tlsConfig, err := storage.GetTLSConfig(cfg.KafkaTrustStorePath)
			if err != nil {
				log.Fatalf("faile to create tls config: %v", err)
			}
			producerCreds, err := parseKafkaAuthCredentials(cfg.ProducerCredentials)
			if err != nil {
				log.Fatal(err.Error())
			}
			consumerCreds, err := parseKafkaAuthCredentials(cfg.ProducerCredentials)
			if err != nil {
				log.Fatal(err.Error())
			}

			stg, err := storage.NewKafkaStorage(ctx, cfg.StorageDBDSN, cfg.StorageTopic, tlsConfig, producerCreds, consumerCreds)
			if err != nil {
				log.Fatalf("Failed to init storage client: %v", err)
			}

			keyStore, err := client.NewLevelDBKeyStore(cfg.Username, cfg.KeyStoreDBDSN)
			if err != nil {
				log.Fatalf("Failed to init key store: %v", err)
			}

			processor := qr.NewCameraProcessor()
			processor.SetDelay(cfg.FramesDelay)
			processor.SetChunkSize(cfg.ChunkSize)

			cli, err := client.NewClient(ctx, cfg.Username, state, stg, keyStore, processor)
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
				if err := cli.StartHTTPServer(cfg.ListenAddress); err != nil {
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
