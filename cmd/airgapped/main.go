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
	"strings"
	"syscall"
	"time"

	passwordTerminal "golang.org/x/crypto/ssh/terminal"

	"github.com/depools/dc4bc/airgapped"
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
}

func NewTerminal(machine *airgapped.Machine) *terminal {
	t := terminal{bufio.NewReader(os.Stdin), machine, make(map[string]*terminalCommand)}
	t.addCommand("read_qr", &terminalCommand{
		commandHandler: t.readQRCommand,
		description:    "Reads QR chunks from camera, handle a decoded operation and returns paths to qr chunks of operation's result",
	})
	t.addCommand("help", &terminalCommand{
		commandHandler: t.helpCommand,
		description:    "shows available commands",
	})
	t.addCommand("show_dkg_pub_key", &terminalCommand{
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
		fmt.Printf("PubKey: %s\n", keyring.PubPoly.Commit().String())
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
		handler, ok := t.commands[strings.Trim(command, "\n")]
		if !ok {
			fmt.Printf("unknown command: %s\n", command)
			continue
		}
		if err = t.enterEncryptionPasswordIfNeeded(); err != nil {
			return err
		}
		t.airgapped.Lock()
		if err := handler.commandHandler(); err != nil {
			fmt.Printf("failled to execute command %s: %v \n", command, err)
			t.airgapped.Unlock()
			continue
		}
		t.airgapped.Unlock()
	}
}

func (t *terminal) dropSensitiveData(passExpiration time.Duration) {
	ticker := time.NewTicker(passExpiration)
	for range ticker.C {
		t.airgapped.DropSensitiveData()
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
	go func() {
		for range c {
			fmt.Printf("Intercepting SIGINT, please type `exit` to stop the machine\n>>> ")
		}
	}()

	t := NewTerminal(air)
	go t.dropSensitiveData(passwordLifeDuration)
	if err = t.run(); err != nil {
		log.Fatalf(err.Error())
	}
}
