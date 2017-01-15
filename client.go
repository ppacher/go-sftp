package sftp

import (
	"io"
	"sync"

	"github.com/nethack42/go-sftp/sshfxp"
)

type Client struct {
	conn io.ReadWriter

	incoming chan sshfxp.Packet
	outgoing chan sshfxp.Packet

	wg sync.WaitGroup
}

func NewClient(rw io.ReadWriter) *Client {
	cli := &Client{
		conn:     rw,
		incoming: make(chan sshfxp.Packet),
		outgoing: make(chan sshfxp.Packet),
	}

	go func(cli *Client) {
		defer cli.wg.Done()

		if err := readConn(cli.conn, cli.incoming); err != nil {
			// TODO
		}
	}(cli)

	go func(cli *Client) {
		defer cli.wg.Done()

		if err := writeConn(cli.conn, cli.outgoing); err != nil {
			// TODO
		}
	}(cli)

	go func(cli *Client) {
		// server incoming and outgoing channels
	}(cli)

	return cli
}

func (cli *Client) Wait() {
	cli.wg.Wait()
}
