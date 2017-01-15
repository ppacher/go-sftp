package sftp

import (
	"io"
	"sync"

	"github.com/nethack42/go-sftp/sshfxp"
)

type Client struct {
	conn io.ReadWriteCloser

	incoming chan sshfxp.Packet
	outgoing chan sshfxp.Packet
	errch    chan error
	ioErr    error

	router *Router

	wg sync.WaitGroup
}

func NewClient(rw io.ReadWriteCloser) *Client {
	cli := &Client{
		conn:     rw,
		incoming: make(chan sshfxp.Packet),
		outgoing: make(chan sshfxp.Packet),
		router:   NewRouter(),
		errch:    make(chan error, 2), // one error per goroutine
	}

	cli.wg.Add(3)

	go func(cli *Client) {
		defer cli.wg.Done()

		cli.errch <- readConn(cli.conn, cli.incoming)
	}(cli)

	go func(cli *Client) {
		defer cli.wg.Done()

		cli.errch <- writeConn(cli.conn, cli.outgoing)
	}(cli)

	go func(cli *Client) {
		defer cli.wg.Done()
	L:
		for {
			select {
			case msg := <-cli.incoming:
				// TODO we currently ignore any error from message handling
				go cli.handleMessage(msg)

			case err := <-cli.errch:
				cli.ioErr = err
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

func (cli *Client) Close() {
	// TODO
	// this will just kill the connection and even messages in queue won't be
	// processed any further.
	cli.conn.Close()
}

func (cli *Client) send(x interface{}) error {
	var pkt sshfxp.Packet

	if err := pkt.Encode(x); err != nil {
		return err
	}

	cli.outgoing <- pkt

	return nil
}

func (cli *Client) handleMessage(msg sshfxp.Packet) error {
	payload, err := msg.Decode()
	if err != nil {
		return err
	}

	if err := cli.router.Resolve(payload); err != nil {
		return err
	}

	return nil
}
