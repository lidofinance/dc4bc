package main

import (
	"bufio"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"os"
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

type terminalCommand struct {
	commandHandler func() error
	description    string
}

type terminal struct {
	reader    *bufio.Reader
	airgapped *airgapped.AirgappedMachine
	commands  map[string]*terminalCommand
}

func NewTerminal(machine *airgapped.AirgappedMachine) *terminal {
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
	t.addCommand("show_address", &terminalCommand{
		commandHandler: t.showAddressCommand,
		description:    "shows an airgapped address",
	})
	t.addCommand("set_address", &terminalCommand{
		commandHandler: t.setAddressCommand,
		description:    "set an airgapped address",
	})
	t.addCommand("show_finished_dkg", &terminalCommand{
		commandHandler: t.showFinishedDKGCommand,
		description:    "shows a list of finished dkg rounds",
	})
	return &t
}

func (t *terminal) addCommand(name string, command *terminalCommand) {
	t.commands[name] = command
}

func (t *terminal) readQRCommand() error {
	qrPaths, err := t.airgapped.HandleQR()
	if err != nil {
		return err
	}

	fmt.Println("An operation in the readed QR code handled successfully, a result operation saved by chunks in following qr codes:")
	for idx, qrPath := range qrPaths {
		fmt.Printf("Operation's chunk #%d: %s\n", idx, qrPath)
	}
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

func (t *terminal) showAddressCommand() error {
	fmt.Println(t.airgapped.GetAddress())
	return nil
}

func (t *terminal) setAddressCommand() error {
	fmt.Printf("Enter your client's address: ")
	address, err := t.reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read address from stdin: %w", err)
	}
	if err = t.airgapped.SetAddress(strings.Trim(address, "\n")); err != nil {
		return fmt.Errorf("failed to save address")
	}
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
		t.airgapped.SetEncryptionKey(password)
		if err = t.airgapped.InitKeys(); err != nil {
			if _, ok := err.(airgapped.DecryptionError); ok {
				continue
			}
			fmt.Printf("Failed to init keys: %v\n", err)
			continue
		}
		break
	}
	return nil
}

func (t *terminal) run() error {
	if t.airgapped.GetAddress() == "" {
		fmt.Println("At first, you need to set address" +
			" of your airgapped machine (should be equal to the client address)")
		if err := t.setAddressCommand(); err != nil {
			return err
		}
	}
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
			fmt.Printf("failled to execute command %s: %v, \n", command, err)
			t.airgapped.Unlock()
			continue
		}
		t.airgapped.Unlock()
	}
}

func (t *terminal) sensitiveDataDrop(passExpiration time.Duration) {
	ticker := time.NewTicker(passExpiration)
	for {
		select {
		case <-ticker.C:
			t.airgapped.DropSensitiveData()
		}
	}
}

var (
	passwordExpiration string
	dbPath             string
)

func init() {
	flag.StringVar(&passwordExpiration, "password_expiration", "10m", "Expiration of the encryption password")
	flag.StringVar(&dbPath, "db_path", "airgapped_db", "Path to airgapped levelDB storage")
}

func main() {
	flag.Parse()

	passwordLifeDuration, err := time.ParseDuration(passwordExpiration)
	if err != nil {
		log.Fatalf("invalid password expiration syntax: %v", err)
	}

	air, err := airgapped.NewAirgappedMachine(dbPath)
	if err != nil {
		log.Fatalf("failed to init airgapped machine %v", err)
	}

	t := NewTerminal(air)
	go t.sensitiveDataDrop(passwordLifeDuration)
	if err = t.run(); err != nil {
		log.Fatalf(err.Error())
	}
}
