package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"gopkg.in/urfave/cli.v1"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/convox/rack/cmd/convox/stdcli"
	"github.com/convox/rack/manifest"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "start",
		Description: "start an app for local development",
		Usage:       "[directory]",
		Action:      cmdStart,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "file, f",
				Value: "docker-compose.yml",
				Usage: "path to an alternate docker compose manifest file",
			},
			cli.BoolFlag{
				Name:  "no-cache",
				Usage: "pull fresh image dependencies",
			},
			cli.IntFlag{
				Name:  "shift",
				Usage: "Shift allocated port numbers by the given amount",
			},
			cli.BoolTFlag{
				Name:  "sync",
				Usage: "synchronize local file changes into the running containers",
			},
		},
	})
}

func cmdStart(c *cli.Context) error {
	go resizeHandler()

	id, err := currentId()
	stdcli.QOSEventSend("cli-start", id, stdcli.QOSEventProperties{Error: err})

	m, err := manifest.LoadFile(c.String("file"))

	if err != nil {
		return stdcli.ExitError(err)
	}

	if shift := c.Int("shift"); shift > 0 {
		m.Shift(shift)
	}

	if pcc, err := m.PortConflicts(); err != nil || len(pcc) > 0 {
		if err != nil {
			stdcli.ExitError(err)
		}

		return stdcli.ExitError(fmt.Errorf("ports in use: %v", pcc))
	}

	dir, app, err := stdcli.DirApp(c, filepath.Dir(c.String("file")))

	if err != nil {
		return stdcli.ExitError(err)
	}

	run := m.Run(dir, app)

	return run.Start()
}

func resizeHandler() {
	manifest.TerminalWidth, _, _ = terminal.GetSize(int(os.Stdin.Fd()))

	sigch := make(chan os.Signal)
	signal.Notify(sigch, syscall.SIGWINCH)

	for {
		<-sigch
		manifest.TerminalWidth, _, _ = terminal.GetSize(int(os.Stdin.Fd()))
	}
}
