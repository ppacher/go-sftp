package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"

	"golang.org/x/net/context"

	"github.com/Sirupsen/logrus"
	"github.com/alecthomas/kingpin"
	"github.com/chzyer/readline"
	"github.com/google/shlex"
	"github.com/nethack42/go-sftp"
)

var (
	debugServer = kingpin.Flag("server", "Path to SFTP server binary").Short('D').String()
	debug       = kingpin.Flag("debug", "Enable debugging").Bool()
)

func startServer(ctx context.Context) *sftp.Client {
	var args []string

	if *debug {
		args = []string{"-l", "debug", "-e"}
	}
	cmd := exec.CommandContext(ctx, *debugServer, args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		logrus.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logrus.Fatal(err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		logrus.Fatal(err)
	}

	go func() {
		io.Copy(os.Stderr, stderr)
	}()

	if err := cmd.Start(); err != nil {
		logrus.Fatal(err)
	}

	cli := sftp.NewClient(stdout, stdin)
	if cli == nil {
		logrus.Fatal(errors.New("NewClient returned nil"))
	}

	return cli
}

func main() {
	kingpin.Parse()

	//sftp.DumpTxPackets = true
	//sftp.DumpRxPackets = true

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cli := startServer(ctx)
	if cli == nil {
		panic("cli is nil")
	}

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "\033[32;1msftp)»\033[0m ",
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})

	if err != nil {
		logrus.Fatal(err)
	}

	for {
		line, err := rl.Readline()

		if err == readline.ErrInterrupt {
			if len(line) == 0 {
				break
			} else {
				continue
			}
		} else if err == io.EOF {
			break
		}

		if err := dispatchCall(cli, line); err != nil {
			break
		}
	}
}

type Command func(*sftp.Client, []string) error

var calls = map[string]Command{
	"exit":   func(*sftp.Client, []string) error { return errors.New("exit") },
	"ls":     listDirectory,
	"mkdir":  mkDir,
	"rmdir":  rmDir,
	"rename": rename,
	"mv":     rename,
}

func dispatchCall(cli *sftp.Client, line string) error {
	tokens, _ := shlex.Split(line)

	if call, ok := calls[tokens[0]]; !ok {
		log.Printf("Unknown command: %s", line)
		return nil
	} else {
		return call(cli, tokens[1:])
	}
	return nil
}

func listDirectory(cli *sftp.Client, params []string) error {
	if len(params) < 1 {
		params = append(params, ".")
	}

	path := params[0]

	handle, err := cli.OpenDir(path)
	if err != nil {
		logrus.Error(err)
		return nil
	}
	defer cli.Close(handle)

	ls, err := cli.ReadDir(handle)
	if err != nil {
		logrus.Error(err)
		return nil
	}

	for _, name := range ls {
		if name.Name() == "." || name.Name() == ".." {
			continue
		}
		fmt.Printf("\033[1m%s\033[0m\n", name.Name())
	}

	return nil
}

func mkDir(cli *sftp.Client, params []string) error {
	if len(params) < 1 {
		log.Println("Missing parameter. Usage: mkdir [path]")
		return nil
	}

	path := params[0]

	if err := cli.MkDir(path, nil); err != nil {
		logrus.Error(err)
	}

	return nil
}

func rmDir(cli *sftp.Client, params []string) error {
	if len(params) < 1 {
		log.Println("Missing parameter. Usage: rmdir [path]")
		return nil
	}

	path := params[0]

	if err := cli.RmDir(path); err != nil {
		logrus.Error(err)
	}

	return nil
}

func rename(cli *sftp.Client, params []string) error {
	if len(params) < 2 {
		log.Println("Missing parameter. Usage: rename [old] [new]")
		return nil
	}

	old := params[0]
	newP := params[1]

	if err := cli.Rename(old, newP); err != nil {
		logrus.Error(err)
	}

	return nil
}
