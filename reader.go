package sftp

import (
	"io"
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/nethack42/go-sftp/sshfxp"
)

type FileReader struct {
	cli ClientConn

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
			if e, ok := err.(*sshfxp.FxpStatusError); ok && e.Code == sshfxp.StatusEOF {
				break
			}
			logrus.Errorf("file reader closed! %v", err)
			break
		}

		n, err := fr.pipe_write.Write(buf)
		if n == 0 || err != nil {
			logrus.Errorf("file reader closed! %v", err)
			break
		}

		length += uint64(n)
	}
}

func NewFileReader(path string, cli ClientConn) (io.Reader, error) {
	handle, err := cli.Open(path, sshfxp.OpenRead, nil)
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
