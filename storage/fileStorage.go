package storage

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"sync"
)

var _ Storage = (*FileStorage)(nil)

type FileStorage struct {
	sync.Mutex
	file   *os.File
	reader *bufio.Reader
}

const (
	EOL = '\n'
)

func InitFileStorage(filename string) (Storage, error) {
	var (
		fs  FileStorage
		err error
	)
	if fs.file, err = os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644); err != nil {
		return nil, err
	}
	fs.reader = bufio.NewReader(fs.file)
	return &fs, nil
}

func (fs *FileStorage) Post(m Message) error {
	fs.Lock()
	defer fs.Unlock()

	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	data = append(data, EOL)
	_, err = fs.file.Write(data)
	return err
}

func (fs *FileStorage) GetMessages(offset int) ([]Message, error) {
	fs.Lock()
	defer fs.Unlock()

	var (
		msgs []Message
		err  error
		row  []byte
		data Message
	)
	if _, err = fs.file.Seek(0, 0); err != nil {
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
	return fs.file.Close()
}
