package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/net/context"

	"github.com/Sirupsen/logrus"
	"github.com/alecthomas/kingpin"
	"github.com/chzyer/readline"
	"github.com/google/shlex"
	"github.com/nethack42/go-sftp"
)

var (
	debugServer  = kingpin.Flag("server", "Path to SFTP server binary").Short('D').String()
	debug        = kingpin.Flag("debug", "Enable debugging").Bool()
	debugPackets = kingpin.Flag("dump-packets", "Dump packets sent between SFTP client and server").Bool()
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

	if *debugPackets {
		sftp.DumpTxPackets = true
		sftp.DumpRxPackets = true
	}

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
		AutoComplete:    buildCompleter(cli),
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

type CommandFunc func(*sftp.Client, []string) error

type Completer func(*sftp.Client) readline.DynamicCompleteFunc

type Command struct {
	Fn        CommandFunc
	Completer Completer
	Aliases   []string
}

var calls = map[string]Command{
	"exit":  Command{func(*sftp.Client, []string) error { return errors.New("exit") }, nil, nil},
	"ls":    Command{listDirectory, lsCompleter, []string{"dir"}},
	"mkdir": Command{mkDir, nil, nil},
	"rmdir": Command{rmDir, nil, nil},
	"mv":    Command{rename, nil, []string{"rename"}},
	"rm":    Command{remove, nil, []string{"del"}},
	"cat":   Command{cat, lsCompleter, nil},
	"get":   Command{get, nil, nil},
	"put":   Command{put, nil, nil},
}

func buildCompleter(cli *sftp.Client) *readline.PrefixCompleter {
	var items []readline.PrefixCompleterInterface

	for key := range calls {
		var comp readline.PrefixCompleterInterface

		var names = []string{key}

		names = append(names, calls[key].Aliases...)

		for _, name := range names {
			if calls[key].Completer != nil {
				comp = readline.PcItem(name, readline.PcItemDynamic(calls[key].Completer(cli)))
			} else {
				comp = readline.PcItem(name)
			}

			items = append(items, comp)
		}
	}

	var completer = readline.NewPrefixCompleter(items...)

	return completer
}

func lsCompleter(cli *sftp.Client) readline.DynamicCompleteFunc {
	return func(line string) []string {
		tokens, _ := shlex.Split(line)
		var dir = "."
		if len(tokens) > 1 {
			dir = tokens[1]
		}

		absolute := dir[0] == '/'

		if dir[0] != '.' && dir[0] != '/' {
			dir = "./" + dir
		}

		for {
			log.Printf("listening %s\n", dir)
			files, err := cli.List(dir)
			if err != nil {
				if len(dir) == 0 {
					dir = "."
				}

				parts := strings.Split(dir, "/")
				if len(parts) == 1 {
					return nil
				}

				ndir := strings.Join(parts[:len(parts)-1], "/")

				if absolute {
					ndir = "/" + ndir
				}

				dir = ndir
				continue
			}

			var res []string
			for _, f := range files {
				if f.Name() == "." || f.Name() == ".." {
					continue
				}
				prefix := dir
				if !strings.HasSuffix(prefix, "/") {
					prefix = prefix + "/"
				}
				res = append(res, prefix+f.Name())
			}

			return res
		}

		return nil
	}
	/*
		return func(line string) []string {
			tokens, _ := shlex.Split(line)
			var s string
			var prefix string

			if len(tokens) >= 2 {
				if tokens[1] == "/" {
					s = "/"
				} else {
					parts := strings.Split(tokens[1], "/")
					if len(parts) > 1 {
						i := len(parts) - 1
						s = strings.Join(parts[:i], "/")
						prefix = parts[i]
					} else {
						s = "."
					}
				}
			} else {
				s = "."
			}

			if res, err := cli.List(s); err != nil {
				return nil
			} else {
				var names []string

				if s == "/" {
					s = ""
				}

				for _, v := range res {
					if v.Name() == "." || v.Name() == ".." {
						continue
					}

					if strings.HasPrefix(v.Name(), prefix) {
						names = append(names, s+"/"+v.Name())
					}
				}

				return names
			}
			return nil
		}
	*/
}

func dispatchCall(cli *sftp.Client, line string) error {
	tokens, _ := shlex.Split(line)

	if len(line) == 0 {
		return nil
	}

	for key, cmd := range calls {
		if key == tokens[0] {
			return cmd.Fn(cli, tokens[1:])
		}

		for _, alias := range cmd.Aliases {
			if alias == tokens[0] {
				return cmd.Fn(cli, tokens[1:])
			}
		}
	}

	log.Printf("Unknown command: %s", line)

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

func remove(cli *sftp.Client, params []string) error {
	if len(params) < 1 {
		log.Println("Missing parameter. Usage: remove [path]")
		return nil
	}

	path := params[0]

	if err := cli.Remove(path); err != nil {
		logrus.Error(err)
	}

	return nil
}

func cat(cli *sftp.Client, params []string) error {
	if len(params) < 1 {
		log.Println("Missing parameter. Usage: cat [path]")
		return nil
	}

	path := params[0]

	reader, err := cli.FileReader(path)
	if err != nil {
		logrus.Error(err)
		return nil
	}

	io.Copy(os.Stderr, reader)

	return nil
}

func get(cli *sftp.Client, params []string) error {
	if len(params) < 2 {
		log.Println("Missing parameter. Usage: get [remote] [local]")
		return nil
	}

	remote, local := params[0], params[1]

	return cli.Get(remote, local)
}

func put(cli *sftp.Client, params []string) error {
	if len(params) < 2 {
		log.Println("Missing parameter. Usage: put [local] [remote]")
		return nil
	}

	local, remote := params[0], params[1]

	return cli.Put(local, remote)
}
