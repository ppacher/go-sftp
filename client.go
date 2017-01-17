package sftp

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/nethack42/go-sftp/sshfxp"
)

type Client struct {
	reader io.ReadCloser
	writer io.WriteCloser

	incoming chan sshfxp.Packet
	outgoing chan sshfxp.Packet
	errch    chan error
	ioErr    error

	router *Router

	version uint32

	wg sync.WaitGroup
}

func NewClient(r io.ReadCloser, w io.WriteCloser) *Client {
	cli := &Client{
		reader:   r,
		writer:   w,
		incoming: make(chan sshfxp.Packet),
		outgoing: make(chan sshfxp.Packet),
		router:   NewRouter(),
		errch:    make(chan error, 2), // one error per goroutine
	}

	cli.wg.Add(2)
	go func(cli *Client) {
		defer cli.wg.Done()
		defer logrus.Infof("SFTP client writer exited")

		cli.errch <- writeConn(cli.writer, cli.outgoing)
	}(cli)

	go func(cli *Client) {
		defer cli.wg.Done()
		defer logrus.Infof("SFTP client reader exited")

		cli.errch <- readConn(cli.reader, cli.incoming)
	}(cli)

	if err := cli.DoHandshake(); err != nil {
		logrus.Errorf("SFTP handshake failed: %s", err)

		// Close outgoing
		close(cli.outgoing)

		cli.reader.Close()
		cli.writer.Close()

		cli.wg.Wait()

		cli.ioErr = err

		return nil
	}

	logrus.Infof("SFTP-handeshake complete. Using SFTP version %d", cli.version)

	cli.wg.Add(1)

	go func(cli *Client) {
		defer cli.wg.Done()

	L:
		for {
			select {
			case msg := <-cli.incoming:
				// TODO we currently ignore any error from message handling
				if err := cli.handleMessage(msg); err != nil {
					logrus.Errorf("failed to handle message: %s", err)
				}

			case err := <-cli.errch:
				if err != nil {
					cli.ioErr = err
					logrus.Errorf("received error: %s", err)
				}
				logrus.Infof("received nil on errch")
				break L
			}
		}

		close(cli.outgoing) // will cause writer to stop if it hasn't already

	}(cli)

	return cli
}

// Wait waits for hte cient goroutines to finish
func (cli *Client) Wait() {
	cli.wg.Wait()
}

// DoHandshake establishes a new SFTP connection and performs the initial
// handshake. The protocol version in use can afterwards be retrieved using the
// Version method
func (cli *Client) DoHandshake() error {
	init := &sshfxp.Init{
		Version: 3,
	}

	if _, err := cli.send(init); err != nil {
		return err
	}

	pkt := <-cli.incoming

	msg, err := pkt.Decode()
	if err != nil {
		return err
	}

	if version, ok := msg.(*sshfxp.Version); !ok {
		return errors.New("unexpected message received")
	} else {
		if version.Version != init.Version {
			return errors.New("unsupported version")
		}

		cli.version = version.Version
	}

	return nil
}

// Version returns the SFTP version used. The result is only valid after
// DoHandshake as been called
func (cli *Client) Version() uint32 {
	return cli.version
}

// OpenDir opens a handle to the directory identified by path
func (cli *Client) OpenDir(path string) (string, error) {
	open := &sshfxp.OpenDir{
		Path: path,
	}

	var err error
	var res_chan <-chan sshfxp.Message

	if res_chan, err = cli.send(open); err != nil {
		return "", err
	}

	// wait for result
	var res interface{} = <-res_chan

	if err := sshfxp.IsError(res); err != nil {
		return "", err
	}

	switch msg := res.(type) {
	case *sshfxp.Handle:
		return msg.Handle, nil
	}

	return "", fmt.Errorf("open_dir: unexpected response: %#v", res)
}

// ReadDir reads directory contents for the given handle and returns a list
// of os.FileInfo
func (cli *Client) ReadDir(handle string) ([]os.FileInfo, error) {
	read := &sshfxp.ReadDir{
		Handle: handle,
	}

	resCh, err := cli.send(read)
	if err != nil {
		return nil, err
	}

	res := <-resCh

	if err := sshfxp.IsError(res); err != nil {
		return nil, err
	}

	switch msg := res.(type) {
	case *sshfxp.Name:
		var res []os.FileInfo
		for _, name := range msg.Names {
			res = append(res, FileInfo{
				name:    name.Filename,
				size:    int64(name.Attr.Size),
				mode:    os.FileMode(name.Attr.Permissions),
				modtime: time.Unix(int64(name.Attr.MTime), 0),
				packet:  name,
			})
		}

		return res, nil
	}

	return nil, errors.New("unexpected response")
}

// Close closes the given file or directory handle
func (cli *Client) Close(handle string) error {
	close := &sshfxp.Close{
		Handle: handle,
	}

	resCh, err := cli.send(close)
	if err != nil {
		return err
	}

	if err := sshfxp.IsError(<-resCh); err != nil {
		return err
	}

	return nil
}

// List returns a list of files and directories in a given path. List wraps
// calles to OpenDir, ReadDir and Close
func (cli *Client) List(path string) ([]os.FileInfo, error) {
	handle, err := cli.OpenDir(path)
	if err != nil {
		return nil, err
	}
	defer cli.Close(handle)

	return cli.ReadDir(handle)
}

// Open opens the file identifided by path using the access mode specified in
// flags. If the file is going to be created, attr can hold additional file
// attributes.
//
// BUG: flags and attr is currently not supported
func (cli *Client) Open(path string, flags uint32, attr os.FileInfo) (string, error) {
	open := &sshfxp.Open{
		Filename:   path,
		PFlags:     flags,
		Attributes: sshfxp.Attr{}, // TODO: not yet supported
	}

	resCh, err := cli.send(open)
	if err != nil {
		return "", err
	}

	res := <-resCh
	if err := sshfxp.IsError(res); err != nil {
		return "", err
	}

	switch msg := res.(type) {
	case *sshfxp.Handle:
		return msg.Handle, nil
	}

	return "", errors.New("unexpected response")
}

// Read reads `length` bytes of data from the file identified by handle and
// starting at offset. The file handle must have been acquired previously by
// calling Open()
func (cli *Client) Read(handle string, offset uint64, length uint32) ([]byte, error) {
	read := &sshfxp.Read{
		Handle: handle,
		Offset: offset,
		Length: length,
	}

	resCh, err := cli.send(read)
	if err != nil {
		return nil, err
	}

	res := <-resCh
	if err := sshfxp.IsError(res); err != nil {
		return nil, err
	}

	switch msg := res.(type) {
	case *sshfxp.Data:
		return []byte(msg.Data), nil
	}

	return nil, errors.New("unexpected response")
}

// Write writes the given slice of data to the file identified by handle starting
// at offset. The file handle must have been acquired previously by calling
// Open()
func (cli *Client) Write(handle string, offset uint64, data []byte) error {
	write := &sshfxp.Write{
		Handle: handle,
		Offset: offset,
		Data:   string(data),
	}

	resCh, err := cli.send(write)
	if err != nil {
		return err
	}

	if err := sshfxp.IsError(<-resCh); err != nil {
		return err
	}

	return nil
}

// Remove removes the file identified by path.
func (cli *Client) Remove(path string) error {
	if resCh, err := cli.send(&sshfxp.Remove{File: path}); err != nil {
		return err
	} else if err := sshfxp.IsError(<-resCh); err != nil {
		return err
	}

	return nil
}

// Rename renames the file or directory identified by oldPath to newPath
func (cli *Client) Rename(oldPath, newPath string) error {
	if resCh, err := cli.send(&sshfxp.Rename{OldPath: oldPath, NewPath: newPath}); err != nil {
		return err
	} else if err := sshfxp.IsError(<-resCh); err != nil {
		return err
	}

	return nil
}

func (cli *Client) MkDir(path string, attr os.FileInfo) error {
	mkdir := &sshfxp.MkDir{
		Path: path,
		Attr: sshfxp.Attr{},
	}

	if resCh, err := cli.send(mkdir); err != nil {
		return err
	} else if err := sshfxp.IsError(<-resCh); err != nil {
		return err
	}

	return nil
}

func (cli *Client) RmDir(path string) error {
	if resCh, err := cli.send(&sshfxp.RmDir{Path: path}); err != nil {
		return err
	} else if err := sshfxp.IsError(<-resCh); err != nil {
		return err
	}

	return nil
}

func (cli *Client) send(x sshfxp.Message) (<-chan sshfxp.Message, error) {
	var pkt sshfxp.Packet
	var res <-chan sshfxp.Message

	if header, ok := (interface{}(x)).(sshfxp.Header); ok {
		id, ch := cli.router.Get()

		header.SetID(id)

		res = ch
	}

	if err := pkt.Encode(x); err != nil {
		return nil, err
	}

	cli.outgoing <- pkt

	return res, nil
}

func (cli *Client) handleMessage(msg sshfxp.Packet) error {
	payload, err := msg.Decode()
	if err != nil {
		return fmt.Errorf("failed to decode message: %s", err)
	}

	if err := cli.router.Resolve(payload); err != nil {
		return err
	}

	return nil
}
