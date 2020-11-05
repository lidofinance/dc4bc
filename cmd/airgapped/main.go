package main

import (
	"bufio"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	passwordTerminal "golang.org/x/crypto/ssh/terminal"

	"github.com/lidofinance/dc4bc/airgapped"
)

func init() {
	runtime.LockOSThread()
}

// terminalCommand holds a description of a command and its handler
type terminalCommand struct {
	commandHandler func() error
	description    string
}

// terminal a basic implementation of a prompt
type terminal struct {
	reader    *bufio.Reader
	airgapped *airgapped.Machine
	commands  map[string]*terminalCommand

	currentCommand string
	stopDroppingSensitiveData chan bool
}

func NewTerminal(machine *airgapped.Machine) *terminal {
	t := terminal{
		bufio.NewReader(os.Stdin),
		machine,
		make(map[string]*terminalCommand),
		"",
		make(chan bool),
	}
	t.addCommand("read_qr", &terminalCommand{
		commandHandler: t.readQRCommand,
		description:    "Reads QR chunks from camera, handle a decoded operation and returns paths to qr chunks of operation's result",
	})
	t.addCommand("help", &terminalCommand{
		commandHandler: t.helpCommand,
		description:    "shows available commands",
	})
	t.addCommand("show_dkg_pubkey", &terminalCommand{
		commandHandler: t.showDKGPubKeyCommand,
		description:    "shows a dkg pub key",
	})
	t.addCommand("show_finished_dkg", &terminalCommand{
		commandHandler: t.showFinishedDKGCommand,
		description:    "shows a list of finished dkg rounds",
	})
	t.addCommand("replay_operations_log", &terminalCommand{
		commandHandler: t.replayOperationLogCommand,
		description:    "replays the operation log for a given dkg round",
	})
	t.addCommand("drop_operations_log", &terminalCommand{
		commandHandler: t.dropOperationLogCommand,
		description:    "drops the operation log for a given dkg round",
	})
	t.addCommand("exit", &terminalCommand{
		commandHandler: func() error {
			log.Fatal("interrupted")
			return nil
		},
		description: "stops the machine",
	})
	t.addCommand("verify_signature", &terminalCommand{
		commandHandler: t.verifySignCommand,
		description:    "verifies a BLS signature of a message",
	})
	t.addCommand("change_configuration", &terminalCommand{
		commandHandler: t.changeConfigurationCommand,
		description:    "changes a configuration variables (frames delay, chunk size, etc...)",
	})
	return &t
}

func (t *terminal) addCommand(name string, command *terminalCommand) {
	t.commands[name] = command
}

func (t *terminal) readQRCommand() error {
	qrPath, err := t.airgapped.HandleQR()
	if err != nil {
		return err
	}

	fmt.Println("An operation in the read QR code handled successfully, a result operation saved by chunks in following qr codes:")
	fmt.Printf("Operation's chunk: %s\n", qrPath)
	return nil
}

func (t *terminal) showDKGPubKeyCommand() error {
	pubkey := t.airgapped.GetPubKey()
	pubkeyBz, err := pubkey.MarshalBinary()
	if err != nil {
		return fmt.Errorf("failed to marshal DKG pub key: %w", err)
	}
	pubKeyBase64 := base64.StdEncoding.EncodeToString(pubkeyBz)
	fmt.Println(pubKeyBase64)
	return nil
}

func (t *terminal) helpCommand() error {
	fmt.Println("Available commands:")
	for commandName, command := range t.commands {
		fmt.Printf("* %s - %s\n", commandName, command.description)
	}
	return nil
}

func (t *terminal) showFinishedDKGCommand() error {
	keyrings, err := t.airgapped.GetBLSKeyrings()
	if err != nil {
		return fmt.Errorf("failed to get a list of finished dkgs: %w", err)
	}
	for dkgID, keyring := range keyrings {
		fmt.Printf("DKG identifier: %s\n", dkgID)
		pubkeyBz, err := keyring.PubPoly.Commit().MarshalBinary()
		if err != nil {
			fmt.Println("failed to marshal pubkey: %w", err)
			continue
		}
		fmt.Printf("PubKey: %s\n", base64.StdEncoding.EncodeToString(pubkeyBz))
		fmt.Println("-----------------------------------------------------")
	}
	return nil
}

func (t *terminal) replayOperationLogCommand() error {
	fmt.Print("> Enter the DKGRoundIdentifier: ")
	dkgRoundIdentifier, err := t.reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read dkgRoundIdentifier: %w", err)
	}

	if err := t.airgapped.ReplayOperationsLog(dkgRoundIdentifier); err != nil {
		return fmt.Errorf("failed to ReplayOperationsLog: %w", err)
	}
	return nil
}

func (t *terminal) changeConfigurationCommand() error {
	fmt.Print("> Enter a new path to save QR codes (leave empty to avoid changes): ")
	newQRCodesfolder, _, err := t.reader.ReadLine()
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	if len(newQRCodesfolder) > 0 {
		t.airgapped.SetResultQRFolder(string(newQRCodesfolder))
		fmt.Printf("Folder to save QR codes was changed to: %s\n", string(newQRCodesfolder))
	}

	fmt.Print("> Enter a new frames delay in 100ths of second (leave empty to avoid changes): ")
	framesDelayInput, _, err := t.reader.ReadLine()
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	if len(framesDelayInput) > 0 {
		framesDelay, err := strconv.Atoi(string(framesDelayInput))
		if err != nil {
			return fmt.Errorf("failed to parse new frames delay: %w", err)
		}
		t.airgapped.SetQRProcessorFramesDelay(framesDelay)
		fmt.Printf("Frames delay was changed to: %d\n", framesDelay)
	}

	fmt.Print("> Enter a new QR chunk size (leave empty to avoid changes): ")
	chunkSizeInput, _, err := t.reader.ReadLine()
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	if len(chunkSizeInput) > 0 {
		chunkSize, err := strconv.Atoi(string(chunkSizeInput))
		if err != nil {
			return fmt.Errorf("failed to parse new chunk size: %w", err)
		}
		t.airgapped.SetQRProcessorChunkSize(chunkSize)
		fmt.Printf("Chunk size was changed to: %d\n", chunkSize)
	}

	fmt.Print("> Enter a password expiration duration (leave empty to avoid changes): ")
	durationInput, _, err := t.reader.ReadLine()
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	if len(durationInput) > 0 {
		duration, err := time.ParseDuration(string(durationInput))
		if err != nil {
			return fmt.Errorf("failed to parse new duration: %w", err)
		}
		t.stopDroppingSensitiveData <- true
		go t.dropSensitiveDataByTicker(duration)
		fmt.Printf("Password expiration was changed to: %s\n", duration.String())
	}
	return nil
}

func (t *terminal) dropOperationLogCommand() error {
	fmt.Print("> Enter the DKGRoundIdentifier: ")
	dkgRoundIdentifier, err := t.reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read dkgRoundIdentifier: %w", err)
	}

	if err := t.airgapped.DropOperationsLog(dkgRoundIdentifier); err != nil {
		return fmt.Errorf("failed to DropOperationsLog: %w", err)
	}
	return nil
}

func (t *terminal) verifySignCommand() error {
	fmt.Print("> Enter the DKGRoundIdentifier: ")
	dkgRoundIdentifier, err := t.reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read dkgRoundIdentifier: %w", err)
	}

	fmt.Print("> Enter the BLS signature: ")
	signature, err := t.reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read BLS signature (base64): %w", err)
	}

	signatureDecoded, err := base64.StdEncoding.DecodeString(strings.Trim(signature, "\n"))
	if err != nil {
		return fmt.Errorf("failed to decode BLS signature: %w", err)
	}

	fmt.Print("> Enter the message which was signed (base64): ")
	message, err := t.reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read dkgRoundIdentifier: %w", err)
	}

	messageDecoded, err := base64.StdEncoding.DecodeString(strings.Trim(message, "\n"))
	if err != nil {
		return fmt.Errorf("failed to decode message: %w", err)
	}

	if err := t.airgapped.VerifySign(messageDecoded, signatureDecoded, strings.Trim(dkgRoundIdentifier, "\n")); err != nil {
		fmt.Printf("Signature is invalid: %v\n", err)
	} else {
		fmt.Println("Signature is correct!")
	}
	return nil
}

func (t *terminal) enterEncryptionPasswordIfNeeded() error {
	t.airgapped.Lock()
	defer t.airgapped.Unlock()

	if !t.airgapped.SensitiveDataRemoved() {
		return nil
	}

	for {
		fmt.Print("Enter encryption password: ")
		password, err := passwordTerminal.ReadPassword(syscall.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		fmt.Println()
		t.airgapped.SetEncryptionKey(password)
		if err = t.airgapped.InitKeys(); err != nil {
			fmt.Printf("Failed to init keys: %v\n", err)
			continue
		}
		break
	}
	return nil
}

func (t *terminal) run() error {
	if err := t.enterEncryptionPasswordIfNeeded(); err != nil {
		return err
	}
	if err := t.helpCommand(); err != nil {
		return err
	}
	fmt.Println("Waiting for command...")
	for {
		fmt.Print(">>> ")
		command, err := t.reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read command: %w", err)
		}

		clearCommand := strings.Trim(command, "\n")
		handler, ok := t.commands[clearCommand]
		if !ok {
			fmt.Printf("unknown command: %s\n", command)
			continue
		}
		if err = t.enterEncryptionPasswordIfNeeded(); err != nil {
			return err
		}
		t.airgapped.Lock()

		t.currentCommand = clearCommand
		if err := handler.commandHandler(); err != nil {
			fmt.Printf("failled to execute command %s: %v \n", command, err)
		}
		t.currentCommand = ""
		t.airgapped.Unlock()
	}
}

func (t *terminal) dropSensitiveDataByTicker(passExpiration time.Duration) {
	ticker := time.NewTicker(passExpiration)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			t.airgapped.DropSensitiveData()
		case <-t.stopDroppingSensitiveData:
			return
		}
	}
}

var (
	passwordExpiration string
	dbPath             string
	framesDelay        int
	chunkSize          int
	qrCodesFolder      string
)

func init() {
	flag.StringVar(&passwordExpiration, "password_expiration", "10m", "Expiration of the encryption password")
	flag.StringVar(&dbPath, "db_path", "airgapped_db", "Path to airgapped levelDB storage")
	flag.IntVar(&framesDelay, "frames_delay", 10, "Delay times between frames in 100ths of a second")
	flag.IntVar(&chunkSize, "chunk_size", 256, "QR-code's chunk size")
	flag.StringVar(&qrCodesFolder, "qr_codes_folder", "/tmp/", "Folder to save result QR codes")
}

func main() {
	flag.Parse()

	passwordLifeDuration, err := time.ParseDuration(passwordExpiration)
	if err != nil {
		log.Fatalf("invalid password expiration syntax: %v", err)
	}

	air, err := airgapped.NewMachine(dbPath)
	if err != nil {
		log.Fatalf("failed to init airgapped machine %v", err)
	}
	air.SetQRProcessorFramesDelay(framesDelay)
	air.SetQRProcessorChunkSize(chunkSize)
	air.SetResultQRFolder(qrCodesFolder)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	t := NewTerminal(air)
	go func() {
		for range c {
			if t.currentCommand == "read_qr" {
				t.airgapped.CloseCameraReader()
				continue
			}
			fmt.Printf("Intercepting SIGINT, please type `exit` to stop the machine\n>>> ")
		}
	}()
	go t.dropSensitiveDataByTicker(passwordLifeDuration)
	if err = t.run(); err != nil {
		log.Fatalf(err.Error())
	}
}
