package sftp

import (
	"io"
	"sync"
)

type FileWriter struct {
	io.WriteCloser

	cli    *Client
	handle string

	pipe_read io.ReadCloser

	wg sync.WaitGroup
}

func (fw *FileWriter) write() {
	defer fw.wg.Done()
	defer fw.cli.Close(fw.handle)
	defer fw.pipe_read.Close()

	offset := uint64(0)
	for {
		p := make([]byte, 1024*1024)

		n, err := fw.pipe_read.Read(p)

		if n > 0 {
			if err := fw.cli.Write(fw.handle, offset, p); err != nil {
				break
			}
			// write file
			offset += uint64(n)
		}

		if err != nil {
			break
		}
	}
}

func NewFileWriter(path string, cli *Client) (*FileWriter, error) {
	handle, err := cli.Open(path, 0, nil)
	if err != nil {
		return nil, err
	}

	pipe_read, pipe_write := io.Pipe()

	writer := &FileWriter{
		WriteCloser: pipe_write,
		pipe_read:   pipe_read,
		cli:         cli,
		handle:      handle,
	}

	writer.wg.Add(1)
	go writer.write()

	return writer, nil
}
