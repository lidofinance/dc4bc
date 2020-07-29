package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/juju/fslock"
	"io"
	"os"
)

var _ Storage = (*FileStorage)(nil)

type FileStorage struct {
	lockFile *fslock.Lock

	dataFile *os.File
	reader   *bufio.Reader
}

const (
	EOL             = '\n'
	defaultLockFile = "/tmp/dc4bc_storage_lock"
)

func countLines(r io.Reader) uint64 {
	var count uint64
	fileScanner := bufio.NewScanner(r)

	for fileScanner.Scan() {
		count++
	}

	return count
}

// InitFileStorage inits append-only file storage
// It takes two arguments: filename - path to a data file, lockFilename (optional) - path to a lock file
func InitFileStorage(filename string, lockFilename ...string) (Storage, error) {
	var (
		fs  FileStorage
		err error
	)
	if len(lockFilename) > 0 {
		fs.lockFile = fslock.New(lockFilename[0])
	} else {
		fs.lockFile = fslock.New(defaultLockFile)
	}

	if fs.dataFile, err = os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644); err != nil {
		return nil, fmt.Errorf("failed to open a data file: %v", err)
	}
	fs.reader = bufio.NewReader(fs.dataFile)
	return &fs, nil
}

// Send sends a message to an append-only data file, returns a message with offset and id
func (fs *FileStorage) Send(m Message) (Message, error) {
	var (
		data []byte
		err  error
	)
	if err = fs.lockFile.Lock(); err != nil {
		return m, fmt.Errorf("failed to lock a file: %v", err)
	}
	defer fs.lockFile.Unlock()

	m.ID = uuid.New().String()

	if _, err = fs.dataFile.Seek(0, 0); err != nil { // otherwise countLines will return zero
		return m, fmt.Errorf("failed to seek a offset to the start of a data file: %v", err)
	}
	m.Offset = countLines(fs.dataFile)

	if data, err = json.Marshal(m); err != nil {
		return m, fmt.Errorf("failed to marshal a message %v: %v", m, err)
	}

	if _, err = fmt.Fprintln(fs.dataFile, string(data)); err != nil {
		return m, fmt.Errorf("failed to write a message to a data file: %v", err)
	}
	return m, err
}

// GetMessages returns a slice of messages from append-only data file with given offset
func (fs *FileStorage) GetMessages(offset int) ([]Message, error) {
	var (
		msgs []Message
		err  error
		row  []byte
		data Message
	)
	if _, err = fs.dataFile.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to seek a offset to the start of a data file: %v", err)
	}
	for {
		row, err = fs.reader.ReadBytes(EOL)
		if err != nil {
			break
		}

		if offset > 0 {
			offset--
			continue
		}

		if err = json.Unmarshal(row, &data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal a message %s: %v", string(row), err)
		}
		msgs = append(msgs, data)
	}
	if err != io.EOF {
		return nil, fmt.Errorf("error while reading data file: %v", err)
	}
	return msgs, nil
}

func (fs *FileStorage) Close() error {
	return fs.dataFile.Close()
}
