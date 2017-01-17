package sftp

import (
	"io"
	"sync"

	"github.com/Sirupsen/logrus"
)

type FileReader struct {
	cli *Client

	handle string

	pipe_read  io.Reader
	pipe_write io.WriteCloser

	wg sync.WaitGroup
}

func (fr *FileReader) Read(p []byte) (int, error) {
	return fr.pipe_read.Read(p)
}

func (fr *FileReader) fetch() {
	defer fr.wg.Done()
	defer fr.pipe_write.Close()

	var length uint64 = 0

	for {
		buf, err := fr.cli.Read(fr.handle, length, 1024*1024)
		if err != nil {
			logrus.Errorf("file reader closed! %v", err)
			break
		}

		n, err := fr.pipe_write.Write(buf)
		if n == 0 || err != nil {
			logrus.Errorf("file reader closed! %v", err)
			break
		}

		length += uint64(n)
		logrus.Infof("fetched data")
	}
}

func NewFileReader(path string, cli *Client) (io.Reader, error) {
	handle, err := cli.Open(path, 0, nil)
	if err != nil {
		return nil, err
	}

	reader := &FileReader{
		cli:    cli,
		handle: handle,
	}

	in, out := io.Pipe()

	reader.pipe_read = in
	reader.pipe_write = out

	reader.wg.Add(1)
	go reader.fetch()

	return reader, nil
}
