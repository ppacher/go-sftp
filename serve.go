package sftp

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/fatih/color"
	"github.com/nethack42/go-sftp/sshfxp"
)

var (
	DumpTxPackets = false
	DumpRxPackets = false
)

func readConn(r io.Reader, ch chan<- sshfxp.Packet) error {
	for {
		var pkt sshfxp.Packet

		if err := pkt.Read(r); err != nil {
			return err
		}

		if DumpRxPackets {
			blob, _ := pkt.Bytes()
			hex := hex.Dump(blob)

			print := color.New(color.FgYellow).SprintfFunc()

			fmt.Fprintf(os.Stderr, print("<<<<<<<<<< receive (type=%d len=%d)\n%s<<<<<<<<<<\n", pkt.Type, pkt.Length, hex))
		}

		ch <- pkt
	}
	return nil
}

func writeConn(w io.Writer, ch <-chan sshfxp.Packet) error {
	for pkt := range ch {

		blob, err := pkt.Bytes()
		if err != nil {
			return err
		}

		if DumpTxPackets {
			hex := hex.Dump(blob)
			print := color.New(color.FgGreen).SprintfFunc()

			fmt.Fprintln(os.Stderr, print(">>>>>>>>>> send (type=%d len=%d)\n%v>>>>>>>>>>\n", pkt.Type, pkt.Length, hex))
		}

		if _, err := w.Write(blob); err != nil {
			return err
		}
	}
	return nil
}
