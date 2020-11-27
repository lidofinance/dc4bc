package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	client "github.com/lidofinance/dc4bc/client/types"
	"github.com/syndtr/goleveldb/leveldb"
	"io"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/lidofinance/dc4bc/airgapped"
)

func init() {
	runtime.LockOSThread()
}

// promptCommand holds a description of a command and its handler
type promptCommand struct {
	commandHandler func() error
	description    string
}

// prompt a basic implementation of a prompt
type prompt struct {
	terminal         *terminal.Terminal
	oldTerminalState *terminal.State
	reader           *bufio.Reader
	airgapped        *airgapped.Machine
	commands         map[string]*promptCommand

	currentCommand            string
	stopDroppingSensitiveData chan bool

	exit chan bool
}

func NewPrompt(machine *airgapped.Machine) (*prompt, error) {
	p := prompt{
		reader:                    bufio.NewReader(os.Stdin),
		airgapped:                 machine,
		commands:                  make(map[string]*promptCommand),
		currentCommand:            "",
		stopDroppingSensitiveData: make(chan bool),
		exit:                      make(chan bool, 1),
	}

	if err := p.makeTerminal(); err != nil {
		return nil, err
	}
	p.initTerminal()

	p.addCommand("read_operation", &promptCommand{
		commandHandler: p.readOperationCommand,
		description:    "Reads QR chunks from camera, handle a decoded operation and returns paths to qr chunks of operation's result",
	})
	p.addCommand("help", &promptCommand{
		commandHandler: p.helpCommand,
		description:    "shows available commands",
	})
	p.addCommand("show_dkg_pubkey", &promptCommand{
		commandHandler: p.showDKGPubKeyCommand,
		description:    "shows a dkg pub key",
	})
	p.addCommand("show_finished_dkg", &promptCommand{
		commandHandler: p.showFinishedDKGCommand,
		description:    "shows a list of finished dkg rounds",
	})
	p.addCommand("replay_operations_log", &promptCommand{
		commandHandler: p.replayOperationLogCommand,
		description:    "replays the operation log for a given dkg round",
	})
	p.addCommand("drop_operations_log", &promptCommand{
		commandHandler: p.dropOperationLogCommand,
		description:    "drops the operation log for a given dkg round",
	})
	p.addCommand("exit", &promptCommand{
		commandHandler: p.exitCommand,
		description:    "stops the machine",
	})
	p.addCommand("verify_signature", &promptCommand{
		commandHandler: p.verifySignCommand,
		description:    "verifies a BLS signature of a message",
	})
	p.addCommand("change_configuration", &promptCommand{
		commandHandler: p.changeConfigurationCommand,
		description:    "changes a configuration variables (frames delay, chunk size, etc...)",
	})
	return &p, nil
}

func (p *prompt) commandAutoCompleteCallback(line string, pos int, key rune) (suggestedCommand string, newPos int, ok bool) {
	if key != '\t' {
		return "", 0, false
	}
	for command := range p.commands {
		if strings.HasPrefix(command, line) {
			return command, len(command), true
		}
	}
	return "", 0, false
}

func (p *prompt) print(a ...interface{}) {
	if _, err := fmt.Fprint(p.terminal, a...); err != nil {
		panic(err)
	}
}

func (p *prompt) println(a ...interface{}) {
	if _, err := fmt.Fprintln(p.terminal, a...); err != nil {
		panic(err)
	}
}

func (p *prompt) printf(format string, a ...interface{}) {
	if _, err := fmt.Fprintf(p.terminal, format, a...); err != nil {
		panic(err)
	}
}

func (p *prompt) exitCommand() error {
	p.exit <- true
	return nil
}

func (p *prompt) addCommand(name string, command *promptCommand) {
	p.commands[name] = command
}

func (p *prompt) readOperationCommand() error {
	p.print("> Enter the base64-encoded Operation: ")

	base64Operation, err := p.reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read base64Operation: %w", err)
	}

	operationBz, err := base64.StdEncoding.DecodeString(base64Operation)
	if err != nil {
		return fmt.Errorf("failed to base64.StdEncoding.DecodeString: %w", err)
	}

	var operation client.Operation
	if err := json.Unmarshal(operationBz, &operation); err != nil {
		return fmt.Errorf("failed to unmarshal Operation: %w", err)
	}

	qrPath, err := p.airgapped.ProcessOperation(operation)
	if err != nil {
		return fmt.Errorf("failed to ProcessOperation: %w", err)
	}

	log.Printf("QR code was saved to: %s\n", qrPath)

	p.println("An operation in the read QR code handled successfully, a result operation saved by chunks in following qr codes:")
	p.printf("Operation's chunk: %s\n", qrPath)
	return nil
}

func (p *prompt) showDKGPubKeyCommand() error {
	pubkey := p.airgapped.GetPubKey()
	pubkeyBz, err := pubkey.MarshalBinary()
	if err != nil {
		return fmt.Errorf("failed to marshal DKG pub key: %w", err)
	}
	pubKeyBase64 := base64.StdEncoding.EncodeToString(pubkeyBz)
	p.println(pubKeyBase64)
	return nil
}

func (p *prompt) helpCommand() error {
	p.println("Available commands:")
	for commandName, command := range p.commands {
		p.printf("* %s - %s\n", commandName, command.description)
	}
	return nil
}

func (p *prompt) showFinishedDKGCommand() error {
	keyrings, err := p.airgapped.GetBLSKeyrings()
	if err != nil {
		return fmt.Errorf("failed to get a list of finished dkgs: %w", err)
	}
	for dkgID, keyring := range keyrings {
		p.printf("DKG identifier: %s\n", dkgID)
		pubkeyBz, err := keyring.PubPoly.Commit().MarshalBinary()
		if err != nil {
			p.println("failed to marshal pubkey: %w", err)
			continue
		}
		p.printf("PubKey: %s\n", base64.StdEncoding.EncodeToString(pubkeyBz))
		p.println("-----------------------------------------------------")
	}
	return nil
}

func (p *prompt) replayOperationLogCommand() error {
	p.print("> Enter the DKGRoundIdentifier: ")
	dkgRoundIdentifier, err := p.reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read dkgRoundIdentifier: %w", err)
	}

	if err := p.airgapped.ReplayOperationsLog(strings.Trim(dkgRoundIdentifier, "\n")); err != nil {
		return fmt.Errorf("failed to ReplayOperationsLog: %w", err)
	}
	return nil
}

func (p *prompt) changeConfigurationCommand() error {
	p.print("> Enter a new path to save QR codes (leave empty to avoid changes): ")
	newQRCodesfolder, _, err := p.reader.ReadLine()
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	if len(newQRCodesfolder) > 0 {
		p.airgapped.SetResultQRFolder(string(newQRCodesfolder))
		p.printf("Folder to save QR codes was changed to: %s\n", string(newQRCodesfolder))
	}

	p.print("> Enter a new frames delay in 100ths of second (leave empty to avoid changes): ")
	framesDelayInput, _, err := p.reader.ReadLine()
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	if len(framesDelayInput) > 0 {
		framesDelay, err := strconv.Atoi(string(framesDelayInput))
		if err != nil {
			return fmt.Errorf("failed to parse new frames delay: %w", err)
		}
		p.airgapped.SetQRProcessorFramesDelay(framesDelay)
		p.printf("Frames delay was changed to: %d\n", framesDelay)
	}

	p.print("> Enter a new QR chunk size (leave empty to avoid changes): ")
	chunkSizeInput, _, err := p.reader.ReadLine()
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	if len(chunkSizeInput) > 0 {
		chunkSize, err := strconv.Atoi(string(chunkSizeInput))
		if err != nil {
			return fmt.Errorf("failed to parse new chunk size: %w", err)
		}
		p.airgapped.SetQRProcessorChunkSize(chunkSize)
		p.printf("Chunk size was changed to: %d\n", chunkSize)
	}

	p.print("> Enter a password expiration duration (leave empty to avoid changes): ")
	durationInput, _, err := p.reader.ReadLine()
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	if len(durationInput) > 0 {
		duration, err := time.ParseDuration(string(durationInput))
		if err != nil {
			return fmt.Errorf("failed to parse new duration: %w", err)
		}
		p.stopDroppingSensitiveData <- true
		go p.dropSensitiveDataByTicker(duration)
		p.printf("Password expiration was changed to: %s\n", duration.String())
	}
	return nil
}

func (p *prompt) dropOperationLogCommand() error {
	p.print("> Enter the DKGRoundIdentifier: ")
	dkgRoundIdentifier, err := p.reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read dkgRoundIdentifier: %w", err)
	}

	if err := p.airgapped.DropOperationsLog(dkgRoundIdentifier); err != nil {
		return fmt.Errorf("failed to DropOperationsLog: %w", err)
	}
	return nil
}

func (p *prompt) verifySignCommand() error {
	p.print("> Enter the DKGRoundIdentifier: ")
	dkgRoundIdentifier, err := p.reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read dkgRoundIdentifier: %w", err)
	}

	p.print("> Enter the BLS signature: ")
	signature, err := p.reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read BLS signature (base64): %w", err)
	}

	signatureDecoded, err := base64.StdEncoding.DecodeString(strings.Trim(signature, "\n"))
	if err != nil {
		return fmt.Errorf("failed to decode BLS signature: %w", err)
	}

	p.print("> Enter the message which was signed (base64): ")
	message, err := p.reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read dkgRoundIdentifier: %w", err)
	}

	messageDecoded, err := base64.StdEncoding.DecodeString(strings.Trim(message, "\n"))
	if err != nil {
		return fmt.Errorf("failed to decode message: %w", err)
	}

	if err := p.airgapped.VerifySign(messageDecoded, signatureDecoded, strings.Trim(dkgRoundIdentifier, "\n")); err != nil {
		p.printf("Signature is invalid: %v\n", err)
	} else {
		p.println("Signature is correct!")
	}
	return nil
}

func (p *prompt) enterEncryptionPasswordIfNeeded() error {
	p.airgapped.Lock()
	defer p.airgapped.Unlock()

	if !p.airgapped.SensitiveDataRemoved() {
		return nil
	}

	repeatPassword := p.airgapped.LoadKeysFromDB() == leveldb.ErrNotFound
	for {
		p.print("Enter encryption password: ")
		password, err := terminal.ReadPassword(syscall.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		p.println()
		if repeatPassword {
			p.print("Confirm encryption password: ")
			confirmedPassword, err := terminal.ReadPassword(syscall.Stdin)
			if err != nil {
				return fmt.Errorf("failed to read password: %w", err)
			}
			p.println()
			if !bytes.Equal(password, confirmedPassword) {
				p.println("Passwords do not match! Try again!")
				continue
			}
		}
		p.airgapped.SetEncryptionKey(password)
		if err = p.airgapped.InitKeys(); err != nil {
			p.printf("Failed to init keys: %v\n", err)
			continue
		}
		break
	}
	return nil
}

func (p *prompt) run() error {
	if err := p.enterEncryptionPasswordIfNeeded(); err != nil {
		return err
	}
	if err := p.helpCommand(); err != nil {
		return err
	}
	p.println("Waiting for command...")
	for {
		select {
		case <-p.exit:
			return nil
		default:
			command, err := p.terminal.ReadLine()
			if err != nil && err != io.EOF {
				return fmt.Errorf("failed to read command: %w", err)
			}
			if err == io.EOF {
				// EOF will be returned by pressing CTRL+C/CTRL+D combinations
				// But somehow after the pressing ReadLine will always return EOF
				// So, to avoid this, we just reload the terminal
				p.reloadTerminal()
				continue
			}

			clearCommand := strings.Trim(command, "\n")
			handler, ok := p.commands[clearCommand]
			if !ok {
				p.printf("unknown command: %s\n", command)
				continue
			}
			if err = p.enterEncryptionPasswordIfNeeded(); err != nil {
				return err
			}
			p.airgapped.Lock()

			p.currentCommand = clearCommand

			// we need to "turn off" terminal lib during command execution to be able to handle OS notifications inside
			// commands and to read data from stdin without terminal features
			p.restoreTerminal()
			if err := handler.commandHandler(); err != nil {
				p.printf("failed to execute command %s: %v \n", command, err)
			}
			// after command done, we turning terminal lib back on
			if err = p.makeTerminal(); err != nil {
				return err
			}

			p.currentCommand = ""
			p.airgapped.Unlock()
		}
	}
}

func (p *prompt) dropSensitiveDataByTicker(passExpiration time.Duration) {
	ticker := time.NewTicker(passExpiration)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			p.airgapped.DropSensitiveData()
		case <-p.stopDroppingSensitiveData:
			return
		}
	}
}

func (p *prompt) makeTerminal() error {
	var err error
	if p.oldTerminalState, err = terminal.MakeRaw(0); err != nil {
		return fmt.Errorf("failed to get current terminal state: %w", err)
	}
	return nil
}

// restoreTerminal restores the terminal connected to the given file descriptor to a
// previous state.
func (p *prompt) restoreTerminal() {
	if err := terminal.Restore(0, p.oldTerminalState); err != nil {
		panic(err)
	}
}

func (p *prompt) initTerminal() {
	p.terminal = terminal.NewTerminal(os.Stdin, ">>> ")
	p.terminal.AutoCompleteCallback = p.commandAutoCompleteCallback
}

func (p *prompt) reloadTerminal() {
	p.restoreTerminal()
	if err := p.makeTerminal(); err != nil {
		panic(err)
	}
	p.initTerminal()
}

func (p *prompt) Close() {
	p.restoreTerminal()
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
	air.SetQRProcessorChunkSize(chunkSize)
	air.SetResultQRFolder(qrCodesFolder)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	p, err := NewPrompt(air)
	if err != nil {
		log.Fatalf(err.Error())
	}
	defer p.Close()

	go func() {
		for range c {
			p.printf("Intercepting SIGINT, please type `exit` to stop the machine\n")
		}
	}()
	go p.dropSensitiveDataByTicker(passwordLifeDuration)
	if err = p.run(); err != nil {
		p.printf("Error occurred: %v", err)
	}
}
