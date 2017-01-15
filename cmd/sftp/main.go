package main

import (
	"io"
	"os"
	"os/exec"

	"github.com/Sirupsen/logrus"
	"github.com/alecthomas/kingpin"
	"github.com/nethack42/go-sftp"
)

var (
	serverPath = kingpin.Flag("server", "Path to sftp server binary").Short('D').String()
)

func main() {
	kingpin.Parse()

	cmd := exec.Command(*serverPath, "-l", "DEBUG", "-e")

	out, err := cmd.StdoutPipe()
	if err != nil {
		logrus.Fatal(err)
	}
	in, err := cmd.StdinPipe()
	if err != nil {
		logrus.Fatal(err)
	}

	errpipe, err := cmd.StderrPipe()
	if err != nil {
		logrus.Fatal(err)
	}

	go func() {
		io.Copy(os.Stderr, errpipe)
	}()

	if err := cmd.Start(); err != nil {
		logrus.Fatal(err)
	}

	cli := sftp.NewClient(out, in)

	handle, err := cli.OpenDir("/home")
	if err != nil {
		logrus.Error(err)
	}

	logrus.Infof("OpenDir(/): %s", handle)

	cli.ReadDir(handle)

	cmd.Wait()
}
