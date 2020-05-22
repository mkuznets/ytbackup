package main

import (
	"log"
	"os"

	"github.com/jessevdk/go-flags"
	"mkuznets.com/go/ytbackup/internal/ytbackup"
)

type Commander interface {
	Init(opts interface{}) error
	Execute(args []string) error
}

type Options struct {
	Common  *ytbackup.Options        `group:"Common Options"`
	History *ytbackup.HistoryCommand `command:"history" description:"show latest watched videos"`
	Config  *ytbackup.ConfigCommand  `command:"config" description:"show current config"`
}

func main() {
	log.SetFlags(0)
	log.SetOutput(os.Stdout)

	var opts Options
	parser := flags.NewParser(&opts, flags.Default)

	parser.CommandHandler = func(command flags.Commander, args []string) error {
		c := command.(Commander)
		if err := c.Init(opts.Common); err != nil {
			return err
		}
		if err := c.Execute(args); err != nil {
			return err
		}
		return nil
	}

	if _, err := parser.Parse(); err != nil {
		switch e := err.(type) {
		case *flags.Error:
			if e.Type == flags.ErrHelp {
				os.Exit(0)
			}
		}
		os.Exit(1)
	}
}
