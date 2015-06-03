package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/nicholaskh/go-gnureadline"
	"github.com/nicholaskh/golib/server"
	client "github.com/nicholaskh/pushd/client/go"
)

const (
	COMMAND_PROMPT = "pushd> "
	WELCOME_HINT   = "Welcome to the Pushd command line interface.  Commands end with \\n\n\n"
	HISTORY_FILE   = ".pushd_history"
)

type CommandLine struct {
	done   bool
	client *client.PushdClient
}

func newCommandLine() *CommandLine {
	this := new(CommandLine)
	this.client = client.NewPushdClient("127.0.0.1:2222", server.NewProtocol(), time.Second*2, time.Second*5)
	gnureadline.ReadHistory(HISTORY_FILE)
	gnureadline.StifleHistory(10)

	return this
}

func (this *CommandLine) run() {
	this.client.Connect()
	go func() {
		for {
			res, err := this.client.Read()
			if err != nil {
				this.hint("Read error: %s", err.Error())
				if err == io.EOF {
					this.client.Close()
					this.done = true
					break
				}
			}
			if res[len(res)-1] != '\n' {
				res = append(res, '\n')
			}
			this.hint(string(res))
			this.rePrompt()
		}
	}()
	prompt := COMMAND_PROMPT
	for {
		if this.done {
			break
		}
		line, err := gnureadline.Readline(prompt)
		if prompt != "" {
			prompt = ""
		}
		if err != nil {
			this.hint("Read user input error: %s", err.Error())
		}
		line = strings.TrimRight(line, "\n")
		this.processInput(line)
	}
}

func (this *CommandLine) processInput(line string) {
	switch line {
	case "quit":
		this.done = true
	case "":
		this.rePrompt()
	default:
		this.client.Write([]byte(line))
		gnureadline.AddHistory(line)
	}
}

func (this *CommandLine) hint(format string, params ...interface{}) {
	fmt.Fprintf(os.Stdout, format, params...)
}

func (this *CommandLine) rePrompt() {
	gnureadline.Rl_reset_line_state()
	gnureadline.Rl_expand_prompt(COMMAND_PROMPT)
	gnureadline.Rl_forced_update_display()
}

func main() {
	cli := newCommandLine()
	cli.hint(WELCOME_HINT)
	cli.run()
	gnureadline.WriteHistory(HISTORY_FILE)
}
