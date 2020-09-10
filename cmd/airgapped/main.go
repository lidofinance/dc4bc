package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"github.com/depools/dc4bc/airgapped"
	"log"
	"os"
	"strings"
)

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

func (t *terminal) run() error {
	reader := bufio.NewReader(os.Stdin)

	if t.airgapped.GetAddress() == "" {
		fmt.Println("At first, you need to set address (name)" +
			" of your airgapped machine (should be equal to the client address)")
		if err := t.setAddressCommand(); err != nil {
			return err
		}
	}
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
