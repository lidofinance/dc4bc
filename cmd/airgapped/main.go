package main

import (
	"bufio"
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
	airgapped *airgapped.AirgappedMachine
	commands  map[string]*terminalCommand
}

func NewTerminal(machine *airgapped.AirgappedMachine) *terminal {
	t := terminal{machine, make(map[string]*terminalCommand)}
	t.addCommand("read_qr", &terminalCommand{
		commandHandler: t.readQRCommand,
		description:    "Reads QR chunks from camera, handle a decoded operation and returns paths to qr chunks of operation's result",
	})
	t.addCommand("help", &terminalCommand{
		commandHandler: t.helpCommand,
		description:    "shows available commands",
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

func (t *terminal) helpCommand() error {
	fmt.Println("Available commands:")
	for commandName, command := range t.commands {
		fmt.Printf("* %s - %s\n", commandName, command.description)
	}
	return nil
}

func (t *terminal) run() error {
	reader := bufio.NewReader(os.Stdin)
	if err := t.helpCommand(); err != nil {
		return err
	}
	fmt.Println("Waiting for commands...")
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
