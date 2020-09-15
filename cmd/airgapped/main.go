package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"

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

func (t *terminal) run() error {
	reader := bufio.NewReader(os.Stdin)

	if err := t.helpCommand(); err != nil {
		return err
	}
	fmt.Println("Waiting for command...")
	for {
		fmt.Print(">>> ")
		command, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read command: %w", err)
		}
		handler, ok := t.commands[strings.Trim(command, "\n")]
		if !ok {
			fmt.Printf("unknown command: %s\n", command)
			continue
		}
		if err := handler.commandHandler(); err != nil {
			fmt.Printf("failled to execute command %s: %v, \n", command, err)
			continue
		}
	}
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("missed path to DB, example: ./airgapped path_to_db")
	}
	air, err := airgapped.NewAirgappedMachine(os.Args[1])
	if err != nil {
		log.Fatalf("failed to init airgapped machine %v", err)
	}

	t := NewTerminal(air)
	if err = t.run(); err != nil {
		log.Fatalf(err.Error())
	}
}
