package storage

import (
	"bufio"
	"bytes"
	"encoding/json"
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

func countLines(r io.Reader) (uint64, error) {
	var count uint64
	buf := make([]byte, bufio.MaxScanTokenSize)
	for {
		bufferSize, err := r.Read(buf)
		if err != nil && err != io.EOF {
			return 0, err
		}

		var buffPosition int
		for {
			i := bytes.IndexByte(buf[buffPosition:], EOL)
			if i == -1 || bufferSize == buffPosition {
				break
			}
			buffPosition += i + 1
			count++
		}
		if err == io.EOF {
			break
		}
	}

	return count, nil
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
		return nil, err
	}
	fs.reader = bufio.NewReader(fs.dataFile)
	return &fs, nil
}

// Send sends a message to an append-only data file, returns a message with offset and id
func (fs *FileStorage) Send(m Message) (Message, error) {
	var (
		err error
	)
	if err = fs.lockFile.Lock(); err != nil {
		return m, err
	}
	defer fs.lockFile.Unlock()

	m.ID = uuid.New().String()

	if _, err = fs.dataFile.Seek(0, 0); err != nil { // otherwise countLines will return zero
		return m, err
	}
	if m.Offset, err = countLines(fs.dataFile); err != nil {
		return m, err
	}
	data, err := json.Marshal(m)
	if err != nil {
		return m, err
	}
	data = append(data, EOL)
	_, err = fs.dataFile.Write(data)
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
		return nil, err
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
			return nil, err
		}
		msgs = append(msgs, data)
	}
	if err != io.EOF {
		return nil, err
	}
	return msgs, nil
}

func (fs *FileStorage) Close() error {
	return fs.dataFile.Close()
}
